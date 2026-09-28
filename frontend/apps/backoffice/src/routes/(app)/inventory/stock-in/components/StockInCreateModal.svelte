<script lang="ts">
	import type { LocationResponse, ProductResponse } from '@erp/types';
	import { Modal, Button, Input, Select, Select2, type Select2Option, Alert, Badge } from '@erp/ui';
	import { StockInFormState, stockInReasonOptions } from '../stock-in.svelte';

	interface Props {
		form: StockInFormState;
		locations: LocationResponse[];
		products: ProductResponse[];
		token: string;
		getProductById: (id: string) => ProductResponse | undefined;
		onSuccess: () => Promise<void>;
	}

	let { form, locations, products, token, getProductById, onSuccess }: Props = $props();

	const locationOptions = $derived<Select2Option[]>(
		locations.map((loc) => ({
			value: loc.id,
			label: `${loc.name} (${loc.code})`
		}))
	);

	const productOptions = $derived<Select2Option[]>(
		products.map((p) => ({
			value: p.id,
			label: `${p.name} [SKU: ${p.sku}]`
		}))
	);
</script>

{#if form.showModal}
	<Modal
		title="Formulir Penerimaan Barang Masuk (Stock In)"
		open={form.showModal}
		size="4xl"
		onclose={() => form.closeModal()}
	>
		<div class="space-y-6">
			{#if form.formError}
				<Alert variant="error" title="Periksa Kembali Input Anda">{form.formError}</Alert>
			{/if}

			<!-- 1. Informasi Dokumen & Gudang Penerima -->
			<div class="rounded-xl border border-neutral-200 bg-neutral-50/60 p-4.5">
				<div class="mb-3.5 flex items-center justify-between border-b border-neutral-200/80 pb-2.5">
					<span class="text-2xs font-bold tracking-wider text-neutral-500 uppercase">
						1. Informasi Dokumen & Gudang Penerima
					</span>
					<span class="text-3xs text-neutral-400">* Wajib diisi</span>
				</div>

				<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
					<div>
						<Select2
							id="form-loc"
							label="Cabang / Gudang Penerima"
							size="lg"
							required
							showRequiredAsterisk
							options={locationOptions}
							bind:value={form.locationId}
						/>
					</div>

					<div>
						<Input
							id="form-date"
							type="date"
							label="Tanggal Dokumen / Surat Jalan"
							required
							showRequiredAsterisk
							bind:value={form.movementDate}
						/>
					</div>

					<div>
						<Select
							id="form-reason"
							label="Kategori Penerimaan"
							required
							showRequiredAsterisk
							bind:value={form.categoryReason}
							options={stockInReasonOptions}
						/>
					</div>

					<div>
						<Input
							id="form-ref"
							label="No. Surat Jalan / Memo Vendor"
							placeholder="Contoh: SJ-2026-88912"
							bind:value={form.refNumber}
						/>
					</div>

					<div class="md:col-span-2">
						<Input
							id="form-notes"
							label="Catatan Pengiriman (Opsional)"
							placeholder="Misal: Diterima dari ekspedisi JNE dalam kardus rapi"
							bind:value={form.notes}
						/>
					</div>
				</div>
			</div>

			<!-- 2. Daftar Rincian Barang (Multi-Item) -->
			<div class="space-y-3">
				<div class="flex items-center justify-between">
					<div>
						<h3 class="text-xs font-bold tracking-wider text-neutral-500 uppercase">
							2. Rincian Barang yang Diterima
						</h3>
						<p class="mt-0.5 text-xs text-neutral-400">
							Pilih produk dan tentukan kuantitas barang fisik yang masuk ke gudang
						</p>
					</div>
					<div class="flex items-center gap-2">
						<Badge variant="default">{form.items.length} Baris Produk</Badge>
						<Button variant="outline" size="sm" onclick={() => form.addItemRow()}>
							<svg class="mr-1.5 h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 4.5v15m7.5-7.5h-15"
								/>
							</svg>
							Tambah Baris
						</Button>
					</div>
				</div>

				<!-- Scanner Cepat Produk / Barcode / SKU -->
				<div class="rounded-xl border border-neutral-300 bg-white p-3 shadow-2xs">
					<label
						for="fast-scan-stockin"
						class="text-2xs mb-1.5 flex items-center justify-between font-semibold text-neutral-700"
					>
						<span class="flex items-center gap-1.5">
							<svg
								class="h-4 w-4 text-neutral-600"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="1.75"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 3.75 9.375v-4.5ZM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 0 1-1.125-1.125v-4.5ZM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 13.5 9.375v-4.5Z"
								/>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M6.75 6.75h.75v.75h-.75v-.75ZM6.75 16.5h.75v.75h-.75v-.75ZM16.5 6.75h.75v.75h-.75v-.75ZM13.5 13.5h3.75m0 0v3.75m0-3.75h3.75m-3.75 3.75v3.75m-3.75-3.75h-3.75"
								/>
							</svg>
							Scan Barcode Produk / SKU Cepat:
						</span>
						<span class="text-3xs font-normal text-neutral-400"
							>Tekan Enter untuk input produk otomatis</span
						>
					</label>
					<div class="relative flex items-center">
						<input
							id="fast-scan-stockin"
							type="text"
							bind:value={form.fastScanCode}
							placeholder="Tembakkan scanner barcode produk atau SKU di sini..."
							onkeydown={(e) => {
								if (e.key === 'Enter') {
									e.preventDefault();
									form.handleFastScan(token, products);
								}
							}}
							disabled={form.fastScanLoading || !form.locationId}
							class="w-full rounded-lg border border-neutral-300 bg-white py-2 pr-20 pl-3 font-mono text-xs text-neutral-900 placeholder:font-sans placeholder:text-neutral-400 focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden disabled:cursor-not-allowed disabled:bg-neutral-100"
						/>
						<button
							type="button"
							onclick={() => form.handleFastScan(token, products)}
							disabled={form.fastScanLoading || !form.fastScanCode.trim() || !form.locationId}
							class="text-2xs absolute right-1 rounded-md bg-neutral-900 px-3 py-1 font-semibold text-white transition-colors hover:bg-neutral-800 disabled:cursor-not-allowed disabled:opacity-40"
						>
							{form.fastScanLoading ? 'Memindai...' : 'Pindai'}
						</button>
					</div>
				</div>

				<div class="space-y-3">
					{#each form.items as item, idx (item.id)}
						{@const selectedProd = getProductById(item.productId)}
						<div
							class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs transition-all hover:border-neutral-300"
						>
							<!-- Baris Atas Item: Badge Nomor & Tombol Hapus -->
							<div class="mb-3 flex items-center justify-between border-b border-neutral-100 pb-2">
								<div class="flex items-center gap-2">
									<span
										class="text-3xs flex h-5.5 w-5.5 items-center justify-center rounded-full bg-neutral-900 font-bold text-white"
									>
										{idx + 1}
									</span>
									<span class="text-xs font-bold text-neutral-800">
										Baris Produk #{idx + 1}
									</span>
									{#if selectedProd}
										<span class="font-mono text-xs text-neutral-500">
											[SKU: {selectedProd.sku}]
										</span>
										{#if selectedProd.flag_serial_tracking}
											<Badge variant="purple" size="sm">Wajib Serial / IMEI</Badge>
										{/if}
									{/if}
								</div>

								{#if form.items.length > 1}
									<button
										type="button"
										onclick={() => form.removeItemRow(idx)}
										class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-rose-600 transition-colors hover:bg-rose-50 hover:text-rose-700"
									>
										<svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
											/>
										</svg>
										Hapus Baris
									</button>
								{/if}
							</div>

							<!-- Input Kolom Produk, Jumlah, dan Keterangan -->
							<div class="grid grid-cols-1 items-start gap-4 md:grid-cols-12">
								<!-- Pilih Produk -->
								<div class="md:col-span-6">
									<Select2
										id={`item-prod-${idx}`}
										label="Pilih Produk"
										size="lg"
										required
										showRequiredAsterisk
										options={productOptions}
										bind:value={item.productId}
										placeholder="Ketik untuk mencari produk..."
										searchPlaceholder="Cari berdasarkan nama atau SKU..."
									/>
								</div>

								<!-- Kuantitas -->
								<div class="md:col-span-2">
									<Input
										id={`item-qty-${idx}`}
										label="Jumlah Masuk"
										type="number"
										min={1}
										required
										showRequiredAsterisk
										bind:value={item.quantity}
									/>
								</div>

								<!-- Catatan per Item -->
								<div class="md:col-span-4">
									<Input
										id={`item-notes-${idx}`}
										label="Keterangan Item"
										placeholder="Opsional (misal: Kondisi segel)"
										bind:value={item.notes}
									/>
								</div>
							</div>

							<!-- Bagian Nomor Seri / IMEI jika produk dilacak nomor serinya -->
							{#if selectedProd?.flag_serial_tracking}
								{@const serialsList = form.getSerialsArray(item)}
								{@const serialCount = serialsList.length}
								{@const isComplete = serialCount === Number(item.quantity)}
								<div class="mt-3.5 rounded-xl border border-amber-200 bg-amber-50/70 p-3.5">
									<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
										<div class="flex items-center gap-1.5">
											<svg
												class="h-4 w-4 text-amber-600"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
												/>
											</svg>
											<span class="text-xs font-bold text-amber-900">
												Nomor Seri / IMEI ({selectedProd.name}):
											</span>
										</div>
										<div class="flex items-center gap-2">
											<span
												class="rounded-full px-2.5 py-0.5 text-xs font-semibold {isComplete
													? 'border border-emerald-300 bg-emerald-100 text-emerald-800'
													: 'border border-amber-300 bg-amber-100 text-amber-800'}"
											>
												{serialCount} dari {item.quantity} nomor seri terisi
											</span>
											{#if serialCount > 0}
												<button
													type="button"
													onclick={() => form.handleClearSerials(idx)}
													class="text-2xs text-neutral-500 underline hover:text-rose-600"
												>
													Reset
												</button>
											{/if}
										</div>
									</div>

									<!-- Input Scan Barcode / Ketik IMEI Satu per Satu -->
									<div class="mb-2.5">
										<label
											for={`barcode-scan-stockin-${idx}`}
											class="text-2xs mb-1 block font-semibold text-neutral-700"
										>
											Scan Barcode / Ketik IMEI Fisik (Tekan Enter):
										</label>
										<div class="relative flex items-center">
											<span class="pointer-events-none absolute left-3 text-neutral-400">
												<svg
													class="h-4 w-4"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="1.5"
												>
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 3.75 9.375v-4.5ZM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 0 1-1.125-1.125v-4.5ZM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 13.5 9.375v-4.5Z"
													/>
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														d="M6.75 6.75h.75v.75h-.75v-.75ZM6.75 16.5h.75v.75h-.75v-.75ZM16.5 6.75h.75v.75h-.75v-.75ZM13.5 13.5h3.75m0 0v3.75m0-3.75h3.75m-3.75 3.75v3.75m-3.75-3.75h-3.75"
													/>
												</svg>
											</span>
											<input
												id={`barcode-scan-stockin-${idx}`}
												type="text"
												bind:value={item.barcodeScanInput}
												placeholder="Tembakkan scanner barcode nomor seri / IMEI..."
												onkeydown={(e) => {
													if (e.key === 'Enter') {
														e.preventDefault();
														form.handleAddSerialByBarcode(idx);
													}
												}}
												class="w-full rounded-xl border border-neutral-300 bg-white py-2 pr-20 pl-9 font-mono text-xs text-neutral-900 placeholder:font-sans placeholder:text-neutral-400 focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden"
											/>
											<button
												type="button"
												onclick={() => form.handleAddSerialByBarcode(idx)}
												disabled={!item.barcodeScanInput.trim()}
												class="text-2xs absolute right-1.5 rounded-lg bg-neutral-900 px-3 py-1 font-semibold text-white transition-colors hover:bg-neutral-800 disabled:cursor-not-allowed disabled:opacity-40"
											>
												Tambah
											</button>
										</div>
									</div>

									<!-- Chips / Badges Nomor Seri Terdaftar -->
									{#if serialCount > 0}
										<div class="mb-2.5 flex flex-wrap gap-1.5">
											{#each serialsList as sn (sn)}
												<span
													class="text-3xs inline-flex items-center gap-1 rounded-md border border-amber-300 bg-white px-2 py-0.5 font-mono text-neutral-800 shadow-2xs"
												>
													<span>{sn}</span>
													<button
														type="button"
														onclick={() => form.handleRemoveSerial(idx, sn)}
														class="text-neutral-400 hover:text-rose-600"
														title="Hapus nomor seri"
													>
														&times;
													</button>
												</span>
											{/each}
										</div>
									{/if}

									<!-- Toggle Input Massal (Bulk Textarea) -->
									<div class="border-t border-amber-200/60 pt-1">
										<button
											type="button"
											onclick={() => (item.showBulkTextarea = !item.showBulkTextarea)}
											class="text-3xs font-medium text-amber-900 underline hover:text-amber-950"
										>
											{item.showBulkTextarea
												? 'Tutup Mode Tempel Banyak (Bulk Textarea)'
												: 'Buka Mode Tempel Banyak Sekaligus (Bulk Paste dari Excel / Manifest)'}
										</button>

										{#if item.showBulkTextarea}
											<div class="mt-2">
												<textarea
													id={`item-serials-${idx}`}
													bind:value={item.serialInput}
													rows={3}
													placeholder="Masukkan 1 nomor seri per baris...&#10;Contoh:&#10;SN-ASUS-2026-001&#10;SN-ASUS-2026-002"
													class="w-full rounded-lg border border-amber-300 bg-white p-2.5 font-mono text-xs text-neutral-900 shadow-2xs focus:border-amber-500 focus:ring-2 focus:ring-amber-500/20 focus:outline-hidden"
												></textarea>
											</div>
										{/if}
									</div>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		</div>

		{#snippet footer()}
			{@const totalUnits = form.items.reduce((acc, cur) => acc + (Number(cur.quantity) || 0), 0)}
			<div class="flex w-full items-center justify-between">
				<div class="text-xs text-neutral-500">
					Total: <span class="font-bold text-neutral-900">{totalUnits}</span> unit ({form.items
						.length}
					produk)
				</div>
				<div class="flex items-center gap-3">
					<Button variant="outline" onclick={() => form.closeModal()}>Batal</Button>
					<Button
						variant="primary"
						loading={form.submitting}
						onclick={() => form.submit(token, getProductById, onSuccess)}
					>
						Simpan & Tambah Stok
					</Button>
				</div>
			</div>
		{/snippet}
	</Modal>
{/if}
