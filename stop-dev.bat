@echo off
setlocal

cd /d "%~dp0"

echo ===================================================================
echo     ERP Retail Modular - Penghenti Service Lokal
echo     Membebaskan Port 8088 (Backend) dan 5173 (Frontend)
echo ===================================================================
echo.

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0stop-dev.ps1"

echo.
echo Selesai.
pause
