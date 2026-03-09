// main.go is the entry point for each distributed node.
//
// Usage:
//
//	go run cmd/node/main.go -config configs/node1.yaml
//	go run cmd/node/main.go -config configs/node2.yaml
//	go run cmd/node/main.go -config configs/node3.yaml
//
// Each node starts:
//  1. gRPC server (for inter-node communication)
//  2. HTTP server (for the web dashboard)
//  3. Connects to all configured peers
//  4. Triggers an initial Bully election to determine the leader
//  5. Starts heartbeat monitoring
//
// The node runs until interrupted (Ctrl+C / SIGINT / SIGTERM).
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"distributed-ticket-system/internal/booking"
	"distributed-ticket-system/internal/bully"
	"distributed-ticket-system/internal/config"
	"distributed-ticket-system/internal/dashboard"
	"distributed-ticket-system/internal/events"
	"distributed-ticket-system/internal/node"
	"distributed-ticket-system/internal/ra"
	pb "distributed-ticket-system/proto/ticket"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "configs/node1.yaml", "Path to node config YAML file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("FATAL: Failed to load config from %s: %v", *configPath, err)
	}

	nodeID := cfg.Node.ID
	log.SetPrefix(fmt.Sprintf("[Node %d] ", nodeID))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.Printf("════════════════════════════════════════════════════")
	log.Printf("  DISTRIBUTED TICKET BOOKING SYSTEM — Node %d", nodeID)
	log.Printf("  gRPC port: %d  |  HTTP port: %d", cfg.Node.GRPCPort, cfg.Node.HTTPPort)
	log.Printf("  Peers: %d configured", len(cfg.Peers))
	log.Printf("════════════════════════════════════════════════════")

	// ─── Initialize the Hub (central coordinator) ───
	hub := node.NewHub(cfg)

	// Log startup event
	hub.Events.LogSimple(events.TypeNodeStarted, hub.Clock.Tick(),
		fmt.Sprintf("Node %d started on gRPC:%d HTTP:%d", nodeID, cfg.Node.GRPCPort, cfg.Node.HTTPPort))

	// ─── Create algorithm services ───
	electionSvc := bully.NewElectionService(hub)
	mutexSvc := ra.NewMutexService(hub)
	bookingSvc := booking.NewService(hub, mutexSvc, hub.Seats)

	// ─── Start gRPC server ───
	grpcAddr := fmt.Sprintf(":%d", cfg.Node.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("FATAL: Failed to listen on %s: %v", grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterElectionServiceServer(grpcServer, electionSvc)
	pb.RegisterMutexServiceServer(grpcServer, mutexSvc)
	pb.RegisterBookingServiceServer(grpcServer, bookingSvc)

	go func() {
		log.Printf("gRPC server listening on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// ─── Start HTTP dashboard server ───
	dashAPI := dashboard.NewAPI(hub, bookingSvc, electionSvc, mutexSvc, hub.Events, hub.Seats)
	go dashAPI.Start(cfg.Node.HTTPPort)
	log.Printf("Dashboard: http://localhost:%d", cfg.Node.HTTPPort)

	// ─── Give gRPC servers time to start, then connect to peers ───
	log.Printf("Waiting 3 seconds for peers to start...")
	time.Sleep(3 * time.Second)

	hub.ConnectToPeers()

	// ─── Initial leader election ───
	log.Printf("Starting initial Bully election...")
	time.Sleep(1 * time.Second)
	go electionSvc.StartElection()

	// ─── Start heartbeat monitoring (as follower initially) ───
	time.Sleep(2 * time.Second)
	go electionSvc.StartHeartbeat()

	// ─── Wait for shutdown signal ───
	log.Printf("════════════════════════════════════════════════════")
	log.Printf("  Node %d is RUNNING. Press Ctrl+C to stop.", nodeID)
	log.Printf("  Dashboard: http://localhost:%d", cfg.Node.HTTPPort)
	log.Printf("════════════════════════════════════════════════════")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down Node %d...", nodeID)
	electionSvc.StopHeartbeat()
	grpcServer.GracefulStop()
	hub.Close()
	log.Printf("Node %d stopped.", nodeID)
}
