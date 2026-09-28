$ports = @(8088, 5173)

foreach ($port in $ports) {
    $connections = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue
    if ($connections) {
        $pids = $connections | Select-Object -ExpandProperty OwningProcess -Unique
        foreach ($procId in $pids) {
            try {
                $proc = Get-Process -Id $procId -ErrorAction Stop
                Stop-Process -Id $procId -Force
                Write-Host " [OK] Berhasil menghentikan proses '$($proc.ProcessName)' (PID $procId) pada port $port" -ForegroundColor Green
            } catch {
                Write-Host " [!] Gagal menghentikan PID $procId pada port $port" -ForegroundColor Yellow
            }
        }
    } else {
        Write-Host " [-] Port $port bebas (tidak ada proses aktif)." -ForegroundColor Gray
    }
}
