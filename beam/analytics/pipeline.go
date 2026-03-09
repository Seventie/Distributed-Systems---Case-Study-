// Package analytics implements an Apache Beam pipeline for processing
// booking event streams and computing analytics.
//
// ═══════════════════════════════════════════════════════════════════════
// APACHE BEAM INTEGRATION
// ═══════════════════════════════════════════════════════════════════════
//
// Apache Beam provides a unified model for batch and stream processing.
// In this project, Beam is used to:
//  1. Read booking events from JSON-lines log files
//  2. Parse and transform event data
//  3. Compute analytics: bookings per seat, success/fail rates, per-node stats
//  4. Output results to JSON files for the dashboard
//
// For the MVP (single-laptop version), the pipeline reads from the event log
// files written by the event logger (logs/node*_events.jsonl).
//
// To run the pipeline:
//
//	go run beam/analytics/pipeline.go -input "logs/node*_events.jsonl" -output "logs/analytics_output"
//
// For production/streaming, this can be extended to read from Kafka, Pub/Sub,
// or a streaming source with windowing.
//
// ═══════════════════════════════════════════════════════════════════════
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/apache/beam/sdks/v2/go/pkg/beam"
	"github.com/apache/beam/sdks/v2/go/pkg/beam/io/textio"
	"github.com/apache/beam/sdks/v2/go/pkg/beam/transforms/stats"
	"github.com/apache/beam/sdks/v2/go/pkg/beam/x/beamx"
)

// BookingEvent mirrors the event structure from the event logger.
type BookingEvent struct {
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	NodeID      int32  `json:"node_id"`
	SeatID      string `json:"seat_id"`
	UserName    string `json:"user_name"`
	LamportTime int64  `json:"lamport_time"`
	Timestamp   int64  `json:"timestamp"`
	Details     string `json:"details"`
}

func init() {
	// Register all DoFns with Beam's type registry
	beam.RegisterFunction(parseEventFn)
	beam.RegisterFunction(extractSeatIDFn)
	beam.RegisterFunction(extractNodeIDFn)
	beam.RegisterFunction(extractEventTypeFn)
	beam.RegisterFunction(isBookingRequestFn)
	beam.RegisterFunction(isBookingSuccessFn)
	beam.RegisterFunction(isBookingFailedFn)
	beam.RegisterFunction(formatKVFn)
}

func main() {
	// Command-line flags
	input := flag.String("input", "logs/node*_events.jsonl", "Input event log file glob pattern")
	output := flag.String("output", "logs/analytics_output", "Output directory/prefix for results")
	flag.Parse()

	// Initialize Beam
	beam.Init()
	p, s := beam.NewPipelineWithRoot()

	log.Println("═══════════════════════════════════════════════════")
	log.Println("  APACHE BEAM ANALYTICS PIPELINE")
	log.Printf("  Input:  %s", *input)
	log.Printf("  Output: %s", *output)
	log.Println("═══════════════════════════════════════════════════")

	// ─── Step 1: Read event log files ───
	// Read all matching log files as lines of text
	files, _ := filepath.Glob(*input)
	if len(files) == 0 {
		log.Fatalf("No input files found matching: %s", *input)
	}

	var allLines beam.PCollection
	for i, file := range files {
		lines := textio.Read(s, file)
		if i == 0 {
			allLines = lines
		} else {
			allLines = beam.Flatten(s, allLines, lines)
		}
	}

	// ─── Step 2: Parse JSON lines into BookingEvent structs ───
	parsed := beam.ParDo(s, parseEventFn, allLines)

	// ─── Step 3: Analytics ───

	// 3a. Bookings per seat (count of BOOKING_REQUEST per seat_id)
	bookingRequests := beam.ParDo(s, filterFn(isBookingRequestFn), parsed)
	seatIDs := beam.ParDo(s, extractSeatIDFn, bookingRequests)
	bookingsPerSeat := stats.Count(s, seatIDs)
	bookingsPerSeatStr := beam.ParDo(s, formatKVFn, bookingsPerSeat)
	textio.Write(s, *output+"_bookings_per_seat.txt", bookingsPerSeatStr)

	// 3b. Successful bookings count
	successEvents := beam.ParDo(s, filterFn(isBookingSuccessFn), parsed)
	successSeatIDs := beam.ParDo(s, extractSeatIDFn, successEvents)
	successPerSeat := stats.Count(s, successSeatIDs)
	successPerSeatStr := beam.ParDo(s, formatKVFn, successPerSeat)
	textio.Write(s, *output+"_success_per_seat.txt", successPerSeatStr)

	// 3c. Failed bookings count
	failEvents := beam.ParDo(s, filterFn(isBookingFailedFn), parsed)
	failSeatIDs := beam.ParDo(s, extractSeatIDFn, failEvents)
	failPerSeat := stats.Count(s, failSeatIDs)
	failPerSeatStr := beam.ParDo(s, formatKVFn, failPerSeat)
	textio.Write(s, *output+"_failed_per_seat.txt", failPerSeatStr)

	// 3d. Events per node
	nodeIDs := beam.ParDo(s, extractNodeIDFn, parsed)
	eventsPerNode := stats.Count(s, nodeIDs)
	eventsPerNodeStr := beam.ParDo(s, formatKVFn, eventsPerNode)
	textio.Write(s, *output+"_events_per_node.txt", eventsPerNodeStr)

	// 3e. Events by type
	eventTypes := beam.ParDo(s, extractEventTypeFn, parsed)
	countByType := stats.Count(s, eventTypes)
	countByTypeStr := beam.ParDo(s, formatKVFn, countByType)
	textio.Write(s, *output+"_events_by_type.txt", countByTypeStr)

	// ─── Step 4: Execute the pipeline ───
	log.Println("Running Beam pipeline...")
	if err := beamx.Run(context.Background(), p); err != nil {
		log.Fatalf("Pipeline execution failed: %v", err)
	}

	log.Println("═══════════════════════════════════════════════════")
	log.Println("  PIPELINE COMPLETE — results written to:")
	log.Printf("    %s_bookings_per_seat.txt", *output)
	log.Printf("    %s_success_per_seat.txt", *output)
	log.Printf("    %s_failed_per_seat.txt", *output)
	log.Printf("    %s_events_per_node.txt", *output)
	log.Printf("    %s_events_by_type.txt", *output)
	log.Println("═══════════════════════════════════════════════════")
}

// ═══════════════════════════════════════════════════════════════════════
// BEAM DoFn FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════

// parseEventFn parses a JSON line into a BookingEvent string representation.
// Returns the serialized event as JSON if valid, skips malformed lines.
func parseEventFn(line string) (string, error) {
	var evt BookingEvent
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		return "", fmt.Errorf("skip malformed line: %w", err)
	}
	// Re-serialize to normalize
	data, _ := json.Marshal(evt)
	return string(data), nil
}

// extractSeatIDFn extracts the seat_id from a JSON event string.
func extractSeatIDFn(eventJSON string) string {
	var evt BookingEvent
	json.Unmarshal([]byte(eventJSON), &evt)
	if evt.SeatID == "" {
		return "N/A"
	}
	return evt.SeatID
}

// extractNodeIDFn extracts the node_id as a string from a JSON event string.
func extractNodeIDFn(eventJSON string) string {
	var evt BookingEvent
	json.Unmarshal([]byte(eventJSON), &evt)
	return fmt.Sprintf("Node_%d", evt.NodeID)
}

// extractEventTypeFn extracts the event_type from a JSON event string.
func extractEventTypeFn(eventJSON string) string {
	var evt BookingEvent
	json.Unmarshal([]byte(eventJSON), &evt)
	return evt.EventType
}

// Filter functions
func isBookingRequestFn(eventJSON string) bool {
	return strings.Contains(eventJSON, `"event_type":"BOOKING_REQUEST"`)
}

func isBookingSuccessFn(eventJSON string) bool {
	return strings.Contains(eventJSON, `"event_type":"BOOKING_SUCCESS"`)
}

func isBookingFailedFn(eventJSON string) bool {
	return strings.Contains(eventJSON, `"event_type":"BOOKING_FAILED"`)
}

// filterFn creates a DoFn that filters events based on a predicate.
func filterFn(predicate func(string) bool) func(string, func(string)) {
	return func(eventJSON string, emit func(string)) {
		if predicate(eventJSON) {
			emit(eventJSON)
		}
	}
}

// formatKVFn formats a key-value pair as "key: value" string.
func formatKVFn(key string, value int) string {
	return fmt.Sprintf("%s: %d", key, value)
}
