// Package seats provides an in-memory seat store for the ticket booking system.
//
// The seat store manages seat availability and booking state.
// Each node maintains its own copy of the seat store, which is kept in
// sync via SyncSeat gRPC calls after a booking is confirmed.
//
// Seat IDs follow the format: "A1", "A2", "B1", "B2", etc.
// Status can be "available" or "booked".
package seats

import (
	"fmt"
	"sort"
	"sync"
)

// Status constants for seat states.
const (
	StatusAvailable = "available"
	StatusBooked    = "booked"
)

// Seat represents a single seat in the venue.
type Seat struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	BookedBy string `json:"booked_by"` // user name who booked, empty if available
}

// Store is a thread-safe in-memory seat store.
type Store struct {
	mu    sync.RWMutex
	seats map[string]*Seat
	order []string // maintains insertion order for consistent display
}

// NewStore creates a seat store with seats generated from the given rows and columns.
// Example: rows=["A","B","C"], colsPerRow=4 → seats A1..A4, B1..B4, C1..C4
func NewStore(rows []string, colsPerRow int) *Store {
	s := &Store{
		seats: make(map[string]*Seat),
		order: make([]string, 0),
	}
	for _, row := range rows {
		for col := 1; col <= colsPerRow; col++ {
			id := fmt.Sprintf("%s%d", row, col)
			s.seats[id] = &Seat{
				ID:     id,
				Status: StatusAvailable,
			}
			s.order = append(s.order, id)
		}
	}
	return s
}

// GetSeat returns a copy of a seat's current state, or nil if not found.
func (s *Store) GetSeat(seatID string) *Seat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seat, exists := s.seats[seatID]
	if !exists {
		return nil
	}
	// Return a copy to avoid race conditions
	copy := *seat
	return &copy
}

// GetAllSeats returns a copy of all seats in display order.
func (s *Store) GetAllSeats() []Seat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Seat, 0, len(s.order))
	for _, id := range s.order {
		if seat, ok := s.seats[id]; ok {
			result = append(result, *seat)
		}
	}
	return result
}

// IsAvailable checks if a seat exists and is available for booking.
func (s *Store) IsAvailable(seatID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seat, exists := s.seats[seatID]
	return exists && seat.Status == StatusAvailable
}

// Book attempts to book a seat for the given user.
// Returns true if successful, false if the seat is already booked or doesn't exist.
// This should only be called while holding the distributed mutex (Ricart-Agrawala).
func (s *Store) Book(seatID, userName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	seat, exists := s.seats[seatID]
	if !exists || seat.Status == StatusBooked {
		return false
	}
	seat.Status = StatusBooked
	seat.BookedBy = userName
	return true
}

// Release marks a seat as available again.
func (s *Store) Release(seatID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	seat, exists := s.seats[seatID]
	if !exists {
		return false
	}
	seat.Status = StatusAvailable
	seat.BookedBy = ""
	return true
}

// ForceUpdate updates a seat's state directly (used for sync from other nodes).
func (s *Store) ForceUpdate(seatID, status, bookedBy string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seat, exists := s.seats[seatID]
	if !exists {
		return
	}
	seat.Status = status
	seat.BookedBy = bookedBy
}

// AvailableCount returns the number of available seats.
func (s *Store) AvailableCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, seat := range s.seats {
		if seat.Status == StatusAvailable {
			count++
		}
	}
	return count
}

// BookedCount returns the number of booked seats.
func (s *Store) BookedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, seat := range s.seats {
		if seat.Status == StatusBooked {
			count++
		}
	}
	return count
}

// GetAvailableSeats returns a sorted list of available seat IDs.
func (s *Store) GetAvailableSeats() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, 0)
	for _, id := range s.order {
		if seat, ok := s.seats[id]; ok && seat.Status == StatusAvailable {
			result = append(result, id)
		}
	}
	sort.Strings(result)
	return result
}

// SeatExists checks whether a seat ID is valid.
func (s *Store) SeatExists(seatID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.seats[seatID]
	return exists
}

// Reset resets all seats to available (for testing).
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, seat := range s.seats {
		seat.Status = StatusAvailable
		seat.BookedBy = ""
	}
}
