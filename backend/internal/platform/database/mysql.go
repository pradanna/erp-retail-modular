package database

import (
	"database/sql"
	"fmt"
	"time"

	// Import blank identifier "_" digunakan untuk mendaftarkan driver MySQL ke database/sql.
	// Driver ini tidak dipanggil langsung, melainkan bekerja di belakang interface standar Go database/sql.
	_ "github.com/go-sql-driver/mysql"
)

// NewConnection membuka pool koneksi ke MySQL menggunakan DSN yang diberikan.
// Fungsi ini mengembalikan *sql.DB yang aman digunakan secara konkuren (thread-safe)
// oleh banyak goroutine/request sekaligus.
func NewConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("gagal menginisialisasi driver database: %w", err)
	}

	// Konfigurasi connection pooling
	// Ini penting agar server tidak kehabisan koneksi dan performa tetap stabil.
	db.SetMaxOpenConns(25)                 // Maksimal 25 koneksi terbuka bersamaan
	db.SetMaxIdleConns(10)                 // Maksimal 10 koneksi stanby/idle
	db.SetConnMaxLifetime(5 * time.Minute) // Koneksi di-refresh tiap 5 menit
	db.SetConnMaxIdleTime(2 * time.Minute) // Koneksi idle dibersihkan setelah 2 menit

	// sql.Open hanya memvalidasi format string DSN, belum benar-benar kontak ke database.
	// Kita gunakan Ping() untuk memastikan koneksi fisik ke MySQL benar-benar tersambung.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("gagal terhubung ke database MySQL (pastikan Laragon/MySQL sudah aktif): %w", err)
	}

	return db, nil
}
