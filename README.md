# Distributed Ticket Booking System

A distributed application demonstrating consensus algorithms (Bully Algorithm, Ricart-Agrawala) and Lamport Clocks for a synchronized ticket booking system, alongside Apache Beam for analytics.

## Running on a Single Machine (Local Testing)
If you want to test the entire cluster on your own laptop:
1. Open 3 or 4 separate terminal windows.
2. In each window, run one node by specifying a local configuration:
   ```bash
   # Terminal 1
   go run cmd/node/main.go -config configs/node1.yaml

   # Terminal 2
   go run cmd/node/main.go -config configs/node2.yaml

   # Terminal 3
   go run cmd/node/main.go -config configs/node3.yaml
   ```
3. Open your browser to `http://localhost:8081` to view Node 1's dashboard.

## Running on Multiple Laptops (Real Cluster)

To connect different laptops, they **must be on the same Wi-Fi network** (or hotspot). You cannot use `localhost`, you must use the actual IP addresses of each laptop.

### Step 1: Find Everyone's IP Address
On each laptop, open a terminal (Command Prompt/PowerShell) and type:
- Windows: `ipconfig` (Look for "IPv4 Address" under Wireless LAN adapter Wi-Fi)
- Mac/Linux: `ifconfig` or `ip addr`

Write down the IP address for each laptop. For example:
- Laptop 1 (Node 1): `192.168.1.10`
- Laptop 2 (Node 2): `192.168.1.11`
- Laptop 3 (Node 3): `192.168.1.12`

### Step 2: Update Configuration Files
In the folder `configs/multi-laptop/`, open the `.yaml` files in your code editor. 
Replace all occurrences of `YOUR_LAPTOP_1_IP`, `YOUR_LAPTOP_2_IP`, etc., with the real IP addresses you found in Step 1.

**Example for `configs/multi-laptop/node1.yaml` (Running on Laptop 1):**
```yaml
node:
  id: 1
  grpc_port: 50051  # Port must be allowed through the Windows Firewall!
  http_port: 8081

peers:
  - id: 2
    address: "192.168.1.11" # IP of Laptop 2
    grpc_port: 50052
  - id: 3
    address: "192.168.1.12" # IP of Laptop 3
    grpc_port: 50053
```
*Do this carefully for `node1.yaml`, `node2.yaml`, and `node3.yaml` on ALL laptops.*

### Step 3: Allow Ports in Windows Firewall (Important!)
Because you are connecting across different machines, the Windows local firewall might block the connections (like gRPC ports 50051-50054 and HTTP ports 8081-8084).
If you see connection errors, ensure the **Windows Defender Firewall** allows these ports or temporarily disable it for testing on a private network.

### Step 4: Run the Nodes
Once the configurations are updated with correct IPs and synced (pushed to Git and pulled on all laptops):
- **On Laptop 1**: `go run cmd/node/main.go -config configs/multi-laptop/node1.yaml`
- **On Laptop 2**: `go run cmd/node/main.go -config configs/multi-laptop/node2.yaml`
- **On Laptop 3**: `go run cmd/node/main.go -config configs/multi-laptop/node3.yaml`

### Step 5: Open the Dashboard
- **On Laptop 1**, open `http://localhost:8081`
- **On Laptop 2**, open `http://localhost:8082`
- **On Laptop 3**, open `http://localhost:8083`
If everything is configured correctly, they will automatically sync seats, and trigger elections over your Wi-Fi!

## Running the Apache Beam Demo Analytics
To run the analytics pipeline that generates stats on total events and bookings:
1. Ensure the system has generated some log data (by clicking around and booking seats).
2. Stop the nodes (`Ctrl+C`).
3. Run the Beam pipeline:
   ```bash
   go run cmd/analytics/main.go
   ```
4. This will parse the `events.log` and output analytics.
