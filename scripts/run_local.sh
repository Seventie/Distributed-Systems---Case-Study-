#!/bin/bash
# run_local.sh — Run all 3 nodes locally
# Run this from the project root: ./scripts/run_local.sh

set -e

echo "==> Building the node binary..."
go build -o bin/node cmd/node/main.go

echo "==> Starting Node 1 (gRPC :50051, HTTP :8081)..."
./bin/node -config configs/node1.yaml &
PID1=$!

sleep 1

echo "==> Starting Node 2 (gRPC :50052, HTTP :8082)..."
./bin/node -config configs/node2.yaml &
PID2=$!

sleep 1

echo "==> Starting Node 3 (gRPC :50053, HTTP :8083)..."
./bin/node -config configs/node3.yaml &
PID3=$!

echo ""
echo "==> All 3 nodes started."
echo "    Node 1 dashboard: http://localhost:8081"
echo "    Node 2 dashboard: http://localhost:8082"
echo "    Node 3 dashboard: http://localhost:8083"
echo ""
echo "    PIDs: $PID1, $PID2, $PID3"
echo "    To stop: kill $PID1 $PID2 $PID3"
echo ""

# Wait for all background jobs
wait
