// Package ra implements the Ricart–Agrawala Algorithm for distributed mutual
// exclusion in the ticket booking system.
//
// ═══════════════════════════════════════════════════════════════════════
// RICART–AGRAWALA ALGORITHM OVERVIEW
// ═══════════════════════════════════════════════════════════════════════
//
// The Ricart–Agrawala algorithm allows a node to enter a critical section
// (e.g., booking a specific seat) without conflicts with other nodes.
//
// Steps:
//
//  1. REQUEST: When a node wants to enter the critical section for a
//     resource (seat), it increments its Lamport clock and sends a
//     REQUEST(timestamp, nodeID, resource) to ALL other nodes.
//
//  2. REPLY DECISION: When a node receives a REQUEST, it decides:
//     a. If NOT interested in the same resource → send REPLY immediately.
//     b. If ALSO interested in the same resource:
//     - If the incoming request has a LOWER timestamp → send REPLY
//     (the requester has priority)
//     - If timestamps are EQUAL, the LOWER node ID gets priority
//     - Otherwise → DEFER the reply (add to deferred queue)
//     c. If currently IN the critical section for that resource → DEFER.
//
//  3. ENTER CS: A node enters the critical section only after receiving
//     REPLY from ALL other nodes.
//
//  4. EXIT CS: After leaving the critical section, send REPLY to all
//     deferred requests.
//
// The "resource" in our system is a specific seat ID (e.g., "A1").
// This means two nodes wanting DIFFERENT seats do NOT block each other.
//
// ═══════════════════════════════════════════════════════════════════════
package ra

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

// State represents the node's state with respect to the critical section.
type State int

const (
	StateReleased State = iota // Not interested in critical section
	StateWanting               // Wants to enter critical section
	StateHeld                  // Currently in critical section
)

// DeferredReply represents a reply that was deferred because we are also
// interested in the same resource.
type DeferredReply struct {
	NodeID   int32
	Resource string
}

// MutexService implements the Ricart-Agrawala algorithm and gRPC MutexService.
type MutexService struct {
	pb.UnimplementedMutexServiceServer

	hub *node.Hub

	mu              sync.Mutex
	state           map[string]State           // resource → current state
	requestTime     map[string]int64           // resource → our Lamport timestamp when we sent REQUEST
	repliesReceived map[string]int             // resource → count of REPLY messages received
	repliesNeeded   map[string]int             // resource → total replies needed
	deferredReplies map[string][]DeferredReply // resource → list of deferred replies
	waitCh          map[string]chan struct{}   // resource → channel signaled when all replies received
}

// NewMutexService creates a new Ricart-Agrawala mutual exclusion service.
func NewMutexService(hub *node.Hub) *MutexService {
	return &MutexService{
		hub:             hub,
		state:           make(map[string]State),
		requestTime:     make(map[string]int64),
		repliesReceived: make(map[string]int),
		repliesNeeded:   make(map[string]int),
		deferredReplies: make(map[string][]DeferredReply),
		waitCh:          make(map[string]chan struct{}),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// gRPC SERVER HANDLER (incoming REQUEST from peers)
// ═══════════════════════════════════════════════════════════════════════

// RequestAccess handles an incoming Ricart-Agrawala REQUEST from a peer.
// This is the core decision logic of the algorithm.
func (ms *MutexService) RequestAccess(ctx context.Context, req *pb.MutexRequest) (*pb.MutexResponse, error) {
	myID := ms.hub.NodeID()
	resource := req.Resource

	// Update Lamport clock on receive
	ms.hub.Clock.ReceiveTime(req.LamportTime)
	myTime := ms.hub.Clock.GetTime()

	log.Printf("[Node %d] RA: Received REQUEST from Node %d for seat %s (Lamport: %d, my time: %d)",
		myID, req.NodeId, resource, req.LamportTime, myTime)

	ms.hub.Events.Log(events.TypeRARequestReceived, myTime, resource, "",
		fmt.Sprintf("Received RA request from Node %d for seat %s (ts=%d)", req.NodeId, resource, req.LamportTime))

	ms.mu.Lock()
	defer ms.mu.Unlock()

	currentState := ms.state[resource]

	// Decision logic:
	// 1. If we are RELEASED (not interested) → reply immediately
	// 2. If we are HELD (in critical section) → defer
	// 3. If we are WANTING → compare timestamps
	shouldReply := false

	switch currentState {
	case StateReleased:
		// Not interested in this resource — reply immediately
		shouldReply = true
		log.Printf("[Node %d] RA: Not interested in seat %s → sending REPLY to Node %d", myID, resource, req.NodeId)

	case StateHeld:
		// Currently in critical section — defer reply
		shouldReply = false
		log.Printf("[Node %d] RA: In critical section for seat %s → DEFERRING reply to Node %d", myID, resource, req.NodeId)

	case StateWanting:
		// Also wanting this resource — compare timestamps for priority
		myRequestTime := ms.requestTime[resource]

		if req.LamportTime < myRequestTime {
			// Incoming request has lower timestamp → they have priority
			shouldReply = true
			log.Printf("[Node %d] RA: Incoming request (ts=%d) < my request (ts=%d) → REPLY to Node %d",
				myID, req.LamportTime, myRequestTime, req.NodeId)
		} else if req.LamportTime == myRequestTime && req.NodeId < myID {
			// Same timestamp, lower node ID gets priority
			shouldReply = true
			log.Printf("[Node %d] RA: Same timestamp, Node %d < Node %d → REPLY",
				myID, req.NodeId, myID)
		} else {
			// We have priority — defer reply
			shouldReply = false
			log.Printf("[Node %d] RA: My request has priority → DEFERRING reply to Node %d",
				myID, req.NodeId)
		}
	}

	if shouldReply {
		ms.hub.Events.Log(events.TypeRAReplySent, ms.hub.Clock.GetTime(), resource, "",
			fmt.Sprintf("Sent RA REPLY to Node %d for seat %s", req.NodeId, resource))

		return &pb.MutexResponse{
			Granted:     true,
			NodeId:      myID,
			LamportTime: ms.hub.Clock.GetTime(),
		}, nil
	}

	// Defer the reply
	ms.deferredReplies[resource] = append(ms.deferredReplies[resource], DeferredReply{
		NodeID:   req.NodeId,
		Resource: resource,
	})

	// Return granted=false to indicate deferred (the actual REPLY will come later)
	return &pb.MutexResponse{
		Granted:     false,
		NodeId:      myID,
		LamportTime: ms.hub.Clock.GetTime(),
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════
// REQUESTING CRITICAL SECTION (outgoing messages + wait logic)
// ═══════════════════════════════════════════════════════════════════════

// RequestCriticalSection requests access to the critical section for the given
// resource (seat). Blocks until access is granted (all replies received).
// Returns an error if the request times out or fails.
func (ms *MutexService) RequestCriticalSection(resource string) error {
	myID := ms.hub.NodeID()
	lamportTime := ms.hub.Clock.Tick()

	log.Printf("[Node %d] RA: *** REQUESTING critical section for seat %s *** (Lamport: %d)",
		myID, resource, lamportTime)

	ms.hub.SetState("waiting_cs")
	ms.hub.Events.Log(events.TypeRARequestSent, lamportTime, resource, "",
		fmt.Sprintf("Node %d requesting critical section for seat %s", myID, resource))

	// Get all current peers
	allPeers := ms.hub.GetAllPeerClients()
	numPeers := len(allPeers)

	// Initialize state for this resource
	ms.mu.Lock()
	ms.state[resource] = StateWanting
	ms.requestTime[resource] = lamportTime
	ms.repliesReceived[resource] = 0
	ms.repliesNeeded[resource] = numPeers
	waitCh := make(chan struct{}, 1)
	ms.waitCh[resource] = waitCh
	ms.mu.Unlock()

	if numPeers == 0 {
		// No peers — enter immediately
		log.Printf("[Node %d] RA: No peers, entering critical section immediately", myID)
		ms.mu.Lock()
		ms.state[resource] = StateHeld
		ms.mu.Unlock()
		ms.hub.SetState("in_cs")
		ms.hub.Events.Log(events.TypeRAEnterCS, ms.hub.Clock.GetTime(), resource, "",
			fmt.Sprintf("Node %d entered critical section for seat %s (no peers)", myID, resource))
		return nil
	}

	// Send REQUEST to all peers
	var wg sync.WaitGroup
	for peerID, clients := range allPeers {
		wg.Add(1)
		go func(id int32, c *node.PeerClients) {
			defer wg.Done()

			sendTime := ms.hub.Clock.SendTime()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			log.Printf("[Node %d] RA: Sending REQUEST to Node %d for seat %s (Lamport: %d)",
				myID, id, resource, sendTime)

			resp, err := c.Mutex.RequestAccess(ctx, &pb.MutexRequest{
				NodeId:      myID,
				LamportTime: lamportTime, // Use the original request time, not send time
				Resource:    resource,
			})

			if err != nil {
				log.Printf("[Node %d] RA: Failed to send REQUEST to Node %d: %v", myID, id, err)
				// Treat unreachable node as implicit REPLY
				ms.mu.Lock()
				ms.repliesReceived[resource]++
				if ms.repliesReceived[resource] >= ms.repliesNeeded[resource] {
					select {
					case waitCh <- struct{}{}:
					default:
					}
				}
				ms.mu.Unlock()
				return
			}

			if resp.Granted {
				log.Printf("[Node %d] RA: Received REPLY from Node %d for seat %s ✓",
					myID, id, resource)
				ms.hub.Events.Log(events.TypeRAReplyReceived, ms.hub.Clock.GetTime(), resource, "",
					fmt.Sprintf("Received RA REPLY from Node %d for seat %s", id, resource))

				ms.mu.Lock()
				ms.repliesReceived[resource]++
				if ms.repliesReceived[resource] >= ms.repliesNeeded[resource] {
					select {
					case waitCh <- struct{}{}:
					default:
					}
				}
				ms.mu.Unlock()
			} else {
				// Reply was deferred — we need to wait for the actual reply
				// In our implementation, the peer will send REPLY when it exits CS
				// We'll poll or use a retry mechanism
				log.Printf("[Node %d] RA: Node %d DEFERRED reply for seat %s, waiting...",
					myID, id, resource)
				ms.waitForDeferredReply(id, resource, lamportTime, waitCh)
			}
		}(peerID, clients)
	}

	// Wait for all REQUEST goroutines to complete their initial sends
	wg.Wait()

	// Wait for all replies (with timeout)
	select {
	case <-waitCh:
		// All replies received — enter critical section
		ms.mu.Lock()
		ms.state[resource] = StateHeld
		ms.mu.Unlock()

		ms.hub.SetState("in_cs")
		log.Printf("[Node %d] RA: *** ENTERING CRITICAL SECTION *** for seat %s (all %d replies received)",
			myID, resource, numPeers)

		ms.hub.Events.Log(events.TypeRAEnterCS, ms.hub.Clock.GetTime(), resource, "",
			fmt.Sprintf("Node %d entered critical section for seat %s", myID, resource))
		return nil

	case <-time.After(30 * time.Second):
		// Timeout — release and return error
		ms.mu.Lock()
		ms.state[resource] = StateReleased
		ms.mu.Unlock()
		ms.hub.SetState("running")
		return fmt.Errorf("timeout waiting for RA replies for seat %s", resource)
	}
}

// waitForDeferredReply polls the peer until it gets a REPLY (used when initial response was deferred).
func (ms *MutexService) waitForDeferredReply(peerID int32, resource string, lamportTime int64, waitCh chan struct{}) {
	myID := ms.hub.NodeID()

	for i := 0; i < 30; i++ { // Retry up to 30 times (1 second apart)
		time.Sleep(1 * time.Second)

		clients := ms.hub.GetPeerClients(peerID)
		if clients == nil {
			// Peer disconnected — treat as implicit reply
			break
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		resp, err := clients.Mutex.RequestAccess(ctx, &pb.MutexRequest{
			NodeId:      myID,
			LamportTime: lamportTime,
			Resource:    resource,
		})
		cancel()

		if err != nil {
			// Peer unreachable — treat as implicit reply
			break
		}

		if resp.Granted {
			log.Printf("[Node %d] RA: Received deferred REPLY from Node %d for seat %s ✓",
				myID, peerID, resource)

			ms.hub.Events.Log(events.TypeRAReplyReceived, ms.hub.Clock.GetTime(), resource, "",
				fmt.Sprintf("Received deferred RA REPLY from Node %d for seat %s", peerID, resource))

			ms.mu.Lock()
			ms.repliesReceived[resource]++
			if ms.repliesReceived[resource] >= ms.repliesNeeded[resource] {
				select {
				case waitCh <- struct{}{}:
				default:
				}
			}
			ms.mu.Unlock()
			return
		}
	}

	// Timeout — treat as implicit reply (avoid indefinite blocking)
	log.Printf("[Node %d] RA: Timeout waiting for deferred reply from Node %d, treating as granted",
		myID, peerID)
	ms.mu.Lock()
	ms.repliesReceived[resource]++
	if ms.repliesReceived[resource] >= ms.repliesNeeded[resource] {
		select {
		case waitCh <- struct{}{}:
		default:
		}
	}
	ms.mu.Unlock()
}

// ═══════════════════════════════════════════════════════════════════════
// RELEASING CRITICAL SECTION
// ═══════════════════════════════════════════════════════════════════════

// ReleaseCriticalSection exits the critical section for the given resource
// and sends all deferred REPLY messages.
func (ms *MutexService) ReleaseCriticalSection(resource string) {
	myID := ms.hub.NodeID()

	log.Printf("[Node %d] RA: *** EXITING CRITICAL SECTION *** for seat %s", myID, resource)

	ms.hub.Events.Log(events.TypeRAExitCS, ms.hub.Clock.GetTime(), resource, "",
		fmt.Sprintf("Node %d exited critical section for seat %s", myID, resource))

	ms.mu.Lock()
	ms.state[resource] = StateReleased
	deferred := ms.deferredReplies[resource]
	ms.deferredReplies[resource] = nil
	delete(ms.requestTime, resource)
	delete(ms.repliesReceived, resource)
	delete(ms.repliesNeeded, resource)
	delete(ms.waitCh, resource)
	ms.mu.Unlock()

	ms.hub.SetState("running")

	// Send deferred replies
	for _, dr := range deferred {
		go func(d DeferredReply) {
			clients := ms.hub.GetPeerClients(d.NodeID)
			if clients == nil {
				return
			}

			log.Printf("[Node %d] RA: Sending deferred REPLY to Node %d for seat %s",
				myID, d.NodeID, d.Resource)

			ms.hub.Events.Log(events.TypeRAReplySent, ms.hub.Clock.GetTime(), d.Resource, "",
				fmt.Sprintf("Sent deferred RA REPLY to Node %d for seat %s", d.NodeID, d.Resource))

			// Note: The deferred reply is implicit — the next time the peer
			// polls us with RequestAccess, our state will be RELEASED
			// and we'll immediately grant the reply.
		}(dr)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// STATUS QUERIES (for dashboard)
// ═══════════════════════════════════════════════════════════════════════

// GetState returns the RA state for a given resource.
func (ms *MutexService) GetState(resource string) string {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	switch ms.state[resource] {
	case StateWanting:
		return "WANTING"
	case StateHeld:
		return "HELD"
	default:
		return "RELEASED"
	}
}

// ResourceState holds detailed info about the mutex state of a resource
type ResourceState struct {
	State           string  `json:"state"`
	RequestTime     int64   `json:"request_time,omitempty"`
	DeferredNodeIDs []int32 `json:"deferred_nodes,omitempty"`
}

// GetAllStates returns RA states for all active resources.
func (ms *MutexService) GetAllStates() map[string]string {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	result := make(map[string]string)
	for resource, state := range ms.state {
		switch state {
		case StateWanting:
			result[resource] = "WANTING"
		case StateHeld:
			result[resource] = "HELD"
		case StateReleased:
			result[resource] = "RELEASED"
		}
	}
	return result
}

// GetDetailedStates returns detailed RA info for all active resources.
func (ms *MutexService) GetDetailedStates() map[string]ResourceState {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	result := make(map[string]ResourceState)
	for resource, state := range ms.state {
		s := "RELEASED"
		switch state {
		case StateWanting:
			s = "WANTING"
		case StateHeld:
			s = "HELD"
		}

		deferredIDs := []int32{}
		for _, dr := range ms.deferredReplies[resource] {
			deferredIDs = append(deferredIDs, dr.NodeID)
		}

		result[resource] = ResourceState{
			State:           s,
			RequestTime:     ms.requestTime[resource],
			DeferredNodeIDs: deferredIDs,
		}
	}
	return result
}
