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
	fmt.Printf("[RUN] Running Database Seeders: module = '%s'\n", *targetModule)
	fmt.Println("==================================================")

	mod := strings.ToLower(strings.TrimSpace(*targetModule))

	// Shared: Peran & Izin PBAC
	if mod == "all" || mod == "auth" || mod == "shared" || mod == "roles" {
		fmt.Println("\n[Shared: Roles & Permissions] Menyuntikkan peran dan izin standar...")

		roleCount, err := auth.SeedRoles(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding peran: %v", err)
		}
		if roleCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d peran baru.\n", roleCount)
		} else {
			fmt.Println("   [INFO] Semua peran sistem sudah terdaftar (idempoten).")
		}

		permCount, err := auth.SeedPermissions(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding izin permission: %v", err)
		}
		if permCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d izin kapabilitas baru.\n", permCount)
		} else {
			fmt.Println("   [INFO] Semua izin sistem sudah terdaftar (idempoten).")
		}

		relCount, err := auth.SeedDefaultRolePermissions(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding relasi peran-izin: %v", err)
		}
		if relCount > 0 {
			fmt.Printf("   [OK] Berhasil menyambungkan %d pemetaan izin peran default.\n", relCount)
		} else {
			fmt.Println("   [INFO] Seluruh pemetaan izin peran sudah sinkron (idempoten).")
		}
	}

	// Shared: Autentikasi & Akun Pengguna Bawaan
	if mod == "all" || mod == "auth" || mod == "shared" {
		fmt.Println("\n[Shared: Auth] Menyuntikkan akun pengguna standar...")
		count, err := auth.SeedDefaultUsers(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding akun auth: %v", err)
		}
		if count > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d akun pengguna standar baru.\n", count)
		} else {
			fmt.Println("   [INFO] Semua akun pengguna standar sudah ada di database (idempoten).")
		}
	}

	// Modul: Inventory
	if mod == "all" || mod == "inventory" {
		fmt.Println("\n[Modul: Inventory] Menyuntikkan master lokasi & cabang standar...")
		locCount, err := inventory.SeedLocations(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding lokasi: %v", err)
		}
		if locCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d lokasi/cabang baru.\n", locCount)
		} else {
			fmt.Println("   [INFO] Seluruh master lokasi standar sudah terdaftar (idempoten).")
		}

		fmt.Println("\n[Modul: Inventory] Menyuntikkan struktur hierarki kategori produk...")
		catCount, err := inventory.SeedCategories(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding kategori: %v", err)
		}
		if catCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d kategori (induk & sub-kategori) baru.\n", catCount)
		} else {
			fmt.Println("   [INFO] Seluruh hierarki kategori standar sudah terdaftar (idempoten).")
		}

		fmt.Println("\n[Modul: Inventory] Menyuntikkan katalog master produk & barcode...")
		prodCount, err := inventory.SeedProducts(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding produk: %v", err)
		}
		if prodCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d produk ritel beserta barcode dan alokasi stok awal.\n", prodCount)
		} else {
			fmt.Println("   [INFO] Seluruh katalog master produk sudah terdaftar (idempoten).")
		}

		fmt.Println("\n[Modul: Inventory] Menyuntikkan foto katalog produk ritel...")
		imgCount, err := inventory.SeedProductImages(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding foto produk: %v", err)
		}
		if imgCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d foto produk ritel (termasuk penyimpanan fisik offline).\n", imgCount)
		} else {
			fmt.Println("   [INFO] Semua foto produk sudah terdaftar dan sinkron (idempoten).")
		}

		fmt.Println("\n[Modul: Inventory] Menyuntikkan template garansi standar...")
		warrCount, err := inventory.SeedWarrantyPolicies(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding garansi inventory: %v", err)
		}
		if warrCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d template kebijakan garansi baru.\n", warrCount)
		} else {
			fmt.Println("   [INFO] Semua template garansi standar sudah ada di database (idempoten).")
		}

		fmt.Println("\n[Modul: Inventory] Menyuntikkan saldo stok awal per cabang...")
		stkCount, err := inventory.SeedStocks(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding stok: %v", err)
		}
		if stkCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d saldo stok cabang baru.\n", stkCount)
		} else {
			fmt.Println("   [INFO] Seluruh saldo stok cabang sudah sinkron (idempoten).")
		}

		fmt.Println("\n[Modul: Inventory] Menyuntikkan unit fisik berserial & IMEI...")
		snCount, err := inventory.SeedSerialUnits(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding serial & IMEI: %v", err)
		}
		if snCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d unit fisik berserial & IMEI baru.\n", snCount)
		} else {
			fmt.Println("   [INFO] Seluruh unit fisik berserial sudah tersinkronkan (idempoten).")
		}

		fmt.Println("\n[Modul: Inventory] Menyuntikkan promo harga khusus cabang (price overrides)...")
		promoCount, err := inventory.SeedPriceOverrides(ctx, db)
		if err != nil {
			log.Fatalf("[ERROR] Gagal seeding promo harga: %v", err)
		}
		if promoCount > 0 {
			fmt.Printf("   [OK] Berhasil menambahkan %d promo harga khusus cabang baru.\n", promoCount)
		} else {
			fmt.Println("   [INFO] Seluruh promo harga khusus cabang sudah tersinkronkan (idempoten).")
		}
	}

	fmt.Println("\n[SUCCESS] Selesai! Seluruh seeder berhasil dieksekusi.")
}
