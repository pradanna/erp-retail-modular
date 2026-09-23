# Arsitektur & Aturan Utama ERP Retail Modular

Dokumen ini adalah **sumber kebenaran utama (Rules)** yang WAJIB dibaca dan dipatuhi oleh AI Agent dalam setiap interaksi, perencanaan, dan pembuatan kode pada proyek **ERP Retail Modular (Go + SvelteKit)** ini.

---

## 1. Identitas Proyek & Arsitektur Utama
- Proyek ini adalah ERP retail yang dijual per instalasi (**single-tenant per toko**).
- Modul-modul bisnis di-unlock sesuai lisensi yang dibeli: **Inventory, Purchasing, Sales, Finance, Ecommerce, Commission**.
- **Pola Arsitektur:**
  - **Backend:** Modular Monolith (Go) dengan pemisahan Bounded Context ala Domain-Driven Design (DDD).
  - **Frontend:** Monorepo (SvelteKit) yang memisahkan **Backoffice SPA** (operasional kasir/admin) dan **Storefront SSR** (ecommerce publik & SEO).
- **Standar ID Entitas:** Seluruh aggregate **WAJIB menggunakan UUIDv7** (urut secara alami/time-ordered, ramah index database, dan aman dari tebakan id berurutan).

---

## 2. Prinsip Inti (Dilarang Keras Dilanggar)

1. **Satu Modul = Satu Bounded Context:**
   - Modul tidak boleh me-`import` package `domain` atau `infrastructure` milik modul lain secara langsung.
2. **Komunikasi Antar Modul Hanya Lewat 2 Jalur:**
   - **Sinkron:** Hanya melalui interface publik (facade) yang diekspos modul pada `module.go`.
   - **Asinkron:** Melalui in-process Event Bus untuk efek samping yang bersifat *eventually-consistent*.
3. **Tidak Ada Foreign Key Lintas Modul di Level Database:**
   - Referensi antar modul disimpan sebagai ID biasa (contoh: `salesman_id`, `product_id`, `supplier_id`), dan divalidasi lewat service call atau domain logic — **BUKAN JOIN SQL lintas modul**.
4. **Mekanisme Unlock Modul Berbasis Lisensi:**
   - Modul yang tidak di-unlock **TIDAK BOLEH ter-mount sama sekali**.
   - Route HTTP tidak boleh terdaftar, event listener tidak boleh subscribe, dan migrasi modul tidak boleh dijalankan.
   - Pengecekan lisensi HANYA diizinkan di titik startup/mounting (`main.go`), **JANGAN PERNAH** menyebarkan `if license.Enabled(...)` di dalam use case/handler individual.
5. **Generic Dulu, Spesifik Retail/Elektronik Belakangan:**
   - Domain model harus menghindari kolom/field spesifik toko elektronik (misal: dilarang membuat kolom `kapasitas_liter` di tabel produk). Gunakan atribut varian generic/fleksibel (JSON object). Elektronik adalah data transaksi, bukan skema database.

---

## 3. Aturan Database & Domain Invariants Global

### 3.1 Prefix Tabel per Modul
- `inv_*` untuk Inventory (contoh: `inv_products`, `inv_stock_transfers`, `inv_barcodes`)
- `purch_*` untuk Purchasing (contoh: `purch_suppliers`, `purch_orders`, `purch_goods_receipts`)
- `sales_*` untuk Sales (contoh: `sales_orders`, `sales_payments`)
- `fin_*` untuk Finance (contoh: `fin_journals`, `fin_accounts`)
- `comm_*` untuk Commission (contoh: `comm_accruals`)
- `ecom_*` untuk Ecommerce
- `shared_*` atau tanpa prefix untuk shared context (contoh: `users`, `locations`, `audit_logs`, `settings`)

### 3.2 Migrasi Database Terpisah
- Folder migrasi database harus dikelompokkan per modul agar modul non-aktif tidak perlu dieksekusi migrasinya.

### 3.3 Domain Invariants Kunci
- **PriceOverride:** Unique constraint aktif per `(product_id, location_id)` — maksimal satu promo aktif.
- **ProductWarranty:** Maksimal satu `WarrantyPolicy` aktif per `type` (`toko` / `pabrik`) per produk.
- **StockTransfer:** Status wajib melalui alur `pending_approval -> approved -> in_transit -> received`. Approval hanya oleh `superadmin` / `owner`.
- **PurchaseOrder:** Status melalui `draft -> pending_approval -> approved -> ordered -> received` (atau `rejected`). Approval oleh `superadmin` / `owner`.
- **GoodsReceipt:** MVP hanya mendukung **100% full receipt** yang dieksekusi oleh `Admin Gudang` (Admin yang di-assign ke Location tujuan).
- **Location:** Memiliki `type` (`physical` atau `online`). Gudang storefront adalah row `Location` biasa, bukan tabel terpisah.

---

## 4. Keamanan & Cross-Cutting Shared

1. **Audit Log (`shared/audit`):**
   - Fondasi akuntabilitas yang **selalu aktif** (tidak bisa dimatikan).
   - Mencatat seluruh aksi penting: approval transfer, approval PO, konfirmasi goods receipt, perubahan promo harga, pencairan komisi.
   - Diisi otomatis dengan subscribe ke event-event penting lewat event bus.
2. **Step-up Authentication (`shared/settings`):**
   - Konfirmasi ulang password sebelum mengeksekusi aksi kritikal (approve transfer stok, approve PO, goods receipt).
   - Divalidasi ganda: **Frontend modal dialog** dan **Backend verification endpoint**.
   - Dikelola via toggle operasional di `shared/settings` (bisa diaktifkan/dimatikan tanpa redeploy, dan berbeda dari lisensi modul).

---

## 5. Standar Kode & Kualitas (Zero-Warning Policy)

### 5.1 TypeScript Strict & Type Safety (Frontend)
- **Dilarang keras memakai `any`** (baik implisit maupun eksplisit).
- Jika tipe belum pasti (misal response dinamis/JSON parse): gunakan `unknown` lalu sempitkan (*narrowing*) via type guard atau Zod.
- Tipe request/response API **tidak boleh ditulis manual ganda**. Sumber tunggal harus dari `/packages/types` (hasil generate OpenAPI atau Zod schema).
- Props Svelte WAJIB dideklarasikan dengan tipe eksplisit. Dilarang memakai `// @ts-ignore`.

### 5.2 Styling & Tema Visual (Tailwind CSS v4)
- Gunakan **Tailwind CSS v4** dengan engine CSS-first (`@theme`).
- **Tema Visual:** Clean, modern, dominan **biru** (bukan hijau), palet netral (putih/abu-abu Slate), card-based layout, dan rounded corner moderat.
- Token warna dan tema didefinisikan satu kali di `/packages/ui` via `@theme`, dilarang hardcode kode hex langsung di dalam komponen.
- Hindari `@apply` berlebihan; tulis utility class Tailwind langsung pada markup komponen.
- **Standar Icon:** Seluruh icon WAJIB menggunakan **Heroicons** (format SVG outline 24x24 atau Heroicons solid). Dilarang memakai library icon lain tanpa izin.
- **Larangan Icon Sparkle:** Dilarang keras memakai icon sparkle dalam bentuk apapun.
- **Larangan Emoticon:** Dilarang keras memakai emoticon atau emoji di dalam antarmuka UI maupun teks dokumentasi kode resmi.

### 5.3 Governance Komponen UI (Component-First Workflow)
- **Komponen Dulu, Halaman Kemudian:** Sebelum membuat tampilan halaman baru, **WAJIB cek `/packages/ui/COMPONENTS.md`**.
- Jika komponen belum ada: **Buat komponen reusable terlebih dahulu** di `/packages/ui/components/`, lengkapi props TypeScript & runes, dan daftarkan ke `/packages/ui/COMPONENTS.md`. Dilarang menulis markup mentah (seperti tag `<button>`, `<input>`, wrapper card khusus) secara acak di halaman.
- Jika sudah ada tapi butuh variasi: **extend via props/variant**, jangan buat komponen duplikat baru.

### 5.4 Pola Arsitektur DDD Modular di Frontend
- Frontend dibangun **per-fase** terstruktur dan bertahap sama seperti pembangunan backend Go.
- Struktur monorepo frontend mencerminkan Bounded Context:
  - `/packages/ui` : Design system, token CSS, dan atomic reusable components.
  - `/packages/types` : Single source of truth tipe entitas dan DTO backend.
  - `/packages/api-client` : Modular HTTP client terpisah per domain modul.
  - `/apps/backoffice/src/routes/(app)/<module>` : Halaman modular terlindungi guard lisensi & auth.

---

## 6. Anti-Pattern yang Wajib Ditolak

- ❌ Me-`import` `internal/modules/X/domain` atau `infrastructure` dari dalam modul `Y` (Harus lewat `module.go` milik `X`).
- ❌ Foreign key constraint atau SQL JOIN lintas tabel milik modul yang berbeda.
- ❌ Menggunakan auto-increment integer sebagai ID utama (Gunakan **UUIDv7**).
- ❌ Pengecekan `license.Enabled()` di dalam handler/use case individual (hanya boleh saat registrasi modul di `main.go`).
- ❌ Hardcode field/kolom spesifik vertikal produk (`kapasitas_liter`, `resolusi_pixel`) sebagai kolom tabel.
- ❌ Modul yang tidak di-unlock mendaftar ke Event Bus lalu melakukan no-op.
- ❌ Anotasi `: any` atau `// @ts-ignore` untuk sekadar meloloskan build atau commit.
- ❌ Menulis markup/komponen baru dari nol di halaman tanpa membuat/cek `/packages/ui/COMPONENTS.md`.
- ❌ Hardcode kode warna hex di komponen tanpa melalui token `@theme`.
- ❌ Menggunakan icon sparkle atau icon di luar standar Heroicons.
- ❌ Menggunakan emoticon / emoji di dalam UI maupun dokumen sistem.

---

## 7. Referensi Workflow & Panduan Detail (Skills)
Untuk prosedur teknis langkah-demi-langkah, gunakan skill berikut:
- **Backend (Go):** `.agents/skills/be-modular-architecture/SKILL.md` (Layering DDD, Facade `module.go`, Event Bus, Wiring).
- **Frontend (SvelteKit):** `.agents/skills/fe-sveltekit-architecture/SKILL.md` (Monorepo, SvelteKit, Tailwind v4, Component Governance, Route Guarding).

---

## 8. Pendekatan Edukasi & Penjelasan Kode (Learning-First Mentality)

Proyek ini dibangun bukan hanya untuk menghasilkan aplikasi yang berfungsi, tetapi juga sebagai **sarana belajar mendalam** bagi user mengenai arsitektur Domain-Driven Design (DDD) layered, ekosistem Go, dan SvelteKit modern.

### Kewajiban Agent Setiap Kali Selesai Memberikan / Mengubah Kode:
1. **Penjelasan yang Mudah Dipahami:**
   - Selalu sertakan penjelasan ringkas, terstruktur, dan edukatif mengenai kode yang baru dibuat atau diubah (hindari sekadar melempar potongan kode mentah).
2. **Alasan Arsitektural (*Why & Where*):**
   - Jelaskan **mengapa** file atau kode tersebut diletakkan di layer/tempat tertentu (contoh: mengapa interface ditaruh di `domain`, mengapa logika pemanggilan repo di `application`, mengapa query SQL di `infrastructure`, atau mengapa DTO di `interfaces`).
3. **Soroti Konsep Kunci Go / SvelteKit:**
   - Berikan catatan edukatif singkat jika ada pola khas Go (contoh: *implicit interfaces*, *struct composition*, *dependency injection*, *pointer vs value receiver*, *UUIDv7 generation*) atau SvelteKit (contoh: *load function*, *runes/props type safety*, *SPA vs SSR context*, *Tailwind v4 `@theme`*).
4. **Hubungkan dengan Prinsip Utama:**
   - Tunjukkan bagaimana implementasi tersebut menjaga prinsip Bounded Context, isolasi database tanpa FK lintas modul, atau lisensi modul yang sedang dipelajari.
5. **Gunakan Analogi Sederhana Dunia Nyata:**
   - **Wajib sertakan analogi intuitif dunia nyata** setiap kali menjelaskan konsep arsitektur, pola kode, atau komponen teknis baru (contoh: *ServeMux* = Resepsionis pengarah jalan, *module.go* = Meja Front Desk modul, *Event Bus* = Pengeras suara mall).
   - Rangkum dan catat konsep-konsep baru serta analoginya ke dalam file [belajar.md](file:///c:/PROJECT/WEBSITE/erp-retail-modular/belajar.md) sebagai jurnal belajar user.

