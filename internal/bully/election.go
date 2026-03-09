// Package bully implements the Bully Algorithm for leader election in a
// distributed system.
//
// ═══════════════════════════════════════════════════════════════════════
// BULLY ALGORITHM OVERVIEW
// ═══════════════════════════════════════════════════════════════════════
//
// The Bully Algorithm elects the node with the HIGHEST ID as the leader.
//
// Steps:
//  1. TRIGGER: A node detects the leader has failed (heartbeat timeout).
//  2. ELECTION: The detecting node sends Election messages to all nodes
//     with HIGHER IDs.
//  3. OK RESPONSE: Any node with a higher ID responds with OK and starts
//     its own election.
//  4. TIMEOUT: If the initiating node receives no OK responses within a
//     timeout, it declares itself the leader.
//  5. COORDINATOR: The winning node broadcasts a Coordinator message to
//     ALL nodes, announcing itself as the new leader.
//
// Leader Failure Detection:
//   - The leader sends periodic heartbeat messages to all nodes.
//   - If a node does not receive a heartbeat within the timeout period,
//     it assumes the leader has failed and starts an election.
//
// ═══════════════════════════════════════════════════════════════════════
package bully

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"distributed-ticket-system/internal/events"
	"distributed-ticket-system/internal/node"
	pb "distributed-ticket-system/proto/ticket"
)

// ElectionService implements the Bully Algorithm and the gRPC ElectionService.
type ElectionService struct {
	pb.UnimplementedElectionServiceServer

	hub *node.Hub

	mu              sync.Mutex
	electionRunning bool
	stopHeartbeat   chan struct{}
	heartbeatStop   sync.Once
}

// NewElectionService creates a new Bully election service.
func NewElectionService(hub *node.Hub) *ElectionService {
	return &ElectionService{
		hub:           hub,
		stopHeartbeat: make(chan struct{}),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// gRPC SERVER HANDLERS (incoming messages from peers)
// ═══════════════════════════════════════════════════════════════════════

// Election handles an incoming Election message from a lower-ID node.
// Per Bully Algorithm: respond with OK and start our own election.
func (es *ElectionService) Election(ctx context.Context, req *pb.ElectionRequest) (*pb.ElectionResponse, error) {
	myID := es.hub.NodeID()

	// Update Lamport clock on receiving this message
	es.hub.Clock.ReceiveTime(req.LamportTime)

	log.Printf("[Node %d] Received ELECTION from Node %d (Lamport: %d)",
		myID, req.NodeId, req.LamportTime)

	es.hub.Events.LogSimple(events.TypeElectionOK, es.hub.Clock.GetTime(),
		formatMsg("Received Election from Node %d, sending OK", req.NodeId))

	// If we have a higher ID, respond OK and start our own election
	if myID > req.NodeId {
		go es.StartElection()
		return &pb.ElectionResponse{
			Ok:     true,
			NodeId: myID,
		}, nil
	}

	// We have a lower or equal ID — should not normally happen
	// (lower nodes don't send to even-lower nodes), but respond anyway
	return &pb.ElectionResponse{
		Ok:     false,
		NodeId: myID,
	}, nil
}

// Coordinator handles an incoming Coordinator announcement from the new leader.
func (es *ElectionService) Coordinator(ctx context.Context, req *pb.CoordinatorRequest) (*pb.CoordinatorResponse, error) {
	myID := es.hub.NodeID()

	// Update Lamport clock
	es.hub.Clock.ReceiveTime(req.LamportTime)

	log.Printf("[Node %d] Received COORDINATOR announcement: Node %d is the new leader (Lamport: %d)",
		myID, req.LeaderId, req.LamportTime)

	// Accept the new leader
	es.hub.SetLeader(req.LeaderId)
	es.hub.SetState("running")

	es.hub.Events.LogSimple(events.TypeElectionCoord, es.hub.Clock.GetTime(),
		formatMsg("New leader elected: Node %d", req.LeaderId))

	// Stop any ongoing election
	es.mu.Lock()
	es.electionRunning = false
	es.mu.Unlock()

	return &pb.CoordinatorResponse{Acknowledged: true}, nil
}

// Heartbeat handles an incoming heartbeat from the leader.
func (es *ElectionService) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	myID := es.hub.NodeID()

	// Update Lamport clock
	es.hub.Clock.ReceiveTime(req.LamportTime)

	// Confirm the leader is alive
	es.hub.SetPeerAlive(req.LeaderId, true)
	es.hub.SetLeader(req.LeaderId)

	return &pb.HeartbeatResponse{
		Alive:  true,
		NodeId: myID,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════
// ELECTION INITIATION (outgoing messages to peers)
// ═══════════════════════════════════════════════════════════════════════

// StartElection initiates a Bully election.
// This is called when:
//   - The node starts up and needs to find/elect a leader
//   - The node detects leader failure (heartbeat timeout)
//   - The node receives an Election message from a lower-ID node
func (es *ElectionService) StartElection() {
	es.mu.Lock()
	if es.electionRunning {
		es.mu.Unlock()
		return // Election already in progress
	}
	es.electionRunning = true
	es.mu.Unlock()

	myID := es.hub.NodeID()
	lamportTime := es.hub.Clock.Tick()

	log.Printf("[Node %d] *** STARTING BULLY ELECTION *** (Lamport: %d)", myID, lamportTime)
	es.hub.SetState("election")

	es.hub.Events.LogSimple(events.TypeElectionStart, lamportTime,
		formatMsg("Node %d starting Bully election", myID))

	// Step 1: Send Election message to all nodes with HIGHER IDs
	higherPeers := es.hub.GetHigherPeerClients()

	if len(higherPeers) == 0 {
		// No higher-ID nodes exist — I am the highest, declare myself leader
		log.Printf("[Node %d] No higher-ID nodes. Declaring self as leader.", myID)
		es.declareLeader()
		return
	}

	// Send Election to each higher-ID peer
	gotOK := false
	var okMu sync.Mutex
	var wg sync.WaitGroup

	for peerID, clients := range higherPeers {
		wg.Add(1)
		go func(id int32, c *node.PeerClients) {
			defer wg.Done()

			sendTime := es.hub.Clock.SendTime()
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Duration(es.hub.Config.Election.TimeoutMS)*time.Millisecond)
			defer cancel()

			log.Printf("[Node %d] Sending ELECTION to Node %d (Lamport: %d)", myID, id, sendTime)

			resp, err := c.Election.Election(ctx, &pb.ElectionRequest{
				NodeId:      myID,
				LamportTime: sendTime,
			})

			if err != nil {
				log.Printf("[Node %d] Node %d did not respond to election: %v", myID, id, err)
				es.hub.SetPeerAlive(id, false)
				return
			}

			if resp.Ok {
				log.Printf("[Node %d] Received OK from Node %d — a higher node will take over", myID, id)
				es.hub.Events.LogSimple(events.TypeElectionOK, es.hub.Clock.GetTime(),
					formatMsg("Received OK from Node %d", id))
				okMu.Lock()
				gotOK = true
				okMu.Unlock()
			}
		}(peerID, clients)
	}

	wg.Wait()

	if !gotOK {
		// No higher-ID node responded — I win the election
		log.Printf("[Node %d] No OK received from higher nodes. I win the election!", myID)
		es.declareLeader()
	} else {
		// A higher-ID node responded — wait for Coordinator message
		log.Printf("[Node %d] Received OK, waiting for Coordinator announcement...", myID)
		es.hub.SetState("waiting_coordinator")

		// Set a timeout: if no Coordinator arrives, restart election
		go func() {
			time.Sleep(time.Duration(es.hub.Config.Election.TimeoutMS*2) * time.Millisecond)
			if es.hub.GetLeader() == -1 || es.hub.GetState() == "waiting_coordinator" {
				log.Printf("[Node %d] Coordinator timeout — restarting election", myID)
				es.mu.Lock()
				es.electionRunning = false
				es.mu.Unlock()
				es.StartElection()
			}
		}()
	}
}

// declareLeader declares this node as the leader and broadcasts Coordinator to all peers.
func (es *ElectionService) declareLeader() {
	myID := es.hub.NodeID()
	lamportTime := es.hub.Clock.Tick()

	es.hub.SetLeader(myID)
	es.hub.SetState("running")

	log.Printf("[Node %d] *** I AM THE NEW LEADER *** (Lamport: %d)", myID, lamportTime)

	es.hub.Events.LogSimple(events.TypeElectionCoord, lamportTime,
		formatMsg("Node %d declared as leader", myID))

	// Broadcast Coordinator message to ALL peers
	allPeers := es.hub.GetAllPeerClients()
	for peerID, clients := range allPeers {
		go func(id int32, c *node.PeerClients) {
			sendTime := es.hub.Clock.SendTime()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			log.Printf("[Node %d] Sending COORDINATOR to Node %d (Lamport: %d)", myID, id, sendTime)

			_, err := c.Election.Coordinator(ctx, &pb.CoordinatorRequest{
				LeaderId:    myID,
				LamportTime: sendTime,
			})
			if err != nil {
				log.Printf("[Node %d] Failed to send Coordinator to Node %d: %v", myID, id, err)
			}
		}(peerID, clients)
	}

	es.mu.Lock()
	es.electionRunning = false
	es.mu.Unlock()

	// Start sending heartbeats as the new leader
	go es.StartHeartbeat()
}

// ═══════════════════════════════════════════════════════════════════════
// HEARTBEAT (leader failure detection)
// ═══════════════════════════════════════════════════════════════════════

// StartHeartbeat begins sending periodic heartbeats if this node is the leader,
// or monitoring heartbeats if this node is a follower.
func (es *ElectionService) StartHeartbeat() {
	myID := es.hub.NodeID()

	if es.hub.IsLeader() {
		es.sendHeartbeats()
	} else {
		es.monitorHeartbeats(myID)
	}
}

// sendHeartbeats sends periodic heartbeats to all peers (called by leader only).
func (es *ElectionService) sendHeartbeats() {
	myID := es.hub.NodeID()
	interval := time.Duration(es.hub.Config.Heartbeat.IntervalMS) * time.Millisecond

	log.Printf("[Node %d] Starting heartbeat broadcast (interval: %v)", myID, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !es.hub.IsLeader() {
				log.Printf("[Node %d] No longer leader, stopping heartbeat", myID)
				return
			}

			allPeers := es.hub.GetAllPeerClients()
			for peerID, clients := range allPeers {
				go func(id int32, c *node.PeerClients) {
					sendTime := es.hub.Clock.SendTime()
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()

					resp, err := c.Election.Heartbeat(ctx, &pb.HeartbeatRequest{
						LeaderId:    myID,
						LamportTime: sendTime,
					})
					if err != nil {
						log.Printf("[Node %d] Heartbeat to Node %d failed: %v", myID, id, err)
						es.hub.SetPeerAlive(id, false)
					} else if resp.Alive {
						es.hub.SetPeerAlive(id, true)
					}
				}(peerID, clients)
			}

			es.hub.Events.LogSimple(events.TypeHeartbeatSent, es.hub.Clock.GetTime(),
				formatMsg("Leader Node %d sent heartbeat", myID))

		case <-es.stopHeartbeat:
			return
		}
	}
}

// monitorHeartbeats watches for leader heartbeats and triggers election on timeout.
func (es *ElectionService) monitorHeartbeats(myID int32) {
	timeout := time.Duration(es.hub.Config.Heartbeat.TimeoutMS) * time.Millisecond

	log.Printf("[Node %d] Monitoring leader heartbeats (timeout: %v)", myID, timeout)

	for {
		time.Sleep(timeout)

		// Check if leader is still alive
		leader := es.hub.GetLeader()
		if leader == -1 || leader == myID {
			continue // No leader set or we are the leader
		}

		// Try to ping the leader
		leaderClients := es.hub.GetPeerClients(leader)
		if leaderClients == nil {
			log.Printf("[Node %d] Leader Node %d not connected — starting election", myID, leader)
			es.hub.Events.LogSimple(events.TypeLeaderFailure, es.hub.Clock.Tick(),
				formatMsg("Leader Node %d not reachable, starting election", leader))
			es.hub.SetLeader(-1)
			go es.StartElection()
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := leaderClients.Election.Heartbeat(ctx, &pb.HeartbeatRequest{
			LeaderId:    leader,
			LamportTime: es.hub.Clock.SendTime(),
		})
		cancel()

		if err != nil {
			log.Printf("[Node %d] *** LEADER NODE %d FAILED *** Starting election...", myID, leader)
			es.hub.Events.LogSimple(events.TypeLeaderFailure, es.hub.Clock.Tick(),
				formatMsg("Leader Node %d failed (heartbeat timeout), starting election", leader))
			es.hub.SetPeerAlive(leader, false)
			es.hub.SetLeader(-1)
			go es.StartElection()
		}
	}
}

// StopHeartbeat stops the heartbeat routine.
func (es *ElectionService) StopHeartbeat() {
	es.heartbeatStop.Do(func() {
		close(es.stopHeartbeat)
	})
}

// IsElectionRunning returns whether an election is currently in progress.
func (es *ElectionService) IsElectionRunning() bool {
	es.mu.Lock()
	defer es.mu.Unlock()
	return es.electionRunning
}

// ═══════════════════════════════════════════════════════════════════════
// HELPER
// ═══════════════════════════════════════════════════════════════════════

func formatMsg(format string, args ...interface{}) string {
	return log.Prefix() + fmt.Sprintf(format, args...)
}
