<script lang="ts">
	import type { ProductResponse } from '@erp/types';
	import { Modal, Button, Table, Badge } from '@erp/ui';
	import {
		StockInDetailState,
		getReasonLabel,
		formatDate,
		formatDateTime,
		formatRupiah,
		getImageUrl,
		copySerials
	} from '../stock-in.svelte';

	interface Props {
		detail: StockInDetailState;
		getProductById: (id: string) => ProductResponse | undefined;
		getCategoryName: (id?: string) => string;
	}

	let { detail, getProductById, getCategoryName }: Props = $props();
</script>

{#if detail.showModal && detail.selectedMovement}
	{@const mov = detail.selectedMovement}
	{@const totalUnits = (mov.items ?? []).reduce((acc, cur) => acc + (Number(cur.quantity) || 0), 0)}
	<Modal
		title={`Rincian Dokumen: ${mov.movement_number}`}
		open={detail.showModal}
		size="4xl"
		onclose={() => detail.close()}
	>
		{#if detail.loading}
			<div class="flex h-64 items-center justify-center">
				<div class="flex items-center gap-3 text-neutral-500">
					<svg class="h-6 w-6 animate-spin" viewBox="0 0 24 24" fill="none">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						></path>
					</svg>
					<span class="text-sm font-medium">Memuat rincian transaksi dokumen...</span>
				</div>
			</div>
		{:else}
			<div class="space-y-6">
				<!-- 1. Ringkasan Informasi Dokumen -->
				<div class="rounded-xl border border-neutral-200 bg-neutral-50/70 p-4.5">
					<!-- Header Bar Ringkasan -->
					<div
						class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-neutral-200/80 pb-3"
					>
						<div class="flex items-center gap-2.5">
							<span class="font-mono text-base font-bold text-neutral-900">
								{mov.movement_number}
							</span>
							<Badge variant="primary">{getReasonLabel(mov.category_reason)}</Badge>
						</div>
						<div class="flex items-center gap-2 text-xs text-neutral-500">
							<span>Waktu Input:</span>
							<span class="font-medium text-neutral-800">{formatDateTime(mov.created_at)}</span>
						</div>
					</div>

					<!-- Grid Metadata Dokumen -->
					<div class="grid grid-cols-2 gap-4 text-xs sm:grid-cols-4">
						<div class="space-y-1">
							<span class="text-neutral-500">Tanggal Dokumen / SJ:</span>
							<div class="font-semibold text-neutral-900">{formatDate(mov.movement_date)}</div>
						</div>

						<div class="space-y-1">
							<span class="text-neutral-500">Cabang / Gudang Penerima:</span>
							<div class="flex items-center gap-1.5 font-semibold text-neutral-900">
								<svg
									class="h-4 w-4 shrink-0 text-neutral-400"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M15 10.5a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
									/>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1 1 15 0Z"
									/>
								</svg>
								<span>{mov.location_name}</span>
							</div>
						</div>

						<div class="space-y-1">
							<span class="text-neutral-500">No. Surat Jalan / Memo:</span>
							<div class="font-mono font-semibold text-neutral-900">
								{mov.reference_number || '-'}
							</div>
						</div>

						<div class="space-y-1">
							<span class="text-neutral-500">Operator Staf:</span>
							<div class="font-semibold text-neutral-900">{mov.executed_by_name}</div>
						</div>
					</div>

					{#if mov.notes}
						<div
							class="mt-3.5 rounded-lg border border-neutral-200/60 bg-white p-3 text-xs text-neutral-700"
						>
							<span class="font-bold text-neutral-900">Catatan Pengiriman:</span>
							{mov.notes}
						</div>
					{/if}
				</div>

				<!-- 2. Rincian Barang yang Diterima -->
				<div class="space-y-3">
					<div class="flex items-center justify-between">
						<div>
							<h4 class="text-xs font-bold tracking-wider text-neutral-700 uppercase">
								Daftar Produk Diterima
							</h4>
							<p class="mt-0.5 text-xs text-neutral-400">
								Rincian unit produk, foto katalog, dan identifikasi nomor seri / IMEI fisik yang
								masuk ke persediaan.
							</p>
						</div>
						<div class="flex items-center gap-2">
							<Badge variant="default">{mov.items?.length ?? 0} Produk</Badge>
							<Badge variant="indigo">{totalUnits} Total Unit Masuk</Badge>
						</div>
					</div>

					<div class="overflow-x-auto rounded-xl border border-neutral-200 bg-white shadow-2xs">
						<Table>
							<thead>
								<tr
									class="border-b border-neutral-200 bg-neutral-50/80 text-left text-xs font-semibold text-neutral-600"
								>
									<th class="w-12 px-3 py-3 text-center">No</th>
									<th class="w-16 px-3 py-3 text-center">Foto</th>
									<th class="px-4 py-3">Produk & Spesifikasi</th>
									<th class="px-4 py-3 text-center">Jumlah Masuk</th>
									<th class="px-4 py-3">Nomor Seri / IMEI Fisik</th>
									<th class="px-4 py-3 text-right">Harga Normal</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100 text-xs">
								{#each mov.items ?? [] as it, idx (it.id || idx)}
									{@const prod = getProductById(it.product_id)}
									{@const catName = getCategoryName(prod?.category_id)}
									<tr class="transition-colors hover:bg-neutral-50/50">
										<!-- Kolom Nomor -->
										<td class="px-3 py-3.5 text-center font-bold text-neutral-400">
											{idx + 1}
										</td>

										<!-- Kolom Foto Produk -->
										<td class="px-3 py-3.5 text-center">
											<div
												class="relative mx-auto h-12 w-12 shrink-0 overflow-hidden rounded-lg border border-neutral-200 bg-neutral-100 shadow-2xs"
											>
												{#if prod?.primary_image_url}
													<img
														src={getImageUrl(prod.primary_image_url)}
														alt={it.product_name}
														class="h-full w-full object-cover"
														onerror={(e) => {
															const target = e.currentTarget as HTMLImageElement;
															target.style.display = 'none';
															const fallback = target.nextElementSibling as HTMLElement | null;
															if (fallback) {
																fallback.classList.remove('hidden');
																fallback.classList.add('flex');
															}
														}}
													/>
													<div
														class="hidden h-full w-full items-center justify-center text-neutral-400"
													>
														<svg
															class="h-6 w-6"
															viewBox="0 0 24 24"
															fill="none"
															stroke="currentColor"
															stroke-width="1.5"
														>
															<path
																stroke-linecap="round"
																stroke-linejoin="round"
																d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 0 0 1.5-1.5V6a1.5 1.5 0 0 0-1.5-1.5H3.75A1.5 1.5 0 0 0 2.25 6v12a1.5 1.5 0 0 0 1.5 1.5Zm10.5-11.25h.008v.008h-.008V8.25Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
															/>
														</svg>
													</div>
												{:else}
													<div
														class="flex h-full w-full items-center justify-center text-neutral-400"
													>
														<svg
															class="h-6 w-6"
															viewBox="0 0 24 24"
															fill="none"
															stroke="currentColor"
															stroke-width="1.5"
														>
															<path
																stroke-linecap="round"
																stroke-linejoin="round"
																d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 0 0 1.5-1.5V6a1.5 1.5 0 0 0-1.5-1.5H3.75A1.5 1.5 0 0 0 2.25 6v12a1.5 1.5 0 0 0 1.5 1.5Zm10.5-11.25h.008v.008h-.008V8.25Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
															/>
														</svg>
													</div>
												{/if}
											</div>
										</td>

										<!-- Kolom Nama Produk, SKU, Brand, Kategori -->
										<td class="px-4 py-3.5">
											<div class="space-y-1">
												<div class="text-sm font-semibold text-neutral-900">
													{it.product_name}
												</div>
												<div class="text-2xs flex flex-wrap items-center gap-1.5">
													<span
														class="rounded bg-neutral-100 px-1.5 py-0.5 font-mono font-medium text-neutral-700"
													>
														SKU: {it.product_sku}
													</span>
													{#if prod?.brand}
														<span class="rounded bg-neutral-100 px-1.5 py-0.5 text-neutral-600">
															Merek: {prod.brand}
														</span>
													{/if}
													{#if catName}
														<span
															class="rounded bg-primary-50 px-1.5 py-0.5 font-medium text-primary-700"
														>
															{catName}
														</span>
													{/if}
													{#if prod?.flag_serial_tracking}
														<Badge variant="purple" size="sm">Tracking Serial</Badge>
													{/if}
												</div>
												{#if it.notes}
													<div class="text-2xs mt-0.5 text-neutral-500 italic">
														Catatan item: {it.notes}
													</div>
												{/if}
											</div>
										</td>

										<!-- Kolom Jumlah Masuk -->
										<td class="px-4 py-3.5 text-center">
											<span
												class="inline-flex items-center rounded-lg bg-emerald-50 px-2.5 py-1 text-xs font-bold text-emerald-700"
											>
												+{it.quantity}
												{prod?.unit || 'Unit'}
											</span>
										</td>

										<!-- Kolom Nomor Seri / IMEI Fisik -->
										<td class="px-4 py-3.5">
											{#if it.serial_numbers && it.serial_numbers.length > 0}
												<div class="space-y-1.5">
													<div class="flex items-center justify-between gap-2">
														<span class="text-3xs font-semibold text-neutral-500">
															{it.serial_numbers.length} Unit Terdaftar:
														</span>
														<button
															type="button"
															onclick={() => copySerials(it.serial_numbers ?? [])}
															class="text-3xs inline-flex items-center gap-1 font-medium text-primary-600 transition-colors hover:text-primary-800"
															title="Salin seluruh nomor seri ke clipboard"
														>
															<svg
																class="h-3 w-3"
																viewBox="0 0 24 24"
																fill="none"
																stroke="currentColor"
																stroke-width="2"
															>
																<path
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 0 1 1.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9 9 9 0 0 0-9 9v1.5m16.5 4.5h-4.5m4.5 0v-4.5"
																/>
															</svg>
															Salin Seri
														</button>
													</div>
													<div class="flex max-h-24 max-w-sm flex-wrap gap-1 overflow-y-auto pr-1">
														{#each it.serial_numbers as sn, sIdx (sn + sIdx)}
															<span
																class="text-3xs inline-flex items-center rounded-md border border-neutral-200 bg-neutral-50 px-1.5 py-0.5 font-mono text-neutral-800"
															>
																{sn}
															</span>
														{/each}
													</div>
												</div>
											{:else if prod?.flag_serial_tracking}
												<span class="text-2xs text-neutral-400 italic">Belum ada seri terinput</span
												>
											{:else}
												<span class="text-2xs text-neutral-400">Non-Serial</span>
											{/if}
										</td>

										<!-- Kolom Harga Normal Retail -->
										<td class="px-4 py-3.5 text-right font-medium text-neutral-800">
											{formatRupiah(prod?.selling_price)}
										</td>
									</tr>
								{/each}
							</tbody>
						</Table>
					</div>
				</div>
			</div>
		{/if}

		{#snippet footer()}
			{@const totalUnits = (mov.items ?? []).reduce(
				(acc, cur) => acc + (Number(cur.quantity) || 0),
				0
			)}
			<div class="flex w-full items-center justify-between">
				<div class="text-xs text-neutral-500">
					Total: <span class="font-bold text-neutral-900">{totalUnits}</span> unit ({mov.items
						?.length ?? 0} produk)
				</div>
				<Button variant="outline" onclick={() => detail.close()}>Tutup</Button>
			</div>
		{/snippet}
	</Modal>
{/if}
