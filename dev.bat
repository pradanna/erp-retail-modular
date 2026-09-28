@echo off
setlocal enabledelayedexpansion

:: ============================================================================
:: ERP Retail Modular - Local Development Launcher
:: Menjalankan Backend (Go) dan Frontend (SvelteKit Backoffice) secara bersamaan
:: ============================================================================

cd /d "%~dp0"

set "TARGET=%~1"
if "%TARGET%"=="" set "TARGET=all"

cls
echo ===================================================================
echo           ERP Retail Modular - Local Development Runner
echo ===================================================================
echo.

:: 1. Validasi Prasyarat: Go
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Golang tidak ditemukan di sistem PATH!
    echo         Pastikan Go sudah terinstall dan terdaftar di environment variable.
    echo.
    pause
    exit /b 1
)

:: 2. Validasi Prasyarat: Node.js & npm
where npm >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Node.js / npm tidak ditemukan di sistem PATH!
    echo         Pastikan Node.js sudah terinstall dan terdaftar di environment variable.
    echo.
    pause
    exit /b 1
)

:: 3. Pengecekan backend/.env
if not exist "%~dp0backend\.env" (
    echo [INFO] File backend\.env tidak ditemukan.
    echo        Menyalin dari backend\.env.example...
    copy "%~dp0backend\.env.example" "%~dp0backend\.env" >nul
    echo [INFO] File backend\.env berhasil dibuat otomatis!
    echo.
)

:: Pengecekan Target Eksekusi
if /i "%TARGET%"=="be" goto start_be_only
if /i "%TARGET%"=="backend" goto start_be_only
if /i "%TARGET%"=="fe" goto start_be_only
if /i "%TARGET%"=="frontend" goto start_fe_only
if /i "%TARGET%"=="all" goto start_all

echo [ERROR] Parameter tidak dikenali: %TARGET%
echo.
echo Penggunaan:
echo   dev.bat              (Menjalankan Backend + Frontend bersamaan)
echo   dev.bat be           (Hanya menjalankan Backend Go)
echo   dev.bat fe           (Hanya menjalankan Frontend Backoffice)
echo.
pause
exit /b 1

:start_be_only
echo [*] Menjalankan Backend Go di jendela baru...
echo     URL Server : http://localhost:8088
echo     Catatan    : Pastikan database MySQL (Laragon/XAMPP) sudah aktif di port 3306!
echo.
start "ERP Backend (Go - Port 8088)" cmd /k "title ERP Backend [Port 8088] && cd /d ""%~dp0backend"" && echo =================================================== && echo   ERP Backend Server (Go net/http - Port 8088) && echo =================================================== && echo. && go run ./cmd/server/main.go"
exit /b 0

:start_fe_only
echo [*] Menjalankan Frontend Backoffice di jendela baru...
echo     URL Client : http://localhost:5173
echo.
start "ERP Frontend (SvelteKit - Port 5173)" cmd /k "title ERP Frontend [Port 5173] && cd /d ""%~dp0frontend"" && echo =================================================== && echo   ERP Frontend Backoffice (SvelteKit - Port 5173) && echo =================================================== && echo. && npm run dev:backoffice"
exit /b 0

:start_all
echo [*] 1. Menjalankan Backend Go di jendela terpisah...
echo        - Port : 8088
echo        - URL  : http://localhost:8088
echo.
start "ERP Backend (Go - Port 8088)" cmd /k "title ERP Backend [Port 8088] && cd /d ""%~dp0backend"" && echo =================================================== && echo   ERP Backend Server (Go net/http - Port 8088) && echo =================================================== && echo. && go run ./cmd/server/main.go"

:: Beri jeda 2 detik agar backend inisialisasi terlebih dahulu
timeout /t 2 /nobreak >nul

echo [*] 2. Menjalankan Frontend Backoffice di jendela terpisah...
echo        - Port : 5173
echo        - URL  : http://localhost:5173
echo.
start "ERP Frontend (SvelteKit - Port 5173)" cmd /k "title ERP Frontend [Port 5173] && cd /d ""%~dp0frontend"" && echo =================================================== && echo   ERP Frontend Backoffice (SvelteKit - Port 5173) && echo =================================================== && echo. && npm run dev:backoffice"

echo ===================================================================
echo  Status: Backend dan Frontend sedang berjalan di 2 jendela terpisah!
echo.
echo  - Backend API : http://localhost:8088
echo  - Backoffice  : http://localhost:5173
echo.
echo  Tip:
echo  1. Pastikan MySQL aktif (default port 3306).
echo  2. Untuk mematikan, cukup tutup masing-masing jendela atau
echo     tekan Ctrl+C di jendela yang bersangkutan.
echo  3. Anda juga dapat menjalankan stop-dev.bat untuk menghentikan
echo     kedua proses secara instan.
echo ===================================================================
echo.
echo Tekan tombol apa saja untuk menutup jendela launcher ini...
pause >nul
exit /b 0
