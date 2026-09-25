package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory"
	"github.com/erp-retail/backend/internal/platform/config"
	"github.com/erp-retail/backend/internal/platform/database"
	"github.com/erp-retail/backend/internal/platform/docs"
	"github.com/erp-retail/backend/internal/platform/logger"
	"github.com/erp-retail/backend/internal/shared/audit"
	"github.com/erp-retail/backend/internal/shared/auth"
	"github.com/erp-retail/backend/internal/shared/event"
	"github.com/erp-retail/backend/internal/shared/license"
)

func main() {
	// ── 1. Setup logger ──────────────────────────────────────────────────────
	// Logger dibuat pertama karena semua komponen lain butuh logger.
	log := logger.New()

	// ── 2. Baca konfigurasi ─────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Error("gagal membaca konfigurasi", "error", err)
		os.Exit(1)
	}
	log.Info("konfigurasi berhasil dimuat", "port", cfg.Port, "modules", cfg.EnabledModules)

	// ── 3. Inisialisasi Database Connection Pool
	db, err := database.NewConnection(cfg.DatabaseDSN)
	if err != nil {
		log.Error("Gagal terhubung ke database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("koneksi database berhasil")

	// ── 4. Buat Event Bus ───────────────────────────────────────────────────
	// Event bus adalah infrastruktur komunikasi asinkron antar modul.
	// Dibuat di sini dan di-inject ke modul-modul yang membutuhkannya.
	bus := event.New()

	// ── 5. Baca Lisensi ─────────────────────────────────────────────────────
	// Lisensi dibaca SEKALI saat startup. Hasilnya menentukan modul mana yang di-mount.
	lic := license.New(cfg.EnabledModules)

	// ── 6. Setup Router ─────────────────────────────────────────────────────
	// net/http standar Go menyediakan http.ServeMux sebagai router.
	// Sejak Go 1.22, ServeMux mendukung method pattern (mis. "GET /path").
	mux := http.NewServeMux()

	// Route kesehatan server — selalu aktif tanpa auth
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"0.1.0"}`)
	})

	// Route dokumentasi API interaktif:
	// - GET /docs         -> Scalar API Reference (UI modern, dark mode)
	// - GET /swagger      -> Swagger UI (UI klasik)
	// - GET /openapi.yaml -> Raw OpenAPI 3.0 specification
	docs.Register(mux)

	// Pastikan folder uploads tersedia dan daftarkan static file server
	_ = os.MkdirAll("./uploads/products", 0755)
	_ = os.MkdirAll("./uploads/categories", 0755)
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// ── 7. Mount middleware auth ─────────────────────────────────────────────
	// auth.Middleware mengembalikan fungsi pembungkus handler.
	// Pola ini disebut "middleware chain" atau "decorator pattern".
	authMiddleware := auth.Middleware(cfg.JWTSecret)

	// ── 7.5 Inisialisasi & Mount Shared Context: Autentikasi, Pengguna & PBAC ───────
	// Auth & Hak Akses (PBAC) adalah Shared Context yang selalu aktif untuk semua instalasi ERP
	userRepo := auth.NewUserRepository(db)
	authService := auth.NewService(userRepo, cfg.JWTSecret, 24*time.Hour)

	permRepo := auth.NewPermissionRepository(db)
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	permService, err := auth.NewPermissionService(initCtx, permRepo)
	initCancel()
	if err != nil {
		log.Error("gagal memuat permission service", "error", err)
		os.Exit(1)
	}

	authHandler := auth.NewHandler(authService, permService, log)
	authHandler.RegisterRoutes(mux, authMiddleware)
	log.Info("shared context auth, users & pbac di-mount")

	// ── 7.6 Inisialisasi & Mount Shared Context: Audit Log ──────────────────
	// Audit Log adalah fondasi akuntabilitas yang selalu aktif untuk merekam jejak aktivitas staf/admin
	auditRepo := audit.NewRepository(db)
	auditService := audit.NewService(auditRepo, log)
	auditService.SubscribeEventBus(bus)
	auditHandler := audit.NewHandler(auditService, permService, log)
	auditHandler.RegisterRoutes(mux, authMiddleware)
	log.Info("shared context audit log di-mount dan subscribe ke event bus")

	// ── 8. Mount Modul Berdasarkan Lisensi ──────────────────────────────────
	// INI ADALAH INTI DARI MEKANISME LISENSI:
	// Modul yang tidak di-unlock TIDAK AKAN di-mount ke router maupun event bus.
	// Route-nya tidak terdaftar, handler-nya tidak ada, subscriber event-nya tidak subscribe.
	// Bukan sekadar disembunyikan di UI — betul-betul tidak ada di server.

	if lic.Enabled("inventory") {
		inventoryMod := inventory.New(db, bus, log)
		inventoryMod.Register(mux, authMiddleware, permService, authService)
		log.Info("modul inventory di-mount dengan otorisasi pbac")
	}

	// Tambahkan modul lain di sini seiring dikembangkan:
	// if lic.Enabled("purchasing") { ... }
	// if lic.Enabled("sales")      { ... }
	// if lic.Enabled("finance")    { ... }
	// if lic.Enabled("commission") { ... }
	// if lic.Enabled("ecommerce")  { ... }

	// ── 9. CORS Middleware ───────────────────────────────────────────────────
	// CORS (Cross-Origin Resource Sharing) diperlukan agar frontend SvelteKit
	// yang berjalan di localhost:5173 bisa memanggil API di localhost:8080.
	// Tanpa CORS, browser akan memblokir request lintas origin demi keamanan.
	//
	// Analoginya: CORS seperti "surat izin tamu" — server backend memberitahu
	// browser bahwa frontend dari alamat tertentu boleh mengakses API-nya.
	corsHandler := corsMiddleware(mux)

	// ── 10. Jalankan HTTP Server ────────────────────────────────────────────
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      corsHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Goroutine untuk menangani graceful shutdown saat menerima sinyal OS (Ctrl+C, SIGTERM).
	// Ini penting agar request yang sedang berjalan bisa selesai dulu sebelum server berhenti.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server berjalan", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Blok di sini sampai ada sinyal shutdown
	<-quit
	log.Info("menerima sinyal shutdown, menghentikan server...")

	// Beri waktu maksimal 30 detik untuk request yang sedang berjalan selesai
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("gagal graceful shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("server berhenti dengan bersih")
}

// corsMiddleware membungkus handler HTTP dengan header CORS yang diperlukan.
// Fungsi ini adalah implementasi sederhana yang cukup untuk development.
// Di production, bisa diganti dengan library CORS yang lebih lengkap jika diperlukan.
//
// Alur CORS:
// 1. Browser mengirim request preflight (OPTIONS) sebelum request asli
// 2. Server membalas dengan header izin (Access-Control-Allow-*)
// 3. Jika header izin cocok, browser melanjutkan request asli
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Izinkan origin frontend dev server (mendukung port 5173, 5174, dll)
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		}
		// Method HTTP yang diizinkan
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		// Header yang boleh dikirim oleh frontend
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// Izinkan browser menyimpan cookie/credentials
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Preflight request: browser mengirim OPTIONS dulu sebelum request asli
		// Kita langsung balas 204 (No Content) tanpa meneruskan ke handler
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
