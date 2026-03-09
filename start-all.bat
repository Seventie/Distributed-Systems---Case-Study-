@echo off
title Distributed Ticket System - ALL NODES
echo ============================================
echo   Starting ALL 3 nodes in separate windows
echo   Wait ~10 seconds then open your browser
echo.
echo   Node 1: http://localhost:8081
echo   Node 2: http://localhost:8082
echo   Node 3: http://localhost:8083
echo   Demo:   http://localhost:8081/demo.html
echo ============================================
start "Node 1" cmd /k "go run cmd/node/main.go -config configs/node1.yaml"
timeout /t 2 /nobreak >nul
start "Node 2" cmd /k "go run cmd/node/main.go -config configs/node2.yaml"
timeout /t 2 /nobreak >nul
start "Node 3" cmd /k "go run cmd/node/main.go -config configs/node3.yaml"
echo.
echo All 3 nodes starting... wait 10 seconds then open:
echo   http://localhost:8081
echo.
pause
