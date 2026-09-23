package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {
	// 1. Muat environment variables dari .env (cek direct dir, lalu fallback ke backend/.env)
	if err := godotenv.Load(); err != nil {
		if errFallback := godotenv.Load("backend/.env"); errFallback != nil {
			log.Println("Peringatan: Tidak dapat memuat file .env atau backend/.env, membaca dari environment sistem...")
		}
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("Error: DB_DSN tidak ditemukan di environment (.env)")
	}

	// 2. Pastikan database ada (jika belum ada, buat otomatis)
	ensureDatabaseExists(dsn)

	// 3. Hubungkan ke database MySQL yang dituju
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Gagal terhubung (ping) ke database: %v", err)
	}

	// 4. Set dialect database ke MySQL untuk Goose
	if err := goose.SetDialect("mysql"); err != nil {
		log.Fatalf("Gagal mengatur dialect goose: %v", err)
	}

	// 4. Baca argumen command line (contoh: up, down, status, reset)
	flag.Parse()
	args := flag.Args()

	command := "up"
	if len(args) > 0 {
		command = strings.ToLower(args[0])
	}

	// 5. Tentukan modul yang akan dimigrasi
	// Selalu jalankan migrasi 'shared' terlebih dahulu sebelum modul-modul bisnis
	modules := []string{"shared", "inventory"} // default modul
	enabledModulesEnv := os.Getenv("ENABLED_MODULES")
	if enabledModulesEnv != "" {
		configuredModules := strings.Split(enabledModulesEnv, ",")
		modules = []string{"shared"}
		for _, m := range configuredModules {
			trimmed := strings.TrimSpace(m)
			if trimmed != "" && trimmed != "shared" {
				modules = append(modules, trimmed)
			}
		}
	}

	fmt.Println("==================================================")
	fmt.Printf("🚀 Running Database Migrations: command = '%s'\n", command)
	fmt.Println("==================================================")

	baseDir := "migrations"
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		if _, errBackend := os.Stat(filepath.Join("backend", "migrations")); errBackend == nil {
			baseDir = filepath.Join("backend", "migrations")
		}
	}

	for _, mod := range modules {
		mod = strings.TrimSpace(mod)
		modDir := filepath.Join(baseDir, mod)

		// Cek apakah folder migrasi modul ada
		if _, err := os.Stat(modDir); os.IsNotExist(err) {
			continue
		}

		// Gunakan tabel goose version terpisah per modul agar migrasi antar modul independen
		tableName := fmt.Sprintf("goose_%s_version", mod)
		if mod == "inventory" {
			// Jika sebelumnya memakai nama default goose_db_version, ganti nama secara mulus
			var count int
			_ = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'goose_db_version'").Scan(&count)
			if count > 0 {
				var newCount int
				_ = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'goose_inventory_version'").Scan(&newCount)
				if newCount == 0 {
					_, _ = db.Exec("RENAME TABLE goose_db_version TO goose_inventory_version")
				}
			}
		}
		goose.SetTableName(tableName)

		fmt.Printf("\n📦 [Modul: %s] Menjalankan migrasi dari: %s (tracking table: %s)\n", mod, modDir, tableName)

		switch command {
		case "up":
			if err := goose.Up(db, modDir); err != nil {
				log.Fatalf("Gagal migrate up pada modul %s: %v", mod, err)
			}
		case "down":
			if err := goose.Down(db, modDir); err != nil {
				log.Fatalf("Gagal migrate down pada modul %s: %v", mod, err)
			}
		case "status":
			if err := goose.Status(db, modDir); err != nil {
				log.Fatalf("Gagal cek status migrasi pada modul %s: %v", mod, err)
			}
		case "reset":
			if err := goose.Reset(db, modDir); err != nil {
				log.Fatalf("Gagal reset migrasi pada modul %s: %v", mod, err)
			}
		default:
			log.Fatalf("Perintah tidak dikenal: %s (Gunakan: up, down, status, reset)", command)
		}
	}

	fmt.Println("\n✅ Selesai! Semua migrasi berhasil diproses.")
}

// ensureDatabaseExists memeriksa apakah database target sudah ada di MySQL.
// Jika belum ada, fungsi ini akan membuatnya secara otomatis.
func ensureDatabaseExists(dsn string) {
	// Contoh DSN: root:@tcp(localhost:3306)/erp_retail?parseTime=true
	parts := strings.Split(dsn, "/")
	if len(parts) < 2 {
		return
	}

	rootDSN := parts[0] + "/"
	dbNameAndParams := parts[1]
	dbName := strings.Split(dbNameAndParams, "?")[0]

	rootDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		log.Printf("Peringatan: Gagal koneksi ke root MySQL: %v", err)
		return
	}
	defer rootDB.Close()

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", dbName)
	if _, err := rootDB.Exec(query); err != nil {
		log.Printf("Peringatan: Gagal memastikan database ada: %v", err)
		return
	}
	fmt.Printf("✨ Database '%s' siap digunakan.\n", dbName)
}

