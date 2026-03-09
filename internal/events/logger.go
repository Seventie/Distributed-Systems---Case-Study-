// Package events provides an event logging system for tracking all distributed
// system activity. Events are stored in-memory for the dashboard, written to a
// JSON-lines log file for Apache Beam analytics, and exposed via the API.
//
// Event Types:
//
//	NODE_STARTED           - Node has started
//	BOOKING_REQUEST        - A booking was requested
//	BOOKING_SUCCESS        - Booking succeeded
//	BOOKING_FAILED         - Booking failed (seat taken or error)
//	ELECTION_START         - Bully election initiated
//	ELECTION_OK            - OK message in Bully algorithm
//	ELECTION_COORDINATOR   - New coordinator announced
//	LEADER_FAILURE         - Leader failure detected
//	RA_REQUEST_SENT        - Ricart-Agrawala request sent to peers
//	RA_REQUEST_RECEIVED    - RA request received from a peer
//	RA_REPLY_SENT          - RA reply sent to a peer
//	RA_REPLY_RECEIVED      - RA reply received from a peer
//	RA_ENTER_CS            - Entered critical section
//	RA_EXIT_CS             - Exited critical section
//	HEARTBEAT_SENT         - Heartbeat sent by leader
//	HEARTBEAT_RECEIVED     - Heartbeat received from leader
//	SEAT_SYNCED            - Seat state synchronized from another node
package events

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Event type constants
const (
	TypeNodeStarted       = "NODE_STARTED"
	TypeBookingRequest    = "BOOKING_REQUEST"
	TypeBookingSuccess    = "BOOKING_SUCCESS"
	TypeBookingFailed     = "BOOKING_FAILED"
	TypeElectionStart     = "ELECTION_START"
	TypeElectionOK        = "ELECTION_OK"
	TypeElectionCoord     = "ELECTION_COORDINATOR"
	TypeLeaderFailure     = "LEADER_FAILURE"
	TypeRARequestSent     = "RA_REQUEST_SENT"
	TypeRARequestReceived = "RA_REQUEST_RECEIVED"
	TypeRAReplySent       = "RA_REPLY_SENT"
	TypeRAReplyReceived   = "RA_REPLY_RECEIVED"
	TypeRAEnterCS         = "RA_ENTER_CS"
	TypeRAExitCS          = "RA_EXIT_CS"
	TypeHeartbeatSent     = "HEARTBEAT_SENT"
	TypeHeartbeatReceived = "HEARTBEAT_RECEIVED"
	TypeSeatSynced        = "SEAT_SYNCED"
)

// Event represents a single logged event in the distributed system.
type Event struct {
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	NodeID      int32  `json:"node_id"`
	SeatID      string `json:"seat_id,omitempty"`
	UserName    string `json:"user_name,omitempty"`
	LamportTime int64  `json:"lamport_time"`
	Timestamp   int64  `json:"timestamp"` // Unix milliseconds
	Details     string `json:"details"`
}

// Logger stores events in-memory and writes them to a JSON-lines file.
type Logger struct {
	mu       sync.RWMutex
	nodeID   int32
	events   []Event
	logFile  *os.File
	maxInMem int // max events kept in memory (ring buffer style)

	// Subscribers receive new events for real-time dashboard updates.
	subMu       sync.RWMutex
	subscribers []chan Event
}

// NewLogger creates a new event logger for the given node.
// Events are also written to logs/node<id>_events.jsonl for Beam analytics.
func NewLogger(nodeID int32) *Logger {
	logFileName := fmt.Sprintf("logs/node%d_events.jsonl", nodeID)

	// Ensure logs directory exists
	_ = os.MkdirAll("logs", 0755)

	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("WARNING: Could not open log file %s: %v\n", logFileName, err)
	}

	return &Logger{
		nodeID:      nodeID,
		events:      make([]Event, 0, 1000),
		logFile:     f,
		maxInMem:    500,
		subscribers: make([]chan Event, 0),
	}
}

// Log records a new event with the given parameters.
func (l *Logger) Log(eventType string, lamportTime int64, seatID, userName, details string) Event {
	evt := Event{
		EventID:     uuid.New().String(),
		EventType:   eventType,
		NodeID:      l.nodeID,
		SeatID:      seatID,
		UserName:    userName,
		LamportTime: lamportTime,
		Timestamp:   time.Now().UnixMilli(),
		Details:     details,
	}

	l.mu.Lock()
	// Ring buffer: remove oldest if at capacity
	if len(l.events) >= l.maxInMem {
		l.events = l.events[1:]
	}
	l.events = append(l.events, evt)
	l.mu.Unlock()

	// Write to log file (non-blocking, best effort)
	if l.logFile != nil {
		data, err := json.Marshal(evt)
		if err == nil {
			l.logFile.Write(append(data, '\n'))
		}
	}

	// Notify subscribers (non-blocking)
	l.subMu.RLock()
	for _, ch := range l.subscribers {
		select {
		case ch <- evt:
		default:
			// Drop if subscriber is too slow
		}
	}
	l.subMu.RUnlock()

	return evt
}

// LogSimple logs an event with just type and details (no seat/user context).
func (l *Logger) LogSimple(eventType string, lamportTime int64, details string) Event {
	return l.Log(eventType, lamportTime, "", "", details)
}

// GetRecent returns the most recent n events (or all if n > total).
func (l *Logger) GetRecent(n int) []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()

	total := len(l.events)
	if n > total {
		n = total
	}
	// Return a copy of the most recent events
	result := make([]Event, n)
	copy(result, l.events[total-n:])
	return result
}

// GetAll returns all events currently in memory.
func (l *Logger) GetAll() []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]Event, len(l.events))
	copy(result, l.events)
	return result
}

// GetByType returns all events of a specific type.
func (l *Logger) GetByType(eventType string) []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var result []Event
	for _, evt := range l.events {
		if evt.EventType == eventType {
			result = append(result, evt)
		}
	}
	return result
}

// Subscribe returns a channel that receives new events in real time.
// The channel has a buffer of 100 events.
func (l *Logger) Subscribe() chan Event {
	ch := make(chan Event, 100)
	l.subMu.Lock()
	l.subscribers = append(l.subscribers, ch)
	l.subMu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (l *Logger) Unsubscribe(ch chan Event) {
	l.subMu.Lock()
	defer l.subMu.Unlock()
	for i, sub := range l.subscribers {
		if sub == ch {
			l.subscribers = append(l.subscribers[:i], l.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// Close flushes and closes the log file.
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
