@echo off
REM Test script for go-lanlink MVP
REM This script tests local multiplayer on the same machine

echo ========================================
echo go-lanlink MVP Test Script
echo ========================================
echo.

REM Check if lanlinkd exists
if not exist "bin\lanlinkd.exe" (
    echo Building lanlinkd...
    go build -o bin/lanlinkd.exe ./cmd/lanlinkd
    if errorlevel 1 (
        echo Build failed!
        pause
        exit /b 1
    )
)

echo Step 1: Starting host server...
echo.
start "go-lanlink Host" cmd /k "bin\lanlinkd.exe host"

echo Waiting for room to be created...
timeout /t 3 /nobreak > nul

echo.
echo Step 2: Starting client server...
echo.
start "go-lanlink Client" cmd /k "bin\lanlinkd.exe join 0000-00 TestPlayer"

echo.
echo ========================================
echo Two terminal windows should have opened:
echo 1. Host server (will show room code)
echo 2. Client server (will try to join)
echo.
echo NOTE: This uses a dummy room code (0000-00)
echo You should:
echo 1. Copy the REAL room code from the Host window
echo 2. Close the Client window
echo 3. Run: bin\lanlinkd.exe join <REAL_ROOM_CODE>
echo ========================================
echo.

pause
