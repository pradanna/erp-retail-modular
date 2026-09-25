<script lang="ts">
	/**
	 * Halaman Manajemen Promo & Harga Khusus Cabang (Price Overrides).
	 *
	 * Fitur Kunci:
	 * - Kalkulator / Simulator Harga Efektif Kasir POS (Live Price Simulator).
	 * - Pembuatan Promo Cabang Baru dengan Validasi Domain Invariant Anti-Tabrakan (Anti-Overlap).
	 * - Manajemen Kuota Flash Sale & Pelacakan Unit Terklaim Real-time.
	 * - Penonaktifan Cepat Promo (Deactivate).
	 * - Filter Cabang, Produk, Status Aktif/Kadaluarsa, dan Pencarian Kampanye.
	 * - Zero-Warning TypeScript & Svelte 5 Runes.
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listAllPriceOverrides,
		createPriceOverride,
		deactivatePriceOverride,
		getEffectivePrice,
		listLocations,
		listProducts
	} from '@erp/api-client';
	import type {
		PriceOverrideResponse,
		EffectivePriceResponse,
		LocationResponse,
		ProductResponse
	} from '@erp/types';
	import { ApiError } from '@erp/types';
	import {
		Button,
		Input,
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
	let overrides = $state<PriceOverrideResponse[]>([]);
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
	let pageSize = $state(10);

	// ── 3. State Live POS Price Simulator ─────────────────────────────────────
	let simLocationId = $state<string>('');
	let simProductId = $state<string>('');
	let simResult = $state<EffectivePriceResponse | null>(null);
	let simLoading = $state(false);
	let simError = $state<string | null>(null);

	// ── 4. State Modal Buat Promo Baru ────────────────────────────────────────
	let showCreateModal = $state(false);
	let newLocationId = $state('');
	let newProductId = $state('');
	let newPromoPrice = $state<number>(0);
	let isQuotaLimited = $state(false);
	let newMaxQty = $state<number>(10);
	let newStartDate = $state('');
	let newEndDate = $state('');
	let newReason = $state('');
	let createSubmitting = $state(false);
	let createError = $state<string | null>(null);

	// ── 5. State Modal Nonaktifkan Promo ──────────────────────────────────────
	let showDeactivateModal = $state(false);
	let targetDeactivateItem = $state<PriceOverrideResponse | null>(null);
	let deactivateSubmitting = $state(false);

	// ── 6. Select2 Options & Helper ───────────────────────────────────────────
	const locationOptions = $derived.by<Select2Option[]>(() => {
		const opts: Select2Option[] = [
			{ value: '', label: 'Semua Cabang / Lokasi', subtext: 'Tampilkan promo di seluruh cabang' }
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

	const productOptions = $derived.by<Select2Option[]>(() => {
		const opts: Select2Option[] = [
			{ value: '', label: 'Semua Produk Katalog', subtext: 'Tampilkan semua produk' }
		];
		for (const p of products) {
			opts.push({
				value: p.id,
				label: `${p.name} (${p.sku})`,
				subtext: `Normal: ${formatRupiah(p.selling_price)} • SKU: ${p.sku}`
			});
		}
		return opts;
	});

	// Select2 Options khusus form pembuatan (tanpa opsi "Semua")
	const formLocationOptions = $derived.by<Select2Option[]>(() => {
		const opts: Select2Option[] = [
			{ value: '', label: '-- Pilih Cabang / Toko --', subtext: 'Cabang yang memberlakukan promo' }
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

	const formProductOptions = $derived.by<Select2Option[]>(() => {
		const opts: Select2Option[] = [
			{ value: '', label: '-- Pilih Produk --', subtext: 'Produk yang akan dipromokan' }
		];
		for (const p of products) {
			opts.push({
				value: p.id,
				label: `${p.name} (${p.sku})`,
				subtext: `Harga Normal: ${formatRupiah(p.selling_price)}`
			});
		}
		return opts;
	});

	// Produk yang terpilih saat membuat promo baru
	const selectedProductForNew = $derived.by(() => {
		return products.find((p) => p.id === newProductId);
	});

	// Hitung kalkulasi diskon otomatis pada form modal
	const formDiscountPreview = $derived.by(() => {
		if (!selectedProductForNew || newPromoPrice <= 0) return null;
		const base = selectedProductForNew.selling_price;
		const diff = base - newPromoPrice;
		const pct = Math.round((diff / base) * 100);
		return {
			basePrice: base,
			promoPrice: newPromoPrice,
			discountAmount: diff,
			discountPct: pct,
			isCheaper: diff > 0
		};
	});

	// ── 7. Filtering & Sorting Tabel ──────────────────────────────────────────
	const nowTime = new Date().getTime();

	const filteredOverrides = $derived.by(() => {
		let result = overrides;

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
				const start = new Date(item.start_date).getTime();
				const end = new Date(item.end_date).getTime();
				const isCurrentlyActive = item.is_active && nowTime >= start && nowTime <= end;
				const isQuotaExhausted = item.max_quantity != null && item.claimed_quantity >= item.max_quantity;

				if (selectedStatusFilter === 'active') {
					return isCurrentlyActive && !isQuotaExhausted;
				}
				if (selectedStatusFilter === 'exhausted') {
					return item.is_active && isQuotaExhausted;
				}
				if (selectedStatusFilter === 'inactive') {
					return !item.is_active || nowTime > end;
				}
				return true;
			});
		}

		// Filter Search Query
		if (searchQuery.trim()) {
			const q = searchQuery.toLowerCase().trim();
			result = result.filter((item) => {
				const reasonMatch = (item.reason ?? '').toLowerCase().includes(q);
				const nameMatch = (item.product_name ?? '').toLowerCase().includes(q);
				const skuMatch = (item.product_sku ?? '').toLowerCase().includes(q);
				const locMatch = (item.location_name ?? '').toLowerCase().includes(q);
				return reasonMatch || nameMatch || skuMatch || locMatch;
			});
		}

		return result;
	});

	// Pagination slicing
	const paginatedOverrides = $derived.by(() => {
		const start = (currentPage - 1) * pageSize;
		return filteredOverrides.slice(start, start + pageSize);
	});

	const totalPages = $derived.by(() => {
		return Math.max(1, Math.ceil(filteredOverrides.length / pageSize));
	});

	// ── 8. KPI Metrics ────────────────────────────────────────────────────────
	const kpiMetrics = $derived.by(() => {
		let total = overrides.length;
		let active = 0;
		let flashSale = 0;
		let totalClaimed = 0;
		let expiredOrInactive = 0;

		for (const o of overrides) {
			const start = new Date(o.start_date).getTime();
			const end = new Date(o.end_date).getTime();
			const isRunning = o.is_active && nowTime >= start && nowTime <= end;
			const isQuotaExhausted = o.max_quantity != null && o.claimed_quantity >= o.max_quantity;

			if (isRunning && !isQuotaExhausted) {
				active++;
			} else {
				expiredOrInactive++;
			}

			if (o.max_quantity != null) {
				flashSale++;
			}
			totalClaimed += o.claimed_quantity;
		}

		return {
			total,
			active,
			flashSale,
			totalClaimed,
			expiredOrInactive
		};
	});

	// ── 9. Fetch Data Awal ────────────────────────────────────────────────────
	async function loadData() {
		loading = true;
		error = null;
		const token = getToken();
		if (!token) {
			error = 'Sesi telah berakhir. Silakan login kembali.';
			loading = false;
			return;
		}

		try {
			const [overridesRes, locationsRes, productsRes] = await Promise.all([
				listAllPriceOverrides(token),
				listLocations(token),
				listProducts(token)
			]);

			overrides = overridesRes;
			locations = locationsRes;
			products = productsRes.data;

			// Inisialisasi simulator dengan pilihan pertama jika belum dipilih
			if (!simLocationId && locations.length > 0) {
				simLocationId = locations[0].id;
			}
			if (!simProductId && products.length > 0) {
				simProductId = products[0].id;
				runSimulator();
			}
		} catch (err) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat data promo harga cabang';
			}
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadData();
	});

	// ── 10. Handler Live POS Price Simulator ──────────────────────────────────
	async function runSimulator() {
		if (!simProductId || !simLocationId) return;

		simLoading = true;
		simError = null;
		const token = getToken();
		if (!token) return;

		try {
			const result = await getEffectivePrice(token, simProductId, simLocationId);
			simResult = result;
		} catch (err) {
			if (err instanceof ApiError) {
				simError = err.message;
			} else if (err instanceof Error) {
				simError = err.message;
			} else {
				simError = 'Gagal menghitung harga efektif kasir';
			}
		} finally {
			simLoading = false;
		}
	}

	function populateSimulator(productId: string, locationId: string) {
		simProductId = productId;
		simLocationId = locationId;
		runSimulator();
		// Scroll smooth ke bagian hero simulator
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}

	// ── 11. Handler Buat Promo Baru ───────────────────────────────────────────
	function openCreateModal() {
		newLocationId = locations[0]?.id || '';
		newProductId = products[0]?.id || '';
		const p = products[0];
		newPromoPrice = p ? Math.round(p.selling_price * 0.9) : 0;
		isQuotaLimited = false;
		newMaxQty = 20;

		// Default rentang tanggal: mulai hari ini hingga 30 hari ke depan
		const now = new Date();
		const in30Days = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
		newStartDate = formatDateTimeForInput(now);
		newEndDate = formatDateTimeForInput(in30Days);
		newReason = 'Promo Diskon Spesial Cabang';

		createError = null;
		createSubmitting = false;
		showCreateModal = true;
	}

	async function handleCreateSubmit() {
		createError = null;

		if (!newProductId) {
			createError = 'Silakan pilih produk yang akan diberikan promo.';
			return;
		}
		if (!newLocationId) {
			createError = 'Silakan pilih cabang / lokasi toko berlakunya promo.';
			return;
		}
		if (newPromoPrice <= 0) {
			createError = 'Harga promo harus lebih dari Rp 0.';
			return;
		}
		if (!newStartDate || !newEndDate) {
			createError = 'Tanggal mulai dan berakhirnya promo wajib diisi.';
			return;
		}
		if (new Date(newEndDate) <= new Date(newStartDate)) {
			createError = 'Tanggal berakhir promo harus setelah tanggal mulai promo.';
			return;
		}
		if (isQuotaLimited && newMaxQty <= 0) {
			createError = 'Batas kuota flash sale harus lebih dari 0.';
			return;
		}

		createSubmitting = true;
		const token = getToken();
		if (!token) return;

		try {
			await createPriceOverride(token, newProductId, {
				location_id: newLocationId,
				promotional_price: newPromoPrice,
				max_quantity: isQuotaLimited ? newMaxQty : undefined,
				start_date: new Date(newStartDate).toISOString(),
				end_date: new Date(newEndDate).toISOString(),
				reason: newReason.trim()
			});

			toast.success('Promo harga khusus cabang berhasil dibuat!');
			showCreateModal = false;
			await loadData();
			// Perbarui simulator jika produknya sama
			if (simProductId === newProductId && simLocationId === newLocationId) {
				runSimulator();
			}
		} catch (err) {
			if (err instanceof ApiError) {
				createError = err.message;
			} else if (err instanceof Error) {
				createError = err.message;
			} else {
				createError = 'Terjadi kesalahan saat membuat promo cabang.';
			}
			toast.error(createError);
		} finally {
			createSubmitting = false;
		}
	}

	// ── 12. Handler Deactivate Promo ──────────────────────────────────────────
	function openDeactivateModal(item: PriceOverrideResponse) {
		targetDeactivateItem = item;
		deactivateSubmitting = false;
		showDeactivateModal = true;
	}

	async function handleDeactivateSubmit() {
		if (!targetDeactivateItem) return;

		deactivateSubmitting = true;
		const token = getToken();
		if (!token) return;

		try {
			await deactivatePriceOverride(token, targetDeactivateItem.id);
			toast.success(`Promo '${targetDeactivateItem.reason}' berhasil dinonaktifkan!`);
			showDeactivateModal = false;
			await loadData();
			if (simProductId === targetDeactivateItem.product_id && simLocationId === targetDeactivateItem.location_id) {
				runSimulator();
			}
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Gagal menonaktifkan promo';
			toast.error(msg);
		} finally {
			deactivateSubmitting = false;
		}
	}

	// ── 13. Formatter Helper ──────────────────────────────────────────────────
	function formatRupiah(num: number): string {
		return 'Rp ' + Math.round(num).toLocaleString('id-ID');
	}

	function formatDate(dt: string): string {
		try {
			return new Date(dt).toLocaleDateString('id-ID', {
				dateStyle: 'medium'
			});
		} catch {
			return dt;
		}
	}

	function formatDateTimeForInput(d: Date): string {
		const pad = (n: number) => n.toString().padStart(2, '0');
		return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
	}

	function getPromoStatusBadge(item: PriceOverrideResponse): { label: string; variant: 'success' | 'warning' | 'danger' | 'default' } {
		if (!item.is_active) {
			return { label: 'Nonaktif', variant: 'default' };
		}
		const start = new Date(item.start_date).getTime();
		const end = new Date(item.end_date).getTime();

		if (nowTime < start) {
			return { label: 'Akan Datang', variant: 'warning' };
		}
		if (nowTime > end) {
			return { label: 'Kadaluarsa', variant: 'default' };
		}
		if (item.max_quantity != null && item.claimed_quantity >= item.max_quantity) {
			return { label: 'Kuota Habis', variant: 'warning' };
		}
		return { label: 'Sedang Aktif', variant: 'success' };
	}
</script>

<svelte:head>
	<title>Promo & Harga Khusus Cabang — ERP Retail</title>
</svelte:head>

<div class="space-y-6">
	<!-- ── Header Halaman ──────────────────────────────────────────────────── -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-neutral-50">
					Promo Cabang
				</h1>
				<Badge variant="indigo" size="sm">Price Override Engine</Badge>
			</div>
			<p class="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
				Kelola harga jual promo spesifik per cabang ritel, kampanye flash sale berkuota, dan hitung harga efektif kasir POS secara otomatis.
			</p>
		</div>

		<div class="flex flex-wrap items-center gap-2.5">
			<Button variant="outline" size="sm" onclick={loadData} disabled={loading}>
				<svg class="mr-1.5 h-4 w-4 {loading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
				</svg>
				Refresh
			</Button>

			<Button variant="primary" size="sm" onclick={openCreateModal}>
				<svg class="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				+ Buat Promo Cabang Baru
			</Button>
		</div>
	</div>

	<!-- ── Hero: Live POS Price Simulator (Kalkulator Kasir) ────────────────── -->
	<div class="rounded-xl border border-neutral-800 bg-neutral-950 p-5 text-neutral-50 shadow-xl dark:border-neutral-800">
		<div class="flex flex-col lg:flex-row lg:items-start lg:justify-between gap-6">
			<!-- Form Pilihan Simulator -->
			<div class="flex-1 space-y-4">
				<div class="flex items-center gap-2">
					<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400 ring-1 ring-indigo-500/20">
						<svg class="h-4.5 w-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
						</svg>
					</div>
					<div>
						<h2 class="text-base font-semibold text-white">Live POS Price Simulator</h2>
						<p class="text-xs text-neutral-400">Simulasikan harga riil yang akan ditagihkan ke konsumen saat kasir men-scan barang di cabang terpilih.</p>
					</div>
				</div>

				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 pt-1">
					<div>
						<label for="sim-location" class="mb-1 block text-xs font-medium text-neutral-300">
							Pilih Cabang Kasir:
						</label>
						<Select2
							id="sim-location"
							bind:value={simLocationId}
							options={formLocationOptions}
							placeholder="Pilih cabang..."
							onchange={runSimulator}
						/>
					</div>

					<div>
						<label for="sim-product" class="mb-1 block text-xs font-medium text-neutral-300">
							Pilih Produk yang Di-scan:
						</label>
						<Select2
							id="sim-product"
							bind:value={simProductId}
							options={formProductOptions}
							placeholder="Pilih produk..."
							onchange={runSimulator}
						/>
					</div>
				</div>

				{#if simError}
					<div class="rounded-lg border border-rose-900/50 bg-rose-950/40 p-2.5 text-xs text-rose-300">
						{simError}
					</div>
				{/if}
			</div>

			<!-- Hasil Output Simulator Kasir POS -->
			<div class="w-full lg:w-96 rounded-lg border border-neutral-800 bg-neutral-900 p-4 transition-all">
				<div class="flex items-start justify-between">
					<div class="text-[11px] font-medium uppercase tracking-wider text-neutral-400">Harga Final di Kasir POS</div>
					{#if simResult}
						{#if simResult.has_discount}
							<Badge variant="success">Promo Aktif</Badge>
						{:else}
							<Badge variant="default">Harga Standar</Badge>
						{/if}
					{/if}
				</div>

				{#if simLoading}
					<div class="py-6 text-center text-xs text-neutral-400">
						Menghitung harga efektif...
					</div>
				{:else if simResult}
					<div class="mt-2">
						<div class="text-2xl font-bold {simResult.has_discount ? 'text-emerald-400' : 'text-white'}">
							{formatRupiah(simResult.effective_price)}
						</div>

						{#if simResult.has_discount}
							<div class="mt-1 flex items-center gap-2 text-xs">
								<span class="text-neutral-400 line-through">{formatRupiah(simResult.base_price)}</span>
								<span class="font-semibold text-emerald-400">Hemat {formatRupiah(simResult.discount_amount)}!</span>
							</div>
						{/if}
					</div>

					<div class="mt-3.5 space-y-1.5 border-t border-neutral-800 pt-2.5 text-xs">
						{#if simResult.promo_reason}
							<div class="flex justify-between">
								<span class="text-neutral-400">Kampanye:</span>
								<span class="font-medium text-neutral-200 text-right">{simResult.promo_reason}</span>
							</div>
						{/if}

						<div class="flex justify-between">
							<span class="text-neutral-400">Status Kuota:</span>
							{#if simResult.remaining_quota != null}
								<span class="font-semibold text-amber-300">Sisa {simResult.remaining_quota} unit</span>
							{:else}
								<span class="text-neutral-300">Tanpa Batas Kuota</span>
							{/if}
						</div>
					</div>
				{:else}
					<div class="py-6 text-center text-xs text-neutral-500">
						Pilih cabang dan produk untuk melihat kalkulasi harga kasir.
					</div>
				{/if}
			</div>
		</div>
	</div>

	<!-- ── KPI Metric Cards ────────────────────────────────────────────────── -->
	<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
		<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
			<div class="text-xs font-medium text-neutral-500 dark:text-neutral-400">Total Promo Didaftarkan</div>
			<div class="mt-1.5 text-2xl font-bold text-neutral-900 dark:text-neutral-100">
				{kpiMetrics.total}
			</div>
			<div class="mt-1 text-[11px] text-neutral-500">Sepanjang operasional ritel</div>
		</div>

		<div class="rounded-xl border border-emerald-200 bg-emerald-50/40 p-4 shadow-sm dark:border-emerald-950 dark:bg-emerald-950/20">
			<div class="text-xs font-medium text-emerald-700 dark:text-emerald-400">Sedang Aktif Berjalan</div>
			<div class="mt-1.5 text-2xl font-bold text-emerald-800 dark:text-emerald-300">
				{kpiMetrics.active}
			</div>
			<div class="mt-1 text-[11px] text-emerald-600/80 dark:text-emerald-400/70">Dapat diklaim di kasir hari ini</div>
		</div>

		<div class="rounded-xl border border-amber-200 bg-amber-50/40 p-4 shadow-sm dark:border-amber-950 dark:bg-amber-950/20">
			<div class="text-xs font-medium text-amber-700 dark:text-amber-400">Flash Sale Berkuota</div>
			<div class="mt-1.5 text-2xl font-bold text-amber-800 dark:text-amber-300">
				{kpiMetrics.flashSale}
			</div>
			<div class="mt-1 text-[11px] text-amber-600/80 dark:text-amber-400/70">Promo dengan limit unit</div>
		</div>

		<div class="rounded-xl border border-blue-200 bg-blue-50/40 p-4 shadow-sm dark:border-blue-950 dark:bg-blue-950/20">
			<div class="text-xs font-medium text-blue-700 dark:text-blue-400">Unit Promo Terklaim</div>
			<div class="mt-1.5 text-2xl font-bold text-blue-800 dark:text-blue-300">
				{kpiMetrics.totalClaimed}
			</div>
			<div class="mt-1 text-[11px] text-blue-600/80 dark:text-blue-400/70">Total barang terjual harga promo</div>
		</div>
	</div>

	<!-- ── Toolbar Filter & Pencarian ──────────────────────────────────────── -->
	<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900 space-y-4">
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
			<div>
				<label for="filter-loc" class="mb-1 block text-xs font-medium text-neutral-700 dark:text-neutral-300">
					Filter Cabang
				</label>
				<Select2
					id="filter-loc"
					bind:value={selectedLocationId}
					options={locationOptions}
					placeholder="Pilih cabang..."
				/>
			</div>

			<div>
				<label for="filter-prod" class="mb-1 block text-xs font-medium text-neutral-700 dark:text-neutral-300">
					Filter Produk
				</label>
				<Select2
					id="filter-prod"
					bind:value={selectedProductId}
					options={productOptions}
					placeholder="Pilih produk..."
				/>
			</div>

			<div>
				<label for="search-promo" class="mb-1 block text-xs font-medium text-neutral-700 dark:text-neutral-300">
					Cari Nama Promo / Alasan
				</label>
				<SearchInput
					bind:value={searchQuery}
					placeholder="Cari nama kampanye, SKU, produk..."
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
					onclick={() => { selectedStatusFilter = 'active'; currentPage = 1; }}
					class="rounded-lg px-3 py-1.5 text-xs font-medium transition {selectedStatusFilter === 'active'
						? 'bg-emerald-600 text-white'
						: 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-950/40 dark:text-emerald-400 dark:hover:bg-emerald-950/70'}"
				>
					Sedang Aktif ({kpiMetrics.active})
				</button>

				<button
					type="button"
					onclick={() => { selectedStatusFilter = 'exhausted'; currentPage = 1; }}
					class="rounded-lg px-3 py-1.5 text-xs font-medium transition {selectedStatusFilter === 'exhausted'
						? 'bg-amber-600 text-white'
						: 'bg-amber-50 text-amber-700 hover:bg-amber-100 dark:bg-amber-950/40 dark:text-amber-400 dark:hover:bg-amber-950/70'}"
				>
					Kuota Habis
				</button>

				<button
					type="button"
					onclick={() => { selectedStatusFilter = 'inactive'; currentPage = 1; }}
					class="rounded-lg px-3 py-1.5 text-xs font-medium transition {selectedStatusFilter === 'inactive'
						? 'bg-neutral-700 text-white'
						: 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-400 dark:hover:bg-neutral-700'}"
				>
					Kadaluarsa / Nonaktif ({kpiMetrics.expiredOrInactive})
				</button>
			</div>

			<div class="text-xs text-neutral-500">
				Menampilkan <span class="font-semibold text-neutral-900 dark:text-white">{filteredOverrides.length}</span> promo
			</div>
		</div>
	</div>

	<!-- ── Alert Error Global ─────────────────────────────────────────────── -->
	{#if error}
		<Alert variant="error" title="Gagal Memuat Data">
			{error}
		</Alert>
	{/if}

	<!-- ── Tabel Daftar Promo Cabang ──────────────────────────────────────── -->
	<div class="rounded-xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
		<Table>
			<thead>
				<tr>
					<th>Kampanye & Alasan</th>
					<th>Produk & SKU</th>
					<th>Cabang Berlaku</th>
					<th>Harga Promo & Diskon</th>
					<th>Kuota & Klaim</th>
					<th>Periode Berlaku</th>
					<th>Status</th>
					<th class="text-right">Aksi</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="8" class="py-12 text-center">
							<div class="flex flex-col items-center justify-center gap-2">
								<svg class="h-6 w-6 animate-spin text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
								</svg>
								<span class="text-xs text-neutral-500">Memuat daftar promo cabang...</span>
							</div>
						</td>
					</tr>
				{:else if paginatedOverrides.length === 0}
					<tr>
						<td colspan="8" class="py-12 text-center">
							<div class="flex flex-col items-center justify-center gap-2 text-neutral-500 dark:text-neutral-400">
								<svg class="h-10 w-10 text-neutral-300 dark:text-neutral-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3zM6 6h.008v.008H6V6z" />
								</svg>
								<div class="text-sm font-medium text-neutral-700 dark:text-neutral-300">Belum ada promo yang sesuai kriteria</div>
								<div class="text-xs">Klik tombol "+ Buat Promo Cabang Baru" untuk menambahkan promo harga khusus pertama.</div>
							</div>
						</td>
					</tr>
				{:else}
					{#each paginatedOverrides as item (item.id)}
						{@const stBadge = getPromoStatusBadge(item)}
						{@const base = item.base_price ?? item.promotional_price}
						{@const discount = base - item.promotional_price}
						{@const discountPct = base > 0 ? Math.round((discount / base) * 100) : 0}

						<tr class="hover:bg-neutral-50/60 dark:hover:bg-neutral-800/40 transition">
							<!-- Kampanye / Alasan Promo -->
							<td>
								<div class="text-sm font-semibold text-neutral-900 dark:text-neutral-100">
									{item.reason || 'Promo Khusus Cabang'}
								</div>
								<div class="text-[11px] font-mono text-neutral-400">
									ID: {item.id.slice(0, 8)}...
								</div>
							</td>

							<!-- Produk & SKU -->
							<td>
								<div class="text-sm font-medium text-neutral-900 dark:text-neutral-100">
									{item.product_name ?? 'Produk'}
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

							<!-- Cabang Berlaku -->
							<td>
								<div class="text-sm text-neutral-800 dark:text-neutral-200">
									{item.location_name ?? 'Semua Cabang'}
								</div>
								{#if item.location_code}
									<div class="text-[11px] font-mono text-neutral-400">
										Kode: {item.location_code}
									</div>
								{/if}
							</td>

							<!-- Harga Promo & Diskon -->
							<td>
								<div class="font-bold text-emerald-600 dark:text-emerald-400">
									{formatRupiah(item.promotional_price)}
								</div>
								{#if discount > 0}
									<div class="mt-0.5 flex items-center gap-1.5 text-xs">
										<span class="text-neutral-400 line-through">{formatRupiah(base)}</span>
										<Badge variant="danger" size="sm">-{discountPct}%</Badge>
									</div>
								{/if}
							</td>

							<!-- Kuota & Klaim -->
							<td>
								{#if item.max_quantity != null}
									<div class="text-xs">
										<div class="flex justify-between font-medium">
											<span>{item.claimed_quantity} / {item.max_quantity} unit</span>
											<span class="text-neutral-400">{Math.round((item.claimed_quantity / item.max_quantity) * 100)}%</span>
										</div>
										<!-- Mini progress bar -->
										<div class="mt-1 h-1.5 w-28 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
											<div
												class="h-full bg-indigo-500 rounded-full"
												style="width: {Math.min(100, Math.round((item.claimed_quantity / item.max_quantity) * 100))}%"
											></div>
										</div>
									</div>
								{:else}
									<div class="text-xs text-neutral-500">
										<div>{item.claimed_quantity} unit terjual</div>
										<span class="text-[11px] text-neutral-400">(Tanpa Batas Kuota)</span>
									</div>
								{/if}
							</td>

							<!-- Periode Berlaku -->
							<td class="text-xs text-neutral-600 dark:text-neutral-400">
								<div>{formatDate(item.start_date)}</div>
								<div class="text-neutral-400">s/d {formatDate(item.end_date)}</div>
							</td>

							<!-- Status Badge -->
							<td>
								<Badge variant={stBadge.variant}>{stBadge.label}</Badge>
							</td>

							<!-- Aksi Dropdown -->
							<td class="text-right whitespace-nowrap">
								<ActionMenu
									items={[
										{
											label: 'Simulasi Kasir POS',
											icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M15.75 15.75V18m-7.5-6.75h.008v.008H8.25v-.008zm0 3h.008v.008H8.25v-.008zm0 3h.008v.008H8.25v-.008zm3-6h.008v.008H11.25v-.008zm0 3h.008v.008H11.25v-.008zm0 3h.008v.008H11.25v-.008zm3-6h.008v.008H14.25v-.008zm0 3h.008v.008H14.25v-.008zM4.5 19.5h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z" /></svg>`,
											onClick: () => populateSimulator(item.product_id, item.location_id)
										},
										...(item.is_active
											? [
													{
														label: 'Nonaktifkan Promo',
														icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" /></svg>`,
														variant: 'danger' as const,
														divider: true,
														onClick: () => openDeactivateModal(item)
													}
												]
											: [])
									]}
								/>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</Table>

		<!-- Pagination Footer -->
		{#if filteredOverrides.length > 0}
			<div class="border-t border-neutral-200 p-4 dark:border-neutral-800">
				<Pagination
					page={currentPage}
					{totalPages}
					totalItems={filteredOverrides.length}
					limit={pageSize}
					onPageChange={(p) => (currentPage = p)}
				/>
			</div>
		{/if}
	</div>
</div>

<!-- ── Modal: Buat Promo Cabang Baru ────────────────────────────────────────── -->
<Modal
	bind:open={showCreateModal}
	title="Buat Promo & Harga Khusus Cabang Baru"
	size="xl"
>
	<div class="space-y-4">
		<div class="rounded-lg bg-neutral-50 p-3 text-xs text-neutral-600 dark:bg-neutral-800/60 dark:text-neutral-300">
			<div class="font-semibold text-neutral-800 dark:text-neutral-100">Prinsip Invariant Anti-Tabrakan Promo:</div>
			<p class="mt-0.5">
				Sistem ERP menjamin bahwa satu produk pada cabang yang sama hanya dapat memiliki satu promo aktif di rentang waktu yang bersamaan.
			</p>
		</div>

		{#if createError}
			<Alert variant="error" title="Gagal Menyimpan Promo">
				{createError}
			</Alert>
		{/if}

		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
			<div>
				<label for="create-location" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
					Cabang / Toko Berlaku <span class="text-rose-500">*</span>
				</label>
				<Select2
					id="create-location"
					bind:value={newLocationId}
					options={formLocationOptions}
					placeholder="Pilih cabang..."
				/>
			</div>

			<div>
				<label for="create-product" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
					Produk yang Dipromokan <span class="text-rose-500">*</span>
				</label>
				<Select2
					id="create-product"
					bind:value={newProductId}
					options={formProductOptions}
					placeholder="Pilih produk..."
				/>
			</div>
		</div>

		<!-- Input Harga Promo & Live Calculation Preview -->
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
			<div>
				<label for="create-price" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
					Harga Jual Promo (Rp) <span class="text-rose-500">*</span>
				</label>
				<Input
					id="create-price"
					thousandSeparator
					prefix="Rp"
					bind:value={newPromoPrice}
					placeholder="0"
					required
				/>
			</div>

			<div class="flex flex-col justify-end">
				{#if formDiscountPreview}
					<div class="rounded-lg border border-neutral-200 bg-neutral-50 p-2.5 text-xs dark:border-neutral-800 dark:bg-neutral-800/40">
						<div class="flex justify-between">
							<span class="text-neutral-500">Harga Normal:</span>
							<span class="font-medium text-neutral-700 dark:text-neutral-300">{formatRupiah(formDiscountPreview.basePrice)}</span>
						</div>
						<div class="flex justify-between mt-1">
							<span class="text-neutral-500">Potongan Diskon:</span>
							<span class="font-bold text-emerald-600 dark:text-emerald-400">
								{formatRupiah(formDiscountPreview.discountAmount)} (-{formDiscountPreview.discountPct}%)
							</span>
						</div>
					</div>
				{/if}
			</div>
		</div>

		<!-- Opsi Kuota Flash Sale -->
		<div class="rounded-lg border border-neutral-200 p-3 dark:border-neutral-800 space-y-3">
			<div class="flex items-center gap-2">
				<input
					type="checkbox"
					id="toggle-quota"
					bind:checked={isQuotaLimited}
					class="h-4 w-4 rounded border-neutral-300 text-indigo-600 focus:ring-indigo-500"
				/>
				<label for="toggle-quota" class="text-xs font-semibold text-neutral-800 dark:text-neutral-200 cursor-pointer">
					Batasi Kuota Promo (Flash Sale Berkuota)
				</label>
			</div>

			{#if isQuotaLimited}
				<div>
					<label for="create-max-qty" class="mb-1 block text-xs text-neutral-600 dark:text-neutral-400">
						Maksimal Kuantitas Promo yang Dapat Terjual (Unit):
					</label>
					<Input
						id="create-max-qty"
						type="number"
						bind:value={newMaxQty}
						placeholder="Contoh: 20"
					/>
					<p class="mt-1 text-[11px] text-neutral-500">
						Setelah kuota ini tercapai di kasir, sistem POS akan otomatis mengembalikan harga ke harga normal.
					</p>
				</div>
			{/if}
		</div>

		<!-- Rentang Waktu Promo -->
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
			<div>
				<label for="create-start-date" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
					Waktu Dimulai <span class="text-rose-500">*</span>
				</label>
				<input
					type="datetime-local"
					id="create-start-date"
					bind:value={newStartDate}
					class="w-full rounded-lg border border-neutral-300 bg-white p-2.5 text-xs text-neutral-900 transition focus:border-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100"
				/>
			</div>

			<div>
				<label for="create-end-date" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
					Waktu Berakhir <span class="text-rose-500">*</span>
				</label>
				<input
					type="datetime-local"
					id="create-end-date"
					bind:value={newEndDate}
					class="w-full rounded-lg border border-neutral-300 bg-white p-2.5 text-xs text-neutral-900 transition focus:border-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100"
				/>
			</div>
		</div>

		<!-- Alasan / Nama Kampanye -->
		<div>
			<label for="create-reason" class="mb-1 block text-xs font-semibold text-neutral-800 dark:text-neutral-200">
				Nama Kampanye / Alasan Promo <span class="text-rose-500">*</span>
			</label>
			<Input
				id="create-reason"
				type="text"
				bind:value={newReason}
				placeholder="Contoh: Promo Grand Opening Surabaya, Flash Sale Weekend Dago..."
				required
			/>
		</div>
	</div>

	{#snippet footer()}
		<div class="flex items-center justify-end gap-2.5">
			<Button
				variant="outline"
				size="sm"
				onclick={() => (showCreateModal = false)}
				disabled={createSubmitting}
			>
				Batal
			</Button>

			<Button
				variant="primary"
				size="sm"
				loading={createSubmitting}
				onclick={handleCreateSubmit}
			>
				Simpan Promo
			</Button>
		</div>
	{/snippet}
</Modal>

<!-- ── Modal: Konfirmasi Nonaktifkan Promo ───────────────────────────────────── -->
<Modal
	bind:open={showDeactivateModal}
	title="Nonaktifkan Promo Cabang"
	size="md"
>
	{#if targetDeactivateItem}
		<div class="space-y-3">
			<p class="text-sm text-neutral-600 dark:text-neutral-300">
				Apakah Anda yakin ingin menonaktifkan kampanye promo:
			</p>
			<div class="rounded-lg border border-neutral-200 bg-neutral-50 p-3 dark:border-neutral-800 dark:bg-neutral-800/40">
				<div class="font-semibold text-neutral-900 dark:text-white">
					{targetDeactivateItem.reason}
				</div>
				<div class="mt-1 text-xs text-neutral-500">
					Produk: <span class="font-medium text-neutral-700 dark:text-neutral-300">{targetDeactivateItem.product_name}</span>
				</div>
				<div class="text-xs text-neutral-500">
					Cabang: <span class="font-medium text-neutral-700 dark:text-neutral-300">{targetDeactivateItem.location_name}</span>
				</div>
			</div>
			<p class="text-xs text-neutral-500">
				Setelah dinonaktifkan, kasir POS di cabang tersebut akan langsung menagihkan harga jual normal standar.
			</p>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex items-center justify-end gap-2.5">
			<Button
				variant="outline"
				size="sm"
				onclick={() => (showDeactivateModal = false)}
				disabled={deactivateSubmitting}
			>
				Batal
			</Button>

			<Button
				variant="danger"
				size="sm"
				loading={deactivateSubmitting}
				onclick={handleDeactivateSubmit}
			>
				Ya, Nonaktifkan Promo
			</Button>
		</div>
	{/snippet}
</Modal>
