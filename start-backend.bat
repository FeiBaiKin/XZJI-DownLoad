@echo off
echo ======================================
echo   Starting Backend Service
echo ======================================
echo.

cd /d "%~dp0backend"

if not exist ".installed" (
    echo [INFO] System not installed yet
    echo [INFO] Please visit: http://localhost:4000/install
    echo.
)

echo [1/2] Checking dependencies...
go mod download >nul 2>&1
if errorlevel 1 (
    echo        [ERROR] Failed to download dependencies
    pause
    exit /b 1
)
echo        Dependencies OK

echo [2/2] Starting backend service...
echo        Port: 4000
echo        API:  http://localhost:4000/api
echo.
echo ======================================
echo.

go run main.go

if errorlevel 1 (
    echo.
    echo [ERROR] Failed to start backend service
    pause
    exit /b 1
)
