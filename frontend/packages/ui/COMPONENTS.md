# COMPONENTS.md — Registry Komponen UI (@erp/ui)

> **WAJIB DIBACA** sebelum membuat komponen baru.
> Jika komponen sudah ada, **extend via props/variant** — jangan buat duplikat.

---

## Komponen Tersedia

### 1. Button

- **File:** `components/Button.svelte`
- **Variants:** `primary` | `secondary` | `danger` | `ghost` | `outline`
- **Sizes:** `sm` | `md` | `lg`
- **Props:** `variant`, `size`, `loading`, `disabled`, `type`, `form`, `fullWidth`, `onclick`
- **Contoh:**
  ```svelte
  <Button variant="primary" loading={isSubmitting} onclick={handleSubmit}>Simpan</Button>
  <Button variant="outline" fullWidth>Batal</Button>
  ```

### 2. Input

- **File:** `components/Input.svelte`
- **Props:** `label`, `type`, `placeholder`, `value` ($bindable), `error`, `disabled`, `required`, `id`, `name`, `min`, `max`, `step`, `autocomplete`, `leadingIcon` (Snippet), `showPasswordToggle` (boolean), `thousandSeparator` (boolean), `prefix` (string)
- **Contoh:**
  ```svelte
  <Input label="Username" bind:value={username} error={usernameError} required>
    {#snippet leadingIcon()}
      <svg class="h-5 w-5 text-neutral-400" viewBox="0 0 24 24" fill="none" stroke="currentColor"
        >...</svg
      >
    {/snippet}
  </Input>
  <Input label="Password" type="password" bind:value={password} showPasswordToggle />
  <Input label="Harga Jual" thousandSeparator prefix="Rp" bind:value={sellingPrice} />
  ```

### 3. Card

- **File:** `components/Card.svelte`
- **Padding:** `none` | `sm` | `md` | `lg`
- **Props:** `padding`, `hover`
- **Contoh:**
  ```svelte
  <Card padding="lg" hover>
    <h3>Judul Card</h3>
    <p>Konten card...</p>
  </Card>
  ```

### 4. Alert

- **File:** `components/Alert.svelte`
- **Variants:** `info` | `success` | `warning` | `error`
- **Props:** `variant`, `title`, `dismissible`
- **Contoh:**
  ```svelte
  <Alert variant="error" title="Gagal Login">Username atau password salah.</Alert>
  ```

---

### 5. Badge

- **File:** `components/Badge.svelte`
- **Variants:** `default` | `primary` | `success` | `warning` | `danger` | `info` | `purple` | `indigo` | `cyan`
- **Sizes:** `sm` | `md`
- **Props:** `variant`, `size`, `children`
- **Contoh:**
  ```svelte
  <Badge variant="primary">Admin</Badge>
  <Badge variant="warning">Pending</Badge>
  <Badge variant="indigo"><span class="font-mono font-semibold">SKU-123</span></Badge>
  <Badge variant="purple">Serial / IMEI</Badge>
  <Badge variant="cyan">PPN</Badge>
  ```

### 6. Modal

- **File:** `components/Modal.svelte`
- **Sizes:** `sm` | `md` | `lg` | `xl` | `2xl` | `3xl` | `4xl` | `5xl`
- **Props:** `open` ($bindable), `title`, `size`, `onclose`, `footer`
- **Contoh:**

```svelte
<Modal bind:open={showModal} title="Tambah Kategori">
  <p>Form konten...</p>
  {#snippet footer()}
    <Button onclick={() => (showModal = false)}>Tutup</Button>
  {/snippet}
</Modal>
```

### 7. Select

- **File:** `components/Select.svelte`
- **Props:** `label`, `value` ($bindable), `options` ({value, label}[]), `placeholder`, `error`, `disabled`, `required`, `id`, `name`
- **Contoh:**
  ```svelte
  <Select label="Tipe Lokasi" bind:value={locationType} options={typeOptions} />
  ```

### 8. Table

- **File:** `components/Table.svelte`
- **Props:** `empty`, `emptyMessage`
- **Contoh:**
  ```svelte
  <Table empty={items.length === 0} emptyMessage="Tidak ada data produk.">
    <thead>
      <tr class="border-b border-neutral-200 bg-neutral-50 text-xs font-semibold text-neutral-600">
        <th class="px-4 py-3">SKU</th>
        <th class="px-4 py-3">Nama Produk</th>
      </tr>
    </thead>
    <tbody class="divide-y divide-neutral-200">
      {#each items as item}
        <tr class="hover:bg-neutral-50">
          <td class="px-4 py-3">{item.sku}</td>
          <td class="px-4 py-3">{item.name}</td>
        </tr>
      {/each}
    </tbody>
  </Table>
  ```

### 9. Pagination

- **File:** `components/Pagination.svelte`
- **Props:** `page`, `totalPages`, `totalItems`, `limit`, `onPageChange`
- **Contoh:**
  ```svelte
  <Pagination {page} {totalPages} {totalItems} {limit} onPageChange={handlePageChange} />
  ```

### 10. SearchInput

- **File:** `components/SearchInput.svelte`
- **Props:** `value` ($bindable), `placeholder`, `onsearch`
- **Contoh:**
  ```svelte
  <SearchInput
    bind:value={searchQuery}
    placeholder="Cari SKU atau nama..."
    onsearch={handleSearch}
  />
  ```

### 11. StepUpModal

- **File:** `components/StepUpModal.svelte`
- **Props:** `open` ($bindable), `title`, `description`, `actionLabel`, `onconfirm`, `oncancel`
- **Contoh:**
  ```svelte
  <StepUpModal
    bind:open={showStepUp}
    title="Otorisasi Approval Mutasi Stok"
    actionLabel="Setujui Transfer"
    onconfirm={handleVerifyAndApprove}
  />
  ```

### 12. Checkbox

- **File:** `components/Checkbox.svelte`
- **Props:** `label`, `checked` ($bindable), `disabled`, `id`, `name`
- **Contoh:**
  ```svelte
  <Checkbox label="Ingat saya" bind:checked={rememberMe} />
  ```

### 13. RichTextEditor (WYSIWYG)

- **File:** `components/RichTextEditor.svelte`
- **Props:** `id`, `label`, `value` ($bindable, string HTML), `placeholder`, `minHeight`, `disabled`, `error`
- **Fitur:** Toolbar format teks (Bold, Italic, Underline, Strike), Headings (H2, H3, P), Lists (Bullet, Numbered), Quote, Link, Horizontal Line, Clear Format, Undo/Redo, serta toggle mode kode sumber HTML (`<>`).
- **Contoh:**
  ```svelte
  <RichTextEditor
    label="Deskripsi Produk"
    bind:value={formDescription}
    placeholder="Tuliskan spesifikasi produk..."
    minHeight="200px"
  />
  ```

### 14. Select2 (Searchable Select / Combobox)

- **File:** `components/Select2.svelte`
- **Props:** `id`, `label`, `value` ($bindable), `options` (`SelectOption[]` atau `Select2Option[]`), `placeholder`, `searchPlaceholder`, `error`, `disabled`, `required`, `showRequiredAsterisk`, `clearable`, `name`, `class`, `placement` (`'auto'` | `'bottom'` | `'top'`, default: `'auto'`), `onchange`
- **Fitur:** Smart Floating Positioning (bebas dari *overflow clipping* modal/dialog), Auto-Dropup jika ruang bawah sempit (< 180px), pelacakan scroll/resize otomatis, pencarian teks real-time di dalam dropdown, navigasi keyboard (Arrow Up/Down, Enter, Escape), tombol pembersih (clearable), dukungan label hierarkis / subtext, filter toolbar tabel (mendukung opsi kosong seperti "Semua Kategori"), dan click-outside detector.
- **Contoh Form:**
  ```svelte
  <Select2
    label="Kategori Produk"
    options={categoryOptions}
    bind:value={selectedCategoryId}
    placeholder="Pilih atau cari kategori..."
    searchPlaceholder="Ketik nama kategori..."
  />
  ```
- **Contoh Filtering Tabel:**
  ```svelte
  <Select2
    options={categoryFilterOptions}
    bind:value={selectedCategoryFilter}
    placeholder="Semua Kategori"
    searchPlaceholder="Cari kategori..."
    clearable={true}
    onchange={handleCategoryFilterChange}
  />
  ```

### 15. Barcode (Code-128 SVG Generator)

- **File:** `components/Barcode.svelte`
- **Props:** `value` (string), `height` (number, default: 48), `moduleWidth` (number, default: 2), `showText` (boolean, default: true), `class` (string)
- **Fitur:** Generator SVG barcode 1D standar Code 128-B berbasis vektor murni tanpa library eksternal. Tajam pada cetakan stiker printer thermal POS maupun kertas biasa.
- **Contoh:**
  ```svelte
  <Barcode value="8806098765432" height={48} showText={true} />
  ```

### 16. Toast & ToastContainer (Floating Notification System)

- **File:** `components/Toast.svelte`, `components/ToastContainer.svelte`, `components/toast.svelte.ts`
- **Tipe Notifikasi:** `success` | `error` | `warning` | `info`
- **Fitur:** Notifikasi melayang global berbasis Svelte 5 runes (`$state`). Mendukung auto-dismiss dengan timer kustom (default: 4 detik), tombol tutup manual (X), aksen garis tepi warna monokrom Obsidian, dan transisi animasi halus (`fly`).
- **Pemasangan di Layout (Cukup Sekali):**
  ```svelte
  <script lang="ts">
    import { ToastContainer } from '@erp/ui';
  </script>

  <ToastContainer />
  ```
- **Pemanggilan dari Komponen / Halaman Apapun:**
  ```svelte
  <script lang="ts">
    import { toast } from '@erp/ui';

    function handleSave() {
      toast.success('Kebijakan garansi berhasil disimpan!');
      // atau toast.error('Gagal memproses data.');
      // atau toast.info('Sesi diperbarui.');
      // atau toast.warning('Stok menipis.');
    }
  </script>
  ```

### 17. MapPicker (Interactive OpenStreetMap / Leaflet Map & GPS Picker)

- **File:** `components/MapPicker.svelte`
- **Props:** `latitude` ($bindable, number | null), `longitude` ($bindable, number | null), `label` (string), `placeholder` (string), `readonly` (boolean), `height` (string, default: '280px'), `zoom` (number, default: 13), `class` (string)
- **Fitur Kunci:**
  - Peta interaktif berbasis OpenStreetMap dan Leaflet (zero API key, 100% gratis, aman dari limit tagihan).
  - Pin penanda lokasi kustom Obsidian yang dapat digeser (_draggable_) atau diletakkan dengan sekali klik pada peta.
  - Dua input numerik Latitude & Longitude dengan sinkronisasi dua arah real-time.
  - Pencarian lokasi/kota terintegrasi via OpenStreetMap Nominatim Geocoding API.
  - Tombol **Lokasi Saya** otomatis memanfaatkan GPS peramban (_HTML5 Geolocation_).
  - Tautan cepat ke Google Maps untuk verifikasi citra satelit eksternal.
- **Contoh:**
  ```svelte
  <MapPicker
    bind:latitude={formLatitude}
    bind:longitude={formLongitude}
    label="Koordinat Cabang & Peta"
  />
  ```

### 18. ActionMenu (Ellipsis Vertical Dropdown Menu)

- **File:** `components/ActionMenu.svelte`
- **Props:** `items` (`ActionMenuItem[]`), `align` (`'right'` | `'left'`, default: `'right'`), `title` (string), `class` (string), `trigger` (Snippet), `children` (Snippet)
- **Tipe Item:** `ActionMenuItem { id?, label, iconSvg?, onclick?, variant?: 'default' | 'danger' | 'primary', disabled?, divider? }`
- **Fitur Kunci:**
  - Tombol aksi titik tiga vertikal (`...` / Heroicons EllipsisVertical) yang ringkas dan elegan untuk baris tabel.
  - Smart Floating Placement (`position: fixed`) dengan koordinat dinamis layar sehingga **tidak pernah terpotong** oleh `overflow-x` tabel.
  - Auto-dropup otomatis jika tombol berada dekat bagian bawah layar.
  - Menutup otomatis saat klik-luar (*click-outside*), tombol Escape, atau saat pengguna memilih item menu.
  - Varian warna item (standar netral, `primary`, dan `danger` merah untuk aksi hapus/batal).
- **Contoh Penggunaan di Tabel:**
  ```svelte
  <ActionMenu
    items={[
      {
        label: 'Lihat Detail',
        iconSvg: 'M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178zM15 12a3 3 0 11-6 0 3 3 0 016 0z',
        onclick: () => openDetail(item)
      },
      {
        label: 'Edit Data',
        iconSvg: 'M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10',
        onclick: () => openEdit(item)
      },
      {
        label: 'Hapus',
        variant: 'danger',
        divider: true,
        iconSvg: 'M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0',
        onclick: () => openDelete(item)
      }
    ]}
  />
  ```

---

## Aturan Penambahan Komponen Baru

1. Buat file di `components/<NamaKomponen>.svelte`
2. Deklarasikan props dengan `interface Props` + `$props()` (Svelte 5 runes)
3. Gunakan token `@theme` untuk warna (dilarang hardcode hex)
4. Gunakan SVG Heroicons outline/solid (dilarang emoji / icon sparkle)
5. Daftarkan di file ini (COMPONENTS.md)
6. Export dari `index.ts`
