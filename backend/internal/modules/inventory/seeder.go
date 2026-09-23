package inventory

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/erp-retail/backend/pkg/uid"
)

// WarrantyPolicySeed mendefinisikan struktur data template garansi standar untuk seeder.
type WarrantyPolicySeed struct {
	Name              string
	Type              string
	DurationMonths    int
	DurationDays      int
	Coverage          string
	ClaimInstructions string
}

// DefaultWarrantyPolicies adalah kumpulan template master garansi standar ritel elektronik.
var DefaultWarrantyPolicies = []WarrantyPolicySeed{
	// --- 1. Garansi Toko (Store Warranties) ---
	{
		Name:              "Garansi Toko Tukar Baru 7 Hari",
		Type:              "toko",
		DurationMonths:    0,
		DurationDays:      7,
		Coverage:          "Penggantian unit baru (1-to-1 replacement) apabila ditemukan cacat produksi atau kerusakan fungsi pabrik dalam 7 hari pertama sejak tanggal pembelian.",
		ClaimInstructions: "Bawa unit lengkap beserta dus box asli, kelengkapan aksesoris, kartu garansi, dan invoice pembelian fisik ke meja customer service cabang toko terdekat.",
	},
	{
		Name:              "Garansi Toko VIP Member 14 Hari",
		Type:              "toko",
		DurationMonths:    0,
		DurationDays:      14,
		Coverage:          "Jaminan tukar baru unit 14 hari khusus pelanggan berstatus Member VIP / Gold apabila terjadi kendala fungsi pabrikan.",
		ClaimInstructions: "Tunjukkan nomor keanggotaan VIP, invoice pembelian, dan bawa unit lengkap ke cabang toko terdekat.",
	},
	{
		Name:              "Garansi Toko Cuci Gudang / Display 3 Hari",
		Type:              "toko",
		DurationMonths:    0,
		DurationDays:      3,
		Coverage:          "Garansi pengetesan fungsi hardware selama 3 hari khusus produk obral cuci gudang atau unit ex-display (tidak mencakup goresan/kondisi kosmetik fisik bawaan).",
		ClaimInstructions: "Bawa unit dan nota khusus obral cuci gudang ke cabang toko tempat transaksi dilakukan.",
	},

	// --- 2. Garansi Resmi Pabrik (Manufacturer / Distributor Warranties) ---
	{
		Name:              "Garansi Resmi Samsung Indonesia 12 Bulan",
		Type:              "pabrik",
		DurationMonths:    12,
		DurationDays:      0,
		Coverage:          "Perbaikan dan penggantian suku cadang asli gratis di seluruh Samsung Authorized Service Center di Indonesia untuk kerusakan non-human error.",
		ClaimInstructions: "Bawa unit dan invoice/faktur pembelian ke Service Center Samsung terdekat atau hubungi call center bebas pulsa 0800-112-8888.",
	},
	{
		Name:              "Garansi Resmi LG Electronics 12 Bulan",
		Type:              "pabrik",
		DurationMonths:    12,
		DurationDays:      0,
		Coverage:          "Servis dan penggantian sparepart gratis dari teknisi resmi PT LG Electronics Indonesia sesuai ketentuan garansi standar.",
		ClaimInstructions: "Kunjungi LG Service Center resmi terdekat dengan membawa nota pembelian atau hubungi Customer Care LG di 14010.",
	},
	{
		Name:              "Garansi Resmi Apple Indonesia (TAM/iBox/Digimap) 12 Bulan",
		Type:              "pabrik",
		DurationMonths:    12,
		DurationDays:      0,
		Coverage:          "Garansi perangkat keras Apple 1 tahun dari distributor resmi Indonesia di seluruh Apple Authorized Service Provider (AASP).",
		ClaimInstructions: "Kunjungi Apple Authorized Service Provider (iBox, Digimap, atau Mitra Care) terdekat dengan membawa unit iPhone/iPad/Mac dan bukti nota pembelian ritel.",
	},
	{
		Name:              "Garansi Resmi Sony Indonesia 12 Bulan",
		Type:              "pabrik",
		DurationMonths:    12,
		DurationDays:      0,
		Coverage:          "Perbaikan gratis suku cadang dan jasa di Sony Authorized Service Center resmi seluruh Indonesia.",
		ClaimInstructions: "Daftarkan serial number produk di situs resmi Sony Indonesia dan bawa kartu jaminan serta invoice ke Sony Service Center.",
	},
	{
		Name:              "Garansi Resmi Asus Indonesia 24 Bulan",
		Type:              "pabrik",
		DurationMonths:    24,
		DurationDays:      0,
		Coverage:          "Garansi global servis dan suku cadang selama 2 tahun di seluruh Asus Service Center resmi untuk laptop dan komponen PC.",
		ClaimInstructions: "Bawa unit laptop dan kartu jaminan resmi ke Asus Service Center terdekat.",
	},
	{
		Name:              "Garansi Resmi Xiaomi Indonesia 15 Bulan",
		Type:              "pabrik",
		DurationMonths:    15,
		DurationDays:      0,
		Coverage:          "Jaminan perbaikan dan suku cadang resmi Xiaomi selama 15 bulan untuk smartphone dan ekosistem perangkat pintar.",
		ClaimInstructions: "Kunjungi Xiaomi Exclusive Service Center terdekat dengan menunjukkan unit dan nota pembelian.",
	},
	{
		Name:              "Garansi Resmi Kompresor Inverter 10 Tahun",
		Type:              "pabrik",
		DurationMonths:    120,
		DurationDays:      0,
		Coverage:          "Jaminan penggantian sparepart motor kompresor inverter pendingin (kulkas/AC) selama 10 tahun.",
		ClaimInstructions: "Hubungi call center resmi brand produsen dengan menyebutkan nomor seri produk dan melampirkan invoice resmi pembelian ritel.",
	},
}

// SeedWarrantyPolicies menyuntikkan template garansi bawaan ke tabel inv_warranty_policies secara idempoten.
// Jika nama template sudah ada, baris tersebut dilewati agar tidak terjadi duplikasi data.
func SeedWarrantyPolicies(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	insertedCount := 0

	queryCheck := `SELECT id FROM inv_warranty_policies WHERE name = ? LIMIT 1`
	queryInsert := `
		INSERT INTO inv_warranty_policies (
			id, name, type, duration_months, duration_days, coverage, claim_instructions,
			is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?)`

	for _, item := range DefaultWarrantyPolicies {
		var existingID string
		err := db.QueryRowContext(ctx, queryCheck, item.Name).Scan(&existingID)
		if err == nil {
			// Template dengan nama ini sudah ada, skip untuk menjaga idempoten
			continue
		}
		if err != sql.ErrNoRows {
			return insertedCount, fmt.Errorf("gagal cek keberadaan template garansi '%s': %w", item.Name, err)
		}

		id := uid.New()
		_, err = db.ExecContext(ctx, queryInsert,
			id, item.Name, item.Type, item.DurationMonths, item.DurationDays,
			item.Coverage, item.ClaimInstructions, now, now,
		)
		if err != nil {
			return insertedCount, fmt.Errorf("gagal insert template garansi '%s': %w", item.Name, err)
		}
		insertedCount++
	}

	return insertedCount, nil
}
