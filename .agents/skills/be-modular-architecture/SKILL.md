---
name: be-modular-architecture
description: Panduan wajib untuk membuat dan memodifikasi kode backend Go pada proyek ERP retail modular. Gunakan skill ini setiap kali membuat modul Go baru, use case/handler baru, entity/aggregate baru, event publisher/subscriber baru, query repository, atau mengubah wiring lisensi di main.go.
---

# Panduan Arsitektur Backend Go (Modular Monolith)

Gunakan panduan ini saat membuat atau mengubah kode Go di dalam proyek backend ERP Retail Modular.

---

## 1. Struktur Folder Backend

```text
/cmd
  /server                  -> main.go: baca lisensi, wiring modul aktif, start HTTP server
/internal
  /modules
    /inventory             -> Master produk, stok per cabang, transfer, garansi, serial number
    /purchasing            -> Master supplier, Purchase Order (PO), goods receipt (100% full)
    /sales                 -> Transaksi POS kasir, sales order, retur
    /finance               -> Jurnal otomatis (OrderPaid & GoodsReceived), piutang & hutang
    /commission            -> Perhitungan akrual bonus & payout salesman
    /ecommerce             -> Integrasi order storefront online
    /<module_name>         -> Struktur 4 layer per modul:
      /domain              -> Entity, value object, repository interface, domain service
      /application         -> Use case (command/query handlers), orkestrasi transaksi
      /infrastructure      -> Implementasi repository (Postgres/MySQL), event publisher
      /interfaces          -> HTTP handler (REST/Fiber/Gin), DTO request/response, route registration
      module.go            -> Facade: interface publik + fungsi Register()
  /shared                  -> Cross-cutting context yang selalu aktif:
    /audit                 -> Audit log recorder (subscribe ke event bus aksi kritikal)
    /settings              -> Konfigurasi operasional (mis. toggle step-up re-authentication)
    /auth                  -> Autentikasi JWT & middleware role RBAC
    /license               -> Engine pembaca lisensi & registry modul aktif
    /event                 -> Kontrak definisi event & in-process event bus
  /platform                -> DB connection, config, logger, migration runner
/pkg                       -> Utility generic lintas project (UUIDv7 generator, crypto, format)
```

Setiap modul di `/internal/modules/` **WAJIB** memiliki 4 sub-layer di atas ditambah file `module.go` di root modul.

---

## 2. Standar ID Entitas (UUIDv7)

- **Wajib UUIDv7:** Semua aggregate root dan entity utama menggunakan **UUIDv7** sebagai primary key.
- **Karakteristik UUIDv7:**
  - Mengandung timestamp unix-ms di bagian awal sehingga **terurut secara alami** (*time-ordered*).
  - Ramah terhadap index B-Tree database (mencegah fragmentasi index acak seperti pada UUIDv4).
  - Aman dan tidak bisa ditebak (*non-sequential guessing*) tidak seperti auto-increment integer.

---

## 3. Aturan Tiap Layer (DDD Layering)

1. **`domain/`**:
   - Berisi entity bisnis murni, value objects, aggregate root, domain event definition, dan **interface** repository.
   - **Aturan Ketat:** Dilarang me-`import` package HTTP, SQL/database, ataupun layer `application`/`infrastructure`/`interfaces`.
2. **`application/`**:
   - Berisi command & query handlers (use cases) yang mengorkestrasi domain entity, memanggil repository via interface, dan memicu event.
   - Tempat implementasi alur bisnis, **bukan** tempat menyimpan aturan domain (aturan validasi domain tetap di `domain/`).
3. **`infrastructure/`**:
   - Berisi implementasi konkret repository (query SQL / ORM), adapter pihak ketiga, dan adapter event bus.
4. **`interfaces/`**:
   - Berisi HTTP handlers, route registration, DTO (Data Transfer Objects) request/response, dan fungsi mapping DTO ↔ Domain.
   - **Aturan:** Dilarang menaruh business logic di sini.
5. **`module.go` (Facade Publik)**:
   - Mengekspos interface publik modul yang boleh diakses modul lain secara sinkron.
   - Menyediakan fungsi `Register(router, deps, eventBus)` yang dipanggil oleh `main.go`.

---

## 4. Event Bus & Kontrak Event Antar Modul

- **Definisi Kontrak:** Semua event didefinisikan di package `shared/event` (nama event & struct payload) agar modul lain tidak perlu me-`import` modul pengirim secara penuh.
- **Payload Berisi ID Saja:** Event payload harus immutable dan hanya membawa ID referensi (contoh: `SalesmanID`, `OrderID`, `PurchaseOrderID`), bukan seluruh objek data.
- **Idempotency:** Setiap subscriber wajib idempotent karena event bisa di-retry.
- **Alur Event Resmi Proyek:**

```text
1. Penjualan Selesai (Sales Module)
   → publish OrderPaidEvent{ OrderID, LocationID, SalesmanID *string, Items, Total }
       ├─→ Inventory   : Kurangi stok / lepaskan reservasi
       ├─→ Finance     : Buat entri jurnal akuntansi + catat piutang jika kredit/tempo
       └─→ Commission  : Hitung akrual komisi (jika SalesmanID != nil & modul aktif)

2. Penerimaan Barang dari Supplier (Purchasing Module)
   → publish GoodsReceivedEvent{ PurchaseOrderID, LocationID, Items, ReceivedBy }
       ├─→ Inventory   : Tambah stok di StockItem pada Location tujuan
       └─→ Finance     : Catat hutang dagang (payable) ke supplier

3. Aksi Kritikal Sensitif
   → publish SensitiveActionEvent{ ActionType, ActorID, ResourceID, Metadata }
       └─→ AuditLog    : Catat ke tabel audit_logs permanen
```

- **Subscriber Lisensi:** Modul yang tidak di-unlock **dilarang subscribe** ke event bus. Registrasi listener hanya terjadi di dalam `Register()` modul bersangkutan saat mounting.

---

## 5. Mekanisme Mounting Lisensi di `main.go`

Mounting modul dilakukan secara kondisional di startup. Jangan pernah menaruh `license.Enabled()` di dalam handler atau use case.

```go
// cmd/server/main.go
lic := license.LoadCurrent()

if lic.Enabled("inventory") {
    inventoryMod := inventory.NewModule(db, eventBus)
    inventoryMod.Register(router)
}

if lic.Enabled("purchasing") {
    purchasingMod := purchasing.NewModule(db, eventBus)
    purchasingMod.Register(router)
}

if lic.Enabled("commission") {
    commissionMod := commission.NewModule(db, eventBus)
    commissionMod.Register(router) // Listener event baru di-subscribe di sini
}
```

---

## 6. Aturan Database & Constraint Bisnis

- **Prefix Tabel:** `inv_*`, `purch_*`, `sales_*`, `fin_*`, `comm_*`, `ecom_*`, `shared_*`.
- **Tidak ada Foreign Key lintas modul:** Referensi cukup simpan ID (UUIDv7) dan validasi via service call / domain logic.
- **Folder Migrasi:** Terpisah per modul (`migrations/inventory`, `migrations/purchasing`, dst.) sehingga modul non-aktif tidak perlu dieksekusi migrasinya.
- **Constraint Domain Penting:**
  - `PriceOverride`: Unique constraint aktif per `(product_id, location_id)` — maksimal satu promo aktif.
  - `ProductWarranty`: Maksimal 1 garansi aktif per `type` (`toko` / `pabrik`) per produk.
  - `StockTransfer`: Status wajib melalui alur `pending_approval -> approved -> in_transit -> received`. Approval hanya oleh `superadmin` / `owner`.
  - `PurchaseOrder`: Status `draft -> pending_approval -> approved -> ordered -> received` (atau `rejected`). Approval oleh `superadmin` / `owner`.
  - `GoodsReceipt`: MVP hanya mendukung **100% full receipt** yang dieksekusi oleh `Admin Gudang`.
  - `Location`: Memiliki field `type`: `physical` atau `online`. Gudang storefront adalah row `Location` biasa, bukan tabel terpisah.

---

## 7. Checklist Sebelum Menulis Kode Baru

### Saat Membuat Modul Baru:
- [ ] Buat 4 layer (`domain`, `application`, `infrastructure`, `interfaces`) + `module.go`.
- [ ] Beri prefix pada semua nama tabel di database (`inv_*`, `purch_*`, `sales_*`, dst).
- [ ] Primary key entity menggunakan **UUIDv7**.
- [ ] Pastikan tidak ada FOREIGN KEY ke tabel modul lain.
- [ ] Daftarkan definisi event baru di `shared/event`.
- [ ] Pasang mounting kondisional di `cmd/server/main.go`.

### Saat Membuat Use Case Baru:
- [ ] Apakah aturan bisnis sudah berada di `domain/`?
- [ ] Jika butuh data dari modul lain, apakah menggunakan interface dari `module.go` milik modul tersebut (bukan direct import)?
- [ ] Jika memicu efek samping di modul lain, apakah sudah memakai Event Bus?
- [ ] Jika aksi sensitif (approve PO/transfer/goods receipt), apakah sudah melewati verifikasi step-up auth di endpoint jika aktif?
- [ ] Cek prinsip generic: hindari hardcode istilah/kolom spesifik vertikal produk (misal: elektronik).
