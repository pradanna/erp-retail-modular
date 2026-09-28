<script lang="ts">
	/**
	 * Halaman Laporan Persediaan & Kartu Stok (Stock Card & Valuation)
	 *
	 * Fitur:
	 * 1. Tab Kartu Stok Interaktif (Buku Besar Kronologis Mutasi per Produk & Cabang).
	 * 2. Tab Valuasi Persediaan (Ringkasan Nilai Aset & Indikator Status Stok Seluruh Produk).
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listLocations,
		listProducts,
		getStockCardReport,
		getStockValuationReport,
		listSerials,
		lookupSerialUnit
	} from '@erp/api-client';
	import type {
		LocationResponse,
		ProductResponse,
		StockCardReportResponse,
		StockValuationItemResponse,
		SerialUnitResponse,
		SerialUnitLookupResponse
	} from '@erp/types';
	import { ApiError } from '@erp/types';
	import {
		Button,
		Input,
		Select2,
		type Select2Option,
		SearchInput,
		Table,
		Pagination,
		Alert,
		Badge,
		Card,
		toast
	} from '@erp/ui';

	// ── 1. State Navigasi Tab ─────────────────────────────────────────────────
	let activeTab = $state<'card' | 'valuation' | 'tracking'>('card');

	// ── 2. Master Data Options ────────────────────────────────────────────────
	let locations = $state<LocationResponse[]>([]);
	let products = $state<ProductResponse[]>([]);
	let errorInit = $state<string | null>(null);

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

	// ── 3. State Tab 1: Kartu Stok (Stock Card) ───────────────────────────────
	let cardLocationId = $state<string>('');
	let cardProductId = $state<string>('');
	let cardStartDate = $state<string>('');
	let cardEndDate = $state<string>('');

	let stockCardReport = $state<StockCardReportResponse | null>(null);
	let loadingCard = $state(false);
	let errorCard = $state<string | null>(null);

	// ── 4. State Tab 2: Valuasi Persediaan ────────────────────────────────────
	let valLocationId = $state<string>('');
	let valuationList = $state<StockValuationItemResponse[]>([]);
	let loadingValuation = $state(false);
	let errorValuation = $state<string | null>(null);
	let searchQuery = $state('');

	// ── 5. Lifecycle ──────────────────────────────────────────────────────────
	onMount(async () => {
		const token = getToken();
		if (!token) return;

		try {
			errorInit = null;

			const [locRes, prodRes] = await Promise.all([
				listLocations(token, true),
				listProducts(token, { limit: 1000 })
			]);

			locations = locRes;
			products = prodRes.data;

			if (locRes.length > 0) {
				cardLocationId = locRes[0]?.id ?? '';
				valLocationId = locRes[0]?.id ?? '';
			}
			if (prodRes.data.length > 0) {
				cardProductId = prodRes.data[0]?.id ?? '';
			}

			// Load default kartu stok dan valuasi
			if (cardLocationId && cardProductId) {
				await loadStockCard();
			}
			await Promise.all([loadValuation(), loadTrackingList()]);
		} catch (err: unknown) {
			errorInit = err instanceof ApiError ? err.message : 'Gagal memuat data master';
		}
	});

	// ── 5. State Tab 3: Pelacakan Barang & Nomor Seri / IMEI ──────────────────
	let trackingSearchInput = $state('');
	let trackingLocationId = $state('');
	let trackingProductId = $state('');
	let trackingStatusFilter = $state<string>('all');
	let trackingSerials = $state<SerialUnitResponse[]>([]);
	let loadingTracking = $state(false);
	let errorTracking = $state<string | null>(null);

	// Unit aktif yang sedang diinspeksi secara mendalam
	let inspectedUnit = $state<SerialUnitLookupResponse | null>(null);
	let loadingInspect = $state(false);
	let inspectError = $state<string | null>(null);

	// Paginasi tabel tracking
	let trackingPage = $state(1);
	const trackingPageSize = 10;

	async function loadTrackingList() {
		const token = getToken();
		if (!token) return;
		try {
			loadingTracking = true;
			errorTracking = null;
			trackingSerials = await listSerials(token);
		} catch (err: unknown) {
			errorTracking = err instanceof ApiError ? err.message : 'Gagal memuat daftar unit serial';
		} finally {
			loadingTracking = false;
		}
	}

	async function trackSpecificSN(snToTrack: string) {
		const cleanSN = snToTrack.trim();
		if (!cleanSN) return;
		const token = getToken();
		if (!token) return;

		try {
			loadingInspect = true;
			inspectError = null;
			const res = await lookupSerialUnit(token, cleanSN);
			inspectedUnit = res;
			toast.success(`Unit SN/IMEI ${cleanSN} berhasil dilacak!`);
		} catch (err: unknown) {
			inspectError =
				err instanceof ApiError ? err.message : `Unit dengan SN/IMEI "${cleanSN}" tidak ditemukan.`;
			inspectedUnit = null;
		} finally {
			loadingInspect = false;
		}
	}

	function handleTrackingScanSubmit(e: Event) {
		e.preventDefault();
		if (!trackingSearchInput.trim()) return;
		trackSpecificSN(trackingSearchInput);
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		toast.success(`Nomor seri ${text} disalin`);
	}

	// Filtered list untuk tab tracking
	const filteredTrackingList = $derived.by(() => {
		let list = trackingSerials;
		if (trackingLocationId) {
			list = list.filter((item) => item.location_id === trackingLocationId);
		}
		if (trackingProductId) {
			list = list.filter((item) => item.product_id === trackingProductId);
		}
		if (trackingStatusFilter !== 'all') {
			list = list.filter((item) => item.status.toLowerCase() === trackingStatusFilter);
		}
		if (trackingSearchInput.trim()) {
			const q = trackingSearchInput.toLowerCase().trim();
			list = list.filter(
				(item) =>
					item.serial_number.toLowerCase().includes(q) ||
					(item.product_name ?? '').toLowerCase().includes(q) ||
					(item.product_sku ?? '').toLowerCase().includes(q) ||
					(item.location_name ?? '').toLowerCase().includes(q)
			);
		}
		return list;
	});

	const paginatedTrackingList = $derived.by(() => {
		const start = (trackingPage - 1) * trackingPageSize;
		return filteredTrackingList.slice(start, start + trackingPageSize);
	});

	const totalTrackingPages = $derived.by(() => {
		return Math.max(1, Math.ceil(filteredTrackingList.length / trackingPageSize));
	});

	// KPI Metrics untuk unit berserial
	const trackingKpi = $derived.by(() => {
		const total = trackingSerials.length;
		let tersedia = 0;
		let terjual = 0;
		let retur = 0;
		for (const u of trackingSerials) {
			const st = u.status.toLowerCase();
			if (st === 'tersedia') tersedia++;
			else if (st === 'terjual') terjual++;
			else if (st === 'retur') retur++;
		}
		return { total, tersedia, terjual, retur };
	});

	function getSerialStatusBadge(status: string): {
		label: string;
		variant: 'success' | 'info' | 'warning' | 'default';
	} {
		switch (status.toLowerCase()) {
			case 'tersedia':
				return { label: 'Tersedia di Toko', variant: 'success' };
			case 'terjual':
				return { label: 'Terjual ke Konsumen', variant: 'info' };
			case 'retur':
				return { label: 'Retur / Cacat', variant: 'warning' };
			default:
				return { label: status, variant: 'default' };
		}
	}

	async function loadStockCard() {
		const token = getToken();
		if (!token) return;

		if (!cardLocationId || !cardProductId) {
			errorCard = 'Silakan pilih Cabang dan Produk terlebih dahulu';
			return;
		}

		try {
			loadingCard = true;
			errorCard = null;

			stockCardReport = await getStockCardReport(token, {
				product_id: cardProductId,
				location_id: cardLocationId,
				start_date: cardStartDate || undefined,
				end_date: cardEndDate || undefined
			});
		} catch (err: unknown) {
			errorCard = err instanceof ApiError ? err.message : 'Gagal memuat kartu stok';
			stockCardReport = null;
		} finally {
			loadingCard = false;
		}
	}

	async function loadValuation() {
		const token = getToken();
		if (!token) return;

		try {
			loadingValuation = true;
			errorValuation = null;

			valuationList = (await getStockValuationReport(token, valLocationId || undefined)) ?? [];
		} catch (err: unknown) {
			errorValuation = err instanceof ApiError ? err.message : 'Gagal memuat valuasi stok';
		} finally {
			loadingValuation = false;
		}
	}

	function handleValuationLocationChange(locId: string) {
		valLocationId = locId;
		loadValuation();
	}

	// Filtered list untuk tab valuasi
	const filteredValuationList = $derived(
		valuationList.filter((item) => {
			if (!searchQuery.trim()) return true;
			const q = searchQuery.toLowerCase();
			return (
				item.product_name.toLowerCase().includes(q) ||
				item.product_sku.toLowerCase().includes(q) ||
				item.category_name.toLowerCase().includes(q)
			);
		})
	);

	// Ringkasan KPI Valuasi
	const totalValuationNominal = $derived(
		filteredValuationList.reduce((acc, curr) => acc + curr.total_valuation, 0)
	);
	const totalPhysicalUnits = $derived(
		filteredValuationList.reduce((acc, curr) => acc + curr.quantity, 0)
	);
	const totalLowStock = $derived(
		filteredValuationList.filter((item) => item.status === 'menipis').length
	);
	const totalOutOfStock = $derived(
		filteredValuationList.filter((item) => item.status === 'habis').length
	);

	function formatRupiah(num: number): string {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			maximumFractionDigits: 0
		}).format(num);
	}

	function formatDateTime(dtStr: string): string {
		try {
			return new Date(dtStr).toLocaleString('id-ID', {
				day: '2-digit',
				month: 'short',
				year: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return dtStr;
		}
	}

	function getMovementTypeBadge(mType: string): {
		label: string;
		variant: 'primary' | 'info' | 'warning' | 'default' | 'danger';
	} {
		switch (mType) {
			case 'stock_in':
				return { label: 'Barang Masuk', variant: 'primary' };
			case 'stock_out':
				return { label: 'Barang Keluar', variant: 'info' };
			case 'opname':
				return { label: 'Stock Opname', variant: 'warning' };
			case 'transfer_in':
				return { label: 'Transfer Masuk', variant: 'primary' };
			case 'transfer_out':
				return { label: 'Transfer Keluar', variant: 'info' };
			default:
				return { label: mType, variant: 'default' };
		}
	}
</script>

<svelte:head>
	<title>Laporan Persediaan & Kartu Stok — ERP Backoffice</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header Halaman -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-neutral-900">
				Laporan Persediaan & Kartu Stok
			</h1>
			<p class="mt-1 text-sm text-neutral-500">
				Buku besar mutasi historis per produk (Bin Card) dan valuasi aset persediaan cabang.
			</p>
		</div>
		<!-- Tab Switcher -->
		<div class="inline-flex rounded-lg border border-neutral-200 bg-white p-1 shadow-2xs">
			<button
				type="button"
				onclick={() => (activeTab = 'card')}
				class="rounded-md px-3.5 py-1.5 text-xs font-semibold transition-colors {activeTab ===
				'card'
					? 'bg-neutral-900 text-white'
					: 'text-neutral-600 hover:text-neutral-900'}"
			>
				Buku Kartu Stok
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'valuation')}
				class="rounded-md px-3.5 py-1.5 text-xs font-semibold transition-colors {activeTab ===
				'valuation'
					? 'bg-neutral-900 text-white'
					: 'text-neutral-600 hover:text-neutral-900'}"
			>
				Valuasi & Status Stok
			</button>
			<button
				type="button"
				onclick={() => {
					activeTab = 'tracking';
					if (trackingSerials.length === 0) loadTrackingList();
				}}
				class="rounded-md px-3.5 py-1.5 text-xs font-semibold transition-colors {activeTab ===
				'tracking'
					? 'bg-neutral-900 text-white'
					: 'text-neutral-600 hover:text-neutral-900'}"
			>
				Pelacakan Barang / IMEI
			</button>
		</div>
	</div>

	{#if errorInit}
		<Alert variant="error" title="Gagal Memuat Data">{errorInit}</Alert>
	{/if}

	<!-- ═══════════════════════════════════════════════════════════════════════ -->
	<!-- TAB 1: KARTU STOK (STOCK CARD LEDGER)                                   -->
	<!-- ═══════════════════════════════════════════════════════════════════════ -->
	{#if activeTab === 'card'}
		<!-- Filter Kartu Stok (Compact) -->
		<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
			<div class="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
				<div>
					<label for="card-loc" class="text-2xs mb-1 block font-semibold text-neutral-600">
						Cabang / Gudang *
					</label>
					<Select2
						id="card-loc"
						size="sm"
						options={locationOptions}
						value={cardLocationId}
						onchange={(val) => (cardLocationId = String(val))}
					/>
				</div>

				<div>
					<label for="card-prod" class="text-2xs mb-1 block font-semibold text-neutral-600">
						Produk *
					</label>
					<Select2
						id="card-prod"
						size="sm"
						options={productOptions}
						value={cardProductId}
						placeholder="Ketik nama / SKU..."
						onchange={(val) => (cardProductId = String(val))}
					/>
				</div>

				<div>
					<Input
						id="card-start"
						label="Mulai Tanggal"
						type="date"
						size="sm"
						bind:value={cardStartDate}
					/>
				</div>

				<div>
					<Input
						id="card-end"
						label="Sampai Tanggal"
						type="date"
						size="sm"
						bind:value={cardEndDate}
					/>
				</div>
			</div>

			<div class="mt-3 flex items-center justify-end">
				<Button variant="primary" size="sm" loading={loadingCard} onclick={loadStockCard}>
					<svg class="mr-1.5 h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z"
						/>
					</svg>
					Tampilkan Kartu Stok
				</Button>
			</div>
		</div>

		{#if errorCard}
			<Alert variant="error" title="Gagal Menampilkan Kartu Stok">{errorCard}</Alert>
		{/if}

		{#if loadingCard}
			<div
				class="flex h-64 items-center justify-center rounded-xl border border-neutral-200 bg-white"
			>
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
					<span>Merekonsiliasi mutasi kartu stok...</span>
				</div>
			</div>
		{:else if stockCardReport}
			<!-- KPI Ringkasan Kartu Stok -->
			<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
				<Card padding="md">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Saldo Awal</span
					>
					<div class="mt-1 text-2xl font-bold text-neutral-800">
						{stockCardReport.opening_balance}
						<span class="text-xs font-normal text-neutral-500">unit</span>
					</div>
				</Card>
				<Card padding="md">
					<span class="text-xs font-semibold tracking-wider text-emerald-600 uppercase"
						>Total Masuk (+)</span
					>
					<div class="mt-1 text-2xl font-bold text-emerald-700">
						+{stockCardReport.total_in}
						<span class="text-xs font-normal text-neutral-500">unit</span>
					</div>
				</Card>
				<Card padding="md">
					<span class="text-xs font-semibold tracking-wider text-rose-600 uppercase"
						>Total Keluar (-)</span
					>
					<div class="mt-1 text-2xl font-bold text-rose-700">
						-{stockCardReport.total_out}
						<span class="text-xs font-normal text-neutral-500">unit</span>
					</div>
				</Card>
				<Card padding="md">
					<span class="text-xs font-semibold tracking-wider text-neutral-900 uppercase"
						>Saldo Akhir Fisik</span
					>
					<div class="mt-1 text-2xl font-bold text-neutral-900">
						{stockCardReport.closing_balance}
						<span class="text-xs font-normal text-neutral-500">unit</span>
					</div>
				</Card>
			</div>

			<!-- Header Produk Kartu Stok -->
			<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-xs">
				<div class="flex flex-col gap-2 text-xs sm:flex-row sm:items-center sm:justify-between">
					<div>
						<span class="text-sm font-bold text-neutral-900">{stockCardReport.product_name}</span>
						<span class="ml-2 font-mono text-neutral-500">SKU: {stockCardReport.product_sku}</span>
					</div>
					<div class="text-neutral-600">
						Cabang: <span class="font-bold text-neutral-900">{stockCardReport.location_name}</span>
					</div>
				</div>
			</div>

			<!-- Tabel Mutasi Kronologis Kartu Stok -->
			<div class="rounded-xl border border-neutral-200 bg-white shadow-xs">
				{#if !stockCardReport.entries || stockCardReport.entries.length === 0}
					<div class="py-12 text-center text-sm text-neutral-500">
						Tidak ada pergerakan stok untuk produk ini pada periode yang dipilih.
					</div>
				{:else}
					<div class="overflow-x-auto">
						<Table borderless>
							<thead>
								<tr
									class="border-b border-neutral-200 bg-neutral-50/75 text-left text-xs font-semibold text-neutral-600"
								>
									<th class="px-5 py-3.5">Tanggal & Waktu</th>
									<th class="px-5 py-3.5">Jenis Mutasi</th>
									<th class="px-5 py-3.5">No. Dokumen</th>
									<th class="px-5 py-3.5">Surat Jalan / Memo</th>
									<th class="px-5 py-3.5">Alasan / Keperluan</th>
									<th class="px-5 py-3.5 text-right">Masuk (+)</th>
									<th class="px-5 py-3.5 text-right">Keluar (-)</th>
									<th class="px-5 py-3.5 text-right">Saldo Berjalan</th>
									<th class="px-5 py-3.5">Petugas</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-200 bg-white text-xs">
								{#each stockCardReport.entries as entry, idx (entry.document_number + '_' + idx)}
									{@const badge = getMovementTypeBadge(entry.movement_type)}
									<tr class="transition-colors hover:bg-neutral-50/80">
										<td class="px-5 py-4 text-neutral-600">
											{formatDateTime(entry.date)}
										</td>
										<td class="px-5 py-4">
											<Badge variant={badge.variant}>{badge.label}</Badge>
										</td>
										<td class="px-5 py-4 font-mono font-semibold text-neutral-900">
											{entry.document_number}
										</td>
										<td class="px-5 py-4 text-neutral-600">
											{entry.reference_number || '-'}
										</td>
										<td class="px-5 py-4 font-medium text-neutral-800">
											{entry.category_reason}
										</td>
										<td class="px-5 py-4 text-right font-bold text-emerald-600">
											{entry.in_quantity > 0 ? `+${entry.in_quantity}` : '-'}
										</td>
										<td class="px-5 py-4 text-right font-bold text-rose-600">
											{entry.out_quantity > 0 ? `-${entry.out_quantity}` : '-'}
										</td>
										<td class="px-5 py-4 text-right font-mono font-bold text-neutral-900">
											{entry.balance}
										</td>
										<td class="px-5 py-4 text-neutral-600">
											{entry.executed_by_name}
										</td>
									</tr>
								{/each}
							</tbody>
						</Table>
					</div>
				{/if}
			</div>
		{/if}
	{/if}

	<!-- ═══════════════════════════════════════════════════════════════════════ -->
	<!-- TAB 2: VALUASI & STATUS STOK CABANG                                     -->
	<!-- ═══════════════════════════════════════════════════════════════════════ -->
	{#if activeTab === 'valuation'}
		<!-- KPI Ringkasan Valuasi -->
		<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
			<Card padding="md">
				<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
					>Total Nilai Aset</span
				>
				<div class="mt-1 text-2xl font-bold text-neutral-900">
					{formatRupiah(totalValuationNominal)}
				</div>
			</Card>
			<Card padding="md">
				<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
					>Total Fisik Unit</span
				>
				<div class="mt-1 text-2xl font-bold text-neutral-800">
					{totalPhysicalUnits} <span class="text-xs font-normal text-neutral-500">unit</span>
				</div>
			</Card>
			<Card padding="md">
				<span class="text-xs font-semibold tracking-wider text-amber-600 uppercase"
					>Stok Menipis</span
				>
				<div class="mt-1 text-2xl font-bold text-amber-700">
					{totalLowStock} <span class="text-xs font-normal text-neutral-500">SKU</span>
				</div>
			</Card>
			<Card padding="md">
				<span class="text-xs font-semibold tracking-wider text-rose-600 uppercase"
					>Stok Habis (Kosong)</span
				>
				<div class="mt-1 text-2xl font-bold text-rose-700">
					{totalOutOfStock} <span class="text-xs font-normal text-neutral-500">SKU</span>
				</div>
			</Card>
		</div>

		<!-- Filter & Pencarian Valuasi (Compact) -->
		<div
			class="flex flex-wrap items-center gap-2.5 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs"
		>
			<div class="w-full sm:w-60">
				<Select2
					id="val-loc"
					size="sm"
					options={locationOptions}
					value={valLocationId}
					placeholder="Semua Cabang / Gudang"
					searchPlaceholder="Cari cabang..."
					clearable={true}
					onchange={(val) => handleValuationLocationChange(String(val))}
				/>
			</div>

			<div class="w-full sm:w-72">
				<SearchInput
					size="sm"
					placeholder="Cari produk, SKU, kategori..."
					bind:value={searchQuery}
				/>
			</div>
		</div>

		{#if errorValuation}
			<Alert variant="error" title="Gagal Memuat Valuasi">{errorValuation}</Alert>
		{/if}

		<!-- Tabel Valuasi Persediaan -->
		<div class="rounded-xl border border-neutral-200 bg-white shadow-xs">
			{#if loadingValuation}
				<div class="flex h-64 items-center justify-center">
					<div class="flex items-center gap-3 text-neutral-500">
						<svg class="h-6 w-6 animate-spin" viewBox="0 0 24 24" fill="none">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
						<span>Menghitung nilai valuasi persediaan...</span>
					</div>
				</div>
			{:else if !filteredValuationList || filteredValuationList.length === 0}
				<div class="py-12 text-center text-sm text-neutral-500">
					Tidak ada data produk yang sesuai dengan kriteria pencarian.
				</div>
			{:else}
				<div class="overflow-x-auto">
					<Table borderless>
						<thead>
							<tr
								class="border-b border-neutral-200 bg-neutral-50/75 text-left text-xs font-semibold text-neutral-600"
							>
								<th class="px-5 py-3.5">SKU</th>
								<th class="px-5 py-3.5">Nama Produk</th>
								<th class="px-5 py-3.5">Kategori</th>
								<th class="px-5 py-3.5">Cabang / Gudang</th>
								<th class="px-5 py-3.5 text-center">Stok Fisik</th>
								<th class="px-5 py-3.5 text-center">Batas Min</th>
								<th class="px-5 py-3.5 text-right">Harga Modal</th>
								<th class="px-5 py-3.5 text-right">Total Valuasi Aset</th>
								<th class="px-5 py-3.5 text-center">Status</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200 bg-white text-xs">
							{#each filteredValuationList as item (item.product_sku + '_' + (item.location_name ?? ''))}
								<tr class="transition-colors hover:bg-neutral-50/80">
									<td class="px-5 py-4 font-mono font-semibold text-neutral-900">
										{item.product_sku}
									</td>
									<td class="px-5 py-4 font-medium text-neutral-900">
										{item.product_name}
									</td>
									<td class="px-5 py-4 text-neutral-600">
										{item.category_name}
									</td>
									<td class="px-5 py-4 text-neutral-700">
										{item.location_name}
									</td>
									<td class="px-5 py-4 text-center font-bold text-neutral-900">
										{item.quantity}
									</td>
									<td class="px-5 py-4 text-center text-neutral-500">
										{item.min_stock}
									</td>
									<td class="px-5 py-4 text-right text-neutral-600">
										{formatRupiah(item.base_price)}
									</td>
									<td class="px-5 py-4 text-right font-bold text-neutral-900">
										{formatRupiah(item.total_valuation)}
									</td>
									<td class="px-5 py-4 text-center">
										{#if item.status === 'aman'}
											<Badge variant="success">Aman</Badge>
										{:else if item.status === 'menipis'}
											<Badge variant="warning">Menipis</Badge>
										{:else}
											<Badge variant="danger">Kosong</Badge>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</Table>
				</div>
			{/if}
		</div>
	{/if}

	<!-- ═══════════════════════════════════════════════════════════════════════ -->
	<!-- TAB 3: PELACAKAN BARANG & NOMOR SERI / IMEI (UNIT AUDIT TRAIL)          -->
	<!-- ═══════════════════════════════════════════════════════════════════════ -->
	{#if activeTab === 'tracking'}
		<!-- ── Hero: Fast Scanner & Serial Lookup Card ──────────────────────── -->
		<div class="rounded-xl border border-neutral-200 bg-white p-5 shadow-2xs">
			<div class="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
				<!-- Form Input Scan SN / IMEI -->
				<div class="flex-1 space-y-3">
					<div class="flex items-center gap-2">
						<div
							class="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200"
						>
							<svg class="h-4.5 w-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z"
								/>
							</svg>
						</div>
						<div>
							<h2 class="text-base font-semibold text-neutral-900">
								Fast Barcode, S/N & IMEI Scanner
							</h2>
							<p class="text-xs text-neutral-500">
								Tembakkan laser scanner barcode atau ketik nomor seri/IMEI untuk melacak riwayat
								mutasi fisik, lokasi saat ini, dan status garansi.
							</p>
						</div>
					</div>

					<form
						onsubmit={handleTrackingScanSubmit}
						class="flex flex-col items-center gap-2 pt-1 sm:flex-row"
					>
						<div class="relative w-full flex-1">
							<div
								class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-neutral-400"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
									/>
								</svg>
							</div>
							<input
								type="text"
								bind:value={trackingSearchInput}
								placeholder="Pindai barcode S/N / IMEI (contoh: SN-SAM-S24U-256-GRY-0101)..."
								class="w-full rounded-lg border border-neutral-300 bg-white py-2.5 pr-10 pl-10 font-mono text-sm text-neutral-900 placeholder-neutral-400 transition placeholder:font-sans focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden"
							/>
							{#if trackingSearchInput}
								<button
									type="button"
									onclick={() => (trackingSearchInput = '')}
									class="absolute inset-y-0 right-0 flex items-center pr-3 text-neutral-400 hover:text-neutral-700"
									aria-label="Hapus teks"
								>
									<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M6 18L18 6M6 6l12 12"
										/>
									</svg>
								</button>
							{/if}
						</div>

						<Button
							variant="primary"
							type="submit"
							loading={loadingInspect}
							disabled={!trackingSearchInput.trim()}
							size="md"
						>
							<svg class="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M13 10V3L4 14h7v7l9-11h-7z"
								/>
							</svg>
							Lacak Unit
						</Button>
					</form>

					{#if inspectError}
						<div
							class="mt-2 rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
						>
							<div class="flex items-center gap-2">
								<svg
									class="h-4 w-4 shrink-0 text-rose-600"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
									/>
								</svg>
								<span>{inspectError}</span>
							</div>
						</div>
					{/if}
				</div>

				<!-- Kartu Paspor Identitas Fisik Unit Terpilih -->
				{#if inspectedUnit}
					{@const stBadge = getSerialStatusBadge(inspectedUnit.serial_unit.status)}
					<div
						class="w-full rounded-xl border border-neutral-200 bg-neutral-50/80 p-4 shadow-2xs transition-all lg:w-96"
					>
						<div class="flex items-start justify-between">
							<div>
								<div class="text-[11px] font-semibold tracking-wider text-neutral-500 uppercase">
									Paspor Unit Fisik
								</div>
								<div class="mt-0.5 font-mono text-base font-bold text-neutral-900">
									{inspectedUnit.serial_unit.serial_number}
								</div>
							</div>
							<Badge variant={stBadge.variant}>{stBadge.label}</Badge>
						</div>

						<div class="mt-3 space-y-1.5 border-t border-neutral-200 pt-2.5 text-xs">
							<div class="flex justify-between">
								<span class="text-neutral-500">Nama Produk:</span>
								<span class="text-right font-medium text-neutral-900"
									>{inspectedUnit.product_name}</span
								>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-500">SKU / Brand:</span>
								<span class="font-mono text-neutral-700"
									>{inspectedUnit.product_sku}
									{inspectedUnit.product_brand ? `(${inspectedUnit.product_brand})` : ''}</span
								>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-500">Lokasi Fisik Terkini:</span>
								<span class="font-medium text-neutral-900"
									>{inspectedUnit.location_name} ({inspectedUnit.location_code})</span
								>
							</div>
							<div class="flex justify-between">
								<span class="text-neutral-500">Tercatat Masuk:</span>
								<span class="text-neutral-700"
									>{formatDateTime(inspectedUnit.serial_unit.created_at)}</span
								>
							</div>
						</div>

						<!-- Timeline Lifecycle Unit -->
						<div class="mt-3.5 border-t border-neutral-200 pt-2.5">
							<div class="mb-2 text-[11px] font-semibold tracking-wider text-neutral-500 uppercase">
								Jejak Siklus Hidup (Audit Trail)
							</div>
							<div class="space-y-2 text-xs">
								<div class="flex items-start gap-2">
									<div class="mt-0.5 h-2 w-2 rounded-full bg-emerald-500"></div>
									<div>
										<span class="font-medium text-neutral-900">Registrasi Fisik di Gudang</span>
										<div class="text-[11px] text-neutral-500">
											{formatDateTime(inspectedUnit.serial_unit.created_at)}
										</div>
									</div>
								</div>
								<div class="flex items-start gap-2">
									<div class="mt-0.5 h-2 w-2 rounded-full bg-indigo-500"></div>
									<div>
										<span class="font-medium text-neutral-900"
											>Alokasi Cabang: {inspectedUnit.location_name}</span
										>
										<div class="text-[11px] text-neutral-500">Status saat ini: {stBadge.label}</div>
									</div>
								</div>
							</div>
						</div>

						<div class="mt-3.5 flex items-center gap-2 border-t border-neutral-200 pt-2.5">
							<Button
								variant="secondary"
								size="sm"
								fullWidth
								onclick={() => copyToClipboard(inspectedUnit!.serial_unit.serial_number)}
							>
								Salin Nomor Seri
							</Button>
							<Button variant="outline" size="sm" onclick={() => (inspectedUnit = null)}>
								Tutup
							</Button>
						</div>
					</div>
				{/if}
			</div>
		</div>

		<!-- ── KPI Metric Cards Unit Berserial ─────────────────────────────────── -->
		<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
			<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
				<div class="text-xs font-medium text-neutral-500">Total Unit Terdaftar</div>
				<div class="mt-1.5 text-2xl font-bold text-neutral-900">{trackingKpi.total}</div>
				<div class="mt-1 text-[11px] text-neutral-400">Unit fisik di sistem</div>
			</div>

			<div class="rounded-xl border border-emerald-200 bg-emerald-50/50 p-3.5 shadow-2xs">
				<div class="text-xs font-medium text-emerald-700">Tersedia di Toko</div>
				<div class="mt-1.5 text-2xl font-bold text-emerald-800">{trackingKpi.tersedia}</div>
				<div class="mt-1 text-[11px] text-emerald-600/80">Siap dijual / dimutasi</div>
			</div>

			<div class="rounded-xl border border-blue-200 bg-blue-50/50 p-3.5 shadow-2xs">
				<div class="text-xs font-medium text-blue-700">Terjual ke Konsumen</div>
				<div class="mt-1.5 text-2xl font-bold text-blue-800">{trackingKpi.terjual}</div>
				<div class="mt-1 text-[11px] text-blue-600/80">Masa garansi aktif</div>
			</div>

			<div class="rounded-xl border border-amber-200 bg-amber-50/50 p-3.5 shadow-2xs">
				<div class="text-xs font-medium text-amber-700">Retur & Cacat</div>
				<div class="mt-1.5 text-2xl font-bold text-amber-800">{trackingKpi.retur}</div>
				<div class="mt-1 text-[11px] text-amber-600/80">Proses inspeksi / klaim</div>
			</div>
		</div>

		<!-- ── Toolbar Filter Penjelajahan Unit (Compact) ───────────────────────── -->
		<div class="space-y-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
			<div class="flex flex-wrap items-center gap-2">
				<div class="w-full sm:w-52">
					<Select2
						id="track-loc"
						size="sm"
						bind:value={trackingLocationId}
						options={locationOptions}
						placeholder="Semua Lokasi & Cabang"
						searchPlaceholder="Cari cabang..."
						clearable={true}
						onchange={() => (trackingPage = 1)}
					/>
				</div>

				<div class="w-full sm:w-56">
					<Select2
						id="track-prod"
						size="sm"
						bind:value={trackingProductId}
						options={productOptions}
						placeholder="Semua Produk Berserial"
						searchPlaceholder="Cari produk..."
						clearable={true}
						onchange={() => (trackingPage = 1)}
					/>
				</div>

				<div class="w-full sm:w-64">
					<SearchInput
						size="sm"
						bind:value={trackingSearchInput}
						placeholder="Cari S/N, IMEI, SKU, produk..."
						onsearch={() => (trackingPage = 1)}
					/>
				</div>

				{#if trackingLocationId || trackingProductId || trackingSearchInput || trackingStatusFilter !== 'all'}
					<button
						type="button"
						onclick={() => {
							trackingLocationId = '';
							trackingProductId = '';
							trackingSearchInput = '';
							trackingStatusFilter = 'all';
							trackingPage = 1;
						}}
						class="px-2 py-1 text-xs text-neutral-500 hover:text-neutral-900 hover:underline"
					>
						Reset Filter
					</button>
				{/if}
			</div>

			<!-- Status Filter Tabs (Compact Pills) -->
			<div
				class="flex flex-wrap items-center justify-between gap-2.5 border-t border-neutral-100 pt-2.5"
			>
				<div class="flex flex-wrap items-center gap-1.5">
					<span class="mr-1 text-xs font-medium text-neutral-500">Status:</span>

					<button
						type="button"
						onclick={() => {
							trackingStatusFilter = 'all';
							trackingPage = 1;
						}}
						class="rounded-lg px-2.5 py-1 text-xs font-medium transition {trackingStatusFilter ===
						'all'
							? 'bg-neutral-900 text-white shadow-2xs'
							: 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200 hover:text-neutral-900'}"
					>
						Semua ({trackingKpi.total})
					</button>

					<button
						type="button"
						onclick={() => {
							trackingStatusFilter = 'tersedia';
							trackingPage = 1;
						}}
						class="rounded-lg px-2.5 py-1 text-xs font-medium transition {trackingStatusFilter ===
						'tersedia'
							? 'bg-emerald-600 text-white shadow-2xs'
							: 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100'}"
					>
						Tersedia ({trackingKpi.tersedia})
					</button>

					<button
						type="button"
						onclick={() => {
							trackingStatusFilter = 'terjual';
							trackingPage = 1;
						}}
						class="rounded-lg px-2.5 py-1 text-xs font-medium transition {trackingStatusFilter ===
						'terjual'
							? 'bg-blue-600 text-white shadow-2xs'
							: 'bg-blue-50 text-blue-700 hover:bg-blue-100'}"
					>
						Terjual ({trackingKpi.terjual})
					</button>

					<button
						type="button"
						onclick={() => {
							trackingStatusFilter = 'retur';
							trackingPage = 1;
						}}
						class="rounded-lg px-2.5 py-1 text-xs font-medium transition {trackingStatusFilter ===
						'retur'
							? 'bg-amber-600 text-white shadow-2xs'
							: 'bg-amber-50 text-amber-700 hover:bg-amber-100'}"
					>
						Retur / Klaim ({trackingKpi.retur})
					</button>
				</div>

				<div class="text-xs text-neutral-500">
					Menampilkan <span class="font-semibold text-neutral-900"
						>{filteredTrackingList.length}</span
					>
					unit
				</div>
			</div>
		</div>

		{#if errorTracking}
			<Alert variant="error" title="Gagal Memuat Data Pelacakan">{errorTracking}</Alert>
		{/if}

		<!-- ── Tabel Daftar Audit Serial & IMEI ───────────────────────────────── -->
		<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
			{#if loadingTracking}
				<div class="flex h-64 items-center justify-center">
					<div class="flex items-center gap-3 text-neutral-500">
						<svg class="h-6 w-6 animate-spin" viewBox="0 0 24 24" fill="none">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
						<span>Memuat database unit fisik serial & IMEI...</span>
					</div>
				</div>
			{:else if paginatedTrackingList.length === 0}
				<div class="py-12 text-center text-sm text-neutral-500">
					Tidak ada data nomor seri atau IMEI yang sesuai dengan kriteria filter.
				</div>
			{:else}
				<div class="overflow-x-auto">
					<Table borderless>
						<thead>
							<tr
								class="border-b border-neutral-200 bg-neutral-50/75 text-left text-xs font-semibold text-neutral-600"
							>
								<th class="px-5 py-3.5">Nomor Seri / IMEI</th>
								<th class="px-5 py-3.5">Model Produk & SKU</th>
								<th class="px-5 py-3.5">Lokasi Fisik Terkini</th>
								<th class="px-5 py-3.5 text-center">Status Siklus</th>
								<th class="px-5 py-3.5">Tercatat Masuk</th>
								<th class="px-5 py-3.5 text-right">Aksi</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200 bg-white text-xs">
							{#each paginatedTrackingList as unit (unit.id)}
								{@const st = getSerialStatusBadge(unit.status)}
								<tr class="transition-colors hover:bg-neutral-50/80">
									<td class="px-5 py-4 font-mono font-bold text-neutral-900">
										<div class="flex items-center gap-1.5">
											<span>{unit.serial_number}</span>
											<button
												type="button"
												onclick={() => copyToClipboard(unit.serial_number)}
												class="rounded p-1 text-neutral-400 transition hover:bg-neutral-100 hover:text-neutral-700"
												title="Salin nomor seri"
											>
												<svg
													class="h-3.5 w-3.5"
													fill="none"
													viewBox="0 0 24 24"
													stroke="currentColor"
												>
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"
													/>
												</svg>
											</button>
										</div>
									</td>
									<td class="px-5 py-4 font-medium text-neutral-900">
										<div>{unit.product_name ?? 'Produk'}</div>
										<div class="mt-1 font-mono text-[11px] text-neutral-500">
											SKU: {unit.product_sku ?? unit.product_id}
											{#if unit.product_brand}
												• {unit.product_brand}{/if}
										</div>
									</td>
									<td class="px-5 py-4 text-neutral-700">
										<div class="flex items-center gap-1.5">
											<svg
												class="h-4 w-4 text-neutral-400"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
												/>
											</svg>
											<span class="font-medium">{unit.location_name ?? 'Gudang'}</span>
										</div>
										{#if unit.location_code}
											<div class="pl-5.5 font-mono text-[11px] text-neutral-400">
												Kode: {unit.location_code}
											</div>
										{/if}
									</td>
									<td class="px-5 py-4 text-center">
										<Badge variant={st.variant}>{st.label}</Badge>
									</td>
									<td class="px-5 py-4 text-neutral-500">
										{formatDateTime(unit.created_at)}
									</td>
									<td class="px-5 py-4 text-right">
										<Button
											variant="outline"
											size="sm"
											onclick={() => {
												trackingSearchInput = unit.serial_number;
												trackSpecificSN(unit.serial_number);
											}}
										>
											<svg
												class="mr-1 h-3.5 w-3.5 text-neutral-600"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
												/>
											</svg>
											Lacak Jejak
										</Button>
									</td>
								</tr>
							{/each}
						</tbody>
					</Table>
				</div>

				<!-- Pagination Footer -->
				{#if filteredTrackingList.length > 0}
					<div class="border-t border-neutral-200 p-4">
						<Pagination
							page={trackingPage}
							totalPages={totalTrackingPages}
							totalItems={filteredTrackingList.length}
							limit={trackingPageSize}
							onPageChange={(p) => (trackingPage = p)}
						/>
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</div>
