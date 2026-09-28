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
		listStocks,
		listSerials,
		lookupBarcode,
		lookupSerialUnit
	} from '@erp/api-client';
	import type {
		StockTransferResponse,
		LocationResponse,
		ProductResponse,
		StockResponse,
		SerialUnitResponse
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
		flagSerialTracking: boolean;
		availableSerials: SerialUnitResponse[];
		selectedSerialIds: string[];
		loadingSerials: boolean;
		barcodeInput: string;
		select2SerialValue: string;
	}
	let transferItemsDraft = $state<TransferItemDraft[]>([]);
	let itemSelectedProductId = $state('');
	let itemQuantity = $state<number>(1);
	let originStocks = $state<StockResponse[]>([]);
	let loadingOriginStocks = $state(false);
	let fastScanCode = $state('');
	let fastScanLoading = $state(false);

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

	const filterFromLocationOptions = $derived<Select2Option[]>([
		{ value: '', label: 'Asal: Semua Cabang' },
		...locations.map((loc) => ({
			value: loc.id,
			label: `Asal: ${loc.name} (${loc.code})`
		}))
	]);

	const filterToLocationOptions = $derived<Select2Option[]>([
		{ value: '', label: 'Tujuan: Semua Cabang' },
		...locations.map((loc) => ({
			value: loc.id,
			label: `Tujuan: ${loc.name} (${loc.code})`
		}))
	]);

	// Dropdown pilihan produk saat menyusun mutasi barang
	const productSelectOptions = $derived.by<Select2Option[]>(() => {
		const stockLookup: Record<string, number> = {};
		for (const st of originStocks) {
			stockLookup[st.product_id] = Math.max(0, st.quantity - st.reserved_quantity);
		}

		return products.map((prod) => {
			const avail = stockLookup[prod.id] ?? 0;
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
		fastScanCode = '';
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

	// ── 10. Multi-Item Builder Actions & Serial Tracking ──────────────────────
	async function loadSerialsForDraftItem(draft: TransferItemDraft) {
		const token = getToken();
		if (!token || !formFromLocationId) return;
		draft.loadingSerials = true;
		try {
			const res = await listSerials(token, {
				product_id: draft.productId,
				location_id: formFromLocationId,
				status: 'tersedia'
			});
			draft.availableSerials = res ?? [];
		} catch (e) {
			console.error('Gagal mengambil daftar nomor seri unit mutasi:', e);
			draft.availableSerials = [];
		} finally {
			draft.loadingSerials = false;
		}
	}

	function getSerialSelect2Options(draft: TransferItemDraft): Select2Option[] {
		const selectedSet = new Set(draft.selectedSerialIds);
		return draft.availableSerials
			.filter((s) => !selectedSet.has(s.id))
			.map((s) => ({
				value: s.id,
				label: s.serial_number,
				subtext: 'Tersedia di cabang asal'
			}));
	}

	function handleAddSerialByBarcodeToTransfer(draft: TransferItemDraft) {
		const sn = draft.barcodeInput.trim();
		if (!sn) return;

		const match = draft.availableSerials.find(
			(u) => u.serial_number.toLowerCase() === sn.toLowerCase()
		);

		if (!match) {
			toast.error(
				`Nomor seri "${sn}" tidak ditemukan atau tidak berstatus tersedia di gudang cabang asal.`
			);
			return;
		}

		if (draft.selectedSerialIds.includes(match.id)) {
			toast.warning(`Nomor seri ${match.serial_number} sudah ada dalam pilihan`);
			draft.barcodeInput = '';
			return;
		}

		if (draft.selectedSerialIds.length >= draft.quantity) {
			toast.warning(
				`Kuantitas mutasi (${draft.quantity}) sudah terpenuhi. Tambah kuantitas jika ingin memindahkan lebih banyak unit.`
			);
			return;
		}

		draft.selectedSerialIds.push(match.id);
		draft.barcodeInput = '';
		toast.success(`Nomor seri ${match.serial_number} berhasil dipilih`);
	}

	function handleSelectSerialFromDropdownToTransfer(draft: TransferItemDraft, serialId: string) {
		if (!serialId) return;

		if (draft.selectedSerialIds.includes(serialId)) {
			draft.select2SerialValue = '';
			return;
		}

		if (draft.selectedSerialIds.length >= draft.quantity) {
			toast.warning(
				`Kuantitas mutasi (${draft.quantity}) sudah terpenuhi. Tambah kuantitas jika ingin memindahkan lebih banyak unit.`
			);
			draft.select2SerialValue = '';
			return;
		}

		draft.selectedSerialIds.push(serialId);
		draft.select2SerialValue = '';
		const matchedUnit = draft.availableSerials.find((s) => s.id === serialId);
		if (matchedUnit) {
			toast.success(`Nomor seri ${matchedUnit.serial_number} berhasil dipilih`);
		}
	}

	function handleAutoPickSerialsToTransfer(draft: TransferItemDraft) {
		const needed = draft.quantity - draft.selectedSerialIds.length;
		if (needed <= 0) {
			toast.info('Seluruh kuantitas nomor seri telah terpenuhi');
			return;
		}

		const selectedSet = new Set(draft.selectedSerialIds);
		const candidates = draft.availableSerials.filter((s) => !selectedSet.has(s.id));

		if (candidates.length === 0) {
			toast.warning('Tidak ada unit nomor seri fisik yang tersedia di cabang asal');
			return;
		}

		const toAdd = candidates.slice(0, needed);
		for (const u of toAdd) {
			draft.selectedSerialIds.push(u.id);
		}

		toast.success(`${toAdd.length} nomor seri berhasil dipilih otomatis secara FIFO`);
	}

	function handleRemoveSerialFromTransfer(draft: TransferItemDraft, serialId: string) {
		draft.selectedSerialIds = draft.selectedSerialIds.filter((id) => id !== serialId);
	}

	function handleClearSerialsFromTransfer(draft: TransferItemDraft) {
		draft.selectedSerialIds = [];
		toast.info('Pilihan nomor seri telah direset');
	}

	async function addItemToDraft(explicitProdId?: string, explicitQty?: number) {
		createError = null;
		const targetProdId = explicitProdId ?? itemSelectedProductId;
		const targetQty = explicitQty ?? itemQuantity;

		if (!targetProdId) {
			createError = 'Pilih produk yang akan dimutasi terlebih dahulu.';
			return;
		}
		if (targetQty <= 0) {
			createError = 'Kuantitas transfer minimal 1 unit.';
			return;
		}

		const prod = productMap.get(targetProdId);
		if (!prod) return;

		// Cek apakah produk sudah ada di daftar draft
		const existing = transferItemsDraft.find((i) => i.productId === prod.id);
		if (existing) {
			createError = `Produk "${prod.name}" sudah ada dalam daftar. Hapus terlebih dahulu jika ingin mengganti kuantitas.`;
			return;
		}

		// Cek ketersediaan stok fisik di cabang asal
		const st = originStocks.find((s) => s.product_id === prod.id);
		const avail = st ? Math.max(0, st.quantity - st.reserved_quantity) : 0;
		if (targetQty > avail) {
			createError = `Kuantitas transfer (${targetQty} unit) melebihi stok siap jual di cabang asal (${avail} unit).`;
			return;
		}

		const isSerial = prod.flag_serial_tracking ?? false;
		const draftItem: TransferItemDraft = {
			productId: prod.id,
			productName: prod.name,
			productSku: prod.sku,
			quantity: targetQty,
			availableStock: avail,
			flagSerialTracking: isSerial,
			availableSerials: [],
			selectedSerialIds: [],
			loadingSerials: false,
			barcodeInput: '',
			select2SerialValue: ''
		};

		transferItemsDraft.push(draftItem);

		if (isSerial && formFromLocationId) {
			await loadSerialsForDraftItem(draftItem);
		}

		// Reset form input item
		itemSelectedProductId = '';
		itemQuantity = 1;
	}

	function removeItemFromDraft(index: number) {
		transferItemsDraft.splice(index, 1);
	}

	// ── Scanner Cepat Barcode / SKU / IMEI ────────────────────────────────────
	async function handleFastScan() {
		const code = fastScanCode.trim();
		if (!code) return;
		if (!formFromLocationId) {
			toast.warning('Pilih cabang asal terlebih dahulu sebelum scan barang.');
			return;
		}

		fastScanLoading = true;
		try {
			const token = getToken();
			if (!token) return;

			// 1. Cek apakah cocok dengan SKU produk
			let matchedProd = products.find((p) => p.sku.toLowerCase() === code.toLowerCase());

			// 2. Jika belum, coba lookup barcode produk
			if (!matchedProd) {
				try {
					const bcRes = await lookupBarcode(token, code);
					if (bcRes && bcRes.product) {
						matchedProd = products.find((p) => p.id === bcRes.product.id) || bcRes.product;
					}
				} catch {
					// Lanjutkan ke serial lookup
				}
			}

			// 3. Jika belum, coba lookup serial unit / IMEI
			if (!matchedProd) {
				try {
					const snRes = await lookupSerialUnit(token, code);
					if (snRes && snRes.serial_unit?.product_id) {
						matchedProd = products.find((p) => p.id === snRes.serial_unit.product_id);
						if (matchedProd) {
							await addProductWithSpecificSerial(
								matchedProd,
								snRes.serial_unit.id,
								snRes.serial_unit.serial_number
							);
							fastScanCode = '';
							return;
						}
					}
				} catch {
					// Tidak ditemukan
				}
			}

			if (matchedProd) {
				const existing = transferItemsDraft.find((i) => i.productId === matchedProd.id);
				if (existing) {
					if (existing.quantity < existing.availableStock) {
						existing.quantity += 1;
						toast.success(`Kuantitas "${matchedProd.name}" bertambah menjadi ${existing.quantity}`);
					} else {
						toast.warning(
							`Stok tersedia "${matchedProd.name}" di cabang asal hanya ${existing.availableStock} unit.`
						);
					}
				} else {
					await addItemToDraft(matchedProd.id, 1);
					toast.success(`Produk "${matchedProd.name}" berhasil ditambahkan ke mutasi!`);
				}
				fastScanCode = '';
			} else {
				toast.error(`Barcode / SKU / IMEI "${code}" tidak ditemukan dalam sistem.`);
			}
		} finally {
			fastScanLoading = false;
		}
	}

	async function addProductWithSpecificSerial(
		prod: ProductResponse,
		serialId: string,
		serialNumber: string
	) {
		let draft = transferItemsDraft.find((i) => i.productId === prod.id);
		if (!draft) {
			const st = originStocks.find((s) => s.product_id === prod.id);
			const avail = st ? Math.max(0, st.quantity - st.reserved_quantity) : 0;
			if (avail < 1) {
				toast.error(`Stok produk "${prod.name}" di cabang asal tidak tersedia.`);
				return;
			}
			draft = {
				productId: prod.id,
				productName: prod.name,
				productSku: prod.sku,
				quantity: 1,
				availableStock: avail,
				flagSerialTracking: true,
				availableSerials: [],
				selectedSerialIds: [],
				loadingSerials: false,
				barcodeInput: '',
				select2SerialValue: ''
			};
			transferItemsDraft.push(draft);
			await loadSerialsForDraftItem(draft);
		}

		if (draft.selectedSerialIds.includes(serialId)) {
			toast.warning(`Nomor seri ${serialNumber} sudah dipilih sebelumnya.`);
			return;
		}

		if (draft.selectedSerialIds.length >= draft.quantity) {
			if (draft.quantity < draft.availableStock) {
				draft.quantity += 1;
			} else {
				toast.warning(
					`Kuantitas produk "${prod.name}" sudah mencapai batas stok asal (${draft.availableStock} unit).`
				);
				return;
			}
		}

		draft.selectedSerialIds.push(serialId);
		toast.success(`Nomor seri ${serialNumber} (${prod.name}) berhasil dipilih!`);
	}

	// ── 11. Eksekusi Buat Mutasi Stok (Create) ─────────────────────────────────
	function openCreateModal() {
		formFromLocationId = locations.length > 0 ? locations[0].id : '';
		formToLocationId = locations.length > 1 ? locations[1].id : '';
		formNotes = '';
		transferItemsDraft = [];
		itemSelectedProductId = '';
		itemQuantity = 1;
		fastScanCode = '';
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

		// Validasi nomor seri untuk produk berpelacakan serial aktif
		for (const item of transferItemsDraft) {
			if (item.flagSerialTracking) {
				if (item.selectedSerialIds.length !== item.quantity) {
					createError = `Produk '${item.productName}' memiliki pelacakan serial aktif: wajib menyertakan ${item.quantity} nomor seri/IMEI (saat ini baru ${item.selectedSerialIds.length} dipilih).`;
					return;
				}
			}
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
					quantity: i.quantity,
					serial_unit_ids: i.flagSerialTracking ? i.selectedSerialIds : undefined
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
			toast.success(
				`Barang "${trf.transfer_number}" telah dikonfirmasi berangkat (Dalam Pengiriman)!`
			);
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
				Kelola alur distribusi permohonan mutasi barang, persetujuan pimpinan, ekspedisi pengiriman,
				dan konfirmasi penerimaan gudang.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" {loading} onclick={loadData}>
				<!-- Refresh Icon -->
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99"
					/>
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
		<button
			type="button"
			onclick={() => {
				statusFilter = 'all';
				currentPage = 1;
			}}
			class="group relative overflow-hidden rounded-xl border p-3.5 text-left transition-all {statusFilter ===
			'all'
				? 'border-neutral-900 bg-neutral-50/80 shadow-2xs ring-1 ring-neutral-900'
				: 'border-neutral-200 bg-white shadow-2xs hover:border-neutral-300 hover:bg-neutral-50/40'}"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-neutral-400 uppercase"
					>Total Mutasi</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg transition-colors {statusFilter ===
					'all'
						? 'bg-neutral-900 text-white shadow-2xs'
						: 'bg-neutral-100 text-neutral-500 group-hover:bg-neutral-200'}"
				>
					<!-- Document Text Icon -->
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
							d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-neutral-800 sm:text-3xl"
					>{stats.total}</span
				>
				<span class="text-[11px] font-normal text-neutral-400">surat</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-neutral-400/80">Surat jalan diterbitkan</p>
		</button>

		<!-- Menunggu Approval -->
		<button
			type="button"
			onclick={() => {
				statusFilter = statusFilter === 'pending_approval' ? 'all' : 'pending_approval';
				currentPage = 1;
			}}
			class="group relative overflow-hidden rounded-xl border p-3.5 text-left transition-all {statusFilter ===
			'pending_approval'
				? 'border-amber-500 bg-amber-50/80 shadow-2xs ring-1 ring-amber-400'
				: 'border-amber-200/90 bg-white shadow-2xs hover:border-amber-400 hover:bg-amber-50/40'}"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-amber-700/80 uppercase"
					>Menunggu Approval</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-100 text-amber-700"
				>
					<!-- Clock Icon -->
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
							d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-amber-900 sm:text-3xl"
					>{stats.pending}</span
				>
				<span class="text-[11px] font-normal text-amber-700/60">surat</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-amber-800/60">Stok asal dibooking</p>
		</button>

		<!-- Disetujui -->
		<button
			type="button"
			onclick={() => {
				statusFilter = statusFilter === 'approved' ? 'all' : 'approved';
				currentPage = 1;
			}}
			class="group relative overflow-hidden rounded-xl border p-3.5 text-left transition-all {statusFilter ===
			'approved'
				? 'border-neutral-900 bg-neutral-50/80 shadow-2xs ring-1 ring-neutral-900'
				: 'border-neutral-200 bg-white shadow-2xs hover:border-neutral-400 hover:bg-neutral-50/40'}"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-neutral-500 uppercase"
					>Disetujui</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-neutral-900 text-white shadow-2xs"
				>
					<!-- Check Badge Icon -->
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
							d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-neutral-900 sm:text-3xl"
					>{stats.approved}</span
				>
				<span class="text-[11px] font-normal text-neutral-400">surat</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-neutral-400/80">Siap dipacking & kirim</p>
		</button>

		<!-- Dalam Pengiriman -->
		<button
			type="button"
			onclick={() => {
				statusFilter = statusFilter === 'in_transit' ? 'all' : 'in_transit';
				currentPage = 1;
			}}
			class="group relative overflow-hidden rounded-xl border p-3.5 text-left transition-all {statusFilter ===
			'in_transit'
				? 'border-indigo-500 bg-indigo-50/80 shadow-2xs ring-1 ring-indigo-400'
				: 'border-indigo-200/90 bg-white shadow-2xs hover:border-indigo-400 hover:bg-indigo-50/40'}"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-indigo-700/80 uppercase"
					>Dalam Perjalanan</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-indigo-100 text-indigo-700"
				>
					<!-- Truck Icon -->
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
							d="M8.25 18.75a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m3 0h6m-9 0H3.375a1.125 1.125 0 01-1.125-1.125V14.25m17.25 4.5a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m3 0h1.125c.621 0 1.129-.504 1.09-1.124a17.902 17.902 0 00-3.213-9.193 2.056 2.056 0 00-1.58-.86H14.25M16.5 18.75h-2.25m0-11.175V3.75A1.125 1.125 0 0013.125 2.625H3.375A1.125 1.125 0 002.25 3.75v10.5m14.25-6.675l-4.5 4.5"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-indigo-900 sm:text-3xl"
					>{stats.inTransit}</span
				>
				<span class="text-[11px] font-normal text-indigo-600/60">surat</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-indigo-700/60">Armada ekspedisi jalan</p>
		</button>

		<!-- Diterima Lengkap -->
		<button
			type="button"
			onclick={() => {
				statusFilter = statusFilter === 'received' ? 'all' : 'received';
				currentPage = 1;
			}}
			class="group relative overflow-hidden rounded-xl border p-3.5 text-left transition-all {statusFilter ===
			'received'
				? 'border-emerald-500 bg-emerald-50/80 shadow-2xs ring-1 ring-emerald-400'
				: 'border-emerald-200/90 bg-white shadow-2xs hover:border-emerald-400 hover:bg-emerald-50/40'}"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-emerald-600/80 uppercase"
					>Diterima Lengkap</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700"
				>
					<!-- Inbox Arrow Down Icon -->
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
							d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m6 4.125l2.25 2.25m0 0l2.25-2.25m-2.25 2.25V3.75"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-2xl font-semibold tracking-tight text-emerald-700 sm:text-3xl"
					>{stats.received}</span
				>
				<span class="text-[11px] font-normal text-emerald-600/60">surat</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-emerald-600/70">Stok tujuan bertambah</p>
		</button>

		<!-- Ditolak -->
		<button
			type="button"
			onclick={() => {
				statusFilter = statusFilter === 'rejected' ? 'all' : 'rejected';
				currentPage = 1;
			}}
			class="group relative overflow-hidden rounded-xl border p-3.5 text-left transition-all {statusFilter ===
			'rejected'
				? 'border-rose-500 bg-rose-50/80 shadow-sm ring-1 ring-rose-400'
				: 'border-rose-200/90 bg-white shadow-2xs hover:border-rose-400 hover:bg-rose-50/40'}"
		>
			<div class="flex items-center justify-between">
				<span class="text-[11px] font-medium tracking-wider text-rose-700/80 uppercase"
					>Ditolak</span
				>
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-rose-100 text-rose-700"
				>
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
				<span class="text-2xl font-semibold tracking-tight text-rose-800 sm:text-3xl"
					>{stats.rejected}</span
				>
				<span class="text-[11px] font-normal text-rose-700/60">surat</span>
			</div>
			<p class="mt-1 text-[10px] font-normal text-rose-800/60">Cadangan dilepaskan</p>
		</button>
	</div>

	<!-- Toolbar Filter & Pencarian -->
	<div class="flex flex-col gap-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
		<!-- Quick Filter Tab Pills -->
		<div class="flex flex-wrap items-center gap-1.5 border-b border-neutral-100 pb-3">
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
					{stats.total}
				</span>
			</button>

			<button
				type="button"
				onclick={() => {
					statusFilter = 'pending_approval';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'pending_approval'
					? 'border-amber-600 bg-amber-600 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-amber-800 shadow-2xs hover:border-neutral-300 hover:bg-amber-50/50'}"
			>
				Menunggu Approval
				<span
					class="py-0.2 text-3xs rounded-full px-1.5 font-semibold {statusFilter ===
					'pending_approval'
						? 'bg-amber-700 text-white'
						: 'border border-amber-200/60 bg-amber-50 text-amber-800'}"
				>
					{stats.pending}
				</span>
			</button>

			<button
				type="button"
				onclick={() => {
					statusFilter = 'approved';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'approved'
					? 'border-primary-700 bg-primary-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-primary-800 shadow-2xs hover:border-neutral-300 hover:bg-primary-50/50'}"
			>
				Disetujui
				<span
					class="py-0.2 text-3xs rounded-full px-1.5 font-semibold {statusFilter === 'approved'
						? 'bg-primary-800 text-white'
						: 'border border-primary-200/60 bg-primary-50 text-primary-800'}"
				>
					{stats.approved}
				</span>
			</button>

			<button
				type="button"
				onclick={() => {
					statusFilter = 'in_transit';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'in_transit'
					? 'border-indigo-700 bg-indigo-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-indigo-800 shadow-2xs hover:border-neutral-300 hover:bg-indigo-50/50'}"
			>
				Dalam Pengiriman
				<span
					class="py-0.2 text-3xs rounded-full px-1.5 font-semibold {statusFilter === 'in_transit'
						? 'bg-indigo-800 text-white'
						: 'border border-indigo-200/60 bg-indigo-50 text-indigo-800'}"
				>
					{stats.inTransit}
				</span>
			</button>

			<button
				type="button"
				onclick={() => {
					statusFilter = 'received';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'received'
					? 'border-emerald-700 bg-emerald-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-emerald-800 shadow-2xs hover:border-neutral-300 hover:bg-emerald-50/50'}"
			>
				Diterima
				<span
					class="py-0.2 text-3xs rounded-full px-1.5 font-semibold {statusFilter === 'received'
						? 'bg-emerald-800 text-white'
						: 'border border-emerald-200/60 bg-emerald-50 text-emerald-800'}"
				>
					{stats.received}
				</span>
			</button>

			<button
				type="button"
				onclick={() => {
					statusFilter = 'rejected';
					currentPage = 1;
				}}
				class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {statusFilter ===
				'rejected'
					? 'border-rose-700 bg-rose-700 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-rose-800 shadow-2xs hover:border-neutral-300 hover:bg-rose-50/50'}"
			>
				Ditolak
				<span
					class="py-0.2 text-3xs rounded-full px-1.5 font-semibold {statusFilter === 'rejected'
						? 'bg-rose-800 text-white'
						: 'border border-rose-200/60 bg-rose-50 text-rose-800'}"
				>
					{stats.rejected}
				</span>
			</button>
		</div>

		<!-- Filter Dropdown & Search (Compact) -->
		<div class="flex flex-wrap items-center gap-2">
			<div class="w-full sm:w-52">
				<Select2
					size="sm"
					options={filterFromLocationOptions}
					value={fromLocationFilter}
					placeholder="Asal: Semua Cabang"
					searchPlaceholder="Cari cabang asal..."
					clearable={true}
					onchange={(val) => {
						fromLocationFilter = String(val);
						currentPage = 1;
					}}
				/>
			</div>

			<div class="w-full sm:w-52">
				<Select2
					size="sm"
					options={filterToLocationOptions}
					value={toLocationFilter}
					placeholder="Tujuan: Semua Cabang"
					searchPlaceholder="Cari cabang tujuan..."
					clearable={true}
					onchange={(val) => {
						toLocationFilter = String(val);
						currentPage = 1;
					}}
				/>
			</div>

			<div class="w-full sm:w-64">
				<SearchInput
					size="sm"
					bind:value={searchQuery}
					placeholder="Cari No. Transfer atau Catatan..."
					onsearch={() => {
						currentPage = 1;
					}}
				/>
			</div>

			{#if fromLocationFilter || toLocationFilter || searchQuery}
				<button
					type="button"
					onclick={() => {
						fromLocationFilter = '';
						toLocationFilter = '';
						searchQuery = '';
						currentPage = 1;
					}}
					class="px-2 py-1 text-xs text-neutral-500 hover:text-neutral-900 hover:underline"
				>
					Reset Filter
				</button>
			{/if}
		</div>
	</div>

	<!-- Tabel Data Mutasi Stok -->
	<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
		<Table
			borderless
			empty={paginatedTransfers.length === 0}
			emptyMessage={loading
				? 'Memuat dokumen mutasi stok...'
				: 'Tidak ada dokumen mutasi yang cocok dengan filter.'}
		>
			<thead>
				<tr
					class="border-b border-neutral-200 bg-neutral-50/75 text-left text-xs font-semibold text-neutral-600"
				>
					<th class="px-5 py-3.5">No. Transfer</th>
					<th class="px-5 py-3.5">Rute Ekspedisi</th>
					<th class="px-5 py-3.5 text-center">Total Item & Unit</th>
					<th class="px-5 py-3.5">Pemohon & Tanggal</th>
					<th class="px-5 py-3.5 text-center">Status Alur</th>
					<th class="px-5 py-3.5 text-right">Aksi Tindakan</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-200 bg-white text-xs">
				{#each paginatedTransfers as trf (trf.id)}
					<tr class="transition-colors hover:bg-neutral-50/80">
						<!-- No. Transfer -->
						<td class="px-5 py-4">
							<button
								type="button"
								onclick={() => openDetailModal(trf)}
								class="block text-left font-mono text-xs font-bold text-neutral-900 hover:text-primary-700 hover:underline"
							>
								{trf.transfer_number}
							</button>
							{#if trf.notes}
								<p class="text-3xs mt-0.5 line-clamp-1 text-neutral-500 italic">"{trf.notes}"</p>
							{/if}
						</td>

						<!-- Rute Ekspedisi: Asal -> Tujuan -->
						<td class="px-5 py-4">
							<div class="flex items-center gap-2">
								<div class="max-w-[130px] truncate">
									<span class="text-xs font-medium text-neutral-800"
										>{locationMap.get(trf.from_location_id)?.name ?? 'Cabang Asal'}</span
									>
									<span class="text-3xs block font-mono text-neutral-400"
										>{locationMap.get(trf.from_location_id)?.code ?? '-'}</span
									>
								</div>

								<!-- Arrow Right Icon -->
								<div
									class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-neutral-100 text-neutral-500"
								>
									<svg
										class="h-3 w-3"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2.5"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3"
										/>
									</svg>
								</div>

								<div class="max-w-[130px] truncate">
									<span class="text-xs font-medium text-neutral-800"
										>{locationMap.get(trf.to_location_id)?.name ?? 'Cabang Tujuan'}</span
									>
									<span class="text-3xs block font-mono text-neutral-400"
										>{locationMap.get(trf.to_location_id)?.code ?? '-'}</span
									>
								</div>
							</div>
						</td>

						<!-- Total Item & Unit -->
						<td class="px-5 py-4 text-center">
							<div class="inline-flex items-center gap-1.5">
								<span class="font-bold text-neutral-900">{calculateTotalUnits(trf.items)}</span>
								<span class="text-3xs text-neutral-500">unit</span>
								<span class="text-neutral-300">•</span>
								<span class="text-3xs text-neutral-500">{trf.items.length} SKU</span>
							</div>
						</td>

						<!-- Pemohon & Tanggal -->
						<td class="px-5 py-4 text-neutral-600">
							<div class="text-xs font-medium text-neutral-800">
								{trf.requested_by_name || trf.requested_by}
							</div>
							<div class="text-3xs text-neutral-400">{formatDate(trf.created_at)}</div>
						</td>

						<!-- Status Badge -->
						<td class="px-5 py-4 text-center">
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
						<td class="px-5 py-4 text-right whitespace-nowrap">
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
				Menampilkan {(currentPage - 1) * pageSize + 1} - {Math.min(
					currentPage * pageSize,
					totalItems
				)} dari {totalItems} mutasi
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
					<label
						for="create-from-location"
						class="text-2xs mb-1 block font-semibold tracking-wide text-neutral-700 uppercase"
					>
						Cabang Asal (Pengirim) <span class="text-rose-500">*</span>
					</label>
					<select
						id="create-from-location"
						bind:value={formFromLocationId}
						onchange={() => handleOriginLocationChange(formFromLocationId)}
						class="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden"
					>
						{#each locations as loc (loc.id)}
							<option value={loc.id}>{loc.name} ({loc.code})</option>
						{/each}
					</select>
					<p class="text-3xs mt-1 text-neutral-500">
						Stok barang di cabang ini akan dicadangkan saat permohonan diajukan.
					</p>
				</div>

				<div>
					<label
						for="create-to-location"
						class="text-2xs mb-1 block font-semibold tracking-wide text-neutral-700 uppercase"
					>
						Cabang Tujuan (Penerima) <span class="text-rose-500">*</span>
					</label>
					<select
						id="create-to-location"
						bind:value={formToLocationId}
						class="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden"
					>
						{#each locations as loc (loc.id)}
							{#if loc.id !== formFromLocationId}
								<option value={loc.id}>{loc.name} ({loc.code})</option>
							{/if}
						{/each}
					</select>
					<p class="text-3xs mt-1 text-neutral-500">
						Stok fisik cabang tujuan akan bertambah saat barang dikonfirmasi tiba.
					</p>
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
			<div class="space-y-3 rounded-xl border border-neutral-200/80 bg-neutral-50/60 p-3.5">
				<div class="flex items-center justify-between">
					<h4 class="text-xs font-bold tracking-wide text-neutral-900 uppercase">
						Pilih Barang yang Dimutasi
					</h4>
					<span class="text-3xs text-neutral-500">
						{transferItemsDraft.length} produk ditambahkan ({transferItemsDraft.reduce(
							(a, b) => a + b.quantity,
							0
						)} unit)
					</span>
				</div>

				<!-- Fast Barcode / SKU / IMEI Scanner Bar (Dual-Mode Input) -->
				<div class="rounded-xl border border-neutral-300 bg-white p-3 shadow-2xs">
					<label
						for="fast-scan-transfer"
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
							id="fast-scan-transfer"
							type="text"
							bind:value={fastScanCode}
							placeholder="Tembakkan scanner barcode produk, SKU, atau nomor seri/IMEI di sini..."
							onkeydown={(e) => {
								if (e.key === 'Enter') {
									e.preventDefault();
									handleFastScan();
								}
							}}
							disabled={fastScanLoading || !formFromLocationId}
							class="w-full rounded-lg border border-neutral-300 bg-white py-2 pr-20 pl-3 font-mono text-xs text-neutral-900 placeholder:font-sans placeholder:text-neutral-400 focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden disabled:cursor-not-allowed disabled:bg-neutral-100"
						/>
						<button
							type="button"
							onclick={handleFastScan}
							disabled={fastScanLoading || !fastScanCode.trim() || !formFromLocationId}
							class="text-2xs absolute right-1 rounded-md bg-neutral-900 px-3 py-1 font-semibold text-white transition-colors hover:bg-neutral-800 disabled:cursor-not-allowed disabled:opacity-40"
						>
							{fastScanLoading ? 'Memindai...' : 'Pindai'}
						</button>
					</div>
				</div>

				<!-- Item Adder Row (Manual Select2 & Kuantitas) -->
				<div class="grid grid-cols-1 gap-2.5 sm:grid-cols-12 sm:items-end">
					<div class="sm:col-span-8">
						<Select2
							label="Atau Cari Manual Produk (Select2)"
							options={productSelectOptions}
							value={itemSelectedProductId}
							placeholder={loadingOriginStocks
								? 'Memuat ketersediaan stok...'
								: 'Cari & pilih produk...'}
							searchPlaceholder="Ketik nama atau SKU produk..."
							onchange={(val) => {
								itemSelectedProductId = String(val);
							}}
						/>
					</div>

					<div class="sm:col-span-2">
						<Input
							label="Jumlah (Unit)"
							type="number"
							bind:value={itemQuantity}
							min={1}
							placeholder="1"
							required
						/>
					</div>

					<div class="sm:col-span-2">
						<Button
							variant="outline"
							fullWidth
							onclick={() => addItemToDraft()}
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
								<tr
									class="text-2xs border-b border-neutral-200 bg-neutral-50 font-semibold text-neutral-500 uppercase"
								>
									<th class="px-3 py-2">Nama Produk & SKU</th>
									<th class="px-3 py-2 text-center">Stok Asal</th>
									<th class="px-3 py-2 text-center">Kuantitas Transfer</th>
									<th class="px-3 py-2 text-right">Aksi</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100">
								{#each transferItemsDraft as draft, idx (draft.productId)}
									<tr class="hover:bg-neutral-50/50">
										<td class="px-3 py-2">
											<div class="flex items-center gap-1.5">
												<span class="block font-medium text-neutral-900">{draft.productName}</span>
												{#if draft.flagSerialTracking}
													<Badge variant="purple" size="sm">Wajib Serial / IMEI</Badge>
												{/if}
											</div>
											<span class="text-3xs font-mono text-neutral-500">{draft.productSku}</span>
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
												class="text-2xs font-semibold text-rose-600 hover:text-rose-800"
											>
												Hapus
											</button>
										</td>
									</tr>

									<!-- Bagian Pilihan Nomor Seri / IMEI jika produk dilacak nomor serinya -->
									{#if draft.flagSerialTracking}
										{@const serialCount = draft.selectedSerialIds.length}
										{@const isComplete = serialCount === Number(draft.quantity)}
										{@const unselectedOptions = getSerialSelect2Options(draft)}
										<tr>
											<td colspan="4" class="bg-neutral-50/60 px-3 pt-0 pb-3">
												<div class="rounded-xl border border-amber-200 bg-amber-50/70 p-3">
													<!-- Header Info & Counter Serial -->
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
															<span class="text-2xs font-bold text-amber-900">
																Nomor Seri / IMEI Fisik ({draft.productName}):
															</span>
															{#if draft.loadingSerials}
																<span class="text-3xs animate-pulse text-amber-700">
																	(Memuat stok serial...)
																</span>
															{:else}
																<span class="text-3xs text-neutral-600">
																	(Tersedia di cabang asal:
																	<strong class="text-neutral-900"
																		>{draft.availableSerials.length}</strong
																	>
																	unit)
																</span>
															{/if}
														</div>

														<div class="flex items-center gap-2">
															<span
																class="text-3xs rounded-full px-2 py-0.5 font-semibold {isComplete
																	? 'border border-emerald-300 bg-emerald-100 text-emerald-800'
																	: 'border border-amber-300 bg-amber-100 text-amber-800'}"
															>
																{serialCount} dari {draft.quantity} nomor seri dipilih
															</span>
															{#if serialCount > 0}
																<button
																	type="button"
																	onclick={() => handleClearSerialsFromTransfer(draft)}
																	class="text-3xs text-neutral-500 underline hover:text-rose-600"
																>
																	Reset
																</button>
															{/if}
														</div>
													</div>

													{#if !draft.loadingSerials && draft.availableSerials.length === 0}
														<div
															class="text-3xs rounded-md border border-rose-200 bg-rose-50 p-2 text-rose-800"
														>
															Peringatan: Tidak ada unit fisik nomor seri berstatus 'tersedia' untuk
															produk ini di cabang asal.
														</div>
													{:else}
														<!-- Dual-Mode: Scan Barcode IMEI vs Select2 Dropdown -->
														<div class="grid grid-cols-1 items-end gap-2.5 sm:grid-cols-2">
															<!-- 1. Scan Barcode / Ketik IMEI Cepat -->
															<div>
																<label
																	for={`serial-scan-${idx}`}
																	class="text-3xs mb-1 block font-semibold text-neutral-700"
																>
																	Scan Barcode / Ketik IMEI Fisik:
																</label>
																<div class="relative flex items-center">
																	<input
																		id={`serial-scan-${idx}`}
																		type="text"
																		bind:value={draft.barcodeInput}
																		placeholder="Tembakkan scanner IMEI..."
																		onkeydown={(e) => {
																			if (e.key === 'Enter') {
																				e.preventDefault();
																				handleAddSerialByBarcodeToTransfer(draft);
																			}
																		}}
																		disabled={isComplete || draft.loadingSerials}
																		class="w-full rounded-lg border border-neutral-300 bg-white py-1.5 pr-14 pl-2 font-mono text-xs text-neutral-900 placeholder:font-sans placeholder:text-neutral-400 focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden disabled:cursor-not-allowed disabled:bg-neutral-100"
																	/>
																	<button
																		type="button"
																		onclick={() => handleAddSerialByBarcodeToTransfer(draft)}
																		disabled={isComplete ||
																			!draft.barcodeInput.trim() ||
																			draft.loadingSerials}
																		class="text-3xs absolute right-1 rounded-md bg-neutral-900 px-2 py-1 font-semibold text-white transition-colors hover:bg-neutral-800 disabled:cursor-not-allowed disabled:opacity-40"
																	>
																		Pilih
																	</button>
																</div>
															</div>

															<!-- 2. Select2 Dropdown Seri Tersedia -->
															<div>
																<Select2
																	id={`item-serial-select-${idx}`}
																	label="Atau Pilih dari Serial Tersedia:"
																	size="sm"
																	options={unselectedOptions}
																	bind:value={draft.select2SerialValue}
																	placeholder={isComplete
																		? 'Kuantitas terpenuhi'
																		: unselectedOptions.length === 0
																			? 'Habis'
																			: 'Pilih nomor seri...'}
																	searchPlaceholder="Cari nomor seri..."
																	disabled={isComplete ||
																		unselectedOptions.length === 0 ||
																		draft.loadingSerials}
																	onchange={(val) => {
																		if (val) {
																			handleSelectSerialFromDropdownToTransfer(draft, String(val));
																		}
																	}}
																/>
															</div>
														</div>

														<!-- Auto Pick & Quick Actions -->
														{#if !isComplete && unselectedOptions.length > 0}
															<div class="mt-2 flex items-center justify-between">
																<span class="text-3xs text-neutral-500">
																	Tips: Anda bisa menembak scanner berkali-kali untuk memilih nomor
																	seri berurutan.
																</span>
																<button
																	type="button"
																	onclick={() => handleAutoPickSerialsToTransfer(draft)}
																	class="text-3xs font-medium text-amber-900 underline hover:text-amber-950"
																>
																	Pilih Otomatis {Math.min(
																		draft.quantity - serialCount,
																		unselectedOptions.length
																	)} Unit Tersisa (FIFO)
																</button>
															</div>
														{/if}

														<!-- Chips / Badges Nomor Seri Terpilih -->
														{#if serialCount > 0}
															<div
																class="mt-2.5 flex flex-wrap gap-1.5 border-t border-amber-200/60 pt-2"
															>
																{#each draft.selectedSerialIds as sId (sId)}
																	{@const unit = draft.availableSerials.find((u) => u.id === sId)}
																	<span
																		class="text-3xs inline-flex items-center gap-1 rounded-md border border-neutral-300 bg-white px-2 py-0.5 font-mono text-neutral-800 shadow-2xs"
																	>
																		<span>{unit?.serial_number ?? sId}</span>
																		<button
																			type="button"
																			onclick={() => handleRemoveSerialFromTransfer(draft, sId)}
																			class="text-neutral-400 hover:text-rose-600"
																			title="Hapus nomor seri"
																		>
																			&times;
																		</button>
																	</span>
																{/each}
															</div>
														{/if}
													{/if}
												</div>
											</td>
										</tr>
									{/if}
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<div
						class="rounded-lg border border-dashed border-neutral-300 p-4 text-center text-xs text-neutral-400"
					>
						Belum ada barang yang ditambahkan ke surat jalan ini.
					</div>
				{/if}
			</div>

			<!-- Catatan Edukatif Invarian -->
			<div class="text-3xs rounded-lg bg-neutral-100 p-2.5 text-neutral-500">
				<strong>Invarian Bisnis Mutasi:</strong> Saat permohonan dibuat, kuantitas produk di cabang
				asal otomatis dialokasikan ke
				<code class="rounded bg-neutral-200 px-1 py-0.5 font-mono text-neutral-800"
					>ReservedQuantity</code
				>
				sehingga tidak dapat dijual ke kasir lain selama menunggu persetujuan.
			</div>
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end gap-2">
				<Button
					variant="outline"
					onclick={() => (showCreateModal = false)}
					disabled={createSubmitting}
				>
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
			<div
				class="flex flex-wrap items-start justify-between gap-3 rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5"
			>
				<div>
					<span class="font-mono text-xs font-bold text-neutral-900"
						>{selectedTransfer.transfer_number}</span
					>
					<div class="text-3xs mt-1 flex items-center gap-2 text-neutral-500">
						<span
							>Diajukan oleh: <strong class="text-neutral-700"
								>{selectedTransfer.requested_by_name || selectedTransfer.requested_by}</strong
							></span
						>
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
				<span class="text-3xs mb-3 block font-semibold tracking-wider text-neutral-400 uppercase"
					>Timeline Alur Status</span
				>
				<div class="text-2xs flex items-center justify-between">
					<!-- Step 1: Diajukan -->
					<div class="flex flex-col items-center text-center">
						<div
							class="text-3xs flex h-7 w-7 items-center justify-center rounded-full bg-neutral-900 font-bold text-white"
						>
							1
						</div>
						<span class="mt-1 font-semibold text-neutral-900">Diajukan</span>
						<span class="text-3xs text-neutral-400">{formatDate(selectedTransfer.created_at)}</span>
					</div>

					<div class="mx-2 h-0.5 flex-1 bg-neutral-200"></div>

					<!-- Step 2: Disetujui -->
					<div class="flex flex-col items-center text-center">
						<div
							class="flex h-7 w-7 items-center justify-center rounded-full {selectedTransfer.status ===
							'rejected'
								? 'bg-rose-600 text-white'
								: ['approved', 'in_transit', 'received'].includes(selectedTransfer.status)
									? 'bg-primary-600 text-white'
									: 'bg-neutral-200 text-neutral-600'} text-3xs font-bold"
						>
							{selectedTransfer.status === 'rejected' ? 'X' : '2'}
						</div>
						<span
							class="mt-1 font-semibold {selectedTransfer.status === 'rejected'
								? 'text-rose-600'
								: ['approved', 'in_transit', 'received'].includes(selectedTransfer.status)
									? 'text-primary-700'
									: 'text-neutral-500'}"
						>
							{selectedTransfer.status === 'rejected' ? 'Ditolak' : 'Disetujui'}
						</span>
						<span class="text-3xs text-neutral-400"
							>{selectedTransfer.approved_by_name || selectedTransfer.approved_by
								? `oleh ${selectedTransfer.approved_by_name || selectedTransfer.approved_by}`
								: '-'}</span
						>
					</div>

					<div class="mx-2 h-0.5 flex-1 bg-neutral-200"></div>

					<!-- Step 3: Dikirim -->
					<div class="flex flex-col items-center text-center">
						<div
							class="flex h-7 w-7 items-center justify-center rounded-full {[
								'in_transit',
								'received'
							].includes(selectedTransfer.status)
								? 'bg-indigo-600 text-white'
								: 'bg-neutral-200 text-neutral-600'} text-3xs font-bold"
						>
							3
						</div>
						<span
							class="mt-1 font-semibold {['in_transit', 'received'].includes(
								selectedTransfer.status
							)
								? 'text-indigo-700'
								: 'text-neutral-500'}"
						>
							Dalam Ekspedisi
						</span>
						<span class="text-3xs text-neutral-400"
							>{['in_transit', 'received'].includes(selectedTransfer.status)
								? 'Armada Jalan'
								: '-'}</span
						>
					</div>

					<div class="mx-2 h-0.5 flex-1 bg-neutral-200"></div>

					<!-- Step 4: Diterima -->
					<div class="flex flex-col items-center text-center">
						<div
							class="flex h-7 w-7 items-center justify-center rounded-full {selectedTransfer.status ===
							'received'
								? 'bg-emerald-600 text-white'
								: 'bg-neutral-200 text-neutral-600'} text-3xs font-bold"
						>
							4
						</div>
						<span
							class="mt-1 font-semibold {selectedTransfer.status === 'received'
								? 'text-emerald-700'
								: 'text-neutral-500'}"
						>
							Diterima Lengkap
						</span>
						<span class="text-3xs text-neutral-400"
							>{selectedTransfer.received_by_name || selectedTransfer.received_by
								? `oleh ${selectedTransfer.received_by_name || selectedTransfer.received_by}`
								: '-'}</span
						>
					</div>
				</div>
			</div>

			<!-- Rute Cabang Asal & Tujuan -->
			<div class="grid grid-cols-1 gap-3 text-xs sm:grid-cols-2">
				<div class="rounded-xl border border-neutral-200 bg-neutral-50/50 p-3">
					<span class="text-3xs font-semibold tracking-wider text-neutral-400 uppercase"
						>Cabang Asal (Pengirim)</span
					>
					<p class="mt-1 font-bold text-neutral-900">
						{locationMap.get(selectedTransfer.from_location_id)?.name ?? '-'}
					</p>
					<p class="text-3xs font-mono text-neutral-500">
						{locationMap.get(selectedTransfer.from_location_id)?.code ?? '-'}
					</p>
					<p class="text-3xs mt-1 line-clamp-1 text-neutral-500">
						{locationMap.get(selectedTransfer.from_location_id)?.address ||
							'Alamat fisik tidak tersedia'}
					</p>
				</div>

				<div class="rounded-xl border border-neutral-200 bg-neutral-50/50 p-3">
					<span class="text-3xs font-semibold tracking-wider text-neutral-400 uppercase"
						>Cabang Tujuan (Penerima)</span
					>
					<p class="mt-1 font-bold text-neutral-900">
						{locationMap.get(selectedTransfer.to_location_id)?.name ?? '-'}
					</p>
					<p class="text-3xs font-mono text-neutral-500">
						{locationMap.get(selectedTransfer.to_location_id)?.code ?? '-'}
					</p>
					<p class="text-3xs mt-1 line-clamp-1 text-neutral-500">
						{locationMap.get(selectedTransfer.to_location_id)?.address ||
							'Alamat fisik tidak tersedia'}
					</p>
				</div>
			</div>

			<!-- Alasan Penolakan Jika Rejected -->
			{#if selectedTransfer.status === 'rejected' && selectedTransfer.rejection_reason}
				<div class="rounded-xl border border-rose-200 bg-rose-50/60 p-3 text-xs text-rose-800">
					<strong class="mb-0.5 block font-semibold">Alasan Penolakan:</strong>
					"{selectedTransfer.rejection_reason}"
				</div>
			{/if}

			<!-- Rincian Barang yang Dimutasi -->
			<div class="overflow-hidden rounded-xl border border-neutral-200">
				<table class="w-full text-left text-xs">
					<thead>
						<tr
							class="text-2xs border-b border-neutral-200 bg-neutral-50 font-semibold tracking-wider text-neutral-500 uppercase"
						>
							<th class="px-4 py-2.5">Produk & SKU</th>
							<th class="px-4 py-2.5 text-center">Kuantitas Permohonan</th>
							<th class="px-4 py-2.5 text-center">Kuantitas Tiba</th>
							<th class="px-4 py-2.5 text-center">Status Unit</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100">
						{#each selectedTransfer.items as it (it.id)}
							<tr>
								<td class="px-4 py-3">
									<span class="block font-semibold text-neutral-900"
										>{productMap.get(it.product_id)?.name ?? 'Produk'}</span
									>
									<span class="text-3xs font-mono text-neutral-500"
										>{productMap.get(it.product_id)?.sku ?? it.product_id}</span
									>
									{#if it.serial_unit_ids && it.serial_unit_ids.length > 0}
										<div class="mt-1">
											<Badge variant="purple" size="sm">
												{it.serial_unit_ids.length} Unit Bernomor Seri
											</Badge>
										</div>
									{/if}
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

				<Button variant="outline" size="sm" onclick={() => (showDetailModal = false)}>Tutup</Button>
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
				<p class="text-3xs mt-1.5 text-neutral-500">
					Alasan penolakan akan dicatat ke dalam dokumen surat jalan dan cadangan stok asal akan
					seketika dilepaskan.
				</p>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end gap-2">
				<Button
					variant="outline"
					onclick={() => (showRejectModal = false)}
					disabled={rejectSubmitting}
				>
					Batal
				</Button>
				<Button variant="danger" loading={rejectSubmitting} onclick={handleExecuteReject}>
					Konfirmasi Tolak Mutasi
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}
