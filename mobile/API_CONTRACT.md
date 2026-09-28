# Dokumentasi Kontrak REST API untuk Agent Mobile Flutter (`mobile/API_CONTRACT.md`)

Dokumen ini adalah **sumber kebenaran tunggal (Single Source of Truth)** untuk spesifikasi request, response, query parameter, dan penanganan error antara aplikasi **Flutter (`Dio`)** dan **Backend Go (`/api/v1/...`)**.

Seluruh class DTO di `lib/shared/models/` dan `lib/modules/inventory/models/` **WAJIB** mengikuti nama field JSON (`snake_case`) dan tipe data yang tertulis di dokumen ini tanpa menggunakan tipe `dynamic` di properti model.

---

## 1. Konfigurasi Dasar HTTP Client (`Dio`)

- **Base URL Default (Development):**
  - Android Emulator: `http://10.0.2.2:8080`
  - Perangkat Fisik / PDA Gudang (Jaringan LAN/Wi-Fi Lokal): `http://<IP_SERVER_GO>:8080`
- **Headers Wajib:**
  ```http
  Content-Type: application/json
  Accept: application/json
  Authorization: Bearer <jwt_token>
  ```
- **Standar Format Error Backend Go (`ErrorResponse`):**
  Setiap respons gagal (`400`, `401`, `403`, `404`, `409`, `422`, `500`) selalu mengembalikan body JSON berikut:
  ```json
  {
    "error": "pesan penjelasan kesalahan dalam bahasa Indonesia"
  }
  ```
- **Standar Format Waktu & Harga:**
  - Tanggal & Waktu penuh (`CreatedAt`, `UpdatedAt`, `StartDate`, `EndDate`): ISO-8601 / RFC3339 (`"2026-09-26T10:15:30Z"`).
  - Tanggal Transaksi (`movement_date`, `adjustment_date`): Format `YYYY-MM-DD` (`"2026-09-26"`).
  - Harga & Nominal Uang (`selling_price`, `purchase_price`, `effective_price`): Bilangan bulat `int` (Rupiah penuh tanpa desimal, memetakan `int64` di Go).

---

## 2. Kontrak Endpoint: Shared Context — Autentikasi & Sesi (`auth`)

### 2.1 Login Pengguna
- **Endpoint:** `POST /api/v1/auth/login`
- **Auth:** Publik (Tanpa Bearer Token)
- **Request Body (`LoginRequest`):**
  ```json
  {
    "username": "gudang_pusat",
    "password": "password123"
  }
  ```
- **Response `200 OK` (`LoginResponse`):**
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "0195f3a0-1234-7000-8000-000000000001",
      "name": "Budi Santoso",
      "username": "gudang_pusat",
      "email": "budi@toko.com",
      "role": "admin",
      "location_id": "0195f3a0-9999-7000-8000-000000000010",
      "is_active": true,
      "created_at": "2026-09-01T08:00:00Z",
      "updated_at": "2026-09-01T08:00:00Z"
    }
  }
  ```
  > **Catatan Penting untuk `SessionController`:**
  > - Jika `user.location_id` bernilai `null` (misalnya `role == "superadmin"` atau `"owner"`), pengguna memiliki akses lintas cabang dan wajib ditampilkan pemilih cabang (*Location Switcher*).
  > - Jika `user.location_id` berisi UUID string, kunci lokasi kerja aktif ke ID tersebut.
- **Error Responses:**
  - `401 Unauthorized`: `{"error": "username atau kata sandi tidak sesuai"}`
  - `403 Forbidden`: `{"error": "akun ini telah dinonaktifkan"}`

---

### 2.2 Ambil Profil Pengguna Aktif
- **Endpoint:** `GET /api/v1/auth/me`
- **Auth:** Wajib Bearer Token
- **Response `200 OK` (`UserResponse`):**
  ```json
  {
    "id": "0195f3a0-1234-7000-8000-000000000001",
    "name": "Budi Santoso",
    "username": "gudang_pusat",
    "email": "budi@toko.com",
    "role": "admin",
    "location_id": "0195f3a0-9999-7000-8000-000000000010",
    "is_active": true,
    "created_at": "2026-09-01T08:00:00Z",
    "updated_at": "2026-09-01T08:00:00Z"
  }
  ```

---

### 2.3 Verifikasi Ulang Password (Step-up Authentication)
Digunakan oleh komponen `ReauthModal` sebelum mengeksekusi aksi kritikal (Approve Transfer Stok, Terima Transfer Stok, atau Submit Stock Opname).
- **Endpoint:** `POST /api/v1/auth/verify-password`
- **Auth:** Wajib Bearer Token
- **Request Body (`VerifyPasswordRequest`):**
  ```json
  {
    "password": "password123"
  }
  ```
- **Response `200 OK` (`VerifyPasswordResponse`):**
  ```json
  {
    "verified": true
  }
  ```
- **Error Responses:**
  - `401 Unauthorized`: `{"error": "kata sandi tidak sesuai"}`

---

### 2.4 Ganti Password Mandiri
- **Endpoint:** `POST /api/v1/auth/change-password`
- **Auth:** Wajib Bearer Token
- **Request Body (`ChangePasswordRequest`):**
  ```json
  {
    "old_password": "password123",
    "new_password": "passwordBaru456"
  }
  ```
- **Response `200 OK`:**
  ```json
  {
    "message": "kata sandi berhasil diperbarui"
  }
  ```

---

## 3. Kontrak Endpoint: Modul Inventory (`/api/v1/inventory`)

### 3.1 Lokasi Cabang & Gudang (`locations`)

#### A. Daftar Semua Lokasi
- **Endpoint:** `GET /api/v1/inventory/locations`
- **Permission:** `inventory.locations.view`
- **Response `200 OK` (`List<LocationResponse>`):**
  ```json
  [
    {
      "id": "0195f3a0-9999-7000-8000-000000000010",
      "code": "GDG-PST",
      "name": "Gudang Pusat",
      "type": "physical",
      "address": "Jl. Industri Raya No. 10",
      "latitude": -6.200000,
      "longitude": 106.816666,
      "is_active": true,
      "created_at": "2026-09-01T08:00:00Z",
      "updated_at": "2026-09-01T08:00:00Z"
    }
  ]
  ```

#### B. Detail Satu Lokasi
- **Endpoint:** `GET /api/v1/inventory/locations/{id}`
- **Permission:** `inventory.locations.view`
- **Response `200 OK`:** Objek tunggal `LocationResponse`.

---

### 3.2 Pemindaian Barcode & Manajemen Barcode Produk (`barcodes`)

#### A. Lookup Produk Berdasarkan Scan Barcode (Endpoint Inti Scanner)
- **Endpoint:** `GET /api/v1/inventory/barcodes/lookup?code={barcode}`
  *(Mendukung query param `?code=...` maupun `?barcode=...`)*
- **Permission:** `inventory.barcodes.view`
- **Response `200 OK` (`ProductLookupResponse`):**
  ```json
  {
    "product": {
      "id": "0195f3b0-1111-7000-8000-000000000001",
      "sku": "ELK-TV-001",
      "category_id": "0195f3a5-0000-7000-8000-000000000001",
      "name": "Smart TV LED 55 Inch 4K",
      "brand": "Samsung",
      "description": "Smart TV UHD 4K",
      "unit": "unit",
      "purchase_price": 6500000,
      "selling_price": 8000000,
      "status": "active",
      "is_ppn": true,
      "flag_serial_tracking": true,
      "weight_gram": 14500,
      "atribut_varian": {
        "ukuran_layar": "55 inch",
        "warna": "Hitam"
      },
      "primary_image_url": "/uploads/products/tv-55.jpg",
      "created_at": "2026-09-01T08:00:00Z",
      "updated_at": "2026-09-01T08:00:00Z"
    },
    "scanned_barcode": {
      "id": "0195f3c0-2222-7000-8000-000000000001",
      "product_id": "0195f3b0-1111-7000-8000-000000000001",
      "barcode": "8806091234567",
      "is_primary": true,
      "created_at": "2026-09-01T08:00:00Z"
    }
  }
  ```
  > **Catatan Penting:** Field `purchase_price` bertipe nullable (`int?` di Dart) karena Backend Go menyembunyikan harga beli (`null`) jika akun yang login tidak memiliki permission `inventory.products.view_cost`.
- **Error Responses:**
  - `404 Not Found`: `{"error": "produk dengan barcode tersebut tidak ditemukan"}`

#### B. Daftar Barcode Milik Satu Produk
- **Endpoint:** `GET /api/v1/inventory/products/{id}/barcodes`
- **Permission:** `inventory.barcodes.view`
- **Response `200 OK`:** `List<BarcodeResponse>`

#### C. Tambah Barcode Baru ke Produk dari Scanner HP
- **Endpoint:** `POST /api/v1/inventory/products/{id}/barcodes`
- **Permission:** `inventory.barcodes.manage`
- **Request Body (`AddBarcodeRequest`):**
  ```json
  {
    "barcode": "8806091234567",
    "is_primary": true
  }
  ```
- **Response `201 Created`:** Objek `BarcodeResponse`.
- **Error Responses:**
  - `409 Conflict`: Barcode sudah terdaftar pada produk lain.

#### D. Hapus Barcode Produk
- **Endpoint:** `DELETE /api/v1/inventory/products/{id}/barcodes/{barcode_id}`
- **Permission:** `inventory.barcodes.manage`
- **Response `200 OK`:** `{"message": "barcode berhasil dihapus"}`

---

### 3.3 Pelacakan & Registrasi Serial Number / IMEI (`serials`)

#### A. Lookup Unit Fisik Berdasarkan Scan Serial Number / IMEI
- **Endpoint:** `GET /api/v1/inventory/serials/lookup?sn={serial_number}`
- **Permission:** `inventory.serials.view`
- **Response `200 OK` (`SerialUnitLookupResponse`):**
  ```json
  {
    "serial_unit": {
      "id": "0195f3d0-3333-7000-8000-000000000001",
      "product_id": "0195f3b0-1111-7000-8000-000000000001",
      "location_id": "0195f3a0-9999-7000-8000-000000000010",
      "serial_number": "SN-SAM55-99887766",
      "status": "tersedia",
      "created_at": "2026-09-10T09:00:00Z",
      "updated_at": "2026-09-10T09:00:00Z"
    },
    "product_name": "Smart TV LED 55 Inch 4K",
    "product_sku": "ELK-TV-001",
    "product_brand": "Samsung",
    "location_name": "Gudang Pusat",
    "location_code": "GDG-PST"
  }
  ```
  > **Nilai Enum `status` Serial Unit:** `"tersedia"`, `"terjual"`, `"retur"`.
- **Error Responses:**
  - `404 Not Found`: `{"error": "unit fisik dengan nomor seri tersebut tidak ditemukan"}`

#### B. Daftar Serial Number (Filter per Produk, Lokasi, Status, atau Pencarian)
- **Endpoint:** `GET /api/v1/inventory/serials?product_id={pid}&location_id={lid}&status={status}&search={q}`
  *(Semua query parameter bersifat opsional)*
- **Permission:** `inventory.serials.view`
- **Response `200 OK` (`List<SerialUnitResponse>`):**
  ```json
  [
    {
      "id": "0195f3d0-3333-7000-8000-000000000001",
      "product_id": "0195f3b0-1111-7000-8000-000000000001",
      "location_id": "0195f3a0-9999-7000-8000-000000000010",
      "serial_number": "SN-SAM55-99887766",
      "status": "tersedia",
      "created_at": "2026-09-10T09:00:00Z",
      "updated_at": "2026-09-10T09:00:00Z",
      "product_name": "Smart TV LED 55 Inch 4K",
      "product_sku": "ELK-TV-001",
      "product_brand": "Samsung",
      "location_name": "Gudang Pusat",
      "location_code": "GDG-PST"
    }
  ]
  ```

#### C. Registrasi Batch Serial Number / IMEI via Scanner
- **Endpoint:** `POST /api/v1/inventory/products/{id}/serials`
- **Permission:** `inventory.serials.register`
- **Request Body (`RegisterSerialUnitsRequest`):**
  ```json
  {
    "location_id": "0195f3a0-9999-7000-8000-000000000010",
    "serial_numbers": [
      "SN-SAM55-99887766",
      "SN-SAM55-99887767",
      "SN-SAM55-99887768"
    ]
  }
  ```
- **Response `201 Created`:** `List<SerialUnitResponse>`
- **Error Responses:**
  - `400 Bad Request`: Produk tidak mengaktifkan pelacakan serial (`flag_serial_tracking == false`).
  - `409 Conflict`: Salah satu nomor seri sudah terdaftar di sistem.

---

### 3.4 Master Produk (`products`) — Untuk Pencarian Manual Saat Barcode Rusak

#### A. Cari & List Produk (Paginated)
- **Endpoint:** `GET /api/v1/inventory/products?search={kata_kunci}&category_id={cid}&status=active&page=1&limit=20`
- **Permission:** `inventory.products.view`
- **Response `200 OK` (`ListProductResponse`):**
  ```json
  {
    "data": [
      {
        "id": "0195f3b0-1111-7000-8000-000000000001",
        "sku": "ELK-TV-001",
        "category_id": "0195f3a5-0000-7000-8000-000000000001",
        "name": "Smart TV LED 55 Inch 4K",
        "brand": "Samsung",
        "description": "Smart TV UHD 4K",
        "unit": "unit",
        "purchase_price": 6500000,
        "selling_price": 8000000,
        "status": "active",
        "is_ppn": true,
        "flag_serial_tracking": true,
        "weight_gram": 14500,
        "atribut_varian": {},
        "primary_image_url": "/uploads/products/tv-55.jpg",
        "created_at": "2026-09-01T08:00:00Z",
        "updated_at": "2026-09-01T08:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20,
    "total_pages": 1
  }
  ```

#### B. Detail Produk Berdasarkan ID
- **Endpoint:** `GET /api/v1/inventory/products/{id}`
- **Permission:** `inventory.products.view`
- **Response `200 OK`:** Objek `ProductResponse`.

---

### 3.5 Transaksi Barang Masuk (`Stock In`) & Barang Keluar (`Stock Out`)

#### A. Simpan Transaksi Barang Masuk (Inbound)
- **Endpoint:** `POST /api/v1/inventory/movements/in`
- **Permission:** `inventory.stocks.adjust`
- **Request Body (`CreateStockMovementRequest`):**
  ```json
  {
    "movement_date": "2026-09-26",
    "location_id": "0195f3a0-9999-7000-8000-000000000010",
    "category_reason": "penerimaan_barang",
    "reference_number": "SJ-SUP-0012",
    "notes": "Penerimaan via Scanner Android",
    "items": [
      {
        "product_id": "0195f3b0-1111-7000-8000-000000000001",
        "quantity": 2,
        "notes": "Kondisi dus baik",
        "serial_numbers": [
          "SN-SAM55-001",
          "SN-SAM55-002"
        ]
      }
    ]
  }
  ```
  > **Aturan Validasi Penting:**
  > - Jika produk memiliki `flag_serial_tracking == true`, maka `serial_numbers` wajib diisi dan panjang array-nya harus sama persis dengan `quantity`.
  > - Jika produk memiliki `flag_serial_tracking == false`, kosongkan atau jangan kirim field `serial_numbers`.
- **Response `201 Created` (`StockMovementResponse`):**
  ```json
  {
    "id": "0195f400-1111-7000-8000-000000000099",
    "movement_number": "MOV-IN-20260926-0001",
    "type": "IN",
    "movement_date": "2026-09-26",
    "location_id": "0195f3a0-9999-7000-8000-000000000010",
    "location_name": "Gudang Pusat",
    "category_reason": "penerimaan_barang",
    "reference_number": "SJ-SUP-0012",
    "notes": "Penerimaan via Scanner Android",
    "executed_by": "0195f3a0-1234-7000-8000-000000000001",
    "executed_by_name": "Budi Santoso",
    "items": [
      {
        "id": "0195f400-2222-7000-8000-000000000100",
        "product_id": "0195f3b0-1111-7000-8000-000000000001",
        "product_name": "Smart TV LED 55 Inch 4K",
        "product_sku": "ELK-TV-001",
        "quantity": 2,
        "notes": "Kondisi dus baik",
        "serial_numbers": ["SN-SAM55-001", "SN-SAM55-002"],
        "created_at": "2026-09-26T10:20:00Z"
      }
    ],
    "created_at": "2026-09-26T10:20:00Z"
  }
  ```

#### B. Simpan Transaksi Barang Keluar (Outbound)
- **Endpoint:** `POST /api/v1/inventory/movements/out`
- **Permission:** `inventory.stocks.adjust`
- **Request Body:** Sama persis dengan `CreateStockMovementRequest`.
- **Response `201 Created`:** Sama persis dengan `StockMovementResponse` (dengan `"type": "OUT"`).

#### C. Daftar Riwayat Pergerakan Barang (Masuk / Keluar)
- **Endpoint:** `GET /api/v1/inventory/movements?type={IN|OUT}&location_id={lid}&category_reason={reason}&start_date={YYYY-MM-DD}&end_date={YYYY-MM-DD}&search={q}&page=1&limit=20`
- **Permission:** `inventory.stocks.view`
- **Response `200 OK`:**
  ```json
  {
    "data": [ /* Array of StockMovementResponse */ ],
    "meta": {
      "page": 1,
      "limit": 20,
      "total_items": 45,
      "total_pages": 3
    }
  }
  ```

#### D. Detail Satu Dokumen Pergerakan Barang
- **Endpoint:** `GET /api/v1/inventory/movements/{id}`
- **Permission:** `inventory.stocks.view`
- **Response `200 OK`:** Objek `StockMovementResponse` lengkap beserta `items` dan `serial_numbers`.

---

### 3.6 Stok Cabang, Peringatan Stok Menipis, & Stock Opname (`stocks`)

#### A. Cek Stok 1 Produk di 1 Lokasi (atau Daftar Semua Stok di 1 Lokasi)
- **Endpoint:**
  - Mode 1 Produk: `GET /api/v1/inventory/stocks?location_id={lid}&product_id={pid}` -> Mengembalikan **Objek Tunggal** `StockResponse`.
  - Mode Semua Produk di Lokasi: `GET /api/v1/inventory/stocks?location_id={lid}` -> Mengembalikan **Array** `List<StockResponse>`.
- **Permission:** `inventory.stocks.view`
- **Response `200 OK` (`StockResponse`):**
  ```json
  {
    "id": "0195f3e0-4444-7000-8000-000000000001",
    "product_id": "0195f3b0-1111-7000-8000-000000000001",
    "location_id": "0195f3a0-9999-7000-8000-000000000010",
    "quantity": 15,
    "reserved_quantity": 2,
    "available_quantity": 13,
    "min_stock": 5,
    "is_low_stock": false,
    "updated_at": "2026-09-26T10:20:00Z"
  }
  ```

#### B. Daftar Peringatan Stok Menipis (*Low Stock Alerts*)
- **Endpoint:** `GET /api/v1/inventory/stocks/alerts?location_id={lid}`
- **Permission:** `inventory.stocks.view`
- **Response `200 OK`:** `List<StockResponse>` (hanya item dengan `quantity <= min_stock`).

#### C. Eksekusi Penyesuaian Stok / Stock Opname (`Adjust Stock`)
- **Endpoint:** `POST /api/v1/inventory/stocks/adjust`
- **Permission:** `inventory.stocks.adjust`
- **Request Body (`AdjustStockRequest`):**
  ```json
  {
    "product_id": "0195f3b0-1111-7000-8000-000000000001",
    "location_id": "0195f3a0-9999-7000-8000-000000000010",
    "adjustment_date": "2026-09-26",
    "new_quantity": 14,
    "reason": "Stock Opname Rutin via Mobile Scanner (Selisih -1)"
  }
  ```
- **Response `200 OK`:** Objek `StockResponse` terbaru setelah disesuaikan.

#### D. Riwayat Penyesuaian Stok / Stock Opname
- **Endpoint:** `GET /api/v1/inventory/stocks/adjustments?location_id={lid}&product_id={pid}&start_date={YYYY-MM-DD}&end_date={YYYY-MM-DD}&page=1&limit=20`
- **Permission:** `inventory.stocks.view`
- **Response `200 OK`:**
  ```json
  {
    "data": [
      {
        "id": "0195f410-5555-7000-8000-000000000001",
        "product_id": "0195f3b0-1111-7000-8000-000000000001",
        "product_name": "Smart TV LED 55 Inch 4K",
        "product_sku": "ELK-TV-001",
        "location_id": "0195f3a0-9999-7000-8000-000000000010",
        "location_name": "Gudang Pusat",
        "adjustment_date": "2026-09-26",
        "previous_quantity": 15,
        "new_quantity": 14,
        "difference": -1,
        "reason": "Stock Opname Rutin via Mobile Scanner (Selisih -1)",
        "adjusted_by": "0195f3a0-1234-7000-8000-000000000001",
        "adjusted_by_name": "Budi Santoso",
        "created_at": "2026-09-26T11:00:00Z"
      }
    ],
    "meta": {
      "page": 1,
      "limit": 20,
      "total_items": 1,
      "total_pages": 1
    }
  }
  ```

---

### 3.7 Mutasi Stok Antar Cabang (`transfers`)

Alur status wajib: `pending_approval` -> `approved` -> `in_transit` -> `received` (atau `rejected`).

#### A. Daftar Dokumen Mutasi Stok
- **Endpoint:** `GET /api/v1/inventory/transfers?from_location_id={lid}&to_location_id={lid}&status={status}`
- **Permission:** `inventory.transfers.view`
- **Response `200 OK` (`List<StockTransferResponse>`):**
  ```json
  [
    {
      "id": "0195f420-6666-7000-8000-000000000001",
      "transfer_number": "TRF-20260926-0001",
      "from_location_id": "0195f3a0-9999-7000-8000-000000000010",
      "to_location_id": "0195f3a0-9999-7000-8000-000000000020",
      "status": "approved",
      "notes": "Pengiriman stok mingguan ke cabang",
      "rejection_reason": "",
      "requested_by": "0195f3a0-1234-7000-8000-000000000001",
      "requested_by_name": "Budi Santoso",
      "approved_by": "0195f3a0-0000-7000-8000-000000000001",
      "approved_by_name": "Super Admin",
      "received_by": null,
      "received_by_name": "",
      "items": [
        {
          "id": "0195f420-7777-7000-8000-000000000002",
          "product_id": "0195f3b0-1111-7000-8000-000000000001",
          "quantity": 1,
          "received_quantity": 0,
          "serial_unit_ids": ["0195f3d0-3333-7000-8000-000000000001"],
          "created_at": "2026-09-26T09:00:00Z"
        }
      ],
      "created_at": "2026-09-26T09:00:00Z",
      "updated_at": "2026-09-26T09:30:00Z"
    }
  ]
  ```

#### B. Detail Satu Dokumen Mutasi Stok
- **Endpoint:** `GET /api/v1/inventory/transfers/{id}`
- **Permission:** `inventory.transfers.view`
- **Response `200 OK`:** Objek `StockTransferResponse`.

#### C. Buat Permohonan Mutasi Stok Baru
- **Endpoint:** `POST /api/v1/inventory/transfers`
- **Permission:** `inventory.transfers.create`
- **Request Body (`CreateStockTransferRequest`):**
  ```json
  {
    "from_location_id": "0195f3a0-9999-7000-8000-000000000010",
    "to_location_id": "0195f3a0-9999-7000-8000-000000000020",
    "notes": "Permohonan mutasi via Mobile Scanner",
    "items": [
      {
        "product_id": "0195f3b0-1111-7000-8000-000000000001",
        "quantity": 1,
        "serial_unit_ids": ["0195f3d0-3333-7000-8000-000000000001"]
      }
    ]
  }
  ```
  > **Catatan Penting:** Berbeda dengan `movements/in` yang mengirimkan string `serial_numbers`, pada `transfers` yang dikirimkan adalah array UUID **`serial_unit_ids`** (diperoleh dari field `serial_unit.id` saat melakukan scan `GET /api/v1/inventory/serials/lookup?sn=...`).
- **Response `201 Created`:** Objek `StockTransferResponse` dengan status `"pending_approval"`.

#### D. Aksi Siklus Hidup Mutasi Stok (`Approve`, `Reject`, `Ship`, `Receive`)
| Aksi | Endpoint | Permission | Request Body | Perubahan Status |
| :--- | :--- | :--- | :--- | :--- |
| **Approve** | `POST /api/v1/inventory/transfers/{id}/approve` | `inventory.transfers.approve` | `{}` (Kosong, wajib Step-up Auth sebelumnya) | `pending_approval` -> `approved` |
| **Reject** | `POST /api/v1/inventory/transfers/{id}/reject` | `inventory.transfers.approve` | `{"reason": "Stok gudang pusat terbatas"}` | `pending_approval` -> `rejected` |
| **Ship (Kirim)** | `POST /api/v1/inventory/transfers/{id}/ship` | `inventory.transfers.ship` | `{}` (Kosong, setelah semua item terverifikasi scan di HP) | `approved` -> `in_transit` |
| **Receive (Terima)** | `POST /api/v1/inventory/transfers/{id}/receive` | `inventory.transfers.receive` | `{}` (Kosong, wajib 100% full receipt scan di HP) | `in_transit` -> `received` |

---

### 3.8 Cek Harga Efektif Cabang, Garansi Produk, & Kartu Stok

#### A. Cek Harga Jual Efektif (Termasuk Promo Aktif Cabang)
- **Endpoint:** `GET /api/v1/inventory/price-overrides/effective-price?product_id={pid}&location_id={lid}`
- **Permission:** `inventory.prices.view`
- **Response `200 OK` (`EffectivePriceResponse`):**
  ```json
  {
    "product_id": "0195f3b0-1111-7000-8000-000000000001",
    "location_id": "0195f3a0-9999-7000-8000-000000000010",
    "base_price": 8000000,
    "effective_price": 7250000,
    "has_discount": true,
    "discount_amount": 750000,
    "remaining_quota": 5,
    "promo_id": "0195f430-8888-7000-8000-000000000001",
    "promo_reason": "Promo Akhir Bulan"
  }
  ```

#### B. Cek Kebijakan Garansi Produk (Toko & Pabrik)
- **Endpoint:** `GET /api/v1/inventory/products/{id}/warranties`
- **Permission:** `inventory.warranties.view`
- **Response `200 OK` (`List<ProductWarrantyResponse>`):**
  ```json
  [
    {
      "id": "0195f440-9999-7000-8000-000000000001",
      "product_id": "0195f3b0-1111-7000-8000-000000000001",
      "warranty_policy_id": "0195f440-0000-7000-8000-000000000001",
      "type": "pabrik",
      "is_active": true,
      "policy": {
        "id": "0195f440-0000-7000-8000-000000000001",
        "name": "Garansi Resmi Samsung Indonesia",
        "type": "pabrik",
        "duration_months": 12,
        "duration_days": 0,
        "coverage": "Panel & Sparepart",
        "claim_instructions": "Bawa nota dan kartu garansi ke Service Center",
        "is_active": true,
        "created_at": "2026-09-01T08:00:00Z",
        "updated_at": "2026-09-01T08:00:00Z"
      },
      "created_at": "2026-09-01T08:00:00Z",
      "updated_at": "2026-09-01T08:00:00Z"
    }
  ]
  ```

#### C. Laporan Kartu Stok Produk di Cabang
- **Endpoint:** `GET /api/v1/inventory/reports/stock-card?product_id={pid}&location_id={lid}&start_date={YYYY-MM-DD}&end_date={YYYY-MM-DD}`
- **Permission:** `inventory.stocks.view`
- **Response `200 OK` (`StockCardReportResponse`):**
  ```json
  {
    "product_id": "0195f3b0-1111-7000-8000-000000000001",
    "product_name": "Smart TV LED 55 Inch 4K",
    "product_sku": "ELK-TV-001",
    "location_id": "0195f3a0-9999-7000-8000-000000000010",
    "location_name": "Gudang Pusat",
    "opening_balance": 10,
    "total_in": 5,
    "total_out": 1,
    "closing_balance": 14,
    "entries": [
      {
        "date": "2026-09-26T10:20:00Z",
        "movement_type": "IN",
        "document_number": "MOV-IN-20260926-0001",
        "reference_number": "SJ-SUP-0012",
        "category_reason": "penerimaan_barang",
        "in_quantity": 2,
        "out_quantity": 0,
        "balance": 12,
        "executed_by_name": "Budi Santoso",
        "notes": "Penerimaan via Scanner Android"
      }
    ]
  }
  ```

---

### 3.9 Master Lokasi & Cabang (`locations`)

#### A. Daftar Seluruh Lokasi Cabang & Gudang
- **Endpoint:** `GET /api/v1/inventory/locations?active_only=true`
- **Permission:** `inventory.locations.view` (Diberikan untuk `admin`, `warehouse`, `cashier`, `superadmin`, `owner`)
- **Headers:** `Authorization: Bearer <token>`
- **Response `200 OK` (`List<LocationResponse>`):**
  ```json
  [
    {
      "id": "01a0ceda-e35c-762f-b19f-8b660299bd2c",
      "code": "WH-JKT-01",
      "name": "Gudang Utama Distribusi Jakarta",
      "type": "physical",
      "address": "Kawasan Industri Pulo Gadung Kav. 12-14, Jakarta Timur",
      "latitude": -6.19142,
      "longitude": 106.91263,
      "is_active": true,
      "created_at": "2026-09-23T15:20:47Z",
      "updated_at": "2026-09-24T12:54:49Z"
    }
  ]
  ```

#### B. Lokasi Penugasan Staf yang Sedang Login
- **Endpoint:** `GET /api/v1/inventory/locations/my`
- **Permission:** Cukup autentikasi login (`Bearer <token>`)
- **Keterangan:** Otomatis mengambil detail lokasi berdasarkan `location_id` yang tersimpan pada token JWT staf.
- **Response `200 OK` (`LocationResponse`):**
  ```json
  {
    "id": "01a0ceda-e35c-762f-b19f-8b660299bd2c",
    "code": "WH-JKT-01",
    "name": "Gudang Utama Distribusi Jakarta",
    "type": "physical",
    "address": "Kawasan Industri Pulo Gadung Kav. 12-14, Jakarta Timur",
    "latitude": -6.19142,
    "longitude": 106.91263,
    "is_active": true,
    "created_at": "2026-09-23T15:20:47Z",
    "updated_at": "2026-09-24T12:54:49Z"
  }
  ```
- **Response `404 Not Found` (Jika akun bersifat global/tanpa penugasan cabang spesifik):**
  ```json
  {
    "error": "akun tidak terikat pada cabang tertentu (akses global)"
  }
  ```

