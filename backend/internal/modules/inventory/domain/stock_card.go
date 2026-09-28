package domain

import "time"

// StockCardEntry merepresentasikan satu baris mutasi dalam riwayat buku besar (kartu stok).
type StockCardEntry struct {
	Date            time.Time `json:"date"`
	MovementType    string    `json:"movement_type"`    // 'in', 'out', 'opname', 'transfer_in', 'transfer_out'
	DocumentNumber  string    `json:"document_number"`  // No. IN / OUT / TRF / OPNAME
	ReferenceNumber string    `json:"reference_number"` // Surat jalan / memo
	CategoryReason  string    `json:"category_reason"`  // Alasan transaksi
	InQuantity      int       `json:"in_quantity"`      // Kuantitas bertambah (+)
	OutQuantity     int       `json:"out_quantity"`     // Kuantitas berkurang (-)
	Balance         int       `json:"balance"`          // Saldo kumulatif setelah mutasi
	ExecutedByName  string    `json:"executed_by_name"` // Nama operator / staf penanggung jawab
	Notes           string    `json:"notes"`            // Catatan
}

// StockCardReport merepresentasikan laporan kartu stok lengkap untuk satu produk di satu cabang.
type StockCardReport struct {
	ProductID      string            `json:"product_id"`
	ProductName    string            `json:"product_name"`
	ProductSKU     string            `json:"product_sku"`
	LocationID     string            `json:"location_id"`
	LocationName   string            `json:"location_name"`
	OpeningBalance int               `json:"opening_balance"`
	TotalIn        int               `json:"total_in"`
	TotalOut       int               `json:"total_out"`
	ClosingBalance int               `json:"closing_balance"`
	Entries        []*StockCardEntry `json:"entries"`
}

// StockValuationItem merepresentasikan ringkasan kuantitas dan nilai aset per produk di cabang.
type StockValuationItem struct {
	ProductID      string  `json:"product_id"`
	ProductSKU     string  `json:"product_sku"`
	ProductName    string  `json:"product_name"`
	CategoryName   string  `json:"category_name"`
	LocationID     string  `json:"location_id"`
	LocationName   string  `json:"location_name"`
	Quantity       int     `json:"quantity"`
	MinStock       int     `json:"min_stock"`
	BasePrice      float64 `json:"base_price"`
	TotalValuation float64 `json:"total_valuation"` // Quantity * BasePrice
	Status         string  `json:"status"`          // 'aman', 'menipis', 'habis'
}
