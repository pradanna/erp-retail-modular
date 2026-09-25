package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/erp-retail/backend/pkg/uid"
)

// ============================================================================
// 1. DATA SEEDER: LOKASI (CABANG FISIK & GUDANG ONLINE)
// ============================================================================

// LocationSeed mendefinisikan template lokasi cabang dan gudang retail.
type LocationSeed struct {
	Code      string
	Name      string
	Type      string
	Address   string
	Latitude  *float64
	Longitude *float64
}

func floatPtr(v float64) *float64 { return &v }

// DefaultLocations memuat daftar lokasi standar untuk operasional toko ritel dengan koordinat GPS nyata.
var DefaultLocations = []LocationSeed{
	{
		Code:      "WH-JKT-01",
		Name:      "Gudang Utama Distribusi Jakarta",
		Type:      "physical",
		Address:   "Kawasan Industri Pulo Gadung Kav. 12-14, Jakarta Timur, DKI Jakarta",
		Latitude:  floatPtr(-6.191420),
		Longitude: floatPtr(106.912630),
	},
	{
		Code:      "STR-SBY-01",
		Name:      "Toko Retail Flagship Surabaya",
		Type:      "physical",
		Address:   "Jl. Basuki Rahmat No. 45-47, Genteng, Surabaya, Jawa Timur",
		Latitude:  floatPtr(-7.262510),
		Longitude: floatPtr(112.742320),
	},
	{
		Code:      "STR-BDG-01",
		Name:      "Toko Cabang Dago Bandung",
		Type:      "physical",
		Address:   "Jl. Ir. H. Djuanda No. 102, Coblong, Bandung, Jawa Barat",
		Latitude:  floatPtr(-6.890520),
		Longitude: floatPtr(107.610410),
	},
	{
		Code:      "WH-ECOM-01",
		Name:      "Gudang Toko Online Storefront",
		Type:      "online",
		Address:   "Kawasan Pergudangan Marunda Center Blok B3, Tarumajaya, Bekasi",
		Latitude:  floatPtr(-6.102340),
		Longitude: floatPtr(106.974510),
	},
}

// SeedLocations menyuntikkan lokasi toko & gudang secara idempoten berdasarkan kode lokasi.
func SeedLocations(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	insertedCount := 0

	queryCheck := `SELECT id, latitude, longitude FROM inv_locations WHERE code = ? LIMIT 1`
	queryInsert := `
		INSERT INTO inv_locations (
			id, code, name, type, address, latitude, longitude, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?)`
	queryUpdateCoords := `UPDATE inv_locations SET latitude = ?, longitude = ? WHERE id = ? AND latitude IS NULL`

	for _, loc := range DefaultLocations {
		var existingID string
		var existingLat, existingLng sql.NullFloat64
		err := db.QueryRowContext(ctx, queryCheck, loc.Code).Scan(&existingID, &existingLat, &existingLng)
		if err == nil {
			// Jika lokasi sudah ada tapi koordinatnya masih kosong, sinkronkan koordinat
			if !existingLat.Valid && loc.Latitude != nil {
				_, _ = db.ExecContext(ctx, queryUpdateCoords, loc.Latitude, loc.Longitude, existingID)
			}
			continue // Sudah ada, jaga idempotensi
		}
		if err != sql.ErrNoRows {
			return insertedCount, fmt.Errorf("gagal cek lokasi '%s': %w", loc.Code, err)
		}

		id := uid.New()
		_, err = db.ExecContext(ctx, queryInsert, id, loc.Code, loc.Name, loc.Type, loc.Address, loc.Latitude, loc.Longitude, now, now)
		if err != nil {
			return insertedCount, fmt.Errorf("gagal insert lokasi '%s': %w", loc.Code, err)
		}
		insertedCount++
	}

	return insertedCount, nil
}

// ============================================================================
// 2. DATA SEEDER: KATEGORI PRODUK (INDUK & HIERARKI SUB-KATEGORI)
// ============================================================================

// CategorySeed mendefinisikan template kategori katalog retail.
type CategorySeed struct {
	Name     string
	Parent   string // Kosong jika Kategori Utama / Induk
	ImageURL string
}

// DefaultCategories memuat struktur hierarki kategori toko retail & elektronik.
var DefaultCategories = []CategorySeed{
	// --- Kategori Utama (Induk) ---
	{Name: "Elektronik Rumah Tangga", Parent: ""},
	{Name: "Gadget & Smartphone", Parent: ""},
	{Name: "Komputer & Laptop", Parent: ""},
	{Name: "Audio & Visual", Parent: ""},
	{Name: "Aksesoris & Perlengkapan", Parent: ""},

	// --- Sub-Kategori: Elektronik Rumah Tangga ---
	{Name: "Kulkas & Pendingin", Parent: "Elektronik Rumah Tangga"},
	{Name: "Mesin Cuci & Pengering", Parent: "Elektronik Rumah Tangga"},
	{Name: "Air Conditioner (AC)", Parent: "Elektronik Rumah Tangga"},

	// --- Sub-Kategori: Gadget & Smartphone ---
	{Name: "Smartphone Android", Parent: "Gadget & Smartphone"},
	{Name: "Apple iPhone", Parent: "Gadget & Smartphone"},
	{Name: "Tablet & E-Reader", Parent: "Gadget & Smartphone"},

	// --- Sub-Kategori: Komputer & Laptop ---
	{Name: "Laptop Bisnis & Kerja", Parent: "Komputer & Laptop"},
	{Name: "Laptop Gaming", Parent: "Komputer & Laptop"},
	{Name: "Aksesoris & Periferal PC", Parent: "Komputer & Laptop"},

	// --- Sub-Kategori: Audio & Visual ---
	{Name: "Smart TV 4K / 8K", Parent: "Audio & Visual"},
	{Name: "Soundbar & Speaker Aktif", Parent: "Audio & Visual"},
	{Name: "Headphone & TWS Earbuds", Parent: "Audio & Visual"},

	// --- Sub-Kategori: Aksesoris & Perlengkapan ---
	{Name: "Kabel Data & Fast Charger", Parent: "Aksesoris & Perlengkapan"},
	{Name: "Power Bank & Baterai", Parent: "Aksesoris & Perlengkapan"},
	{Name: "Case & Screen Protector", Parent: "Aksesoris & Perlengkapan"},
}

// SeedCategories menyuntikkan kategori induk dan turunan ke inv_categories secara idempoten.
func SeedCategories(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	insertedCount := 0

	// Map untuk menyimpan ID kategori berdasarkan nama agar sub-kategori bisa menunjuk parent_id yang tepat
	categoryIDMap := make(map[string]string)

	// Ambil semua kategori yang sudah ada di database untuk mengisi cache map
	rows, err := db.QueryContext(ctx, `SELECT id, name FROM inv_categories`)
	if err == nil {
		for rows.Next() {
			var id, name string
			if errScan := rows.Scan(&id, &name); errScan == nil {
				categoryIDMap[name] = id
			}
		}
		rows.Close()
	}

	queryInsert := `
		INSERT INTO inv_categories (id, name, parent_id, image_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	// 1. Eksekusi Kategori Utama (Parent == "")
	for _, cat := range DefaultCategories {
		if cat.Parent != "" {
			continue
		}
		if _, exists := categoryIDMap[cat.Name]; exists {
			continue // Sudah ada di database
		}

		id := uid.New()
		var imgURL sql.NullString
		if cat.ImageURL != "" {
			imgURL = sql.NullString{String: cat.ImageURL, Valid: true}
		}

		_, err := db.ExecContext(ctx, queryInsert, id, cat.Name, nil, imgURL, now, now)
		if err != nil {
			return insertedCount, fmt.Errorf("gagal insert kategori induk '%s': %w", cat.Name, err)
		}
		categoryIDMap[cat.Name] = id
		insertedCount++
	}

	// 2. Eksekusi Sub-Kategori (Parent != "")
	for _, cat := range DefaultCategories {
		if cat.Parent == "" {
			continue
		}
		if _, exists := categoryIDMap[cat.Name]; exists {
			continue // Sudah ada di database
		}

		parentID, parentExists := categoryIDMap[cat.Parent]
		if !parentExists {
			return insertedCount, fmt.Errorf("kategori induk '%s' tidak ditemukan untuk sub-kategori '%s'", cat.Parent, cat.Name)
		}

		id := uid.New()
		var imgURL sql.NullString
		if cat.ImageURL != "" {
			imgURL = sql.NullString{String: cat.ImageURL, Valid: true}
		}

		_, err := db.ExecContext(ctx, queryInsert, id, cat.Name, parentID, imgURL, now, now)
		if err != nil {
			return insertedCount, fmt.Errorf("gagal insert sub-kategori '%s': %w", cat.Name, err)
		}
		categoryIDMap[cat.Name] = id
		insertedCount++
	}

	return insertedCount, nil
}

// ============================================================================
// 3. DATA SEEDER: PRODUK & MULTI-BARCODE & STOK AWAL
// ============================================================================

// ProductImageSeed mendefinisikan template foto produk untuk seeder.
type ProductImageSeed struct {
	Filename  string
	RemoteURL string
	IsPrimary bool
	SortOrder int
}

// ProductSeed mendefinisikan template produk ritel beserta barcode dan variannya.
type ProductSeed struct {
	SKU                string
	CategoryName       string
	Name               string
	Brand              string
	Description        string
	Unit               string
	PurchasePrice      int64
	SellingPrice       int64
	IsPPN              bool
	FlagSerialTracking bool
	WeightGram         int
	AtributVarian      map[string]any
	PrimaryBarcode     string
	InitialStock       int
	Images             []ProductImageSeed
}

// DefaultProducts memuat 12 produk retail realistis (gabungan serial-tracked & non-serial) dengan foto berkualitas.
var DefaultProducts = []ProductSeed{
	{
		SKU:                "SAM-S24U-256-GRY",
		CategoryName:       "Smartphone Android",
		Name:               "Samsung Galaxy S24 Ultra 12GB/256GB Titanium Gray",
		Brand:              "Samsung",
		Description:        "Flagship smartphone Samsung dengan Galaxy AI, bodi titanium tangguh, kamera 200MP, dan stylus S-Pen bawaan.",
		Unit:               "pcs",
		PurchasePrice:      18500000,
		SellingPrice:       21999000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         232,
		AtributVarian:      map[string]any{"warna": "Titanium Gray", "ram": "12GB", "storage": "256GB"},
		PrimaryBarcode:     "8806095312345",
		InitialStock:       25,
		Images: []ProductImageSeed{
			{
				Filename:  "sam-s24u-gray-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1610945265064-0e34e5519bbf?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "sam-s24u-gray-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1580910051074-3eb694886505?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "APL-IP15P-128-NAT",
		CategoryName:       "Apple iPhone",
		Name:               "Apple iPhone 15 Pro 128GB Natural Titanium",
		Brand:              "Apple",
		Description:        "Smartphone Apple dengan chip A17 Pro bertenaga, tombol Action kustom, kamera Pro 48MP, dan koneksi USB-C.",
		Unit:               "pcs",
		PurchasePrice:      17500000,
		SellingPrice:       20999000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         187,
		AtributVarian:      map[string]any{"warna": "Natural Titanium", "storage": "128GB"},
		PrimaryBarcode:     "195949012345",
		InitialStock:       30,
		Images: []ProductImageSeed{
			{
				Filename:  "apl-ip15p-nat-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1695048133142-1a20484d2569?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "apl-ip15p-nat-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1592750475338-74b7b21085ab?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "ASU-ROG-G16-4070",
		CategoryName:       "Laptop Gaming",
		Name:               "Asus ROG Zephyrus G16 RTX 4070 16GB/1TB Eclipse Gray",
		Brand:              "Asus",
		Description:        "Laptop gaming ultra-tipis dengan prosesor Intel Core Ultra 9, GPU Nvidia GeForce RTX 4070, dan layar OLED 2.5K 240Hz.",
		Unit:               "unit",
		PurchasePrice:      28000000,
		SellingPrice:       32499000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         1850,
		AtributVarian:      map[string]any{"processor": "Intel Core Ultra 9", "gpu": "RTX 4070", "ram": "16GB", "ssd": "1TB"},
		PrimaryBarcode:     "4711081123456",
		InitialStock:       12,
		Images: []ProductImageSeed{
			{
				Filename:  "asu-rog-g16-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1603302576837-37561b2e2302?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "asu-rog-g16-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1588872657578-7efd1f1555ed?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "APL-MBA-M3-256-MID",
		CategoryName:       "Laptop Bisnis & Kerja",
		Name:               "Apple MacBook Air M3 13 Inch 8GB/256GB Midnight",
		Brand:              "Apple",
		Description:        "Laptop produktivitas harian ramping tanpa kipas, ditenagai chip Apple M3 dengan daya tahan baterai hingga 18 jam.",
		Unit:               "unit",
		PurchasePrice:      15500000,
		SellingPrice:       18499000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         1240,
		AtributVarian:      map[string]any{"chip": "Apple M3", "ram": "8GB", "ssd": "256GB", "warna": "Midnight"},
		PrimaryBarcode:     "195949678901",
		InitialStock:       20,
		Images: []ProductImageSeed{
			{
				Filename:  "apl-mba-m3-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1517336714731-489689fd1ca8?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "apl-mba-m3-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1611186871348-b1ce696e52c9?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "LG-TV-55UR8050",
		CategoryName:       "Smart TV 4K / 8K",
		Name:               "LG Smart TV 55 Inch 4K UHD ThinQ AI",
		Brand:              "LG",
		Description:        "Televisi pintar resolusi 4K dengan sistem operasi webOS ThinQ AI, HDR10 Pro, dan suara virtual 5.1 surround sound.",
		Unit:               "unit",
		PurchasePrice:      6200000,
		SellingPrice:       7499000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         14200,
		AtributVarian:      map[string]any{"ukuran_layar": "55 Inch", "resolusi": "4K UHD", "sistem_operasi": "webOS"},
		PrimaryBarcode:     "8806091234567",
		InitialStock:       15,
		Images: []ProductImageSeed{
			{
				Filename:  "lg-tv-55-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1593359677879-a4bb92f829d1?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "lg-tv-55-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1461151304267-38535e780c79?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "SNY-WH1000XM5-BLK",
		CategoryName:       "Headphone & TWS Earbuds",
		Name:               "Sony WH-1000XM5 Wireless Noise Cancelling Headphones Black",
		Brand:              "Sony",
		Description:        "Headphone nirkabel kelas atas dengan peredam bising aktif (ANC) terbaik di industri dan mikrofon jernih untuk panggilan.",
		Unit:               "unit",
		PurchasePrice:      4200000,
		SellingPrice:       5199000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         250,
		AtributVarian:      map[string]any{"warna": "Black", "fitur": "Active Noise Cancelling", "daya_baterai": "30 Jam"},
		PrimaryBarcode:     "4548736132566",
		InitialStock:       18,
		Images: []ProductImageSeed{
			{
				Filename:  "sny-wh1000xm5-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "sny-wh1000xm5-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1546435770-a3e426bf472b?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "SAM-REF-RT38-SIL",
		CategoryName:       "Kulkas & Pendingin",
		Name:               "Samsung Kulkas 2 Pintu Inverter 385L All-Around Cooling",
		Brand:              "Samsung",
		Description:        "Lemari es dua pintu hemat energi dengan teknologi Digital Inverter, deodorizing filter anti bau, dan kompresor bergaransi 10 tahun.",
		Unit:               "unit",
		PurchasePrice:      5800000,
		SellingPrice:       6999000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         60000,
		AtributVarian:      map[string]any{"kapasitas": "385 Liter", "tipe": "2 Pintu Inverter", "warna": "Elegant Inox"},
		PrimaryBarcode:     "8806098765432",
		InitialStock:       8,
		Images: []ProductImageSeed{
			{
				Filename:  "sam-ref-rt38-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1584992236310-6edddc08acff?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "sam-ref-rt38-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1571175443880-49e1d25b2bc5?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "LOG-MXM3S-GRP",
		CategoryName:       "Aksesoris & Periferal PC",
		Name:               "Logitech MX Master 3S Wireless Performance Mouse Graphite",
		Brand:              "Logitech",
		Description:        "Mouse ergonomis presisi tinggi dengan sensor optik 8000 DPI yang dapat bekerja di atas kaca dan tombol klik hening (Quiet Clicks).",
		Unit:               "unit",
		PurchasePrice:      1350000,
		SellingPrice:       1699000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         141,
		AtributVarian:      map[string]any{"warna": "Graphite", "konektivitas": "Bluetooth & Logi Bolt", "sensor": "8000 DPI"},
		PrimaryBarcode:     "097855173456",
		InitialStock:       35,
		Images: []ProductImageSeed{
			{
				Filename:  "log-mxm3s-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1615663245857-ac93bb7c39e7?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "log-mxm3s-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1527864550417-7fd91fc51a46?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "MI-AP4-COMPACT",
		CategoryName:       "Elektronik Rumah Tangga",
		Name:               "Xiaomi Smart Air Purifier 4 Compact HEPA Filter",
		Brand:              "Xiaomi",
		Description:        "Pembersih udara ruangan ringkas dengan filter 3-in-1 HEPA untuk menyaring debu PM2.5 dan alergen dengan kontrol aplikasi Mi Home.",
		Unit:               "unit",
		PurchasePrice:      950000,
		SellingPrice:       1299000,
		IsPPN:              true,
		FlagSerialTracking: true,
		WeightGram:         2200,
		AtributVarian:      map[string]any{"warna": "White", "cakupan_ruangan": "27 m2", "tipe_filter": "True HEPA"},
		PrimaryBarcode:     "6934177789012",
		InitialStock:       20,
		Images: []ProductImageSeed{
			{
				Filename:  "mi-ap4-compact-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1585771724684-38269d6639fd?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "mi-ap4-compact-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1527799820374-dcf8d9d4a388?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	// --- Produk Non-Serial Tracking (Aksesoris Harian) ---
	{
		SKU:                "ANK-PB-20K-30W",
		CategoryName:       "Power Bank & Baterai",
		Name:               "Anker PowerBank 20.000mAh 30W PowerCore Fast Charging",
		Brand:              "Anker",
		Description:        "Pengisi daya portabel kapasitas besar 20000mAh dengan port USB-C 30W Power Delivery untuk smartphone dan tablet.",
		Unit:               "pcs",
		PurchasePrice:      450000,
		SellingPrice:       699000,
		IsPPN:              true,
		FlagSerialTracking: false,
		WeightGram:         350,
		AtributVarian:      map[string]any{"kapasitas": "20000mAh", "daya_output": "30W PD", "warna": "Black"},
		PrimaryBarcode:     "194644023456",
		InitialStock:       80,
		Images: []ProductImageSeed{
			{
				Filename:  "ank-pb-20k-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1609091839311-d5365f9ff1c5?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "ank-pb-20k-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1594818379496-da1e345b0ded?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "ANK-CB-CC-100W",
		CategoryName:       "Kabel Data & Fast Charger",
		Name:               "Anker Kabel Data USB-C to USB-C 100W Braided 1.8M",
		Brand:              "Anker",
		Description:        "Kabel USB-C nilon rajut tebal tahan tekukan 25.000 kali, mendukung pengisian cepat daya tinggi hingga 100 Watt.",
		Unit:               "pcs",
		PurchasePrice:      120000,
		SellingPrice:       199000,
		IsPPN:              true,
		FlagSerialTracking: false,
		WeightGram:         65,
		AtributVarian:      map[string]any{"panjang": "1.8 Meter", "daya_maksimal": "100W", "material": "Nylon Braided"},
		PrimaryBarcode:     "194644087654",
		InitialStock:       150,
		Images: []ProductImageSeed{
			{
				Filename:  "ank-cb-cc-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1583863788434-e58a36330cf0?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "ank-cb-cc-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1622445262464-84b1456045b6?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
	{
		SKU:                "SPG-TG-EZFIT-CLR",
		CategoryName:       "Case & Screen Protector",
		Name:               "Spigen Tempered Glass Glas.tR EZ Fit Full Cover",
		Brand:              "Spigen",
		Description:        "Pelindung layar kaca temper 9H dengan baki aplikator EZ Fit inovatif untuk pemasangan presisi bebas gelembung udara.",
		Unit:               "set",
		PurchasePrice:      110000,
		SellingPrice:       189000,
		IsPPN:              true,
		FlagSerialTracking: false,
		WeightGram:         50,
		AtributVarian:      map[string]any{"isi_box": "2 Pcs Tempered Glass", "kekerasan": "9H Hardness", "warna": "Clear"},
		PrimaryBarcode:     "8809811234567",
		InitialStock:       100,
		Images: []ProductImageSeed{
			{
				Filename:  "spg-tg-ezfit-1.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1601784551446-20c9e07cdbdb?w=800&auto=format&fit=crop&q=80",
				IsPrimary: true,
				SortOrder: 1,
			},
			{
				Filename:  "spg-tg-ezfit-2.jpg",
				RemoteURL: "https://images.unsplash.com/photo-1584438784894-089d6a62b8fa?w=800&auto=format&fit=crop&q=80",
				IsPrimary: false,
				SortOrder: 2,
			},
		},
	},
}

// SeedProducts menyuntikkan katalog produk, barcode utama, dan stok awal ke database.
func SeedProducts(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	insertedCount := 0

	// 1. Ambil cache ID kategori dari database
	categoryMap := make(map[string]string)
	catRows, err := db.QueryContext(ctx, `SELECT id, name FROM inv_categories`)
	if err == nil {
		for catRows.Next() {
			var id, name string
			if errScan := catRows.Scan(&id, &name); errScan == nil {
				categoryMap[name] = id
			}
		}
		catRows.Close()
	}

	// 2. Ambil ID lokasi utama (Gudang Utama Jakarta) untuk inisialisasi stok
	var primaryLocationID string
	err = db.QueryRowContext(ctx, `SELECT id FROM inv_locations WHERE code = 'WH-JKT-01' LIMIT 1`).Scan(&primaryLocationID)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("gagal query lokasi WH-JKT-01: %w", err)
	}

	queryCheckSKU := `SELECT id FROM inv_products WHERE sku = ? LIMIT 1`
	queryInsertProduct := `
		INSERT INTO inv_products (
			id, sku, category_id, name, brand, description, unit,
			purchase_price, selling_price, status,
			is_ppn, flag_serial_tracking, weight_gram, atribut_varian,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?, ?, ?, ?, ?)`

	queryInsertBarcode := `
		INSERT INTO inv_barcodes (id, product_id, barcode, is_primary, created_at)
		VALUES (?, ?, ?, TRUE, ?)`

	queryInsertStock := `
		INSERT INTO inv_stocks (id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at)
		VALUES (?, ?, ?, ?, 0, 5, ?)
		ON DUPLICATE KEY UPDATE quantity = VALUES(quantity)`

	uploadDir := resolveUploadDir()
	httpClient := &http.Client{Timeout: 15 * time.Second}

	for _, p := range DefaultProducts {
		var existingID string
		err := db.QueryRowContext(ctx, queryCheckSKU, p.SKU).Scan(&existingID)
		if err == nil {
			// Produk sudah ada, pastikan fotonya disinkronkan jika belum ada
			_, _ = seedImagesForProduct(ctx, db, uploadDir, httpClient, existingID, p.Images, now)
			continue
		}
		if err != sql.ErrNoRows {
			return insertedCount, fmt.Errorf("gagal cek produk SKU '%s': %w", p.SKU, err)
		}

		productID := uid.New()
		categoryID := categoryMap[p.CategoryName]

		var catIDParam sql.NullString
		if categoryID != "" {
			catIDParam = sql.NullString{String: categoryID, Valid: true}
		}

		varianJSON, errMarshal := json.Marshal(p.AtributVarian)
		if errMarshal != nil {
			varianJSON = []byte("{}")
		}

		_, err = db.ExecContext(ctx, queryInsertProduct,
			productID, p.SKU, catIDParam, p.Name, p.Brand, p.Description, p.Unit,
			p.PurchasePrice, p.SellingPrice,
			p.IsPPN, p.FlagSerialTracking, p.WeightGram, string(varianJSON),
			now, now,
		)
		if err != nil {
			return insertedCount, fmt.Errorf("gagal insert produk SKU '%s': %w", p.SKU, err)
		}

		// Daftarkan Primary Barcode produk
		if p.PrimaryBarcode != "" {
			barcodeID := uid.New()
			_, _ = db.ExecContext(ctx, queryInsertBarcode, barcodeID, productID, p.PrimaryBarcode, now)
		}

		// Suntikkan Stok Awal di Gudang Utama jika lokasi terdaftar
		if primaryLocationID != "" && p.InitialStock > 0 {
			stockID := uid.New()
			_, _ = db.ExecContext(ctx, queryInsertStock, stockID, productID, primaryLocationID, p.InitialStock, now)
		}

		// Suntikkan Foto-foto Produk
		_, _ = seedImagesForProduct(ctx, db, uploadDir, httpClient, productID, p.Images, now)

		insertedCount++
	}

	return insertedCount, nil
}

// SeedProductImages menyuntikkan foto-foto produk ritel ke inv_product_images secara idempoten.
// Mengunduh foto ke folder uploads/products jika berkas lokal belum ada.
func SeedProductImages(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	totalInserted := 0

	uploadDir := resolveUploadDir()
	httpClient := &http.Client{Timeout: 15 * time.Second}

	queryGetProductBySKU := `SELECT id FROM inv_products WHERE sku = ? LIMIT 1`

	for _, p := range DefaultProducts {
		if len(p.Images) == 0 {
			continue
		}

		var productID string
		err := db.QueryRowContext(ctx, queryGetProductBySKU, p.SKU).Scan(&productID)
		if err != nil {
			continue // Produk belum ada di database
		}

		n, errSeed := seedImagesForProduct(ctx, db, uploadDir, httpClient, productID, p.Images, now)
		if errSeed != nil {
			return totalInserted, errSeed
		}
		totalInserted += n
	}

	return totalInserted, nil
}

func seedImagesForProduct(
	ctx context.Context,
	db *sql.DB,
	uploadDir string,
	client *http.Client,
	productID string,
	images []ProductImageSeed,
	now time.Time,
) (int, error) {
	if len(images) == 0 {
		return 0, nil
	}

	queryCheckImage := `SELECT id FROM inv_product_images WHERE product_id = ? AND (url = ? OR url = ?) LIMIT 1`
	queryCheckPrimary := `SELECT COUNT(*) FROM inv_product_images WHERE product_id = ? AND is_primary = TRUE`
	queryInsertImage := `
		INSERT INTO inv_product_images (id, product_id, url, is_primary, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	var hasPrimaryCount int
	_ = db.QueryRowContext(ctx, queryCheckPrimary, productID).Scan(&hasPrimaryCount)

	inserted := 0
	for _, img := range images {
		finalURL := "/uploads/products/" + img.Filename
		localPath := filepath.Join(uploadDir, img.Filename)

		// Cek apakah file fisik lokal sudah ada
		if _, errStat := os.Stat(localPath); os.IsNotExist(errStat) && img.RemoteURL != "" {
			req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, img.RemoteURL, nil)
			if errReq == nil {
				resp, errDo := client.Do(req)
				if errDo == nil && resp.StatusCode == http.StatusOK {
					outFile, errCreate := os.Create(localPath)
					if errCreate == nil {
						_, _ = io.Copy(outFile, resp.Body)
						_ = outFile.Close()
					}
					_ = resp.Body.Close()
				} else {
					// Fallback ke direct URL jika download lokal gagal
					finalURL = img.RemoteURL
				}
			} else {
				finalURL = img.RemoteURL
			}
		}

		// Cek apakah URL gambar ini sudah terdaftar di database
		var existingID string
		errCheck := db.QueryRowContext(ctx, queryCheckImage, productID, finalURL, img.RemoteURL).Scan(&existingID)
		if errCheck == nil {
			continue // Sudah ada
		}

		isPrimary := img.IsPrimary
		if hasPrimaryCount == 0 && img.SortOrder == 1 {
			isPrimary = true
		}

		imageID := uid.New()
		_, err := db.ExecContext(ctx, queryInsertImage,
			imageID, productID, finalURL, isPrimary, img.SortOrder, now,
		)
		if err == nil {
			if isPrimary {
				hasPrimaryCount++
			}
			inserted++
		}
	}

	return inserted, nil
}

func resolveUploadDir() string {
	candidates := []string{
		"./uploads/products",
		"./backend/uploads/products",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c
		}
	}
	_ = os.MkdirAll("./backend/uploads/products", 0755)
	_ = os.MkdirAll("./uploads/products", 0755)
	if fi, err := os.Stat("./backend/uploads/products"); err == nil && fi.IsDir() {
		return "./backend/uploads/products"
	}
	return "./uploads/products"
}

// ============================================================================
// 4. DATA SEEDER: TEMPLATE MASTER KEBIJAKAN GARANSI
// ============================================================================

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
			continue // Template sudah ada, skip untuk menjaga idempoten
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

// SeedStocks menyuntikkan data saldo persediaan awal untuk seluruh produk aktif ke setiap cabang secara idempoten.
func SeedStocks(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	insertedCount := 0

	// 1. Ambil seluruh lokasi aktif
	locRows, err := db.QueryContext(ctx, `SELECT id, code FROM inv_locations WHERE is_active = TRUE`)
	if err != nil {
		return 0, fmt.Errorf("gagal query lokasi untuk seed stok: %w", err)
	}
	defer locRows.Close()

	type locInfo struct {
		id   string
		code string
	}
	var locations []locInfo
	for locRows.Next() {
		var l locInfo
		if err := locRows.Scan(&l.id, &l.code); err != nil {
			return 0, err
		}
		locations = append(locations, l)
	}

	// 2. Ambil seluruh produk aktif
	prodRows, err := db.QueryContext(ctx, `SELECT id, sku FROM inv_products WHERE status = 'active'`)
	if err != nil {
		return 0, fmt.Errorf("gagal query produk untuk seed stok: %w", err)
	}
	defer prodRows.Close()

	type prodInfo struct {
		id  string
		sku string
	}
	var products []prodInfo
	for prodRows.Next() {
		var p prodInfo
		if err := prodRows.Scan(&p.id, &p.sku); err != nil {
			return 0, err
		}
		products = append(products, p)
	}

	queryCheck := `SELECT id FROM inv_stocks WHERE product_id = ? AND location_id = ? LIMIT 1`
	queryInsert := `
		INSERT INTO inv_stocks (
			id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`

	for _, loc := range locations {
		for i, prod := range products {
			var existingID string
			err := db.QueryRowContext(ctx, queryCheck, prod.id, loc.id).Scan(&existingID)
			if err == nil {
				continue // Sudah ada stok untuk kombinasi ini, skip
			}
			if err != sql.ErrNoRows {
				return insertedCount, fmt.Errorf("gagal cek stok produk %s di lokasi %s: %w", prod.sku, loc.code, err)
			}

			// Variasi kuantitas yang realistis untuk demo retail
			var qty, reserved, minStock int
			switch loc.code {
			case "WH-JKT-01", "WH-PST": // Gudang Distribusi Utama
				qty = 40 + (i%5)*15 // 40 - 100 unit
				reserved = i % 4    // 0 - 3 booking
				minStock = 10
			case "STR-SBY-01", "TK-SBY": // Toko Flagship Surabaya
				if i == 0 {
					qty = 0 // Habis
					minStock = 5
				} else if i == 1 {
					qty = 2 // Menipis (Alert!)
					minStock = 5
				} else {
					qty = 15 + (i%4)*6 // 15 - 33 unit
					reserved = i % 3
					minStock = 5
				}
			case "STR-BDG-01": // Toko Cabang Bandung
				if i == 2 {
					qty = 1 // Menipis (Alert!)
					minStock = 4
				} else {
					qty = 8 + (i%3)*5 // 8 - 18 unit
					minStock = 4
				}
			case "WH-ECOM-01": // Gudang E-Commerce
				qty = 25 + (i%4)*10 // 25 - 55 unit
				reserved = (i % 2) * 2
				minStock = 8
			default:
				qty = 10 + (i%3)*5
				minStock = 5
			}

			stockID := uid.New()
			_, err = db.ExecContext(ctx, queryInsert,
				stockID, prod.id, loc.id, qty, reserved, minStock, now,
			)
			if err != nil {
				return insertedCount, fmt.Errorf("gagal insert stok: %w", err)
			}
			insertedCount++
		}
	}

	return insertedCount, nil
}

// SeedSerialUnits memasukkan data contoh unit fisik berserial (S/N & IMEI) untuk produk ber-flag_serial_tracking.
func SeedSerialUnits(ctx context.Context, db *sql.DB) (int, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, sku, brand FROM inv_products WHERE flag_serial_tracking = TRUE`)
	if err != nil {
		return 0, fmt.Errorf("gagal query produk serial: %w", err)
	}
	defer rows.Close()

	type prodInfo struct {
		id, sku, brand string
	}
	var prods []prodInfo
	for rows.Next() {
		var p prodInfo
		if err := rows.Scan(&p.id, &p.sku, &p.brand); err != nil {
			return 0, err
		}
		prods = append(prods, p)
	}
	if len(prods) == 0 {
		return 0, nil
	}

	locRows, err := db.QueryContext(ctx, `SELECT id, code FROM inv_locations WHERE is_active = TRUE`)
	if err != nil {
		return 0, fmt.Errorf("gagal query lokasi: %w", err)
	}
	defer locRows.Close()

	type locInfo struct {
		id, code string
	}
	var locs []locInfo
	for locRows.Next() {
		var l locInfo
		if err := locRows.Scan(&l.id, &l.code); err != nil {
			return 0, err
		}
		locs = append(locs, l)
	}
	if len(locs) == 0 {
		return 0, nil
	}

	now := time.Now().UTC()
	queryInsert := `
		INSERT INTO inv_serial_units (id, product_id, location_id, serial_number, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`

	insertedCount := 0
	for _, p := range prods {
		for locIdx, l := range locs {
			statuses := []string{"tersedia", "tersedia", "terjual", "retur"}
			for i, st := range statuses {
				sn := fmt.Sprintf("SN-%s-%02d%02d", p.sku, locIdx+1, i+1)
				unitID := uid.New()
				res, err := db.ExecContext(ctx, queryInsert, unitID, p.id, l.id, sn, st, now, now)
				if err != nil {
					return insertedCount, fmt.Errorf("gagal insert serial '%s': %w", sn, err)
				}
				if n, _ := res.RowsAffected(); n > 0 {
					insertedCount++
				}
			}
		}
	}

	return insertedCount, nil
}

// PriceOverrideSeed mendefinisikan template promo harga khusus per cabang.
type PriceOverrideSeed struct {
	ProductSKU       string
	LocationCode     string
	PromotionalPrice int64
	MaxQuantity      *int
	ClaimedQuantity  int
	DaysStartOffset  int  // Hari relatif terhadap hari ini (negatif = sudah mulai)
	DaysDuration     int  // Durasi promo dalam hari
	Reason           string
	IsActive         bool
}

func intPtr(v int) *int { return &v }

// DefaultPriceOverrides memuat contoh promo realistis untuk demo Backoffice.
var DefaultPriceOverrides = []PriceOverrideSeed{
	{
		ProductSKU:       "SAM-S24U-256-GRY",
		LocationCode:     "STR-SBY-01",
		PromotionalPrice: 19999000,
		MaxQuantity:      intPtr(20),
		ClaimedQuantity:  7,
		DaysStartOffset:  -7,
		DaysDuration:     30,
		Reason:           "Promo Grand Opening Flagship Surabaya",
		IsActive:         true,
	},
	{
		ProductSKU:       "APL-IP15P-128-NAT",
		LocationCode:     "STR-BDG-01",
		PromotionalPrice: 18999000,
		MaxQuantity:      intPtr(15),
		ClaimedQuantity:  15, // Kuota telah habis!
		DaysStartOffset:  -3,
		DaysDuration:     7,
		Reason:           "Flash Sale Weekend Dago Bandung",
		IsActive:         true,
	},
	{
		ProductSKU:       "TV-LG-43-UQ7500",
		LocationCode:     "WH-JKT-01",
		PromotionalPrice: 3999000,
		MaxQuantity:      nil, // Unlimited
		ClaimedQuantity:  12,
		DaysStartOffset:  -10,
		DaysDuration:     30,
		Reason:           "Diskon Spesial Smart TV Jakarta",
		IsActive:         true,
	},
	{
		ProductSKU:       "KUL-SHARP-2D-205",
		LocationCode:     "WH-ECOM-01",
		PromotionalPrice: 2899000,
		MaxQuantity:      intPtr(30),
		ClaimedQuantity:  9,
		DaysStartOffset:  -5,
		DaysDuration:     20,
		Reason:           "Super Deal Online E-Commerce Kulkas",
		IsActive:         true,
	},
	{
		ProductSKU:       "ASU-ROG-G16-4070",
		LocationCode:     "STR-SBY-01",
		PromotionalPrice: 26999000,
		MaxQuantity:      intPtr(5),
		ClaimedQuantity:  5,
		DaysStartOffset:  -30,
		DaysDuration:     20, // Expired 10 hari lalu
		Reason:           "Early Bird ROG Gaming Surabaya",
		IsActive:         false,
	},
}

// SeedPriceOverrides memasukkan contoh data promo harga cabang ke tabel inv_price_overrides.
func SeedPriceOverrides(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	insertedCount := 0

	queryCheck := `SELECT id FROM inv_price_overrides WHERE product_id = ? AND location_id = ? AND reason = ? LIMIT 1`
	queryInsert := `
		INSERT INTO inv_price_overrides (
			id, product_id, location_id, promotional_price, max_quantity, claimed_quantity,
			start_date, end_date, reason, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for _, p := range DefaultPriceOverrides {
		var productID string
		err := db.QueryRowContext(ctx, `SELECT id FROM inv_products WHERE sku = ? LIMIT 1`, p.ProductSKU).Scan(&productID)
		if err != nil {
			continue // Produk belum ada, skip
		}

		var locationID string
		err = db.QueryRowContext(ctx, `SELECT id FROM inv_locations WHERE code = ? LIMIT 1`, p.LocationCode).Scan(&locationID)
		if err != nil {
			continue // Lokasi belum ada, skip
		}

		var existingID string
		err = db.QueryRowContext(ctx, queryCheck, productID, locationID, p.Reason).Scan(&existingID)
		if err == nil {
			continue // Sudah ada, skip
		}

		id := uid.New()
		startDate := now.AddDate(0, 0, p.DaysStartOffset)
		endDate := startDate.AddDate(0, 0, p.DaysDuration)

		_, err = db.ExecContext(ctx, queryInsert,
			id, productID, locationID, p.PromotionalPrice, p.MaxQuantity, p.ClaimedQuantity,
			startDate, endDate, p.Reason, p.IsActive, now, now,
		)
		if err != nil {
			return insertedCount, fmt.Errorf("gagal insert promo '%s': %w", p.Reason, err)
		}
		insertedCount++
	}

	return insertedCount, nil
}


