#!/bin/bash
# generate_proto.sh — Generate Go code from .proto definitions
# Run this from the project root: ./scripts/generate_proto.sh

set -e

PROTO_DIR="proto/ticket"
OUT_DIR="proto/ticket"

echo "==> Generating Go protobuf and gRPC code..."

protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  -I . \
  ${PROTO_DIR}/ticket.proto

echo "==> Done. Generated files in ${OUT_DIR}/"
echo "    - ticket.pb.go"
echo "    - ticket_grpc.pb.go"
