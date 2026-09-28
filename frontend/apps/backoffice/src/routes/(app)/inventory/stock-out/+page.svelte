<script lang="ts">
	/**
	 * Halaman Transaksi Barang Keluar (Outbound / Stock Out)
	 *
	 * Memungkinkan staf gudang/admin mencatat pengeluaran barang untuk barang rusak/scrap,
	 * pemakaian internal kantor, sampel display, atau penjualan bebas non-kasir.
	 */
	import { onMount, untrack } from 'svelte';
	import { getToken, getUser } from '$lib/stores/auth.svelte';
	import { listLocations, listProducts, listCategories, listStockMovements } from '@erp/api-client';
	import type {
		LocationResponse,
		ProductResponse,
		CategoryResponse,
		StockMovementResponse
	} from '@erp/types';
	import { ApiError } from '@erp/types';
	import {
		Button,
		Select2,
		type Select2Option,
		Table,
		Pagination,
		Alert,
		Badge,
		toast
	} from '@erp/ui';
	import {
		StockOutFormState,
		StockOutDetailState,
		getReasonLabel,
		formatDate,
		formatDateTime
	} from './stock-out.svelte';
	import StockOutCreateModal from './components/StockOutCreateModal.svelte';
	import StockOutDetailModal from './components/StockOutDetailModal.svelte';

	// ── 0. Otorisasi Peran PBAC ───────────────────────────────────────────────
	const currentUser = $derived(getUser());
	const isWarehouseStaff = $derived(currentUser?.role === 'warehouse');
	const userLocationId = $derived(currentUser?.location_id);
	const canCreateStockOut = $derived(
		currentUser?.role === 'owner' ||
			currentUser?.role === 'superadmin' ||
			currentUser?.role === 'admin' ||
			currentUser?.role === 'warehouse'
	);

	// ── 1. State Master & Daftar Transaksi ─────────────────────────────────────
	let locations = $state<LocationResponse[]>([]);
	let selectedLocationId = $state<string>('');
	let products = $state<ProductResponse[]>([]);
	let categories = $state<CategoryResponse[]>([]);
	let movements = $state<StockMovementResponse[]>([]);

	let loading = $state(true);
	let loadingList = $state(false);
	let error = $state<string | null>(null);

	function getFirstDayOfMonth(): string {
		const now = new Date();
		const year = now.getFullYear();
		const month = String(now.getMonth() + 1).padStart(2, '0');
		return `${year}-${month}-01`;
	}

	function getTodayDate(): string {
		const now = new Date();
		const year = now.getFullYear();
		const month = String(now.getMonth() + 1).padStart(2, '0');
		const day = String(now.getDate()).padStart(2, '0');
		return `${year}-${month}-${day}`;
	}

	// Pagination & Filter Tanggal
	let currentPage = $state(1);
	let pageSize = $state(10);
	let totalRecords = $state(0);
	let filterStartDate = $state<string>(getFirstDayOfMonth());
	let filterEndDate = $state<string>(getTodayDate());
	const isDefaultDate = $derived(
		filterStartDate === getFirstDayOfMonth() && filterEndDate === getTodayDate()
	);

	// ── 2. Controllers (State Reaktif) ────────────────────────────────────────
	const form = new StockOutFormState();
	const detail = new StockOutDetailState();

	// ── 3. Helper Lookup ──────────────────────────────────────────────────────
	function getProductById(id: string): ProductResponse | undefined {
		return products.find((p) => p.id === id);
	}

	function getCategoryName(catId?: string): string {
		if (!catId) return '';
		return categories.find((c) => c.id === catId)?.name || '';
	}

	const filterLocationOptions = $derived<Select2Option[]>(
		isWarehouseStaff && userLocationId
			? locations
					.filter((loc) => loc.id === userLocationId)
					.map((loc) => ({
						value: loc.id,
						label: `${loc.name} (${loc.code})`
					}))
			: [
					{ value: '', label: 'Semua Cabang / Gudang' },
					...locations.map((loc) => ({
						value: loc.id,
						label: `${loc.name} (${loc.code})`
					}))
				]
	);

	// ── 4. Lifecycle & Data Fetching ──────────────────────────────────────────
	onMount(async () => {
		const token = getToken();
		if (!token) return;

		try {
			loading = true;
			error = null;

			const [locRes, prodRes, catRes] = await Promise.all([
				listLocations(token, true),
				listProducts(token, { limit: 1000 }),
				listCategories(token)
			]);

			if (isWarehouseStaff && userLocationId) {
				locations = locRes.filter((l) => l.id === userLocationId);
				selectedLocationId = userLocationId;
			} else {
				locations = locRes;
				if (locRes.length > 0) {
					selectedLocationId = locRes[0]?.id ?? '';
				}
			}

			await loadMovements();
		} catch (err: unknown) {
			error = err instanceof ApiError ? err.message : 'Gagal memuat data awal';
		} finally {
			loading = false;
			initialLoaded = true;
		}
	});

	// Reaktif Filter Otomatis (State-Based)
	let initialLoaded = $state(false);
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;

	$effect(() => {
		void selectedLocationId;
		void filterStartDate;
		void filterEndDate;

		if (!initialLoaded) return;

		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => {
			untrack(() => {
				currentPage = 1;
				loadMovements();
			});
		}, 150);

		return () => {
			clearTimeout(debounceTimer);
		};
	});

	async function loadMovements() {
		const token = getToken();
		if (!token) return;

		try {
			loadingList = true;
			const res = await listStockMovements(token, {
				location_id: selectedLocationId || undefined,
				type: 'out',
				start_date: filterStartDate || undefined,
				end_date: filterEndDate || undefined,
				page: currentPage,
				limit: pageSize
			});
			movements = res.data ?? [];
			totalRecords = res.total ?? 0;
		} catch (err: unknown) {
			toast.error(err instanceof ApiError ? err.message : 'Gagal memuat riwayat barang keluar');
		} finally {
			loadingList = false;
		}
	}

	function handleResetDateFilter() {
		filterStartDate = getFirstDayOfMonth();
		filterEndDate = getTodayDate();
	}

	function handlePageChange(newPage: number) {
		currentPage = newPage;
		loadMovements();
	}
</script>

<div class="space-y-6">
	<!-- Header Halaman & Tombol Aksi Utama -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="mb-1 flex items-center gap-2 text-xs text-neutral-500">
				<span>Inventori</span>
				<span>/</span>
				<span class="font-medium text-neutral-800">Barang Keluar (Outbound)</span>
			</div>
			<h1 class="text-xl font-bold tracking-tight text-neutral-900">
				Pengeluaran Barang Fisik (Stock Out)
			</h1>
			<p class="mt-0.5 text-xs text-neutral-500">
				Pencatatan pengeluaran barang fisik untuk rusak/scrap, pemakaian kantor, sampel display,
				atau penjualan non-kasir.
			</p>
		</div>

		{#if canCreateStockOut}
			<Button
				variant="primary"
				size="md"
				onclick={() => form.openModal(selectedLocationId || locations[0]?.id || '')}
			>
				<svg class="mr-1.5 h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 4.5v15m7.5-7.5h-15"
					/>
				</svg>
				Catat Barang Keluar
			</Button>
		{/if}
	</div>

	{#if error}
		<Alert variant="error" title="Terjadi Kesalahan">{error}</Alert>
	{/if}

	<!-- Filter Toolbar: Lokasi & Rentang Tanggal (Compact & Rapi) -->
	<div
		class="flex flex-col gap-2.5 rounded-xl border border-neutral-200 bg-white p-3 shadow-2xs sm:flex-row sm:items-center sm:justify-between"
	>
		<div class="flex flex-wrap items-center gap-2">
			<!-- Filter Cabang / Gudang -->
			<div class="w-full sm:w-56">
				{#if isWarehouseStaff && userLocationId}
					<div
						class="flex h-8 items-center gap-1.5 rounded-lg border border-neutral-200 bg-neutral-50 px-2.5 text-xs font-semibold text-neutral-800"
						title="Lokasi penugasan gudang Anda"
					>
						<svg class="h-3.5 w-3.5 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
						</svg>
						<span class="truncate">{locations.find(l => l.id === userLocationId)?.name || 'Gudang Anda'}</span>
					</div>
				{:else}
					<Select2
						id="filter-loc"
						size="sm"
						options={filterLocationOptions}
						bind:value={selectedLocationId}
						placeholder="Semua Cabang / Gudang"
						searchPlaceholder="Cari nama atau kode gudang..."
					/>
				{/if}
			</div>

			<!-- Rentang Tanggal Terpadu (Single Unified Capsule - Compact) -->
			<div
				class="flex h-8 items-center gap-1.5 rounded-lg border border-neutral-300 bg-white px-2.5 shadow-2xs transition-all focus-within:border-neutral-900 focus-within:ring-2 focus-within:ring-neutral-900/10"
			>
				<svg
					class="h-3.5 w-3.5 shrink-0 text-neutral-400"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M6.75 3v2.25M17.25 3v2.253 3.75m3 0a1.5 1.5 0 0 1-1.5 1.5H5.25a1.5 1.5 0 0 1-1.5-1.5m16.5 0v11.25A2.25 2.25 0 0 1 18 21H6a2.25 2.25 0 0 1-2.25-2.25V7.5"
					/>
				</svg>
				<span class="text-2xs font-medium text-neutral-400">Periode:</span>
				<input
					type="date"
					id="filter-start"
					aria-label="Dari Tanggal"
					title="Dari Tanggal"
					bind:value={filterStartDate}
					class="w-28 cursor-pointer bg-transparent font-mono text-xs text-neutral-800 focus:outline-hidden sm:w-30"
				/>
				<span class="text-2xs font-medium text-neutral-400">s/d</span>
				<input
					type="date"
					id="filter-end"
					aria-label="Sampai Tanggal"
					title="Sampai Tanggal"
					bind:value={filterEndDate}
					class="w-28 cursor-pointer bg-transparent font-mono text-xs text-neutral-800 focus:outline-hidden sm:w-30"
				/>
			</div>

			<!-- Tombol Reset Tanggal -->
			<Button
				variant="outline"
				size="sm"
				disabled={isDefaultDate}
				onclick={handleResetDateFilter}
				title="Kembalikan rentang tanggal ke awal bulan ini"
			>
				<svg
					class="h-3.5 w-3.5 text-neutral-500"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99"
					/>
				</svg>
				<span>Reset</span>
			</Button>
		</div>

		<!-- Status Ringkasan Transaksi di Kanan -->
		<div class="flex items-center gap-2 text-xs text-neutral-500">
			<span class="inline-block h-2 w-2 rounded-full bg-emerald-500"></span>
			<span>Total Riwayat:</span>
			<span class="font-semibold text-neutral-900">{totalRecords} dokumen</span>
		</div>
	</div>

	<!-- Tabel Riwayat Dokumen Barang Keluar -->
	<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
		<div class="flex items-center justify-between border-b border-neutral-200 px-4 py-3">
			<div class="flex items-center gap-2">
				<h2 class="text-sm font-semibold text-neutral-800">Daftar Dokumen Barang Keluar</h2>
				<Badge variant="default">{totalRecords} Total Dokumen</Badge>
			</div>
			{#if loadingList}
				<span class="text-2xs animate-pulse text-neutral-500">Memperbarui data...</span>
			{/if}
		</div>

		{#if loading}
			<div class="flex h-64 items-center justify-center">
				<div class="flex items-center gap-2 text-neutral-500">
					<svg class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						></path>
					</svg>
					<span class="text-xs">Memuat riwayat transaksi...</span>
				</div>
			</div>
		{:else if movements.length === 0}
			<div class="p-8 text-center">
				<div
					class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-neutral-100 text-neutral-400"
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
							d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
						/>
					</svg>
				</div>
				<h3 class="text-sm font-semibold text-neutral-800">Tidak Ada Data Barang Keluar</h3>
				<p class="mt-1 text-xs text-neutral-500">
					Belum ada catatan pengeluaran barang untuk filter cabang dan periode tanggal yang dipilih.
				</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<Table>
					<thead>
						<tr
							class="border-b border-neutral-200 bg-neutral-50/80 text-left text-xs font-semibold text-neutral-600"
						>
							<th class="px-4 py-3">No. Dokumen</th>
							<th class="px-4 py-3">Tanggal Keluar</th>
							<th class="px-4 py-3">Waktu Rekam</th>
							<th class="px-4 py-3">Cabang / Gudang Asal</th>
							<th class="px-4 py-3">Alasan / Kategori</th>
							<th class="px-4 py-3">Memo / Penerima</th>
							<th class="px-4 py-3 text-center">Item</th>
							<th class="px-4 py-3">Operator</th>
							<th class="px-4 py-3 text-right">Aksi</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100 text-sm">
						{#each movements as mov (mov.id)}
							<tr class="transition-colors hover:bg-neutral-50/80">
								<td class="px-4 py-3.5 font-mono font-semibold text-neutral-900">
									{mov.movement_number}
								</td>
								<td class="px-4 py-3.5 font-medium text-neutral-900">
									{formatDate(mov.movement_date)}
								</td>
								<td class="px-4 py-3.5 text-xs text-neutral-500">
									{formatDateTime(mov.created_at)}
								</td>
								<td class="px-4 py-3.5 font-medium text-neutral-800">
									{mov.location_name}
								</td>
								<td class="px-4 py-3.5">
									<Badge variant="default">{getReasonLabel(mov.category_reason)}</Badge>
								</td>
								<td class="px-4 py-3.5 text-neutral-600">
									{mov.reference_number || '-'}
								</td>
								<td class="px-4 py-3.5 text-center font-bold text-neutral-900">
									{mov.items?.length ?? 0}
								</td>
								<td class="px-4 py-3.5 text-neutral-600">
									{mov.executed_by_name}
								</td>
								<td class="px-4 py-3.5 text-right">
									<Button
										variant="outline"
										size="sm"
										onclick={() => {
											const token = getToken();
											if (token) detail.open(token, mov.id);
										}}
									>
										Detail Item
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</Table>
			</div>

			<!-- Pagination -->
			<div class="border-t border-neutral-200 px-4 py-3">
				<Pagination
					page={currentPage}
					totalPages={Math.ceil(totalRecords / pageSize) || 1}
					totalItems={totalRecords}
					limit={pageSize}
					onPageChange={handlePageChange}
				/>
			</div>
		{/if}
	</div>
</div>

<!-- Modal Formulir Catat Barang Keluar Baru -->
<StockOutCreateModal
	{form}
	{locations}
	{products}
	token={getToken() ?? ''}
	{getProductById}
	onSuccess={loadMovements}
/>

<!-- Modal Detail Dokumen Barang Keluar -->
<StockOutDetailModal {detail} {getProductById} {getCategoryName} />
