# 🗺️ Roadmap Pengembangan & Rencana Fase Selanjutnya (Next Phase)

Dokumen ini mencatat ringkasan status saat ini dan daftar modul pada proyek **ERP Retail Modular (Go + SvelteKit)** ini.

---

## 🟢 Status Terakhir: Modul Inventory SELESAI 100% (Feature Complete)! 🎉

Seluruh 6 fase bisnis retail pada **Modul Inventory** telah selesai dibangun dari fondasi hingga HTTP endpoints, dengan standard **Domain-Driven Design (DDD)**, **Zero-Warning Policy**, **UUIDv7**, dan **isolasi Bounded Context**:

1. **Master Kategori (`Category`) — 100%**
   - Hierarki 2 level (Parent - Child).
   - Pencegahan referensi melingkar (_circular reference_).
   - CRUD lengkap + API docs.
2. **Master Produk (`Product`) — 100%**
   - Generic master (SKU unik internal, harga beli/jual global, atribut varian JSON).
   - Patch status produk (`active`, `inactive`, `discontinued`).
   - CRUD lengkap + API docs.
3. **Master Lokasi (`Location`) — 100%**
   - Dukungan cabang fisik (`physical`) dan gudang virtual online (`online`).
   - Kode cabang unik, aktivasi/deaktivasi.
   - CRUD lengkap + API docs.
4. **Operasional Stok per Cabang (`StockItem`) — 100%**
   - Domain entity & invariant ketat (`Quantity >= 0`, `ReservedQuantity <= Quantity`, `AvailableQuantity()`).
   - Repository dengan row-level locking MySQL (`SELECT ... FOR UPDATE` via `AtomicMutate`).
   - Opname stok, alert minimum stock, reservasi stok.
5. **Multi-Barcode Pabrik per Produk (`ProductBarcode`) — 100%**
   - Migrasi database `inv_barcodes` dengan `ON DELETE CASCADE`.
   - Primary barcode tunggal per produk, pencarian kilat kasir POS via `FindProductByBarcode`.
6. **Pelacakan Unit Fisik Serial Number / IMEI (`SerialUnit`) — 100%**
   - Migrasi database `inv_serial_units`.
   - Value object `SerialStatus` (`tersedia`, `terjual`, `retur`).
   - Validasi ketat `flag_serial_tracking = true`, batch insert transaksional.
7. **Promo / Harga Khusus per Cabang (`PriceOverride`) — 100%**
   - Migrasi database `inv_price_overrides`.
   - Anti-overlap interval promo, kuota flash sale dengan atomic decrement.
   - Algoritma kasir POS `GetEffectivePrice`.
8. **Mutasi Stok Antar Cabang (`StockTransfer`) — 100%**
   - Migrasi database `inv_stock_transfers`, `inv_stock_transfer_items`, `inv_stock_transfer_item_serials`.
   - Aggregate Root multi-item, nomor urut surat jalan `TRF-YYYYMM-XXXX`.
   - Siklus 4 status: `pending_approval -> approved -> in_transit -> received` (atau `rejected`).
   - Pemindahan otomatis lokasi unit berserial saat barang tiba di cabang tujuan.
9. **Kebijakan Garansi Toko & Pabrik (`WarrantyPolicy` & `ProductWarranty`) — 100%**
   - Migrasi database `006_create_inv_warranties_table.sql` (`inv_warranty_policies`, `inv_product_warranties`).
   - Master template garansi (durasi bulan & hari, cakupan, petunjuk klaim, kalkulasi tanggal kedaluwarsa).
   - Invariant Kunci: **Maksimal satu garansi aktif per tipe (`toko` / `pabrik`) per produk**. Penugasan garansi baru bertipe sama otomatis menonaktifkan garansi aktif sebelumnya secara atomik.
   - 5 REST endpoints di `/api/v1/inventory/warranties/*`.
   - Facade sinkron: `InventoryService.GetProductWarranties(productID)`.

---

## 📊 Matriks Penyelesaian Modul Inventory

| Fitur / Sub-Modul      | Domain & Unit Tests | Migrasi Database | Application Use Cases | REST Handlers & Routes | Facade Sinkron | OpenAPI & `api.http` |   Status    |
| :--------------------- | :-----------------: | :--------------: | :-------------------: | :--------------------: | :------------: | :------------------: | :---------: |
| **Kategori & Brand**   |         ✅          |     ✅ (001)     |          ✅           |           ✅           |       —        |          ✅          | **SELESAI** |
| **Master Produk**      |         ✅          |     ✅ (001)     |          ✅           |           ✅           |       ✅       |          ✅          | **SELESAI** |
| **Master Lokasi**      |         ✅          |     ✅ (001)     |          ✅           |           ✅           |       ✅       |          ✅          | **SELESAI** |
| **Stok & Opname**      |         ✅          |     ✅ (001)     |          ✅           |           ✅           |       ✅       |          ✅          | **SELESAI** |
| **Multi-Barcode**      |         ✅          |     ✅ (002)     |          ✅           |           ✅           |       ✅       |          ✅          | **SELESAI** |
| **Serial Unit / IMEI** |         ✅          |     ✅ (003)     |          ✅           |           ✅           |       ✅       |          ✅          | **SELESAI** |
| **Price Override**     |         ✅          |     ✅ (004)     |          ✅           |           ✅           |       ✅       |          ✅          | **SELESAI** |
| **Stock Transfer**     |         ✅          |     ✅ (005)     |          ✅           |           ✅           |       —        |          ✅          | **SELESAI** |
| **Garansi (Warranty)** |         ✅          |     ✅ (006)     |          ✅           |           ✅           |       ✅       |          ✅          | **SELESAI** |

---

## 🚀 Rencana Fase Selanjutnya: Memilih Modul Berikutnya

Dengan tuntasnya seluruh fondasi dan fitur operasional Modul Inventory, kita siap melangkah ke modul bisnis berikutnya sesuai arsitektur ERP:

```text
               ┌──────────────────────────────┐
               │    MODUL INVENTORY (✅ 100%) │
               └──────────────┬───────────────┘
                              │
             ┌────────────────┴────────────────┐
             ▼                                 ▼
┌──────────────────────────────┐ ┌──────────────────────────────┐
│  PILIHAN A: MODUL PURCHASING │ │    PILIHAN B: MODUL SALES    │
│  - Master Supplier & Kontak  │ │  - Kasir POS (Point of Sale) │
│  - Purchase Order (PO) Multi │ │  - Sales Order & Keranjang   │
│  - Approval PO (Owner)       │ │  - Validasi Serial / IMEI    │
│  - Goods Receipt (Gudang)    │ │  - Pembayaran Multi-Metode   │
│  - Penerimaan bertambah stok │ │  - Cetak Nota & Garansi      │
└──────────────────────────────┘ └──────────────────────────────┘
```

### Opsi A: Modul Purchasing (Pengadaan & Pembelian dari Supplier)

- Alur: Menyiapkan data pemasok (supplier), membuat Purchase Order (PO), approval PO oleh Owner, dan penerimaan barang (Goods Receipt) yang otomatis menambah stok gudang di Modul Inventory.

### Opsi B: Modul Sales (Kasir POS & Penjualan Retail)

- Alur: Transaksi penjualan kasir di cabang, pencarian produk via barcode scanner, pemotongan stok otomatis, pemotongan kuota promo `PriceOverride`, penandaan unit `SerialUnit` menjadi status `terjual`, dan pencetakan nota/kartu garansi `ProductWarranty`.
