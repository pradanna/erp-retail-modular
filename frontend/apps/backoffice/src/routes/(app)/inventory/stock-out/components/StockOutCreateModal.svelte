<script lang="ts">
	import type { LocationResponse, ProductResponse } from '@erp/types';
	import { Modal, Button, Input, Select, Select2, type Select2Option, Alert, Badge } from '@erp/ui';
	import { StockOutFormState, stockOutReasonOptions } from '../stock-out.svelte';

	interface Props {
		form: StockOutFormState;
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
		title="Formulir Pengeluaran Barang Keluar (Stock Out)"
		open={form.showModal}
		size="4xl"
		onclose={() => form.closeModal()}
	>
		<div class="space-y-6">
			{#if form.formError}
				<Alert variant="error" title="Periksa Kembali Input Anda">{form.formError}</Alert>
			{/if}

			<!-- 1. Informasi Dokumen & Gudang Asal -->
			<div class="rounded-xl border border-neutral-200 bg-neutral-50/60 p-4.5">
				<div class="mb-3.5 flex items-center justify-between border-b border-neutral-200/80 pb-2.5">
					<span class="text-2xs font-bold tracking-wider text-neutral-500 uppercase">
						1. Informasi Dokumen & Gudang Asal
					</span>
					<span class="text-3xs text-neutral-400">* Wajib diisi</span>
				</div>

				<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
					<div>
						<Select2
							id="form-loc"
							label="Cabang / Gudang Asal"
							size="lg"
							required
							showRequiredAsterisk
							options={locationOptions}
							value={form.locationId}
							onchange={(val: string | number) =>
								form.handleLocationChange(String(val), token, getProductById)}
						/>
					</div>

					<div>
						<Input
							id="form-date"
							type="date"
							label="Tanggal Dokumen / Pengeluaran"
							required
							showRequiredAsterisk
							bind:value={form.movementDate}
						/>
					</div>

					<div>
						<Select
							id="form-reason"
							label="Keperluan / Alasan Pengeluaran"
							required
							showRequiredAsterisk
							bind:value={form.categoryReason}
							options={stockOutReasonOptions}
						/>
					</div>

					<div>
						<Input
							id="form-ref"
							label="No. Memo / Penerima / Divisi Peminta"
							placeholder="Contoh: Budi (Divisi Teknisi Toko)"
							bind:value={form.refNumber}
						/>
					</div>

					<div class="md:col-span-2">
						<Input
							id="form-notes"
							label="Keterangan Pengeluaran (Opsional)"
							placeholder="Misal: Layar TV pecah tidak sengaja di display"
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
							2. Rincian Barang yang Dikeluarkan
						</h3>
						<p class="mt-0.5 text-xs text-neutral-400">
							Pilih produk dan tentukan kuantitas barang fisik yang dikeluarkan dari gudang
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

				<!-- Scanner Cepat Produk / Barcode / IMEI -->
				<div class="rounded-xl border border-neutral-300 bg-white p-3 shadow-2xs">
					<label
						for="fast-scan-stockout"
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
							Scan Barcode Produk / SKU / IMEI Cepat:
						</span>
						<span class="text-3xs font-normal text-neutral-400">Tekan Enter untuk input instan</span
						>
					</label>
					<div class="relative flex items-center">
						<input
							id="fast-scan-stockout"
							type="text"
							bind:value={form.fastScanCode}
							placeholder="Tembakkan scanner barcode produk, SKU, atau nomor seri/IMEI di sini..."
							onkeydown={(e) => {
								if (e.key === 'Enter') {
									e.preventDefault();
									form.handleFastScan(token, products, getProductById);
								}
							}}
							disabled={form.fastScanLoading || !form.locationId}
							class="w-full rounded-lg border border-neutral-300 bg-white py-2 pr-20 pl-3 font-mono text-xs text-neutral-900 placeholder:font-sans placeholder:text-neutral-400 focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden disabled:cursor-not-allowed disabled:bg-neutral-100"
						/>
						<button
							type="button"
							onclick={() => form.handleFastScan(token, products, getProductById)}
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
										value={item.productId}
										placeholder="Ketik untuk mencari produk..."
										searchPlaceholder="Cari berdasarkan nama atau SKU..."
										onchange={(val: string | number) =>
											form.handleProductChange(idx, String(val), token, getProductById)}
									/>
								</div>

								<!-- Kuantitas -->
								<div class="md:col-span-2">
									<Input
										id={`item-qty-${idx}`}
										label="Jumlah Keluar"
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
										label="Alasan Item"
										placeholder="Opsional (misal: Rusak pecah layar)"
										bind:value={item.notes}
									/>
								</div>
							</div>

							<!-- Bagian Pilihan Nomor Seri / IMEI jika produk dilacak nomor serinya -->
							{#if selectedProd?.flag_serial_tracking}
								{@const serialCount = item.selectedSerials.length}
								{@const isComplete = serialCount === Number(item.quantity)}
								{@const unselectedOptions = form.getSerialSelect2Options(item)}
								<div class="mt-3.5 rounded-xl border border-amber-200 bg-amber-50/70 p-4">
									<!-- Header Info & Progress Baris Seri -->
									<div class="mb-3 flex flex-wrap items-center justify-between gap-2">
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
												Pilih Nomor Seri / IMEI Fisik ({selectedProd.name}):
											</span>
											{#if item.loadingSerials}
												<span class="text-2xs animate-pulse text-amber-700">
													(Memuat stok nomor seri...)
												</span>
											{:else}
												<span class="text-2xs text-neutral-600">
													(Tersedia di gudang:
													<strong class="text-neutral-900">{item.availableSerials.length}</strong>
													unit)
												</span>
											{/if}
										</div>

										<div class="flex items-center gap-2">
											<span
												class="rounded-full px-2.5 py-0.5 text-xs font-semibold {isComplete
													? 'border border-emerald-300 bg-emerald-100 text-emerald-800'
													: 'border border-amber-300 bg-amber-100 text-amber-800'}"
											>
												{serialCount} dari {item.quantity} nomor seri dipilih
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

									{#if !item.loadingSerials && item.availableSerials.length === 0}
										<div
											class="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
										>
											Peringatan: Tidak ada unit fisik nomor seri berstatus 'tersedia' untuk produk
											ini di cabang terpilih.
										</div>
									{:else}
										<!-- Dua Metode Input: Scan Barcode / IMEI & Dropdown Select2 -->
										<div class="grid grid-cols-1 items-end gap-3 md:grid-cols-2">
											<!-- 1. Scan Barcode / Ketik IMEI Cepat -->
											<div>
												<label
													for={`barcode-scan-${idx}`}
													class="text-2xs mb-1 block font-semibold text-neutral-700"
												>
													Scan Barcode / Ketik IMEI Fisik:
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
														id={`barcode-scan-${idx}`}
														type="text"
														bind:value={item.barcodeInput}
														placeholder="Tembakkan scanner barcode / ketik SN..."
														onkeydown={(e) => {
															if (e.key === 'Enter') {
																e.preventDefault();
																form.handleAddSerialByBarcode(idx);
															}
														}}
														disabled={isComplete || item.loadingSerials}
														class="w-full rounded-xl border border-neutral-300 bg-white py-2 pr-14 pl-9 font-mono text-xs text-neutral-900 placeholder:font-sans placeholder:text-neutral-400 focus:border-neutral-900 focus:ring-2 focus:ring-neutral-900/10 focus:outline-hidden disabled:cursor-not-allowed disabled:bg-neutral-100"
													/>
													<button
														type="button"
														onclick={() => form.handleAddSerialByBarcode(idx)}
														disabled={isComplete ||
															!item.barcodeInput.trim() ||
															item.loadingSerials}
														class="text-2xs absolute right-1.5 rounded-lg bg-neutral-900 px-2.5 py-1 font-semibold text-white transition-colors hover:bg-neutral-800 disabled:cursor-not-allowed disabled:opacity-40"
													>
														Pilih
													</button>
												</div>
											</div>

											<!-- 2. Pilihan Dropdown Select2 Seri Tersedia -->
											<div>
												<Select2
													id={`item-serial-select-${idx}`}
													label="Atau Pilih dari Daftar Serial Tersedia:"
													size="md"
													options={unselectedOptions}
													bind:value={item.select2Value}
													placeholder={isComplete
														? 'Kuantitas telah terpenuhi'
														: unselectedOptions.length === 0
															? 'Tidak ada serial tersisa'
															: 'Pilih nomor seri...'}
													searchPlaceholder="Cari nomor seri..."
													disabled={isComplete ||
														unselectedOptions.length === 0 ||
														item.loadingSerials}
													onchange={(val: string | number) => {
														if (val) {
															form.handleSelectSerialFromDropdown(idx, String(val));
														}
													}}
												/>
											</div>
										</div>

										<!-- Panduan & Tombol Pilih Otomatis -->
										{#if !isComplete && unselectedOptions.length > 0}
											<div
												class="text-2xs mt-2 flex flex-wrap items-center justify-between gap-1 text-neutral-500"
											>
												<span
													>Tips: Tembakkan barcode scanner gun langsung untuk input berurutan cepat.</span
												>
												<button
													type="button"
													onclick={() => form.handleAutoFillSerials(idx)}
													class="font-medium text-neutral-800 underline hover:text-neutral-950"
												>
													Pilihkan otomatis {Number(item.quantity) - serialCount} unit pertama
												</button>
											</div>
										{/if}

										<!-- Chip Badge Daftar Serial Terpilih -->
										{#if item.selectedSerials.length > 0}
											<div class="mt-3 border-t border-amber-200/80 pt-2.5">
												<span
													class="text-2xs mb-1.5 block font-bold tracking-wider text-neutral-500 uppercase"
												>
													Nomor Seri Fisik Terpilih ({item.selectedSerials.length} unit):
												</span>
												<div
													class="flex max-h-36 flex-wrap gap-1.5 overflow-y-auto rounded-lg border border-amber-100 bg-white/70 p-2"
												>
													{#each item.selectedSerials as sn, sIdx (sn)}
														<span
															class="inline-flex items-center gap-1.5 rounded-lg bg-neutral-900 px-2.5 py-1 font-mono text-xs text-white shadow-2xs"
														>
															<span class="text-2xs text-neutral-400">#{sIdx + 1}</span>
															<span>{sn}</span>
															<button
																type="button"
																onclick={() => form.handleRemoveSerial(idx, sIdx)}
																class="ml-1 rounded p-0.5 text-neutral-400 transition-colors hover:bg-neutral-800 hover:text-rose-400"
																title="Hapus nomor seri ini"
															>
																<svg
																	class="h-3 w-3"
																	viewBox="0 0 24 24"
																	fill="none"
																	stroke="currentColor"
																	stroke-width="2.5"
																	stroke-linecap="round"
																	stroke-linejoin="round"
																>
																	<line x1="18" y1="6" x2="6" y2="18" />
																	<line x1="6" y1="6" x2="18" y2="18" />
																</svg>
															</button>
														</span>
													{/each}
												</div>
											</div>
										{/if}
									{/if}
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
						Simpan & Kurangi Stok
					</Button>
				</div>
			</div>
		{/snippet}
	</Modal>
{/if}
