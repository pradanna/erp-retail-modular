---
name: erp-modular-architecture
description: Aturan wajib untuk menulis kode backend Go dan frontend SvelteKit pada proyek ERP retail modular ini. WAJIB dibaca dan diikuti setiap kali membuat modul baru, use case baru, endpoint baru, entity/aggregate baru, event baru, atau mengubah struktur folder — baik di backend (Go) maupun frontend (SvelteKit). Cakupan: modular monolith, layering ala DDD (domain/application/infrastructure/interfaces), boundary antar modul, event bus in-process, mekanisme unlock modul berbasis lisensi, aturan database per-modul, dan struktur monorepo frontend (backoffice SPA + storefront SSR). Selalu konsultasi skill ini sebelum menulis kode baru pada repo ini, jangan hanya mengikuti pola dari file terdekat yang kebetulan terbuka.
---

# Arsitektur ERP Retail Modular (Go + SvelteKit)

Proyek ini adalah ERP retail yang dijual per instalasi (single-tenant per toko), dengan modul yang bisa di-unlock sesuai lisensi: **Inventory, Sales, Finance, Ecommerce, Commission**. Pola arsitektur: **modular monolith** di backend, **dua aplikasi dalam satu monorepo** di frontend. Skill ini adalah sumber kebenaran untuk struktur kode — ikuti ini, bukan menebak dari file yang kebetulan sudah ada.

## 1. Prinsip Inti (jangan dilanggar)

1. **Satu modul = satu bounded context.** Modul tidak boleh `import` package `domain` atau `infrastructure` milik modul lain secara langsung.
2. **Komunikasi antar modul hanya lewat dua jalur:**
   - **Sinkron** → lewat interface publik (facade) yang diekspos modul di `module.go`.
   - **Asinkron** → lewat event bus in-process, untuk efek samping yang boleh eventually-consistent.
3. **Tidak ada foreign key lintas modul di level database.** Referensi antar modul disimpan sebagai ID biasa, divalidasi lewat service call — bukan JOIN.
4. **Modul yang tidak di-unlock tidak boleh ter-mount sama sekali** — bukan hanya disembunyikan di UI, tapi route-nya benar-benar tidak terdaftar dan listener event-nya tidak subscribe.
5. **Generic dulu, spesifik elektronik belakangan.** Domain model harus menghindari istilah spesifik toko elektronik di level struktur (mis. jangan bikin field `kapasitas_liter` di tabel product) — pakai atribut varian fleksibel. Elektronik adalah data/contoh, bukan skema.

## 2. Struktur Folder Backend (Go)

```
/cmd
  /server                  -> main.go: baca license, wiring semua modul
/internal
  /modules
    /inventory
      /domain              -> entity, value object, repository interface, domain service
      /application         -> use case (command/query handler)
      /infrastructure       -> implementasi repo (Postgres/MySQL), event publisher
      /interfaces           -> http handler, DTO, route registration
      module.go             -> facade: interface publik + fungsi Register()
    /sales
    /finance
    /commission
    /ecommerce
    /shared                 -> auth, user, location, license/module-registry, event bus
  /platform                 -> db connection, config, logger, migration runner
/pkg                        -> util generic lintas project (bukan domain logic)
```

Setiap modul WAJIB punya keempat layer (`domain`, `application`, `infrastructure`, `interfaces`) plus satu `module.go` di root modul.

### Aturan tiap layer

- **domain/** — entity, value object, aggregate root, repository **interface** (bukan implementasi), domain event definition, domain service (business rule murni). Tidak boleh import package database atau http.
- **application/** — use case/command handler yang orkestrasi domain + repository (via interface) + publish event. Ini tempat logic "apa yang terjadi kalau X", bukan aturan bisnis itu sendiri (itu di domain).
- **infrastructure/** — implementasi konkret repository (query SQL/ORM), implementasi event publisher, adapter ke sistem luar (payment gateway, dsb).
- **interfaces/** — HTTP handler, DTO request/response, mapping DTO↔domain, route registration. Tidak boleh berisi business logic.
- **module.go** — expose interface publik modul (facade) yang boleh dipakai modul lain, plus `Register(router, deps, eventBus)` yang dipanggil dari `main.go` HANYA kalau modul itu enabled di lisensi.

## 3. Event Bus & Kontrak Event

Event adalah kontrak resmi antar modul — treat seperti public API.

- Semua event didefinisikan di `shared/event` (nama, payload), bukan di dalam modul yang men-publish, supaya modul lain bisa depend ke definisi event tanpa depend ke seluruh modul sumber.
- Event WAJIB immutable dan berisi ID (bukan objek penuh) untuk referensi lintas modul — contoh: `SalesmanID`, bukan objek `Salesman` penuh.
- Contoh alur wajib yang sudah disepakati:

```
Order dibayar (Sales)
  → publish OrderPaid{ OrderID, LocationID, SalesmanID *string, Items, Total }
      → Inventory   : kurangi stok / lepas reservasi
      → Finance     : buat jurnal + piutang jika kredit
      → Commission  : hitung akrual bonus (hanya jika SalesmanID != nil DAN modul aktif)
```

- Subscriber wajib **idempotent** — event bisa saja diproses ulang (retry), jangan asumsikan sekali proses.
- Modul yang tidak di-unlock **tidak boleh subscribe** — jangan subscribe lalu no-op di dalam handler; jangan register handler-nya sama sekali.

## 4. Mekanisme Unlock Modul (Lisensi)

- License dibaca sekali saat startup (`shared/license`), hasilnya daftar modul yang aktif untuk instalasi ini.
- Di `main.go`, tiap modul di-*mount* secara kondisional:

```go
if license.Enabled("inventory") {
    inventoryModule.Register(router, deps, eventBus)
}
if license.Enabled("commission") {
    commissionModule.Register(router, deps, eventBus)
    // subscribe hanya terjadi di dalam Register() modul ini
}
```

- **Jangan pernah** taruh pengecekan `if license.Enabled(...)` di dalam handler/use case individual — itu tandanya modul tidak ter-boundary dengan benar. Pengecekan hanya boleh terjadi satu kali, di titik mounting.

## 5. Aturan Database

- Satu tabel = milik satu modul. Prefix nama tabel sesuai modul: `inv_products`, `sales_orders`, `fin_journals`, `comm_accruals`.
- Tidak ada foreign key constraint lintas modul.
- Migration dikelompokkan per modul (folder migration terpisah), supaya modul yang tidak di-unlock tidak perlu migration-nya dijalankan.
- Aturan bisnis yang sudah disepakati dan HARUS tercermin di constraint/domain:
  - `PriceOverride`: unique constraint aktif per **(product_id, location_id)** — maksimal satu promo aktif.
  - `StockTransfer`: status wajib melalui alur `pending_approval → approved → in_transit → received`; hanya role `superadmin`/`owner` yang bisa transisi ke `approved`.
  - `Location` punya `type`: `physical` atau `online` — gudang storefront adalah row `Location` biasa, bukan tabel terpisah.

## 6. Struktur Frontend (SvelteKit Monorepo)

```
/apps
  /backoffice     -> SPA murni (export const ssr = false), untuk Inventory/Sales/Finance/Commission
  /storefront     -> SSR/prerender aktif (untuk SEO), untuk Ecommerce publik
/packages
  /ui             -> komponen shared
  /api-client     -> fetch wrapper ke backend Go
  /types          -> shared TypeScript types
```

- Routing di `backoffice` mengikuti struktur modul: `routes/inventory/...`, `routes/sales/...`, dst.
- Menu DAN akses route di `backoffice` di-guard oleh `enabled_modules` yang dikirim backend saat login — jangan hanya sembunyikan menu di sidebar, cegah juga akses langsung via URL di level `load` function.
- `storefront` tidak boleh depend ke kode internal `backoffice`; keduanya hanya berbagi lewat `/packages`.

### 6.1 Aturan TypeScript Strict (wajib, semua app & package)

- `tsconfig.json` di setiap app/package (`backoffice`, `storefront`, `ui`, `api-client`, `types`) WAJIB mengaktifkan:
  ```json
  {
    "compilerOptions": {
      "strict": true,
      "noImplicitAny": true,
      "noUncheckedIndexedAccess": true,
      "exactOptionalPropertyTypes": true
    }
  }
  ```
- **`any` dilarang di seluruh codebase** — termasuk implisit (`noImplicitAny`) maupun eksplisit (`: any`). ESLint wajib mengaktifkan rule `@typescript-eslint/no-explicit-any` sebagai **error**, bukan warning.
- Kalau tipe benar-benar belum diketahui (mis. hasil `JSON.parse`, response API sebelum divalidasi): pakai `unknown`, lalu sempit-kan tipenya lewat type guard atau schema validator (mis. Zod) sebelum dipakai — jangan langsung cast ke `any`.
- Tipe request/response API tidak boleh ditulis manual dua kali (BE Go dan FE terpisah). Pilih salah satu:
  - Generate TypeScript types dari OpenAPI/Swagger spec yang diekspos backend Go, taruh hasilnya di `/packages/types`, atau
  - Definisikan schema di `/packages/types` pakai Zod dan pakai itu sebagai satu-satunya sumber tipe request/response di kedua app.
- `catch (err)` di TypeScript otomatis bertipe `unknown`, bukan `any` — jangan pernah anotasi ulang jadi `any`; sempitkan dengan `err instanceof Error` sebelum akses `.message`.
- Komponen Svelte: props WAJIB dideklarasikan dengan tipe eksplisit (pakai `interface Props` atau generic `$props<Props>()` sesuai versi Svelte yang dipakai) — tidak boleh mengandalkan inferensi implisit yang jatuh ke `any`.
- Library pihak ketiga tanpa tipe: buat file deklarasi `.d.ts` sendiri di `/packages/types`, jangan longgarkan aturan global lewat `// @ts-ignore` atau `any` lokal.

### 6.2 Styling: Tailwind CSS (versi terbaru, Svelte-native)

- Pakai **Tailwind CSS v4** (bukan v3) — versi ini pakai engine CSS-first (`@import "tailwindcss"` di satu file CSS, konfigurasi lewat `@theme` di CSS, bukan lagi `tailwind.config.js` berbasis JS). Cek versi terbaru saat instalasi karena Tailwind rilis cukup cepat.
- Instalasi lewat plugin resmi `@tailwindcss/vite` (bukan PostCSS manual) — ini yang direkomendasikan untuk project berbasis Vite/SvelteKit karena build lebih cepat dan setup lebih sedikit.
- Token desain (warna brand, spacing custom, dsb) didefinisikan sekali di `/packages/ui` lewat `@theme`, di-*import* oleh `backoffice` dan `storefront` — supaya kedua app konsisten visual tanpa duplikasi konfigurasi.
- Kelas Tailwind ditulis langsung di markup komponen Svelte (`class="..."`), **hindari** `@apply` berlebihan di file CSS terpisah — itu menghilangkan salah satu manfaat utama utility-first dan bikin refactor lebih susah dilacak.
- Urutan kelas Tailwind dirapikan otomatis oleh Prettier plugin (lihat 6.3), jangan diurutkan manual.

### 6.3 Lint & Format: ESLint + Prettier (wajib, zero-warning)

Tujuannya: kode yang lolos commit/CI harus **nol warning**, bukan cuma nol error.

- Setup **ESLint** dengan `typescript-eslint` (strict config) + `eslint-plugin-svelte` untuk lint file `.svelte`.
- Setup **Prettier** dengan `prettier-plugin-svelte` (format markup Svelte) dan `prettier-plugin-tailwindcss` (auto-sort kelas Tailwind sesuai konvensi resmi) — dua plugin ini wajib, bukan opsional.
- Konfigurasi ESLint di-set agar **warning diperlakukan sebagai kegagalan**, bukan cuma error:
  ```json
  { "scripts": { "lint": "eslint . --max-warnings=0" } }
  ```
  Gunakan `--max-warnings=0` di script lint maupun di CI, supaya tidak ada warning yang lolos tanpa disadari.
- ESLint dan Prettier tidak boleh saling konflik aturan formatting — pakai `eslint-config-prettier` untuk mematikan rule ESLint yang tumpang tindih dengan Prettier (Prettier yang menang untuk urusan format, ESLint fokus ke correctness/code quality).
- Konfigurasi (`eslint.config.js`, `.prettierrc`) ditaruh satu kali di root monorepo dan dipakai bersama oleh `apps/backoffice`, `apps/storefront`, dan semua `/packages/*` — jangan duplikat config per app.
- Jalankan lint + format check sebagai bagian dari pre-commit hook (mis. `lint-staged` + `husky`) supaya kode yang ber-warning tidak sempat masuk ke commit, bukan baru ketahuan saat CI.

### 6.4 Tema Visual (Storefront & Backoffice)

Arah desain: **clean, modern, cocok untuk kalangan umum** (bukan niche/dark-theme/eksperimental) — banyak whitespace, hierarki tipografi jelas, card-based layout, rounded corner moderat (bukan tajam, bukan terlalu bulat). Warna utama: **biru** (bukan hijau), dengan palet netral (putih/abu) sebagai basis, sesuai pola referensi berikut:

- **Hero section** — headline besar + subheadline singkat + CTA button warna primer + carousel indicator (dots)
- **Grid kategori** — card sederhana: ikon/gambar produk representatif + label kategori, grid 4 kolom di desktop
- **Grid produk** — card: gambar produk, nama, harga (format Rupiah), rating bintang, badge diskon jika ada
- **Halaman detail produk (PDP)** — galeri gambar di kiri, info produk (nama, harga, varian warna/opsi, CTA "Tambah ke Keranjang") di kanan, section rekomendasi ("You May Also Like") di bawah

Definisikan token warna biru ini sekali di `/packages/ui` lewat `@theme` (Tailwind v4), dipakai bersama oleh `backoffice` dan `storefront` — jangan hardcode kode hex warna di komponen individual:

```css
@theme {
  --color-primary-50: #eff6ff;
  --color-primary-500: #2563eb;
  --color-primary-600: #1d4ed8;
  --color-primary-700: #1e40af;
}
```

Detail nuansa biru (tone terang/gelap, aksen sekunder) dan skala tipografi didiskusikan lagi saat mulai styling komponen konkret — bagian ini hanya mengunci **arah** (biru, clean, modern, grid card-based), bukan nilai final tiap token.

### 6.5 Governance Komponen (wajib sebelum membuat halaman/tampilan baru)

Tujuan: mencegah duplikasi komponen (mis. 3 versi `Button` berbeda gaya) yang bikin UI tidak konsisten antar halaman/antar app.

- **Sebelum membuat tampilan/halaman baru, WAJIB cek dulu `/packages/ui` apakah komponen yang dibutuhkan sudah ada** (Button, Card, Badge, Input, Modal, ProductCard, RatingStars, dll) — jangan langsung menulis markup+style baru dari nol di dalam route/page.
- Kalau komponen sudah ada tapi butuh variasi (ukuran, warna, state baru): **extend lewat props/variant** pada komponen yang sama, jangan bikin file/komponen duplikat dengan nama berbeda.
- Kalau komponen benar-benar belum ada: buat di `/packages/ui/components/<NamaKomponen>.svelte`, dengan props bertipe eksplisit (lihat aturan 6.1), lalu **daftarkan** ke `/packages/ui/COMPONENTS.md` (index singkat: nama komponen, fungsi, props utama) supaya menjadi rujukan pencarian berikutnya — file ini WAJIB dibaca sebelum menambah komponen baru, bukan hanya ditulis sekali lalu dilupakan.
- Komponen generic (Button, Card, Input, dsb) tidak boleh mengandung logic spesifik domain (mis. jangan taruh logic harga/diskon di dalam `Card` generic) — komponen domain-spesifik (`ProductCard`, `CategoryCard`) boleh dibangun di atas komponen generic tadi, dan tetap didaftarkan di index yang sama.
- `backoffice` dan `storefront` sama-sama wajib pakai komponen dari `/packages/ui` untuk elemen UI dasar — style lokal per app hanya untuk layout/komposisi, bukan untuk elemen dasar seperti tombol/input/badge.

## 7. Checklist Sebelum Membuat Kode Baru

**Menambah modul baru:**
- [ ] Buat 4 subfolder layer + `module.go`
- [ ] Tabel baru pakai prefix modul, tanpa FK ke modul lain
- [ ] Definisikan event yang di-publish/di-subscribe di `shared/event`
- [ ] Tambahkan pengecekan `license.Enabled("<modul>")` HANYA di titik mounting di `main.go`

**Menambah use case/fitur di modul yang sudah ada:**
- [ ] Business rule baru masuk ke `domain/`, bukan `application/` atau `interfaces/`
- [ ] Kalau butuh data dari modul lain: pakai interface dari `module.go` modul tersebut, jangan import langsung
- [ ] Kalau efeknya "efek samping" (bukan bagian inti dari use case ini): pertimbangkan event, bukan pemanggilan langsung

**Menambah field/entity yang berbau spesifik-elektronik:**
- [ ] Cek dulu: bisakah ini jadi atribut varian generic, bukan kolom khusus?

**Menulis kode TypeScript baru (FE):**
- [ ] Tidak ada `any` eksplisit maupun implisit — pakai `unknown` + type guard/Zod kalau tipe belum pasti
- [ ] Tipe request/response API bersumber dari `/packages/types`, bukan didefinisikan ulang lokal
- [ ] Props komponen Svelte punya tipe eksplisit
- [ ] Styling pakai kelas Tailwind langsung di markup, token custom dari `/packages/ui`
- [ ] `npm run lint` (atau setara) lolos dengan nol warning sebelum commit

**Membuat halaman/tampilan baru (FE):**
- [ ] Sudah cek `/packages/ui/COMPONENTS.md` — komponen yang dibutuhkan sudah ada atau belum?
- [ ] Kalau ada tapi kurang variasi: extend props/variant, bukan duplikat komponen baru
- [ ] Kalau benar-benar baru: dibuat di `/packages/ui`, props bertipe eksplisit, lalu didaftarkan ke `COMPONENTS.md`
- [ ] Warna & gaya visual mengikuti tema biru/clean/modern di 6.4, bukan warna ad-hoc

## 8. Anti-pattern yang Harus Ditolak

- Import `internal/modules/X/domain` dari dalam modul Y → **selalu salah**, ganti dengan interface di `module.go` milik X.
- Foreign key lintas tabel modul berbeda → **selalu salah**.
- Pengecekan `license.Enabled()` tersebar di banyak tempat (bukan cuma di mounting) → tandanya modul bocor ke modul lain.
- Field/tabel yang hardcode istilah elektronik (`kapasitas_liter`, `ukuran_layar_inch` sebagai kolom) → seharusnya jadi baris di tabel atribut varian generic.
- Anotasi `: any` atau `// @ts-ignore` untuk "biar cepat lolos build" → **selalu salah**, perbaiki tipenya atau pakai `unknown` + narrowing.
- Menulis komponen/markup baru dari nol di dalam halaman tanpa cek `/packages/ui` dulu → berisiko duplikasi (mis. 3 versi Button berbeda gaya) → **selalu cek COMPONENTS.md dulu**.
- Hardcode warna hex langsung di komponen (bukan lewat token `@theme`) → **selalu salah**, akan menyulitkan konsistensi tema biru di seluruh app.
