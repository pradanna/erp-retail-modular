package logger

import (
	"log/slog"
	"os"
)

// New membuat dan mengembalikan logger struktural berbasis log/slog (standar Go sejak 1.21).
// Output berformat JSON agar mudah di-baca oleh log aggregator (Loki, Datadog, dsb).
//
// Kenapa log/slog bukan fmt.Println?
// - Structured logging: setiap field punya key, mudah di-filter/query
// - JSON output: mesin (monitoring tool) bisa parse otomatis
// - Level-based: hanya tampilkan log sesuai level (DEBUG hanya saat development)
func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Ubah ke slog.LevelDebug untuk development verbose
	})
	return slog.New(handler)
}
