<script lang="ts">
	/**
	 * Halaman Manajemen Serial Number & IMEI Tracking (Backoffice).
	 *
	 * Fitur Kunci:
	 * - Fast Barcode / IMEI Lookup Scanner: scan langsung nomor seri fisik untuk verifikasi lokasi & status riil unit.
	 * - Batch Serial Registration Modal: pendaftaran massal serial unit saat barang tiba via scanner laser / multi-line input.
	 * - Filter & Tabular View: pemfilteran unit berdasarkan cabang/lokasi, produk berserial, status siklus hidup, dan pencarian S/N.
	 * - Domain Invariant State Transition: siklus hidup terproteksi (tersedia -> terjual -> retur -> tersedia).
	 * - Zero-Warning TypeScript & Svelte 5 Runes.
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listSerials,
		lookupSerialUnit,
		registerSerialUnits,
		updateSerialStatus,
		listLocations,
		listProducts
	} from '@erp/api-client';
	import type {
		SerialUnitResponse,
		SerialUnitLookupResponse,
		LocationResponse,
		ProductResponse,
		SerialStatus
	} from '@erp/types';
	import { ApiError } from '@erp/types';
	import {
		Button,
		Select2,
		type Select2Option,
		Table,
		Pagination,
		Modal,
		Alert,
		Badge,
		SearchInput,
		ActionMenu,
		type ActionMenuItem,
		toast
	} from '@erp/ui';

	// ── 1. State Utama ────────────────────────────────────────────────────────
	let serials = $state<SerialUnitResponse[]>([]);
	let locations = $state<LocationResponse[]>([]);
	let products = $state<ProductResponse[]>([]);

	let loading = $state(true);
	let error = $state<string | null>(null);

	// ── 2. Filter & Pencarian ─────────────────────────────────────────────────
	let searchQuery = $state('');
	let selectedLocationId = $state<string>('');
	let selectedProductId = $state<string>('');
	let selectedStatusFilter = $state<string>('all');

	// Pagination
	let currentPage = $state(1);
	let pageSize = $state(15);

	// ── 3. State Fast Scanner / Lookup ────────────────────────────────────────
	let scanInput = $state('');
	let scanLoading = $state(false);
	let scanResult = $state<SerialUnitLookupResponse | null>(null);
	let scanError = $state<string | null>(null);

	// ── 4. State Modal Batch Registration ─────────────────────────────────────
	let showRegisterModal = $state(false);
	let regProductId = $state('');
	let regLocationId = $state('');
	let regSerialsRaw = $state('');
	let regSubmitting = $state(false);
	let regError = $state<string | null>(null);

	// ── 5. State Modal Update Status ──────────────────────────────────────────
	let showStatusModal = $state(false);
	let statusTargetItem = $state<SerialUnitResponse | null>(null);
	let statusTargetNewStatus = $state<SerialStatus>('tersedia');
	let statusSubmitting = $state(false);
	let statusError = $state<string | null>(null);

	// ── 6. Filter Produk yang Mengaktifkan Serial Tracking ─────────────────────
	const serialTrackedProducts = $derived.by(() => {
		return products.filter((p) => p.flag_serial_tracking);
	});

	// Select2 Options: Lokasi
	const locationOptions = $derived.by<Select2Option[]>(() => {
		const opts: Select2Option[] = [
			{ value: '', label: 'Semua Lokasi & Cabang', subtext: 'Tampilkan unit di seluruh cabang' }
		];
		for (const loc of locations) {
			opts.push({
				value: loc.id,
				label: `${loc.name} (${loc.code})`,
				subtext: loc.type === 'physical' ? 'Toko Fisik' : 'Gudang Online'
			});
		}
		return opts;
	});

	// Select2 Options: Produk Berserial
	const productFilterOptions = $derived.by<Select2Option[]>(() => {
		const opts: Select2Option[] = [
			{ value: '', label: 'Semua Produk Berserial', subtext: 'Tampilkan semua model barang' }
		];
		for (const p of serialTrackedProducts) {
			opts.push({
				value: p.id,
				label: `${p.name} (${p.sku})`,
				subtext: `${p.brand ? p.brand + ' - ' : ''}SKU: ${p.sku}`
			});
		}
		return opts;
	});

	// Select2 Options: Pendaftaran Produk (Hanya yang berserial aktif)
	const regProductOptions = $derived.by<Select2Option[]>(() => {
		const opts: Select2Option[] = [
			{ value: '', label: '-- Pilih Produk Berserial --', subtext: 'Wajib mengaktifkan serial tracking' }
		];
		for (const p of serialTrackedProducts) {
			opts.push({
				value: p.id,
				label: `${p.name} (${p.sku})`,
				subtext: `${p.brand ? p.brand + ' - ' : ''}SKU: ${p.sku}`
			});
		}
		return opts;
	});

	// Select2 Options: Lokasi Pendaftaran
	const regLocationOptions = $derived.by<Select2Option[]>(() => {
		const opts: Select2Option[] = [
			{ value: '', label: '-- Pilih Cabang / Gudang Tujuan --', subtext: 'Lokasi penerimaan fisik barang' }
		];
		for (const loc of locations) {
			opts.push({
				value: loc.id,
				label: `${loc.name} (${loc.code})`,
				subtext: loc.address || undefined
			});
		}
		return opts;
	});

	// Parsing input teks multi-line nomor seri
	const parsedRegSerials = $derived.by(() => {
		if (!regSerialsRaw.trim()) return [];
		const lines = regSerialsRaw
			.split(/[\r\n,]+/)
			.map((s) => s.trim())
			.filter((s) => s.length > 0);
		return lines;
	});

	// Deteksi duplikasi internal di dalam textarea pendaftaran
	const regDuplicateCount = $derived.by(() => {
		const seen = new Set<string>();
		let dupes = 0;
		for (const sn of parsedRegSerials) {
			if (seen.has(sn)) {
				dupes++;
			}
			seen.add(sn);
		}
		return dupes;
	});

	// ── 7. Filtering & Sorting Data Tabel ─────────────────────────────────────
	const filteredSerials = $derived.by(() => {
		let result = serials;

		// Filter Lokasi
		if (selectedLocationId) {
			result = result.filter((item) => item.location_id === selectedLocationId);
		}

		// Filter Produk
		if (selectedProductId) {
			result = result.filter((item) => item.product_id === selectedProductId);
		}

		// Filter Status
		if (selectedStatusFilter !== 'all') {
			result = result.filter((item) => {
				const st = item.status.toLowerCase();
				if (selectedStatusFilter === 'tersedia') return st === 'tersedia' || st === 'available';
				if (selectedStatusFilter === 'terjual') return st === 'terjual' || st === 'sold';
				if (selectedStatusFilter === 'retur') return st === 'retur' || st === 'returned' || st === 'defective';
				return true;
			});
		}

		// Filter Search Query
		if (searchQuery.trim()) {
			const q = searchQuery.toLowerCase().trim();
			result = result.filter((item) => {
				const snMatch = item.serial_number.toLowerCase().includes(q);
				const nameMatch = (item.product_name ?? '').toLowerCase().includes(q);
				const skuMatch = (item.product_sku ?? '').toLowerCase().includes(q);
				const locMatch = (item.location_name ?? '').toLowerCase().includes(q);
				return snMatch || nameMatch || skuMatch || locMatch;
			});
		}

		return result;
	});

	// Pagination slicing
	const paginatedSerials = $derived.by(() => {
		const start = (currentPage - 1) * pageSize;
		return filteredSerials.slice(start, start + pageSize);
	});

	// Total Pages
	const totalPages = $derived.by(() => {
		return Math.max(1, Math.ceil(filteredSerials.length / pageSize));
	});

	// ── 8. KPI Cards Metrics ──────────────────────────────────────────────────
	const kpiMetrics = $derived.by(() => {
		let total = serials.length;
		let tersedia = 0;
		let terjual = 0;
		let retur = 0;

		const uniqueProds = new Set<string>();
		const uniqueLocs = new Set<string>();

		for (const u of serials) {
			const st = u.status.toLowerCase();
			if (st === 'tersedia' || st === 'available') tersedia++;
			else if (st === 'terjual' || st === 'sold') terjual++;
			else if (st === 'retur' || st === 'returned' || st === 'defective') retur++;

			if (u.product_id) uniqueProds.add(u.product_id);
			if (u.location_id) uniqueLocs.add(u.location_id);
		}

		return {
			total,
			tersedia,
			terjual,
			retur,
			totalProducts: uniqueProds.size,
			totalLocations: uniqueLocs.size
		};
	});

	// ── 9. Fetch Data Awal ────────────────────────────────────────────────────
	async function loadData() {
		loading = true;
		error = null;
		const token = getToken();
		if (!token) {
			error = 'Sesi login tidak valid. Silakan login kembali.';
			loading = false;
			return;
		}

		try {
			const [serialsRes, locationsRes, productsRes] = await Promise.all([
				listSerials(token),
				listLocations(token),
				listProducts(token)
			]);

			serials = serialsRes;
			locations = locationsRes;
			products = productsRes.data;
		} catch (err) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat data serial dan inventaris';
			}
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadData();
	});

	// ── 10. Handler Fast Scanner / Lookup ─────────────────────────────────────
	async function handleScanSubmit(e?: Event) {
		if (e) e.preventDefault();
		const query = scanInput.trim();
		if (!query) return;

		const token = getToken();
		if (!token) {
			toast.error('Sesi telah berakhir');
			return;
		}

		scanLoading = true;
		scanError = null;
		scanResult = null;

		try {
			const result = await lookupSerialUnit(token, query);
			scanResult = result;
			toast.success(`Unit '${query}' ditemukan!`);
		} catch (err) {
			if (err instanceof ApiError) {
				scanError = err.message;
			} else if (err instanceof Error) {
				scanError = err.message;
			} else {
				scanError = `Unit dengan nomor seri '${query}' tidak ditemukan di sistem.`;
			}
			toast.error(scanError);
		} finally {
			scanLoading = false;
		}
	}

	function clearScan() {
		scanInput = '';
		scanResult = null;
		scanError = null;
	}

	function useScanInTable(sn: string) {
		searchQuery = sn;
		scanInput = sn;
		handleScanSubmit();
	}

	// ── 11. Handler Batch Registration ───────────────────────────────────────
	function openRegisterModal() {
		regProductId = serialTrackedProducts[0]?.id || '';
		regLocationId = locations[0]?.id || '';
		regSerialsRaw = '';
		regError = null;
		regSubmitting = false;
		showRegisterModal = true;
	}

	async function handleRegisterSubmit() {
		regError = null;

		if (!regProductId) {
			regError = 'Silakan pilih produk berserial yang akan didaftarkan.';
			return;
		}
		if (!regLocationId) {
			regError = 'Silakan pilih cabang / lokasi penerima unit.';
			return;
		}
		if (parsedRegSerials.length === 0) {
			regError = 'Masukkan minimal satu nomor seri atau IMEI yang valid.';
			return;
		}
		if (regDuplicateCount > 0) {
			regError = `Terdapat ${regDuplicateCount} nomor seri duplikat di dalam daftar input. Harap hapus duplikat sebelum melanjutkan.`;
			return;
		}

		const token = getToken();
		if (!token) {
			regError = 'Sesi telah berakhir. Silakan login kembali.';
			return;
		}

		regSubmitting = true;

		try {
			const newUnits = await registerSerialUnits(token, regProductId, {
				location_id: regLocationId,
				serial_numbers: parsedRegSerials
			});

			toast.success(`Berhasil mendaftarkan ${newUnits.length} unit fisik baru!`);
			showRegisterModal = false;
			await loadData();
		} catch (err) {
			if (err instanceof ApiError) {
				regError = err.message;
			} else if (err instanceof Error) {
				regError = err.message;
			} else {
				regError = 'Terjadi kesalahan saat mendaftarkan nomor seri.';
			}
			toast.error(regError);
		} finally {
			regSubmitting = false;
		}
	}

	// ── 12. Handler Status Transition Modal ──────────────────────────────────
	function openStatusModal(item: SerialUnitResponse) {
		statusTargetItem = item;
		const st = item.status.toLowerCase();
		if (st === 'tersedia' || st === 'available') {
			statusTargetNewStatus = 'terjual';
		} else if (st === 'terjual' || st === 'sold') {
			statusTargetNewStatus = 'retur';
		} else {
			statusTargetNewStatus = 'tersedia';
		}
		statusError = null;
		statusSubmitting = false;
		showStatusModal = true;
	}

	async function handleStatusSubmit() {
		if (!statusTargetItem) return;

		const token = getToken();
		if (!token) {
			statusError = 'Sesi telah berakhir. Silakan login kembali.';
			return;
		}

		statusSubmitting = true;
		statusError = null;

		try {
			await updateSerialStatus(token, statusTargetItem.id, statusTargetNewStatus);
			toast.success(
				`Status unit '${statusTargetItem.serial_number}' berhasil diubah menjadi '${statusTargetNewStatus}'!`
			);
			showStatusModal = false;

			// Jika sedang ada hasil lookup aktif untuk unit ini, perbarui juga
			if (scanResult && scanResult.serial_unit.id === statusTargetItem.id) {
				scanResult.serial_unit.status = statusTargetNewStatus;
			}

			await loadData();
		} catch (err) {
			if (err instanceof ApiError) {
				statusError = err.message;
			} else if (err instanceof Error) {
				statusError = err.message;
			} else {
				statusError = 'Gagal memperbarui status serial unit.';
			}
			toast.error(statusError);
		} finally {
			statusSubmitting = false;
		}
	}

	// Format status label & badge variant
	function getStatusBadge(st: string): { label: string; variant: 'success' | 'info' | 'warning' | 'default' } {
		const s = st.toLowerCase();
		if (s === 'tersedia' || s === 'available') {
			return { label: 'Tersedia', variant: 'success' };
		}
		if (s === 'terjual' || s === 'sold') {
			return { label: 'Terjual', variant: 'info' };
		}
		if (s === 'retur' || s === 'returned' || s === 'defective') {
			return { label: 'Retur / Klaim', variant: 'warning' };
		}
		return { label: st, variant: 'default' };
	}

	function formatDate(dt: string): string {
		try {
			return new Date(dt).toLocaleString('id-ID', {
				dateStyle: 'medium',
				timeStyle: 'short'
			});
		} catch {
			return dt;
		}
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		toast.info(`Disalin ke clipboard: ${text}`);
	}
</script>

<svelte:head>
	<title>Serial Number & IMEI Tracking — ERP Retail</title>
</svelte:head>

<div class="space-y-6">
	<!-- ── Header Halaman ──────────────────────────────────────────────────── -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-neutral-50">
					Serial & IMEI
				</h1>
				<Badge variant="purple" size="sm">Unit Fisik Individu</Badge>
			</div>
			<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
				Lacak posisi, histori mutasi, dan siklus hidup unit barang bernilai tinggi (smartphone, laptop, TV) secara presisi.
			</p>
		</div>

		<div class="flex flex-wrap items-center gap-2.5">
			<Button variant="outline" size="sm" onclick={loadData} disabled={loading}>
				<svg class="mr-1.5 h-4 w-4 {loading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				Refresh
			</Button>

			<Button variant="primary" size="sm" onclick={openRegisterModal}>
				<svg class="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				+ Daftarkan Unit Baru (Batch)
			</Button>
		</div>
	</div>

	<!-- ── Hero Scanner & Fast Lookup Card ─────────────────────────────────── -->
	<div class="rounded-xl border border-neutral-800 bg-neutral-950 p-5 text-neutral-50 shadow-xl dark:border-neutral-800">
		<div class="flex flex-col lg:flex-row lg:items-start lg:justify-between gap-6">
			<!-- Bagian Input Scanner -->
			<div class="flex-1 space-y-3">
				<div class="flex items-center gap-2">
					<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20">
						<svg class="h-4.5 w-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
						</svg>
					</div>
					<div>
						<h2 class="text-base font-semibold text-white">Fast Barcode & IMEI Scanner</h2>
						<p class="text-xs text-neutral-400">Tembakkan barcode scanner laser atau ketik nomor seri untuk cek kepemilikan unit dan validasi garansi.</p>
					</div>
				</div>

				<form onsubmit={handleScanSubmit} class="flex flex-col sm:flex-row items-center gap-2 pt-1">
					<div class="relative w-full flex-1">
						<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-neutral-500">
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
							</svg>
						</div>
						<input
							type="text"
							bind:value={scanInput}
							placeholder="Pindai barcode S/N / IMEI (contoh: SN-SAM-S24U-256-GRY-0101)..."
							class="w-full rounded-lg border border-neutral-700 bg-neutral-900/90 py-2.5 pl-10 pr-10 font-mono text-sm text-white placeholder-neutral-500 transition focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
						/>
						{#if scanInput}
							<button
								type="button"
								onclick={clearScan}
								class="absolute inset-y-0 right-0 flex items-center pr-3 text-neutral-400 hover:text-white"
								aria-label="Hapus teks"
							>
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
								</svg>
							</button>
						{/if}
					</div>

					<Button variant="primary" type="submit" loading={scanLoading} disabled={!scanInput.trim()} size="md">
						<svg class="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
						</svg>
						Cek Unit
					</Button>
				</form>

				{#if scanError}
					<div class="mt-2 rounded-lg border border-rose-900/50 bg-rose-950/40 p-3 text-xs text-rose-300">
						<div class="flex items-center gap-2">
							<svg class="h-4 w-4 shrink-0 text-rose-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
							</svg>
							<span>{scanError}</span>
						</div>
					</div>
				{/if}
			</div>

			<!-- Hasil Kartu Scanner Terpilih -->
			{#if scanResult}
				{@const stBadge = getStatusBadge(scanResult.serial_unit.status)}
				<div class="w-full lg:w-96 rounded-lg border border-neutral-800 bg-neutral-900 p-4 transition-all">
					<div class="flex items-start justify-between">
						<div>
							<div class="text-[11px] font-medium uppercase tracking-wider text-neutral-400">Hasil Pindai Fisik</div>
							<div class="mt-0.5 font-mono text-base font-bold text-emerald-400">
								{scanResult.serial_unit.serial_number}
							</div>
						</div>
						<Badge variant={stBadge.variant}>{stBadge.label}</Badge>
					</div>

					<div class="mt-3 space-y-1.5 border-t border-neutral-800 pt-2.5 text-xs">
						<div class="flex justify-between">
							<span class="text-neutral-400">Produk:</span>
							<span class="font-medium text-neutral-200 text-right">{scanResult.product_name}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-neutral-400">SKU / Brand:</span>
							<span class="font-mono text-neutral-300">{scanResult.product_sku} {scanResult.product_brand ? `(${scanResult.product_brand})` : ''}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-neutral-400">Lokasi Sekarang:</span>
							<span class="font-medium text-emerald-300">{scanResult.location_name} ({scanResult.location_code})</span>
						</div>
						<div class="flex justify-between">
							<span class="text-neutral-400">Tercatat:</span>
							<span class="text-neutral-300">{formatDate(scanResult.serial_unit.created_at)}</span>
						</div>
					</div>

					<div class="mt-3.5 flex items-center gap-2 border-t border-neutral-800 pt-2.5">
						<Button
							variant="outline"
							size="sm"
							fullWidth
							onclick={() => openStatusModal(scanResult!.serial_unit)}
						>
							Ubah Status
						</Button>
						<Button
							variant="secondary"
							size="sm"
							onclick={() => copyToClipboard(scanResult!.serial_unit.serial_number)}
						>
							Salin
						</Button>
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- ── KPI Metric Cards ────────────────────────────────────────────────── -->
	<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
		<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
			<div class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Total Unit Fisik</div>
			<div class="mt-1.5 text-2xl font-bold text-neutral-900 dark:text-neutral-100">
				{kpiMetrics.total}
			</div>
			<div class="mt-1 text-[11px] text-neutral-500">Unit terdaftar di sistem</div>
		</div>

		<div class="rounded-xl border border-emerald-200 bg-emerald-50/40 p-4 shadow-sm dark:border-emerald-950 dark:bg-emerald-950/20">
			<div class="text-xs font-medium text-emerald-700 dark:text-emerald-400">Tersedia di Toko</div>
			<div class="mt-1.5 text-2xl font-bold text-emerald-800 dark:text-emerald-300">
				{kpiMetrics.tersedia}
			</div>
			<div class="mt-1 text-[11px] text-emerald-600/80 dark:text-emerald-400/70">Siap dijual / dimutasi</div>
		</div>

		<div class="rounded-xl border border-blue-200 bg-blue-50/40 p-4 shadow-sm dark:border-blue-950 dark:bg-blue-950/20">
			<div class="text-xs font-medium text-blue-700 dark:text-blue-400">Terjual ke Konsumen</div>
			<div class="mt-1.5 text-2xl font-bold text-blue-800 dark:text-blue-300">
				{kpiMetrics.terjual}
			</div>
			<div class="mt-1 text-[11px] text-blue-600/80 dark:text-blue-400/70">Masa garansi aktif</div>
		</div>

		<div class="rounded-xl border border-amber-200 bg-amber-50/40 p-4 shadow-sm dark:border-amber-950 dark:bg-amber-950/20">
			<div class="text-xs font-medium text-amber-700 dark:text-amber-400">Retur & Cacat</div>
			<div class="mt-1.5 text-2xl font-bold text-amber-800 dark:text-amber-300">
				{kpiMetrics.retur}
			</div>
			<div class="mt-1 text-[11px] text-amber-600/80 dark:text-amber-400/70">Proses inspeksi / klaim</div>
		</div>

		<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
			<div class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Model Produk</div>
			<div class="mt-1.5 text-2xl font-bold text-neutral-900 dark:text-neutral-100">
				{kpiMetrics.totalProducts}
			</div>
			<div class="mt-1 text-[11px] text-neutral-500">SKU berserial aktif</div>
		</div>

		<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
			<div class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Titik Cabang</div>
			<div class="mt-1.5 text-2xl font-bold text-neutral-900 dark:text-neutral-100">
				{kpiMetrics.totalLocations}
			</div>
			<div class="mt-1 text-[11px] text-neutral-500">Cabang & gudang retail</div>
		</div>
	</div>

	<!-- ── Toolbar Filter & Pencarian ──────────────────────────────────────── -->
	<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900 space-y-4">
		<!-- Baris Filter Dropdown & Search -->
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
			<div>
				<label for="filter-location" class="mb-1 block text-xs font-medium text-neutral-700 dark:text-neutral-300">
					Filter Cabang / Lokasi
				</label>
				<Select2
					id="filter-location"
					bind:value={selectedLocationId}
					options={locationOptions}
					placeholder="Pilih lokasi cabang..."
				/>
			</div>

			<div>
				<label for="filter-product" class="mb-1 block text-xs font-medium text-neutral-700 dark:text-neutral-300">
					Filter Model Produk
				</label>
				<Select2
					id="filter-product"
					bind:value={selectedProductId}
					options={productFilterOptions}
					placeholder="Pilih model produk..."
				/>
			</div>

			<div>
				<label for="search-sn" class="mb-1 block text-xs font-medium text-neutral-700 dark:text-neutral-300">
					Cari Nomor Seri / Nama Produk
				</label>
				<SearchInput
					bind:value={searchQuery}
					placeholder="Cari S/N, IMEI, SKU, nama produk..."
				/>
			</div>
		</div>

		<!-- Status Tabs Filter -->
		<div class="flex flex-wrap items-center justify-between gap-3 border-t border-neutral-100 pt-3 dark:border-neutral-800">
			<div class="flex flex-wrap items-center gap-1.5">
				<span class="mr-1 text-xs font-medium text-neutral-500">Status:</span>

				<button
					type="button"
					onclick={() => { selectedStatusFilter = 'all'; currentPage = 1; }}
					class="rounded-lg px-3 py-1.5 text-xs font-medium transition {selectedStatusFilter === 'all'
						? 'bg-neutral-900 text-white dark:bg-white dark:text-neutral-900'
						: 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-400 dark:hover:bg-neutral-700'}"
				>
					Semua ({kpiMetrics.total})
				</button>

				<button
					type="button"
					onclick={() => { selectedStatusFilter = 'tersedia'; currentPage = 1; }}
					class="rounded-lg px-3 py-1.5 text-xs font-medium transition {selectedStatusFilter === 'tersedia'
						? 'bg-emerald-600 text-white'
						: 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-950/40 dark:text-emerald-400 dark:hover:bg-emerald-950/70'}"
				>
					Tersedia ({kpiMetrics.tersedia})
				</button>

				<button
					type="button"
					onclick={() => { selectedStatusFilter = 'terjual'; currentPage = 1; }}
					class="rounded-lg px-3 py-1.5 text-xs font-medium transition {selectedStatusFilter === 'terjual'
						? 'bg-blue-600 text-white'
						: 'bg-blue-50 text-blue-700 hover:bg-blue-100 dark:bg-blue-950/40 dark:text-blue-400 dark:hover:bg-blue-950/70'}"
				>
					Terjual ({kpiMetrics.terjual})
				</button>

				<button
					type="button"
					onclick={() => { selectedStatusFilter = 'retur'; currentPage = 1; }}
					class="rounded-lg px-3 py-1.5 text-xs font-medium transition {selectedStatusFilter === 'retur'
						? 'bg-amber-600 text-white'
						: 'bg-amber-50 text-amber-700 hover:bg-amber-100 dark:bg-amber-950/40 dark:text-amber-400 dark:hover:bg-amber-950/70'}"
				>
					Retur / Klaim ({kpiMetrics.retur})
				</button>
			</div>

			<div class="text-xs text-neutral-500">
				Menampilkan <span class="font-semibold text-neutral-900 dark:text-white">{filteredSerials.length}</span> unit
			</div>
		</div>
	</div>

	<!-- ── Alert Error Global ─────────────────────────────────────────────── -->
	{#if error}
		<Alert variant="error" title="Gagal Memuat Data">
			{error}
		</Alert>
	{/if}

	<!-- ── Tabel Daftar Serial Number ──────────────────────────────────────── -->
	<div class="rounded-xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
		<Table>
			<thead>
				<tr>
					<th>Nomor Seri / IMEI</th>
					<th>Produk & SKU</th>
					<th>Lokasi Fisik</th>
					<th>Status Siklus</th>
					<th>Tercatat Masuk</th>
					<th class="text-right">Aksi</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="6" class="py-12 text-center">
							<div class="flex flex-col items-center justify-center gap-2">
								<svg class="h-6 w-6 animate-spin text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
								</svg>
								<span class="text-xs text-neutral-500">Memuat daftar unit serial...</span>
							</div>
						</td>
					</tr>
				{:else if paginatedSerials.length === 0}
					<tr>
						<td colspan="6" class="py-12 text-center">
							<div class="flex flex-col items-center justify-center gap-2 text-neutral-500 dark:text-neutral-400">
								<svg class="h-10 w-10 text-neutral-300 dark:text-neutral-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.75 3.104v5.714a2.25 2.25 0 01-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 014.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.943A4.5 4.5 0 0115.84 17H8.16a4.5 4.5 0 01-2.39-.757L4.2 15.3m15.6 0v2.4a2.25 2.25 0 01-2.25 2.25H6.45A2.25 2.25 0 014.2 17.7v-2.4" />
								</svg>
								<div class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Tidak ada nomor seri yang cocok</div>
								<div class="text-xs">Coba sesuaikan filter cabang, produk, atau kata kunci pencarian Anda.</div>
							</div>
						</td>
					</tr>
				{:else}
					{#each paginatedSerials as item (item.id)}
						{@const st = getStatusBadge(item.status)}
						<tr class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition">
							<!-- Nomor Seri & Salin -->
							<td class="font-mono text-xs">
								<div class="flex items-center gap-2">
									<span class="font-bold text-neutral-900 dark:text-neutral-100">{item.serial_number}</span>
									<button
										type="button"
										onclick={() => copyToClipboard(item.serial_number)}
										title="Salin nomor seri"
										class="text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 transition"
										aria-label="Salin nomor seri"
									>
										<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
										</svg>
									</button>
								</div>
							</td>

							<!-- Produk & SKU -->
							<td>
								<div class="text-sm font-medium text-neutral-900 dark:text-neutral-100">
									{item.product_name ?? 'Produk Terdaftar'}
								</div>
								<div class="mt-0.5 flex items-center gap-1.5 text-xs text-neutral-500">
									<Badge variant="indigo" size="sm">
										<span class="font-mono">{item.product_sku ?? item.product_id}</span>
									</Badge>
									{#if item.product_brand}
										<span>• {item.product_brand}</span>
									{/if}
								</div>
							</td>

							<!-- Lokasi Fisik -->
							<td>
								<div class="flex items-center gap-1.5 text-sm text-neutral-800 dark:text-neutral-200">
									<svg class="h-4 w-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
									</svg>
									<span>{item.location_name ?? 'Gudang'}</span>
								</div>
								{#if item.location_code}
									<div class="text-[11px] font-mono text-neutral-400 pl-5.5">
										Kode: {item.location_code}
									</div>
								{/if}
							</td>

							<!-- Status Badge -->
							<td>
								<Badge variant={st.variant}>{st.label}</Badge>
							</td>

							<!-- Tanggal Masuk -->
							<td class="text-xs text-neutral-500">
								{formatDate(item.created_at)}
							</td>

							<!-- Aksi Dropdown -->
							<td class="text-right whitespace-nowrap">
								<ActionMenu
									items={[
										{
											label: 'Ubah Status Unit',
											icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" /></svg>`,
											onClick: () => openStatusModal(item)
										},
										{
											label: 'Pindai di Scanner Simulator',
											icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>`,
											onClick: () => useScanInTable(item.serial_number)
										},
										{
											label: 'Salin Nomor Seri',
											icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" /></svg>`,
											divider: true,
											onClick: () => {
												navigator.clipboard.writeText(item.serial_number);
												toast.success(`SN ${item.serial_number} disalin`);
											}
										}
									]}
								/>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</Table>

		<!-- Pagination Footer -->
		{#if filteredSerials.length > 0}
			<div class="border-t border-neutral-200 p-4 dark:border-neutral-800">
				<Pagination
					page={currentPage}
					{totalPages}
					totalItems={filteredSerials.length}
					limit={pageSize}
					onPageChange={(p) => (currentPage = p)}
				/>
			</div>
		{/if}
	</div>
</div>

<!-- ── Modal 1: Batch Serial Registration ───────────────────────────────────── -->
<Modal
	bind:open={showRegisterModal}
	title="Daftarkan Nomor Seri / IMEI Baru (Batch Scanner)"
	size="xl"
>
	<div class="space-y-4">
		<div class="rounded-lg bg-neutral-50 p-3 text-xs text-neutral-600 dark:bg-neutral-800/60 dark:text-neutral-300">
			<div class="font-semibold text-neutral-800 dark:text-neutral-100">Alur Pendaftaran Gudang & Penerimaan:</div>
			<p class="mt-0.5">
				Gunakan scanner laser untuk menembak barcode SN/IMEI di kardus produk satu per satu. Setiap tembakan barcode scanner akan otomatis mengisi baris baru.
			</p>
		</div>

		{#if regError}
			<Alert variant="error" title="Gagal Mendaftarkan Serial">
				{regError}
			</Alert>
		{/if}

		<div>
			<label for="reg-product" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
				Produk Berserial <span class="text-rose-500">*</span>
			</label>
			<Select2
				id="reg-product"
				bind:value={regProductId}
				options={regProductOptions}
				placeholder="Pilih produk..."
			/>
		</div>

		<div>
			<label for="reg-location" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
				Cabang / Gudang Penerima <span class="text-rose-500">*</span>
			</label>
			<Select2
				id="reg-location"
				bind:value={regLocationId}
				options={regLocationOptions}
				placeholder="Pilih lokasi..."
			/>
		</div>

		<div>
			<div class="flex items-center justify-between mb-1">
				<label for="reg-serials-raw" class="text-xs font-semibold text-neutral-800 dark:text-neutral-200">
					Daftar Nomor Seri / IMEI (Multi-line) <span class="text-rose-500">*</span>
				</label>
				<span class="text-xs text-neutral-500">
					Terdeteksi: <span class="font-bold text-emerald-600 dark:text-emerald-400">{parsedRegSerials.length}</span> unit
				</span>
			</div>
			<textarea
				id="reg-serials-raw"
				bind:value={regSerialsRaw}
				rows="7"
				placeholder="Arahkan laser scanner ke barcode S/N atau masukkan satu nomor seri per baris:&#10;SN-APL-IP15P-001&#10;SN-APL-IP15P-002&#10;356938035643809"
				class="w-full rounded-lg border border-neutral-300 bg-white p-3 font-mono text-xs text-neutral-900 transition focus:border-neutral-900 focus:outline-none focus:ring-1 focus:ring-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100 dark:focus:border-white dark:focus:ring-white"
			></textarea>
			{#if regDuplicateCount > 0}
				<div class="mt-1 text-xs text-rose-500">
					Peringatan: Terdeteksi {regDuplicateCount} nomor seri duplikat di dalam daftar input!
				</div>
			{/if}
		</div>
	</div>

	{#snippet footer()}
		<div class="flex items-center justify-end gap-2.5">
			<Button
				variant="outline"
				size="sm"
				onclick={() => (showRegisterModal = false)}
				disabled={regSubmitting}
			>
				Batal
			</Button>

			<Button
				variant="primary"
				size="sm"
				loading={regSubmitting}
				disabled={parsedRegSerials.length === 0 || regDuplicateCount > 0}
				onclick={handleRegisterSubmit}
			>
				Daftarkan {parsedRegSerials.length > 0 ? `${parsedRegSerials.length} Unit` : ''}
			</Button>
		</div>
	{/snippet}
</Modal>

<!-- ── Modal 2: Update Serial Status ────────────────────────────────────────── -->
<Modal
	bind:open={showStatusModal}
	title="Ubah Status Siklus Hidup Serial"
	size="md"
>
	{#if statusTargetItem}
		<div class="space-y-4">
			<div class="rounded-lg border border-neutral-200 bg-neutral-50 p-3.5 dark:border-neutral-800 dark:bg-neutral-800/40">
				<div class="text-[11px] font-medium uppercase tracking-wider text-neutral-400">Nomor Seri / IMEI</div>
				<div class="mt-0.5 font-mono text-base font-bold text-neutral-900 dark:text-neutral-50">
					{statusTargetItem.serial_number}
				</div>
				<div class="mt-2 text-xs text-neutral-600 dark:text-neutral-300">
					Produk: <span class="font-medium text-neutral-900 dark:text-white">{statusTargetItem.product_name ?? 'Produk'}</span>
				</div>
				<div class="mt-1 flex items-center gap-2 text-xs">
					<span class="text-neutral-500">Status Saat Ini:</span>
					{#if statusTargetItem}
						{@const currSt = getStatusBadge(statusTargetItem.status)}
						<Badge variant={currSt.variant} size="sm">{currSt.label}</Badge>
					{/if}
				</div>
			</div>

			{#if statusError}
				<Alert variant="error" title="Gagal Ubah Status">
					{statusError}
				</Alert>
			{/if}

			<div>
				<label for="select-new-status" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
					Pilih Status Baru <span class="text-rose-500">*</span>
				</label>
				<select
					id="select-new-status"
					bind:value={statusTargetNewStatus}
					class="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 transition focus:border-neutral-900 focus:outline-none focus:ring-1 focus:ring-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100 dark:focus:border-white"
				>
					<option value="tersedia">Tersedia (Siap Jual / Dimutasi)</option>
					<option value="terjual">Terjual (Transaksi Kasir Selesai)</option>
					<option value="retur">Retur (Pengembalian Pelanggan / Klaim Garansi)</option>
				</select>
			</div>

			<div class="rounded-lg bg-neutral-50 p-3 text-xs text-neutral-600 dark:bg-neutral-800/60 dark:text-neutral-400">
				<span class="font-semibold text-neutral-800 dark:text-neutral-200">Aturan Domain Invariant:</span>
				<ul class="mt-1 list-disc pl-4 space-y-1">
					<li>Unit 'Tersedia' hanya boleh dijual menjadi 'Terjual'.</li>
					<li>Unit 'Terjual' hanya boleh diretur menjadi 'Retur'.</li>
					<li>Unit 'Retur' setelah selesai servis/inspeksi dapat dikembalikan menjadi 'Tersedia'.</li>
				</ul>
			</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex items-center justify-end gap-2.5">
			<Button
				variant="outline"
				size="sm"
				onclick={() => (showStatusModal = false)}
				disabled={statusSubmitting}
			>
				Batal
			</Button>

			<Button
				variant="primary"
				size="sm"
				loading={statusSubmitting}
				onclick={handleStatusSubmit}
			>
				Simpan Status Baru
			</Button>
		</div>
	{/snippet}
</Modal>
