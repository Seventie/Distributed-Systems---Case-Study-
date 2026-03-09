// Package clock implements Lamport Logical Clocks for distributed event ordering.
//
// Lamport Clock Rules:
//  1. Before each local event, increment the clock: time++
//  2. When sending a message, increment the clock and attach the timestamp.
//  3. When receiving a message with timestamp t:
//     local_time = max(local_time, t) + 1
//
// This ensures a partial causal ordering of events across all nodes.
// If event A causally precedes event B, then L(A) < L(B).
// However, L(A) < L(B) does NOT necessarily mean A caused B.
package clock

import (
	"sync"
)

// LamportClock is a thread-safe Lamport logical clock.
type LamportClock struct {
	mu   sync.Mutex
	time int64
}

// NewLamportClock creates a new Lamport clock initialized to 0.
func NewLamportClock() *LamportClock {
	return &LamportClock{time: 0}
}

// Tick increments the clock by 1 (local event) and returns the new time.
// Use this before performing any local event (e.g., booking a seat, starting election).
func (lc *LamportClock) Tick() int64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.time++
	return lc.time
}

// Update updates the clock on receiving a message with the given remote timestamp.
// Implements: local_time = max(local_time, received_time) + 1
// Returns the new local time.
func (lc *LamportClock) Update(receivedTime int64) int64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if receivedTime > lc.time {
		lc.time = receivedTime
	}
	lc.time++
	return lc.time
}

// GetTime returns the current clock value without incrementing.
func (lc *LamportClock) GetTime() int64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.time
}

// SendTime increments the clock (preparing to send) and returns the timestamp
// that should be attached to the outgoing message.
func (lc *LamportClock) SendTime() int64 {
	return lc.Tick()
}

// ReceiveTime processes an incoming message timestamp and returns new local time.
// Alias for Update — use whichever reads better in context.
func (lc *LamportClock) ReceiveTime(remoteTime int64) int64 {
	return lc.Update(remoteTime)
}
