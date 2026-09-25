# Aturan & Sistem Desain Frontend (ERP Retail Modular)

Dokumen ini adalah spesifikasi resmi mengenai **sistem desain, tema visual, aturan komponen, dan alur pengerjaan frontend** untuk seluruh aplikasi (`apps/backoffice` dan `apps/storefront`).

---

## 1. Filosofi & Karakter Visual

Antarmuka ERP retail dirancang untuk efisiensi operasional tinggi (kasir, kepala gudang, admin pembelian, akuntan).

- **Corporate & Trustworthy:** Dominan warna biru _Royal Enterprise_ yang memberikan kesan stabilitas, ketelitian data, dan profesionalisme.
- **High Information Density (Scannable):** Teks ringkas, pemisahan garis border tipis, dan tipografi monospaced/tabular untuk angka uang serta barcode.
- **Card-Based Architecture:** Seluruh modul dibungkus dalam Card putih bersih (`bg-white`) di atas kanvas abu-abu lembut (`bg-neutral-50`).
- **Zero-Distraction Policy:**
  - DILARANG menggunakan emoticon / emoji dalam antarmuka maupun dokumen teknis.
  - DILARANG menggunakan icon efek sparkle dalam bentuk apapun.
  - Seluruh icon WAJIB menggunakan standar **Heroicons** (outline 24x24 atau solid 20x20).

---

## 2. Token Warna & Palet Resmi (Tailwind CSS v4 `@theme`)

Sumber kebenaran tunggal warna berada di `/packages/ui/styles/theme.css`. Dilarang _hardcode_ kode hex di komponen.

### 2.1 Primary (Gen-E Monochrome Obsidian / Black)

- `primary-50` (`#f4f4f5`): Background menu aktif halus, avatar tint, badge tint
- `primary-100` (`#e4e4e7`): Background badge aktif, chip filter
- `primary-200` (`#d4d4d8`): Border halus badge & aksen
- `primary-500` (`#27272a`): Hover tombol aksi utama, border focus ring
- **`primary-600` (`#09090b`): Tombol aksi utama (CTA), tab navigasi aktif, avatar icon**
- `primary-700` (`#000000`): Teks kontras tinggi, tombol aktif solid
- `primary-900` (`#000000`): Teks kontras tertinggi

### 2.2 Neutral & Secondary (Slate Cool-Gray)

- `neutral-50` (`#f8fafc`): Kanvas latar belakang utama aplikasi
- `neutral-100` (`#f1f5f9`): Header tabel, tombol sekunder (batal, filter)
- `neutral-200` (`#e2e8f0`): Garis pembatas (border) kartu, divider tabel
- `neutral-400` (`#94a3b8`): Placeholder input form
- `neutral-500` (`#64748b`): Teks keterangan pendukung, icon sekunder
- `neutral-700` (`#334155`): Teks label form, konten tabel
- `neutral-900` (`#0f172a`): Judul utama, sidebar latar belakang gelap

### 2.3 Status Semantik Bisnis

- **Success** (`success-600` `#16a34a`): Status _Received_, _Approved_, _In Stock_, _Lunas_
- **Warning** (`warning-500` `#f59e0b`): Status _Pending Approval_, _In Transit_, _Low Stock_
- **Danger** (`danger-600` `#dc2626`): Status _Rejected_, _Out of Stock_, Tombol Hapus/Batal
- **Info** (`info-600` `#2563eb` / `info-50` `#eff6ff`): Status _Kategori Utama_, catatan panduan, badge informatif

---

## 3. Tata Kelola Komponen (Component-First Workflow)

Sebelum membuat halaman atau form baru:

1. **Wajib Cek Registry:** Buka `/packages/ui/COMPONENTS.md` untuk melihat komponen yang sudah tersedia.
2. **Komponen Dulu, Halaman Kemudian:**
   - Dilarang menulis tag mentah `<button class="...">`, `<input class="...">`, atau kontainer modal buatan sendiri di file halaman rute.
   - Buat komponen reusable di `/packages/ui/components/<NamaKomponen>.svelte` terlebih dahulu.
   - Daftarkan komponen ke `/packages/ui/COMPONENTS.md` dan export di `/packages/ui/index.ts`.
3. **Standar Penulisan Komponen Svelte 5:**
   - Gunakan interface TypeScript eksplisit untuk deklarasi props.
   - Gunakan runes `$props()`, `$bindable()`, `$state()`, dan `$derived()`.
   - Dilarang menggunakan anotasi `: any` atau `// @ts-ignore`.

---

## 4. Standar Icon (Heroicons Only)

- Seluruh kebutuhan visual icon menggunakan **Heroicons** versi outline (stroke 1.5/2.0) atau solid.
- Contoh pola implementasi inline SVG Heroicons yang konsisten:
  ```svelte
  <!-- Heroicons Outline 24x24 -->
  <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
    <path stroke-linecap="round" stroke-linejoin="round" d="..." />
  </svg>
  ```
- Dilarang mencampur dengan FontAwesome, Lucide, Material Icons, atau emoji.

---

## 5. Alur Pengerjaan Bertahap (Modular Phase-by-Phase)

Pengerjaan frontend dilakukan per-fase yang terisolasi dan dapat diverifikasi:

- **Fase FE-1: Fondasi Core UI & Autentikasi**
  - Pustaka komponen dasar: Button, Input, Card, Alert, Badge.
  - Halaman Login Backoffice terhubung ke API backend `/api/v1/auth/login`.
  - Store auth reaktif (Svelte 5 runes) dengan persistensi token di `localStorage`.
- **Fase FE-2: App Shell & Navigasi Modular**
  - Layout induk `(app)` dengan Corporate Dark Sidebar dan Topbar navigasi.
  - Auth guard di level rute (redirect ke `/login` jika sesi kedaluwarsa).
  - Filter menu sidebar dinamis berdasarkan lisensi modul yang aktif.
- **Fase FE-3: Modul Inventory**
  - Halaman Master Produk, Varian, dan Kategori.
  - Manajemen Multi-Barcode dan Serial Number Lookup.
  - Transfer Stok antar cabang dengan step-up authentication.
- **Fase Selanjutnya:** Purchasing -> Sales -> Finance -> Commission.
