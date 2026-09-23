package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config menyimpan seluruh konfigurasi aplikasi yang dibaca dari environment variables.
// Pola ini disebut "config struct" — dependensi dikumpulkan di satu tempat,
// lalu di-inject ke setiap komponen yang membutuhkannya.
type Config struct {
	// Server
	Port string

	// Database
	DatabaseDSN string

	// License — daftar nama modul yang aktif untuk instalasi ini
	EnabledModules []string

	// Auth
	JWTSecret string
}

// Load membaca konfigurasi dari environment variables dan mengembalikan struct Config.
// Fungsi ini dipanggil SEKALI di main.go, hasilnya di-pass ke seluruh komponen.
func Load() (*Config, error) {
	// Membaca file .env jika ada (sangat berguna untuk development lokal).
	// Sengaja abaikan error jika file .env tidak ditemukan, karena di server production
	// konfigurasi biasanya langsung di-inject ke sistem operasi (OS env) oleh Docker/VPS.
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("env DB_DSN wajib diisi (contoh: root:@tcp(localhost:3306)/erp_retail?parseTime=true)")
	}

	jwtSecret := getEnv("JWT_SECRET", "changeme-in-production")

	// ENABLED_MODULES berisi daftar nama modul yang dipisah koma.
	// Contoh: ENABLED_MODULES=inventory,purchasing,sales
	modulesEnv := getEnv("ENABLED_MODULES", "inventory")
	modules := []string{}
	for _, m := range strings.Split(modulesEnv, ",") {
		m = strings.TrimSpace(strings.ToLower(m))
		if m != "" {
			modules = append(modules, m)
		}
	}

	return &Config{
		Port:           port,
		DatabaseDSN:    dsn,
		EnabledModules: modules,
		JWTSecret:      jwtSecret,
	}, nil
}

// getEnv membaca env var; jika tidak ada, kembalikan nilai default.
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
