# Distributed Ticket Booking and Analytics System

> **Go · gRPC · Protocol Buffers · Apache Beam · Docker · Bully Algorithm · Ricart–Agrawala · Lamport Clocks**

A distributed ticket booking system demonstrating core distributed systems concepts. Runs locally on one laptop (3 processes or Docker containers) and can scale to 3 physical machines with only config changes.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Folder Structure](#folder-structure)
3. [Prerequisites & Setup](#prerequisites--setup)
4. [Running Locally (Single Laptop)](#running-locally-single-laptop)
5. [Running with Docker](#running-with-docker)
6. [Multi-Laptop Deployment](#multi-laptop-deployment)
7. [Distributed Algorithms](#distributed-algorithms)
8. [Apache Beam Analytics](#apache-beam-analytics)
9. [Dashboard & API Reference](#dashboard--api-reference)
10. [Demo Scenarios](#demo-scenarios)
11. [Report / Viva Help](#report--viva-help)

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                    DISTRIBUTED ARCHITECTURE                       │
│                                                                   │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐          │
│  │   Node 1    │    │   Node 2    │    │   Node 3    │          │
│  │  gRPC:50051 │◄──►│  gRPC:50052 │◄──►│  gRPC:50053 │          │
│  │  HTTP:8081  │    │  HTTP:8082  │    │  HTTP:8083  │          │
│  │             │    │             │    │             │          │
│  │ ┌─────────┐ │    │ ┌─────────┐ │    │ ┌─────────┐ │          │
│  │ │ Lamport │ │    │ │ Lamport │ │    │ │ Lamport │ │          │
│  │ │  Clock  │ │    │ │  Clock  │ │    │ │  Clock  │ │          │
│  │ └─────────┘ │    │ └─────────┘ │    │ └─────────┘ │          │
│  │ ┌─────────┐ │    │ ┌─────────┐ │    │ ┌─────────┐ │          │
│  │ │ Seat    │ │    │ │ Seat    │ │    │ │ Seat    │ │          │
│  │ │ Store   │ │    │ │ Store   │ │    │ │ Store   │ │          │
│  │ └─────────┘ │    │ └─────────┘ │    │ └─────────┘ │          │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘          │
│         │                  │                  │                   │
│         └──────────────────┼──────────────────┘                   │
│                            │                                      │
│              ┌─────────────▼─────────────┐                       │
│              │    gRPC Communication      │                       │
│              │  • Bully Election msgs     │                       │
│              │  • RA Mutex requests       │                       │
│              │  • Booking requests        │                       │
│              │  • Seat sync updates       │                       │
│              │  • Heartbeat messages      │                       │
│              └───────────────────────────┘                       │
│                                                                   │
│  ┌───────────────────────────────────────────────────┐           │
│  │              Apache Beam Pipeline                  │           │
│  │  Reads event logs → Computes analytics:            │           │
│  │  • Bookings per seat  • Success/fail rates         │           │
│  │  • Events per node    • Event type distribution    │           │
│  └───────────────────────────────────────────────────┘           │
│                                                                   │
│  ┌───────────────────────────────────────────────────┐           │
│  │              HTML Dashboard (per node)              │           │
│  │  • Booking form      • Seat map grid               │           │
│  │  • Node status       • Leader display              │           │
│  │  • Lamport clocks    • RA states                   │           │
│  │  • Algorithm flow    • Event log (SSE live)        │           │
│  │  • Analytics summary                               │           │
│  └───────────────────────────────────────────────────┘           │
└──────────────────────────────────────────────────────────────────┘
```

### What Runs on Each Node

Each node is an identical binary running with a different config file:

| Component | Description |
|-----------|-------------|
| **gRPC Server** | Listens for inter-node messages (elections, RA, seat sync, heartbeats) |
| **HTTP Server** | Serves the dashboard UI and REST API |
| **Lamport Clock** | Tracks logical time across all events |
| **Seat Store** | In-memory seat state, synced via gRPC after bookings |
| **Bully Service** | Leader election and heartbeat monitoring |
| **RA Mutex Service** | Distributed mutual exclusion for seat booking |
| **Booking Service** | Coordinates booking flow: RA → book → sync |
| **Event Logger** | Logs all events to memory + JSON file (for Beam) |

### How gRPC is Used

All inter-node communication uses gRPC with Protocol Buffers:

- **ElectionService**: `Election()`, `Coordinator()`, `Heartbeat()`
- **MutexService**: `RequestAccess()`
- **BookingService**: `RequestBooking()`, `SyncSeat()`, `GetAllSeats()`

### How the HTML Frontend Works

The dashboard is served as static files by each node's HTTP server.
- **Polling**: Status, seats, analytics are polled every 2-5 seconds
- **SSE (Server-Sent Events)**: Real-time event streaming for the event log and algorithm flow
- **REST API**: Booking requests are POSTed to `/api/book`

---

## Folder Structure

```
distributed-ticket-system/
├── cmd/
│   └── node/
│       └── main.go              # Entry point — starts gRPC + HTTP + algorithms
├── internal/
│   ├── booking/
│   │   └── service.go           # Booking flow: RA mutex → book seat → sync
│   ├── bully/
│   │   └── election.go          # Bully Algorithm: election + heartbeat
│   ├── clock/
│   │   └── lamport.go           # Lamport Logical Clock implementation
│   ├── config/
│   │   └── config.go            # YAML config loading
│   ├── dashboard/
│   │   └── api.go               # HTTP REST API + SSE for dashboard
│   ├── events/
│   │   └── logger.go            # Event logging (memory + file + subscribers)
│   ├── node/
│   │   └── hub.go               # Central Hub: state, peers, connections
│   ├── ra/
│   │   └── mutex.go             # Ricart–Agrawala mutual exclusion
│   └── seats/
│       └── store.go             # In-memory seat storage
├── proto/
│   └── ticket/
│       └── ticket.proto         # Protocol Buffer definitions + gRPC services
├── beam/
│   └── analytics/
│       └── pipeline.go          # Apache Beam analytics pipeline
├── web/
│   └── static/
│       ├── index.html           # Dashboard HTML
│       ├── style.css            # Dashboard styles
│       └── app.js               # Dashboard JavaScript
├── docker/
│   ├── Dockerfile               # Multi-stage Docker build
│   ├── docker-compose.yml       # Run 3 nodes in containers
│   └── docker-configs/          # Docker-specific YAML configs
├── configs/
│   ├── node1.yaml               # Local Node 1 config
│   ├── node2.yaml               # Local Node 2 config
│   └── node3.yaml               # Local Node 3 config
├── scripts/
│   ├── generate_proto.sh        # Protobuf code generation (Linux/Mac)
│   ├── generate_proto.ps1       # Protobuf code generation (Windows)
│   ├── run_local.sh             # Run 3 nodes locally (Linux/Mac)
│   └── run_local.ps1            # Run 3 nodes locally (Windows)
├── logs/                        # Event log files (JSON lines)
├── go.mod
├── .gitignore
└── README.md
```

---

## Prerequisites & Setup

### 1. Install Go (1.21+)

Download from https://go.dev/dl/

```bash
go version  # Should show go1.21+
```

### 2. Install Protocol Buffers Compiler

**Windows (with Chocolatey):**
```powershell
choco install protoc
```

**Linux:**
```bash
sudo apt install -y protobuf-compiler
```

**Mac:**
```bash
brew install protobuf
```

### 3. Install Go Protobuf Plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Make sure `$GOPATH/bin` is in your PATH.

### 4. Generate Protobuf Code

From the project root:

**Windows:**
```powershell
.\scripts\generate_proto.ps1
```

**Linux/Mac:**
```bash
chmod +x scripts/generate_proto.sh
./scripts/generate_proto.sh
```

This generates `proto/ticket/ticket.pb.go` and `proto/ticket/ticket_grpc.pb.go`.

### 5. Install Go Dependencies

```bash
cd distributed-ticket-system
go mod tidy
```

---

## Running Locally (Single Laptop)

### Option A: Three Terminal Windows

Open 3 separate terminals and run from the project root:

**Terminal 1 — Node 1:**
```bash
go run cmd/node/main.go -config configs/node1.yaml
```

**Terminal 2 — Node 2:**
```bash
go run cmd/node/main.go -config configs/node2.yaml
```

**Terminal 3 — Node 3:**
```bash
go run cmd/node/main.go -config configs/node3.yaml
```

> **Tip:** Start Node 3 first (highest ID) so it becomes the initial leader via Bully election.

### Option B: Script

**Windows:**
```powershell
.\scripts\run_local.ps1
```

**Linux/Mac:**
```bash
./scripts/run_local.sh
```

### Access Dashboards

- **Node 1:** http://localhost:8081
- **Node 2:** http://localhost:8082
- **Node 3:** http://localhost:8083

All three dashboards show the same seat state (synced via gRPC).

---

## Running with Docker

```bash
cd docker
docker-compose up --build
```

This builds the image and starts 3 containers on a shared `ticket-net` bridge network. The Docker configs use container hostnames (`node1`, `node2`, `node3`) instead of `localhost`.

Same dashboard URLs apply: http://localhost:8081, :8082, :8083.

To stop:
```bash
docker-compose down
```

---

## Multi-Laptop Deployment

To run across 3 physical laptops on the same network:

### 1. Find Each Laptop's IP

```bash
# Windows
ipconfig
# Linux/Mac
ip addr show  # or: hostname -I
```

Example:
- Laptop A: `192.168.1.100` → Node 1
- Laptop B: `192.168.1.101` → Node 2
- Laptop C: `192.168.1.102` → Node 3

### 2. Create Config Files

On **Laptop A** (`configs/node1.yaml`):
```yaml
node:
  id: 1
  grpc_port: 50051
  http_port: 8081
peers:
  - id: 2
    address: "192.168.1.101"
    grpc_port: 50052
  - id: 3
    address: "192.168.1.102"
    grpc_port: 50053
seats:
  rows: ["A", "B", "C"]
  cols_per_row: 4
heartbeat:
  interval_ms: 2000
  timeout_ms: 5000
election:
  timeout_ms: 3000
```

Similarly for Laptops B and C (changing the `node.id` and peer addresses).

### 3. Open Firewall Ports

On each laptop, allow incoming TCP on:
- **gRPC:** 50051 (or whichever port that node uses)
- **HTTP:** 8081 (or whichever port)

**Windows:**
```powershell
netsh advfirewall firewall add rule name="Ticket gRPC" dir=in action=allow protocol=tcp localport=50051
netsh advfirewall firewall add rule name="Ticket HTTP" dir=in action=allow protocol=tcp localport=8081
```

### 4. Run Each Node

On each laptop:
```bash
go run cmd/node/main.go -config configs/node1.yaml   # on Laptop A
go run cmd/node/main.go -config configs/node2.yaml   # on Laptop B
go run cmd/node/main.go -config configs/node3.yaml   # on Laptop C
```

### 5. Access Dashboard

Open a browser on any laptop and visit any node's HTTP port:
- `http://192.168.1.100:8081` (from any laptop)

---

## Distributed Algorithms

### 1. Bully Algorithm (Leader Election)

**Purpose:** Elect the node with the highest ID as leader.

**When triggered:**
- Node startup (initial election)
- Leader heartbeat timeout
- Manual trigger via dashboard

**Flow:**
```
Node 1 detects leader failure
  ├──► Sends ELECTION to Node 2, Node 3 (higher IDs)
  │
  ├── Node 2 receives ELECTION from Node 1
  │   ├──► Sends OK to Node 1 (I have higher ID)
  │   └──► Starts its own election → sends ELECTION to Node 3
  │
  ├── Node 3 receives ELECTION from Node 2
  │   ├──► Sends OK to Node 2
  │   └──► No higher nodes → declares self LEADER
  │        └──► Broadcasts COORDINATOR(3) to Node 1, Node 2
  │
  └── All nodes set leader = Node 3
```

**Implementation:** [internal/bully/election.go](internal/bully/election.go)

### 2. Ricart–Agrawala Algorithm (Distributed Mutual Exclusion)

**Purpose:** Ensure only one node books a specific seat at a time.

**When used:** Before every seat booking operation.

**Flow:**
```
Node 1 wants to book seat A1
  ├──► Increments Lamport clock
  ├──► Sends REQUEST(timestamp=5, node=1, seat=A1) to Node 2, Node 3
  │
  ├── Node 2: Not interested in A1 → sends REPLY immediately
  ├── Node 3: Also wants A1 (timestamp=7)
  │   └── 5 < 7, so Node 1 has priority → sends REPLY
  │
  ├── Node 1 receives REPLY from all → enters Critical Section
  │   └── Books seat A1
  │   └── Exits Critical Section
  │   └── Sends deferred replies (if any)
  │
  └── Syncs seat update to Node 2, Node 3 via SyncSeat gRPC
```

**Priority rule:** Lower Lamport timestamp wins. Ties broken by lower Node ID.

**Implementation:** [internal/ra/mutex.go](internal/ra/mutex.go)

### 3. Lamport Logical Clocks

**Purpose:** Establish causal ordering of events across nodes.

**Rules:**
1. Before each local event: `time++`
2. When sending a message: `time++`, attach `time` to message
3. When receiving a message with timestamp `t`: `time = max(time, t) + 1`

**Used in:**
- All gRPC messages carry Lamport timestamps
- RA request priority is based on Lamport time
- Event log entries include Lamport time for ordering

**Implementation:** [internal/clock/lamport.go](internal/clock/lamport.go)

---

## Apache Beam Analytics

The Apache Beam pipeline reads event log files and computes batch analytics.

### Running the Pipeline

After generating some booking events:

```bash
go run beam/analytics/pipeline.go \
  -input "logs/node*_events.jsonl" \
  -output "logs/analytics_output"
```

### Output Files

| File | Content |
|------|---------|
| `analytics_output_bookings_per_seat.txt` | Request count per seat |
| `analytics_output_success_per_seat.txt` | Successful bookings per seat |
| `analytics_output_failed_per_seat.txt` | Failed bookings per seat |
| `analytics_output_events_per_node.txt` | Event count per node |
| `analytics_output_events_by_type.txt` | Count by event type |

### How Events Flow to Beam

```
User books seat → Booking Service logs event → Event Logger writes JSON-line to logs/
Beam pipeline reads log files → ParDo transforms → stats.Count → textio.Write output
```

For future streaming: replace file I/O with Kafka/Pub-Sub source and windowed aggregations.

---

## Dashboard & API Reference

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Dashboard HTML page |
| `POST` | `/api/book` | Book a seat: `{"user_name":"Alice","seat_id":"A1"}` |
| `GET` | `/api/seats` | All seats with status |
| `GET` | `/api/status` | Node status, leader, peers, Lamport clock, RA states |
| `GET` | `/api/events?limit=50&type=BOOKING_SUCCESS` | Recent events |
| `GET` | `/api/analytics` | Analytics summary |
| `POST` | `/api/election` | Manually trigger Bully election |
| `GET` | `/api/stream` | SSE stream for real-time events |

### Dashboard Features

| Panel | Shows |
|-------|-------|
| **Node Status** | Node ID, state, leader, Lamport clock, peers |
| **Book a Seat** | Form to submit booking + result |
| **Seat Map** | Visual grid of all seats (click to select) |
| **RA States** | Active Ricart–Agrawala mutex states per seat |
| **Algorithm Flow** | Live feed of algorithm events with labels |
| **Analytics** | Booking counts, success/fail, per-seat bar chart |
| **Event Log** | Full event log with type filters and Lamport times |

---

## Demo Scenarios

### Demo 1: Normal Booking

**Setup:** All 3 nodes running, Node 3 is leader.

1. Open Node 1 dashboard (http://localhost:8081)
2. Enter name "Alice", seat "A1", click Book
3. **Expected:**
   - Event log shows: RA_REQUEST_SENT → RA_REPLY_RECEIVED × 2 → RA_ENTER_CS → BOOKING_SUCCESS → RA_EXIT_CS → SEAT_SYNCED
   - Seat A1 turns red (booked) on all dashboards
   - Analytics: 1 request, 1 success

### Demo 2: Concurrent Booking (Same Seat)

**Setup:** All 3 nodes running.

1. Open Node 1 and Node 2 dashboards side by side
2. On Node 1: Enter "Alice", seat "B1"
3. On Node 2: Enter "Bob", seat "B1"
4. Submit both nearly simultaneously
5. **Expected:**
   - One node gets RA priority (lower Lamport timestamp or lower node ID)
   - That node books B1 successfully
   - The other node gets BOOKING_FAILED
   - Dashboard shows RA_REQUEST_SENT, deferred replies, then resolution
   - Only one BOOKING_SUCCESS for B1

### Demo 3: Leader Failure and Bully Election

**Setup:** All 3 nodes running, Node 3 is leader.

1. Kill Node 3 (Ctrl+C in its terminal)
2. Wait ~5 seconds for heartbeat timeout
3. **Expected on Node 1 and 2 dashboards:**
   - LEADER_FAILURE event
   - ELECTION_START event
   - Node 1 sends Election to Node 2
   - Node 2 sends OK to Node 1
   - Node 2 has no higher nodes → declares self leader
   - ELECTION_COORDINATOR: Node 2 is new leader
   - Dashboard leader display updates to "Node 2"

### Demo 4: Lamport Clock Ordering

**Setup:** All nodes running.

1. Make several bookings on different nodes
2. Check the event log on any dashboard
3. **Expected:**
   - All events have Lamport timestamps
   - Events from a single node are monotonically increasing
   - Cross-node events reflect causal ordering
   - RA requests show timestamp-based priority decisions

### Demo 5: Apache Beam Analytics

**Setup:** After making 5-10 bookings across different nodes.

1. Run the Beam pipeline:
   ```bash
   go run beam/analytics/pipeline.go -input "logs/node*_events.jsonl" -output "logs/analytics_output"
   ```
2. **Expected output files:**
   - `analytics_output_bookings_per_seat.txt` showing request distribution
   - `analytics_output_success_per_seat.txt` showing which seats were booked
   - `analytics_output_events_per_node.txt` showing activity per node

---

## Report / Viva Help

### Abstract

This project implements a distributed ticket booking system using Go, demonstrating three core distributed systems algorithms: Bully Algorithm for leader election, Ricart–Agrawala Algorithm for distributed mutual exclusion, and Lamport Logical Clocks for causal event ordering. The system uses gRPC with Protocol Buffers for inter-node communication, Apache Beam (Go SDK) for event stream analytics, and Docker for containerized deployment. A web-based dashboard provides real-time visualization of algorithm behavior and system state.

### Problem Statement

In a distributed ticket booking system, multiple nodes may receive concurrent booking requests for the same seat. Without proper coordination, this leads to double bookings (mutual exclusion violation), orphaned leader roles (no failure recovery), and inconsistent event ordering (no causal clock). We need mechanisms to elect a coordinator, prevent concurrent modifications, and establish event order across nodes.

### Objectives

1. Implement Bully Algorithm for automatic leader election with failure recovery
2. Implement Ricart–Agrawala Algorithm for distributed mutual exclusion in seat booking
3. Implement Lamport Logical Clocks for causal event ordering across nodes
4. Use gRPC + Protocol Buffers for efficient, type-safe inter-node communication
5. Use Apache Beam Go SDK for processing booking event streams and generating analytics
6. Provide a visual dashboard for educational demonstration of algorithm behavior
7. Support deployment on a single laptop (3 processes) and across 3 physical machines

### Software Used

| Software | Purpose |
|----------|---------|
| Go 1.21+ | Backend implementation language |
| gRPC | Inter-node RPC communication framework |
| Protocol Buffers | Serialization format for gRPC messages |
| Apache Beam Go SDK | Event stream processing and analytics |
| Docker | Containerization for deployment |
| HTML/CSS/JavaScript | Frontend dashboard |
| YAML | Configuration files |

### Distributed System Type

This is a **decentralized distributed system** with a **coordinator-based architecture**:
- All nodes are peers with identical code
- The Bully Algorithm elects one node as the coordinator/leader
- The leader monitors node health via heartbeats
- Mutual exclusion is fully decentralized (Ricart–Agrawala is peer-to-peer, no central authority)
- Seat state is replicated across all nodes

### Why Each Algorithm Is Used

| Algorithm | Why Used |
|-----------|----------|
| **Bully Algorithm** | Provides automatic leader election. When the leader fails, the remaining nodes detect the failure via heartbeat timeout and elect a new leader. The highest-ID node wins, ensuring deterministic outcomes. |
| **Ricart–Agrawala** | Provides distributed mutual exclusion without a central coordinator. When two nodes try to book the same seat, RA ensures only one enters the critical section. Uses Lamport timestamps for priority. |
| **Lamport Clocks** | Provides a partial causal ordering of events across all nodes. Essential for RA's timestamp-based priority and for the event log to show meaningful ordering of distributed events. |

### Why Apache Beam

Apache Beam provides a unified programming model for batch and stream processing. In this project:
- Event logs from all nodes are processed as a batch pipeline
- Beam's `ParDo` transforms parse events, `stats.Count` aggregates them
- Results show booking patterns, success rates, and node activity
- The same pipeline can be extended for real-time streaming with Kafka/Pub-Sub

### Possible Viva Questions & Answers

**Q1: What is the Bully Algorithm?**
The Bully Algorithm is a leader election algorithm where the node with the highest process ID becomes the leader. When a node detects the leader has failed, it sends Election messages to all nodes with higher IDs. If no higher node responds, it declares itself leader. If a higher node responds (OK), that node takes over the election.

**Q2: What happens if two nodes try to book the same seat simultaneously?**
The Ricart–Agrawala algorithm handles this. Both nodes send REQUEST messages to all peers with their Lamport timestamps. The node with the lower timestamp (or lower node ID as tiebreaker) gets priority and receives REPLY from all peers first. It enters the critical section and books the seat. The other node's request is deferred until the first node exits.

**Q3: Why not use a central lock server instead of Ricart–Agrawala?**
A central lock server is a single point of failure. Ricart–Agrawala is fully decentralized — it works even without a designated coordinator. It demonstrates true distributed mutual exclusion where all nodes participate in the decision.

**Q4: What are Lamport Logical Clocks? Why not use wall clocks?**
Lamport clocks are logical counters that establish causal ordering. Wall clocks (system time) cannot be perfectly synchronized across machines — clock skew causes incorrect ordering. Lamport clocks guarantee: if event A causally precedes B, then L(A) < L(B), without needing synchronized physical clocks.

**Q5: How does the system handle a node crashing?**
The leader sends periodic heartbeats. If a follower doesn't receive heartbeats within the timeout, it assumes the leader has failed and starts a Bully election. If a follower crashes, the leader detects it when heartbeats fail. Other nodes treat unreachable peers as giving implicit RA replies to prevent deadlock.

**Q6: What is gRPC and why is it used here?**
gRPC is a high-performance RPC framework using HTTP/2 and Protocol Buffers. We use it because: (1) it provides strongly-typed service definitions via .proto files, (2) it supports bidirectional streaming, (3) it generates client/server code automatically, (4) it's efficient for the frequent small messages between nodes (elections, heartbeats, RA requests).

**Q7: How does Apache Beam fit into this project?**
Beam processes the event logs generated by all nodes. It reads JSON-lines log files, parses events, and computes analytics like bookings per seat, success rates, and events per node. Beam's value is its unified model — the same pipeline code can run on different runners (direct, Flink, Dataflow) and can be extended from batch to streaming.

**Q8: Can this system scale beyond 3 nodes?**
Yes. The algorithms support any number of nodes. To add a node: give it a unique ID, add its address to all other configs, and update peer lists. However, Ricart–Agrawala requires O(n) messages per critical section request (n = number of nodes), so performance degrades linearly. For large clusters, a different mutex algorithm (like token-based) would be more efficient.

**Q9: What is the difference between Lamport clocks and Vector clocks?**
Lamport clocks provide a partial order: if L(A) < L(B), we can't be sure A causally precedes B. Vector clocks provide the complete causal history: V(A) < V(B) if and only if A causally precedes B. We use Lamport clocks for simplicity and because RA only needs the partial ordering property — ties are broken by node ID.

**Q10: How do you ensure seat state consistency across nodes?**
After a node books a seat (inside the RA critical section), it broadcasts a `SyncSeat` gRPC message to all peers with the updated seat state. Each peer updates its local seat store. The critical section ensures no concurrent modifications, so all sync messages are serializable.
