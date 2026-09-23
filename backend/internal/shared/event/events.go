package event

// Nama-nama event resmi. Menggunakan konstanta string mencegah typo
// dan memudahkan refactoring (cari semua usage dengan mudah).
const (
	EventOrderPaid      = "order.paid"
	EventGoodsReceived  = "purchasing.goods_received"
)

// OrderPaidPayload adalah payload event saat transaksi penjualan berhasil dibayar.
// Diterbitkan oleh modul Sales, dikonsumsi oleh Inventory, Finance, dan Commission.
//
// ATURAN PENTING: payload hanya berisi ID referensi, bukan objek domain penuh.
// Ini menjaga agar subscriber tidak tightly-coupled ke struktur data publisher.
type OrderPaidPayload struct {
	OrderID    string  // ID transaksi penjualan (UUIDv7)
	LocationID string  // ID cabang/gudang tempat transaksi
	SalesmanID *string // nullable — Commission hanya proses jika ini terisi
	Total      int64   // Total dalam satuan sen/paling kecil (hindari float untuk uang)
	Items      []OrderPaidItem
}

// OrderPaidItem adalah detail per baris item dalam OrderPaidPayload.
type OrderPaidItem struct {
	ProductID string
	Quantity  int
	UnitPrice int64
}

// GoodsReceivedPayload adalah payload event saat penerimaan barang dari supplier selesai.
// Diterbitkan oleh modul Purchasing, dikonsumsi oleh Inventory dan Finance.
type GoodsReceivedPayload struct {
	PurchaseOrderID string // ID Purchase Order yang barangnya diterima
	LocationID      string // ID gudang tempat barang diterima
	ReceivedBy      string // UserID Admin Gudang yang melakukan konfirmasi
	Items           []GoodsReceivedItem
}

// GoodsReceivedItem adalah detail per baris item dalam GoodsReceivedPayload.
type GoodsReceivedItem struct {
	ProductID       string
	Quantity        int
	PurchasePriceID string // referensi ke harga beli saat PO dibuat
}
