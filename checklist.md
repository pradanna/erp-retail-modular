# Panduan Pengujian & Checklist Fitur ERP Retail Modular

Dokumen ini adalah lembar kerja pengujian manual (*Manual Quality Assurance Checklist*) komprehensif untuk memverifikasi seluruh fungsionalitas sistem **ERP Retail Modular (Go + SvelteKit)**, diurutkan mulai dari data master kategori produk hingga modul operasional dan keamanan sistem.

---

## Informasi Lingkungan Pengujian
- **URL Frontend (Backoffice):** [http://localhost:5173](http://localhost:5173)
- **URL Backend API:** [http://localhost:8088](http://localhost:8088)
- **Dokumentasi API Interaktif (Scalar):** [http://localhost:8088/docs](http://localhost:8088/docs)
- **Dokumentasi API (Swagger UI):** [http://localhost:8088/swagger](http://localhost:8088/swagger)

### Akun Standar Pengujian (Seeded Accounts)
| Peran (Role) | Username | Password | Keterangan Akses |
| :--- | :--- | :--- | :--- |
| **Owner** | `owner` | `password123` | Pemilik Bisnis, bypass seluruh otorisasi sistem |
| **Superadmin** | `superadmin` | `password123` | Administrator Sistem Utama, bypass sistem |
| **Admin Cabang** | `admin_pusat` | `password123` | Admin Operasional, memiliki hak lihat HPP & Stock Opname |
| **Admin Gudang** | `gudang_01` | `password123` | Staf Gudang, HPP tersamar, hak mutasi & penerimaan |
| **Kasir** | `kasir_01` | `password123` | Kasir Kas, HPP tersamar, hak pencarian barang & harga jual |

---

## 1. Data Master: Kategori Produk (`/master/categories`)
File sumber: [categories/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/master/categories/+page.svelte)

- [x] **1.1 Navigasi & Tampilan Antarmuka**
  - [x] Buka menu **Data Master -> Kategori Produk** di navigasi sidebar.
  - [x] Pastikan kartu ringkasan menampilkan statistik total kategori, kategori induk, dan subkategori.
  - [x] Pastikan tabel menampilkan kolom gambar/thumbnail, nama kategori, slug, kategori induk, jumlah produk terkait, dan menu aksi.

- [x] **1.2 Tambah Kategori Induk Baru**
  - [x] Klik tombol **Tambah Kategori**.
  - [x] Isi field nama kategori (contoh: `Audio & Aksesoris`).
  - [x] Pastikan slug terisi otomatis sesuai nama kategori (contoh: `audio-aksesoris`).
  - [x] Kosongkan pilihan kategori induk (jadikan kategori tingkat atas).
  - [x] Unggah gambar/ikon kategori (mendukung file PNG, JPG, WEBP).
  - [x] Pastikan preview gambar langsung tampil di dalam modal.
  - [x] Klik **Simpan Kategori** dan pastikan notifikasi toast sukses muncul.
  - [x] Pastikan kategori baru langsung muncul di tabel tanpa perlu refresh halaman manual.

- [x] **1.3 Tambah Subkategori**
  - [x] Klik kembali **Tambah Kategori**.
  - [x] Isi nama subkategori (contoh: `Headphone Bluetooth`).
  - [x] Pada dropdown **Kategori Induk**, pilih `Audio & Aksesoris`.
  - [x] Simpan dan pastikan di tabel subkategori tersebut menampilkan badge nama induknya dengan benar.

- [x] **1.4 Edit Data Kategori**
  - [x] Klik tombol titik tiga (Action Menu) pada baris kategori, lalu pilih **Edit Kategori**.
  - [x] Ubah nama deskripsi atau ganti foto gambar kategori dengan file baru.
  - [x] Simpan perubahan dan verifikasi data di tabel telah ter-update seketika.

- [x] **1.5 Validasi Keamanan Penghapusan (Foreign Key Integrity)**
  - [x] Coba hapus kategori yang memiliki relasi produk aktif (contoh: kategori `Smartphone`).
  - [x] Sistem wajib menolak dan menampilkan pesan peringatan bahwa kategori masih menaungi produk.
  - [x] Coba hapus kategori baru yang masih kosong tanpa produk.
  - [x] Sistem berhasil menghapus baris kategori tersebut secara bersih.

---

## 2. Data Master: Katalog Produk (`/master/products`)
File sumber: [products/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/master/products/+page.svelte)

- [ ] **2.1 Tampilan & Filter Katalog**
  - [ ] Buka menu **Data Master -> Katalog Produk**.
  - [ ] Uji filter pencarian cepat berdasarkan nama barang, SKU, atau brand.
  - [ ] Uji filter dropdown kategori dan filter status (Aktif / Nonaktif).
  - [ ] Pastikan navigasi paginasi tabel berfungsi mulus saat berpindah halaman.

- [ ] **2.2 Tambah Produk Baru**
  - [ ] Klik **Tambah Produk Baru**.
  - [ ] Isi Kode SKU unik (contoh: `SONY-WH1000XM5-BLK`).
  - [ ] Isi Nama Barang, Brand, Deskripsi, dan pilih Kategori yang telah dibuat sebelumnya.
  - [ ] Tentukan Satuan Unit (contoh: `pcs` atau `unit`).
  - [ ] Masukkan **Berat (Kg)** (contoh: `50` untuk 50 kg, atau `0.5` untuk 500 gram), dan pastikan sistem otomatis mengonversinya ke satuan basis database (gram).
  - [ ] Isi **Harga Beli (HPP / Modal)** dan **Harga Jual Standar**: Pastikan saat mengetik angka nominal (misal `1500000`), sistem otomatis menampilkan titik pemisah ribuan (`1.500.000`) dengan prefix `Rp` untuk mencegah salah hitung jumlah nol.
  - [ ] Tentukan status PPN (dikenakan pajak atau tidak).
  - [ ] Aktifkan opsi **Pelacakan Nomor Seri (Serial Number / IMEI)** jika barang memerlukan kontrol serial fisik.
  - [ ] Tambahkan **Atribut Varian Generik** (contoh: `warna: Hitam`, `konektivitas: Wireless`):
    - [ ] Uji **Tombol Enter pada Input Varian**: Ketik nama atribut varian lalu tekan `Enter` (kursor otomatis melompat ke nilai atribut); ketik nilai varian lalu tekan `Enter` (sistem otomatis menambah baris varian baru dan mengarahkan kursor ke input baru, tanpa menyimpan form secara prematur).
    - [ ] Uji **Fixed Sticky Modal Footer**: Scroll konten modal produk yang panjang; pastikan tombol aksi ("Batal" dan "Simpan / Tambah Produk") tetap menempel (*fixed/sticky*) di bagian bawah modal tanpa ikut ter-scroll hilang ke atas/bawah.
  - [ ] Simpan produk dan pastikan produk berhasil tersimpan ke database.

- [ ] **2.3 Galeri Foto Produk**
  - [ ] Buka form modal Tambah/Edit Produk.
  - [ ] Uji fitur **Drag & Drop**: Seret berkas gambar (JPG, PNG, WebP) dari File Explorer komputer langsung ke kotak dropzone.
  - [ ] Pastikan kotak dropzone menyala aktif dengan highlight warna primary saat berkas melintas di atasnya (*dragover*).
  - [ ] Lepaskan (*drop*) berkas gambar, pastikan gambar langsung masuk ke daftar antrean pratinjau foto secara mulus tanpa error.
  - [ ] Uji unggah 2 atau lebih foto galeri produk.
  - [ ] Uji fitur **Drag to Reorder Kartu Foto**: Seret salah satu kartu foto dan letakkan di posisi paling kiri (index 0).
  - [ ] Pastikan foto di posisi paling kiri otomatis menjadi **Foto Utama** (ditandai dengan badge "Utama" dan border hijau/indigo).
  - [ ] Hapus salah satu foto dan pastikan berkas terhapus secara bersih.

- [ ] **2.4 Proteksi HPP via PBAC (Field-Level Security & Masking)**
  - [ ] Login menggunakan akun `admin_pusat` / `owner`: Kolom Harga Pokok (HPP) dan marjin laba tampil terbuka.
  - [ ] Login menggunakan akun `warehouse` atau `kasir_01`:
    - [ ] Kolom Harga Pokok di tabel menampilkan teks tersamar `••••••••` dengan ikon gembok kecil.
    - [ ] Buka Network tab di browser (F12) saat memuat `/api/v1/inventory/products`: Pastikan field `purchase_price` bernilai `null` murni dari server (bukan sekadar disembunyikan CSS).

---

## 3. Data Master: Multi-Barcode Produk (`/master/barcodes`)
File sumber: [barcodes/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/master/barcodes/+page.svelte)

- [ ] **3.1 Tampilan Multi-Barcode**
  - [ ] Buka menu **Data Master -> Barcode Produk**.
  - [ ] Pilih produk tertentu untuk melihat seluruh barcode yang terdaftar.

- [ ] **3.2 Pendaftaran Multi-Barcode per SKU**
  - [ ] Klik **Tambah Barcode**.
  - [ ] Masukkan kode barcode unik (contoh barcode box distributor atau barcode unit ritel).
  - [ ] Pilih jenis standar: `EAN-13`, `UPC-A`, `Code-128`, atau `Custom`.
  - [ ] Set status barcode sebagai **Barcode Utama (Primary)**.
  - [ ] Daftarkan barcode kedua untuk produk yang sama sebagai barcode alternatif.

- [ ] **3.3 Uji Coba Quick Scanner / Lookup Barcode**
  - [ ] Masukkan kode barcode ke kolom uji scanner cepat.
  - [ ] Sistem berhasil menemukan produk yang cocok secara instan tanpa mengekspos harga beli.

---

## 4. Data Master: Kebijakan Garansi Produk (`/master/warranties`)
File sumber: [warranties/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/master/warranties/+page.svelte)

- [ ] **4.1 Pembuatan Master Kebijakan Garansi**
  - [ ] Buka menu **Data Master -> Garansi Produk**.
  - [ ] Klik **Buat Kebijakan Garansi**.
  - [ ] Tentukan Nama Kebijakan (contoh: `Garansi Resmi Distributor 12 Bulan`).
  - [ ] Pilih Tipe Garansi: `Garansi Resmi Pabrik (Pabrik)` atau `Garansi Toko (Toko)`.
  - [ ] Tentukan Durasi Masa Garansi (contoh: 12 Bulan, 0 Hari).
  - [ ] Tuliskan Cakupan Perlindungan (*Coverage*) dan Petunjuk Prosedur Klaim.
  - [ ] Simpan kebijakan garansi baru.

- [ ] **4.2 Penetapan Garansi ke Produk (Product Warranty)**
  - [ ] Buka tab **Penetapan Garansi Produk**.
  - [ ] Tautkan kebijakan garansi yang telah dibuat ke produk tertentu.
  - [ ] Uji Invariant Sistem: Coba tetapkan dua garansi aktif dengan tipe yang sama (`toko`) ke satu produk -> Sistem wajib menolak (maksimal 1 garansi aktif per tipe).

---

## 5. Data Master: Cabang & Lokasi Gudang (`/master/locations`)
File sumber: [locations/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/master/locations/+page.svelte)

- [ ] **5.1 Tampilan Cabang & Indikator Tipe**
  - [ ] Buka menu **Data Master -> Cabang & Lokasi**.
  - [ ] Pastikan tabel membedakan cabang tipe **Fisik (Physical Store)** dan **Online (Ecommerce Warehouse)**.

- [ ] **5.2 Tambah Cabang Baru dengan Integrasi Peta Leaflet**
  - [ ] Klik **Tambah Lokasi Cabang**.
  - [ ] Isi Kode Lokasi (contoh: `TK-BDG`), Nama Cabang (contoh: `Toko Cabang Bandung`), dan Alamat.
  - [ ] Klik tombol **Pilih di Peta (Map Picker)**.
  - [ ] Uji fitur pencarian alamat otomatis (*OSM Nominatim Geocoding*).
  - [ ] Geser marker pin peta ke titik lokasi toko yang akurat.
  - [ ] Pastikan koordinat Latitude & Longitude terisi otomatis secara presisi.
  - [ ] Simpan lokasi baru.

- [ ] **5.3 Modal Preview Peta Lokasi**
  - [ ] Pada baris tabel lokasi yang telah memiliki koordinat, klik tombol lihat peta.
  - [ ] Modal peta satelit/OpenStreetMap berhasil terbuka dan menunjukkan posisi cabang.

---

## 6. Inventaris & Stok: Monitoring, Opname & Valuasi (`/inventory/stocks`)
File sumber: [stocks/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/inventory/stocks/+page.svelte)

- [ ] **6.1 Monitoring Stok Multicabang**
  - [ ] Buka menu **Inventaris & Stok -> Stok Cabang**.
  - [ ] Filter stok berdasarkan cabang tertentu.
  - [ ] Verifikasi kolom: Stok Fisik (*On Hand*), Kuantitas Dicadangkan (*Reserved*), dan Stok Tersedia (*Available*).
  - [ ] Pastikan rumus konsisten: `Stok Tersedia = Stok Fisik - Stok Reservasi`.

- [ ] **6.2 Detail Stok & Valuasi Aset Terproteksi**
  - [ ] Klik baris stok untuk membuka modal **Rincian Detail Stok**.
  - [ ] Verifikasi kalkulasi **Valuasi Aset Total = Stok Fisik x HPP**.
  - [ ] Pastikan akun `warehouse` atau `kasir` melihat nilai modal dan valuasi terenkripsi (`••••••••` - Akses Dibatasi).

- [ ] **6.3 Eksekusi Stock Opname Fisik (Pimpinan Only)**
  - [ ] Login sebagai `admin_pusat`, `owner`, atau `superadmin`.
  - [ ] Klik menu aksi -> pilih **Stock Opname (Penyesuaian)**.
  - [ ] Masukkan jumlah fisik riil di rak (contoh: stok tercatat 10, fisik riil 9).
  - [ ] Sistem otomatis menampilkan selisih kuantitas `-1 unit`.
  - [ ] Pilih alasan opname (contoh: `Barang Rusak / Cacat`).
  - [ ] Tuliskan catatan berita acara opname.
  - [ ] Klik **Simpan Penyesuaian Stok** -> Angka stok fisik langsung ter-update di database.

- [ ] **6.4 Hak Akses PBAC Terkunci untuk Staf Gudang & Kasir**
  - [ ] Login sebagai `gudang_01` atau `kasir_01`.
  - [ ] Buka `/inventory/stocks`.
  - [ ] Pastikan tombol/opsi **Stock Opname** tidak muncul di antarmuka.
  - [ ] Coba eksekusi manual request API `POST /api/v1/inventory/stocks/adjust` via console -> Server membalas `403 Forbidden`.

- [ ] **6.5 Riwayat Berita Acara Opname (Stock Adjustment History)**
  - [ ] Klik tombol **Riwayat Opname** di header halaman stok.
  - [ ] Modal menampilkan tabel seluruh aktivitas opname: Nama Barang, Lokasi, Stok Lama, Stok Baru, Selisih, Alasan, Catatan, Nama Staf Pelaksana, dan Timestamp.

---

## 7. Inventaris & Stok: Mutasi Antar Cabang (`/inventory/transfers`)
File sumber: [transfers/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/inventory/transfers/+page.svelte)

- [ ] **7.1 Pembuatan Permohonan Surat Jalan Mutasi**
  - [ ] Buka menu **Inventaris & Stok -> Mutasi Stok**.
  - [ ] Klik **Buat Permohonan Mutasi**.
  - [ ] Pilih Cabang Asal dan Cabang Tujuan (sistem menolak jika asal dan tujuan sama).
  - [ ] Tambahkan item produk ke dalam builder mutasi.
  - [ ] Pastikan kuantitas yang diajukan tidak melebihi stok yang tersedia di cabang asal.
  - [ ] Masukkan catatan pengiriman dan submit.
  - [ ] Dokumen mutasi baru terbentuk dengan status `pending_approval`.
  - [ ] Cek halaman stok cabang asal: kuantitas reservasi (*reserved_quantity*) bertambah otomatis.

- [ ] **7.2 Alur Penolakan Mutasi (Reject)**
  - [ ] Login sebagai pimpinan (`superadmin`/`owner`).
  - [ ] Buka salah satu mutasi berstatus `pending_approval`.
  - [ ] Klik **Tolak Permohonan**.
  - [ ] Masukkan alasan penolakan wajib (contoh: `Kapasitas gudang cabang tujuan sedang penuh`).
  - [ ] Dokumen berubah status menjadi `rejected` dan reservasi stok asal otomatis dilepaskan.

- [ ] **7.3 Alur Persetujuan Mutasi (Approve)**
  - [ ] Buka dokumen mutasi lain berstatus `pending_approval`.
  - [ ] Klik **Setujui Permohonan (Approve)**.
  - [ ] Status berubah menjadi `approved` (Disetujui Siap Kirim).
  - [ ] **Verifikasi Human-Readable Approver:** Buka modal rincian surat jalan, pastikan pada Timeline Step 2 tertulis **Disetujui oleh <Nama Lengkap Staf>** (bukan deretan UUID mentah).

- [ ] **7.4 Alur Keberangkatan Ekspedisi (Ship)**
  - [ ] Klik tombol **Konfirmasi Pengiriman (Ship)**.
  - [ ] Status berubah menjadi `in_transit` (Dalam Pengiriman).
  - [ ] Cek stok cabang asal: stok fisik terpotong resmi, kuantitas reservasi kembali nol.

- [ ] **7.5 Alur Penerimaan Barang Cabang Tujuan (Receive)**
  - [ ] Klik tombol **Konfirmasi Penerimaan Barang (Receive)**.
  - [ ] Status mutasi berubah menjadi `received` (Diterima Lengkap).
  - [ ] Cek stok cabang tujuan: stok fisik otomatis bertambah sesuai unit yang dikirim.
  - [ ] Jika produk berserial, seluruh unit serial berpindah lokasi ke cabang tujuan.

---

## 8. Inventaris & Stok: Pelacakan Serial & IMEI (`/inventory/serials`)
File sumber: [serials/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/inventory/serials/+page.svelte)

- [ ] **8.1 Pendaftaran Nomor Seri Unit Fisik**
  - [ ] Buka menu **Inventaris & Stok -> Serial & IMEI**.
  - [ ] Klik **Registrasi Nomor Seri**.
  - [ ] Pilih produk bertipe serial dan tentukan lokasi penyimpanan awal.
  - [ ] Masukkan nomor seri/IMEI unik (contoh: `SN-S24U-9901`).
  - [ ] Simpan dan pastikan unit berstatus awal `available`.

- [ ] **8.2 Siklus Hidup Status Serial**
  - [ ] Uji pembaruan status unit serial melalui menu aksi:
    - [ ] `available` (Tersedia siap jual).
    - [ ] `defective` (Unit cacat pabrik / rusak).
    - [ ] `transferred` (Sedang dalam proses mutasi).
  - [ ] Verifikasi riwayat pergerakan unit serial tercatat di sistem.

---

## 9. Inventaris & Stok: Promo & Price Override Cabang (`/inventory/price-overrides`)
File sumber: [price-overrides/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/inventory/price-overrides/+page.svelte)

- [ ] **9.1 Pembuatan Promo Diskon Khusus Cabang**
  - [ ] Buka menu **Inventaris & Stok -> Promo Cabang**.
  - [ ] Klik **Buat Promo Cabang Baru**.
  - [ ] Pilih produk dan cabang berlakunya promo.
  - [ ] Masukkan **Harga Promo Khusus** (harus lebih rendah dari harga jual normal).
  - [ ] Tentukan kuota unit promo (contoh: 10 unit) dan rentang tanggal promo aktif.
  - [ ] Masukkan alasan promo (contoh: `Promo Diskon Akhir Pekan Cabang Surabaya`).
  - [ ] Simpan promo baru.

- [ ] **9.2 Validasi Domain Invariant Promo**
  - [ ] Coba buat promo kedua untuk produk dan cabang yang sama pada rentang waktu aktif yang sama.
  - [ ] Sistem wajib menolak (maksimal 1 promo aktif per pasangan produk dan lokasi).

- [ ] **9.3 Pengujian Harga Efektif (Effective Price Calculation)**
  - [ ] Buka detail produk di cabang bersangkutan -> Harga yang berlaku adalah harga promo.
  - [ ] Uji deaktivasi promo manual -> Harga seketika kembali ke harga jual normal.

---

## 10. Sistem & Otorisasi: Manajemen Staf & Pengguna (`/users`)
File sumber: [users/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/users/+page.svelte)

- [ ] **10.1 Monitoring Pengguna Sistem**
  - [ ] Buka menu **Sistem & Otorisasi -> Staf & Pengguna**.
  - [ ] Pastikan kartu statistik menampilkan total staf, staf aktif, staf nonaktif, dan distribusi peran.
  - [ ] Verifikasi tabel menampilkan avatar inisial, nama staf, username, email, badge peran semantik, dan cabang penugasan.

- [ ] **10.2 Registrasi Akun Staf Baru**
  - [ ] Klik tombol **Tambah Staf Baru**.
  - [ ] Isi Nama Lengkap (contoh: `Budi Hartono`), Username (`budi_gudang`), Email, dan Password awal.
  - [ ] Pilih Peran: `warehouse` (Admin Gudang).
  - [ ] Pilih Penugasan Cabang via dropdown Select2.
  - [ ] Simpan dan pastikan akun baru langsung terdaftar di tabel.

- [ ] **10.3 Pembekuan & Pengaktifan Akun Staf**
  - [ ] Klik menu aksi pada akun staf -> pilih **Nonaktifkan Akun**.
  - [ ] Konfirmasi pembekuan akun.
  - [ ] Buka sesi browser incognito / logout, lalu coba login dengan akun tersebut.
  - [ ] Sistem wajib menolak login dengan pesan bahwa akun telah dinonaktifkan.
  - [ ] Login kembali sebagai admin dan aktifkan kembali akun tersebut -> Staf dapat login kembali secara normal.

---

## 11. Sistem & Otorisasi: Matriks Hak Akses Peran / PBAC (`/roles`)
File sumber: [roles/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/roles/+page.svelte)

- [ ] **11.1 Tampilan Matriks Otorisasi Granular**
  - [ ] Buka menu **Sistem & Otorisasi -> Hak Akses Peran**.
  - [ ] Pastikan seluruh 38 izin kapabilitas granular tersusun rapi per Bounded Context (`inventory`, `shared`).
  - [ ] Pastikan kolom peran `owner` dan `superadmin` terkunci aman dengan badge **Bypass Sistem Penuh**.

- [ ] **11.2 Modifikasi Hak Akses Dinamis (Dynamic PBAC Toggle)**
  - [ ] Pilih peran `warehouse`.
  - [ ] Coba centang atau cabut izin tertentu (contoh: cabut `inventory.stocks.view`).
  - [ ] Pastikan tombol simpan berubah status mendeteksi perubahan (*dirty state indicator*).
  - [ ] Klik **Simpan Perubahan Peran**.
  - [ ] Verifikasi bahwa perubahan izin tersimpan ke database `shared_role_permissions` dan cache in-memory RAM backend langsung ter-update seketika.

- [ ] **11.3 Pengujian Dampak Izin Secara Nyata**
  - [ ] Kembalikan izin `warehouse` ke default.
  - [ ] Pastikan izin `inventory.stocks.adjust` dan `inventory.products.view_cost` tidak tercentang untuk peran gudang dan kasir.
  - [ ] Login sebagai `gudang_01` dan pastikan proteksi menu serta data masking berjalan sempurna.

---

## 12. Sistem & Otorisasi: Log Audit Forensik Sistem (`/audit-logs`)
File sumber: [audit-logs/+page.svelte](file:///c:/PROJECT/WEBSITE/erp-retail-modular/frontend/apps/backoffice/src/routes/%28app%29/audit-logs/+page.svelte)

- [ ] **12.1 Rekam Jejak Otomatis via Event Bus**
  - [ ] Buka menu **Sistem & Otorisasi -> Log Audit Sistem**.
  - [ ] Verifikasi bahwa setiap aksi penting (Stock Opname, Approval Mutasi, Pembuatan Staf Baru) otomatis tercatat tanpa perlu input manual.
  - [ ] Verifikasi kolom tabel: Waktu Kejadian, Nama Aktor & Peran, Kategori Aksi, dan Deskripsi Aktivitas.

- [ ] **12.2 Modal Forensik Payload & Filter**
  - [ ] Uji filter berdasarkan nama pengguna, jenis aksi, atau rentang tanggal.
  - [ ] Klik salah satu baris log audit untuk membuka modal detail.
  - [ ] Pastikan modal menampilkan metadata lengkap, alamat IP klien, User Agent, dan JSON payload riil sebelum dan sesudah perubahan.

---

## 13. Fitur Keamanan & Cross-Cutting Sistem
- [ ] **13.1 Mekanisme Login & Sesi JWT**
  - [ ] Coba login dengan kombinasi password salah -> Sistem membalas error kredensial aman tanpa membocorkan eksistensi username.
  - [ ] Login sukses -> Token JWT tersimpan di memory Runes dan LocalStorage.
  - [ ] Klik tombol **Logout** di pojok kanan atas topbar -> Token dibersihkan dan pengguna diarahkan ke `/login`.
  - [ ] Coba akses paksa URL `/inventory/stocks` saat belum login -> *Route Guard* otomatis menolak dan me-redirect ke `/login`.

- [ ] **13.2 Mekanisme Step-Up Authentication**
  - [ ] Uji modal step-up password confirmation saat melakukan aksi berdampak tinggi (misal: approval transfer atau penyesuaian stok).
  - [ ] Password divalidasi ganda di backend sebelum aksi operasional dieksekusi.

- [ ] **13.3 Kepatuhan Lisensi Modul (Zero Overhead Startup)**
  - [ ] Buka file [main.go](file:///c:/PROJECT/WEBSITE/erp-retail-modular/backend/cmd/server/main.go).
  - [ ] Pastikan modul yang tidak diaktifkan pada lisensi tidak ter-mount ke router HTTP maupun event bus server.

---

## Lembar Catatan Hasil Pengujian
| Tanggal Pengujian | Penguji (Tester) | Modul yang Diuji | Status (Pass/Fail) | Catatan Temuan |
| :--- | :--- | :--- | :--- | :--- |
| 25 Sep 2026 | User / Lead QA | Data Master - Kategori Produk | PASS | Seluruh alur (navigasi, tambah induk, subkategori, edit, validasi foreign key, penyederhanaan label & icon) lulus uji. |
| | | | | |
