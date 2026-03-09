@echo off
title Distributed Ticket System - Node 1
color 0B
echo ============================================
echo   DISTRIBUTED TICKET BOOKING SYSTEM
echo   Starting Node 1  (gRPC:50051 / HTTP:8081)
echo ============================================
echo.
echo  Dashboard: http://localhost:8081
echo  Demo Page: http://localhost:8081/demo.html
echo.
echo  Press Ctrl+C to stop.
echo ============================================
echo.
go run cmd/node/main.go -config configs/node1.yaml
pause
