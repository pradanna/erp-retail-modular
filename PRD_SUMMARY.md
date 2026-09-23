# PRD Summary — ERP Retail Modular
### Studi Kasus Awal: Toko Elektronik

> **Sumber:** [`prd.txt`](file:///c:/PROJECT/WEBSITE/erp-retail-modular/prd.txt)  
> **Versi PRD:** 1.1 (Draft — revisi setelah breakdown domain Inventory & Purchasing)  
> **Status:** Untuk validasi sebelum masuk implementasi detail per modul  

---

## 1. Ringkasan Eksekutif

Sistem ERP retail **modular**, dijual per instalasi (**single-tenant** per toko, bukan SaaS multi-tenant). Setiap toko mendapat instalasi sendiri dengan modul-modul yang bisa **diaktifkan (unlock) sesuai lisensi** yang dibeli.

Meskipun studi kasus pertama adalah **toko elektronik** (garansi ganda toko & pabrik, serial number/IMEI, transaksi tinggi, komisi salesman), sistem dirancang **generic untuk kebutuhan retail secara umum**.

---

## 2. Tech Stack & Standar Teknis

| Layer | Teknologi | Catatan |
|---|---|---|
| **Backend** | **Go** | Modular monolith — 1 binary/deploy per instalasi |
| **Standard ID** | **UUIDv7** | Wajib di seluruh aggregate (urut alami, ramah index database) |
| **Frontend Backoffice** | **SvelteKit** (SPA, CSR) | Operasional kasir/admin (`export const ssr = false`) |
| **Frontend Storefront** | **SvelteKit** (SSR/prerender) | E-commerce publik ramah SEO |
| **Shared Packages** | `ui`, `api-client`, `types` | Monorepo package bersama lintas backoffice & storefront |
| **Database** | SQL (PostgreSQL / MySQL) | Tabel dipisah per modul (`inv_*`, `purch_*`, dll.), **tanpa FK lintas modul** |

### Pola Arsitektur Backend
- **Modular Monolith:** Bounded context per modul (`domain`, `application`, `infrastructure`, `interfaces`, `module.go`).
- **Komunikasi Antar Modul:**
  - **Sinkron:** Facade interface publik (`module.go`).
  - **Asinkron:** Event bus in-process (`shared/event`).
- **Lisensi Startup:** Modul yang tidak diaktifkan **tidak di-mount** ke router maupun event bus (bukan sekadar disembunyikan di UI).
- **Cross-cutting Shared:** `shared/audit`, `shared/settings` (toggle step-up auth), `shared/auth`, `shared/license`, `shared/event`.

---

## 3. Cakupan Modul MVP

| # | Modul | Fungsi Utama | Wajib di MVP | Sifat Lisensi |
|---|---|---|---|---|
| 1 | **Inventory** | Master produk generic, stok per cabang, transfer stok, serial/IMEI, garansi | ✅ Ya | Core |
| 2 | **Purchasing** | Master supplier, Purchase Order (PO), goods receipt (100% full receipt) | ✅ Ya | Core / Unlockable |
| 3 | **Sales** | Transaksi POS kasir, sales order, retur penjualan | ✅ Ya | Core |
| 4 | **Finance** | Jurnal otomatis (`OrderPaid` & `GoodsReceived`), piutang & hutang supplier | ✅ Ya | Unlockable |
| 5 | **Ecommerce** | Storefront publik, order online (gudang virtual online) | ✅ Ya | Unlockable |
| 6 | **Commission** | Skema komisi terstruktur, akrual bonus salesman dari `OrderPaid` | ✅ Ya | Unlockable (Bisa off total) |

> **Catatan Pembayaran Storefront:** Payment gateway online (Midtrans/Xendit) **di luar scope MVP** — pembayaran awal storefront menggunakan metode manual/offline.

---

## 4. Peran, Approval & Keamanan Aksi Sensitif

### 4.1 Role Matrix
- **Admin:** Operasional harian cabang; request transfer stok & PO; **tidak bisa approve**.
- **Admin Gudang:** Role Admin yang di-assign ke Location gudang tertentu; melakukan **konfirmasi penerimaan barang (goods receipt)** di gudang tersebut.
- **Superadmin:** Approve transfer stok & Purchase Order; visibilitas penuh seluruh aktivitas toko.
- **Owner:** Setara Superadmin; akses penuh lintas cabang dan manajemen lisensi/settings.

### 4.2 Audit Log (Selalu Aktif)
- Fondasi akuntabilitas di `shared/audit` (tidak bisa dimatikan).
- Mencatat: siapa approve transfer stok, approve PO, konfirmasi goods receipt, ubah promo harga, proses pencairan komisi.
- Mengisi data dengan subscribe ke event bus secara pasif.

### 4.3 Step-up Authentication (Toggle di Settings)
- Konfirmasi ulang password sebelum aksi kritikal (approve transfer, approve PO, goods receipt).
- Divalidasi ganda: **FE (modal dialog konfirmasi password)** dan **BE (re-verifikasi wajib di endpoint)**.
- Dapat diaktifkan/dinonaktifkan kapan saja oleh Superadmin/Owner via `shared/settings` tanpa redeploy.

---

## 5. Detail Fungsional per Modul

### 5.1 Inventory
- **Master Produk (`Product`):**
  - Kolom: `id` (UUIDv7), `sku` (kode internal toko), `category_id` (kategori level 2), `name`, `brand`, `description`, `unit`, `purchase_price`, `selling_price` (harga jual global), `status`, `is_ppn` (boolean, default true), `flag_serial_tracking` (boolean), `weight_kg`, `dimensions`, `atribut_varian` (object generic JSON).
  - Entitas terpisah (One-to-Many):
    - `ProductBarcode`: multi-barcode pabrik per produk.
    - `ProductImage`: foto produk.
    - `ProductWarranty`: garansi produk (join master `WarrantyPolicy`, maks 1 aktif per tipe toko/pabrik).
- **Kategori Hierarkis:** 2 level saja (Level 1 → Level 2). Produk selalu mengarah ke Level 2.
- **Stok & Lokasi:**
  - `Location`: tipe `physical` atau `online` (gudang storefront).
  - `StockItem`: stok tercatat per `(Product, Location)` lengkap dengan `minimum_stock` (reorder point cabang).
- **Serial Tracking:** `SerialUnit` (tersedia / terjual / retur).
- **Price Override (Promo):** Strict maks **1 promo aktif** per kombinasi `(product_id, location_id)`.
- **Transfer Stok:** Alur `pending_approval -> approved -> in_transit -> received` (approval Superadmin/Owner).

### 5.2 Purchasing (Modul Baru MVP)
- **Master `Supplier`:** Menyimpan informasi vendor dan histori harga beli (Inventory tidak menyimpan data supplier).
- **`PurchaseOrder` (PO):**
  - Alur status: `draft -> pending_approval -> approved -> ordered -> received` (atau `rejected`).
  - Approval PO wajib oleh Superadmin/Owner.
- **Penerimaan Barang (`GoodsReceipt`):**
  - MVP hanya mendukung **full receipt (100%)** — tidak ada cicilan penerimaan.
  - Dieksekusi oleh **Admin Gudang** di Location tujuan.
  - Menerbitkan event `GoodsReceived`.

### 5.3 Sales
- POS kasir di cabang (memilih Location).
- Sales order untuk skema kredit/tempo.
- Relasi salesman opsional per order.
- Menerbitkan event `OrderPaid` setelah transaksi sukses.

### 5.4 Finance
- Jurnal otomatis dari event `OrderPaid` (pencatatan piutang kasir/kredit).
- Jurnal otomatis dari event `GoodsReceived` (pencatatan hutang/payable ke supplier).
- Laporan keuangan per cabang dan konsolidasi.
- Kas masuk/keluar operasional non-transaksi.

### 5.5 Ecommerce
- Storefront publik dengan sumber stok dari `Location` online.
- Order online masuk sebagai `Sales Order` ke gudang online.
- Checkout MVP menggunakan pembayaran manual/offline.

### 5.6 Commission
- Salesman terhubung langsung ke entity `User`.
- Skema komisi terstruktur di DB (per kategori produk atau target).
- Akrual otomatis dari event `OrderPaid` (jika `SalesmanID != nil`).
- Dapat dinonaktifkan total tanpa mengganggu operasional Sales.

---

## 6. Alur Event Antar Modul

```text
1. Penjualan Selesai:
   OrderPaid { OrderID, LocationID, SalesmanID?, Items, Total }
       ├── Inventory   : Kurangi stok / lepaskan reservasi
       ├── Finance     : Buat jurnal akuntansi + catat piutang jika tempo
       └── Commission  : Hitung akrual komisi (jika SalesmanID ada & modul aktif)

2. Penerimaan Barang dari Supplier:
   GoodsReceived { PurchaseOrderID, LocationID, Items, ReceivedBy }
       ├── Inventory   : Tambah quantity stok (StockItem)
       └── Finance     : Catat hutang dagang (payable) ke supplier

3. Aksi Kritikal Sensitif:
   Approve Transfer / Approve PO / Goods Receipt
       ├── AuditLog    : Catat log audit permanen (shared/audit)
       └── Step-Up Auth: Minta verifikasi password jika require_reauth aktif
```

---

## 7. Urutan Build yang Direkomendasikan

```text
1. Inventory  ──→  2. Purchasing  ──→  3. Sales  ──→  4. Finance & Commission  ──→  5. Ecommerce
```
*Alasan:* Purchasing membutuhkan master produk dari Inventory; Sales & Purchasing menerbitkan event utama untuk Finance; Commission bergantung pada Sales.

---

## 8. Out of Scope (MVP)
- Payment gateway online pihak ketiga (Midtrans/Xendit).
- Partial / cicilan penerimaan barang (goods receipt partial).
- Skema kredit/cicilan kompleks di modul Purchasing.
- HRIS / payroll penuh.
- Multi-currency.
- Integrasi marketplace (Tokopedia/Shopee).
- General ledger & COA custom kompleks.
- Dynamic rule-engine untuk komisi.

---

## 9. Open Items
1. **Keseragaman Tarif PPN:** Konfirmasi apakah semua produk retail flat 11% tax-inclusive tanpa pengecualian.
2. **Cakupan Step-Up Auth Tambahan:** Apakah step-up auth juga diwajibkan untuk ubah `PriceOverride` dan `CommissionPayout`.
3. **Detail Struktur `CommissionRule`:** Pemodelan skema berbasis kategori produk vs tier omzet bulanan.
