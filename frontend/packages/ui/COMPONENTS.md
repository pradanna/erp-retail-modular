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
- **Props:** `label`, `type`, `placeholder`, `value` ($bindable), `error`, `disabled`, `required`, `id`, `name`, `autocomplete`, `leadingIcon` (Snippet), `showPasswordToggle` (boolean)
- **Contoh:**
  ```svelte
  <Input label="Username" bind:value={username} error={usernameError} required>
    {#snippet leadingIcon()}
      <svg class="h-5 w-5 text-neutral-400" viewBox="0 0 24 24" fill="none" stroke="currentColor">...</svg>
    {/snippet}
  </Input>
  <Input label="Password" type="password" bind:value={password} showPasswordToggle />
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

## Aturan Penambahan Komponen Baru

1. Buat file di `components/<NamaKomponen>.svelte`
2. Deklarasikan props dengan `interface Props` + `$props()` (Svelte 5 runes)
3. Gunakan token `@theme` untuk warna (dilarang hardcode hex)
4. Daftarkan di file ini (COMPONENTS.md)
5. Export dari `index.ts`
   <Badge variant="warning">Pending Approval</Badge>

````

### 6. Modal
- **File:** `components/Modal.svelte`
- **Sizes:** `sm` | `md` | `lg` | `xl`
- **Props:** `open` ($bindable), `title`, `size`, `onclose`, `footer`
- **Contoh:**
```svelte
<Modal bind:open={showModal} title="Tambah Kategori">
  <p>Form konten...</p>
  {#snippet footer()}
    <Button onclick={() => showModal = false}>Tutup</Button>
  {/snippet}
</Modal>
````

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

---

## Aturan Penambahan Komponen Baru

1. Buat file di `components/<NamaKomponen>.svelte`
2. Deklarasikan props dengan `interface Props` + `$props()` (Svelte 5 runes)
3. Gunakan token `@theme` untuk warna (dilarang hardcode hex)
4. Gunakan SVG Heroicons outline/solid (dilarang emoji / icon sparkle)
5. Daftarkan di file ini (COMPONENTS.md)
6. Export dari `index.ts`
