// Package dashboard provides HTTP API endpoints for the web frontend.
//
// The dashboard API serves:
//   - Static HTML/CSS/JS files for the web UI
//   - REST endpoints for booking, seat status, node status, events, analytics
//   - Server-Sent Events (SSE) endpoint for real-time event streaming
//
// All endpoints return JSON responses (except the SSE stream and static files).
package dashboard

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"distributed-ticket-system/internal/booking"
	"distributed-ticket-system/internal/bully"
	"distributed-ticket-system/internal/events"
	"distributed-ticket-system/internal/node"
	"distributed-ticket-system/internal/ra"
	"distributed-ticket-system/internal/seats"
)

// API serves the HTTP dashboard and REST endpoints.
type API struct {
	hub     *node.Hub
	booking *booking.Service
	bully   *bully.ElectionService
	ra      *ra.MutexService
	events  *events.Logger
	seats   *seats.Store
}

// NewAPI creates a new dashboard API handler.
func NewAPI(
	hub *node.Hub,
	bookingSvc *booking.Service,
	bullySvc *bully.ElectionService,
	raSvc *ra.MutexService,
	evtLogger *events.Logger,
	seatStore *seats.Store,
) *API {
	return &API{
		hub:     hub,
		booking: bookingSvc,
		bully:   bullySvc,
		ra:      raSvc,
		events:  evtLogger,
		seats:   seatStore,
	}
}

// Start begins serving the dashboard HTTP server on the given port.
func (a *API) Start(port int) {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/book", a.corsMiddleware(a.handleBook))
	mux.HandleFunc("/api/seats", a.corsMiddleware(a.handleSeats))
	mux.HandleFunc("/api/status", a.corsMiddleware(a.handleStatus))
	mux.HandleFunc("/api/events", a.corsMiddleware(a.handleEvents))
	mux.HandleFunc("/api/analytics", a.corsMiddleware(a.handleAnalytics))
	mux.HandleFunc("/api/election", a.corsMiddleware(a.handleTriggerElection))
	mux.HandleFunc("/api/stream", a.handleSSE)

	// Static files (HTML, CSS, JS)
	staticDir := filepath.Join("web", "static")
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fs)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("[Node %d] Dashboard HTTP server starting on %s", a.hub.NodeID(), addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("[Node %d] Dashboard HTTP server error: %v", a.hub.NodeID(), err)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// MIDDLEWARE
// ═══════════════════════════════════════════════════════════════════════

func (a *API) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ═══════════════════════════════════════════════════════════════════════
// API HANDLERS
// ═══════════════════════════════════════════════════════════════════════

// POST /api/book — Submit a booking request
// Body: {"user_name": "Alice", "seat_id": "A1"}
func (a *API) handleBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	var req struct {
		UserName string `json:"user_name"`
		SeatID   string `json:"seat_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.UserName == "" || req.SeatID == "" {
		writeError(w, http.StatusBadRequest, "user_name and seat_id are required")
		return
	}

	result := a.booking.BookSeat(req.UserName, req.SeatID)
	writeJSON(w, result)
}

// GET /api/seats — Get all seats and their status
func (a *API) handleSeats(w http.ResponseWriter, r *http.Request) {
	allSeats := a.seats.GetAllSeats()
	resp := map[string]interface{}{
		"seats":           allSeats,
		"available_count": a.seats.AvailableCount(),
		"booked_count":    a.seats.BookedCount(),
	}
	writeJSON(w, resp)
}

// GET /api/status — Get node status (leader, peers, clocks, RA states)
func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	peerStatuses := a.hub.GetPeerStatuses()
	peers := make([]map[string]interface{}, 0)
	for id, alive := range peerStatuses {
		peers = append(peers, map[string]interface{}{
			"node_id": id,
			"alive":   alive,
		})
	}

	resp := map[string]interface{}{
		"node_id":          a.hub.NodeID(),
		"leader_id":        a.hub.GetLeader(),
		"is_leader":        a.hub.IsLeader(),
		"state":            a.hub.GetState(),
		"lamport_time":     a.hub.Clock.GetTime(),
		"election_running": a.bully.IsElectionRunning(),
		"ra_states":        a.ra.GetDetailedStates(),
		"peers":            peers,
	}
	writeJSON(w, resp)
}

// GET /api/events?limit=50&type=BOOKING_SUCCESS — Get recent events
func (a *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	eventType := r.URL.Query().Get("type")

	limit := 100
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	var evts []events.Event
	if eventType != "" {
		evts = a.events.GetByType(eventType)
	} else {
		evts = a.events.GetRecent(limit)
	}

	writeJSON(w, map[string]interface{}{
		"events": evts,
		"count":  len(evts),
	})
}

// GET /api/analytics — Get booking analytics summary
func (a *API) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	allEvents := a.events.GetAll()

	// Compute analytics from in-memory events
	bookingsPerSeat := make(map[string]int)
	bookingsPerNode := make(map[int32]int)
	successCount := 0
	failCount := 0
	requestCount := 0
	electionCount := 0
	raRequestCount := 0

	for _, evt := range allEvents {
		switch evt.EventType {
		case events.TypeBookingRequest:
			requestCount++
			if evt.SeatID != "" {
				bookingsPerSeat[evt.SeatID]++
			}
		case events.TypeBookingSuccess:
			successCount++
			bookingsPerNode[evt.NodeID]++
		case events.TypeBookingFailed:
			failCount++
		case events.TypeElectionStart:
			electionCount++
		case events.TypeRARequestSent:
			raRequestCount++
		}
	}

	// Find most requested seats
	type seatCount struct {
		Seat  string `json:"seat"`
		Count int    `json:"count"`
	}
	topSeats := make([]seatCount, 0)
	for seat, count := range bookingsPerSeat {
		topSeats = append(topSeats, seatCount{Seat: seat, Count: count})
	}

	resp := map[string]interface{}{
		"total_requests":    requestCount,
		"successful":        successCount,
		"failed":            failCount,
		"bookings_per_seat": bookingsPerSeat,
		"bookings_per_node": bookingsPerNode,
		"top_seats":         topSeats,
		"elections":         electionCount,
		"ra_requests":       raRequestCount,
		"total_events":      len(allEvents),
	}
	writeJSON(w, resp)
}

// POST /api/election — Manually trigger a leader election (for demo)
func (a *API) handleTriggerElection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	log.Printf("[Node %d] Manual election triggered via dashboard", a.hub.NodeID())
	a.hub.Events.LogSimple(events.TypeElectionStart, a.hub.Clock.Tick(),
		fmt.Sprintf("Manual election triggered on Node %d", a.hub.NodeID()))

	a.hub.SetLeader(-1)
	go a.bully.StartElection()

	writeJSON(w, map[string]string{
		"message": "Election triggered",
	})
}

// ═══════════════════════════════════════════════════════════════════════
// SERVER-SENT EVENTS (real-time event stream)
// ═══════════════════════════════════════════════════════════════════════

// GET /api/stream — SSE endpoint for real-time event streaming to the dashboard
func (a *API) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to events
	ch := a.events.Subscribe()
	defer a.events.Unsubscribe(ch)

	log.Printf("[Node %d] SSE client connected", a.hub.NodeID())

	// Keep-alive ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-ticker.C:
			// Send keep-alive comment
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			log.Printf("[Node %d] SSE client disconnected", a.hub.NodeID())
			return
		}
	}
}
