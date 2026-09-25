<script lang="ts">
	/**
	 * Halaman Katalog Produk — Data Master (Gen-E Enterprise).
	 *
	 * Fitur:
	 * - Ringkasan metrik produk (Total, Aktif, Draft, Serial Tracking)
	 * - Filter & pencarian cepat (SKU / Nama, Kategori, Status)
	 * - Tabel katalog produk lengkap (SKU, Nama, Kategori, Harga Pokok & Jual, Serial Flag, Status)
	 * - Modal Tambah & Ubah Produk (SKU, Nama, Kategori, Harga, Unit, Serial Tracking, Atribut Varian Dinamis)
	 * - Modal Ubah Status Cepat (Aktif, Draft, Diarsipkan)
	 * - Paginasi data server-side
	 * - Notifikasi feedback aksi (Alert sukses & error)
	 */
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listProducts,
		createProduct,
		updateProduct,
		setProductStatus,
		listCategories,
		uploadProductImage,
		listProductImages,
		deleteProductImage,
		setPrimaryProductImage,
		listBarcodes
	} from '@erp/api-client';
	import type {
		ProductResponse,
		CategoryResponse,
		ProductStatus,
		CreateProductRequest,
		UpdateProductRequest,
		ProductImageResponse,
		BarcodeResponse
	} from '@erp/types';
	import { PRODUCT_STATUS_LABELS, ApiError } from '@erp/types';
	import {
		Button,
		Input,
		Table,
		Pagination,
		Modal,
		Alert,
		Badge,
		Checkbox,
		SearchInput,
		RichTextEditor,
		Select2,
		Barcode,
		ActionMenu,
		toast,
		type Select2Option
	} from '@erp/ui';

	// State data produk & kategori
	let products = $state<ProductResponse[]>([]);
	let categories = $state<CategoryResponse[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// State Modal Detail Produk
	let showDetailModal = $state(false);
	let detailProduct = $state<ProductResponse | null>(null);
	let detailBarcodes = $state<BarcodeResponse[]>([]);
	let detailImages = $state<ProductImageResponse[]>([]);
	let detailActiveImage = $state<string>('');
	let loadingDetail = $state(false);
	let copiedBarcodeId = $state<string | null>(null);

	// State paginasi & filter
	let searchQuery = $state('');
	let selectedCategoryFilter = $state('');
	let selectedStatusFilter = $state('');
	let currentPage = $state(1);
	let pageSize = $state(10);
	let totalRecords = $state(0);
	const totalPages = $derived(Math.max(1, Math.ceil(totalRecords / pageSize)));

	// Map ID kategori ke nama untuk rendering tabel instan
	const categoryMap = $derived.by(() => {
		const map: Record<string, string> = {};
		for (const cat of categories) {
			map[cat.id] = cat.name;
		}
		return map;
	});

	// Opsi dropdown filter kategori untuk Select2 (Searchable Select)
	const categoryFilterOptions = $derived<Select2Option[]>([
		{ value: '', label: 'Semua Kategori' },
		...categories.map((c) => ({
			value: c.id,
			label: c.name,
			subtext:
				c.parent_id && categoryMap[c.parent_id] ? `Induk: ${categoryMap[c.parent_id]}` : undefined
		}))
	]);

	// Opsi dropdown kategori form untuk Select2 (Searchable Select)
	const categorySelect2Options = $derived<Select2Option[]>(
		categories.map((c) => ({
			value: c.id,
			label: c.name,
			subtext:
				c.parent_id && categoryMap[c.parent_id] ? `Induk: ${categoryMap[c.parent_id]}` : undefined
		}))
	);

	// Opsi dropdown filter status untuk Select2
	const statusFilterOptions: Select2Option[] = [
		{ value: '', label: 'Semua Status' },
		{ value: 'active', label: 'Aktif', subtext: 'Siap ditransaksikan' },
		{ value: 'draft', label: 'Draft', subtext: 'Tahap perencanaan' },
		{ value: 'archived', label: 'Diarsipkan', subtext: 'Tidak dijual' }
	];

	// Opsi status untuk modal ganti status
	const statusOptions: Select2Option[] = [
		{ value: 'active', label: 'Aktif (Siap Ditransaksikan)' },
		{ value: 'draft', label: 'Draft (Tahap Perencanaan)' },
		{ value: 'archived', label: 'Diarsipkan (Tidak Dijual)' }
	];

	// Metrik ringkasan
	const totalActive = $derived(products.filter((p) => p.status === 'active').length);
	const totalDraft = $derived(products.filter((p) => p.status === 'draft').length);
	const totalSerialTracked = $derived(products.filter((p) => p.flag_serial_tracking).length);

	// Form State (Tambah & Edit)
	let showFormModal = $state(false);
	let isEditing = $state(false);
	let editingId = $state<string | null>(null);
	let formSku = $state('');
	let formCategoryId = $state('');
	let formName = $state('');
	let formBrand = $state('');
	let formDescription = $state('');
	let formUnit = $state('PCS');
	let formPurchasePrice = $state(0);
	let formSellingPrice = $state(0);
	let formIsPpn = $state(false);
	let formFlagSerialTracking = $state(false);
	let formWeightKg = $state<number | string>('');

	// Atribut varian generik dinamis (Key-Value)
	interface VariantField {
		key: string;
		value: string;
	}
	let formVariants = $state<VariantField[]>([]);

	let formSubmitting = $state(false);
	let formError = $state<string | null>(null);

	// Foto Produk State
	let productImages = $state<ProductImageResponse[]>([]);
	let loadingImages = $state(false);
	let imageError = $state<string | null>(null);
	let uploadingImage = $state(false);
	let isDragging = $state(false);
	let pendingFiles = $state<{ file: File; previewUrl: string }[]>([]);

	// State Drag & Drop Reorder Foto (Paling kiri otomatis foto utama)
	let draggedPhotoIndex = $state<number | null>(null);
	let dragOverPhotoIndex = $state<number | null>(null);
	let dragTargetList = $state<'pending' | 'editing' | null>(null);

	// Helper URL Gambar
	function getImageUrl(url?: string | null): string {
		if (!url) return '';
		if (url.startsWith('http')) return url;
		return `http://localhost:8088${url}`;
	}

	async function loadProductImagesData(productId: string) {
		const token = getToken();
		if (!token) return;
		loadingImages = true;
		imageError = null;
		try {
			productImages = await listProductImages(token, productId);
		} catch (err: unknown) {
			console.error('Gagal memuat foto produk:', err);
		} finally {
			loadingImages = false;
		}
	}

	async function handleUploadSingleFile(file: File) {
		const token = getToken();
		if (!token) return;

		if (isEditing && editingId) {
			uploadingImage = true;
			imageError = null;
			try {
				const uploaded = await uploadProductImage(token, editingId, file);
				productImages = [...productImages, uploaded];
				if (uploaded.is_primary) {
					await loadProducts();
				}
			} catch (err: unknown) {
				if (err instanceof ApiError) {
					imageError = err.message;
				} else {
					imageError = 'Gagal mengunggah foto';
				}
			} finally {
				uploadingImage = false;
			}
		} else {
			const previewUrl = URL.createObjectURL(file);
			pendingFiles = [...pendingFiles, { file, previewUrl }];
		}
	}

	async function processFiles(files: File[]) {
		const allowedTypes = ['image/jpeg', 'image/png', 'image/webp'];
		const maxSizeBytes = 5 * 1024 * 1024; // 5 MB

		imageError = null;

		for (const file of files) {
			if (!allowedTypes.includes(file.type)) {
				imageError = `Format berkas "${file.name}" tidak didukung. Gunakan format JPG, PNG, atau WebP.`;
				continue;
			}
			if (file.size > maxSizeBytes) {
				imageError = `Ukuran berkas "${file.name}" melebihi batas 5 MB.`;
				continue;
			}
			await handleUploadSingleFile(file);
		}
	}

	function handleFileInputChange(e: Event) {
		const target = e.target as HTMLInputElement;
		if (!target.files || target.files.length === 0) return;
		const files = Array.from(target.files);
		processFiles(files);
		target.value = '';
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (!uploadingImage) {
			isDragging = true;
		}
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		isDragging = false;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		isDragging = false;
		if (uploadingImage) return;

		const dt = e.dataTransfer;
		if (!dt || !dt.files || dt.files.length === 0) return;

		const files = Array.from(dt.files);
		processFiles(files);
	}

	async function handleDeleteImage(imageId: string) {
		const token = getToken();
		if (!token || !editingId) return;

		uploadingImage = true;
		imageError = null;
		try {
			await deleteProductImage(token, editingId, imageId);
			productImages = productImages.filter((img) => img.id !== imageId);
			await loadProducts();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				imageError = err.message;
			} else {
				imageError = 'Gagal menghapus foto';
			}
		} finally {
			uploadingImage = false;
		}
	}

	async function handleSetPrimaryImage(imageId: string) {
		const token = getToken();
		if (!token || !editingId) return;

		uploadingImage = true;
		imageError = null;
		try {
			await setPrimaryProductImage(token, editingId, imageId);
			const updated = productImages.map((img) => ({
				...img,
				is_primary: img.id === imageId
			}));
			// Pastikan foto utama otomatis dipindah ke index 0 (paling kiri)
			const primaryIndex = updated.findIndex((img) => img.id === imageId);
			if (primaryIndex > 0) {
				const [primaryImg] = updated.splice(primaryIndex, 1);
				productImages = [primaryImg, ...updated];
			} else {
				productImages = updated;
			}
			await loadProducts();
			toast.success('Foto utama berhasil ditetapkan.');
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				imageError = err.message;
			} else {
				imageError = 'Gagal menetapkan foto utama';
			}
		} finally {
			uploadingImage = false;
		}
	}

	// Handlers Drag & Drop Reorder Foto Kartu
	function handleCardDragStart(e: DragEvent, index: number, listType: 'pending' | 'editing') {
		if (!e.dataTransfer) return;
		e.dataTransfer.effectAllowed = 'move';
		e.dataTransfer.setData('text/plain', String(index));
		draggedPhotoIndex = index;
		dragTargetList = listType;
	}

	function handleCardDragOver(e: DragEvent, index: number, listType: 'pending' | 'editing') {
		if (dragTargetList !== listType || draggedPhotoIndex === null) return;
		e.preventDefault();
		e.stopPropagation();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
		if (dragOverPhotoIndex !== index) {
			dragOverPhotoIndex = index;
		}
	}

	function handleCardDragLeave(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
	}

	async function handleCardDrop(
		e: DragEvent,
		targetIndex: number,
		listType: 'pending' | 'editing'
	) {
		if (dragTargetList !== listType || draggedPhotoIndex === null) return;
		e.preventDefault();
		e.stopPropagation();

		const sourceIndex = draggedPhotoIndex;
		draggedPhotoIndex = null;
		dragOverPhotoIndex = null;
		dragTargetList = null;

		if (sourceIndex === targetIndex) return;

		if (listType === 'pending') {
			const updated = [...pendingFiles];
			const [movedItem] = updated.splice(sourceIndex, 1);
			updated.splice(targetIndex, 0, movedItem);
			pendingFiles = updated;
			toast.info('Urutan foto diubah. Foto paling kiri otomatis menjadi Foto Utama.');
		} else if (listType === 'editing') {
			const updated = [...productImages];
			const [movedItem] = updated.splice(sourceIndex, 1);
			updated.splice(targetIndex, 0, movedItem);
			productImages = updated;

			// Jika foto yang ditarik ke posisi 0 (paling kiri) belum primary di backend
			if (targetIndex === 0 && !movedItem.is_primary) {
				await handleSetPrimaryImage(movedItem.id);
			} else if (sourceIndex === 0 && !updated[0].is_primary) {
				// Jika foto utama lama dipindah keluar dari posisi 0, jadikan foto baru di posisi 0 sebagai primary
				await handleSetPrimaryImage(updated[0].id);
			} else {
				toast.info('Urutan foto berhasil diubah.');
			}
		}
	}

	function handleCardDragEnd() {
		draggedPhotoIndex = null;
		dragOverPhotoIndex = null;
		dragTargetList = null;
	}

	function handleRemovePendingFile(index: number) {
		const item = pendingFiles[index];
		if (item) {
			URL.revokeObjectURL(item.previewUrl);
		}
		pendingFiles = pendingFiles.filter((_, i) => i !== index);
	}

	function closeFormModal() {
		showFormModal = false;
		for (const item of pendingFiles) {
			URL.revokeObjectURL(item.previewUrl);
		}
		pendingFiles = [];
		productImages = [];
		imageError = null;
		isDragging = false;
		draggedPhotoIndex = null;
		dragOverPhotoIndex = null;
		dragTargetList = null;
	}

	// Modal Ubah Status
	let showStatusModal = $state(false);
	let statusTargetProduct = $state<ProductResponse | null>(null);
	let newStatusSelection = $state<ProductStatus>('active');
	let statusSubmitting = $state(false);
	let statusError = $state<string | null>(null);

	// Format mata uang Rupiah
	function formatRupiah(amount: number): string {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			maximumFractionDigits: 0
		}).format(amount);
	}

	function formatDate(dateStr?: string | null): string {
		if (!dateStr) return '-';
		try {
			return new Date(dateStr).toLocaleDateString('id-ID', {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}

	async function openDetailModal(prod: ProductResponse) {
		detailProduct = prod;
		detailBarcodes = [];
		detailImages = [];
		detailActiveImage = prod.primary_image_url ? getImageUrl(prod.primary_image_url) : '';
		loadingDetail = true;
		showDetailModal = true;

		const token = getToken();
		if (!token) {
			loadingDetail = false;
			return;
		}

		try {
			const [bcs, imgs] = await Promise.all([
				listBarcodes(token, prod.id).catch(() => []),
				listProductImages(token, prod.id).catch(() => [])
			]);
			detailBarcodes = bcs;
			detailImages = imgs;
			if (imgs.length > 0) {
				const primary = imgs.find((i) => i.is_primary) ?? imgs[0];
				detailActiveImage = getImageUrl(primary.url);
			}
		} catch (err: unknown) {
			console.error('Gagal memuat data detail produk:', err);
		} finally {
			loadingDetail = false;
		}
	}

	function copyBarcode(code: string, id: string) {
		if (navigator.clipboard) {
			navigator.clipboard.writeText(code);
		}
		copiedBarcodeId = id;
		setTimeout(() => {
			if (copiedBarcodeId === id) {
				copiedBarcodeId = null;
			}
		}, 2000);
	}

	async function loadCategoriesData() {
		const token = getToken();
		if (!token) return;
		try {
			const data = await listCategories(token);
			categories = data;
		} catch (err: unknown) {
			console.error('Gagal memuat kategori:', err);
		}
	}

	async function loadProducts() {
		const token = getToken();
		if (!token) return;

		loading = true;
		error = null;
		try {
			const res = await listProducts(token, {
				search: searchQuery.trim() || undefined,
				category_id: selectedCategoryFilter || undefined,
				status: selectedStatusFilter || undefined,
				page: currentPage,
				limit: pageSize
			});

			products = res.data ?? [];
			totalRecords = res.total ?? 0;
			currentPage = res.page ?? 1;
			pageSize = res.limit ?? 10;
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat data katalog produk';
			}
		} finally {
			loading = false;
		}
	}

	function handleSearch(query: string) {
		searchQuery = query;
		currentPage = 1;
		loadProducts();
	}

	function handleCategoryChange(val: string | number) {
		selectedCategoryFilter = String(val);
		currentPage = 1;
		loadProducts();
	}

	function handleStatusFilterChange(val: string | number) {
		selectedStatusFilter = String(val);
		currentPage = 1;
		loadProducts();
	}

	function handlePageChange(newPage: number) {
		currentPage = newPage;
		loadProducts();
	}

	function openCreateModal() {
		isEditing = false;
		editingId = null;
		formSku = '';
		formCategoryId = categories.length > 0 ? categories[0].id : '';
		formName = '';
		formBrand = '';
		formDescription = '';
		formUnit = 'PCS';
		formPurchasePrice = 0;
		formSellingPrice = 0;
		formIsPpn = false;
		formFlagSerialTracking = false;
		formWeightKg = '';
		formVariants = [];
		formError = null;
		productImages = [];
		pendingFiles = [];
		imageError = null;
		isDragging = false;
		draggedPhotoIndex = null;
		dragOverPhotoIndex = null;
		dragTargetList = null;
		showFormModal = true;
	}

	function openEditModal(prod: ProductResponse) {
		isEditing = true;
		editingId = prod.id;
		formSku = prod.sku;
		formCategoryId = prod.category_id;
		formName = prod.name;
		formBrand = prod.brand ?? '';
		formDescription = prod.description ?? '';
		formUnit = prod.unit || 'PCS';
		formPurchasePrice = prod.purchase_price || 0;
		formSellingPrice = prod.selling_price || 0;
		formIsPpn = prod.is_ppn || false;
		formFlagSerialTracking = prod.flag_serial_tracking || false;
		formWeightKg = prod.weight_gram ? prod.weight_gram / 1000 : '';

		// Parse atribut varian ke list key-value
		const variants: VariantField[] = [];
		if (prod.atribut_varian && typeof prod.atribut_varian === 'object') {
			for (const [k, v] of Object.entries(prod.atribut_varian)) {
				variants.push({ key: k, value: String(v) });
			}
		}
		formVariants = variants;

		formError = null;
		productImages = [];
		pendingFiles = [];
		imageError = null;
		isDragging = false;
		draggedPhotoIndex = null;
		dragOverPhotoIndex = null;
		dragTargetList = null;
		showFormModal = true;
		loadProductImagesData(prod.id);
	}

	function focusVariantInput(index: number, field: 'key' | 'value') {
		setTimeout(() => {
			const el = document.getElementById(`variant-${field}-${index}`) as HTMLInputElement | null;
			el?.focus();
		}, 30);
	}

	function addVariantField() {
		formVariants = [...formVariants, { key: '', value: '' }];
		const newIndex = formVariants.length - 1;
		focusVariantInput(newIndex, 'key');
	}

	function removeVariantField(index: number) {
		formVariants = formVariants.filter((_, i) => i !== index);
	}

	function handleVariantKeyDown(e: KeyboardEvent, index: number, field: 'key' | 'value') {
		if (e.key === 'Enter') {
			e.preventDefault();
			e.stopPropagation();

			if (field === 'key' && formVariants[index]?.key.trim() && !formVariants[index]?.value) {
				focusVariantInput(index, 'value');
			} else {
				addVariantField();
			}
		}
	}

	async function handleSubmitForm(e: SubmitEvent) {
		e.preventDefault();
		const token = getToken();
		if (!token) return;

		if (!formName.trim()) {
			formError = 'Nama produk wajib diisi';
			return;
		}

		if (!isEditing && !formSku.trim()) {
			formError = 'SKU produk wajib diisi';
			return;
		}

		if (!formCategoryId) {
			formError = 'Kategori produk wajib dipilih';
			return;
		}

		if (formSellingPrice < formPurchasePrice) {
			formError = 'Harga jual tidak boleh lebih rendah dari harga pokok (HPP)';
			return;
		}

		// Rangkum atribut varian menjadi Record<string, unknown>
		const variantRecord: Record<string, unknown> = {};
		for (const vf of formVariants) {
			if (vf.key.trim()) {
				variantRecord[vf.key.trim()] = vf.value.trim();
			}
		}

		formSubmitting = true;
		formError = null;

		// Konversi bobot input (Kg) ke satuan basis database (Gram): 1 kg = 1000 gram
		const numWeight =
			typeof formWeightKg === 'string' ? parseFloat(formWeightKg) : Number(formWeightKg);
		const weightInGram =
			!isNaN(numWeight) && numWeight > 0 ? Math.round(numWeight * 1000) : undefined;

		try {
			if (isEditing && editingId) {
				const updatePayload: UpdateProductRequest = {
					name: formName.trim(),
					brand: formBrand.trim() || undefined,
					description: formDescription.trim() || undefined,
					unit: formUnit.trim() || 'PCS',
					purchase_price: formPurchasePrice,
					selling_price: formSellingPrice,
					is_ppn: formIsPpn,
					flag_serial_tracking: formFlagSerialTracking,
					weight_gram: weightInGram,
					atribut_varian: Object.keys(variantRecord).length > 0 ? variantRecord : undefined
				};

				await updateProduct(token, editingId, updatePayload);
				toast.success(`Produk "${formName}" berhasil diperbarui.`);
			} else {
				const createPayload: CreateProductRequest = {
					sku: formSku.trim().toUpperCase(),
					category_id: formCategoryId,
					name: formName.trim(),
					brand: formBrand.trim() || undefined,
					description: formDescription.trim() || undefined,
					unit: formUnit.trim() || 'PCS',
					purchase_price: formPurchasePrice,
					selling_price: formSellingPrice,
					is_ppn: formIsPpn,
					flag_serial_tracking: formFlagSerialTracking,
					weight_gram: weightInGram,
					atribut_varian: Object.keys(variantRecord).length > 0 ? variantRecord : undefined
				};

				const created = await createProduct(token, createPayload);
				toast.success(
					`Produk baru "${formName}" (SKU: ${formSku.toUpperCase()}) berhasil ditambahkan.`
				);

				// Unggah berkas gambar yang diantrekan jika ada
				if (pendingFiles.length > 0) {
					for (let i = 0; i < pendingFiles.length; i++) {
						const item = pendingFiles[i];
						try {
							await uploadProductImage(token, created.id, item.file, i === 0);
						} catch (uploadErr) {
							console.error('Gagal upload pending foto:', uploadErr);
						} finally {
							URL.revokeObjectURL(item.previewUrl);
						}
					}
					pendingFiles = [];
				}
			}

			closeFormModal();
			await loadProducts();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				formError = err.message;
			} else if (err instanceof Error) {
				formError = err.message;
			} else {
				formError = 'Terjadi kesalahan saat menyimpan produk';
			}
		} finally {
			formSubmitting = false;
		}
	}

	function openStatusModal(prod: ProductResponse) {
		statusTargetProduct = prod;
		newStatusSelection = prod.status;
		statusError = null;
		showStatusModal = true;
	}

	async function handleStatusSubmit() {
		const token = getToken();
		if (!token || !statusTargetProduct) return;

		statusSubmitting = true;
		statusError = null;
		try {
			await setProductStatus(token, statusTargetProduct.id, newStatusSelection);
			toast.success(
				`Status produk "${statusTargetProduct.name}" berhasil diubah menjadi "${PRODUCT_STATUS_LABELS[newStatusSelection]}".`
			);
			showStatusModal = false;
			await loadProducts();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				statusError = err.message;
			} else if (err instanceof Error) {
				statusError = err.message;
			} else {
				statusError = 'Gagal memperbarui status produk';
			}
		} finally {
			statusSubmitting = false;
		}
	}

	onMount(async () => {
		await Promise.all([loadCategoriesData(), loadProducts()]);
	});
</script>

<svelte:head>
	<title>Katalog Produk — Data Master Gen-E</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header Halaman & Aksi Utama -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-neutral-900">Katalog Produk</h1>
			<p class="mt-1 text-sm text-neutral-500">
				Kelola data master barang dagangan, spesifikasi harga, penomoran seri, dan varian produk.
			</p>
		</div>
		<div class="flex items-center gap-3">
			<Button variant="primary" onclick={openCreateModal}>
				<svg
					class="mr-2 h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Tambah Produk
			</Button>
		</div>
	</div>

	<!-- Notifikasi Feedback Global -->
	{#if error}
		<Alert variant="error" dismissible title="Terjadi Kesalahan">
			{error}
		</Alert>
	{/if}

	<!-- Kartu Ringkasan Metrik -->
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<!-- Total Katalog -->
		<div class="rounded-xl border border-neutral-200/80 bg-white p-5 shadow-xs">
			<div class="flex items-center justify-between">
				<span class="text-xs font-medium tracking-wider text-neutral-500 uppercase"
					>Total Katalog</span
				>
				<span class="rounded-lg bg-neutral-100 p-2 text-neutral-600">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
						/>
					</svg>
				</span>
			</div>
			<div class="mt-4 flex items-baseline gap-2">
				<span class="text-2xl font-bold text-neutral-900">{totalRecords}</span>
				<span class="text-xs text-neutral-400">item barang</span>
			</div>
		</div>

		<!-- Produk Aktif -->
		<div class="rounded-xl border border-neutral-200/80 bg-white p-5 shadow-xs">
			<div class="flex items-center justify-between">
				<span class="text-xs font-medium tracking-wider text-emerald-600 uppercase"
					>Produk Aktif</span
				>
				<span class="rounded-lg bg-emerald-50 p-2 text-emerald-600">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				</span>
			</div>
			<div class="mt-4 flex items-baseline gap-2">
				<span class="text-2xl font-bold text-emerald-700">{totalActive}</span>
				<span class="text-xs text-neutral-400">siap dijual</span>
			</div>
		</div>

		<!-- Produk Draft -->
		<div class="rounded-xl border border-neutral-200/80 bg-white p-5 shadow-xs">
			<div class="flex items-center justify-between">
				<span class="text-xs font-medium tracking-wider text-amber-600 uppercase">Draft Produk</span
				>
				<span class="rounded-lg bg-amber-50 p-2 text-amber-600">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10"
						/>
					</svg>
				</span>
			</div>
			<div class="mt-4 flex items-baseline gap-2">
				<span class="text-2xl font-bold text-amber-700">{totalDraft}</span>
				<span class="text-xs text-neutral-400">belum dipublikasi</span>
			</div>
		</div>

		<!-- Pelacakan Serial / IMEI -->
		<div class="rounded-xl border border-neutral-200/80 bg-white p-5 shadow-xs">
			<div class="flex items-center justify-between">
				<span class="text-xs font-medium tracking-wider text-purple-600 uppercase"
					>Lacak Seri / IMEI</span
				>
				<span class="rounded-lg bg-purple-50 p-2 text-purple-600">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M10.5 1.5H8.25A2.25 2.25 0 006 3.75v16.5a2.25 2.25 0 002.25 2.25h7.5A2.25 2.25 0 0018 20.25V3.75a2.25 2.25 0 00-2.25-2.25H13.5m-3 0V3h3V1.5m-3 0h3m-3 18.75h3"
						/>
					</svg>
				</span>
			</div>
			<div class="mt-4 flex items-baseline gap-2">
				<span class="text-2xl font-bold text-purple-700">{totalSerialTracked}</span>
				<span class="text-xs text-neutral-400">katalog ber-IMEI</span>
			</div>
		</div>
	</div>

	<!-- Bar Pencarian & Filter Cepat -->
	<div
		class="flex flex-col gap-3 rounded-xl border border-neutral-200/80 bg-white p-4 shadow-xs md:flex-row md:items-center md:justify-between"
	>
		<div class="w-full md:max-w-xs">
			<SearchInput
				placeholder="Cari SKU atau nama produk..."
				value={searchQuery}
				onsearch={handleSearch}
			/>
		</div>

		<div class="flex flex-wrap items-center gap-3">
			<div class="w-56">
				<Select2
					options={categoryFilterOptions}
					bind:value={selectedCategoryFilter}
					placeholder="Semua Kategori"
					searchPlaceholder="Cari kategori..."
					clearable={true}
					onchange={handleCategoryChange}
				/>
			</div>

			<div class="w-48">
				<Select2
					options={statusFilterOptions}
					bind:value={selectedStatusFilter}
					placeholder="Semua Status"
					searchPlaceholder="Cari status..."
					clearable={true}
					onchange={handleStatusFilterChange}
				/>
			</div>
		</div>
	</div>

	<!-- Tabel Katalog Produk -->
	<Table
		empty={!loading && products.length === 0}
		emptyTitle="Belum Ada Katalog Produk"
		emptyMessage="Tambahkan katalog produk pertama untuk mulai mendata stok, harga, dan varian."
		{loading}
	>
		<thead>
			<tr
				class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-bold tracking-wider text-neutral-500 uppercase"
			>
				<th class="px-4 py-3 text-left">SKU</th>
				<th class="px-4 py-3 text-left">Nama & Merek</th>
				<th class="px-4 py-3 text-left">Kategori</th>
				<th class="px-4 py-3 text-right">Harga Pokok (HPP)</th>
				<th class="px-4 py-3 text-right">Harga Jual</th>
				<th class="px-4 py-3 text-center">Lacak Unit</th>
				<th class="px-4 py-3 text-center">Status</th>
				<th class="px-4 py-3 text-right">Aksi</th>
			</tr>
		</thead>
		<tbody class="divide-y divide-neutral-100">
			{#each products as prod (prod.id)}
				<tr class="transition-colors hover:bg-neutral-50/70">
					<!-- SKU Monospace Soft Indigo -->
					<td class="px-4 py-3 whitespace-nowrap">
						<Badge variant="indigo" size="sm">
							<span class="font-mono font-semibold tracking-tight">{prod.sku}</span>
						</Badge>
					</td>

					<!-- Nama Produk, Foto & Brand (Klik untuk buka detail) -->
					<td class="px-4 py-3">
						<button
							type="button"
							onclick={() => openDetailModal(prod)}
							class="group/title flex items-center gap-3 text-left focus:outline-hidden"
						>
							<!-- Thumbnail Foto Produk -->
							<div
								class="relative h-11 w-11 shrink-0 overflow-hidden rounded-xl border border-neutral-200/80 bg-neutral-100 shadow-2xs transition-colors group-hover/title:border-primary-400"
							>
								{#if prod.primary_image_url}
									<img
										src={getImageUrl(prod.primary_image_url)}
										alt={prod.name}
										class="h-full w-full object-cover"
									/>
								{:else}
									<div class="flex h-full w-full items-center justify-center text-neutral-400">
										<svg
											class="h-5 w-5"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											stroke-width="1.5"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z"
											/>
										</svg>
									</div>
								{/if}
							</div>

							<div class="flex flex-col">
								<span
									class="text-sm font-semibold text-neutral-900 transition-colors group-hover/title:text-primary-600"
									>{prod.name}</span
								>
								<div class="mt-0.5 flex items-center gap-2 text-xs text-neutral-500">
									{#if prod.brand}
										<span>{prod.brand}</span>
										<span>•</span>
									{/if}
									<span>Satuan: {prod.unit}</span>
									{#if prod.weight_gram}
										<span>•</span>
										<span>{(prod.weight_gram / 1000).toLocaleString('id-ID')} kg</span>
									{/if}
								</div>
							</div>
						</button>
					</td>

					<!-- Kategori -->
					<td class="px-4 py-3 whitespace-nowrap">
						<span class="text-xs text-neutral-600">
							{categoryMap[prod.category_id] ?? 'Kategori Lain'}
						</span>
					</td>

					<!-- Harga Pokok -->
					<td class="px-4 py-3 text-right whitespace-nowrap">
						{#if prod.purchase_price !== null && prod.purchase_price !== undefined}
							<span class="text-xs text-neutral-600">
								{formatRupiah(prod.purchase_price)}
							</span>
						{:else}
							<span class="font-mono text-xs text-neutral-400" title="Tersensor (Butuh Izin PBAC)">
								••••••••
							</span>
						{/if}
					</td>

					<!-- Harga Jual -->
					<td class="px-4 py-3 text-right whitespace-nowrap">
						<span class="text-sm font-semibold text-neutral-900">
							{formatRupiah(prod.selling_price)}
						</span>
						{#if prod.is_ppn}
							<span
								class="ml-1 inline-block rounded border border-cyan-200 bg-cyan-50 px-1.5 py-0.5 text-[10px] font-medium text-cyan-700"
							>
								PPN
							</span>
						{/if}
					</td>

					<!-- Lacak Seri / IMEI Soft Purple -->
					<td class="px-4 py-3 text-center whitespace-nowrap">
						{#if prod.flag_serial_tracking}
							<Badge variant="purple" size="sm">Serial / IMEI</Badge>
						{:else}
							<span class="text-xs text-neutral-400">Biasa (Qty)</span>
						{/if}
					</td>

					<!-- Status Produk -->
					<td class="px-4 py-3 text-center whitespace-nowrap">
						{#if prod.status === 'active'}
							<Badge variant="success" size="sm">Aktif</Badge>
						{:else if prod.status === 'draft'}
							<Badge variant="warning" size="sm">Draft</Badge>
						{:else}
							<Badge variant="default" size="sm">Diarsipkan</Badge>
						{/if}
					</td>

					<!-- Tombol Aksi (Titik Tiga Dropdown) -->
					<td class="px-4 py-3 text-right whitespace-nowrap">
						<div class="flex items-center justify-end">
							<ActionMenu
								items={[
									{
										label: 'Lihat Detail',
										iconSvg:
											'M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178zM15 12a3 3 0 11-6 0 3 3 0 016 0z',
										onclick: () => openDetailModal(prod)
									},
									{
										label: 'Kelola Barcode',
										iconSvg:
											'M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z',
										onclick: () => goto(`/master/barcodes?product_id=${prod.id}`)
									},
									{
										label: 'Ubah Status',
										iconSvg:
											'M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99',
										onclick: () => openStatusModal(prod)
									},
									{
										label: 'Edit Produk',
										divider: true,
										iconSvg:
											'M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10',
										onclick: () => openEditModal(prod)
									}
								]}
							/>
						</div>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>

	<!-- Paginasi -->
	{#if totalRecords > 0}
		<div class="border-t border-neutral-200/80 px-4 py-3">
			<Pagination
				page={currentPage}
				{totalPages}
				totalItems={totalRecords}
				limit={pageSize}
				onPageChange={handlePageChange}
			/>
		</div>
	{/if}
</div>

<!-- Modal Form Tambah / Ubah Produk -->
<Modal
	open={showFormModal}
	title={isEditing ? 'Ubah Informasi Katalog Produk' : 'Tambah Katalog Produk Baru'}
	size="4xl"
	onclose={() => (showFormModal = false)}
>
	<form id="product-form" onsubmit={handleSubmitForm} class="space-y-5">
		{#if formError}
			<Alert variant="error" title="Gagal Menyimpan">{formError}</Alert>
		{/if}

		<!-- Grid 2 Kolom Modal -->
		<div class="grid grid-cols-1 items-start gap-8 lg:grid-cols-2">
			<!-- KOLOM KIRI: Identitas Produk, Harga & Pajak, Deskripsi -->
			<div class="space-y-4">
				<!-- Baris 1: SKU & Kategori -->
				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
					<div>
						<label for="form-sku" class="mb-1.5 block text-xs font-medium text-neutral-700">
							Kode SKU <span class="text-rose-500">*</span>
						</label>
						<Input
							id="form-sku"
							placeholder="Contoh: EL-SM-001"
							bind:value={formSku}
							disabled={isEditing}
							required
						/>
						{#if isEditing}
							<p class="mt-1 text-[11px] text-neutral-400">
								SKU tidak dapat diubah setelah dibuat.
							</p>
						{/if}
					</div>

					<div>
						<label for="form-category" class="mb-1.5 block text-xs font-medium text-neutral-700">
							Kategori Produk <span class="text-rose-500">*</span>
						</label>
						<Select2
							id="form-category"
							options={categorySelect2Options}
							bind:value={formCategoryId}
							placeholder="Pilih atau cari kategori..."
							searchPlaceholder="Ketik untuk mencari kategori..."
							disabled={isEditing}
							required
						/>
					</div>
				</div>

				<!-- Baris 2: Nama Produk -->
				<div>
					<label for="form-name" class="mb-1.5 block text-xs font-medium text-neutral-700">
						Nama Produk <span class="text-rose-500">*</span>
					</label>
					<Input
						id="form-name"
						placeholder="Contoh: Smartphone Galaxy S24 Ultra 256GB"
						bind:value={formName}
						required
					/>
				</div>

				<!-- Baris 3: Merek, Satuan, Berat -->
				<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
					<div>
						<label for="form-brand" class="mb-1.5 block text-xs font-medium text-neutral-700">
							Merek / Brand
						</label>
						<Input id="form-brand" placeholder="Contoh: Samsung" bind:value={formBrand} />
					</div>

					<div>
						<label for="form-unit" class="mb-1.5 block text-xs font-medium text-neutral-700">
							Satuan Kemasan
						</label>
						<Input id="form-unit" placeholder="PCS / UNIT" bind:value={formUnit} />
					</div>

					<div>
						<label for="form-weight" class="mb-1.5 block text-xs font-medium text-neutral-700">
							Berat (Kg)
						</label>
						<Input
							id="form-weight"
							type="number"
							step="any"
							min="0"
							placeholder="0"
							bind:value={formWeightKg}
						/>
					</div>
				</div>

				<!-- Baris 4: Harga Pokok, Harga Jual, dan PPN (Berdekatan) -->
				<div class="space-y-3 rounded-xl border border-neutral-200/80 bg-neutral-50/50 p-4">
					<div class="flex items-center justify-between">
						<span class="text-xs font-semibold text-neutral-900">Penetapan Harga & Pajak</span>
						<Badge variant="cyan" size="sm">IDR (Rupiah)</Badge>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div>
							<label
								for="form-purchase-price"
								class="mb-1.5 block text-xs font-medium text-neutral-700"
							>
								Harga Pokok (HPP) <span class="text-rose-500">*</span>
							</label>
							<Input
								id="form-purchase-price"
								thousandSeparator
								prefix="Rp"
								placeholder="0"
								bind:value={formPurchasePrice}
								required
							/>
						</div>

						<div>
							<label
								for="form-selling-price"
								class="mb-1.5 block text-xs font-medium text-neutral-700"
							>
								Harga Jual Standar <span class="text-rose-500">*</span>
							</label>
							<Input
								id="form-selling-price"
								thousandSeparator
								prefix="Rp"
								placeholder="0"
								bind:value={formSellingPrice}
								required
							/>
						</div>
					</div>

					<!-- Checkbox PPN Berdekatan dengan Harga -->
					<div class="border-t border-neutral-200/70 pt-2">
						<Checkbox
							id="form-ppn"
							label="Harga jual sudah termasuk PPN (Pajak Pertambahan Nilai)"
							bind:checked={formIsPpn}
						/>
						<p class="mt-1 ml-6 text-[11px] text-neutral-500">
							Centang jika nominal harga jual di atas sudah merupakan harga bersih konsumen
							(termasuk PPN).
						</p>
					</div>
				</div>
			</div>

			<!-- KOLOM KANAN: Wajib Lacak IMEI & Atribut Varian -->
			<div class="space-y-4">
				<!-- Box Wajib Lacak Nomor Seri / IMEI -->
				<div class="rounded-xl border border-purple-200/80 bg-purple-50/40 p-4">
					<div class="flex items-start gap-3">
						<div class="pt-0.5">
							<Checkbox id="form-serial" bind:checked={formFlagSerialTracking} />
						</div>
						<div class="flex-1">
							<div class="flex items-center gap-2">
								<label
									for="form-serial"
									class="cursor-pointer text-xs font-semibold text-neutral-900"
								>
									Wajib Lacak Nomor Seri / IMEI
								</label>
								<Badge variant="purple" size="sm">Unit Fisik</Badge>
							</div>
							<p class="mt-1 text-[11px] leading-relaxed text-neutral-600">
								Aktifkan untuk barang yang wajib dicatat nomor seri uniknya per unit (misal: IMEI
								smartphone, Serial Number laptop/TV). Inventaris akan mewajibkan scan serial saat
								penerimaan dan penjualan.
							</p>
						</div>
					</div>
				</div>

				<!-- Box Atribut Varian Generik -->
				<div class="space-y-3 rounded-xl border border-neutral-200/80 bg-white p-4">
					<div class="flex items-center justify-between">
						<div>
							<div class="flex items-center gap-2">
								<h3 class="text-xs font-semibold text-neutral-900">Atribut Varian Generik</h3>
								<Badge variant="default" size="sm">Opsional</Badge>
							</div>
							<p class="mt-0.5 text-[11px] text-neutral-500">
								Spesifikasi fleksibel (contoh: Warna, Kapasitas, Daya, Dimensi).
							</p>
						</div>
						<Button variant="secondary" size="sm" type="button" onclick={addVariantField}>
							+ Tambah Atribut
						</Button>
					</div>

					{#if formVariants.length > 0}
						<div class="space-y-2 pt-1">
							{#each formVariants as vf, idx (idx)}
								<div class="flex items-center gap-2">
									<div class="w-1/2">
										<Input
											id={`variant-key-${idx}`}
											placeholder="Nama (misal: Warna)"
											bind:value={vf.key}
											onkeydown={(e) => handleVariantKeyDown(e, idx, 'key')}
										/>
									</div>
									<div class="w-1/2">
										<Input
											id={`variant-value-${idx}`}
											placeholder="Nilai (misal: Titanium Gray)"
											bind:value={vf.value}
											onkeydown={(e) => handleVariantKeyDown(e, idx, 'value')}
										/>
									</div>
									<button
										type="button"
										onclick={() => removeVariantField(idx)}
										class="shrink-0 rounded-lg p-2 text-rose-500 transition-colors hover:bg-rose-50 hover:text-rose-700"
										title="Hapus Atribut"
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
								</div>
							{/each}
						</div>
					{:else}
						<div
							class="rounded-lg border border-dashed border-neutral-200 bg-neutral-50/50 p-6 text-center"
						>
							<svg
								class="mx-auto h-7 w-7 text-neutral-300"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="1.5"
									d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75"
								/>
							</svg>
							<p class="mt-2 text-xs font-medium text-neutral-600">Belum ada atribut varian</p>
							<p class="mt-0.5 text-[11px] text-neutral-400">
								Klik tombol "+ Tambah Atribut" untuk menambahkan spesifikasi khusus.
							</p>
						</div>
					{/if}
				</div>
			</div>

			<!-- Foto Produk & Galeri (Melintang 2 Kolom) -->
			<div
				class="space-y-3 rounded-xl border border-neutral-200/80 bg-neutral-50/40 p-4 lg:col-span-2"
			>
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<span class="text-xs font-bold text-neutral-900">Foto Produk</span>
						<span
							class="inline-flex items-center rounded-full bg-neutral-200/70 px-2 py-0.5 text-[11px] font-semibold text-neutral-700"
						>
							{isEditing ? productImages.length : pendingFiles.length} foto
						</span>
					</div>
					{#if uploadingImage}
						<span class="animate-pulse text-xs font-medium text-primary-600">Memproses foto...</span
						>
					{/if}
				</div>

				{#if imageError}
					<Alert variant="error" title="Gagal Memproses Foto" dismissible>{imageError}</Alert>
				{/if}

				<!-- Dropzone Unggah Foto -->
				<label
					ondragover={handleDragOver}
					ondragenter={handleDragOver}
					ondragleave={handleDragLeave}
					ondrop={handleDrop}
					class="flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed p-5 text-center transition-all duration-150 {isDragging
						? 'scale-[1.005] border-primary-500 bg-primary-50/40 ring-4 ring-primary-500/10'
						: 'border-neutral-300 bg-white hover:border-primary-500 hover:bg-primary-50/10'}"
				>
					<input
						type="file"
						accept="image/jpeg,image/png,image/webp"
						multiple
						onchange={handleFileInputChange}
						class="sr-only"
						disabled={uploadingImage}
					/>
					<div class="pointer-events-none flex flex-col items-center justify-center">
						<svg
							class="mb-1.5 h-8 w-8 {isDragging
								? 'scale-110 text-primary-600'
								: 'text-neutral-400'} transition-all"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M12 16.5V9.75m0 0l3 3m-3-3l-3 3M6.75 19.5a4.5 4.5 0 01-1.41-8.775 5.25 5.25 0 0110.233-2.33 3 3 0 013.758 3.848A3.752 3.752 0 0118 19.5H6.75z"
							/>
						</svg>
						<span
							class="text-xs font-semibold {isDragging ? 'text-primary-700' : 'text-neutral-900'}"
						>
							{isDragging
								? 'Lepaskan berkas foto di sini...'
								: 'Klik atau seret foto ke sini untuk mengunggah'}
						</span>
						<span class="mt-0.5 text-[11px] text-neutral-400">
							Mendukung format JPG, PNG, atau WebP (Maksimal 5 MB per berkas)
						</span>
					</div>
				</label>

				<!-- Daftar Foto yang Diunggah -->
				{#if isEditing}
					{#if loadingImages}
						<div class="flex items-center justify-center p-4 text-xs text-neutral-400">
							<svg
								class="mr-2 h-4 w-4 animate-spin text-neutral-400"
								fill="none"
								viewBox="0 0 24 24"
							>
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
							Memuat foto produk...
						</div>
					{:else if productImages.length > 0}
						<div class="space-y-2">
							<div
								class="flex items-center gap-1.5 rounded-lg border border-neutral-200/60 bg-neutral-100/60 px-3 py-1.5 text-[11px] text-neutral-600"
							>
								<!-- Heroicons InformationCircle 14x14 -->
								<svg
									class="h-3.5 w-3.5 shrink-0 text-neutral-400"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z"
									/>
								</svg>
								<span
									>Seret kartu foto untuk mengatur urutan. Foto paling kiri otomatis menjadi <strong
										>Foto Utama</strong
									>.</span
								>
							</div>

							<div
								class="grid grid-cols-2 gap-3 sm:grid-cols-4 md:grid-cols-6"
								role="list"
								aria-label="Daftar foto produk"
							>
								{#each productImages as img, idx (img.id)}
									<div
										role="listitem"
										draggable="true"
										ondragstart={(e) => handleCardDragStart(e, idx, 'editing')}
										ondragover={(e) => handleCardDragOver(e, idx, 'editing')}
										ondragenter={(e) => handleCardDragOver(e, idx, 'editing')}
										ondragleave={handleCardDragLeave}
										ondrop={(e) => handleCardDrop(e, idx, 'editing')}
										ondragend={handleCardDragEnd}
										class="group relative flex cursor-grab flex-col overflow-hidden rounded-xl border bg-white shadow-2xs transition-all duration-150 active:cursor-grabbing {img.is_primary ||
										idx === 0
											? 'border-emerald-500 ring-2 ring-emerald-500/20'
											: 'border-neutral-200'} {draggedPhotoIndex === idx &&
										dragTargetList === 'editing'
											? 'scale-95 border-dashed border-primary-400 opacity-40'
											: ''} {dragOverPhotoIndex === idx &&
										dragTargetList === 'editing' &&
										draggedPhotoIndex !== idx
											? 'scale-[1.03] border-primary-600 ring-4 ring-primary-500/25'
											: ''}"
									>
										<div class="relative aspect-square w-full overflow-hidden bg-neutral-100">
											<img
												src={getImageUrl(img.url)}
												alt="Foto produk"
												class="pointer-events-none h-full w-full object-cover transition-transform select-none group-hover:scale-105"
											/>
											<!-- Grip Handle Indicator -->
											<div
												class="absolute top-1.5 right-1.5 flex items-center justify-center rounded-md bg-black/55 p-1 text-white opacity-80 backdrop-blur-xs transition-opacity group-hover:opacity-100"
												title="Seret untuk memindahkan urutan"
											>
												<svg
													class="h-3.5 w-3.5"
													fill="none"
													viewBox="0 0 24 24"
													stroke="currentColor"
													stroke-width="2.5"
												>
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														d="M3.75 9h16.5m-16.5 6h16.5"
													/>
												</svg>
											</div>

											{#if img.is_primary || idx === 0}
												<div class="absolute top-1.5 left-1.5">
													<Badge variant="success" size="sm">Utama</Badge>
												</div>
											{/if}
										</div>
										<div class="flex items-center justify-between p-2">
											{#if img.is_primary || idx === 0}
												<span class="text-[11px] font-semibold text-emerald-700">Foto Utama</span>
											{:else}
												<button
													type="button"
													onclick={(e) => {
														e.stopPropagation();
														handleSetPrimaryImage(img.id);
													}}
													class="text-[11px] font-medium text-neutral-600 transition-colors hover:text-primary-600"
												>
													Jadikan Utama
												</button>
											{/if}
											<button
												type="button"
												onclick={(e) => {
													e.stopPropagation();
													handleDeleteImage(img.id);
												}}
												class="rounded-md p-1 text-neutral-400 transition-colors hover:bg-rose-50 hover:text-rose-600"
												title="Hapus foto"
											>
												<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="1.5"
														d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"
													/>
												</svg>
											</button>
										</div>
									</div>
								{/each}
							</div>
						</div>
					{/if}
				{:else}
					{#if pendingFiles.length > 0}
						<div class="space-y-2">
							<div
								class="flex items-center gap-1.5 rounded-lg border border-neutral-200/60 bg-neutral-100/60 px-3 py-1.5 text-[11px] text-neutral-600"
							>
								<!-- Heroicons InformationCircle 14x14 -->
								<svg
									class="h-3.5 w-3.5 shrink-0 text-neutral-400"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
									stroke-width="2"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z"
									/>
								</svg>
								<span
									>Seret kartu foto untuk mengatur urutan. Foto paling kiri otomatis menjadi <strong
										>Foto Utama</strong
									>.</span
								>
							</div>

							<div
								class="grid grid-cols-2 gap-3 sm:grid-cols-4 md:grid-cols-6"
								role="list"
								aria-label="Daftar foto produk"
							>
								{#each pendingFiles as item, idx (item.previewUrl)}
									<div
										role="listitem"
										draggable="true"
										ondragstart={(e) => handleCardDragStart(e, idx, 'pending')}
										ondragover={(e) => handleCardDragOver(e, idx, 'pending')}
										ondragenter={(e) => handleCardDragOver(e, idx, 'pending')}
										ondragleave={handleCardDragLeave}
										ondrop={(e) => handleCardDrop(e, idx, 'pending')}
										ondragend={handleCardDragEnd}
										class="group relative flex cursor-grab flex-col overflow-hidden rounded-xl border bg-white shadow-2xs transition-all duration-150 active:cursor-grabbing {idx ===
										0
											? 'border-indigo-500 ring-2 ring-indigo-500/20'
											: 'border-neutral-200'} {draggedPhotoIndex === idx &&
										dragTargetList === 'pending'
											? 'scale-95 border-dashed border-primary-400 opacity-40'
											: ''} {dragOverPhotoIndex === idx &&
										dragTargetList === 'pending' &&
										draggedPhotoIndex !== idx
											? 'scale-[1.03] border-primary-600 ring-4 ring-primary-500/25'
											: ''}"
									>
										<div class="relative aspect-square w-full overflow-hidden bg-neutral-100">
											<img
												src={item.previewUrl}
												alt="Pratinjau foto"
												class="pointer-events-none h-full w-full object-cover select-none"
											/>
											<!-- Grip Handle Indicator -->
											<div
												class="absolute top-1.5 right-1.5 flex items-center justify-center rounded-md bg-black/55 p-1 text-white opacity-80 backdrop-blur-xs transition-opacity group-hover:opacity-100"
												title="Seret untuk memindahkan urutan"
											>
												<svg
													class="h-3.5 w-3.5"
													fill="none"
													viewBox="0 0 24 24"
													stroke="currentColor"
													stroke-width="2.5"
												>
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														d="M3.75 9h16.5m-16.5 6h16.5"
													/>
												</svg>
											</div>

											{#if idx === 0}
												<div class="absolute top-1.5 left-1.5">
													<Badge variant="indigo" size="sm">Utama</Badge>
												</div>
											{/if}
										</div>
										<div class="flex items-center justify-between p-2">
											{#if idx === 0}
												<span class="text-[11px] font-semibold text-indigo-700">Foto Utama</span>
											{:else}
												<span class="text-[11px] text-neutral-400">Foto #{idx + 1}</span>
											{/if}
											<button
												type="button"
												onclick={(e) => {
													e.stopPropagation();
													handleRemovePendingFile(idx);
												}}
												class="rounded-md p-1 text-neutral-400 transition-colors hover:bg-rose-50 hover:text-rose-600"
												title="Batalkan foto"
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
										</div>
									</div>
								{/each}
							</div>
						</div>
					{/if}
				{/if}
			</div>

			<!-- Deskripsi Lengkap Produk (WYSIWYG Editor Melintang 2 Kolom) -->
			<div class="lg:col-span-2">
				<RichTextEditor
					id="form-description"
					label="Deskripsi Lengkap Produk (Storefront / E-Commerce)"
					bind:value={formDescription}
					placeholder="Tuliskan spesifikasi lengkap, keunggulan fitur, atau ketentuan garansi produk..."
					minHeight="160px"
				/>
			</div>
		</div>
	</form>

	{#snippet footer()}
		<Button variant="secondary" type="button" onclick={closeFormModal} disabled={formSubmitting}>
			Batal
		</Button>
		<Button variant="primary" type="submit" form="product-form" loading={formSubmitting}>
			{isEditing ? 'Simpan Perubahan' : 'Tambah Produk'}
		</Button>
	{/snippet}
</Modal>

<!-- Modal Ganti Status Produk -->
<Modal
	open={showStatusModal}
	title="Ubah Status Publikasi Produk"
	size="md"
	onclose={() => (showStatusModal = false)}
>
	<div class="space-y-4">
		{#if statusError}
			<Alert variant="error" title="Gagal">{statusError}</Alert>
		{/if}

		<p class="text-xs text-neutral-600">
			Pilih status operasional baru untuk produk <strong class="text-neutral-900"
				>{statusTargetProduct?.name}</strong
			>
			(SKU: <span class="font-mono">{statusTargetProduct?.sku}</span>):
		</p>

		<div>
			<Select2 options={statusOptions} bind:value={newStatusSelection} clearable={false} />
		</div>

		<div class="rounded-lg bg-neutral-50 p-3 text-[11px] text-neutral-500">
			<p>
				<strong>Catatan:</strong> Produk berstatus <em>Draft</em> atau <em>Diarsipkan</em> tidak akan
				muncul di menu kasir (POS) maupun katalog toko online storefront.
			</p>
		</div>
	</div>

	{#snippet footer()}
		<Button
			variant="secondary"
			onclick={() => (showStatusModal = false)}
			disabled={statusSubmitting}
		>
			Batal
		</Button>
		<Button variant="primary" onclick={handleStatusSubmit} loading={statusSubmitting}>
			Simpan Status
		</Button>
	{/snippet}
</Modal>

<!-- Modal Detail Informasi Produk yang Luas -->
<Modal bind:open={showDetailModal} title="Detail Informasi Produk" size="4xl">
	{#if detailProduct}
		<div class="space-y-6">
			<!-- Header Ringkas Produk -->
			<div
				class="flex flex-col gap-3 rounded-2xl border border-neutral-200/80 bg-neutral-50/60 p-4 sm:flex-row sm:items-center sm:justify-between"
			>
				<div class="flex items-center gap-3">
					<Badge variant="indigo" size="md">
						<span class="font-mono font-semibold tracking-tight">{detailProduct.sku}</span>
					</Badge>
					<div>
						<h3 class="text-base font-bold text-neutral-900">{detailProduct.name}</h3>
						<p class="text-xs text-neutral-500">
							Kategori: <span class="font-semibold text-neutral-800"
								>{categoryMap[detailProduct.category_id] ?? 'Kategori Lain'}</span
							>
							{#if detailProduct.brand}
								<span class="mx-1.5">•</span>
								Merek: <span class="font-semibold text-neutral-800">{detailProduct.brand}</span>
							{/if}
						</p>
					</div>
				</div>

				<div class="flex items-center gap-2">
					{#if detailProduct.status === 'active'}
						<Badge variant="success" size="md">Aktif</Badge>
					{:else if detailProduct.status === 'draft'}
						<Badge variant="warning" size="md">Draft</Badge>
					{:else}
						<Badge variant="default" size="md">Diarsipkan</Badge>
					{/if}

					{#if detailProduct.flag_serial_tracking}
						<Badge variant="purple" size="md">Serial / IMEI Unit</Badge>
					{:else}
						<Badge variant="default" size="md">Biasa (Qty)</Badge>
					{/if}
				</div>
			</div>

			<!-- Grid 2 Kolom Luas -->
			<div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
				<!-- Kolom Kiri: Galeri Foto, Finansial & Deskripsi (lg:col-span-7) -->
				<div class="space-y-5 lg:col-span-7">
					<!-- Galeri Foto Produk -->
					<div
						class="overflow-hidden rounded-2xl border border-neutral-200 bg-white p-3 shadow-2xs"
					>
						<div
							class="flex h-56 w-full items-center justify-center overflow-hidden rounded-xl border border-neutral-100 bg-neutral-50 sm:h-64"
						>
							{#if detailActiveImage}
								<img
									src={detailActiveImage}
									alt={detailProduct.name}
									class="h-full w-full object-contain p-2"
								/>
							{:else}
								<div class="flex flex-col items-center justify-center text-neutral-300">
									<svg class="h-12 w-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="1.5"
											d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z"
										/>
									</svg>
									<span class="mt-1 text-xs text-neutral-400">Belum ada foto produk</span>
								</div>
							{/if}
						</div>

						<!-- Thumbnail Galeri Jika > 1 Foto -->
						{#if detailImages.length > 1}
							<div class="mt-3 flex items-center gap-2 overflow-x-auto pb-1">
								{#each detailImages as img (img.id)}
									{@const imgUrl = getImageUrl(img.url)}
									<button
										type="button"
										onclick={() => (detailActiveImage = imgUrl)}
										class="relative h-14 w-14 shrink-0 overflow-hidden rounded-lg border-2 transition-all {detailActiveImage ===
										imgUrl
											? 'border-primary-600 ring-2 ring-primary-500/20'
											: 'border-neutral-200 opacity-70 hover:opacity-100'}"
									>
										<img src={imgUrl} alt="Thumbnail" class="h-full w-full object-cover" />
										{#if img.is_primary}
											<span
												class="absolute inset-x-0 bottom-0 bg-primary-600 py-0.5 text-center text-[9px] leading-none font-bold text-white"
												>Utama</span
											>
										{/if}
									</button>
								{/each}
							</div>
						{/if}
					</div>

					<!-- Ringkasan Finansial & Harga Pokok/Jual -->
					<div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
						<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
							<span class="text-[11px] font-medium text-neutral-500">Harga Pokok (HPP)</span>
							{#if detailProduct.purchase_price !== null && detailProduct.purchase_price !== undefined}
								<p class="mt-1 text-base font-bold text-neutral-800">
									{formatRupiah(detailProduct.purchase_price)}
								</p>
							{:else}
								<div class="mt-1 flex items-center gap-1.5 text-neutral-400">
									<svg
										class="h-3.5 w-3.5 shrink-0 text-neutral-400"
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
								<span class="text-[9.5px] text-neutral-400">Tersensor</span>
							{/if}
						</div>

						<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
							<div class="flex items-center justify-between">
								<span class="text-[11px] font-medium text-neutral-500">Harga Jual</span>
								{#if detailProduct.is_ppn}
									<span
										class="rounded border border-cyan-200 bg-cyan-50 px-1 text-[9px] font-bold text-cyan-700"
										>PPN</span
									>
								{/if}
							</div>
							<p class="mt-1 text-base font-bold text-primary-700">
								{formatRupiah(detailProduct.selling_price)}
							</p>
						</div>

						<div
							class="col-span-2 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs sm:col-span-1"
						>
							<span class="text-[11px] font-medium text-neutral-500">Estimasi Margin</span>
							{#if detailProduct.purchase_price !== null && detailProduct.purchase_price !== undefined}
								<p
									class="mt-1 text-base font-bold {detailProduct.selling_price >=
									detailProduct.purchase_price
										? 'text-emerald-700'
										: 'text-rose-700'}"
								>
									{detailProduct.selling_price >= detailProduct.purchase_price
										? '+'
										: ''}{formatRupiah(detailProduct.selling_price - detailProduct.purchase_price)}
									<span class="text-xs font-normal text-neutral-400">
										({detailProduct.purchase_price > 0
											? (
													((detailProduct.selling_price - detailProduct.purchase_price) /
														detailProduct.purchase_price) *
													100
												).toFixed(1)
											: '0'}%)
									</span>
								</p>
							{:else}
								<p class="mt-1 font-mono text-xs text-neutral-400">Terkunci</p>
								<span class="text-[9.5px] text-neutral-400">Butuh Izin PBAC</span>
							{/if}
						</div>
					</div>

					<!-- Detail Fisik (Satuan, Berat) -->
					<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
						<span class="text-[11px] font-bold tracking-wider text-neutral-400 uppercase"
							>Informasi Kemasan & Logistik</span
						>
						<div class="mt-2 grid grid-cols-2 gap-3 text-xs sm:grid-cols-3">
							<div>
								<span class="text-neutral-500">Satuan:</span>
								<span class="ml-1 font-semibold text-neutral-900"
									>{detailProduct.unit || 'UNIT'}</span
								>
							</div>
							<div>
								<span class="text-neutral-500">Berat Fisik:</span>
								<span class="ml-1 font-semibold text-neutral-900"
									>{detailProduct.weight_gram
										? `${(detailProduct.weight_gram / 1000).toLocaleString('id-ID')} Kg`
										: '-'}</span
								>
							</div>
							<div>
								<span class="text-neutral-500">Diperbarui:</span>
								<span class="ml-1 font-medium text-neutral-700"
									>{formatDate(detailProduct.updated_at)}</span
								>
							</div>
						</div>
					</div>

					<!-- Atribut Varian Dinamis Jika Ada -->
					{#if detailProduct.atribut_varian && typeof detailProduct.atribut_varian === 'object' && Object.keys(detailProduct.atribut_varian).length > 0}
						<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
							<span class="text-[11px] font-bold tracking-wider text-neutral-400 uppercase"
								>Atribut Varian Produk</span
							>
							<div class="mt-2 flex flex-wrap gap-2">
								{#each Object.entries(detailProduct.atribut_varian) as [k, v] (k)}
									<div
										class="flex items-center rounded-lg border border-neutral-200 bg-neutral-50 px-2.5 py-1 text-xs"
									>
										<span class="font-medium text-neutral-500">{k}:</span>
										<span class="ml-1.5 font-bold text-neutral-800">{String(v)}</span>
									</div>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Deskripsi Produk Berformat -->
					<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs">
						<div class="mb-3 flex items-center justify-between border-b border-neutral-100 pb-2">
							<span class="text-xs font-bold tracking-wider text-neutral-700 uppercase"
								>Deskripsi Spesifikasi</span
							>
							<span class="text-[11px] text-neutral-400">Format WYSIWYG</span>
						</div>
						{#if detailProduct.description}
							<div class="prose prose-sm max-h-56 max-w-none overflow-y-auto pr-2 text-neutral-700">
								<!-- eslint-disable-next-line svelte/no-at-html-tags -->
								{@html detailProduct.description}
							</div>
						{:else}
							<p class="text-xs text-neutral-400 italic">
								Tidak ada deskripsi rinci untuk produk ini.
							</p>
						{/if}
					</div>
				</div>

				<!-- Kolom Kanan: Daftar Barcode-Barcode yang Sudah Dimasukkan (lg:col-span-5) -->
				<div class="space-y-4 lg:col-span-5">
					<div class="flex items-center justify-between">
						<div>
							<h4 class="text-sm font-bold text-neutral-900">Daftar Barcode Terdaftar</h4>
							<p class="text-[11px] text-neutral-500">Kode barcode aktif produk ini</p>
						</div>
						<Badge variant="purple" size="sm">{detailBarcodes.length} Barcode</Badge>
					</div>

					{#if loadingDetail}
						<div
							class="flex flex-col items-center justify-center rounded-xl border border-neutral-200 bg-white p-8"
						>
							<svg class="h-6 w-6 animate-spin text-neutral-400" fill="none" viewBox="0 0 24 24">
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
							<span class="mt-2 text-xs text-neutral-500">Memuat barcode...</span>
						</div>
					{:else if detailBarcodes.length === 0}
						<div
							class="flex flex-col items-center justify-center rounded-2xl border-2 border-dashed border-neutral-200 bg-neutral-50/50 p-6 text-center"
						>
							<div
								class="flex h-10 w-10 items-center justify-center rounded-xl bg-neutral-100 text-neutral-400"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="1.5"
										d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z"
									/>
								</svg>
							</div>
							<p class="mt-3 text-xs font-semibold text-neutral-800">Belum Ada Barcode</p>
							<p class="mt-1 max-w-xs text-[11px] text-neutral-500">
								Produk ini belum memiliki nomor barcode yang didaftarkan untuk scanner kasir.
							</p>
							<div class="mt-3">
								<Button
									variant="outline"
									size="sm"
									onclick={() => {
										showDetailModal = false;
										goto(`/master/barcodes?product_id=${detailProduct?.id}`);
									}}
								>
									+ Daftarkan Barcode Sekarang
								</Button>
							</div>
						</div>
					{:else}
						<div class="max-h-[500px] space-y-3 overflow-y-auto pr-1">
							{#each detailBarcodes as bc (bc.id)}
								<div
									class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs transition-all hover:border-neutral-300"
								>
									<div class="mb-2 flex items-center justify-between">
										{#if bc.is_primary}
											<Badge variant="indigo" size="sm">Utama</Badge>
										{:else}
											<Badge variant="default" size="sm">Barcode</Badge>
										{/if}
										<div class="flex items-center gap-1.5">
											<button
												type="button"
												onclick={() => copyBarcode(bc.barcode, bc.id)}
												class="rounded-md p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700"
												title="Salin kode barcode"
											>
												{#if copiedBarcodeId === bc.id}
													<span class="text-[10px] font-semibold text-emerald-600">Disalin!</span>
												{:else}
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
															d="M15.666 3.849A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.599m7.332 0c.055.194.084.4.084.615v.999a2.25 2.25 0 01-2.25 2.25H9.75a2.25 2.25 0 01-2.25-2.25V4.464c0-.214.03-.42.084-.615m7.332 0a2.25 2.25 0 012.25 2.25v13.5a2.25 2.25 0 01-2.25 2.25H6.75a2.25 2.25 0 01-2.25-2.25V6.75a2.25 2.25 0 012.25-2.25"
														/>
													</svg>
												{/if}
											</button>
										</div>
									</div>

									<!-- Visual Render Barcode SVG Asli -->
									<div
										class="flex flex-col items-center justify-center rounded-lg border border-neutral-100 bg-neutral-50/70 p-2"
									>
										<Barcode value={bc.barcode} height={38} showText={false} class="max-w-full" />
										<span class="mt-1 font-mono text-xs font-bold tracking-widest text-neutral-800"
											>{bc.barcode}</span
										>
									</div>

									<div class="mt-2 flex items-center justify-between text-[11px] text-neutral-400">
										<span>Didaftarkan: {formatDate(bc.created_at)}</span>
										<button
											type="button"
											onclick={() => {
												showDetailModal = false;
												goto(`/master/barcodes?product_id=${detailProduct?.id}`);
											}}
											class="font-medium text-primary-600 transition-colors hover:text-primary-800"
										>
											Cetak Label
										</button>
									</div>
								</div>
							{/each}

							<div class="pt-1">
								<Button
									variant="outline"
									size="sm"
									fullWidth
									onclick={() => {
										showDetailModal = false;
										goto(`/master/barcodes?product_id=${detailProduct?.id}`);
									}}
								>
									Buka Pengelola Barcode Lengkap
								</Button>
							</div>
						</div>
					{/if}
				</div>
			</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex w-full items-center justify-between">
			<Button variant="secondary" size="sm" onclick={() => (showDetailModal = false)}>Tutup</Button>

			<div class="flex items-center gap-2">
				{#if detailProduct}
					<Button
						variant="outline"
						size="sm"
						onclick={() => {
							const id = detailProduct?.id;
							showDetailModal = false;
							if (id) goto(`/master/barcodes?product_id=${id}`);
						}}
					>
						<svg class="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="1.75"
								d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z"
							/>
						</svg>
						Kelola Barcode
					</Button>

					<Button
						variant="primary"
						size="sm"
						onclick={() => {
							const target = detailProduct;
							showDetailModal = false;
							if (target) openEditModal(target);
						}}
					>
						<svg class="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="1.75"
								d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10"
							/>
						</svg>
						Ubah Data Produk
					</Button>
				{/if}
			</div>
		</div>
	{/snippet}
</Modal>
