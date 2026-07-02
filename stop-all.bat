@echo off
echo.
echo ======================================
echo   Stopping GO-XZJI Services
echo ======================================
echo.

cd /d "%~dp0"

echo [1/2] Stopping backend service...
taskkill /F /FI "WINDOWTITLE eq GO-XZJI Backend*" >nul 2>&1
if errorlevel 1 (
    echo        Backend window not found
) else (
    echo        Backend stopped [OK]
)

echo [2/2] Stopping frontend service...
taskkill /F /FI "WINDOWTITLE eq GO-XZJI Frontend*" >nul 2>&1
if errorlevel 1 (
    echo        Frontend window not found
) else (
    echo        Frontend stopped [OK]
)

echo.
echo ======================================
echo   All services stopped
echo ======================================
echo.
pause
