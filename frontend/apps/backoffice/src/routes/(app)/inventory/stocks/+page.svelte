<script lang="ts">
	/**
	 * Halaman Manajemen Stok Cabang & Stock Opname (Inventaris & Stok).
	 *
	 * Fitur Kunci:
	 * - Pemilihan Cabang & Lokasi Aktif (Store / Warehouse Selector).
	 * - Kartu Statistik KPI Real-Time (Total SKU, Fisik, Siap Jual, Dipesan, Alert Menipis, Habis).
	 * - Tabel Saldo Stok Gabungan (Katalog Produk + Data Stok Cabang Terpilih).
	 * - Tab Cepat Filter Status (Semua, Stok Menipis, Stok Habis, Stok Aman).
	 * - Pencarian Cepat Nama / SKU dan Filter Kategori Produk.
	 * - Modal Stock Opname (Penyesuaian Kuantitas Fisik Riil dengan Kalkulasi Selisih Dinamis & Alasan).
	 * - Modal Atur Batas Minimum Stok (Low Stock Alert Threshold).
	 * - Modal Detail & Audit Status Stok per Produk.
	 * - Zero-Warning TypeScript & Svelte 5 Runes ($derived.by).
	 */
	import { onMount } from 'svelte';
	import { getToken, getUser } from '$lib/stores/auth.svelte';
	import {
		listLocations,
		listProducts,
		listCategories,
		listStocks,
		adjustStock,
		updateMinStock,
		listStockAdjustments,
		listSerials,
		getStockCardReport
	} from '@erp/api-client';
	import type {
		LocationResponse,
		ProductResponse,
		CategoryResponse,
		StockResponse,
		StockAdjustmentResponse,
		SerialUnitResponse,
		StockCardReportResponse
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
		toast
	} from '@erp/ui';

	// ── 0. Otorisasi Peran PBAC ───────────────────────────────────────────────
	const currentUser = $derived(getUser());
	const isWarehouseStaff = $derived(currentUser?.role === 'warehouse');
	const userLocationId = $derived(currentUser?.location_id);
	const canAdjustStock = $derived(
		currentUser?.role === 'owner' ||
			currentUser?.role === 'superadmin' ||
			currentUser?.role === 'admin' ||
			currentUser?.role === 'warehouse'
	);

	// ── 1. State Utama ────────────────────────────────────────────────────────
	let locations = $state<LocationResponse[]>([]);
	let selectedLocationId = $state<string>('');
	let products = $state<ProductResponse[]>([]);
	let categories = $state<CategoryResponse[]>([]);
	let stocks = $state<StockResponse[]>([]);

	let loadingStocks = $state(false);
	let error = $state<string | null>(null);

	// ── 2. Filter & Pencarian ─────────────────────────────────────────────────
	let searchQuery = $state('');
	let selectedCategoryFilter = $state<string>('');
	let statusFilter = $state<'all' | 'low' | 'out' | 'safe'>('all');

	// Pagination
	let currentPage = $state(1);
	let pageSize = $state(10);

	// ── 3. State Modal Stock Opname ───────────────────────────────────────────
	let showAdjustModal = $state(false);
	let adjustingItem = $state<EnrichedStock | null>(null);
	let adjustNewQty = $state<number>(0);
	let adjustDate = $state<string>(new Date().toISOString().split('T')[0]);
	let adjustReasonPreset = $state<string>('Hasil Stock Opname Fisik Rutin');
	let adjustReasonCustom = $state<string>('');
	let adjustSubmitting = $state(false);
	let adjustError = $state<string | null>(null);

	// Preset alasan opname
	const reasonPresets = [
		'Hasil Stock Opname Fisik Rutin',
		'Koreksi Selisih Hitung / Salah Catat Kasir',
		'Barang Rusak / Pecah / Cacat di Toko',
		'Penerimaan Retur dari Konsumen',
		'Penyesuaian Barang Sampel Display Toko',
		'Lainnya (Tuliskan Keterangan Mandiri)'
	];

	// ── 4. State Modal Update Min Stock ───────────────────────────────────────
	let showMinStockModal = $state(false);
	let minStockItem = $state<EnrichedStock | null>(null);
	let newMinStockVal = $state<number>(0);
	let minStockSubmitting = $state(false);
	let minStockError = $state<string | null>(null);

	// ── 5. State Modal Detail & Mutasi Stok (Kartu Stok & Serial/IMEI) ────────
	let showDetailModal = $state(false);
	let detailItem = $state<EnrichedStock | null>(null);
	let detailActiveTab = $state<'card' | 'serials' | 'summary'>('card');

	// State Tab 1: Kartu Stok (Buku Besar Mutasi)
	let detailCardLoading = $state(false);
	let detailCardError = $state<string | null>(null);
	let detailCardReport = $state<StockCardReportResponse | null>(null);
	let detailCardStartDate = $state<string>('');
	let detailCardEndDate = $state<string>('');

	// State Tab 2: Serial / IMEI
	let detailSerialsLoading = $state(false);
	let detailSerialsError = $state<string | null>(null);
	let detailSerialsList = $state<SerialUnitResponse[]>([]);
	let detailSerialSearch = $state<string>('');
	let detailSerialStatusFilter = $state<string>('all');

	const filteredDetailSerials = $derived.by(() => {
		let list = detailSerialsList ?? [];
		if (detailSerialStatusFilter !== 'all') {
			if (detailSerialStatusFilter === 'tersedia') {
				list = list.filter((s) => s.status === 'tersedia' || s.status === 'available');
			} else if (detailSerialStatusFilter === 'dipesan') {
				list = list.filter((s) => s.status === 'reserved');
			} else if (detailSerialStatusFilter === 'terjual') {
				list = list.filter((s) => s.status === 'terjual' || s.status === 'sold');
			} else {
				list = list.filter((s) => (s.status as string) === detailSerialStatusFilter);
			}
		}
		if (detailSerialSearch.trim()) {
			const q = detailSerialSearch.toLowerCase().trim();
			list = list.filter((s) => s.serial_number.toLowerCase().includes(q));
		}
		return list;
	});

	const serialAvailableCount = $derived(
		(detailSerialsList ?? []).filter((s) => s.status === 'tersedia' || s.status === 'available')
			.length
	);
	const serialReservedCount = $derived(
		(detailSerialsList ?? []).filter((s) => s.status === 'reserved').length
	);
	const serialSoldCount = $derived(
		(detailSerialsList ?? []).filter((s) => s.status === 'terjual' || s.status === 'sold').length
	);

	// ── 5.5 State Modal Riwayat Stock Opname (Audit Trail Ledger) ───────────────
	let showHistoryModal = $state(false);
	let historyLoading = $state(false);
	let historyTargetProduct = $state<EnrichedStock | null>(null);
	let historyItems = $state<StockAdjustmentResponse[]>([]);
	let historyTotalItems = $state(0);
	let historyTotalPages = $state(1);
	let historyPage = $state(1);
	let historyStartDate = $state<string>('');
	let historyEndDate = $state<string>('');
	const historyPageSize = 8;

	// ── 6. Enriched Stock Interface ───────────────────────────────────────────
	interface EnrichedStock {
		productId: string;
		productName: string;
		productSku: string;
		categoryName: string;
		productPrice: number;
		purchasePrice?: number | null | undefined;
		hasSerial: boolean;
		stockId?: string;
		locationId: string;
		quantity: number;
		reservedQuantity: number;
		availableQuantity: number;
		minStock: number;
		isLowStock: boolean;
		updatedAt?: string;
	}

	// Format Rupiah
	function formatRupiah(amount: number): string {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0,
			maximumFractionDigits: 0
		}).format(amount);
	}

	// Format Tanggal
	function formatDate(dateStr?: string): string {
		if (!dateStr) return '-';
		try {
			const d = new Date(dateStr);
			return d.toLocaleString('id-ID', {
				day: '2-digit',
				month: 'short',
				year: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return dateStr;
		}
	}

	function formatOnlyDate(dateStr?: string): string {
		if (!dateStr) return '-';
		try {
			const d = new Date(dateStr);
			return d.toLocaleDateString('id-ID', {
				day: '2-digit',
				month: 'short',
				year: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}

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

	// Pilihan Cabang untuk Select2
	const locationOptions = $derived<Select2Option[]>(
		locations.map((loc) => ({
			value: loc.id,
			label: `${loc.name} (${loc.code})`,
			subtext:
				loc.type === 'physical'
					? `Toko Fisik • ${loc.address || 'Alamat fisik'}`
					: 'Gudang Virtual E-Commerce'
		}))
	);

	// Cabang Terpilih
	const selectedLocation = $derived<LocationResponse | undefined>(
		locations.find((l) => l.id === selectedLocationId)
	);

	// Pilihan Kategori untuk Filter
	const categoryFilterOptions = $derived<Select2Option[]>([
		{ value: '', label: 'Semua Kategori' },
		...categories.map((c) => ({
			value: c.id,
			label: c.name
		}))
	]);

	// Map Kategori ID -> Nama Kategori
	const categoryMap = $derived<Map<string, string>>(new Map(categories.map((c) => [c.id, c.name])));

	// Gabungkan data Produk Katalog dengan Data Stok Cabang
	const enrichedStocks = $derived.by<EnrichedStock[]>(() => {
		const stockMap: Record<string, StockResponse> = {};
		for (const s of stocks) {
			stockMap[s.product_id] = s;
		}

		return products.map((prod) => {
			const st = stockMap[prod.id];
			const qty = st ? st.quantity : 0;
			const rsv = st ? st.reserved_quantity : 0;
			const min = st ? st.min_stock : 0;
			const avail = Math.max(0, qty - rsv);
			const isLow = st ? st.is_low_stock : false;

			return {
				productId: prod.id,
				productName: prod.name,
				productSku: prod.sku,
				categoryName: prod.category_id ? (categoryMap.get(prod.category_id) ?? '-') : '-',
				productPrice: prod.selling_price,
				purchasePrice: prod.purchase_price,
				hasSerial: prod.flag_serial_tracking,
				stockId: st?.id,
				locationId: selectedLocationId,
				quantity: qty,
				reservedQuantity: rsv,
				availableQuantity: avail,
				minStock: min,
				isLowStock: isLow,
				updatedAt: st?.updated_at
			};
		});
	});

	// Statistik KPI
	const stats = $derived.by(() => {
		const all = enrichedStocks;
		let totalQty = 0;
		let totalAvail = 0;
		let totalReserved = 0;
		let lowStockCount = 0;
		let outOfStockCount = 0;

		for (const item of all) {
			totalQty += item.quantity;
			totalAvail += item.availableQuantity;
			totalReserved += item.reservedQuantity;
			if (item.quantity === 0) {
				outOfStockCount++;
			} else if (item.isLowStock || (item.minStock > 0 && item.quantity <= item.minStock)) {
				lowStockCount++;
			}
		}

		return {
			totalSkus: all.length,
			totalQty,
			totalAvail,
			totalReserved,
			lowStockCount,
			outOfStockCount
		};
	});

	// Filter & Pencarian
	const filteredStocks = $derived.by<EnrichedStock[]>(() => {
		let list = enrichedStocks;

		// Filter status tab
		if (statusFilter === 'low') {
			list = list.filter(
				(i: EnrichedStock) =>
					(i.isLowStock || (i.minStock > 0 && i.quantity <= i.minStock)) && i.quantity > 0
			);
		} else if (statusFilter === 'out') {
			list = list.filter((i: EnrichedStock) => i.quantity === 0);
		} else if (statusFilter === 'safe') {
			list = list.filter(
				(i: EnrichedStock) =>
					i.quantity > 0 && !i.isLowStock && (i.minStock === 0 || i.quantity > i.minStock)
			);
		}

		// Filter Kategori
		if (selectedCategoryFilter) {
			list = list.filter((i: EnrichedStock) => {
				const prod = products.find((p) => p.id === i.productId);
				return prod?.category_id === selectedCategoryFilter;
			});
		}

		// Pencarian SKU atau Nama
		if (searchQuery.trim()) {
			const q = searchQuery.toLowerCase().trim();
			list = list.filter(
				(i: EnrichedStock) =>
					i.productName.toLowerCase().includes(q) || i.productSku.toLowerCase().includes(q)
			);
		}

		return list;
	});

	// Pagination Data
	const totalItems = $derived(filteredStocks.length);
	const totalPages = $derived(Math.max(1, Math.ceil(totalItems / pageSize)));
	const paginatedStocks = $derived.by<EnrichedStock[]>(() => {
		const start = (currentPage - 1) * pageSize;
		return filteredStocks.slice(start, start + pageSize);
	});

	// ── 7. Data Fetching ──────────────────────────────────────────────────────
	async function loadInitialData() {
		error = null;
		try {
			const token = getToken();
			if (!token) throw new Error('Sesi autentikasi telah berakhir. Silakan login kembali.');

			const [locList, prodRes, catList] = await Promise.all([
				listLocations(token, true),
				listProducts(token, { limit: 100 }),
				listCategories(token)
			]);

			locations = locList;
			products = prodRes.data;
			categories = catList;

			// Jika user adalah admin gudang, kunci mutlak ke gudang penugasannya
			if (isWarehouseStaff && userLocationId) {
				const myLoc = locList.filter((l) => l.id === userLocationId);
				locations = myLoc.length > 0 ? myLoc : locList;
				selectedLocationId = userLocationId;
				await fetchLocationStocks(userLocationId);
			} else {
				locations = locList;
				if (locList.length > 0) {
					if (userLocationId && locList.some((l) => l.id === userLocationId)) {
						selectedLocationId = userLocationId;
					} else {
						selectedLocationId = locList[0].id;
					}
					await fetchLocationStocks(selectedLocationId);
				}
			}
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat data master inventaris.';
			}
		}
	}

	async function fetchLocationStocks(locationId: string) {
		if (!locationId) return;
		const targetLocId = isWarehouseStaff && userLocationId ? userLocationId : locationId;
		loadingStocks = true;
		try {
			const token = getToken();
			if (!token) throw new Error('Token tidak ditemukan');
			const data = await listStocks(token, targetLocId);
			stocks = data;
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Gagal memuat saldo stok cabang';
			toast.error(msg);
		} finally {
			loadingStocks = false;
		}
	}

	function handleLocationChange(newLocId: string) {
		if (isWarehouseStaff && userLocationId && newLocId !== userLocationId) {
			toast.error('Admin gudang hanya diizinkan melihat stok di gudang yang ditugaskan.');
			return;
		}
		selectedLocationId = newLocId;
		currentPage = 1;
		fetchLocationStocks(newLocId);
	}

	// ── 8. Modal Actions: Stock Opname ────────────────────────────────────────
	function openAdjustModal(item: EnrichedStock) {
		adjustingItem = item;
		adjustDate = new Date().toISOString().split('T')[0];
		adjustNewQty = item.quantity;
		adjustReasonPreset = 'Hasil Stock Opname Fisik Rutin';
		adjustReasonCustom = '';
		adjustError = null;
		showAdjustModal = true;
	}

	async function handleExecuteAdjust() {
		if (!adjustingItem || !selectedLocationId) return;

		// Validasi
		if (adjustNewQty < 0) {
			adjustError = 'Kuantitas fisik baru tidak boleh bernilai negatif.';
			return;
		}
		if (adjustNewQty < adjustingItem.reservedQuantity) {
			adjustError = `Kuantitas baru (${adjustNewQty}) tidak boleh lebih kecil dari stok yang sedang direservasi (${adjustingItem.reservedQuantity} unit).`;
			return;
		}

		let finalReason = adjustReasonPreset;
		if (adjustReasonPreset.startsWith('Lainnya')) {
			if (!adjustReasonCustom.trim()) {
				adjustError = 'Silakan tuliskan keterangan alasan penyesuaian stok.';
				return;
			}
			finalReason = adjustReasonCustom.trim();
		}

		adjustSubmitting = true;
		adjustError = null;
		try {
			const token = getToken();
			if (!token) throw new Error('Token tidak ditemukan');

			await adjustStock(token, {
				product_id: adjustingItem.productId,
				location_id: selectedLocationId,
				adjustment_date: adjustDate,
				new_quantity: adjustNewQty,
				reason: finalReason
			});

			toast.success(
				`Stok produk "${adjustingItem.productName}" berhasil disesuaikan menjadi ${adjustNewQty} unit.`
			);
			showAdjustModal = false;
			await fetchLocationStocks(selectedLocationId);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				adjustError = err.message;
			} else if (err instanceof Error) {
				adjustError = err.message;
			} else {
				adjustError = 'Terjadi kesalahan saat menyimpan penyesuaian stok.';
			}
		} finally {
			adjustSubmitting = false;
		}
	}

	// ── 9. Modal Actions: Update Min Stock ────────────────────────────────────
	function openMinStockModal(item: EnrichedStock) {
		minStockItem = item;
		newMinStockVal = item.minStock;
		minStockError = null;
		showMinStockModal = true;
	}

	async function handleExecuteMinStock() {
		if (!minStockItem || !selectedLocationId) return;

		if (newMinStockVal < 0) {
			minStockError = 'Batas minimum stok tidak boleh negatif.';
			return;
		}

		minStockSubmitting = true;
		minStockError = null;
		try {
			const token = getToken();
			if (!token) throw new Error('Token tidak ditemukan');

			await updateMinStock(token, {
				product_id: minStockItem.productId,
				location_id: selectedLocationId,
				min_stock: newMinStockVal
			});

			toast.success(
				`Batas minimum stok "${minStockItem.productName}" diperbarui menjadi ${newMinStockVal} unit.`
			);
			showMinStockModal = false;
			await fetchLocationStocks(selectedLocationId);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				minStockError = err.message;
			} else if (err instanceof Error) {
				minStockError = err.message;
			} else {
				minStockError = 'Gagal memperbarui batas minimum stok.';
			}
		} finally {
			minStockSubmitting = false;
		}
	}

	// ── 10. Modal Actions: Detail, Kartu Stok & Serial/IMEI ───────────────────
	async function openDetailModal(
		item: EnrichedStock,
		defaultTab: 'card' | 'serials' | 'summary' = 'card'
	) {
		detailItem = item;
		detailActiveTab = item.hasSerial && defaultTab === 'serials' ? 'serials' : defaultTab;
		detailCardStartDate = getFirstDayOfMonth();
		detailCardEndDate = getTodayDate();
		detailSerialSearch = '';
		detailSerialStatusFilter = 'all';
		showDetailModal = true;

		if (detailActiveTab === 'card') {
			await loadDetailStockCard();
		} else if (detailActiveTab === 'serials' && item.hasSerial) {
			await loadDetailSerials();
		}
	}

	async function switchDetailTab(tab: 'card' | 'serials' | 'summary') {
		detailActiveTab = tab;
		if (tab === 'card' && !detailCardReport && !detailCardLoading) {
			await loadDetailStockCard();
		} else if (tab === 'serials' && detailSerialsList.length === 0 && !detailSerialsLoading) {
			await loadDetailSerials();
		}
	}

	async function loadDetailStockCard() {
		if (!detailItem) return;
		const token = getToken();
		if (!token) return;

		try {
			detailCardLoading = true;
			detailCardError = null;
			detailCardReport = await getStockCardReport(token, {
				product_id: detailItem.productId,
				location_id: selectedLocationId,
				start_date: detailCardStartDate || undefined,
				end_date: detailCardEndDate || undefined
			});
		} catch (err: unknown) {
			detailCardError = err instanceof ApiError ? err.message : 'Gagal memuat kartu stok produk';
			detailCardReport = null;
		} finally {
			detailCardLoading = false;
		}
	}

	async function loadDetailSerials() {
		if (!detailItem || !detailItem.hasSerial) return;
		const token = getToken();
		if (!token) return;

		try {
			detailSerialsLoading = true;
			detailSerialsError = null;
			const res = await listSerials(token, {
				product_id: detailItem.productId,
				location_id: selectedLocationId
			});
			detailSerialsList = res ?? [];
		} catch (err: unknown) {
			detailSerialsError =
				err instanceof ApiError ? err.message : 'Gagal memuat daftar nomor seri produk';
			detailSerialsList = [];
		} finally {
			detailSerialsLoading = false;
		}
	}

	function copyAllAvailableSerials() {
		const available = (detailSerialsList ?? [])
			.filter((s) => s.status === 'tersedia')
			.map((s) => s.serial_number);
		if (available.length === 0) {
			toast.info('Tidak ada nomor seri yang berstatus tersedia.');
			return;
		}
		navigator.clipboard.writeText(available.join('\n'));
		toast.success(`${available.length} nomor seri berhasil disalin ke clipboard!`);
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

	function getSerialStatusBadge(st: string): {
		label: string;
		variant: 'success' | 'warning' | 'default' | 'danger' | 'info';
	} {
		switch (st) {
			case 'tersedia':
				return { label: 'Tersedia', variant: 'success' };
			case 'dipesan':
				return { label: 'Dipesan', variant: 'warning' };
			case 'terjual':
				return { label: 'Terjual', variant: 'default' };
			case 'retur':
				return { label: 'Retur', variant: 'info' };
			case 'rusak':
				return { label: 'Rusak', variant: 'danger' };
			default:
				return { label: st, variant: 'default' };
		}
	}

	// ── 11. Modal Actions: Riwayat Stock Opname ─────────────────────────────────
	async function loadAdjustHistory(page: number = 1) {
		historyLoading = true;
		historyPage = page;
		try {
			const token = getToken();
			if (!token) return;
			const res = await listStockAdjustments(token, {
				location_id: selectedLocationId || undefined,
				product_id: historyTargetProduct?.productId || undefined,
				start_date: historyStartDate || undefined,
				end_date: historyEndDate || undefined,
				page,
				limit: historyPageSize
			});
			historyItems = res.data;
			historyTotalItems = res.meta.total_items;
			historyTotalPages = res.meta.total_pages;
		} catch (err: unknown) {
			console.error(err);
			toast.error('Gagal memuat riwayat stock opname');
		} finally {
			historyLoading = false;
		}
	}

	function openProductHistoryModal(item: EnrichedStock) {
		historyTargetProduct = item;
		historyStartDate = getFirstDayOfMonth();
		historyEndDate = getTodayDate();
		showHistoryModal = true;
		loadAdjustHistory(1);
	}

	function openLocationHistoryModal() {
		historyTargetProduct = null;
		historyStartDate = getFirstDayOfMonth();
		historyEndDate = getTodayDate();
		showHistoryModal = true;
		loadAdjustHistory(1);
	}

	function formatDateTime(iso: string): string {
		try {
			const d = new Date(iso);
			return d.toLocaleString('id-ID', {
				day: 'numeric',
				month: 'short',
				year: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return iso;
		}
	}

	onMount(() => {
		loadInitialData();
	});
</script>

<svelte:head>
	<title>Stok Cabang & Opname — Inventaris & Stok Gen-E</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header & Cabang Selector Toolbar -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<h1 class="text-xl font-bold tracking-tight text-neutral-900">Stok Cabang & Opname</h1>
				<Badge variant="primary" size="sm">Inventaris</Badge>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Monitor saldo fisik persediaan produk, kuantitas siap jual, unit reservasi, dan eksekusi
				stock opname per cabang.
			</p>
		</div>

		<!-- Dropdown Pemilihan Cabang (Compact) -->
		<div class="flex items-center gap-2">
			{#if isWarehouseStaff && selectedLocation}
				<!-- Badge Lokasi Terkunci untuk Admin Gudang -->
				<div
					class="flex items-center gap-2 rounded-xl border border-neutral-200/90 bg-white px-3.5 py-1.5 shadow-2xs"
					title="Akun Admin Gudang terikat khusus pada gudang ini"
				>
					<svg
						class="h-3.5 w-3.5 text-neutral-500"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z"
						/>
					</svg>
					<span class="text-xs font-bold text-neutral-900">
						{selectedLocation.name}
					</span>
					<Badge variant="indigo" size="sm">Gudang Anda</Badge>
				</div>
			{:else}
				<div class="w-60 sm:w-72">
					<Select2
						size="sm"
						options={locationOptions}
						value={selectedLocationId}
						placeholder="Pilih Cabang..."
						searchPlaceholder="Cari cabang / gudang..."
						onchange={(val) => handleLocationChange(String(val))}
					/>
				</div>
			{/if}

			<Button
				variant="outline"
				size="sm"
				loading={loadingStocks}
				onclick={() => selectedLocationId && fetchLocationStocks(selectedLocationId)}
			>
				<!-- Arrow Path Refresh Icon -->
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99"
					/>
				</svg>
			</Button>
		</div>
	</div>

	<!-- Info Banner Cabang Terpilih -->
	{#if selectedLocation}
		<div
			class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-200/80 bg-neutral-50/70 px-4 py-2.5 text-xs text-neutral-600 shadow-2xs backdrop-blur-xs"
		>
			<div class="flex flex-wrap items-center gap-2.5">
				<!-- Building Storefront Icon -->
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-neutral-900 text-white shadow-xs"
				>
					<svg
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M13.5 21v-7.5a.75.75 0 01.75-.75h3a.75.75 0 01.75.75V21m-4.5 0H2.36m11.14 0H18m0 0h3.64m-1.39 0V9.349m-16.5 11.65V9.35m0 0a3.001 3.001 0 003.75-.615A2.993 2.993 0 009 9.75c.696 0 1.345-.237 1.865-.635A2.991 2.991 0 0012 9.75c.696 0 1.345-.237 1.865-.635A2.991 2.991 0 0015 9.75c.696 0 1.345-.237 1.865-.635A3.001 3.001 0 0020.625 9.35M3.75 9.35l.937-4.685A1.5 1.5 0 016.155 3.39h11.69a1.5 1.5 0 011.468 1.275l.937 4.685"
						/>
					</svg>
				</div>
				<div>
					<div class="flex items-center gap-2">
						<span class="font-semibold text-neutral-900">{selectedLocation.name}</span>
						<Badge variant="indigo" size="sm"
							><span class="font-mono">{selectedLocation.code}</span></Badge
						>
						<Badge variant={selectedLocation.type === 'physical' ? 'cyan' : 'purple'} size="sm">
							{selectedLocation.type === 'physical' ? 'Toko Fisik' : 'Gudang Online'}
						</Badge>
					</div>
					<p class="text-3xs line-clamp-1 text-neutral-500">
						{selectedLocation.address || 'Alamat cabang belum diisi'}
					</p>
				</div>
			</div>

			{#if selectedLocation.latitude && selectedLocation.longitude}
				<a
					href="https://www.google.com/maps?q={selectedLocation.latitude},{selectedLocation.longitude}"
					target="_blank"
					rel="noreferrer"
					class="text-2xs inline-flex items-center gap-1 font-medium text-neutral-700 hover:text-neutral-900 hover:underline"
				>
					<!-- Map Pin Icon -->
					<svg
						class="h-3.5 w-3.5 text-rose-500"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M15 10.5a3 3 0 11-6 0 3 3 0 016 0z"
						/>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1115 0z"
						/>
					</svg>
					Buka di Maps ({selectedLocation.latitude.toFixed(4)}, {selectedLocation.longitude.toFixed(
						4
					)})
				</a>
			{/if}
		</div>
	{/if}

	<!-- Error Alert Global -->
	{#if error}
		<Alert variant="error" title="Gagal Memuat Data">{error}</Alert>
	{/if}

	<!-- Kartu Statistik KPI -->
	<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
		<!-- Total SKU -->
		<div
			class="relative overflow-hidden rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs transition hover:border-neutral-300"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-neutral-400 uppercase"
					>Total SKU</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-neutral-100 text-neutral-500"
				>
					<!-- Cube Icon -->
					<svg
						class="h-3.5 w-3.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M21 7.5l-9-5.25L3 7.5m18 0l-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-neutral-800 sm:text-3xl"
					>{stats.totalSkus}</span
				>
				<span class="text-[11px] font-normal text-neutral-400">item</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-neutral-400/80">Katalog terdaftar</p>
		</div>

		<!-- Stok Fisik -->
		<div
			class="relative overflow-hidden rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs transition hover:border-neutral-300"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-neutral-400 uppercase"
					>Stok Fisik</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-neutral-900 text-white shadow-2xs"
				>
					<!-- Building Library Icon -->
					<svg
						class="h-3.5 w-3.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 5.625c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125m16.5 5.625c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-neutral-900 sm:text-3xl"
					>{stats.totalQty}</span
				>
				<span class="text-[11px] font-normal text-neutral-400">unit</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-neutral-400/80">Unit di rak & gudang</p>
		</div>

		<!-- Siap Jual -->
		<div
			class="relative overflow-hidden rounded-xl border border-emerald-200/90 bg-emerald-50/20 p-3.5 shadow-2xs transition hover:border-emerald-300"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-emerald-600/80 uppercase"
					>Siap Jual</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700"
				>
					<!-- Check Circle Icon -->
					<svg
						class="h-3.5 w-3.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2.2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-emerald-700 sm:text-3xl"
					>{stats.totalAvail}</span
				>
				<span class="text-[11px] font-normal text-emerald-600/60">unit</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-emerald-600/70">Unit bebas diproses</p>
		</div>

		<!-- Dipesan -->
		<div
			class="relative overflow-hidden rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs transition hover:border-neutral-300"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-amber-700/80 uppercase"
					>Dipesan</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-100 text-amber-700"
				>
					<!-- Lock Icon -->
					<svg
						class="h-3.5 w-3.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-neutral-800 sm:text-3xl"
					>{stats.totalReserved}</span
				>
				<span class="text-[11px] font-normal text-neutral-400">unit</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-neutral-400/80">Alokasi pesanan/mutasi</p>
		</div>

		<!-- Menipis -->
		<button
			type="button"
			onclick={() => {
				statusFilter = statusFilter === 'low' ? 'all' : 'low';
				currentPage = 1;
			}}
			class="relative overflow-hidden rounded-xl border p-3.5 text-left transition-all {statusFilter ===
			'low'
				? 'border-amber-500 bg-amber-50/80 shadow-sm ring-2 ring-amber-400'
				: 'border-amber-200/90 bg-white shadow-2xs hover:border-amber-400 hover:bg-amber-50/40'}"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-amber-700/80 uppercase"
					>Menipis</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-100 text-amber-700"
				>
					<!-- Exclamation Triangle Icon -->
					<svg
						class="h-3.5 w-3.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2.2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-amber-900 sm:text-3xl"
					>{stats.lowStockCount}</span
				>
				<span class="text-[11px] font-normal text-amber-700/60">SKU</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-amber-800/60">Perlu reorder segera</p>
		</button>

		<!-- Habis -->
		<button
			type="button"
			onclick={() => {
				statusFilter = statusFilter === 'out' ? 'all' : 'out';
				currentPage = 1;
			}}
			class="relative overflow-hidden rounded-xl border p-3.5 text-left transition-all {statusFilter ===
			'out'
				? 'border-rose-500 bg-rose-50/80 shadow-sm ring-2 ring-rose-400'
				: 'border-rose-200/90 bg-white shadow-2xs hover:border-rose-400 hover:bg-rose-50/40'}"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-rose-700/80 uppercase">Habis</span>
				<div class="flex h-7 w-7 items-center justify-center rounded-lg bg-rose-100 text-rose-700">
					<!-- X Circle Icon -->
					<svg
						class="h-3.5 w-3.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2.2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-rose-900 sm:text-3xl"
					>{stats.outOfStockCount}</span
				>
				<span class="text-[11px] font-normal text-rose-700/60">SKU</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-rose-800/60">Saldo fisik 0 unit</p>
		</button>
	</div>

	<!-- Filter Tabs & Pencarian -->
	<div
		class="flex flex-col gap-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs sm:flex-row sm:items-center sm:justify-between"
	>
		<!-- Quick Filter Tab Pills -->
		<div class="flex flex-wrap items-center gap-1.5">
			<button
				type="button"
				onclick={() => {
					statusFilter = 'all';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'all'
					? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-neutral-600 shadow-2xs hover:border-neutral-300 hover:bg-neutral-50'}"
			>
				Semua
				<span
					class="py-0.2 text-3xs rounded-full px-1.5 font-semibold {statusFilter === 'all'
						? 'bg-neutral-800 text-neutral-300'
						: 'bg-neutral-100 text-neutral-600'}"
				>
					{stats.totalSkus}
				</span>
			</button>

			<button
				type="button"
				onclick={() => {
					statusFilter = 'low';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'low'
					? 'border-amber-600 bg-amber-600 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-amber-800 shadow-2xs hover:border-neutral-300 hover:bg-amber-50/50'}"
			>
				Menipis (Alert)
				<span
					class="py-0.2 text-3xs rounded-full px-1.5 font-semibold {statusFilter === 'low'
						? 'bg-amber-700 text-white'
						: 'border border-amber-200/60 bg-amber-50 text-amber-800'}"
				>
					{stats.lowStockCount}
				</span>
			</button>

			<button
				type="button"
				onclick={() => {
					statusFilter = 'out';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'out'
					? 'border-rose-600 bg-rose-600 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-rose-800 shadow-2xs hover:border-neutral-300 hover:bg-rose-50/50'}"
			>
				Habis (0 Unit)
				<span
					class="py-0.2 text-3xs rounded-full px-1.5 font-semibold {statusFilter === 'out'
						? 'bg-rose-700 text-white'
						: 'border border-rose-200/60 bg-rose-50 text-rose-800'}"
				>
					{stats.outOfStockCount}
				</span>
			</button>

			<button
				type="button"
				onclick={() => {
					statusFilter = 'safe';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'safe'
					? 'border-emerald-700 bg-emerald-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-emerald-800 shadow-2xs hover:border-neutral-300 hover:bg-emerald-50/50'}"
			>
				Stok Aman
			</button>
		</div>

		<!-- Search Input & Kategori Filter & Riwayat Opname -->
		<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
			<Button variant="outline" size="sm" onclick={openLocationHistoryModal}>
				<svg
					class="h-4 w-4 text-neutral-500"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.75"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
				Riwayat Opname
			</Button>

			<div class="w-full sm:w-48">
				<Select2
					size="sm"
					options={categoryFilterOptions}
					value={selectedCategoryFilter}
					placeholder="Semua Kategori"
					searchPlaceholder="Cari kategori..."
					clearable={true}
					onchange={(val) => {
						selectedCategoryFilter = String(val);
						currentPage = 1;
					}}
				/>
			</div>

			<div class="w-full sm:w-56">
				<SearchInput
					size="sm"
					bind:value={searchQuery}
					placeholder="Cari SKU atau nama..."
					onsearch={() => {
						currentPage = 1;
					}}
				/>
			</div>
		</div>
	</div>

	<!-- Tabel Saldo Stok Cabang -->
	<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
		<Table
			borderless
			empty={paginatedStocks.length === 0}
			emptyMessage={loadingStocks
				? 'Memuat saldo stok cabang...'
				: 'Tidak ada produk yang cocok dengan kriteria filter.'}
		>
			<thead>
				<tr
					class="border-b border-neutral-200 bg-neutral-50/75 text-left text-xs font-semibold text-neutral-600"
				>
					<th class="px-5 py-3.5">Produk & SKU</th>
					<th class="px-5 py-3.5">Kategori</th>
					<th class="px-5 py-3.5 text-right">Harga Retail</th>
					<th class="px-5 py-3.5 text-center">Fisik</th>
					<th class="px-5 py-3.5 text-center">Dipesan</th>
					<th class="px-5 py-3.5 text-center">Tersedia</th>
					<th class="px-5 py-3.5 text-center">Min. Stok</th>
					<th class="px-5 py-3.5 text-center">Status</th>
					<th class="px-5 py-3.5 text-right">Aksi</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-200 bg-white text-xs">
				{#each paginatedStocks as item (item.productId)}
					<tr
						class="transition-colors hover:bg-neutral-50/80 {item.quantity === 0
							? 'bg-rose-50/20'
							: item.isLowStock
								? 'bg-amber-50/20'
								: ''}"
					>
						<!-- Produk & SKU -->
						<td class="px-5 py-4">
							<div class="flex items-start gap-2.5">
								<div
									class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-neutral-200 bg-neutral-100 text-neutral-500 shadow-2xs"
								>
									<!-- Cube Icon -->
									<svg
										class="h-4 w-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="1.75"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M21 7.5l-9-5.25L3 7.5m18 0l-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9"
										/>
									</svg>
								</div>
								<div>
									<button
										type="button"
										onclick={() => openDetailModal(item)}
										class="line-clamp-1 text-left font-semibold text-neutral-900 hover:text-primary-700 hover:underline"
										title="Klik untuk detail info stok"
									>
										{item.productName}
									</button>
									<div class="mt-0.5 flex items-center gap-1.5">
										<span class="text-3xs font-mono font-medium text-neutral-500"
											>{item.productSku}</span
										>
										{#if item.hasSerial}
											<Badge variant="purple" size="sm">Serial</Badge>
										{/if}
									</div>
								</div>
							</div>
						</td>

						<!-- Kategori -->
						<td class="px-5 py-4 text-neutral-600">
							<span class="line-clamp-1">{item.categoryName}</span>
						</td>

						<!-- Harga Retail -->
						<td class="px-5 py-4 text-right font-medium text-neutral-800">
							{formatRupiah(item.productPrice)}
						</td>

						<!-- Kuantitas Fisik -->
						<td class="px-5 py-4 text-center">
							<span
								class="font-mono text-sm font-bold {item.quantity === 0
									? 'text-rose-600'
									: 'text-neutral-900'}"
							>
								{item.quantity}
							</span>
						</td>

						<!-- Dipesan (Reserved) -->
						<td class="px-5 py-4 text-center">
							{#if item.reservedQuantity > 0}
								<span
									class="rounded-md border border-amber-200/80 bg-amber-50 px-2 py-0.5 font-mono text-xs font-semibold text-amber-700"
								>
									{item.reservedQuantity}
								</span>
							{:else}
								<span class="font-mono text-xs text-neutral-400">0</span>
							{/if}
						</td>

						<!-- Tersedia (Available) -->
						<td class="px-5 py-4 text-center">
							<span
								class="font-mono text-sm font-semibold {item.availableQuantity === 0
									? 'text-neutral-400'
									: 'text-emerald-700'}"
							>
								{item.availableQuantity}
							</span>
						</td>

						<!-- Min. Stok -->
						<td class="px-5 py-4 text-center font-mono text-xs text-neutral-500">
							{item.minStock}
						</td>

						<!-- Status Badge -->
						<td class="px-5 py-4 text-center">
							{#if item.quantity === 0}
								<Badge variant="danger" size="sm">Habis</Badge>
							{:else if item.isLowStock || (item.minStock > 0 && item.quantity <= item.minStock)}
								<Badge variant="warning" size="sm">Menipis</Badge>
							{:else}
								<Badge variant="success" size="sm">Aman</Badge>
							{/if}
						</td>

						<!-- Tombol Aksi Dropdown -->
						<td class="px-5 py-4 text-right whitespace-nowrap">
							<ActionMenu
								items={[
									{
										label: 'Buku Kartu Stok (Mutasi)',
										icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" /></svg>`,
										onClick: () => openDetailModal(item, 'card')
									},
									...(item.hasSerial
										? [
												{
													label: 'Daftar Unit Serial / IMEI',
													icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z" /></svg>`,
													onClick: () => openDetailModal(item, 'serials')
												}
											]
										: []),
									{
										label: 'Detail Ringkasan Stok',
										icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" /></svg>`,
										divider: true,
										onClick: () => openDetailModal(item, 'summary')
									},
									...(canAdjustStock
										? [
												{
													label: 'Stock Opname (Penyesuaian)',
													icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" /></svg>`,
													onClick: () => openAdjustModal(item)
												}
											]
										: []),
									{
										label: 'Riwayat Opname Barang',
										icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>`,
										onClick: () => openProductHistoryModal(item)
									},
									{
										label: 'Atur Batas Minimum Stok',
										icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75" /></svg>`,
										onClick: () => openMinStockModal(item)
									},
									{
										label: 'Salin SKU Produk',
										icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75" /></svg>`,
										divider: true,
										onClick: () => {
											navigator.clipboard.writeText(item.productSku);
											toast.success(`SKU ${item.productSku} disalin`);
										}
									}
								]}
							/>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	</div>

	<!-- Pagination -->
	{#if totalPages > 1}
		<div class="flex items-center justify-between">
			<span class="text-xs text-neutral-500">
				Menampilkan {(currentPage - 1) * pageSize + 1} - {Math.min(
					currentPage * pageSize,
					totalItems
				)} dari {totalItems} produk
			</span>
			<Pagination
				page={currentPage}
				{totalPages}
				{totalItems}
				limit={pageSize}
				onPageChange={(p) => (currentPage = p)}
			/>
		</div>
	{/if}
</div>

<!-- ======================================================================= -->
<!-- MODAL 1: STOCK OPNAME (PENYESUAIAN STOK FISIK)                          -->
<!-- ======================================================================= -->
{#if showAdjustModal && adjustingItem}
	<Modal bind:open={showAdjustModal} title="Stock Opname (Penyesuaian Fisik Stok)" size="xl">
		<div class="space-y-4.5">
			<!-- Ringkasan Produk & Cabang -->
			<div class="rounded-xl border border-neutral-200 bg-neutral-50/70 p-4">
				<div class="flex items-start justify-between gap-3">
					<div>
						<h4 class="text-base font-bold text-neutral-900">{adjustingItem.productName}</h4>
						<div class="text-2xs mt-1 flex flex-wrap items-center gap-2 text-neutral-500">
							<span class="font-mono">{adjustingItem.productSku}</span>
							<span>•</span>
							<span>{adjustingItem.categoryName}</span>
							<span>•</span>
							<span class="font-semibold text-neutral-800"
								>{formatRupiah(adjustingItem.productPrice)}</span
							>
						</div>
					</div>
					<Badge variant="indigo" size="sm">{selectedLocation?.code ?? 'CABANG'}</Badge>
				</div>

				<div class="mt-3.5 grid grid-cols-3 gap-3 border-t border-neutral-200/80 pt-3 text-center">
					<div class="rounded-lg border border-neutral-200/60 bg-white/70 p-2 shadow-2xs">
						<span class="text-3xs font-medium text-neutral-400 uppercase">Fisik Saat Ini</span>
						<p class="font-mono text-sm font-bold text-neutral-900">
							{adjustingItem.quantity} unit
						</p>
					</div>
					<div class="rounded-lg border border-amber-200/60 bg-amber-50/40 p-2 shadow-2xs">
						<span class="text-3xs font-medium text-amber-600 uppercase">Dipesan (Lock)</span>
						<p class="font-mono text-sm font-bold text-amber-700">
							{adjustingItem.reservedQuantity} unit
						</p>
					</div>
					<div class="rounded-lg border border-emerald-200/60 bg-emerald-50/40 p-2 shadow-2xs">
						<span class="text-3xs font-medium text-emerald-600 uppercase">Siap Jual</span>
						<p class="font-mono text-sm font-bold text-emerald-700">
							{adjustingItem.availableQuantity} unit
						</p>
					</div>
				</div>
			</div>

			<!-- Error Feedback Modal -->
			{#if adjustError}
				<Alert variant="error" title="Gagal Penyesuaian Stok">{adjustError}</Alert>
			{/if}

			<!-- Tanggal Pelaksanaan & Input Kuantitas Baru -->
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<Input label="Tanggal Pelaksanaan Opname" type="date" bind:value={adjustDate} required />
					<p class="text-3xs mt-1.5 text-neutral-400">
						Tanggal saat penghitungan fisik riil dilakukan di toko/gudang.
					</p>
				</div>
				<div>
					<Input
						label="Kuantitas Fisik Riil (Hasil Opname)"
						type="number"
						bind:value={adjustNewQty}
						placeholder="Contoh: 25"
						required
					/>
					<p class="text-3xs mt-1.5 leading-relaxed text-neutral-400">
						Kuantitas baru minimum: <strong class="text-neutral-600"
							>{adjustingItem.reservedQuantity} unit</strong
						> (tidak boleh kurang dari reservasi).
					</p>
				</div>
			</div>

			<!-- Kalkulasi Selisih Dinamis -->
			<div
				class="flex flex-col justify-center rounded-xl border border-neutral-200 bg-neutral-50/60 p-3.5 shadow-2xs"
			>
				<span class="text-3xs font-medium tracking-wider text-neutral-400 uppercase"
					>Kalkulasi Selisih Stok:</span
				>
				{#if adjustNewQty > adjustingItem.quantity}
					<div class="mt-1 flex items-center gap-1.5 text-emerald-700">
						<!-- Plus Circle Icon -->
						<svg
							class="h-4 w-4 shrink-0"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M12 9v6m3-3H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span class="font-mono text-sm font-bold"
							>+{adjustNewQty - adjustingItem.quantity} unit (Bertambah)</span
						>
					</div>
					<p class="text-3xs mt-0.5 text-neutral-500">
						Saldo persediaan fisik cabang akan ditingkatkan.
					</p>
				{:else if adjustNewQty < adjustingItem.quantity}
					<div class="mt-1 flex items-center gap-1.5 text-rose-700">
						<!-- Minus Circle Icon -->
						<svg
							class="h-4 w-4 shrink-0"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M15 12H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span class="font-mono text-sm font-bold"
							>-{adjustingItem.quantity - adjustNewQty} unit (Berkurang)</span
						>
					</div>
					<p class="text-3xs mt-0.5 text-neutral-500">
						Saldo persediaan fisik cabang akan dipotong.
					</p>
				{:else}
					<div class="mt-1 flex items-center gap-1.5 text-neutral-600">
						<span class="font-mono text-sm font-semibold">0 unit (Tidak Berubah)</span>
					</div>
					<p class="text-3xs mt-0.5 text-neutral-400">
						Kuantitas fisik sama dengan saldo tercatat saat ini.
					</p>
				{/if}
			</div>

			<!-- Alasan Penyesuaian (Preset Dropdown) -->
			<div>
				<label
					for="adjust-reason-select"
					class="text-2xs mb-1.5 block font-semibold tracking-wide text-neutral-700 uppercase"
				>
					Alasan Penyesuaian Stok (Audit Trail)
				</label>
				<select
					id="adjust-reason-select"
					bind:value={adjustReasonPreset}
					class="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden"
				>
					{#each reasonPresets as p (p)}
						<option value={p}>{p}</option>
					{/each}
				</select>
			</div>

			<!-- Custom Reason jika memilih 'Lainnya' -->
			{#if adjustReasonPreset.startsWith('Lainnya')}
				<div>
					<Input
						label="Keterangan Rinci Alasan"
						bind:value={adjustReasonCustom}
						placeholder="Jelaskan alasan penyesuaian untuk laporan pembukuan..."
						required
					/>
				</div>
			{/if}
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end gap-2">
				<Button
					variant="outline"
					onclick={() => (showAdjustModal = false)}
					disabled={adjustSubmitting}
				>
					Batal
				</Button>
				<Button variant="primary" loading={adjustSubmitting} onclick={handleExecuteAdjust}>
					Konfirmasi & Simpan Stok
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}

<!-- ======================================================================= -->
<!-- MODAL 2: UBAH BATAS MINIMUM STOK (ALERT THRESHOLD)                      -->
<!-- ======================================================================= -->
{#if showMinStockModal && minStockItem}
	<Modal bind:open={showMinStockModal} title="Atur Batas Minimum Stok (Low Stock Alert)" size="md">
		<div class="space-y-4">
			<div class="rounded-xl border border-neutral-200 bg-neutral-50/70 p-3">
				<h4 class="font-bold text-neutral-900">{minStockItem.productName}</h4>
				<p class="text-2xs font-mono text-neutral-500">{minStockItem.productSku}</p>
				<div class="mt-2 flex items-center gap-4 text-xs text-neutral-600">
					<span
						>Stok Fisik Saat Ini: <strong class="text-neutral-900"
							>{minStockItem.quantity} unit</strong
						></span
					>
					<span
						>Batas Saat Ini: <strong class="text-neutral-900">{minStockItem.minStock} unit</strong
						></span
					>
				</div>
			</div>

			{#if minStockError}
				<Alert variant="error" title="Gagal Menyimpan">{minStockError}</Alert>
			{/if}

			<div>
				<Input
					label="Ambang Batas Minimum Baru (Min Stock)"
					type="number"
					bind:value={newMinStockVal}
					placeholder="Contoh: 5"
					required
				/>
				<p class="text-3xs mt-1.5 text-neutral-500">
					Sistem akan otomatis memicu peringatan <strong>Low Stock Alert</strong> dan status
					<strong>Menipis</strong> apabila saldo fisik di cabang ini berada pada atau di bawah angka ini.
				</p>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end gap-2">
				<Button
					variant="outline"
					onclick={() => (showMinStockModal = false)}
					disabled={minStockSubmitting}
				>
					Batal
				</Button>
				<Button variant="primary" loading={minStockSubmitting} onclick={handleExecuteMinStock}>
					Simpan Batas Min
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}

<!-- ======================================================================= -->
<!-- MODAL 3: DETAIL LENGKAP STOK CABANG, KARTU STOK & SERIAL / IMEI         -->
<!-- ======================================================================= -->
{#if showDetailModal && detailItem}
	<Modal
		bind:open={showDetailModal}
		title={`Detail & Mutasi Stok: ${detailItem.productName}`}
		size="4xl"
	>
		<div class="space-y-4">
			<!-- Header Ringkasan Produk & Lokasi -->
			<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-4 shadow-2xs">
				<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
					<div>
						<div class="flex items-center gap-2">
							<h3 class="text-base font-bold text-neutral-900">{detailItem.productName}</h3>
							<Badge
								variant={detailItem.quantity === 0
									? 'danger'
									: detailItem.isLowStock
										? 'warning'
										: 'success'}
							>
								{detailItem.quantity === 0 ? 'Habis' : detailItem.isLowStock ? 'Menipis' : 'Aman'}
							</Badge>
						</div>
						<div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-neutral-500">
							<span class="font-mono font-medium text-neutral-700"
								>SKU: {detailItem.productSku}</span
							>
							<span>•</span>
							<span
								>Kategori: <strong class="text-neutral-700">{detailItem.categoryName}</strong></span
							>
							<span>•</span>
							<span
								>Cabang: <strong class="text-neutral-700"
									>{selectedLocation?.name ?? '-'} ({selectedLocation?.code ?? '-'})</strong
								></span
							>
							{#if detailItem.hasSerial}
								<Badge variant="purple" size="sm">Wajib Serial / IMEI</Badge>
							{/if}
						</div>
					</div>

					<!-- Saldo Ringkas -->
					<div class="flex items-center gap-3">
						<div
							class="rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-center shadow-2xs"
						>
							<span class="text-3xs font-medium text-neutral-500 uppercase">Fisik</span>
							<p class="font-mono text-sm font-bold text-neutral-900">{detailItem.quantity} unit</p>
						</div>
						<div
							class="rounded-lg border border-emerald-200 bg-emerald-50/50 px-3 py-1.5 text-center shadow-2xs"
						>
							<span class="text-3xs font-medium text-emerald-600 uppercase">Siap Jual</span>
							<p class="font-mono text-sm font-bold text-emerald-700">
								{detailItem.availableQuantity} unit
							</p>
						</div>
						<div
							class="rounded-lg border border-amber-200 bg-amber-50/50 px-3 py-1.5 text-center shadow-2xs"
						>
							<span class="text-3xs font-medium text-amber-600 uppercase">Dipesan</span>
							<p class="font-mono text-sm font-bold text-amber-700">
								{detailItem.reservedQuantity} unit
							</p>
						</div>
					</div>
				</div>
			</div>

			<!-- Navigasi Tab Segmented -->
			<div class="flex items-center gap-1 border-b border-neutral-200">
				<button
					type="button"
					onclick={() => switchDetailTab('card')}
					class="flex items-center gap-2 border-b-2 px-4 py-2.5 text-xs font-semibold transition-colors {detailActiveTab ===
					'card'
						? 'border-neutral-900 text-neutral-900'
						: 'border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700'}"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25"
						/>
					</svg>
					<span>Buku Kartu Stok (Mutasi)</span>
				</button>

				{#if detailItem.hasSerial}
					<button
						type="button"
						onclick={() => switchDetailTab('serials')}
						class="flex items-center gap-2 border-b-2 px-4 py-2.5 text-xs font-semibold transition-colors {detailActiveTab ===
						'serials'
							? 'border-neutral-900 text-neutral-900'
							: 'border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700'}"
					>
						<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="1.75"
								d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z"
							/>
						</svg>
						<span>Unit Serial / IMEI Fisik ({detailSerialsList.length})</span>
					</button>
				{/if}

				<button
					type="button"
					onclick={() => switchDetailTab('summary')}
					class="flex items-center gap-2 border-b-2 px-4 py-2.5 text-xs font-semibold transition-colors {detailActiveTab ===
					'summary'
						? 'border-neutral-900 text-neutral-900'
						: 'border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700'}"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z"
						/>
					</svg>
					<span>Ringkasan Nilai & Audit</span>
				</button>
			</div>

			<!-- TAB 1: BUKU KARTU STOK (MUTASI MASUK & KELUAR) -->
			{#if detailActiveTab === 'card'}
				<div class="space-y-4">
					<!-- Filter Periode Kartu Stok -->
					<div
						class="flex flex-wrap items-end gap-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs"
					>
						<div class="w-40">
							<Input label="Mulai Tanggal" type="date" bind:value={detailCardStartDate} />
						</div>
						<div class="w-40">
							<Input label="Sampai Tanggal" type="date" bind:value={detailCardEndDate} />
						</div>
						<Button
							variant="primary"
							size="sm"
							loading={detailCardLoading}
							onclick={loadDetailStockCard}
						>
							Terapkan Filter
						</Button>
						<Button
							variant="outline"
							size="sm"
							onclick={() => {
								detailCardStartDate = getFirstDayOfMonth();
								detailCardEndDate = getTodayDate();
								loadDetailStockCard();
							}}
						>
							Bulan Ini
						</Button>
					</div>

					{#if detailCardLoading}
						<div class="flex h-48 items-center justify-center">
							<div class="flex items-center gap-3 text-sm text-neutral-500">
								<svg class="h-5 w-5 animate-spin text-neutral-500" viewBox="0 0 24 24" fill="none">
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
								<span>Merekonsiliasi mutasi kartu stok...</span>
							</div>
						</div>
					{:else if detailCardError}
						<Alert variant="error" title="Gagal Memuat Kartu Stok">{detailCardError}</Alert>
					{:else if detailCardReport}
						<!-- KPI Ringkasan Kartu Stok -->
						<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
							<div class="rounded-xl border border-neutral-200 bg-white p-3 shadow-2xs">
								<span class="text-3xs font-semibold tracking-wider text-neutral-500 uppercase"
									>Saldo Awal</span
								>
								<div class="mt-1 text-xl font-bold text-neutral-800">
									{detailCardReport.opening_balance}
									<span class="text-xs font-normal text-neutral-500">unit</span>
								</div>
							</div>
							<div class="rounded-xl border border-emerald-200 bg-emerald-50/40 p-3 shadow-2xs">
								<span class="text-3xs font-semibold tracking-wider text-emerald-600 uppercase"
									>Total Masuk (+)</span
								>
								<div class="mt-1 text-xl font-bold text-emerald-700">
									+{detailCardReport.total_in}
									<span class="text-xs font-normal text-neutral-500">unit</span>
								</div>
							</div>
							<div class="rounded-xl border border-rose-200 bg-rose-50/40 p-3 shadow-2xs">
								<span class="text-3xs font-semibold tracking-wider text-rose-600 uppercase"
									>Total Keluar (-)</span
								>
								<div class="mt-1 text-xl font-bold text-rose-700">
									-{detailCardReport.total_out}
									<span class="text-xs font-normal text-neutral-500">unit</span>
								</div>
							</div>
							<div
								class="rounded-xl border border-neutral-300 bg-neutral-900 p-3 text-white shadow-2xs"
							>
								<span class="text-3xs font-semibold tracking-wider text-neutral-300 uppercase"
									>Saldo Akhir Fisik</span
								>
								<div class="mt-1 text-xl font-bold text-white">
									{detailCardReport.closing_balance}
									<span class="text-xs font-normal text-neutral-300">unit</span>
								</div>
							</div>
						</div>

						<!-- Tabel Mutasi Kronologis -->
						<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
							{#if !detailCardReport.entries || detailCardReport.entries.length === 0}
								<div class="py-12 text-center text-xs text-neutral-500">
									<svg
										class="mx-auto mb-2 h-8 w-8 text-neutral-300"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="1.5"
											d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"
										/>
									</svg>
									Tidak ada riwayat pergerakan stok untuk produk ini pada periode tanggal yang dipilih.
								</div>
							{:else}
								<div class="max-h-96 overflow-x-auto">
									<Table>
										<thead>
											<tr
												class="text-3xs border-b border-neutral-200 bg-neutral-50/80 text-left font-semibold tracking-wider text-neutral-600 uppercase"
											>
												<th class="px-3.5 py-2.5">Tanggal & Waktu</th>
												<th class="px-3.5 py-2.5">Jenis Mutasi</th>
												<th class="px-3.5 py-2.5">No. Dokumen</th>
												<th class="px-3.5 py-2.5">Surat Jalan / Memo</th>
												<th class="px-3.5 py-2.5">Asal / Keperluan</th>
												<th class="px-3.5 py-2.5 text-right">Masuk (+)</th>
												<th class="px-3.5 py-2.5 text-right">Keluar (-)</th>
												<th class="px-3.5 py-2.5 text-right">Saldo</th>
												<th class="px-3.5 py-2.5">Petugas</th>
												<th class="px-3.5 py-2.5">Catatan</th>
											</tr>
										</thead>
										<tbody class="divide-y divide-neutral-100 text-xs">
											{#each detailCardReport.entries as entry, idx (entry.document_number + '_' + idx)}
												{@const badge = getMovementTypeBadge(entry.movement_type)}
												<tr class="transition-colors hover:bg-neutral-50/80">
													<td class="px-3.5 py-2.5 whitespace-nowrap text-neutral-600">
														{formatDateTime(entry.date)}
													</td>
													<td class="px-3.5 py-2.5 whitespace-nowrap">
														<Badge variant={badge.variant} size="sm">{badge.label}</Badge>
													</td>
													<td
														class="px-3.5 py-2.5 font-mono font-semibold whitespace-nowrap text-neutral-900"
													>
														{entry.document_number}
													</td>
													<td class="px-3.5 py-2.5 whitespace-nowrap text-neutral-600">
														{entry.reference_number || '-'}
													</td>
													<td class="px-3.5 py-2.5 font-medium whitespace-nowrap text-neutral-800">
														{entry.category_reason}
													</td>
													<td
														class="px-3.5 py-2.5 text-right font-bold whitespace-nowrap text-emerald-600"
													>
														{entry.in_quantity > 0 ? `+${entry.in_quantity}` : '-'}
													</td>
													<td
														class="px-3.5 py-2.5 text-right font-bold whitespace-nowrap text-rose-600"
													>
														{entry.out_quantity > 0 ? `-${entry.out_quantity}` : '-'}
													</td>
													<td
														class="px-3.5 py-2.5 text-right font-mono font-bold whitespace-nowrap text-neutral-900"
													>
														{entry.balance}
													</td>
													<td class="px-3.5 py-2.5 whitespace-nowrap text-neutral-600">
														{entry.executed_by_name || '-'}
													</td>
													<td class="max-w-xs truncate px-3.5 py-2.5 text-neutral-500 italic">
														{entry.notes || '-'}
													</td>
												</tr>
											{/each}
										</tbody>
									</Table>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			{/if}

			<!-- TAB 2: UNIT SERIAL / IMEI FISIK -->
			{#if detailActiveTab === 'serials' && detailItem.hasSerial}
				<div class="space-y-4">
					<!-- Filter & Pencarian Serial -->
					<div
						class="flex flex-col gap-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs sm:flex-row sm:items-center sm:justify-between"
					>
						<!-- Status Filter Pills -->
						<div class="flex flex-wrap items-center gap-1.5 text-xs">
							<button
								type="button"
								onclick={() => (detailSerialStatusFilter = 'all')}
								class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors {detailSerialStatusFilter ===
								'all'
									? 'bg-neutral-900 text-white'
									: 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'}"
							>
								Semua ({detailSerialsList.length})
							</button>
							<button
								type="button"
								onclick={() => (detailSerialStatusFilter = 'tersedia')}
								class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors {detailSerialStatusFilter ===
								'tersedia'
									? 'bg-emerald-600 text-white'
									: 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100'}"
							>
								Tersedia ({serialAvailableCount})
							</button>
							<button
								type="button"
								onclick={() => (detailSerialStatusFilter = 'dipesan')}
								class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors {detailSerialStatusFilter ===
								'dipesan'
									? 'bg-amber-600 text-white'
									: 'bg-amber-50 text-amber-700 hover:bg-amber-100'}"
							>
								Dipesan ({serialReservedCount})
							</button>
							<button
								type="button"
								onclick={() => (detailSerialStatusFilter = 'terjual')}
								class="rounded-lg px-2.5 py-1 text-xs font-semibold transition-colors {detailSerialStatusFilter ===
								'terjual'
									? 'bg-neutral-600 text-white'
									: 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'}"
							>
								Terjual ({serialSoldCount})
							</button>
						</div>

						<div class="flex items-center gap-2">
							<div class="w-56">
								<SearchInput bind:value={detailSerialSearch} placeholder="Cari nomor seri..." />
							</div>
							<Button
								variant="outline"
								size="sm"
								onclick={copyAllAvailableSerials}
								title="Salin semua nomor seri berstatus tersedia"
							>
								<svg class="mr-1 h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
									/>
								</svg>
								Salin Tersedia
							</Button>
						</div>
					</div>

					{#if detailSerialsLoading}
						<div class="flex h-48 items-center justify-center">
							<div class="flex items-center gap-3 text-sm text-neutral-500">
								<svg class="h-5 w-5 animate-spin text-neutral-500" viewBox="0 0 24 24" fill="none">
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
								<span>Memuat daftar nomor seri fisik...</span>
							</div>
						</div>
					{:else if detailSerialsError}
						<Alert variant="error" title="Gagal Memuat Nomor Seri">{detailSerialsError}</Alert>
					{:else if filteredDetailSerials.length === 0}
						<div
							class="rounded-xl border border-neutral-200 bg-white py-12 text-center text-xs text-neutral-500"
						>
							<svg
								class="mx-auto mb-2 h-8 w-8 text-neutral-300"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="1.5"
									d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z"
								/>
							</svg>
							Tidak ada nomor seri / IMEI yang sesuai dengan filter saat ini.
						</div>
					{:else}
						<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
							<div class="max-h-96 overflow-x-auto">
								<Table>
									<thead>
										<tr
											class="text-3xs border-b border-neutral-200 bg-neutral-50/80 text-left font-semibold tracking-wider text-neutral-600 uppercase"
										>
											<th class="w-12 px-4 py-2.5 text-center">No</th>
											<th class="px-4 py-2.5">Nomor Seri / IMEI</th>
											<th class="px-4 py-2.5 text-center">Status</th>
											<th class="px-4 py-2.5">Lokasi / Gudang</th>
											<th class="px-4 py-2.5">Tanggal Registrasi</th>
											<th class="px-4 py-2.5 text-right">Aksi</th>
										</tr>
									</thead>
									<tbody class="divide-y divide-neutral-100 text-xs">
										{#each filteredDetailSerials as s, idx (s.id)}
											{@const badge = getSerialStatusBadge(s.status)}
											<tr class="transition-colors hover:bg-neutral-50/80">
												<td class="text-3xs px-4 py-2.5 text-center font-mono text-neutral-400">
													{idx + 1}
												</td>
												<td class="px-4 py-2.5">
													<div class="flex items-center gap-2">
														<span class="font-mono font-bold text-neutral-900"
															>{s.serial_number}</span
														>
														<button
															type="button"
															onclick={() => {
																navigator.clipboard.writeText(s.serial_number);
																toast.success(`Nomor seri ${s.serial_number} disalin`);
															}}
															class="rounded p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700"
															title="Salin nomor seri"
														>
															<svg
																class="h-3.5 w-3.5"
																viewBox="0 0 24 24"
																fill="none"
																stroke="currentColor"
															>
																<path
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	stroke-width="2"
																	d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
																/>
															</svg>
														</button>
													</div>
												</td>
												<td class="px-4 py-2.5 text-center whitespace-nowrap">
													<Badge variant={badge.variant} size="sm">{badge.label}</Badge>
												</td>
												<td class="px-4 py-2.5 whitespace-nowrap text-neutral-600">
													{s.location_name || selectedLocation?.name || '-'}
												</td>
												<td class="px-4 py-2.5 whitespace-nowrap text-neutral-500">
													{formatDateTime(s.created_at)}
												</td>
												<td class="px-4 py-2.5 text-right whitespace-nowrap">
													<a
														href="/inventory/serials?sn={encodeURIComponent(s.serial_number)}"
														class="text-xs font-semibold text-neutral-900 hover:underline"
													>
														Buka Scanner
													</a>
												</td>
											</tr>
										{/each}
									</tbody>
								</Table>
							</div>
						</div>
					{/if}
				</div>
			{/if}

			<!-- TAB 3: RINGKASAN NILAI & AUDIT STOK -->
			{#if detailActiveTab === 'summary'}
				<div class="space-y-4">
					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-4">
						<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
							<span class="text-3xs font-medium text-neutral-400 uppercase">Kuantitas Fisik</span>
							<p class="font-mono text-xl font-bold text-neutral-900">{detailItem.quantity} unit</p>
							<span class="text-3xs text-neutral-500">Saldo barang riil di rak</span>
						</div>

						<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
							<span class="text-3xs font-medium text-emerald-600 uppercase"
								>Kuantitas Siap Jual</span
							>
							<p class="font-mono text-xl font-bold text-emerald-700">
								{detailItem.availableQuantity} unit
							</p>
							<span class="text-3xs text-neutral-500">Bebas ditransaksikan kasir</span>
						</div>

						<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
							<span class="text-3xs font-medium text-amber-600 uppercase">Kuantitas Dipesan</span>
							<p class="font-mono text-xl font-bold text-amber-700">
								{detailItem.reservedQuantity} unit
							</p>
							<span class="text-3xs text-neutral-500">Tertahan pesanan / mutasi</span>
						</div>

						<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
							<span class="text-3xs font-medium text-neutral-400 uppercase"
								>Batas Minimum Alert</span
							>
							<p class="font-mono text-xl font-bold text-neutral-700">{detailItem.minStock} unit</p>
							<span class="text-3xs text-neutral-500">Ambang batas stok menipis</span>
						</div>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<!-- Harga Retail Jual -->
						<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs">
							<span class="text-3xs font-semibold tracking-wider text-neutral-500 uppercase"
								>Harga Retail Jual</span
							>
							<p class="mt-1 font-mono text-xl font-bold text-neutral-900">
								{formatRupiah(detailItem.productPrice)}
							</p>
							<p class="mt-1 text-xs text-neutral-500">
								Patokan harga jual resmi ke konsumen di cabang ini.
							</p>
						</div>

						<!-- Harga Pokok (HPP / Modal) dengan Proteksi PBAC -->
						<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs">
							<span class="text-3xs font-semibold tracking-wider text-neutral-500 uppercase"
								>Harga Pokok (HPP) & Valuasi Persediaan</span
							>
							{#if detailItem.purchasePrice !== null && detailItem.purchasePrice !== undefined}
								<p class="mt-1 font-mono text-xl font-bold text-neutral-900">
									{formatRupiah(detailItem.purchasePrice)}
								</p>
								<p class="mt-1 text-xs font-semibold text-emerald-700">
									Total Valuasi Aset: {formatRupiah(detailItem.purchasePrice * detailItem.quantity)}
								</p>
							{:else}
								<div class="mt-2 flex items-center gap-1.5 text-neutral-400">
									<svg
										class="h-4 w-4 shrink-0 text-neutral-400"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z"
										/>
									</svg>
									<span class="font-mono text-sm font-semibold tracking-widest">••••••••</span>
								</div>
								<p class="text-3xs mt-1 text-neutral-400">
									Nilai HPP tersensor (Hanya Superadmin/Owner)
								</p>
							{/if}
						</div>
					</div>

					<!-- Metadata Audit Trail Database -->
					<div class="rounded-xl border border-neutral-200 bg-neutral-50/60 p-3.5 text-xs">
						<div class="grid grid-cols-1 gap-2 text-neutral-600 sm:grid-cols-2">
							<div class="flex items-center justify-between border-r border-neutral-200/60 pr-4">
								<span>ID Baris Stok Database:</span>
								<span class="font-mono font-medium text-neutral-900"
									>{detailItem.stockId ?? 'Belum teralokasi fisik'}</span
								>
							</div>
							<div class="flex items-center justify-between pl-2">
								<span>Pembaruan Terakhir Sistem:</span>
								<span class="font-medium text-neutral-900">{formatDate(detailItem.updatedAt)}</span>
							</div>
						</div>
					</div>
				</div>
			{/if}
		</div>

		{#snippet footer()}
			<div class="flex w-full items-center justify-between">
				<div>
					{#if canAdjustStock}
						<Button
							variant="outline"
							size="sm"
							onclick={() => {
								showDetailModal = false;
								if (detailItem) openAdjustModal(detailItem);
							}}
						>
							Lakukan Stock Opname
						</Button>
					{/if}
				</div>
				<div class="flex items-center gap-2">
					<Button variant="primary" size="sm" onclick={() => (showDetailModal = false)}>
						Tutup
					</Button>
				</div>
			</div>
		{/snippet}
	</Modal>
{/if}

<!-- ======================================================================= -->
<!-- MODAL 4: RIWAYAT STOCK OPNAME (AUDIT TRAIL LEDGER)                     -->
<!-- ======================================================================= -->
{#if showHistoryModal}
	<Modal
		bind:open={showHistoryModal}
		title={historyTargetProduct
			? `Riwayat Opname: ${historyTargetProduct.productName}`
			: `Riwayat Stock Opname — Cabang ${selectedLocation?.name ?? ''}`}
		size="3xl"
	>
		<div class="space-y-4">
			{#if historyTargetProduct}
				<div
					class="flex items-center justify-between rounded-xl border border-neutral-200 bg-neutral-50/70 p-3"
				>
					<div>
						<h4 class="text-sm font-bold text-neutral-900">{historyTargetProduct.productName}</h4>
						<p class="text-2xs font-mono text-neutral-500">
							{historyTargetProduct.productSku} • {historyTargetProduct.categoryName}
						</p>
					</div>
					<div class="text-right">
						<span class="text-3xs font-medium text-neutral-400 uppercase">Stok Fisik Saat Ini</span>
						<p class="font-mono text-sm font-bold text-neutral-900">
							{historyTargetProduct.quantity} unit
						</p>
					</div>
				</div>
			{/if}

			<!-- Filter Periode Riwayat -->
			<div
				class="flex flex-wrap items-center gap-3 rounded-xl border border-neutral-200 bg-neutral-50/70 p-3"
			>
				<div class="flex items-center gap-2 text-xs text-neutral-600">
					<span class="text-3xs font-semibold text-neutral-500 uppercase">Mulai:</span>
					<input
						type="date"
						bind:value={historyStartDate}
						class="rounded-lg border border-neutral-300 bg-white px-2.5 py-1 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:outline-hidden"
					/>
				</div>
				<div class="flex items-center gap-2 text-xs text-neutral-600">
					<span class="text-3xs font-semibold text-neutral-500 uppercase">Sampai:</span>
					<input
						type="date"
						bind:value={historyEndDate}
						class="rounded-lg border border-neutral-300 bg-white px-2.5 py-1 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:outline-hidden"
					/>
				</div>
				<div class="flex items-center gap-2">
					<Button size="sm" variant="primary" onclick={() => loadAdjustHistory(1)}>Filter</Button>
					{#if historyStartDate !== getFirstDayOfMonth() || historyEndDate !== getTodayDate()}
						<Button
							size="sm"
							variant="outline"
							onclick={() => {
								historyStartDate = getFirstDayOfMonth();
								historyEndDate = getTodayDate();
								loadAdjustHistory(1);
							}}
						>
							Reset
						</Button>
					{/if}
				</div>
			</div>

			{#if historyLoading}
				<div class="flex items-center justify-center py-12 text-xs text-neutral-400">
					<!-- Spinner -->
					<svg class="mr-2 h-5 w-5 animate-spin text-neutral-500" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						></path>
					</svg>
					Memuat rekaman riwayat stock opname...
				</div>
			{:else if historyItems.length === 0}
				<div class="flex flex-col items-center justify-center py-12 text-center">
					<div
						class="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-100 text-neutral-400"
					>
						<svg
							class="h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
					</div>
					<h5 class="mt-3 text-sm font-semibold text-neutral-800">Belum Ada Riwayat Opname</h5>
					<p class="mt-1 text-xs text-neutral-500">
						{historyTargetProduct
							? 'Produk ini belum pernah mengalami penyesuaian fisik (stock opname).'
							: 'Belum ada catatan penyesuaian fisik stok di cabang ini.'}
					</p>
				</div>
			{:else}
				<div class="overflow-x-auto rounded-xl border border-neutral-200">
					<table class="w-full text-left text-xs">
						<thead
							class="text-2xs border-b border-neutral-200 bg-neutral-50/80 font-semibold tracking-wider text-neutral-600 uppercase"
						>
							<tr>
								<th class="px-3.5 py-2.5">Tgl Opname</th>
								<th class="px-3.5 py-2.5">Waktu Rekam Sistem</th>
								{#if !historyTargetProduct}
									<th class="px-3.5 py-2.5">Produk</th>
								{/if}
								<th class="px-3.5 py-2.5 text-center">Saldo Awal → Akhir</th>
								<th class="px-3.5 py-2.5 text-center">Selisih</th>
								<th class="px-3.5 py-2.5">Alasan (Audit Trail)</th>
								<th class="px-3.5 py-2.5">Oleh</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-100 bg-white">
							{#each historyItems as adj (adj.id)}
								<tr class="transition-colors hover:bg-neutral-50/60">
									<td class="px-3.5 py-2.5 whitespace-nowrap">
										<span class="font-semibold text-neutral-900"
											>{formatOnlyDate(adj.adjustment_date)}</span
										>
									</td>
									<td class="text-2xs px-3.5 py-2.5 whitespace-nowrap text-neutral-500">
										{formatDateTime(adj.created_at)}
									</td>
									{#if !historyTargetProduct}
										<td class="px-3.5 py-2.5">
											<div class="font-medium text-neutral-900">{adj.product_name}</div>
											<div class="text-3xs font-mono text-neutral-400">{adj.product_sku}</div>
										</td>
									{/if}
									<td class="px-3.5 py-2.5 text-center font-mono whitespace-nowrap">
										<span class="text-neutral-500">{adj.previous_quantity}</span>
										<span class="mx-1 text-neutral-300">→</span>
										<strong class="font-bold text-neutral-900">{adj.new_quantity}</strong>
										<span class="text-3xs ml-0.5 text-neutral-400">unit</span>
									</td>
									<td class="px-3.5 py-2.5 text-center whitespace-nowrap">
										{#if adj.difference > 0}
											<Badge variant="success" size="sm">+{adj.difference} unit</Badge>
										{:else if adj.difference < 0}
											<Badge variant="danger" size="sm">{adj.difference} unit</Badge>
										{:else}
											<Badge variant="default" size="sm">0 unit</Badge>
										{/if}
									</td>
									<td class="px-3.5 py-2.5 text-xs text-neutral-700">
										{adj.reason}
									</td>
									<td class="text-2xs px-3.5 py-2.5 whitespace-nowrap text-neutral-500">
										{adj.adjusted_by_name}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				{#if historyTotalPages > 1}
					<div class="flex items-center justify-between pt-2">
						<span class="text-2xs text-neutral-500">
							Menampilkan {(historyPage - 1) * historyPageSize + 1} - {Math.min(
								historyPage * historyPageSize,
								historyTotalItems
							)} dari {historyTotalItems} catatan
						</span>
						<Pagination
							page={historyPage}
							totalPages={historyTotalPages}
							totalItems={historyTotalItems}
							limit={historyPageSize}
							onPageChange={(p) => loadAdjustHistory(p)}
						/>
					</div>
				{/if}
			{/if}
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end">
				<Button variant="primary" size="sm" onclick={() => (showHistoryModal = false)}>
					Tutup
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}
