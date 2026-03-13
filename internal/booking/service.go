// Package booking implements the ticket booking service.
//
// The booking flow is:
//  1. User submits a booking request via the HTTP dashboard.
//  2. The receiving node uses the Ricart-Agrawala algorithm to acquire
//     mutual exclusion for the requested seat.
//  3. Once in the critical section, the node checks seat availability.
//  4. If available, it books the seat and broadcasts a SyncSeat update
//     to all peers to keep seat state consistent.
//  5. The node exits the critical section (releasing deferred RA replies).
//  6. A BookingEvent is logged for analytics.
//
// This service also implements the gRPC BookingService for inter-node
// booking request forwarding and seat synchronization.
package booking

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"distributed-ticket-system/internal/events"
	"distributed-ticket-system/internal/node"
	"distributed-ticket-system/internal/ra"
	"distributed-ticket-system/internal/seats"
	pb "distributed-ticket-system/proto/ticket"
)

// BookingResult is the result of a booking attempt (for the dashboard API).
type BookingResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	SeatID   string `json:"seat_id"`
	UserName string `json:"user_name"`
	NodeID   int32  `json:"node_id"`
}

// Service handles ticket booking logic with distributed mutual exclusion.
type Service struct {
	pb.UnimplementedBookingServiceServer

	hub   *node.Hub
	mutex *ra.MutexService
	seats *seats.Store
}

// NewService creates a new booking service.
func NewService(hub *node.Hub, mutex *ra.MutexService, seatStore *seats.Store) *Service {
	return &Service{
		hub:   hub,
		mutex: mutex,
		seats: seatStore,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// PUBLIC API (called by the dashboard HTTP handler)
// ═══════════════════════════════════════════════════════════════════════

// BookSeat handles a booking request from the user.
// This is the main entry point for the booking flow.
func (s *Service) BookSeat(userName, seatID string) BookingResult {
	myID := s.hub.NodeID()
	lamportTime := s.hub.Clock.Tick()

	log.Printf("[Node %d] BOOKING: User '%s' requesting seat '%s' (Lamport: %d)",
		myID, userName, seatID, lamportTime)

	s.hub.Events.Log(events.TypeBookingRequest, lamportTime, seatID, userName,
		fmt.Sprintf("User '%s' requesting seat '%s' via Node %d", userName, seatID, myID))

	// Validation
	if !s.seats.SeatExists(seatID) {
		msg := fmt.Sprintf("Seat '%s' does not exist", seatID)
		s.hub.Events.Log(events.TypeBookingFailed, s.hub.Clock.Tick(), seatID, userName, msg)
		return BookingResult{Success: false, Message: msg, SeatID: seatID, UserName: userName, NodeID: myID}
	}

	// Quick check: is the seat already booked? (optimization, not authoritative)
	if !s.seats.IsAvailable(seatID) {
		msg := fmt.Sprintf("Seat '%s' is already booked", seatID)
		s.hub.Events.Log(events.TypeBookingFailed, s.hub.Clock.Tick(), seatID, userName, msg)
		return BookingResult{Success: false, Message: msg, SeatID: seatID, UserName: userName, NodeID: myID}
	}

	// Step 1: Request critical section via Ricart-Agrawala
	log.Printf("[Node %d] BOOKING: Requesting critical section for seat '%s'...", myID, seatID)
	err := s.mutex.RequestCriticalSection(seatID)
	if err != nil {
		msg := fmt.Sprintf("Failed to acquire mutex for seat '%s': %v", seatID, err)
		log.Printf("[Node %d] BOOKING: %s", myID, msg)
		s.hub.Events.Log(events.TypeBookingFailed, s.hub.Clock.Tick(), seatID, userName, msg)
		return BookingResult{Success: false, Message: msg, SeatID: seatID, UserName: userName, NodeID: myID}
	}

	// Step 2: We are now in the critical section — book the seat
	// Double-check availability (authoritative check under mutex)
	success := s.seats.Book(seatID, userName)

	if success {
		lamportNow := s.hub.Clock.Tick()

		log.Printf("[Node %d] BOOKING: *** SEAT '%s' BOOKED for '%s' *** (Lamport: %d)",
			myID, seatID, userName, lamportNow)

		s.hub.Events.Log(events.TypeBookingSuccess, lamportNow, seatID, userName,
			fmt.Sprintf("Seat '%s' booked for '%s' by Node %d", seatID, userName, myID))

		// Step 3: Sync the seat update to all peers *BEFORE* releasing the mutex
		// This ensures consistency across the cluster before anyone else can try to book it.
		s.syncSeatToAllPeers(seatID, seats.StatusBooked, userName)

		// Step 4: Release critical section
		s.mutex.ReleaseCriticalSection(seatID)

		return BookingResult{
			Success:  true,
			Message:  fmt.Sprintf("Seat '%s' successfully booked for '%s'!", seatID, userName),
			SeatID:   seatID,
			UserName: userName,
			NodeID:   myID,
		}
	}

	// Step 4: Release critical section even if booking failed
	s.mutex.ReleaseCriticalSection(seatID)

	// Seat was booked between our initial check and critical section entry
	msg := fmt.Sprintf("Seat '%s' was already booked by another node", seatID)
	log.Printf("[Node %d] BOOKING: %s", myID, msg)
	s.hub.Events.Log(events.TypeBookingFailed, s.hub.Clock.Tick(), seatID, userName, msg)
	return BookingResult{Success: false, Message: msg, SeatID: seatID, UserName: userName, NodeID: myID}
}

// ReleaseSeat releases a booked seat and syncs it across the cluster.
func (s *Service) ReleaseSeat(seatID string) BookingResult {
	myID := s.hub.NodeID()
	
	if !s.seats.SeatExists(seatID) {
		return BookingResult{Success: false, Message: "Seat not found", SeatID: seatID, NodeID: myID}
	}

	success := s.seats.Release(seatID)
	if success {
		log.Printf("[Node %d] RELEASE: Seat '%s' released", myID, seatID)
		s.hub.Events.LogSimple(events.TypeSeatSynced, s.hub.Clock.Tick(), 
			fmt.Sprintf("Seat '%s' released by Node %d", seatID, myID))
		
		// Sync to all peers
		s.syncSeatToAllPeers(seatID, seats.StatusAvailable, "")
		return BookingResult{Success: true, Message: "Seat released", SeatID: seatID, NodeID: myID}
	}
	
	return BookingResult{Success: false, Message: "Failed to release seat", SeatID: seatID, NodeID: myID}
}

// ResetSeats resets all seats and syncs to all peers.
func (s *Service) ResetSeats() {
	s.seats.Reset()
	s.hub.Events.LogSimple(events.TypeSeatSynced, s.hub.Clock.Tick(), "All seats reset to available")
	
	allSeats := s.seats.GetAllSeats()
	for _, seat := range allSeats {
		s.syncSeatToAllPeers(seat.ID, seats.StatusAvailable, "")
	}
}

// syncSeatToAllPeers broadcasts a seat update to all connected peers and waits for them to acknowledge.
func (s *Service) syncSeatToAllPeers(seatID, status, bookedBy string) {
	myID := s.hub.NodeID()
	// Get all peer stubs (Hub now returns offline ones too)
	allPeers := s.hub.GetAllPeerClients()

	if len(allPeers) == 0 && len(s.hub.Config.Peers) == 0 {
		return
	}

	var wg sync.WaitGroup
	for peerID, clients := range allPeers {
		wg.Add(1)
		go func(id int32, c *node.PeerClients) {
			defer wg.Done()

			sendTime := s.hub.Clock.SendTime()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			log.Printf("[Node %d] Syncing seat '%s' to Node %d", myID, seatID, id)

			_, err := c.Booking.SyncSeat(ctx, &pb.SeatUpdate{
				SeatId:      seatID,
				Status:      status,
				BookedBy:    bookedBy,
				NodeId:      myID,
				LamportTime: sendTime,
			})
			if err != nil {
				log.Printf("[Node %d] Failed to sync seat '%s' to Node %d: %v", myID, seatID, id, err)
			}
		}(peerID, clients)
	}

	// Wait for all sync requests to complete (or time out)
	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════════════
// gRPC SERVER HANDLERS (incoming messages from peers)
// ═══════════════════════════════════════════════════════════════════════

// RequestBooking handles an incoming booking request from another node.
// In this architecture, each node can accept bookings directly from users.
// This gRPC method is for forwarding requests (e.g., to the leader).
func (s *Service) RequestBooking(ctx context.Context, req *pb.BookingRequest) (*pb.BookingResponse, error) {
	myID := s.hub.NodeID()
	s.hub.Clock.ReceiveTime(req.LamportTime)

	log.Printf("[Node %d] Received booking request from Node %d: user='%s' seat='%s'",
		myID, req.RequestingNode, req.UserName, req.SeatId)

	result := s.BookSeat(req.UserName, req.SeatId)

	return &pb.BookingResponse{
		Success:      result.Success,
		Message:      result.Message,
		SeatId:       result.SeatID,
		BookedByNode: result.NodeID,
	}, nil
}

// SyncSeat handles an incoming seat synchronization update from another node.
// This keeps the local seat store consistent with the rest of the cluster.
func (s *Service) SyncSeat(ctx context.Context, req *pb.SeatUpdate) (*pb.SeatUpdateResponse, error) {
	myID := s.hub.NodeID()
	s.hub.Clock.ReceiveTime(req.LamportTime)

	log.Printf("[Node %d] Received SEAT SYNC: seat='%s' status='%s' bookedBy='%s' from Node %d (Lamport: %d)",
		myID, req.SeatId, req.Status, req.BookedBy, req.NodeId, req.LamportTime)

	s.seats.ForceUpdate(req.SeatId, req.Status, req.BookedBy)

	s.hub.Events.Log(events.TypeSeatSynced, s.hub.Clock.GetTime(), req.SeatId, req.BookedBy,
		fmt.Sprintf("Seat '%s' synced from Node %d: %s by '%s'", req.SeatId, req.NodeId, req.Status, req.BookedBy))

	return &pb.SeatUpdateResponse{Acknowledged: true}, nil
}

// GetAllSeats returns all seats and their current status.
func (s *Service) GetAllSeats(ctx context.Context, req *pb.SeatsRequest) (*pb.SeatsResponse, error) {
	allSeats := s.seats.GetAllSeats()
	pbSeats := make([]*pb.SeatInfo, len(allSeats))
	for i, seat := range allSeats {
		pbSeats[i] = &pb.SeatInfo{
			SeatId:   seat.ID,
			Status:   seat.Status,
			BookedBy: seat.BookedBy,
		}
	}
	return &pb.SeatsResponse{Seats: pbSeats}, nil
}
