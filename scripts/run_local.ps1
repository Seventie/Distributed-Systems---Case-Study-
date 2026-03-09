# run_local.ps1 — Run all 3 nodes locally on Windows
# Run this from the project root: .\scripts\run_local.ps1

$ErrorActionPreference = "Stop"

Write-Host "==> Building the node binary..."
go build -o bin/node.exe cmd/node/main.go

Write-Host "==> Starting Node 1 (gRPC :50051, HTTP :8081)..."
$proc1 = Start-Process -FilePath ".\bin\node.exe" -ArgumentList "-config", "configs\node1.yaml" -PassThru -NoNewWindow

Start-Sleep -Seconds 1

Write-Host "==> Starting Node 2 (gRPC :50052, HTTP :8082)..."
$proc2 = Start-Process -FilePath ".\bin\node.exe" -ArgumentList "-config", "configs\node2.yaml" -PassThru -NoNewWindow

Start-Sleep -Seconds 1

Write-Host "==> Starting Node 3 (gRPC :50053, HTTP :8083)..."
$proc3 = Start-Process -FilePath ".\bin\node.exe" -ArgumentList "-config", "configs\node3.yaml" -PassThru -NoNewWindow

Write-Host ""
Write-Host "==> All 3 nodes started."
Write-Host "    Node 1 dashboard: http://localhost:8081"
Write-Host "    Node 2 dashboard: http://localhost:8082"
Write-Host "    Node 3 dashboard: http://localhost:8083"
Write-Host ""
Write-Host "    PIDs: $($proc1.Id), $($proc2.Id), $($proc3.Id)"
Write-Host "    To stop: Stop-Process -Id $($proc1.Id),$($proc2.Id),$($proc3.Id)"
Write-Host ""
Write-Host "Press Ctrl+C to stop all nodes..."

try {
    Wait-Process -Id $proc1.Id, $proc2.Id, $proc3.Id
} catch {
    Write-Host "Stopping all nodes..."
    Stop-Process -Id $proc1.Id -ErrorAction SilentlyContinue
    Stop-Process -Id $proc2.Id -ErrorAction SilentlyContinue
    Stop-Process -Id $proc3.Id -ErrorAction SilentlyContinue
}
