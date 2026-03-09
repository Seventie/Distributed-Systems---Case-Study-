@echo off
title Distributed Ticket System - Node 2
color 0E
echo ============================================
echo   DISTRIBUTED TICKET BOOKING SYSTEM
echo   Starting Node 2  (gRPC:50052 / HTTP:8082)
echo ============================================
echo.
echo  Dashboard: http://localhost:8082
echo.
echo  Press Ctrl+C to stop.
echo ============================================
echo.
go run cmd/node/main.go -config configs/node2.yaml
pause
