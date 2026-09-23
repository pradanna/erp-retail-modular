package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	"github.com/erp-retail/backend/internal/modules/inventory"
	"github.com/erp-retail/backend/internal/shared/auth"
)

func main() {
	// 1. Muat environment variables dari .env
	if err := godotenv.Load(); err != nil {
		if errFallback := godotenv.Load("backend/.env"); errFallback != nil {
			log.Println("Peringatan: Tidak dapat memuat file .env atau backend/.env, membaca dari environment sistem...")
		}
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("Error: DB_DSN tidak ditemukan di environment (.env)")
	}

	// 2. Hubungkan ke database MySQL
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Gagal terhubung ke database MySQL: %v", err)
	}

	// 3. Baca flag module (opsional)
	targetModule := flag.String("module", "all", "Modul yang akan di-seed (contoh: auth, inventory, all)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("==================================================")
	fmt.Printf("🌱 Running Database Seeders: module = '%s'\n", *targetModule)
	fmt.Println("==================================================")

	mod := strings.ToLower(strings.TrimSpace(*targetModule))

	// Shared: Autentikasi & Akun Pengguna Bawaan
	if mod == "all" || mod == "auth" || mod == "shared" {
		fmt.Println("\n🔐 [Shared: Auth] Menyuntikkan akun pengguna standar...")
		count, err := auth.SeedDefaultUsers(ctx, db)
		if err != nil {
			log.Fatalf("❌ Gagal seeding akun auth: %v", err)
		}
		if count > 0 {
			fmt.Printf("   ✅ Berhasil menambahkan %d akun pengguna standar baru!\n", count)
		} else {
			fmt.Println("   ℹ️  Semua akun pengguna standar sudah ada di database (idempoten / no-op).")
		}
	}

	// Modul: Inventory
	if mod == "all" || mod == "inventory" {
		fmt.Println("\n📦 [Modul: Inventory] Menyuntikkan template garansi standar...")
		count, err := inventory.SeedWarrantyPolicies(ctx, db)
		if err != nil {
			log.Fatalf("❌ Gagal seeding garansi inventory: %v", err)
		}
		if count > 0 {
			fmt.Printf("   ✅ Berhasil menambahkan %d template kebijakan garansi baru!\n", count)
		} else {
			fmt.Println("   ℹ️  Semua template garansi standar sudah ada di database (idempoten / no-op).")
		}
	}

	fmt.Println("\n✨ Selesai! Seluruh seeder berhasil dieksekusi.")
}
