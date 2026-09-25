<script lang="ts">
	/**
	 * Halaman Manajemen Barcode Produk — Data Master (Gen-E Enterprise).
	 *
	 * Fitur:
	 * - Pemilihan produk aktif & sinkronisasi URL query parameter (?product_id=...)
	 * - Tabel daftar barcode produk (Primary vs Sekunder)
	 * - Modal pendaftaran barcode baru (EAN-13, Code-128, dsb.)
	 * - Hapus barcode sekunder/lama
	 * - Simulator Laser Barcode Scanner (Tes pembacaan instan API lookup)
	 */
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listProducts,
		listBarcodes,
		addBarcode,
		deleteBarcode,
		lookupBarcode
	} from '@erp/api-client';
	import type { ProductResponse, BarcodeResponse, ProductLookupResponse } from '@erp/types';
	import { ApiError } from '@erp/types';
	import {
		Button,
		Input,
		Table,
		Modal,
		Alert,
		Badge,
		Checkbox,
		SearchInput,
		Barcode,
		ActionMenu,
		type ActionMenuItem,
		toast
	} from '@erp/ui';

	// State produk & barcode
	let products = $state<ProductResponse[]>([]);
	let selectedProductId = $state<string>('');
	let barcodes = $state<BarcodeResponse[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let productSearchQuery = $state('');

	// Simulator Laser Scanner State
	let scanInput = $state('');
	let scanLoading = $state(false);
	let scanResult = $state<ProductLookupResponse | null>(null);
	let scanError = $state<string | null>(null);

	// Modal Tambah Barcode
	let showAddModal = $state(false);
	let formBarcode = $state('');
	let formIsPrimary = $state(false);
	let addLoading = $state(false);
	let addError = $state<string | null>(null);

	// Modal Hapus Barcode
	let showDeleteModal = $state(false);
	let targetBarcode = $state<BarcodeResponse | null>(null);
	let deleteLoading = $state(false);
	let deleteError = $state<string | null>(null);

	// Modal Cetak Label Barcode
	let showPrintModal = $state(false);
	let printTargetBarcode = $state<BarcodeResponse | null>(null);
	let printCopies = $state(1);
	let printFormat = $state<'single' | 'sheet'>('single');
	let printIncludeStoreName = $state(true);
	let printIncludeProductName = $state(true);
	let printIncludePrice = $state(true);

	// Filter produk berdasarkan input pencarian (nama, SKU, merek)
	const filteredProducts = $derived(
		products.filter((p) => {
			if (!productSearchQuery.trim()) return true;
			const q = productSearchQuery.toLowerCase();
			return (
				p.name.toLowerCase().includes(q) ||
				p.sku.toLowerCase().includes(q) ||
				(p.brand && p.brand.toLowerCase().includes(q))
			);
		})
	);

	// Detail produk yang sedang dipilih
	const currentProduct = $derived(products.find((p) => p.id === selectedProductId) ?? null);

	// Barcode yang cocok dari hasil scan terakhir untuk di-highlight di tabel
	const scannedBarcodeMatch = $derived(scanResult?.scanned_barcode?.barcode ?? null);

	async function loadProductList() {
		const token = getToken();
		if (!token) return;

		try {
			const res = await listProducts(token, { limit: 100 });
			products = res.data ?? [];

			// Periksa apakah ada param product_id di URL
			const urlProductId = page.url.searchParams.get('product_id');
			if (urlProductId && products.some((p) => p.id === urlProductId)) {
				selectedProductId = urlProductId;
				await loadProductBarcodes(urlProductId);
			} else if (products.length > 0 && !selectedProductId) {
				selectedProductId = products[0].id;
				await loadProductBarcodes(products[0].id);
			}
		} catch (err: unknown) {
			console.error('Gagal memuat list produk:', err);
		}
	}

	async function loadProductBarcodes(productId: string) {
		if (!productId) {
			barcodes = [];
			return;
		}

		const token = getToken();
		if (!token) return;

		loading = true;
		error = null;
		try {
			barcodes = await listBarcodes(token, productId);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else {
				error = 'Gagal memuat data barcode produk';
			}
		} finally {
			loading = false;
		}
	}

	async function selectProduct(productId: string, forceReload = false) {
		if (selectedProductId === productId && !forceReload) return;
		selectedProductId = productId;
		await goto(`/master/barcodes?product_id=${productId}`, { replaceState: true });
		await loadProductBarcodes(productId);
	}

	function openAddModal() {
		if (!selectedProductId) return;
		formBarcode = '';
		formIsPrimary = barcodes.length === 0; // Otomatis jadikan primary jika belum punya barcode sama sekali
		addError = null;
		showAddModal = true;
	}

	async function handleSubmitAdd(e: SubmitEvent) {
		e.preventDefault();
		const token = getToken();
		if (!token || !selectedProductId) return;

		if (!formBarcode.trim()) {
			addError = 'Kode barcode wajib diisi';
			return;
		}

		addLoading = true;
		addError = null;
		try {
			await addBarcode(token, selectedProductId, {
				barcode: formBarcode.trim(),
				is_primary: formIsPrimary
			});

			toast.success(`Barcode "${formBarcode.trim()}" berhasil ditambahkan.`);
			showAddModal = false;
			await loadProductBarcodes(selectedProductId);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				addError = err.message;
			} else {
				addError = 'Gagal menambahkan barcode baru';
			}
		} finally {
			addLoading = false;
		}
	}

	function confirmDelete(b: BarcodeResponse) {
		targetBarcode = b;
		deleteError = null;
		showDeleteModal = true;
	}

	async function handleDelete() {
		const token = getToken();
		if (!token || !selectedProductId || !targetBarcode) return;

		deleteLoading = true;
		deleteError = null;
		try {
			await deleteBarcode(token, selectedProductId, targetBarcode.id);
			toast.success(`Barcode "${targetBarcode.barcode}" berhasil dihapus.`);
			showDeleteModal = false;
			await loadProductBarcodes(selectedProductId);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				deleteError = err.message;
			} else {
				deleteError = 'Gagal menghapus barcode';
			}
		} finally {
			deleteLoading = false;
		}
	}

	function openPrintModal(b: BarcodeResponse) {
		printTargetBarcode = b;
		printCopies = 1;
		showPrintModal = true;
	}

	function executePrint() {
		if (!printTargetBarcode || !currentProduct) return;

		const copies = Math.max(1, Math.min(500, Number(printCopies) || 1));
		const storeName = 'GEN-E RETAIL';
		const productName = currentProduct.name;
		const priceFormatted = new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			maximumFractionDigits: 0
		}).format(currentProduct.selling_price);

		const printWindow = window.open('', '_blank', 'width=800,height=600');
		if (!printWindow) {
			alert('Harap izinkan pop-up browser untuk mencetak barcode.');
			return;
		}

		const previewSvg = document.querySelector('#barcode-preview-container svg')?.outerHTML || '';
		const isThermal = printFormat === 'single';

		let labelsHtml = '';
		for (let i = 0; i < copies; i++) {
			labelsHtml += `
				<div class="label-card">
					${printIncludeStoreName ? `<div class="store-name">${storeName}</div>` : ''}
					${printIncludeProductName ? `<div class="product-name">${productName}</div>` : ''}
					<div class="barcode-wrapper">${previewSvg}</div>
					${printIncludePrice ? `<div class="price">${priceFormatted}</div>` : ''}
				</div>
			`;
		}

		printWindow.document.write(`
			<!DOCTYPE html>
			<html lang="id">
			<head>
				<meta charset="UTF-8">
				<title>Cetak Barcode - ${productName}</title>
				<style>
					* { box-sizing: border-box; margin: 0; padding: 0; }
					body {
						font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
						background: white;
						color: #000;
						padding: ${isThermal ? '0' : '8mm'};
					}
					@page {
						size: ${isThermal ? '50mm 35mm' : 'A4 portrait'};
						margin: ${isThermal ? '2mm' : '8mm'};
					}
					.container {
						display: flex;
						flex-wrap: wrap;
						gap: ${isThermal ? '0' : '4mm'};
						justify-content: ${isThermal ? 'center' : 'flex-start'};
					}
					.label-card {
						width: 48mm;
						height: 32mm;
						padding: 2mm 3mm;
						border: ${isThermal ? 'none' : '1px dashed #d1d5db'};
						page-break-inside: avoid;
						${isThermal ? 'page-break-after: always; break-after: page;' : ''}
						display: flex;
						flex-direction: column;
						align-items: center;
						justify-content: center;
						text-align: center;
						overflow: hidden;
					}
					.store-name {
						font-size: 8px;
						font-weight: 800;
						letter-spacing: 1.5px;
						text-transform: uppercase;
						color: #404040;
						margin-bottom: 0.8mm;
					}
					.product-name {
						font-size: 9px;
						font-weight: 700;
						line-height: 1.15;
						max-height: 20px;
						overflow: hidden;
						margin-bottom: 0.8mm;
					}
					.barcode-wrapper {
						display: flex;
						justify-content: center;
						align-items: center;
						width: 100%;
						margin: 0.5mm 0;
					}
					.barcode-wrapper svg {
						max-width: 42mm;
						max-height: 14mm;
						height: auto;
					}
					.price {
						font-size: 10px;
						font-weight: 800;
						margin-top: 0.8mm;
					}
					@media screen {
						body { background: #f3f4f6; padding: 20px; }
						.label-card { background: white; margin-bottom: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
					}
				</style>
			</head>
			<body>
				<div class="container">
					${labelsHtml}
				</div>
				<script>
					window.onload = function() {
						window.focus();
						window.print();
						window.onafterprint = function() { window.close(); };
					};
				${'<' + '/script>'}
			</body>
			</html>
		`);

		printWindow.document.close();
		showPrintModal = false;
	}

	async function handleLaserScan(e: SubmitEvent) {
		e.preventDefault();
		const token = getToken();
		if (!token) return;

		const cleanCode = scanInput.trim();
		if (!cleanCode) return;

		scanLoading = true;
		scanError = null;
		scanResult = null;

		try {
			const res = await lookupBarcode(token, cleanCode);
			scanResult = res;

			// Otomatis sinkronkan dan pilih produk yang cocok dengan barcode yang discan
			if (res.product && res.product.id) {
				// Pastikan produk ada di daftar produk lokal
				if (!products.some((p) => p.id === res.product.id)) {
					products = [res.product, ...products];
				}

				// Jika filter pencarian sedang menyaring dan tidak mencakup produk ini, bersihkan filter
				if (productSearchQuery.trim()) {
					const q = productSearchQuery.toLowerCase();
					const match =
						res.product.name.toLowerCase().includes(q) ||
						res.product.sku.toLowerCase().includes(q) ||
						(res.product.brand && res.product.brand.toLowerCase().includes(q));
					if (!match) {
						productSearchQuery = '';
					}
				}

				await selectProduct(res.product.id, true);

				// Scroll baris produk pada tabel katalog agar tampak di layar
				setTimeout(() => {
					const rowEl = document.getElementById(`product-row-${res.product.id}`);
					if (rowEl) {
						rowEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
					}
				}, 60);
			}
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				if (err.status === 404) {
					scanError = `Barcode "${cleanCode}" tidak terdaftar di sistem.`;
				} else {
					scanError = err.message || 'Terjadi kesalahan saat memverifikasi barcode.';
				}
			} else {
				scanError = 'Terjadi kesalahan saat memverifikasi barcode.';
			}
		} finally {
			scanLoading = false;
		}
	}

	onMount(async () => {
		await loadProductList();
	});
</script>

<svelte:head>
	<title>Barcode Produk — Data Master Gen-E</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header Halaman -->
	<div>
		<h1 class="text-2xl font-bold tracking-tight text-neutral-900">Barcode Produk</h1>
		<p class="mt-1 text-sm text-neutral-500">
			Kelola relasi nomor kode batang (EAN, UPC, Code-128) per produk dan uji pemindaian kasir.
		</p>
	</div>

	<!-- Notifikasi Feedback Global -->
	{#if error}
		<Alert variant="error" dismissible title="Terjadi Kesalahan">
			{error}
		</Alert>
	{/if}

	<!-- Bagian 1: Simulator Laser Scanner Kasir -->
	<div class="rounded-xl border border-primary-200/80 bg-primary-50/40 p-5 shadow-xs">
		<div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
			<div class="max-w-xl">
				<div class="flex items-center gap-2 text-primary-700">
					<svg
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z"
						/>
					</svg>
					<h2 class="text-sm font-semibold">Simulator Laser Scanner Kasir</h2>
				</div>
				<p class="mt-1 text-xs text-neutral-600">
					Tes cepat kehandalan lookup barcode seperti perangkat keras scanner fisik di meja kasir.
				</p>
			</div>

			<form onsubmit={handleLaserScan} class="flex w-full items-center gap-2 sm:w-auto">
				<div class="w-64 sm:w-80">
					<Input placeholder="Scan atau ketik kode barcode..." bind:value={scanInput} />
				</div>
				<Button variant="primary" type="submit" loading={scanLoading}>Lookup</Button>
			</form>
		</div>

		<!-- Hasil Pemindaian Simulator -->
		{#if scanError}
			<div class="mt-4">
				<Alert variant="error" title="Tidak Ditemukan">{scanError}</Alert>
			</div>
		{/if}

		{#if scanResult}
			<div
				class="mt-4 flex flex-col gap-3 rounded-lg border border-emerald-200 bg-emerald-50/80 p-4 sm:flex-row sm:items-center sm:justify-between"
			>
				<div class="flex items-center gap-3">
					<span
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-600 text-white"
					>
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
						</svg>
					</span>
					<div>
						<span class="text-xs font-semibold tracking-wider text-emerald-800 uppercase"
							>Barcode Cocok Terverifikasi</span
						>
						<h3 class="text-sm font-bold text-neutral-900">{scanResult.product.name}</h3>
						<div class="mt-0.5 flex items-center gap-2 text-xs text-neutral-600">
							<span>SKU: <strong class="font-mono">{scanResult.product.sku}</strong></span>
							<span>•</span>
							<span
								>Barcode: <strong class="font-mono">{scanResult.scanned_barcode.barcode}</strong
								></span
							>
							{#if scanResult.scanned_barcode.is_primary}
								<Badge variant="primary" size="sm">Barcode Utama</Badge>
							{/if}
						</div>
					</div>
				</div>

				<div class="text-right">
					<span class="text-xs text-neutral-500">Harga Jual</span>
					<div class="text-base font-bold text-neutral-900">
						{new Intl.NumberFormat('id-ID', {
							style: 'currency',
							currency: 'IDR',
							maximumFractionDigits: 0
						}).format(scanResult.product.selling_price)}
					</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- Bagian 2: Master-Detail Katalog Produk (Kiri) & Daftar Barcode (Kanan) -->
	<div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
		<!-- Sisi Kiri: Pencarian & Tabel Katalog Produk (Master) -->
		<div class="space-y-4 lg:col-span-5">
			<div class="overflow-hidden rounded-xl border border-neutral-200/80 bg-white shadow-xs">
				<!-- Header Panel Katalog -->
				<div class="border-b border-neutral-200/80 bg-white px-4 py-3.5">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<h2 class="text-sm font-semibold text-neutral-900">Katalog Produk</h2>
							<span
								class="inline-flex items-center rounded-full bg-neutral-100 px-2 py-0.5 text-[11px] font-medium text-neutral-600"
							>
								{filteredProducts.length}
							</span>
						</div>
						{#if productSearchQuery}
							<span class="text-[11px] text-neutral-400">Filter aktif</span>
						{/if}
					</div>

					<!-- Input Pencarian Produk -->
					<div class="mt-3">
						<SearchInput
							bind:value={productSearchQuery}
							placeholder="Cari nama, SKU, merek produk..."
							debounceMs={150}
							class="w-full max-w-none"
						/>
					</div>
				</div>

				<!-- Tabel Produk yang Bisa Diklik -->
				<div class="max-h-[460px] overflow-y-auto">
					<table class="w-full text-left">
						<thead
							class="sticky top-0 z-10 border-b border-neutral-200 bg-neutral-50/95 text-[11px] font-bold tracking-wider text-neutral-500 uppercase backdrop-blur-xs"
						>
							<tr>
								<th class="px-3.5 py-2.5">Produk</th>
								<th class="px-2.5 py-2.5">SKU</th>
								<th class="px-2.5 py-2.5 text-center">Tipe</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-100">
							{#each filteredProducts as p (p.id)}
								<tr
									id="product-row-{p.id}"
									role="button"
									tabindex="0"
									onclick={() => selectProduct(p.id)}
									onkeydown={(e) => {
										if (e.key === 'Enter' || e.key === ' ') {
											e.preventDefault();
											selectProduct(p.id);
										}
									}}
									class="group cursor-pointer transition-colors focus:bg-primary-50/50 focus:outline-hidden {p.id ===
									selectedProductId
										? 'border-l-4 border-l-primary-600 bg-primary-50/80 font-semibold'
										: 'border-l-4 border-l-transparent hover:bg-neutral-50/70'}"
								>
									<td class="px-3.5 py-2.5">
										<div class="flex items-center gap-1.5">
											{#if p.id === selectedProductId}
												<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-primary-600"></span>
											{/if}
											<div
												class="max-w-[170px] truncate text-xs font-medium text-neutral-900 sm:max-w-[210px]"
											>
												{p.name}
											</div>
										</div>
										<div class="text-[11px] text-neutral-400">
											{p.brand || 'Tanpa Merek'}
										</div>
									</td>
									<td class="px-2.5 py-2.5 whitespace-nowrap">
										<Badge variant="indigo" size="sm">
											<span class="font-mono text-[10px]">{p.sku}</span>
										</Badge>
									</td>
									<td class="px-2.5 py-2.5 text-center whitespace-nowrap">
										{#if p.flag_serial_tracking}
											<Badge variant="purple" size="sm">IMEI</Badge>
										{:else}
											<span class="text-[11px] text-neutral-400">{p.unit}</span>
										{/if}
									</td>
								</tr>
							{:else}
								<tr>
									<td colspan="3" class="p-8 text-center">
										<p class="text-xs font-medium text-neutral-600">Produk tidak ditemukan</p>
										{#if productSearchQuery}
											<p class="mt-0.5 text-[11px] text-neutral-400">
												Tidak ada produk cocok dengan "{productSearchQuery}"
											</p>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Info Ringkas Produk Terpilih di Bawah Tabel -->
				{#if currentProduct}
					<div class="border-t border-neutral-200/80 bg-neutral-50/50 p-4">
						<div class="flex items-start justify-between gap-3">
							<div>
								<span class="text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
									Produk Terpilih
								</span>
								<h3 class="text-xs font-bold text-neutral-900">{currentProduct.name}</h3>
								<div class="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-neutral-500">
									<span
										>SKU: <strong class="font-mono text-neutral-800">{currentProduct.sku}</strong
										></span
									>
									<span>•</span>
									<span
										>Merek: <strong class="text-neutral-800">{currentProduct.brand || '-'}</strong
										></span
									>
									<span>•</span>
									<span
										>Satuan: <strong class="text-neutral-800">{currentProduct.unit}</strong></span
									>
								</div>
							</div>
							{#if currentProduct.flag_serial_tracking}
								<Badge variant="purple" size="sm">Serial / IMEI</Badge>
							{/if}
						</div>
					</div>
				{/if}
			</div>
		</div>

		<!-- Sisi Kanan: Tabel Barcode Produk Terpilih (Detail) -->
		<div class="lg:col-span-7">
			<div class="rounded-xl border border-neutral-200/80 bg-white shadow-xs">
				<div class="border-b border-neutral-200/80 px-5 py-4">
					<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
						<div>
							<div class="flex items-center gap-2">
								<h2 class="text-sm font-semibold text-neutral-900">Daftar Barcode Terdaftar</h2>
								<span
									class="inline-flex items-center rounded-full bg-neutral-100 px-2 py-0.5 text-[11px] font-medium text-neutral-600"
								>
									{barcodes.length} barcode aktif
								</span>
							</div>
							{#if currentProduct}
								<p class="mt-0.5 text-xs text-neutral-500">
									Produk: <span class="font-medium text-neutral-800">{currentProduct.name}</span>
								</p>
							{/if}
						</div>

						<!-- Tombol Tambah Barcode dipindahkan ke panel ini -->
						<div>
							<Button
								variant="primary"
								size="sm"
								onclick={openAddModal}
								disabled={!selectedProductId}
							>
								<svg
									class="mr-1.5 h-3.5 w-3.5"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
								>
									<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
								</svg>
								Tambah Barcode
							</Button>
						</div>
					</div>
				</div>

				<Table
					empty={!loading && barcodes.length === 0}
					emptyTitle="Belum Ada Barcode"
					emptyMessage={selectedProductId
						? 'Produk ini belum memiliki nomor barcode terdaftar. Klik "+ Tambah Barcode" di atas.'
						: 'Pilih salah satu produk di katalog kiri terlebih dahulu.'}
					{loading}
				>
					<thead>
						<tr
							class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-bold tracking-wider text-neutral-500 uppercase"
						>
							<th class="px-4 py-3 text-left">Nomor Barcode</th>
							<th class="px-4 py-3 text-center">Peran Barcode</th>
							<th class="px-4 py-3 text-left">Waktu Didaftarkan</th>
							<th class="px-4 py-3 text-right">Aksi</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100">
						{#each barcodes as b (b.id)}
							<tr
								class="transition-colors {b.barcode === scannedBarcodeMatch
									? 'border-l-4 border-l-emerald-600 bg-emerald-50/70'
									: 'border-l-4 border-l-transparent hover:bg-neutral-50/70'}"
							>
								<!-- Barcode Monospace & Bold -->
								<td class="px-4 py-3 whitespace-nowrap">
									<div class="flex items-center gap-2">
										<svg
											class="h-4 w-4 {b.barcode === scannedBarcodeMatch
												? 'text-emerald-600'
												: 'text-neutral-400'}"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="1.5"
												d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5z"
											/>
										</svg>
										<span
											class="font-mono text-sm font-bold tracking-wider {b.barcode ===
											scannedBarcodeMatch
												? 'font-extrabold text-emerald-950'
												: 'text-neutral-900'}"
										>
											{b.barcode}
										</span>
										{#if b.barcode === scannedBarcodeMatch}
											<span
												class="inline-flex items-center rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-semibold text-emerald-800"
											>
												Cocok Scan
											</span>
										{/if}
									</div>
								</td>

								<!-- Peran / Primary Flag -->
								<td class="px-4 py-3 text-center whitespace-nowrap">
									{#if b.is_primary}
										<Badge variant="primary" size="sm">Barcode Utama</Badge>
									{:else}
										<Badge variant="default" size="sm">Sekunder</Badge>
									{/if}
								</td>

								<!-- Tanggal Registrasi -->
								<td class="px-4 py-3 text-xs whitespace-nowrap text-neutral-500">
									{new Date(b.created_at).toLocaleDateString('id-ID', {
										day: 'numeric',
										month: 'short',
										year: 'numeric',
										hour: '2-digit',
										minute: '2-digit'
									})}
								</td>

								<!-- Tombol Aksi Dropdown -->
								<td class="px-4 py-3 text-right whitespace-nowrap">
									<ActionMenu
										items={[
											{
												label: 'Cetak Label Barcode',
												icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M6.72 13.829c-.24.03-.48.062-.72.096m.72-.096a42.415 42.415 0 0110.56 0m-10.56 0L6.34 18m10.94-4.171c.24.03.48.062.72.096m-.72-.096L17.66 18m0 0l.229 2.523a1.125 1.125 0 01-1.12 1.227H7.231c-.662 0-1.18-.568-1.12-1.227L6.34 18m11.318 0h1.091A2.25 2.25 0 0021 15.75V9.456c0-1.081-.768-2.015-1.837-2.175a48.055 48.055 0 00-1.913-.247M6.34 18H5.25A2.25 2.25 0 013 15.75V9.456c0-1.081.768-2.015 1.837-2.175a48.041 48.041 0 011.913-.247m10.5 0a48.536 48.536 0 00-10.5 0m10.5 0V3.375c0-.621-.504-1.125-1.125-1.125h-8.25c-.621 0-1.125.504-1.125 1.125v3.656l10.5 0z" /></svg>`,
												onClick: () => openPrintModal(b)
											},
											{
												label: 'Salin Kode Barcode',
												icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" /></svg>`,
												onClick: () => {
													navigator.clipboard.writeText(b.barcode);
													toast.success('Kode barcode disalin ke clipboard');
												}
											},
											{
												label: 'Hapus Barcode',
												icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>`,
												variant: 'danger',
												divider: true,
												onClick: () => confirmDelete(b)
											}
										]}
									/>
								</td>
							</tr>
						{/each}
					</tbody>
				</Table>
			</div>
		</div>
	</div>
</div>

<!-- Modal Tambah Barcode Baru -->
<Modal
	open={showAddModal}
	title="Daftarkan Barcode Produk"
	size="md"
	onclose={() => (showAddModal = false)}
>
	<form onsubmit={handleSubmitAdd} class="space-y-4">
		{#if addError}
			<Alert variant="error" title="Gagal">{addError}</Alert>
		{/if}

		<div>
			<label for="form-barcode" class="mb-1.5 block text-xs font-medium text-neutral-700">
				Nomor Barcode Fisik <span class="text-rose-500">*</span>
			</label>
			<Input
				id="form-barcode"
				placeholder="Contoh: 8991234567890"
				bind:value={formBarcode}
				required
			/>
			<p class="mt-1 text-[11px] text-neutral-400">
				Dapat berupa kode EAN-13, UPC, Code-128 dari kemasan pabrik atau stiker cetakan toko.
			</p>
		</div>

		<div class="rounded-xl border border-neutral-200 bg-neutral-50/50 p-3.5">
			<Checkbox
				id="form-is-primary"
				label="Jadikan sebagai Barcode Utama (Primary)"
				bind:checked={formIsPrimary}
			/>
			<p class="mt-1 pl-6 text-[11px] text-neutral-500">
				Barcode utama dicetak pada struk belanja dan menjadi acuan default label rak.
			</p>
		</div>

		<div class="flex items-center justify-end gap-3 border-t border-neutral-200/80 pt-4">
			<Button
				variant="secondary"
				type="button"
				onclick={() => (showAddModal = false)}
				disabled={addLoading}
			>
				Batal
			</Button>
			<Button variant="primary" type="submit" loading={addLoading}>Daftarkan Barcode</Button>
		</div>
	</form>
</Modal>

<!-- Modal Konfirmasi Hapus Barcode -->
<Modal
	open={showDeleteModal}
	title="Konfirmasi Hapus Barcode"
	size="sm"
	onclose={() => (showDeleteModal = false)}
>
	<div class="space-y-4">
		{#if deleteError}
			<Alert variant="error" title="Gagal">{deleteError}</Alert>
		{/if}

		<p class="text-xs text-neutral-600">
			Apakah Anda yakin ingin menghapus nomor barcode <strong class="font-mono text-neutral-900"
				>{targetBarcode?.barcode}</strong
			>?
		</p>

		{#if targetBarcode?.is_primary}
			<div class="rounded-lg bg-amber-50 p-3 text-[11px] text-amber-700">
				<strong>Peringatan:</strong> Barcode ini adalah barcode utama produk. Pastikan mendaftarkan atau
				menetapkan barcode utama baru setelah penghapusan.
			</div>
		{/if}

		<div class="flex items-center justify-end gap-3 border-t border-neutral-200/80 pt-4">
			<Button
				variant="secondary"
				onclick={() => (showDeleteModal = false)}
				disabled={deleteLoading}
			>
				Batal
			</Button>
			<Button variant="danger" onclick={handleDelete} loading={deleteLoading}>Hapus Barcode</Button>
		</div>
	</div>
</Modal>

<!-- Modal Cetak Label Barcode -->
<Modal
	open={showPrintModal}
	title="Cetak Label Barcode"
	size="2xl"
	onclose={() => (showPrintModal = false)}
>
	<div class="space-y-6">
		{#if currentProduct && printTargetBarcode}
			<!-- Info Produk & Barcode -->
			<div class="rounded-xl border border-neutral-200 bg-neutral-50/70 p-4">
				<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
					<div>
						<span class="text-[10px] font-bold tracking-wider text-neutral-400 uppercase"
							>Produk Target</span
						>
						<h3 class="text-sm font-bold text-neutral-900">{currentProduct.name}</h3>
						<div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-neutral-600">
							<span>SKU: <strong class="font-mono">{currentProduct.sku}</strong></span>
							<span>•</span>
							<span
								>Barcode: <strong class="font-mono font-bold text-neutral-900"
									>{printTargetBarcode.barcode}</strong
								></span
							>
							{#if printTargetBarcode.is_primary}
								<Badge variant="primary" size="sm">Utama</Badge>
							{/if}
						</div>
					</div>
					<div class="sm:text-right">
						<span class="text-[10px] text-neutral-400">Harga Satuan</span>
						<div class="text-sm font-bold text-neutral-900">
							{new Intl.NumberFormat('id-ID', {
								style: 'currency',
								currency: 'IDR',
								maximumFractionDigits: 0
							}).format(currentProduct.selling_price)}
						</div>
					</div>
				</div>
			</div>

			<!-- Grid Pengaturan Cetak & Live Preview -->
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
				<!-- Pengaturan Cetak -->
				<div class="space-y-4">
					<!-- Jumlah Lembar Stiker -->
					<div>
						<label for="print-copies" class="mb-1.5 block text-xs font-semibold text-neutral-900">
							Jumlah Salinan / Lembar Barcode <span class="text-rose-500">*</span>
						</label>
						<div class="flex items-center gap-2">
							<div class="w-28">
								<Input id="print-copies" type="number" min="1" max="500" bind:value={printCopies} />
							</div>
							<span class="text-xs text-neutral-500">label stiker</span>
						</div>
						<!-- Tombol Cepat Qty -->
						<div class="mt-2 flex flex-wrap items-center gap-1.5">
							<span class="text-[11px] text-neutral-400">Pilihan cepat:</span>
							{#each [1, 2, 5, 10, 20, 50] as qty (qty)}
								<button
									type="button"
									onclick={() => (printCopies = qty)}
									class="rounded-md border px-2 py-0.5 text-xs font-medium transition-colors {printCopies ===
									qty
										? 'border-primary-600 bg-primary-50 font-semibold text-primary-700'
										: 'border-neutral-200 bg-white text-neutral-600 hover:border-neutral-300'}"
								>
									{qty}x
								</button>
							{/each}
						</div>
					</div>

					<!-- Format Kertas / Layout -->
					<div>
						<span class="mb-1.5 block text-xs font-semibold text-neutral-900"
							>Format Kertas / Printer</span
						>
						<div class="grid grid-cols-2 gap-2">
							<button
								type="button"
								onclick={() => (printFormat = 'single')}
								class="flex flex-col items-start rounded-xl border p-3 text-left transition-all {printFormat ===
								'single'
									? 'border-primary-600 bg-primary-50/60 ring-2 ring-primary-500/20'
									: 'border-neutral-200 bg-white hover:border-neutral-300'}"
							>
								<span class="text-xs font-bold text-neutral-900">Stiker Thermal POS</span>
								<span class="mt-0.5 text-[11px] text-neutral-500">Roll continuous (50x35mm)</span>
							</button>

							<button
								type="button"
								onclick={() => (printFormat = 'sheet')}
								class="flex flex-col items-start rounded-xl border p-3 text-left transition-all {printFormat ===
								'sheet'
									? 'border-primary-600 bg-primary-50/60 ring-2 ring-primary-500/20'
									: 'border-neutral-200 bg-white hover:border-neutral-300'}"
							>
								<span class="text-xs font-bold text-neutral-900">Lembar Kertas A4</span>
								<span class="mt-0.5 text-[11px] text-neutral-500">Multi-label grid berjejer</span>
							</button>
						</div>
					</div>

					<!-- Opsi Tampilan Stiker -->
					<div class="space-y-2 rounded-xl border border-neutral-200 bg-neutral-50/50 p-3.5">
						<span class="text-[11px] font-bold tracking-wider text-neutral-700 uppercase"
							>Elemen Stiker</span
						>
						<Checkbox
							id="print-show-store"
							label="Sertakan Nama Toko (GEN-E RETAIL)"
							bind:checked={printIncludeStoreName}
						/>
						<Checkbox
							id="print-show-name"
							label="Sertakan Nama Produk"
							bind:checked={printIncludeProductName}
						/>
						<Checkbox
							id="print-show-price"
							label="Sertakan Harga Jual"
							bind:checked={printIncludePrice}
						/>
					</div>
				</div>

				<!-- Live Preview Stiker -->
				<div
					class="flex flex-col items-center justify-center rounded-xl border border-neutral-200/80 bg-neutral-100/70 p-6"
				>
					<div class="mb-3 flex w-full items-center justify-between px-1">
						<span class="text-[11px] font-bold tracking-wider text-neutral-500 uppercase"
							>Pratinjau Stiker</span
						>
						<span class="text-[11px] font-medium text-neutral-500">Ukuran ~50x35mm</span>
					</div>

					<!-- Kartu Stiker Barcode -->
					<div
						class="flex min-h-[160px] w-72 flex-col items-center justify-between rounded-xl border border-neutral-300 bg-white p-4 text-center shadow-xs"
					>
						{#if printIncludeStoreName}
							<div class="text-[10px] font-extrabold tracking-widest text-neutral-500 uppercase">
								GEN-E RETAIL
							</div>
						{/if}

						{#if printIncludeProductName}
							<div class="mt-1 line-clamp-1 text-xs font-bold text-neutral-900">
								{currentProduct.name}
							</div>
						{/if}

						<!-- Barcode SVG Rendered -->
						<div id="barcode-preview-container" class="my-2 flex w-full justify-center">
							<Barcode
								value={printTargetBarcode.barcode}
								height={42}
								showText={true}
								class="max-w-[220px]"
							/>
						</div>

						{#if printIncludePrice}
							<div class="mt-1 text-xs font-extrabold text-neutral-900">
								{new Intl.NumberFormat('id-ID', {
									style: 'currency',
									currency: 'IDR',
									maximumFractionDigits: 0
								}).format(currentProduct.selling_price)}
							</div>
						{/if}
					</div>

					<p class="mt-3 text-center text-xs text-neutral-500">
						Total yang akan dicetak: <strong class="text-neutral-900">{printCopies} label</strong>
					</p>
				</div>
			</div>

			<!-- Footer Modal -->
			<div class="flex items-center justify-end gap-3 border-t border-neutral-200/80 pt-4">
				<Button variant="secondary" onclick={() => (showPrintModal = false)}>Batal</Button>
				<Button variant="primary" onclick={executePrint}>
					<svg
						class="mr-2 h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M6.72 13.829c-.24.03-.48.062-.72.096m.72-.096a42.415 42.415 0 0110.56 0m-10.56 0L6.34 18m10.94-4.171c.24.03.48.062.72.096m-.72-.096L17.66 18m0 0l.229 2.523a1.125 1.125 0 01-1.12 1.227H7.231c-.662 0-1.18-.568-1.12-1.227L6.34 18m11.318 0h1.091A2.25 2.25 0 0021 15.75V9.456c0-1.081-.768-2.015-1.837-2.175a48.055 48.055 0 00-1.913-.247M6.34 18H5.25A2.25 2.25 0 013 15.75V9.456c0-1.081.768-2.015 1.837-2.175a48.041 48.041 0 011.913-.247m10.5 0a48.536 48.536 0 00-10.5 0m10.5 0V3.375c0-.621-.504-1.125-1.125-1.125h-8.25c-.621 0-1.125.504-1.125 1.125v3.656l10.5 0z"
						/>
					</svg>
					Cetak {printCopies} Label
				</Button>
			</div>
		{/if}
	</div>
</Modal>
