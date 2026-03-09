# generate_proto.ps1 — Generate Go code from .proto definitions
# Run this from the project root: .\scripts\generate_proto.ps1

$ErrorActionPreference = "Stop"

$PROTO_DIR = "proto\ticket"

Write-Host "==> Generating Go protobuf and gRPC code..."

protoc `
  --go_out=. `
  --go_opt=paths=source_relative `
  --go-grpc_out=. `
  --go-grpc_opt=paths=source_relative `
  -I . `
  "$PROTO_DIR\ticket.proto"

Write-Host "==> Done. Generated files in $PROTO_DIR\"
Write-Host "    - ticket.pb.go"
Write-Host "    - ticket_grpc.pb.go"
