<script lang="ts">
	/**
	 * Halaman Manajemen Mutasi Stok Antar Cabang (Stock Transfers).
	 *
	 * Fitur Kunci:
	 * - Siklus Hidup Lengkap Mutasi: pending_approval -> approved -> in_transit -> received (atau rejected).
	 * - Multi-Item Transfer Builder dengan pengecekan stok tersedia di cabang asal.
	 * - Approval & Rejection dengan pencatatan alasan penolakan dan pelepasan reservasi otomatis.
	 * - Ekspedisi Pengiriman (Ship) yang memotong stok fisik asal.
	 * - Penerimaan Barang Gudang (Receive) yang menambah stok fisik tujuan.
	 * - Kartu Statistik KPI Mutasi Real-Time.
	 * - Filter Status Cepat, Filter Cabang Asal & Tujuan, dan Pencarian No. Surat Jalan.
	 * - Modal Surat Jalan Digital Lengkap dengan Timeline Status Stepper.
	 * - Zero-Warning TypeScript & Svelte 5 Runes ($derived.by).
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listStockTransfers,
		createStockTransfer,
		approveStockTransfer,
		rejectStockTransfer,
		shipStockTransfer,
		receiveStockTransfer,
		listLocations,
		listProducts,
		listStocks
	} from '@erp/api-client';
	import type {
		StockTransferResponse,
		LocationResponse,
		ProductResponse,
		StockResponse,
		TransferStatus
	} from '@erp/types';
	import { TRANSFER_STATUS_LABELS, ApiError } from '@erp/types';
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

	// ── 1. State Data Utama ───────────────────────────────────────────────────
	let transfers = $state<StockTransferResponse[]>([]);
	let locations = $state<LocationResponse[]>([]);
	let products = $state<ProductResponse[]>([]);

	let loading = $state(true);
	let error = $state<string | null>(null);

	// ── 2. Filter & Pencarian ─────────────────────────────────────────────────
	let statusFilter = $state<string>('all');
	let fromLocationFilter = $state<string>('');
	let toLocationFilter = $state<string>('');
	let searchQuery = $state('');

	// Pagination
	let currentPage = $state(1);
	let pageSize = $state(10);

	// ── 3. State Modal Buat Permohonan Mutasi ──────────────────────────────────
	let showCreateModal = $state(false);
	let formFromLocationId = $state('');
	let formToLocationId = $state('');
	let formNotes = $state('');
	let createSubmitting = $state(false);
	let createError = $state<string | null>(null);

	// Multi-item builder state
	interface TransferItemDraft {
		productId: string;
		productName: string;
		productSku: string;
		quantity: number;
		availableStock: number;
	}
	let transferItemsDraft = $state<TransferItemDraft[]>([]);
	let itemSelectedProductId = $state('');
	let itemQuantity = $state<number>(1);
	let originStocks = $state<StockResponse[]>([]);
	let loadingOriginStocks = $state(false);

	// ── 4. State Modal Detail / Surat Jalan ───────────────────────────────────
	let showDetailModal = $state(false);
	let selectedTransfer = $state<StockTransferResponse | null>(null);
	let actionSubmitting = $state(false);

	// ── 5. State Modal Tolak Mutasi (Reject) ──────────────────────────────────
	let showRejectModal = $state(false);
	let rejectingTransferId = $state<string | null>(null);
	let rejectReason = $state('');
	let rejectSubmitting = $state(false);
	let rejectError = $state<string | null>(null);

	// ── 6. Helper Mapping & Format ────────────────────────────────────────────
	const locationMap = $derived<Map<string, LocationResponse>>(
		new Map(locations.map((l) => [l.id, l]))
	);

	const productMap = $derived<Map<string, ProductResponse>>(
		new Map(products.map((p) => [p.id, p]))
	);

	const locationSelectOptions = $derived<Select2Option[]>(
		locations.map((loc) => ({
			value: loc.id,
			label: `${loc.name} (${loc.code})`,
			subtext: loc.type === 'physical' ? 'Toko / Cabang Fisik' : 'Gudang Online Storefront'
		}))
	);

	const filterLocationOptions = $derived<Select2Option[]>([
		{ value: '', label: 'Semua Cabang' },
		...locations.map((loc) => ({
			value: loc.id,
			label: `${loc.name} (${loc.code})`
		}))
	]);

	// Dropdown pilihan produk saat menyusun mutasi barang
	const productSelectOptions = $derived.by<Select2Option[]>(() => {
		const stockLookup = new Map<string, number>();
		for (const st of originStocks) {
			stockLookup.set(st.product_id, Math.max(0, st.quantity - st.reserved_quantity));
		}

		return products.map((prod) => {
			const avail = stockLookup.get(prod.id) ?? 0;
			return {
				value: prod.id,
				label: `${prod.name} (${prod.sku})`,
				subtext: formFromLocationId
					? `Tersedia di asal: ${avail} unit`
					: 'Pilih cabang asal terlebih dahulu'
			};
		});
	});

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

	// Hitung total unit yang dimutasi dalam satu dokumen
	function calculateTotalUnits(items: StockTransferResponse['items']): number {
		return items.reduce((acc, it) => acc + it.quantity, 0);
	}

	// ── 7. Statistik KPI Mutasi ───────────────────────────────────────────────
	const stats = $derived.by(() => {
		let pending = 0;
		let approved = 0;
		let inTransit = 0;
		let received = 0;
		let rejected = 0;

		for (const t of transfers) {
			switch (t.status) {
				case 'pending_approval':
					pending++;
					break;
				case 'approved':
					approved++;
					break;
				case 'in_transit':
					inTransit++;
					break;
				case 'received':
					received++;
					break;
				case 'rejected':
					rejected++;
					break;
			}
		}

		return {
			total: transfers.length,
			pending,
			approved,
			inTransit,
			received,
			rejected
		};
	});

	// ── 8. Filter & Pagination ────────────────────────────────────────────────
	const filteredTransfers = $derived.by<StockTransferResponse[]>(() => {
		let list = transfers;

		// Filter Tab Status
		if (statusFilter !== 'all') {
			list = list.filter((t: StockTransferResponse) => t.status === statusFilter);
		}

		// Filter Cabang Asal
		if (fromLocationFilter) {
			list = list.filter((t: StockTransferResponse) => t.from_location_id === fromLocationFilter);
		}

		// Filter Cabang Tujuan
		if (toLocationFilter) {
			list = list.filter((t: StockTransferResponse) => t.to_location_id === toLocationFilter);
		}

		// Pencarian Nomor Surat Jalan / Catatan
		if (searchQuery.trim()) {
			const q = searchQuery.toLowerCase().trim();
			list = list.filter((t: StockTransferResponse) => {
				const num = t.transfer_number.toLowerCase();
				const notes = t.notes.toLowerCase();
				const fromLoc = (locationMap.get(t.from_location_id)?.name ?? '').toLowerCase();
				const toLoc = (locationMap.get(t.to_location_id)?.name ?? '').toLowerCase();
				return num.includes(q) || notes.includes(q) || fromLoc.includes(q) || toLoc.includes(q);
			});
		}

		return list;
	});

	const totalItems = $derived(filteredTransfers.length);
	const totalPages = $derived(Math.max(1, Math.ceil(totalItems / pageSize)));
	const paginatedTransfers = $derived.by<StockTransferResponse[]>(() => {
		const start = (currentPage - 1) * pageSize;
		return filteredTransfers.slice(start, start + pageSize);
	});

	// ── 9. Data Fetching ──────────────────────────────────────────────────────
	async function loadData() {
		loading = true;
		error = null;
		try {
			const token = getToken();
			if (!token) throw new Error('Sesi autentikasi telah berakhir. Silakan login kembali.');

			const [trfList, locList, prodRes] = await Promise.all([
				listStockTransfers(token),
				listLocations(token, true),
				listProducts(token, { limit: 100 })
			]);

			transfers = trfList;
			locations = locList;
			products = prodRes.data;
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat data mutasi stok.';
			}
		} finally {
			loading = false;
		}
	}

	// Mengambil saldo stok cabang asal untuk validasi kuantitas saat draft mutasi
	async function handleOriginLocationChange(locId: string) {
		formFromLocationId = locId;
		transferItemsDraft = [];
		itemSelectedProductId = '';
		if (!locId) {
			originStocks = [];
			return;
		}

		loadingOriginStocks = true;
		try {
			const token = getToken();
			if (!token) return;
			originStocks = await listStocks(token, locId);
		} catch {
			originStocks = [];
		} finally {
			loadingOriginStocks = false;
		}
	}

	// ── 10. Multi-Item Builder Actions ────────────────────────────────────────
	function addItemToDraft() {
		createError = null;
		if (!itemSelectedProductId) {
			createError = 'Pilih produk yang akan dimutasi terlebih dahulu.';
			return;
		}
		if (itemQuantity <= 0) {
			createError = 'Kuantitas transfer minimal 1 unit.';
			return;
		}

		const prod = productMap.get(itemSelectedProductId);
		if (!prod) return;

		// Cek apakah produk sudah ada di daftar draft
		const existingIndex = transferItemsDraft.findIndex((i) => i.productId === prod.id);
		if (existingIndex >= 0) {
			createError = `Produk "${prod.name}" sudah ada dalam daftar. Hapus terlebih dahulu jika ingin mengganti kuantitas.`;
			return;
		}

		// Cek ketersediaan stok fisik di cabang asal
		const st = originStocks.find((s) => s.product_id === prod.id);
		const avail = st ? Math.max(0, st.quantity - st.reserved_quantity) : 0;
		if (itemQuantity > avail) {
			createError = `Kuantitas transfer (${itemQuantity} unit) melebihi stok siap jual di cabang asal (${avail} unit).`;
			return;
		}

		transferItemsDraft.push({
			productId: prod.id,
			productName: prod.name,
			productSku: prod.sku,
			quantity: itemQuantity,
			availableStock: avail
		});

		// Reset form input item
		itemSelectedProductId = '';
		itemQuantity = 1;
	}

	function removeItemFromDraft(index: number) {
		transferItemsDraft.splice(index, 1);
	}

	// ── 11. Eksekusi Buat Mutasi Stok (Create) ─────────────────────────────────
	function openCreateModal() {
		formFromLocationId = locations.length > 0 ? locations[0].id : '';
		formToLocationId = locations.length > 1 ? locations[1].id : '';
		formNotes = '';
		transferItemsDraft = [];
		itemSelectedProductId = '';
		itemQuantity = 1;
		createError = null;
		if (formFromLocationId) {
			handleOriginLocationChange(formFromLocationId);
		}
		showCreateModal = true;
	}

	async function handleExecuteCreate() {
		createError = null;
		if (!formFromLocationId || !formToLocationId) {
			createError = 'Cabang asal dan cabang tujuan wajib dipilih.';
			return;
		}
		if (formFromLocationId === formToLocationId) {
			createError = 'Cabang asal dan cabang tujuan tidak boleh sama.';
			return;
		}
		if (transferItemsDraft.length === 0) {
			createError = 'Tambahkan minimal 1 item barang yang akan dimutasi.';
			return;
		}

		createSubmitting = true;
		try {
			const token = getToken();
			if (!token) throw new Error('Token tidak ditemukan');

			await createStockTransfer(token, {
				from_location_id: formFromLocationId,
				to_location_id: formToLocationId,
				notes: formNotes.trim() || undefined,
				items: transferItemsDraft.map((i) => ({
					product_id: i.productId,
					quantity: i.quantity
				}))
			});

			toast.success('Permohonan mutasi stok berhasil diajukan! Stok asal otomatis dicadangkan.');
			showCreateModal = false;
			await loadData();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				createError = err.message;
			} else if (err instanceof Error) {
				createError = err.message;
			} else {
				createError = 'Gagal mengajukan permohonan mutasi stok.';
			}
		} finally {
			createSubmitting = false;
		}
	}

	// ── 12. Eksekusi Aksi Lifecycle Mutasi (Approve, Reject, Ship, Receive) ────
	async function handleApprove(trf: StockTransferResponse) {
		actionSubmitting = true;
		try {
			const token = getToken();
			if (!token) throw new Error('Token tidak ditemukan');
			await approveStockTransfer(token, trf.id);
			toast.success(`Dokumen mutasi "${trf.transfer_number}" berhasil disetujui!`);
			if (selectedTransfer?.id === trf.id) {
				showDetailModal = false;
			}
			await loadData();
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Gagal menyetujui transfer';
			toast.error(msg);
		} finally {
			actionSubmitting = false;
		}
	}

	function openRejectModal(trfId: string) {
		rejectingTransferId = trfId;
		rejectReason = '';
		rejectError = null;
		showRejectModal = true;
	}

	async function handleExecuteReject() {
		if (!rejectingTransferId) return;
		if (!rejectReason.trim()) {
			rejectError = 'Silakan tuliskan alasan penolakan mutasi.';
			return;
		}

		rejectSubmitting = true;
		try {
			const token = getToken();
			if (!token) throw new Error('Token tidak ditemukan');
			await rejectStockTransfer(token, rejectingTransferId, rejectReason.trim());
			toast.success('Permohonan mutasi telah ditolak. Cadangan stok asal berhasil dilepaskan.');
			showRejectModal = false;
			if (selectedTransfer?.id === rejectingTransferId) {
				showDetailModal = false;
			}
			await loadData();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				rejectError = err.message;
			} else if (err instanceof Error) {
				rejectError = err.message;
			} else {
				rejectError = 'Gagal menolak permohonan mutasi.';
			}
		} finally {
			rejectSubmitting = false;
		}
	}

	async function handleShip(trf: StockTransferResponse) {
		actionSubmitting = true;
		try {
			const token = getToken();
			if (!token) throw new Error('Token tidak ditemukan');
			await shipStockTransfer(token, trf.id);
			toast.success(`Barang "${trf.transfer_number}" telah dikonfirmasi berangkat (Dalam Pengiriman)!`);
			if (selectedTransfer?.id === trf.id) {
				showDetailModal = false;
			}
			await loadData();
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Gagal mengonfirmasi pengiriman';
			toast.error(msg);
		} finally {
			actionSubmitting = false;
		}
	}

	async function handleReceive(trf: StockTransferResponse) {
		actionSubmitting = true;
		try {
			const token = getToken();
			if (!token) throw new Error('Token tidak ditemukan');
			await receiveStockTransfer(token, trf.id);
			toast.success(`Barang "${trf.transfer_number}" berhasil diterima lengkap di cabang tujuan!`);
			if (selectedTransfer?.id === trf.id) {
				showDetailModal = false;
			}
			await loadData();
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Gagal mengonfirmasi penerimaan';
			toast.error(msg);
		} finally {
			actionSubmitting = false;
		}
	}

	function openDetailModal(trf: StockTransferResponse) {
		selectedTransfer = trf;
		showDetailModal = true;
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>Mutasi Stok Antar Cabang — Inventaris & Stok Gen-E</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header & Tombol Buat Mutasi -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<h1 class="text-xl font-bold tracking-tight text-neutral-900">Mutasi Stok Antar Cabang</h1>
				<Badge variant="primary" size="sm">Distribusi & Logistik</Badge>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Kelola alur distribusi permohonan mutasi barang, persetujuan pimpinan, ekspedisi pengiriman, dan konfirmasi penerimaan gudang.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<Button
				variant="outline"
				size="sm"
				loading={loading}
				onclick={loadData}
			>
				<!-- Refresh Icon -->
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
				</svg>
			</Button>

			<Button variant="primary" size="sm" onclick={openCreateModal}>
				<!-- Plus Icon -->
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Buat Permohonan Mutasi
			</Button>
		</div>
	</div>

	<!-- Error Alert Global -->
	{#if error}
		<Alert variant="error" title="Gagal Memuat Data">{error}</Alert>
	{/if}

	<!-- Kartu Statistik KPI Mutasi -->
	<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
		<!-- Total Mutasi -->
		<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
			<div class="flex items-center justify-between">
				<span class="text-3xs font-semibold tracking-wider text-neutral-400 uppercase">Total Mutasi</span>
				<!-- Document Text Icon -->
				<svg class="h-4 w-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
				</svg>
			</div>
			<div class="mt-2 text-xl font-bold tracking-tight text-neutral-900">{stats.total}</div>
			<span class="text-3xs text-neutral-500">Surat jalan diterbitkan</span>
		</div>

		<!-- Menunggu Persetujuan -->
		<div class="rounded-xl border border-amber-200/80 bg-amber-50/40 p-3.5 shadow-2xs">
			<div class="flex items-center justify-between">
				<span class="text-3xs font-semibold tracking-wider text-amber-700 uppercase">Menunggu Approval</span>
				<!-- Clock Icon -->
				<svg class="h-4 w-4 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
			<div class="mt-2 text-xl font-bold tracking-tight text-amber-800">{stats.pending}</div>
			<span class="text-3xs text-amber-600">Stok asal dibooking</span>
		</div>

		<!-- Disetujui -->
		<div class="rounded-xl border border-primary-200/80 bg-primary-50/40 p-3.5 shadow-2xs">
			<div class="flex items-center justify-between">
				<span class="text-3xs font-semibold tracking-wider text-primary-700 uppercase">Disetujui</span>
				<!-- Check Badge Icon -->
				<svg class="h-4 w-4 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
			<div class="mt-2 text-xl font-bold tracking-tight text-primary-800">{stats.approved}</div>
			<span class="text-3xs text-primary-600">Siap dipacking & kirim</span>
		</div>

		<!-- Dalam Pengiriman -->
		<div class="rounded-xl border border-indigo-200/80 bg-indigo-50/40 p-3.5 shadow-2xs">
			<div class="flex items-center justify-between">
				<span class="text-3xs font-semibold tracking-wider text-indigo-700 uppercase">Dalam Perjalanan</span>
				<!-- Truck Icon -->
				<svg class="h-4 w-4 text-indigo-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M8.25 18.75a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m3 0h6m-9 0H3.375a1.125 1.125 0 01-1.125-1.125V14.25m17.25 4.5a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m3 0h1.125c.621 0 1.129-.504 1.09-1.124a17.902 17.902 0 00-3.213-9.193 2.056 2.056 0 00-1.58-.86H14.25M16.5 18.75h-2.25m0-11.175V3.75A1.125 1.125 0 0013.125 2.625H3.375A1.125 1.125 0 002.25 3.75v10.5m14.25-6.675l-4.5 4.5" />
				</svg>
			</div>
			<div class="mt-2 text-xl font-bold tracking-tight text-indigo-800">{stats.inTransit}</div>
			<span class="text-3xs text-indigo-600">Armada ekspedisi jalan</span>
		</div>

		<!-- Diterima Lengkap -->
		<div class="rounded-xl border border-emerald-200/80 bg-emerald-50/40 p-3.5 shadow-2xs">
			<div class="flex items-center justify-between">
				<span class="text-3xs font-semibold tracking-wider text-emerald-700 uppercase">Diterima Lengkap</span>
				<!-- Inbox Arrow Down Icon -->
				<svg class="h-4 w-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m6 4.125l2.25 2.25m0 0l2.25-2.25m-2.25 2.25V3.75" />
				</svg>
			</div>
			<div class="mt-2 text-xl font-bold tracking-tight text-emerald-800">{stats.received}</div>
			<span class="text-3xs text-emerald-600">Stok tujuan bertambah</span>
		</div>

		<!-- Ditolak -->
		<div class="rounded-xl border border-rose-200/80 bg-rose-50/40 p-3.5 shadow-2xs">
			<div class="flex items-center justify-between">
				<span class="text-3xs font-semibold tracking-wider text-rose-700 uppercase">Ditolak</span>
				<!-- X Circle Icon -->
				<svg class="h-4 w-4 text-rose-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
			<div class="mt-2 text-xl font-bold tracking-tight text-rose-800">{stats.rejected}</div>
			<span class="text-3xs text-rose-600">Cadangan dilepaskan</span>
		</div>
	</div>

	<!-- Toolbar Filter & Pencarian -->
	<div class="flex flex-col gap-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
		<!-- Quick Filter Tab Pills -->
		<div class="flex flex-wrap items-center gap-1.5 border-b border-neutral-100 pb-3">
			<button
				type="button"
				onclick={() => { statusFilter = 'all'; currentPage = 1; }}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter === 'all'
					? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-neutral-600 hover:border-neutral-300 hover:bg-neutral-50 shadow-2xs'}"
			>
				Semua
				<span class="rounded-full px-1.5 py-0.2 text-3xs font-semibold {statusFilter === 'all' ? 'bg-neutral-800 text-neutral-300' : 'bg-neutral-100 text-neutral-600'}">
					{stats.total}
				</span>
			</button>

			<button
				type="button"
				onclick={() => { statusFilter = 'pending_approval'; currentPage = 1; }}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter === 'pending_approval'
					? 'border-amber-600 bg-amber-600 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-amber-800 hover:border-neutral-300 hover:bg-amber-50/50 shadow-2xs'}"
			>
				Menunggu Approval
				<span class="rounded-full px-1.5 py-0.2 text-3xs font-semibold {statusFilter === 'pending_approval' ? 'bg-amber-700 text-white' : 'bg-amber-50 text-amber-800 border border-amber-200/60'}">
					{stats.pending}
				</span>
			</button>

			<button
				type="button"
				onclick={() => { statusFilter = 'approved'; currentPage = 1; }}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter === 'approved'
					? 'border-primary-700 bg-primary-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-primary-800 hover:border-neutral-300 hover:bg-primary-50/50 shadow-2xs'}"
			>
				Disetujui
				<span class="rounded-full px-1.5 py-0.2 text-3xs font-semibold {statusFilter === 'approved' ? 'bg-primary-800 text-white' : 'bg-primary-50 text-primary-800 border border-primary-200/60'}">
					{stats.approved}
				</span>
			</button>

			<button
				type="button"
				onclick={() => { statusFilter = 'in_transit'; currentPage = 1; }}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter === 'in_transit'
					? 'border-indigo-700 bg-indigo-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-indigo-800 hover:border-neutral-300 hover:bg-indigo-50/50 shadow-2xs'}"
			>
				Dalam Pengiriman
				<span class="rounded-full px-1.5 py-0.2 text-3xs font-semibold {statusFilter === 'in_transit' ? 'bg-indigo-800 text-white' : 'bg-indigo-50 text-indigo-800 border border-indigo-200/60'}">
					{stats.inTransit}
				</span>
			</button>

			<button
				type="button"
				onclick={() => { statusFilter = 'received'; currentPage = 1; }}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter === 'received'
					? 'border-emerald-700 bg-emerald-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-emerald-800 hover:border-neutral-300 hover:bg-emerald-50/50 shadow-2xs'}"
			>
				Diterima
				<span class="rounded-full px-1.5 py-0.2 text-3xs font-semibold {statusFilter === 'received' ? 'bg-emerald-800 text-white' : 'bg-emerald-50 text-emerald-800 border border-emerald-200/60'}">
					{stats.received}
				</span>
			</button>

			<button
				type="button"
				onclick={() => { statusFilter = 'rejected'; currentPage = 1; }}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter === 'rejected'
					? 'border-rose-700 bg-rose-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-rose-800 hover:border-neutral-300 hover:bg-rose-50/50 shadow-2xs'}"
			>
				Ditolak
				<span class="rounded-full px-1.5 py-0.2 text-3xs font-semibold {statusFilter === 'rejected' ? 'bg-rose-800 text-white' : 'bg-rose-50 text-rose-800 border border-rose-200/60'}">
					{stats.rejected}
				</span>
			</button>
		</div>

		<!-- Filter Dropdown & Search -->
		<div class="grid grid-cols-1 gap-2.5 sm:grid-cols-3">
			<div>
				<Select2
					options={filterLocationOptions}
					value={fromLocationFilter}
					placeholder="Cabang Asal (Semua)"
					searchPlaceholder="Cari asal..."
					clearable={true}
					onchange={(val) => { fromLocationFilter = String(val); currentPage = 1; }}
				/>
			</div>

			<div>
				<Select2
					options={filterLocationOptions}
					value={toLocationFilter}
					placeholder="Cabang Tujuan (Semua)"
					searchPlaceholder="Cari tujuan..."
					clearable={true}
					onchange={(val) => { toLocationFilter = String(val); currentPage = 1; }}
				/>
			</div>

			<div>
				<SearchInput
					bind:value={searchQuery}
					placeholder="Cari No. Transfer atau Catatan..."
					onsearch={() => { currentPage = 1; }}
				/>
			</div>
		</div>
	</div>

	<!-- Tabel Data Mutasi Stok -->
	<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
		<Table empty={paginatedTransfers.length === 0} emptyMessage={loading ? 'Memuat dokumen mutasi stok...' : 'Tidak ada dokumen mutasi yang cocok dengan filter.'}>
			<thead>
				<tr class="border-b border-neutral-200 bg-neutral-50/80 text-left text-2xs font-semibold tracking-wider text-neutral-600 uppercase">
					<th class="px-4 py-3">No. Transfer</th>
					<th class="px-4 py-3">Rute Ekspedisi</th>
					<th class="px-4 py-3 text-center">Total Item & Unit</th>
					<th class="px-4 py-3">Pemohon & Tanggal</th>
					<th class="px-4 py-3 text-center">Status Alur</th>
					<th class="px-4 py-3 text-right">Aksi Tindakan</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-100 text-xs">
				{#each paginatedTransfers as trf (trf.id)}
					<tr class="transition-colors hover:bg-neutral-50/60">
						<!-- No. Transfer -->
						<td class="px-4 py-3.5">
							<button
								type="button"
								onclick={() => openDetailModal(trf)}
								class="font-mono text-xs font-bold text-neutral-900 hover:text-primary-700 hover:underline text-left block"
							>
								{trf.transfer_number}
							</button>
							{#if trf.notes}
								<p class="mt-0.5 text-3xs text-neutral-500 line-clamp-1 italic">"{trf.notes}"</p>
							{/if}
						</td>

						<!-- Rute Ekspedisi: Asal -> Tujuan -->
						<td class="px-4 py-3.5">
							<div class="flex items-center gap-2">
								<div class="max-w-[130px] truncate">
									<span class="font-medium text-neutral-800 text-xs">{locationMap.get(trf.from_location_id)?.name ?? 'Cabang Asal'}</span>
									<span class="block font-mono text-3xs text-neutral-400">{locationMap.get(trf.from_location_id)?.code ?? '-'}</span>
								</div>

								<!-- Arrow Right Icon -->
								<div class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-neutral-100 text-neutral-500">
									<svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
										<path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
									</svg>
								</div>

								<div class="max-w-[130px] truncate">
									<span class="font-medium text-neutral-800 text-xs">{locationMap.get(trf.to_location_id)?.name ?? 'Cabang Tujuan'}</span>
									<span class="block font-mono text-3xs text-neutral-400">{locationMap.get(trf.to_location_id)?.code ?? '-'}</span>
								</div>
							</div>
						</td>

						<!-- Total Item & Unit -->
						<td class="px-4 py-3.5 text-center">
							<div class="inline-flex items-center gap-1.5">
								<span class="font-bold text-neutral-900">{calculateTotalUnits(trf.items)}</span>
								<span class="text-neutral-500 text-3xs">unit</span>
								<span class="text-neutral-300">•</span>
								<span class="text-neutral-500 text-3xs">{trf.items.length} SKU</span>
							</div>
						</td>

						<!-- Pemohon & Tanggal -->
						<td class="px-4 py-3.5 text-neutral-600">
							<div class="text-xs font-medium text-neutral-800">{trf.requested_by_name || trf.requested_by}</div>
							<div class="text-3xs text-neutral-400">{formatDate(trf.created_at)}</div>
						</td>

						<!-- Status Badge -->
						<td class="px-4 py-3.5 text-center">
							{#if trf.status === 'pending_approval'}
								<Badge variant="warning" size="sm">Menunggu Approval</Badge>
							{:else if trf.status === 'approved'}
								<Badge variant="primary" size="sm">Disetujui</Badge>
							{:else if trf.status === 'in_transit'}
								<Badge variant="purple" size="sm">Dalam Pengiriman</Badge>
							{:else if trf.status === 'received'}
								<Badge variant="success" size="sm">Diterima Lengkap</Badge>
							{:else if trf.status === 'rejected'}
								<Badge variant="danger" size="sm">Ditolak</Badge>
							{:else}
								<Badge variant="default" size="sm">{trf.status}</Badge>
							{/if}
						</td>

						<!-- Aksi Tindakan Sesuai Lifecycle -->
						<td class="px-4 py-3.5 text-right whitespace-nowrap">
							<ActionMenu
								items={[
									{
										label: 'Lihat Rincian Surat Jalan',
										icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>`,
										onClick: () => openDetailModal(trf)
									},
									...(trf.status === 'pending_approval'
										? [
												{
													label: 'Setujui Permohonan Mutasi',
													icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M4.5 12.75l6 6 9-13.5" /></svg>`,
													variant: 'primary' as const,
													divider: true,
													disabled: actionSubmitting,
													onClick: () => handleApprove(trf)
												},
												{
													label: 'Tolak Permohonan Mutasi',
													icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M6 18L18 6M6 6l12 12" /></svg>`,
													variant: 'danger' as const,
													disabled: actionSubmitting,
													onClick: () => openRejectModal(trf.id)
												}
											]
										: []),
									...(trf.status === 'approved'
										? [
												{
													label: 'Kirim Armada (Ship)',
													icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M8.25 18.75a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m3 0h6m-9 0H3.375a1.125 1.125 0 01-1.125-1.125V14.25m17.25 4.5a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m3 0h1.125c.621 0 1.129-.504 1.09-1.124a17.902 17.902 0 00-3.213-9.193 2.056 2.056 0 00-1.58-.86H14.25M16.5 18.75h-2.25m0-11.175V3.75A1.125 1.125 0 0013.125 2.625H3.375A1.125 1.125 0 002.25 3.75v10.5m14.25-6.675l-4.5 4.5" /></svg>`,
													variant: 'primary' as const,
													divider: true,
													disabled: actionSubmitting,
													onClick: () => handleShip(trf)
												}
											]
										: []),
									...(trf.status === 'in_transit'
										? [
												{
													label: 'Konfirmasi Terima Barang (Receive)',
													icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>`,
													variant: 'primary' as const,
													divider: true,
													disabled: actionSubmitting,
													onClick: () => handleReceive(trf)
												}
											]
										: [])
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
				Menampilkan {(currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, totalItems)} dari {totalItems} mutasi
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
<!-- MODAL 1: BUAT PERMOHONAN MUTASI STOK BARU (MULTI-ITEM)                  -->
<!-- ======================================================================= -->
{#if showCreateModal}
	<Modal bind:open={showCreateModal} title="Buat Permohonan Mutasi Stok" size="2xl">
		<div class="space-y-4">
			{#if createError}
				<Alert variant="error" title="Gagal Mengajukan Mutasi">{createError}</Alert>
			{/if}

			<!-- Rute Asal & Tujuan -->
			<div class="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
				<div>
					<label for="create-from-location" class="block text-2xs font-semibold tracking-wide text-neutral-700 uppercase mb-1">
						Cabang Asal (Pengirim) <span class="text-rose-500">*</span>
					</label>
					<select
						id="create-from-location"
						bind:value={formFromLocationId}
						onchange={() => handleOriginLocationChange(formFromLocationId)}
						class="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:outline-hidden focus:ring-1 focus:ring-neutral-900"
					>
						{#each locations as loc}
							<option value={loc.id}>{loc.name} ({loc.code})</option>
						{/each}
					</select>
					<p class="mt-1 text-3xs text-neutral-500">Stok barang di cabang ini akan dicadangkan saat permohonan diajukan.</p>
				</div>

				<div>
					<label for="create-to-location" class="block text-2xs font-semibold tracking-wide text-neutral-700 uppercase mb-1">
						Cabang Tujuan (Penerima) <span class="text-rose-500">*</span>
					</label>
					<select
						id="create-to-location"
						bind:value={formToLocationId}
						class="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:outline-hidden focus:ring-1 focus:ring-neutral-900"
					>
						{#each locations as loc}
							{#if loc.id !== formFromLocationId}
								<option value={loc.id}>{loc.name} ({loc.code})</option>
							{/if}
						{/each}
					</select>
					<p class="mt-1 text-3xs text-neutral-500">Stok fisik cabang tujuan akan bertambah saat barang dikonfirmasi tiba.</p>
				</div>
			</div>

			<!-- Catatan / Surat Jalan -->
			<div>
				<Input
					label="Keterangan / Nomor Surat Jalan Fisik"
					bind:value={formNotes}
					placeholder="Contoh: Pengiriman unit display & stok penjualan cabang Surabaya..."
				/>
			</div>

			<!-- Multi-Item Builder Section -->
			<div class="rounded-xl border border-neutral-200/80 bg-neutral-50/60 p-3.5 space-y-3">
				<div class="flex items-center justify-between">
					<h4 class="text-xs font-bold text-neutral-900 uppercase tracking-wide">Pilih Barang yang Dimutasi</h4>
					<span class="text-3xs text-neutral-500">
						{transferItemsDraft.length} produk ditambahkan ({transferItemsDraft.reduce((a, b) => a + b.quantity, 0)} unit)
					</span>
				</div>

				<!-- Item Adder Row -->
				<div class="grid grid-cols-1 gap-2.5 sm:grid-cols-12 sm:items-end">
					<div class="sm:col-span-8">
						<Select2
							label="Produk"
							options={productSelectOptions}
							value={itemSelectedProductId}
							placeholder={loadingOriginStocks ? 'Memuat ketersediaan stok...' : 'Cari & pilih produk...'}
							searchPlaceholder="Ketik nama atau SKU produk..."
							onchange={(val) => { itemSelectedProductId = String(val); }}
						/>
					</div>

					<div class="sm:col-span-2">
						<Input
							label="Jumlah (Unit)"
							type="number"
							bind:value={itemQuantity}
							placeholder="1"
							required
						/>
					</div>

					<div class="sm:col-span-2">
						<Button
							variant="outline"
							fullWidth
							onclick={addItemToDraft}
							disabled={!itemSelectedProductId || loadingOriginStocks}
						>
							+ Tambah
						</Button>
					</div>
				</div>

				<!-- Daftar Item yang Telah Ditambahkan -->
				{#if transferItemsDraft.length > 0}
					<div class="overflow-hidden rounded-lg border border-neutral-200 bg-white">
						<table class="w-full text-left text-xs">
							<thead>
								<tr class="border-b border-neutral-200 bg-neutral-50 text-2xs font-semibold text-neutral-500 uppercase">
									<th class="px-3 py-2">Nama Produk & SKU</th>
									<th class="px-3 py-2 text-center">Stok Asal</th>
									<th class="px-3 py-2 text-center">Kuantitas Transfer</th>
									<th class="px-3 py-2 text-right">Aksi</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100">
								{#each transferItemsDraft as draft, idx}
									<tr>
										<td class="px-3 py-2">
											<span class="font-medium text-neutral-900 block">{draft.productName}</span>
											<span class="font-mono text-3xs text-neutral-500">{draft.productSku}</span>
										</td>
										<td class="px-3 py-2 text-center font-mono text-neutral-600">
											{draft.availableStock} unit
										</td>
										<td class="px-3 py-2 text-center font-mono font-bold text-neutral-900">
											{draft.quantity} unit
										</td>
										<td class="px-3 py-2 text-right">
											<button
												type="button"
												onclick={() => removeItemFromDraft(idx)}
												class="text-rose-600 hover:text-rose-800 text-2xs font-semibold"
											>
												Hapus
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<div class="rounded-lg border border-dashed border-neutral-300 p-4 text-center text-xs text-neutral-400">
						Belum ada barang yang ditambahkan ke surat jalan ini.
					</div>
				{/if}
			</div>

			<!-- Catatan Edukatif Invarian -->
			<div class="rounded-lg bg-neutral-100 p-2.5 text-3xs text-neutral-500">
				<strong>Invarian Bisnis Mutasi:</strong> Saat permohonan dibuat, kuantitas produk di cabang asal otomatis
				dialokasikan ke <code class="rounded bg-neutral-200 px-1 py-0.5 font-mono text-neutral-800">ReservedQuantity</code>
				sehingga tidak dapat dijual ke kasir lain selama menunggu persetujuan.
			</div>
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end gap-2">
				<Button variant="outline" onclick={() => (showCreateModal = false)} disabled={createSubmitting}>
					Batal
				</Button>
				<Button
					variant="primary"
					loading={createSubmitting}
					onclick={handleExecuteCreate}
					disabled={transferItemsDraft.length === 0}
				>
					Ajukan Permohonan Mutasi
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}

<!-- ======================================================================= -->
<!-- MODAL 2: DETAIL SURAT JALAN & TIMELINE STATUS ALUR                      -->
<!-- ======================================================================= -->
{#if showDetailModal && selectedTransfer}
	<Modal bind:open={showDetailModal} title="Rincian Surat Jalan Mutasi Stok" size="3xl">
		<div class="space-y-4">
			<!-- Header Surat Jalan -->
			<div class="flex flex-wrap items-start justify-between gap-3 rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
				<div>
					<span class="font-mono text-xs font-bold text-neutral-900">{selectedTransfer.transfer_number}</span>
					<div class="mt-1 flex items-center gap-2 text-3xs text-neutral-500">
						<span>Diajukan oleh: <strong class="text-neutral-700">{selectedTransfer.requested_by_name || selectedTransfer.requested_by}</strong></span>
						<span>•</span>
						<span>{formatDate(selectedTransfer.created_at)}</span>
					</div>
				</div>

				<div>
					{#if selectedTransfer.status === 'pending_approval'}
						<Badge variant="warning" size="md">Menunggu Approval</Badge>
					{:else if selectedTransfer.status === 'approved'}
						<Badge variant="primary" size="md">Disetujui Siap Kirim</Badge>
					{:else if selectedTransfer.status === 'in_transit'}
						<Badge variant="purple" size="md">Dalam Pengiriman</Badge>
					{:else if selectedTransfer.status === 'received'}
						<Badge variant="success" size="md">Diterima Lengkap</Badge>
					{:else if selectedTransfer.status === 'rejected'}
						<Badge variant="danger" size="md">Ditolak</Badge>
					{/if}
				</div>
			</div>

			<!-- Status Stepper Alur Logistik -->
			<div class="rounded-xl border border-neutral-200 bg-white p-4">
				<span class="text-3xs font-semibold tracking-wider text-neutral-400 uppercase block mb-3">Timeline Alur Status</span>
				<div class="flex items-center justify-between text-2xs">
					<!-- Step 1: Diajukan -->
					<div class="flex flex-col items-center text-center">
						<div class="flex h-7 w-7 items-center justify-center rounded-full bg-neutral-900 text-white font-bold text-3xs">
							1
						</div>
						<span class="mt-1 font-semibold text-neutral-900">Diajukan</span>
						<span class="text-3xs text-neutral-400">{formatDate(selectedTransfer.created_at)}</span>
					</div>

					<div class="h-0.5 flex-1 bg-neutral-200 mx-2"></div>

					<!-- Step 2: Disetujui -->
					<div class="flex flex-col items-center text-center">
						<div class="flex h-7 w-7 items-center justify-center rounded-full {selectedTransfer.status === 'rejected' ? 'bg-rose-600 text-white' : ['approved', 'in_transit', 'received'].includes(selectedTransfer.status) ? 'bg-primary-600 text-white' : 'bg-neutral-200 text-neutral-600'} font-bold text-3xs">
							{selectedTransfer.status === 'rejected' ? 'X' : '2'}
						</div>
						<span class="mt-1 font-semibold {selectedTransfer.status === 'rejected' ? 'text-rose-600' : ['approved', 'in_transit', 'received'].includes(selectedTransfer.status) ? 'text-primary-700' : 'text-neutral-500'}">
							{selectedTransfer.status === 'rejected' ? 'Ditolak' : 'Disetujui'}
						</span>
						<span class="text-3xs text-neutral-400">{selectedTransfer.approved_by_name || selectedTransfer.approved_by ? `oleh ${selectedTransfer.approved_by_name || selectedTransfer.approved_by}` : '-'}</span>
					</div>

					<div class="h-0.5 flex-1 bg-neutral-200 mx-2"></div>

					<!-- Step 3: Dikirim -->
					<div class="flex flex-col items-center text-center">
						<div class="flex h-7 w-7 items-center justify-center rounded-full {['in_transit', 'received'].includes(selectedTransfer.status) ? 'bg-indigo-600 text-white' : 'bg-neutral-200 text-neutral-600'} font-bold text-3xs">
							3
						</div>
						<span class="mt-1 font-semibold {['in_transit', 'received'].includes(selectedTransfer.status) ? 'text-indigo-700' : 'text-neutral-500'}">
							Dalam Ekspedisi
						</span>
						<span class="text-3xs text-neutral-400">{['in_transit', 'received'].includes(selectedTransfer.status) ? 'Armada Jalan' : '-'}</span>
					</div>

					<div class="h-0.5 flex-1 bg-neutral-200 mx-2"></div>

					<!-- Step 4: Diterima -->
					<div class="flex flex-col items-center text-center">
						<div class="flex h-7 w-7 items-center justify-center rounded-full {selectedTransfer.status === 'received' ? 'bg-emerald-600 text-white' : 'bg-neutral-200 text-neutral-600'} font-bold text-3xs">
							4
						</div>
						<span class="mt-1 font-semibold {selectedTransfer.status === 'received' ? 'text-emerald-700' : 'text-neutral-500'}">
							Diterima Lengkap
						</span>
						<span class="text-3xs text-neutral-400">{selectedTransfer.received_by_name || selectedTransfer.received_by ? `oleh ${selectedTransfer.received_by_name || selectedTransfer.received_by}` : '-'}</span>
					</div>
				</div>
			</div>

			<!-- Rute Cabang Asal & Tujuan -->
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 text-xs">
				<div class="rounded-xl border border-neutral-200 p-3 bg-neutral-50/50">
					<span class="text-3xs font-semibold tracking-wider text-neutral-400 uppercase">Cabang Asal (Pengirim)</span>
					<p class="mt-1 font-bold text-neutral-900">{locationMap.get(selectedTransfer.from_location_id)?.name ?? '-'}</p>
					<p class="font-mono text-3xs text-neutral-500">{locationMap.get(selectedTransfer.from_location_id)?.code ?? '-'}</p>
					<p class="mt-1 text-3xs text-neutral-500 line-clamp-1">{locationMap.get(selectedTransfer.from_location_id)?.address || 'Alamat fisik tidak tersedia'}</p>
				</div>

				<div class="rounded-xl border border-neutral-200 p-3 bg-neutral-50/50">
					<span class="text-3xs font-semibold tracking-wider text-neutral-400 uppercase">Cabang Tujuan (Penerima)</span>
					<p class="mt-1 font-bold text-neutral-900">{locationMap.get(selectedTransfer.to_location_id)?.name ?? '-'}</p>
					<p class="font-mono text-3xs text-neutral-500">{locationMap.get(selectedTransfer.to_location_id)?.code ?? '-'}</p>
					<p class="mt-1 text-3xs text-neutral-500 line-clamp-1">{locationMap.get(selectedTransfer.to_location_id)?.address || 'Alamat fisik tidak tersedia'}</p>
				</div>
			</div>

			<!-- Alasan Penolakan Jika Rejected -->
			{#if selectedTransfer.status === 'rejected' && selectedTransfer.rejection_reason}
				<div class="rounded-xl border border-rose-200 bg-rose-50/60 p-3 text-xs text-rose-800">
					<strong class="font-semibold block mb-0.5">Alasan Penolakan:</strong>
					"{selectedTransfer.rejection_reason}"
				</div>
			{/if}

			<!-- Rincian Barang yang Dimutasi -->
			<div class="overflow-hidden rounded-xl border border-neutral-200">
				<table class="w-full text-left text-xs">
					<thead>
						<tr class="border-b border-neutral-200 bg-neutral-50 text-2xs font-semibold tracking-wider text-neutral-500 uppercase">
							<th class="px-4 py-2.5">Produk & SKU</th>
							<th class="px-4 py-2.5 text-center">Kuantitas Permohonan</th>
							<th class="px-4 py-2.5 text-center">Kuantitas Tiba</th>
							<th class="px-4 py-2.5 text-center">Status Unit</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100">
						{#each selectedTransfer.items as it}
							<tr>
								<td class="px-4 py-3">
									<span class="font-semibold text-neutral-900 block">{productMap.get(it.product_id)?.name ?? 'Produk'}</span>
									<span class="font-mono text-3xs text-neutral-500">{productMap.get(it.product_id)?.sku ?? it.product_id}</span>
								</td>
								<td class="px-4 py-3 text-center font-mono font-bold text-neutral-900">
									{it.quantity} unit
								</td>
								<td class="px-4 py-3 text-center font-mono font-bold text-emerald-700">
									{it.received_quantity} unit
								</td>
								<td class="px-4 py-3 text-center">
									{#if selectedTransfer.status === 'received'}
										<Badge variant="success" size="sm">100% Full Received</Badge>
									{:else if selectedTransfer.status === 'in_transit'}
										<Badge variant="purple" size="sm">Dalam Armada</Badge>
									{:else if selectedTransfer.status === 'rejected'}
										<Badge variant="danger" size="sm">Batal</Badge>
									{:else}
										<Badge variant="warning" size="sm">Menunggu Ekspedisi</Badge>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-between">
				<!-- Contextual Lifecycle Actions -->
				<div class="flex items-center gap-2">
					{#if selectedTransfer?.status === 'pending_approval'}
						<Button
							variant="primary"
							size="sm"
							loading={actionSubmitting}
							onclick={() => selectedTransfer && handleApprove(selectedTransfer)}
						>
							Setujui Mutasi
						</Button>
						<Button
							variant="danger"
							size="sm"
							loading={actionSubmitting}
							onclick={() => {
								if (selectedTransfer) {
									const id = selectedTransfer.id;
									openRejectModal(id);
								}
							}}
						>
							Tolak
						</Button>
					{:else if selectedTransfer?.status === 'approved'}
						<Button
							variant="primary"
							size="sm"
							loading={actionSubmitting}
							onclick={() => selectedTransfer && handleShip(selectedTransfer)}
						>
							Konfirmasi Pengiriman (Ship)
						</Button>
					{:else if selectedTransfer?.status === 'in_transit'}
						<Button
							variant="primary"
							size="sm"
							loading={actionSubmitting}
							onclick={() => selectedTransfer && handleReceive(selectedTransfer)}
						>
							Konfirmasi Penerimaan Gudang (Receive)
						</Button>
					{/if}
				</div>

				<Button variant="outline" size="sm" onclick={() => (showDetailModal = false)}>
					Tutup
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}

<!-- ======================================================================= -->
<!-- MODAL 3: PENOLAKAN MUTASI (REJECT REASON)                               -->
<!-- ======================================================================= -->
{#if showRejectModal}
	<Modal bind:open={showRejectModal} title="Tolak Permohonan Mutasi Stok" size="md">
		<div class="space-y-4">
			{#if rejectError}
				<Alert variant="error" title="Gagal Menolak">{rejectError}</Alert>
			{/if}

			<div>
				<Input
					label="Alasan Penolakan (Wajib)"
					bind:value={rejectReason}
					placeholder="Contoh: Stok di gudang asal sedang diprioritaskan untuk order ekspor..."
					required
				/>
				<p class="mt-1.5 text-3xs text-neutral-500">
					Alasan penolakan akan dicatat ke dalam dokumen surat jalan dan cadangan stok asal akan seketika dilepaskan.
				</p>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end gap-2">
				<Button variant="outline" onclick={() => (showRejectModal = false)} disabled={rejectSubmitting}>
					Batal
				</Button>
				<Button
					variant="danger"
					loading={rejectSubmitting}
					onclick={handleExecuteReject}
				>
					Konfirmasi Tolak Mutasi
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}
