---
name: fe-sveltekit-architecture
description: Panduan wajib untuk membuat dan memodifikasi kode frontend SvelteKit pada monorepo ERP retail modular. Gunakan skill ini setiap kali membuat halaman baru di backoffice/storefront, routing modul, komponen UI bersama di /packages/ui, konsumsi API backend, styling Tailwind v4, linting ESLint/Prettier, modal step-up auth, atau konfigurasi guard akses modul berdasarkan lisensi.
---

# Panduan Arsitektur Frontend SvelteKit (Monorepo)

Gunakan panduan ini saat membuat atau mengubah kode frontend pada proyek ERP Retail Modular.

---

## 1. Struktur Monorepo Frontend

```text
/apps
  /backoffice     -> SvelteKit SPA murni (`export const ssr = false`) untuk operasional kasir/admin
                     (Modul: Inventory, Purchasing, Sales/POS, Finance, Commission)
  /storefront     -> SvelteKit dengan SSR/Prerender aktif untuk Ecommerce publik & SEO
/packages
  /ui             -> Komponen reusable (Button, Card, Modal, Input, Badge, ProductCard, dll)
  /api-client     -> Type-safe wrapper untuk komunikasi ke backend REST API Go
  /types          -> Shared TypeScript definitions (DTO, response schema Zod/OpenAPI, enum)
```

---

## 2. Prinsip Isolasi Aplikasi
- **Independensi Storefront:** Aplikasi `storefront` **DILARANG KERAS** mengimpor kode internal dari `backoffice` (dan sebaliknya).
- **Berbagi Kode:** Semua komponen UI, client service, dan tipe data yang dipakai bersama harus berada di `/packages/`.

---

## 3. Struktur Routing & Lisensi Module Guard di Backoffice

- **Struktur Route per Modul:**
  - `routes/(app)/inventory/...`   -> Master produk, stok per cabang, transfer stok, garansi
  - `routes/(app)/purchasing/...`  -> Master supplier, Purchase Order, Goods Receipt
  - `routes/(app)/sales/...`       -> POS kasir, sales order, retur
  - `routes/(app)/finance/...`     -> Jurnal, piutang, hutang supplier
  - `routes/(app)/commission/...`  -> Skema bonus salesman, akrual & pencairan
- **Lisensi Guard:** Backend mengirimkan array `enabled_modules` saat login.
- **Proteksi Akses:** Selain menyembunyikan navigasi di sidebar, wajib blokir akses URL langsung pada level `load` function atau routing guard (redirect / throw 403 Forbidden).

---

## 4. Keamanan: Step-up Authentication Modal Dialog

Untuk aksi kritikal di Backoffice (Approve Transfer Stok, Approve Purchase Order, Konfirmasi Goods Receipt):
- Jika `Settings.require_reauth` aktif di backend, frontend **wajib menampilkan Modal Dialog Konfirmasi Password** sebelum mengirim request mutasi sensitif.
- Komponen modal step-up auth diletakkan di `/packages/ui/components/ReauthModal.svelte` dan didaftarkan di `COMPONENTS.md`.
- Token/password dikirim ke endpoint mutasi backend untuk diverifikasi ulang di server.

---

## 5. Aturan TypeScript Strict (Wajib di Semua App & Package)

Setiap `tsconfig.json` (`backoffice`, `storefront`, `ui`, `api-client`, `types`) **WAJIB** mengaktifkan:
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

- **`any` Dilarang:** Baik implisit maupun eksplisit. ESLint wajib menetapkan `@typescript-eslint/no-explicit-any` sebagai **error**.
- **Data Tak Tentu:** Gunakan `unknown`, lalu sempitkan dengan *type guard* atau schema validator (Zod) sebelum diakses.
- **Single Source of Truth untuk Tipe API:**
  - Tipe request/response tidak boleh ditulis ulang manual. Gunakan generator dari spec OpenAPI Go backend atau skema Zod di `/packages/types`.
- **Catch Clause:** `catch (err)` bertipe `unknown`. Sempitkan dengan `if (err instanceof Error)` sebelum membaca `.message`.
- **Svelte Props:** Props wajib bertipe eksplisit (menggunakan `interface Props` atau generic `$props<Props>()`).
- **Pihak Ketiga Tanpa Tipe:** Buat deklarasi `.d.ts` di `/packages/types`, dilarang memakai `// @ts-ignore`.

---

## 6. Styling: Tailwind CSS v4 (Svelte-Native)

- Gunakan **Tailwind CSS v4** dengan arsitektur CSS-first (`@import "tailwindcss"` di file CSS, bukan `tailwind.config.js`).
- Integrasi melalui plugin resmi `@tailwindcss/vite`.
- Token desain didefinisikan satu kali di `/packages/ui` via `@theme`:
  ```css
  @theme {
    --color-primary-50: #eff6ff;
    --color-primary-500: #2563eb;
    --color-primary-600: #1d4ed8;
    --color-primary-700: #1e40af;
  }
  ```
- Tulis utility class langsung pada atribut `class="..."` di markup komponen. **Hindari** `@apply` berlebihan di file CSS terpisah.
- Urutan utility class dirapikan secara otomatis oleh `prettier-plugin-tailwindcss`.

---

## 7. Tema Visual & Pola Halaman

- **Karakter Desain:** Clean, modern, dominan **biru** (bukan hijau), palet netral (putih/abu-abu), card-based layout, dan rounded corner moderat.
- **Pola Komponen Halaman:**
  - **Hero Section:** Headline besar + subheadline singkat + CTA primer + indikator dots.
  - **Grid Kategori:** Card sederhana: ikon/gambar + label, 4 kolom di desktop (2 level hierarki kategori).
  - **Grid Produk:** Card: gambar, judul, format harga Rupiah (`Intl.NumberFormat`), bintang rating, diskon badge.
  - **Detail Produk (PDP):** Galeri kiri, info produk & opsi varian di kanan, section rekomendasi di bawah.
  - **Storefront Checkout (MVP):** Menggunakan instruksi pembayaran manual/offline (transfer bank/kasir), tanpa payment gateway otomatis pihak ketiga.

---

## 8. Governance Komponen UI (`/packages/ui`)

Untuk mencegah redundansi (contoh: 3 variasi `Button` berbeda gaya):
1. **Wajib Cek Index:** Sebelum membuat komponen baru, periksa index dokumentasi di `/packages/ui/COMPONENTS.md`.
2. **Extend, Jangan Duplikasi:** Jika butuh variasi (warna, ukuran, state), tambahkan varian props pada komponen yang sudah ada.
3. **Pendaftaran Komponen Baru:** Buat di `/packages/ui/components/<NamaKomponen>.svelte`, dengan tipe props eksplisit, lalu daftarkan ke `COMPONENTS.md`.
4. **Pemisahan Generic vs Domain:** Komponen dasar (`Button`, `Card`, `Input`, `Modal`) tidak boleh mengandung logic domain. Komponen domain (`ProductCard`, `ReauthModal`) dibangun di atas komponen dasar tersebut.

---

## 9. Linting & Formatting (Zero-Warning Policy)

- ESLint dikonfigurasi dengan:
  ```json
  { "scripts": { "lint": "eslint . --max-warnings=0" } }
  ```
- Menggunakan `eslint-plugin-svelte`, `typescript-eslint` strict, `prettier-plugin-svelte`, dan `prettier-plugin-tailwindcss`.
- Gunakan `eslint-config-prettier` agar tidak ada konflik antara linter dan formatter.

---

## 10. Checklist Sebelum Menulis Kode Frontend

- [ ] Cek `/packages/ui/COMPONENTS.md` sebelum membuat UI baru.
- [ ] Props komponen didefinisikan dengan tipe eksplisit.
- [ ] Tidak ada anotasi `: any` atau `// @ts-ignore`.
- [ ] Warna menggunakan token `@theme` (palet biru / netral), bukan hardcode hex.
- [ ] Route di backoffice terproteksi oleh `enabled_modules`.
- [ ] Aksi sensitif menyertakan dialog re-auth jika step-up authentication aktif.
- [ ] Menjalankan `npm run lint` dengan hasil nol error dan nol warning.
