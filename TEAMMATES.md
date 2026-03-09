# 👋 Teammate Setup Guide — Distributed Ticket Booking System

Follow these steps exactly and you'll be up and running in ~5 minutes.

---

## ✅ Step 1 — Install Go (once, only needed once)

1. Go to: **https://go.dev/dl/**
2. Download: **`go1.21.x.windows-amd64.msi`** (or the latest stable)
3. Run the installer — keep all defaults
4. Open a **new** PowerShell/CMD and type:
   ```
   go version
   ```
   You should see: `go version go1.21.x ...`

> **macOS / Linux?** Download the `.pkg` or `.tar.gz` from the same link.

---

## ✅ Step 2 — Clone the project

Open PowerShell / Terminal and run:

```powershell
git clone https://github.com/Seventie/Distributed-Systems---Case-Study-.git
cd "Distributed-Systems---Case-Study-"
```

---

## ✅ Step 3 — Download Go dependencies (internet needed, once only)

```powershell
go mod tidy
```

Wait for it to finish — it downloads all libraries automatically.

---

## ✅ Step 4 — Decide: Same laptop or multi-laptop?

### Option A — All nodes on ONE laptop (demo / presentation)

Run each command in a **separate PowerShell window**:

**Window 1:**
```powershell
go run cmd/node/main.go -config configs/node1.yaml
```

**Window 2:**
```powershell
go run cmd/node/main.go -config configs/node2.yaml
```

**Window 3:**
```powershell
go run cmd/node/main.go -config configs/node3.yaml
```

Then open your browser:
- Node 1 Dashboard → http://localhost:8081
- Node 2 Dashboard → http://localhost:8082
- Node 3 Dashboard → http://localhost:8083
- **🎬 Demo Flows** → http://localhost:8081/demo.html

---

### Option B — Three separate laptops over Wi-Fi

> **All laptops must be on the SAME Wi-Fi network.**

**Step 4a — Find each laptop's IP address:**

On each laptop open PowerShell and run:
```powershell
ipconfig
```
Look for `IPv4 Address` under your Wi-Fi adapter. Example: `192.168.1.10`

**Step 4b — Edit the multi-laptop config for your node:**

Open `configs/multi-laptop/` — there are 3 config files.

Edit the one for your node and replace the IP placeholders:
- `LAPTOP_A_IP` → IP address of the laptop running Node 1
- `LAPTOP_B_IP` → IP address of the laptop running Node 2
- `LAPTOP_C_IP` → IP address of the laptop running Node 3

**Step 4c — Run your node:**

| Laptop | Command |
|--------|---------|
| Laptop A (Node 1) | `go run cmd/node/main.go -config configs/multi-laptop/node1.yaml` |
| Laptop B (Node 2) | `go run cmd/node/main.go -config configs/multi-laptop/node2.yaml` |
| Laptop C (Node 3) | `go run cmd/node/main.go -config configs/multi-laptop/node3.yaml` |

**Step 4d — Open dashboard from any laptop:**
```
http://LAPTOP_A_IP:8081
http://LAPTOP_B_IP:8082
http://LAPTOP_C_IP:8083
```

**Step 4e — Windows Firewall (if nodes can't connect):**

Run this on each laptop (as Admin):
```powershell
netsh advfirewall firewall add rule name="DS Nodes" dir=in action=allow protocol=tcp localport=50051-50053
netsh advfirewall firewall add rule name="DS HTTP" dir=in action=allow protocol=tcp localport=8081-8083
```

---

## ✅ Step 5 — Quick scripts (Windows only)

We included `.bat` scripts so you can just double-click:

| Script | What it does |
|--------|-------------|
| `start-node1.bat` | Starts Node 1 (localhost mode) |
| `start-node2.bat` | Starts Node 2 (localhost mode) |
| `start-node3.bat` | Starts Node 3 (localhost mode) |

Just double-click the one for your node.

---

## ❓ Troubleshooting

| Problem | Fix |
|---------|-----|
| `go: command not found` | Go not installed — do Step 1 |
| `port already in use` | Another process on 5005x/808x — restart terminal |
| `connection refused` from peers | Start all nodes within 10 seconds of each other |
| Browser shows blank page | Wait 5 secs after node starts, then refresh |
| Multi-laptop: nodes can't see each other | Check firewall (Step 4e) + same Wi-Fi |
| `go mod tidy` errors | Need internet — connect and retry |

---

## 📦 What gets downloaded (go mod tidy)

Everything is automatic — you do NOT manually install:
- gRPC / Protocol Buffers (auto via Go modules)
- Apache Beam SDK (auto via Go modules)  
- All other libraries

You only need **Go** installed.

---

## 🎬 Demo page

After starting nodes, open:  
**http://localhost:8081/demo.html**

Shows 5 animated algorithm walkthroughs:
1. 🗳️ Bully Leader Election
2. 🎫 Normal Seat Booking (RA Mutex)
3. ⚡ Concurrent Booking Conflict
4. 💔 Node Failure & Recovery
5. 🕐 Lamport Clock Sync
