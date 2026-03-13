// Package node provides the Hub — the central coordinator for a single node
// in the distributed ticket booking system.
//
// The Hub aggregates all shared state and services:
//   - Configuration (node ID, peer list)
//   - Lamport Clock (for event ordering)
//   - Event Logger (for tracking all system events)
//   - Seat Store (for ticket availability)
//   - Peer gRPC connections (for inter-node communication)
//   - Leader tracking (for Bully algorithm results)
//
// All algorithm services (Bully, Ricart-Agrawala, Booking) receive a pointer
// to the Hub to access shared resources and peer connections.
package node

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"distributed-ticket-system/internal/clock"
	"distributed-ticket-system/internal/config"
	"distributed-ticket-system/internal/events"
	"distributed-ticket-system/internal/seats"
	pb "distributed-ticket-system/proto/ticket"
)

// PeerClients bundles all gRPC client stubs for a single peer node.
type PeerClients struct {
	Election pb.ElectionServiceClient
	Mutex    pb.MutexServiceClient
	Booking  pb.BookingServiceClient
}

// Hub is the central coordinator for a single node.
type Hub struct {
	Config *config.Config
	Clock  *clock.LamportClock
	Events *events.Logger
	Seats  *seats.Store

	mu          sync.RWMutex
	leaderID    int32
	isAlive     bool
	state       string // "running", "election", "waiting_cs", "in_cs"
	peerConns   map[int32]*grpc.ClientConn
	peerClients map[int32]*PeerClients
	peerAlive   map[int32]bool
}

// NewHub creates a new Hub with the given configuration and services.
func NewHub(cfg *config.Config) *Hub {
	lc := clock.NewLamportClock()
	el := events.NewLogger(cfg.Node.ID)
	ss := seats.NewStore(cfg.Seats.Rows, cfg.Seats.ColsPerRow)

	h := &Hub{
		Config:      cfg,
		Clock:       lc,
		Events:      el,
		Seats:       ss,
		leaderID:    -1, // No leader initially
		isAlive:     true,
		state:       "running",
		peerConns:   make(map[int32]*grpc.ClientConn),
		peerClients: make(map[int32]*PeerClients),
		peerAlive:   make(map[int32]bool),
	}

	return h
}

// NodeID returns this node's ID.
func (h *Hub) NodeID() int32 {
	return h.Config.Node.ID
}

// ---------- Leader Management ----------

// SetLeader updates the known leader ID.
func (h *Hub) SetLeader(leaderID int32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.leaderID = leaderID
	log.Printf("[Node %d] Leader set to Node %d", h.Config.Node.ID, leaderID)
}

// GetLeader returns the current leader's node ID (-1 if unknown).
func (h *Hub) GetLeader() int32 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.leaderID
}

// IsLeader returns true if this node is the current leader.
func (h *Hub) IsLeader() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.leaderID == h.Config.Node.ID
}

// ---------- State Management ----------

// SetState updates the node's current operational state.
func (h *Hub) SetState(state string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.state = state
}

// GetState returns the node's current operational state.
func (h *Hub) GetState() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.state
}

// ---------- Peer Connection Management ----------

// ConnectToPeers establishes gRPC connections to all configured peers concurrently.
// It also starts a background goroutine to periodically retry failed connections.
func (h *Hub) ConnectToPeers() {
	for _, peer := range h.Config.Peers {
		go h.connectWithRetry(peer)
	}
}

func (h *Hub) connectWithRetry(p config.PeerConfig) {
	addr := p.PeerAddress()
	for h.isAlive {
		h.mu.RLock()
		alreadyConnected := h.peerAlive[p.ID]
		h.mu.RUnlock()

		if !alreadyConnected {
			log.Printf("[Node %d] Attempting to connect to peer Node %d at %s...", h.Config.Node.ID, p.ID, addr)

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			conn, err := grpc.DialContext(ctx, addr,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			cancel()

			if err == nil {
				h.mu.Lock()
				if old, ok := h.peerConns[p.ID]; ok {
					old.Close()
				}
				h.peerConns[p.ID] = conn
				h.peerClients[p.ID] = &PeerClients{
					Election: pb.NewElectionServiceClient(conn),
					Mutex:    pb.NewMutexServiceClient(conn),
					Booking:  pb.NewBookingServiceClient(conn),
				}
				h.peerAlive[p.ID] = true
				h.mu.Unlock()
				log.Printf("[Node %d] Successfully connected to peer Node %d", h.Config.Node.ID, p.ID)
			} else {
				log.Printf("[Node %d] Connection to Node %d failed, will retry: %v", h.Config.Node.ID, p.ID, err)
			}
		}
		// Wait before next retry attempt
		time.Sleep(5 * time.Second)
	}
}

// ConnectToPeer establishes or re-establishes a connection to a specific peer.
func (h *Hub) ConnectToPeer(nodeID int32) error {
	peer := h.Config.GetPeer(nodeID)
	if peer == nil {
		return fmt.Errorf("unknown peer: %d", nodeID)
	}

	addr := peer.PeerAddress()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		h.mu.Lock()
		h.peerAlive[nodeID] = false
		h.mu.Unlock()
		return fmt.Errorf("failed to connect to Node %d: %w", nodeID, err)
	}

	h.mu.Lock()
	// Close old connection if exists
	if old, ok := h.peerConns[nodeID]; ok {
		old.Close()
	}
	h.peerConns[nodeID] = conn
	h.peerClients[nodeID] = &PeerClients{
		Election: pb.NewElectionServiceClient(conn),
		Mutex:    pb.NewMutexServiceClient(conn),
		Booking:  pb.NewBookingServiceClient(conn),
	}
	h.peerAlive[nodeID] = true
	h.mu.Unlock()

	return nil
}

// GetPeerClients returns the gRPC clients for a specific peer, or nil if not connected.
func (h *Hub) GetPeerClients(nodeID int32) *PeerClients {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.peerClients[nodeID]
}

// GetAllPeerClients returns a map of all peer clients (including offline ones).
func (h *Hub) GetAllPeerClients() map[int32]*PeerClients {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[int32]*PeerClients)
	for id, clients := range h.peerClients {
		result[id] = clients
	}
	return result
}

// GetHigherPeerClients returns clients for peers with higher node IDs (for Bully).
func (h *Hub) GetHigherPeerClients() map[int32]*PeerClients {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[int32]*PeerClients)
	for id, clients := range h.peerClients {
		if id > h.Config.Node.ID && h.peerAlive[id] {
			result[id] = clients
		}
	}
	return result
}

// ---------- Peer Status ----------

// SetPeerAlive updates the alive status of a peer.
func (h *Hub) SetPeerAlive(nodeID int32, alive bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peerAlive[nodeID] = alive
}

// IsPeerAlive checks if a peer is currently marked as alive.
func (h *Hub) IsPeerAlive(nodeID int32) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.peerAlive[nodeID]
}

// GetPeerStatuses returns a map of peer ID -> alive status.
func (h *Hub) GetPeerStatuses() map[int32]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[int32]bool)
	for id, alive := range h.peerAlive {
		result[id] = alive
	}
	return result
}

// ---------- Cleanup ----------

// Close gracefully shuts down all peer connections and the event logger.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, conn := range h.peerConns {
		log.Printf("[Node %d] Closing connection to Node %d", h.Config.Node.ID, id)
		conn.Close()
	}
	h.Events.Close()
}
