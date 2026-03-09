@echo off
title Distributed Ticket System - Node 3
color 0A
echo ============================================
echo   DISTRIBUTED TICKET BOOKING SYSTEM
echo   Starting Node 3  (gRPC:50053 / HTTP:8083)
echo ============================================
echo.
echo  Dashboard: http://localhost:8083
echo.
echo  Press Ctrl+C to stop.
echo ============================================
echo.
go run cmd/node/main.go -config configs/node3.yaml
pause
