@echo off
echo ======================================
echo   Starting All Services
echo ======================================
echo.
echo   Backend: http://localhost:4000
echo   Frontend: http://localhost:3000
echo.
echo ======================================
echo.

cd /d "%~dp0"

if not exist "backend\.installed" (
    echo [INFO] System not installed yet
    echo [INFO] Please visit: http://localhost:4000/install
    echo.
    pause
)

echo [1/3] Starting backend service in new window...
start "GO-XZJI Backend - Port 4000" cmd /c "cd /d "%~dp0" && start-backend.bat"

echo [2/3] Waiting for backend to be ready (8 seconds)...
timeout /t 8 /nobreak >nul

echo [3/3] Starting frontend service in new window...
start "GO-XZJI Frontend - Port 3000" cmd /c "cd /d "%~dp0" && start-frontend.bat"

echo.
echo ======================================
echo   All services started!
echo ======================================
echo.
echo   Backend:  http://localhost:4000
echo   Frontend: http://localhost:3000
echo   API:      http://localhost:4000/api
echo.

if not exist "backend\.installed" (
    echo   [FIRST RUN] Visit: http://localhost:4000/install
    echo.
)

echo   [INFO] Services are running in separate windows
echo   [INFO] Close windows to stop services
echo   [INFO] Or run: stop-all.bat
echo.
echo   Opening browser in 5 seconds...
timeout /t 5 /nobreak >nul

if exist "backend\.installed" (
    start http://localhost:3000
) else (
    start http://localhost:4000/install
)

echo.
echo   Browser opened. Press any key to close this window.
echo   (Services will continue running)
echo.
pause >nul
