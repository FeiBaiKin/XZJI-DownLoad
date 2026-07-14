@echo off
setlocal EnableExtensions
cd /d "%~dp0"

set "ROOT=%CD%"
set "PACKAGE_DIR=%ROOT%\deploy-package"
set "ARCHIVE=%ROOT%\xzji-deploy.tar.gz"

if not exist "%ROOT%\frontend\package.json" (
    echo [ERROR] frontend\package.json not found.
    exit /b 1
)
if not exist "%ROOT%\backend\go.mod" (
    echo [ERROR] backend\go.mod not found.
    exit /b 1
)

echo [1/6] Building frontend with: npx vite build
pushd "%ROOT%\frontend"
call npx vite build
if errorlevel 1 (
    popd
    echo [ERROR] Frontend build failed.
    exit /b 1
)
popd

echo [2/6] Recreating package directory
if exist "%PACKAGE_DIR%" rmdir /s /q "%PACKAGE_DIR%"
mkdir "%PACKAGE_DIR%\migrations" || exit /b 1
mkdir "%PACKAGE_DIR%\public" || exit /b 1
mkdir "%PACKAGE_DIR%\uploads" || exit /b 1

echo [3/6] Building Linux AMD64 backend
pushd "%ROOT%\backend"
set "GOOS=linux"
set "GOARCH=amd64"
set "CGO_ENABLED=0"
go build -trimpath -ldflags "-s -w" -o "%PACKAGE_DIR%\xzji-linux-amd64" .
if errorlevel 1 (
    popd
    echo [ERROR] Backend build failed.
    exit /b 1
)
popd

echo [4/6] Copying release files
copy /Y "%ROOT%\backend\config.yaml.example" "%PACKAGE_DIR%\config.yaml.example" >nul || exit /b 1
xcopy "%ROOT%\backend\migrations\*" "%PACKAGE_DIR%\migrations\" /E /I /H /Y >nul || exit /b 1
xcopy "%ROOT%\frontend\dist\*" "%PACKAGE_DIR%\public\" /E /I /H /Y >nul || exit /b 1

rem Never copy backend\uploads. The release contains only this placeholder.
type nul > "%PACKAGE_DIR%\uploads\.gitkeep"
for /f "delims=" %%F in ('dir /s /b /a-d "%PACKAGE_DIR%\uploads"') do (
    if /I not "%%~fF"=="%PACKAGE_DIR%\uploads\.gitkeep" (
        echo [ERROR] Unexpected upload file in package: %%F
        exit /b 1
    )
)

echo [5/6] Creating archive
if exist "%ARCHIVE%" del /q "%ARCHIVE%"
tar -czf "%ARCHIVE%" -C "%PACKAGE_DIR%" .
if errorlevel 1 (
    echo [ERROR] Archive creation failed.
    exit /b 1
)

echo [6/6] Verifying upload placeholder
set "UPLOAD_ENTRY_COUNT=0"
for /f "delims=" %%F in ('tar -tzf "%ARCHIVE%" ^| findstr /R /C:"^./uploads/"') do (
    set /a UPLOAD_ENTRY_COUNT+=1
    if /I not "%%F"=="./uploads/" if /I not "%%F"=="./uploads/.gitkeep" (
        echo [ERROR] Real upload file found in archive: %%F
        exit /b 1
    )
)
if "%UPLOAD_ENTRY_COUNT%"=="0" (
    echo [ERROR] uploads placeholder missing from archive.
    exit /b 1
)

echo.
echo [OK] Package created: %ARCHIVE%
echo [OK] Runtime uploads were excluded; only uploads/.gitkeep is included.
exit /b 0
