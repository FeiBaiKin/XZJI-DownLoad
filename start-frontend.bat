@echo off
echo ======================================
echo   Starting Frontend Service
echo ======================================
echo.

cd /d "%~dp0frontend"

if not exist "node_modules" (
    echo [1/2] First run, installing dependencies...
    echo        Using npm...
    call npm install
    if errorlevel 1 (
        echo        [ERROR] Failed to install dependencies
        pause
        exit /b 1
    )
) else (
    echo [1/2] Dependencies already installed
)

echo [2/2] Starting frontend dev server...
echo        Port:  3000
echo        URL:   http://localhost:3000
echo        Proxy: /api -^> http://localhost:4000
echo.
echo ======================================
echo.

call npm run dev

if errorlevel 1 (
    echo.
    echo [ERROR] Failed to start frontend service
    pause
    exit /b 1
)
