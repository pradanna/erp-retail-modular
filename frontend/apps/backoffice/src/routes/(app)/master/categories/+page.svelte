<script lang="ts">
	/**
	 * Halaman Manajemen Kategori Produk (Data Master).
	 *
	 * Fitur:
	 * - Tab 1: Daftar Tabel (Tampilan tabel tabular dengan pencarian & ringkasan statistik)
	 * - Tab 2: Hierarki Tree (Tampilan pohon visual interaktif dengan indikator turunan level 1, 2, dst)
	 * - Filter pencarian nama kategori seketika (SearchInput)
	 * - Modal Tambah & Ubah Kategori (Select induk, Input nama, URL gambar)
	 * - Tombol instan "+ Sub" langsung dari node pohon untuk menambahkan anak kategori
	 * - Modal Konfirmasi Hapus Kategori
	 * - Umpan balik notifikasi Alert sukses & error
	 */
	import { onMount } from 'svelte';
	import { SvelteMap, SvelteSet } from 'svelte/reactivity';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listCategories,
		createCategory,
		updateCategory,
		deleteCategory,
		uploadCategoryImage
	} from '@erp/api-client';
	import type { CategoryResponse } from '@erp/types';
	import { ApiError } from '@erp/types';
	import {
		Button,
		Input,
		Select2,
		type Select2Option,
		Table,
		Modal,
		Alert,
		Badge,
		SearchInput,
		ActionMenu,
		toast
	} from '@erp/ui';

	let categories = $state<CategoryResponse[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Tab aktif: 'table' atau 'tree' (default: 'tree')
	let activeTab = $state<'table' | 'tree'>('tree');

	// Filter & Pencarian
	let searchQuery = $state('');
	let selectedHierarchyFilter = $state<string>('all');

	const hierarchyFilterOptions: Select2Option[] = [
		{ value: 'all', label: 'Semua Hierarki' },
		{
			value: 'root',
			label: 'Hanya Kategori Utama',
			subtext: 'Kategori tingkat atas / tanpa induk'
		},
		{ value: 'sub', label: 'Hanya Sub-Kategori', subtext: 'Memiliki kategori induk' }
	];

	// State ekspansi node pohon (ID -> boolean)
	let expandedNodes = $state<Record<string, boolean>>({});

	// Tipe struktur data node hierarki kategori
	interface CategoryTreeNode {
		category: CategoryResponse;
		children: CategoryTreeNode[];
		level: number;
		totalDescendants: number;
	}

	// Modal Tambah / Edit
	let showFormModal = $state(false);
	let isEditing = $state(false);
	let editingId = $state<string | null>(null);
	let formName = $state('');
	let formParentId = $state<string>('');
	let formImageUrl = $state('');
	let formSubmitting = $state(false);
	let formError = $state<string | null>(null);
	let uploadingImage = $state(false);
	let isDragging = $state(false);
	let uploadError = $state<string | null>(null);
	let fileInputElement = $state<HTMLInputElement | null>(null);
	let showManualUrlInput = $state(false);

	function getImageUrl(url?: string | null): string {
		if (!url) return '';
		if (url.startsWith('http')) return url;
		return `http://localhost:8088${url}`;
	}

	async function handleUploadFile(file: File) {
		uploadError = null;
		const token = getToken();
		if (!token) {
			uploadError = 'Sesi telah berakhir, silakan masuk kembali';
			return;
		}

		const validTypes = ['image/jpeg', 'image/png', 'image/webp'];
		if (!validTypes.includes(file.type)) {
			uploadError = 'Format berkas tidak didukung (gunakan JPG, PNG, atau WebP)';
			return;
		}

		if (file.size > 5 * 1024 * 1024) {
			uploadError = 'Ukuran berkas melebihi batas maksimal 5 MB';
			return;
		}

		uploadingImage = true;
		try {
			const res = await uploadCategoryImage(token, file);
			formImageUrl = res.url;
			toast.success('Gambar kategori berhasil diunggah');
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				uploadError = err.message;
			} else if (err instanceof Error) {
				uploadError = err.message;
			} else {
				uploadError = 'Gagal mengunggah gambar kategori';
			}
		} finally {
			uploadingImage = false;
		}
	}

	function handleFileChange(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files[0]) {
			handleUploadFile(target.files[0]);
		}
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
		handleUploadFile(dt.files[0]);
	}

	function removeImage() {
		formImageUrl = '';
		if (fileInputElement) {
			fileInputElement.value = '';
		}
	}

	// Modal Hapus
	let showDeleteModal = $state(false);
	let deletingCategory = $state<CategoryResponse | null>(null);
	let deleteLoading = $state(false);
	let deleteError = $state<string | null>(null);

	async function loadCategories() {
		const token = getToken();
		if (!token) return;

		loading = true;
		error = null;
		try {
			const data = await listCategories(token);
			categories = data;

			// Buka semua node secara default pada muatan awal
			const exp: Record<string, boolean> = {};
			for (const cat of data) {
				exp[cat.id] = true;
			}
			expandedNodes = exp;
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat data kategori';
			}
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadCategories();
	});

	// Filter pencarian dan hierarki untuk tabel
	const filteredCategories = $derived(
		categories.filter((cat) => {
			const matchesSearch = cat.name.toLowerCase().includes(searchQuery.toLowerCase().trim());
			const matchesHierarchy =
				selectedHierarchyFilter === 'all' ||
				(selectedHierarchyFilter === 'root' && !cat.parent_id) ||
				(selectedHierarchyFilter === 'sub' && !!cat.parent_id);
			return matchesSearch && matchesHierarchy;
		})
	);

	// Statistik Ringkasan
	const totalCategories = $derived(categories.length);
	const parentCategoriesCount = $derived(categories.filter((c) => !c.parent_id).length);
	const subCategoriesCount = $derived(categories.filter((c) => !!c.parent_id).length);

	// Map ID kategori ke nama untuk display kolom Parent
	const categoryMap = $derived(new SvelteMap(categories.map((c) => [c.id, c.name])));

	// Membangun pohon hierarki lengkap dari daftar kategori
	const categoryTree = $derived.by((): CategoryTreeNode[] => {
		const parentMap = new SvelteMap<string, CategoryResponse[]>();
		const allIds = new SvelteSet(categories.map((c) => c.id));
		const roots: CategoryResponse[] = [];

		for (const cat of categories) {
			if (!cat.parent_id || !allIds.has(cat.parent_id)) {
				roots.push(cat);
			} else {
				const list = parentMap.get(cat.parent_id) || [];
				list.push(cat);
				parentMap.set(cat.parent_id, list);
			}
		}

		roots.sort((a, b) => a.name.localeCompare(b.name, 'id'));

		function buildNode(
			category: CategoryResponse,
			level: number,
			visited: SvelteSet<string>
		): CategoryTreeNode {
			const nextVisited = new SvelteSet(visited);
			nextVisited.add(category.id);

			const rawChildren = (parentMap.get(category.id) || []).filter((c) => !nextVisited.has(c.id));
			rawChildren.sort((a, b) => a.name.localeCompare(b.name, 'id'));

			const children = rawChildren.map((c) => buildNode(c, level + 1, nextVisited));
			const totalDescendants = children.reduce((acc, curr) => acc + 1 + curr.totalDescendants, 0);

			return {
				category,
				children,
				level,
				totalDescendants
			};
		}

		return roots.map((r) => buildNode(r, 1, new SvelteSet()));
	});

	// Tree yang disaring berdasarkan query pencarian (termasuk leluhur / turunan yang cocok)
	const filteredCategoryTree = $derived.by((): CategoryTreeNode[] => {
		const q = searchQuery.toLowerCase().trim();
		if (!q) return categoryTree;

		function filterNode(node: CategoryTreeNode): CategoryTreeNode | null {
			const matchesSelf = node.category.name.toLowerCase().includes(q);
			const filteredChildren: CategoryTreeNode[] = [];

			for (const child of node.children) {
				const res = filterNode(child);
				if (res) filteredChildren.push(res);
			}

			if (matchesSelf || filteredChildren.length > 0) {
				return {
					...node,
					children: filteredChildren
				};
			}
			return null;
		}

		const result: CategoryTreeNode[] = [];
		for (const root of categoryTree) {
			const res = filterNode(root);
			if (res) result.push(res);
		}
		return result;
	});

	// Menghitung kedalaman tingkat pohon maksimal
	const maxTreeDepth = $derived.by((): number => {
		if (categoryTree.length === 0) return 0;
		function getDepth(node: CategoryTreeNode): number {
			if (node.children.length === 0) return node.level;
			return Math.max(...node.children.map(getDepth));
		}
		return Math.max(...categoryTree.map(getDepth));
	});

	// Opsi dropdown kategori induk untuk Select2
	const parentOptions = $derived((): Select2Option[] => {
		const opts: Select2Option[] = [
			{ value: '', label: 'Tidak ada (Jadikan Kategori Utama)', subtext: 'Tingkat Kategori Utama' }
		];
		for (const cat of categories) {
			// Hindari cyclic parent jika sedang mengedit diri sendiri
			if (!isEditing || cat.id !== editingId) {
				opts.push({
					value: cat.id,
					label: cat.name,
					subtext:
						cat.parent_id && categoryMap.get(cat.parent_id)
							? `Induk: ${categoryMap.get(cat.parent_id)}`
							: undefined
				});
			}
		}
		return opts;
	});

	function isExpanded(id: string): boolean {
		return expandedNodes[id] ?? true;
	}

	function toggleExpand(id: string) {
		expandedNodes[id] = !isExpanded(id);
	}

	function expandAll() {
		const next: Record<string, boolean> = {};
		for (const cat of categories) {
			next[cat.id] = true;
		}
		expandedNodes = next;
	}

	function collapseAll() {
		const next: Record<string, boolean> = {};
		for (const cat of categories) {
			next[cat.id] = false;
		}
		expandedNodes = next;
	}

	function openAddModal() {
		isEditing = false;
		editingId = null;
		formName = '';
		formParentId = '';
		formImageUrl = '';
		formError = null;
		uploadError = null;
		uploadingImage = false;
		isDragging = false;
		showManualUrlInput = false;
		if (fileInputElement) fileInputElement.value = '';
		showFormModal = true;
	}

	function openAddSubModal(parentCategory: CategoryResponse) {
		isEditing = false;
		editingId = null;
		formName = '';
		formParentId = parentCategory.id;
		formImageUrl = '';
		formError = null;
		uploadError = null;
		uploadingImage = false;
		isDragging = false;
		showManualUrlInput = false;
		if (fileInputElement) fileInputElement.value = '';
		showFormModal = true;
	}

	function openEditModal(category: CategoryResponse) {
		isEditing = true;
		editingId = category.id;
		formName = category.name;
		formParentId = category.parent_id ?? '';
		formImageUrl = category.image_url ?? '';
		formError = null;
		uploadError = null;
		uploadingImage = false;
		isDragging = false;
		showManualUrlInput = false;
		if (fileInputElement) fileInputElement.value = '';
		showFormModal = true;
	}

	function openDeleteModal(category: CategoryResponse) {
		deletingCategory = category;
		deleteError = null;
		showDeleteModal = true;
	}

	async function handleSubmitForm(e?: Event) {
		e?.preventDefault();
		formError = null;

		if (!formName.trim()) {
			formError = 'Nama kategori wajib diisi';
			return;
		}

		const token = getToken();
		if (!token) return;

		formSubmitting = true;
		try {
			if (isEditing && editingId) {
				await updateCategory(token, editingId, {
					name: formName.trim(),
					parent_id: formParentId || undefined,
					image_url: formImageUrl.trim() || undefined
				});
				toast.success(`Kategori "${formName.trim()}" berhasil diperbarui`);
			} else {
				await createCategory(token, {
					name: formName.trim(),
					parent_id: formParentId || undefined,
					image_url: formImageUrl.trim() || undefined
				});
				toast.success(`Kategori baru "${formName.trim()}" berhasil ditambahkan`);
			}

			showFormModal = false;
			await loadCategories();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				formError = err.message;
			} else if (err instanceof Error) {
				formError = err.message;
			} else {
				formError = 'Terjadi kesalahan saat menyimpan kategori';
			}
		} finally {
			formSubmitting = false;
		}
	}

	async function handleConfirmDelete() {
		if (!deletingCategory) return;

		const token = getToken();
		if (!token) return;

		deleteLoading = true;
		deleteError = null;
		try {
			await deleteCategory(token, deletingCategory.id);
			toast.success(`Kategori "${deletingCategory.name}" berhasil dihapus`);
			showDeleteModal = false;
			deletingCategory = null;
			await loadCategories();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				deleteError = err.message;
			} else if (err instanceof Error) {
				deleteError = err.message;
			} else {
				deleteError = 'Gagal menghapus kategori';
			}
		} finally {
			deleteLoading = false;
		}
	}

	function formatDate(dateStr: string): string {
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
</script>

<svelte:head>
	<title>Kategori Produk — Data Master Gen-E</title>
</svelte:head>

<!-- Snippet untuk Highlight Kata Kunci Pencarian -->
{#snippet highlightText(text: string, query: string)}
	{#if !query.trim()}
		{text}
	{:else}
		{@const q = query.toLowerCase().trim()}
		{@const idx = text.toLowerCase().indexOf(q)}
		{#if idx >= 0}
			{text.slice(0, idx)}<mark class="rounded bg-amber-100 px-0.5 font-bold text-amber-900"
				>{text.slice(idx, idx + q.length)}</mark
			>{text.slice(idx + q.length)}
		{:else}
			{text}
		{/if}
	{/if}
{/snippet}

<!-- Snippet Rekursif Node Pohon Hierarki -->
{#snippet treeNode(node: CategoryTreeNode)}
	<div class="group/node relative">
		<!-- Kartu Node Kategori -->
		<div
			class="flex items-center justify-between rounded-xl border border-neutral-200/90 bg-white p-3 shadow-2xs transition-all hover:border-neutral-300 hover:shadow-xs sm:p-3.5 {node.level ===
			1
				? 'border-l-4 border-l-primary-600 bg-white'
				: 'border-l border-neutral-200/90 bg-white'}"
		>
			<div class="flex min-w-0 items-center gap-3">
				<!-- Tombol Expand / Collapse -->
				{#if node.children.length > 0}
					<button
						type="button"
						onclick={() => toggleExpand(node.category.id)}
						class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-neutral-200 text-neutral-500 transition-all hover:bg-neutral-100 hover:text-neutral-900 focus:outline-hidden"
						title={isExpanded(node.category.id) ? 'Tutup sub-kategori' : 'Buka sub-kategori'}
					>
						<!-- Heroicons ChevronRight -->
						<svg
							class="h-4 w-4 transition-transform duration-200 {isExpanded(node.category.id)
								? 'rotate-90'
								: ''}"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path d="M8.25 4.5l7.5 7.5-7.5 7.5" />
						</svg>
					</button>
				{:else}
					<div class="flex h-7 w-7 shrink-0 items-center justify-center text-neutral-300">
						<span class="h-1.5 w-1.5 rounded-full bg-neutral-300"></span>
					</div>
				{/if}

				<!-- Ikon Kategori / Gambar Thumbnail -->
				{#if node.category.image_url}
					<img
						src={getImageUrl(node.category.image_url)}
						alt={node.category.name}
						class="h-8 w-8 shrink-0 rounded-lg border border-neutral-200 object-cover shadow-2xs"
					/>
				{:else}
					<div
						class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg {node.level === 1
							? 'bg-primary-50 text-primary-700'
							: 'bg-neutral-100 text-neutral-600'}"
					>
						{#if node.level === 1}
							<!-- Squares2X2 untuk Kategori Utama -->
							<svg
								class="h-4 w-4"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<path
									d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z"
								/>
							</svg>
						{:else}
							<!-- Folder untuk Sub-kategori -->
							<svg
								class="h-4 w-4"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<path
									d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z"
								/>
							</svg>
						{/if}
					</div>
				{/if}

				<!-- Nama Kategori & Lencana Badge -->
				<div class="min-w-0">
					<div class="flex flex-wrap items-center gap-2">
						<span class="text-sm font-semibold text-neutral-900">
							{@render highlightText(node.category.name, searchQuery)}
						</span>

						{#if node.level === 1}
							<Badge variant="info" size="sm">Kategori Utama</Badge>
						{:else}
							<Badge variant="default" size="sm">Level {node.level}</Badge>
						{/if}

						{#if node.children.length > 0}
							<Badge variant="purple" size="sm">{node.children.length} Sub-kategori</Badge>
						{/if}
					</div>

					<div class="mt-0.5 flex items-center gap-2 text-[11px] text-neutral-400">
						{#if node.category.parent_id && categoryMap.has(node.category.parent_id)}
							<span>
								Induk: <span class="font-medium text-neutral-600"
									>{categoryMap.get(node.category.parent_id)}</span
								>
							</span>
							<span>•</span>
						{/if}
						<span class="font-mono text-neutral-400">ID: {node.category.id.slice(0, 8)}...</span>
					</div>
				</div>
			</div>

			<!-- Tombol Aksi Cepat Node (Titik Tiga Dropdown) -->
			<div class="ml-2 flex shrink-0 items-center">
				<ActionMenu
					items={[
						{
							label: 'Tambah Sub-Kategori',
							iconSvg: 'M12 4.5v15m7.5-7.5h-15',
							onclick: () => openAddSubModal(node.category)
						},
						{
							label: 'Ubah Kategori',
							iconSvg:
								'M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10',
							onclick: () => openEditModal(node.category)
						},
						{
							label: 'Hapus Kategori',
							variant: 'danger',
							divider: true,
							iconSvg:
								'M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0',
							onclick: () => openDeleteModal(node.category)
						}
					]}
				/>
			</div>
		</div>

		<!-- Cabang Turunan Anak (Nested Children Branch) -->
		{#if node.children.length > 0 && isExpanded(node.category.id)}
			<div
				class="relative mt-2.5 ml-4 space-y-2.5 border-l-2 border-neutral-200 pl-4 sm:ml-6 sm:pl-6"
			>
				{#each node.children as child (child.category.id)}
					{@render treeNode(child)}
				{/each}
			</div>
		{/if}
	</div>
{/snippet}

<div class="space-y-6">
	<!-- Page Header -->
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
		<div>
			<h2 class="text-xl font-bold tracking-tight text-neutral-900 sm:text-2xl">Kategori Produk</h2>
			<p class="text-xs text-neutral-500 sm:text-sm">
				Kelola struktur dan hierarki kategori katalog barang toko retail Anda.
			</p>
		</div>

		<Button variant="primary" onclick={openAddModal}>
			<!-- Heroicons Plus 20x20 -->
			<svg
				class="h-4 w-4"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<path d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			<span>Tambah Kategori</span>
		</Button>
	</div>

	<!-- Notifikasi Umpan Balik -->
	{#if error}
		<Alert variant="error" dismissible title="Terjadi Kesalahan">
			{error}
		</Alert>
	{/if}

	<!-- Statistik Ringkas Kategori (3 Cards dengan Icon Heroicons) -->
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
		<!-- Total Kategori -->
		<div class="rounded-xl border border-neutral-200/80 bg-white p-4 shadow-xs">
			<div class="flex items-center justify-between">
				<span class="text-xs font-medium tracking-wider text-neutral-500 uppercase">Total Kategori</span>
				<span class="rounded-lg bg-neutral-100 p-2 text-neutral-700">
					<!-- Heroicons Squares2X2 20x20 -->
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z"
						/>
					</svg>
				</span>
			</div>
			<div class="mt-3 flex items-baseline gap-2">
				<span class="text-2xl font-bold text-neutral-900">{totalCategories}</span>
				<span class="text-xs text-neutral-400">kategori terdaftar</span>
			</div>
		</div>

		<!-- Kategori Utama -->
		<div class="rounded-xl border border-neutral-200/80 bg-white p-4 shadow-xs">
			<div class="flex items-center justify-between">
				<span class="text-xs font-medium tracking-wider text-sky-600 uppercase">Kategori Utama</span>
				<span class="rounded-lg bg-sky-50 p-2 text-sky-600">
					<!-- Heroicons Folder 20x20 -->
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z"
						/>
					</svg>
				</span>
			</div>
			<div class="mt-3 flex items-baseline gap-2">
				<span class="text-2xl font-bold text-sky-700">{parentCategoriesCount}</span>
				<span class="text-xs text-neutral-400">kategori tingkat utama</span>
			</div>
		</div>

		<!-- Sub-Kategori -->
		<div class="rounded-xl border border-neutral-200/80 bg-white p-4 shadow-xs">
			<div class="flex items-center justify-between">
				<span class="text-xs font-medium tracking-wider text-purple-600 uppercase">Sub-Kategori</span>
				<span class="rounded-lg bg-purple-50 p-2 text-purple-600">
					<!-- Heroicons ListBullet 20x20 -->
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="1.75"
							d="M8.25 6.75h12M8.25 12h12m-12 5.25h12M3.75 6.75h.007v.008H3.75V6.75zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zM3.75 12h.007v.008H3.75V12zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm-.375 5.25h.007v.008H3.75v-.008zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z"
						/>
					</svg>
				</span>
			</div>
			<div class="mt-3 flex items-baseline gap-2">
				<span class="text-2xl font-bold text-purple-700">{subCategoriesCount}</span>
				<span class="text-xs text-neutral-400">bagian turunan</span>
			</div>
		</div>
	</div>

	<!-- Tab Switcher (Daftar Tabel vs Hierarki Tree) -->
	<div class="border-b border-neutral-200">
		<nav class="-mb-px flex space-x-8" aria-label="Tabs Kategori">
			<button
				type="button"
				onclick={() => (activeTab = 'table')}
				class="inline-flex items-center gap-2 border-b-2 py-3.5 text-xs font-semibold whitespace-nowrap transition-colors {activeTab ===
				'table'
					? 'border-primary-600 text-primary-600'
					: 'border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700'}"
			>
				<!-- Heroicons TableCells 18x18 -->
				<svg
					class="h-4 w-4"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path
						d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z"
					/>
				</svg>
				<span>Daftar Tabel ({totalCategories})</span>
			</button>

			<button
				type="button"
				onclick={() => (activeTab = 'tree')}
				class="inline-flex items-center gap-2 border-b-2 py-3.5 text-xs font-semibold whitespace-nowrap transition-colors {activeTab ===
				'tree'
					? 'border-primary-600 text-primary-600'
					: 'border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700'}"
			>
				<!-- Heroicons Bars3BottomLeft / FolderTree 18x18 -->
				<svg
					class="h-4 w-4"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path d="M3.75 6.75h16.5M3.75 12h10.5m-10.5 5.25h16.5" />
				</svg>
				<span>Hierarki Tree ({parentCategoriesCount} Utama)</span>
			</button>
		</nav>
	</div>

	<!-- TAB 1: DAFTAR TABEL -->
	{#if activeTab === 'table'}

		<!-- Bar Filter & Pencarian (1-Row Flex Sejajar) -->
		<div
			class="flex flex-col gap-3 rounded-xl border border-neutral-200/80 bg-white p-4 shadow-xs md:flex-row md:items-center md:justify-between"
		>
			<div class="flex flex-1 flex-wrap items-center gap-3">
				<div class="w-full sm:w-72">
					<SearchInput
						bind:value={searchQuery}
						placeholder="Cari nama kategori..."
						debounceMs={200}
						class="w-full max-w-none"
					/>
				</div>

				<!-- Filter Hierarki via Select2 -->
				<div class="w-full sm:w-64">
					<Select2
						options={hierarchyFilterOptions}
						bind:value={selectedHierarchyFilter}
						placeholder="Semua Hierarki"
						clearable={false}
					/>
				</div>
			</div>

			{#if searchQuery || selectedHierarchyFilter !== 'all'}
				<p class="text-xs whitespace-nowrap text-neutral-500">
					Ditemukan <span class="font-semibold text-neutral-800">{filteredCategories.length}</span>
					dari {totalCategories} kategori
				</p>
			{/if}
		</div>

		<!-- Tabel Kategori -->
		<Table
			empty={!loading && filteredCategories.length === 0}
			emptyTitle="Belum Ada Kategori"
			emptyMessage={searchQuery
				? 'Tidak ada kategori yang cocok dengan pencarian Anda.'
				: 'Silakan klik tombol Tambah Kategori untuk membuat kategori produk pertama.'}
			{loading}
		>
			<thead>
				<tr
					class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-bold tracking-wider text-neutral-500 uppercase"
				>
					<th class="px-4 py-3">Nama Kategori</th>
					<th class="px-4 py-3">Tipe / Hierarki</th>
					<th class="px-4 py-3">Kategori Induk</th>
					<th class="hidden px-4 py-3 md:table-cell">ID Kategori</th>
					<th class="hidden px-4 py-3 sm:table-cell">Diperbarui</th>
					<th class="px-4 py-3 text-right">Aksi</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-100">
				{#each filteredCategories as category (category.id)}
					<tr class="transition-colors hover:bg-neutral-50/70">
						<!-- Nama Kategori -->
						<td class="px-4 py-3">
							<div class="flex items-center gap-2.5">
								{#if category.image_url}
									<img
										src={getImageUrl(category.image_url)}
										alt={category.name}
										class="h-8 w-8 shrink-0 rounded-lg border border-neutral-200 object-cover shadow-2xs"
									/>
								{:else}
									<div
										class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-info-50 text-info-600"
									>
										{#if category.parent_id}
											<!-- Sub-category icon (Heroicons Folder) -->
											<svg
												class="h-4 w-4"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												stroke-width="2"
												stroke-linecap="round"
												stroke-linejoin="round"
											>
												<path
													d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z"
												/>
											</svg>
										{:else}
											<!-- Parent category icon (Heroicons Squares2X2) -->
											<svg
												class="h-4 w-4"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												stroke-width="2"
												stroke-linecap="round"
												stroke-linejoin="round"
											>
												<path
													d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z"
												/>
											</svg>
										{/if}
									</div>
								{/if}
								<div>
									<span class="font-semibold text-neutral-900">{category.name}</span>
								</div>
							</div>
						</td>

						<!-- Tipe / Hierarki -->
						<td class="px-4 py-3">
							{#if category.parent_id}
								<Badge variant="default" size="sm">Sub-Kategori</Badge>
							{:else}
								<Badge variant="info" size="sm">Kategori Utama</Badge>
							{/if}
						</td>

						<!-- Kategori Induk -->
						<td class="px-4 py-3 text-neutral-600">
							{#if category.parent_id}
								<span class="font-medium text-neutral-800">
									{categoryMap.get(category.parent_id) || category.parent_id}
								</span>
							{:else}
								<span class="text-neutral-400 italic">-</span>
							{/if}
						</td>

						<!-- ID Monospace -->
						<td class="hidden px-4 py-3 font-mono text-[11px] text-neutral-400 md:table-cell">
							{category.id}
						</td>

						<!-- Tanggal -->
						<td class="hidden px-4 py-3 text-neutral-500 sm:table-cell">
							{formatDate(category.updated_at)}
						</td>

						<!-- Aksi (Titik Tiga Dropdown) -->
						<td class="px-4 py-3 text-right whitespace-nowrap">
							<div class="flex items-center justify-end">
								<ActionMenu
									items={[
										{
											label: 'Tambah Sub-Kategori',
											iconSvg: 'M12 4.5v15m7.5-7.5h-15',
											onclick: () => openAddSubModal(category)
										},
										{
											label: 'Ubah Kategori',
											iconSvg:
												'M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10',
											onclick: () => openEditModal(category)
										},
										{
											label: 'Hapus Kategori',
											variant: 'danger',
											divider: true,
											iconSvg:
												'M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0',
											onclick: () => openDeleteModal(category)
										}
									]}
								/>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}

	<!-- TAB 2: HIERARKI TREE -->
	{#if activeTab === 'tree'}

		<!-- Toolbar Kontrol Pohon & Pencarian -->
		<div
			class="flex flex-col justify-between gap-3 rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs sm:flex-row sm:items-center"
		>
			<div class="w-full sm:w-80">
				<SearchInput
					bind:value={searchQuery}
					placeholder="Saring atau cari kategori dalam pohon..."
					debounceMs={150}
				/>
			</div>

			<div class="flex flex-wrap items-center gap-2">
				<Button variant="secondary" size="sm" onclick={expandAll}>
					<!-- Heroicons ArrowsPointingOut 16x16 -->
					<svg
						class="h-3.5 w-3.5"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path
							d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15"
						/>
					</svg>
					<span>Buka Semua</span>
				</Button>

				<Button variant="secondary" size="sm" onclick={collapseAll}>
					<!-- Heroicons ArrowsPointingIn 16x16 -->
					<svg
						class="h-3.5 w-3.5"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path
							d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5l5.25 5.25"
						/>
					</svg>
					<span>Tutup Semua</span>
				</Button>

				<Button variant="primary" size="sm" onclick={openAddModal}>
					<!-- Heroicons Plus 16x16 -->
					<svg
						class="h-3.5 w-3.5"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					<span>Kategori Utama</span>
				</Button>
			</div>
		</div>

		<!-- Kontainer Pohon Hierarki Visual -->
		{#if loading}
			<div
				class="flex flex-col items-center justify-center rounded-2xl border border-neutral-200 bg-white p-12 text-center shadow-2xs"
			>
				<svg class="h-8 w-8 animate-spin text-neutral-400" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
					></circle>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					></path>
				</svg>
				<p class="mt-3 text-xs font-medium text-neutral-500">Menyusun struktur pohon hierarki...</p>
			</div>
		{:else if filteredCategoryTree.length === 0}
			<div
				class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-neutral-300 bg-white p-12 text-center shadow-2xs"
			>
				<div
					class="flex h-12 w-12 items-center justify-center rounded-xl bg-neutral-100 text-neutral-400"
				>
					<svg
						class="h-6 w-6"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="1.5"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path d="M3.75 6.75h16.5M3.75 12h10.5m-10.5 5.25h16.5" />
					</svg>
				</div>
				<h3 class="mt-4 text-sm font-semibold text-neutral-900">
					{searchQuery ? 'Kategori Tidak Ditemukan' : 'Belum Ada Kategori'}
				</h3>
				<p class="mt-1 max-w-sm text-xs text-neutral-500">
					{searchQuery
						? 'Tidak ada kategori dalam struktur hierarki yang cocok dengan kata kunci pencarian Anda.'
						: 'Mulai buat kategori utama untuk menyusun pohon pengelompokan produk retail Anda.'}
				</p>
				{#if !searchQuery}
					<div class="mt-4">
						<Button variant="primary" size="sm" onclick={openAddModal}>
							Tambah Kategori Pertama
						</Button>
					</div>
				{/if}
			</div>
		{:else}
			<div
				class="space-y-3 rounded-2xl border border-neutral-200 bg-neutral-50/40 p-4 shadow-2xs sm:p-6"
			>
				<div class="flex items-center justify-between border-b border-neutral-200/80 pb-2">
					<div class="flex items-center gap-2 text-xs text-neutral-500">
						<span class="inline-block h-2 w-2 rounded-full bg-primary-600"></span>
						<span>Garis biru menandai <strong>Kategori Utama</strong></span>
						<span class="mx-1">•</span>
						<span>Garis vertikal menghubungkan <strong>Sub-Kategori</strong></span>
					</div>
					{#if searchQuery}
						<span class="text-xs text-neutral-500">
							Menampilkan <strong class="text-neutral-800">{filteredCategoryTree.length}</strong> pohon kategori utama
						</span>
					{/if}
				</div>

				<div class="space-y-3 pt-1">
					{#each filteredCategoryTree as rootNode (rootNode.category.id)}
						{@render treeNode(rootNode)}
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- Modal Form Tambah / Edit Kategori -->
<Modal
	bind:open={showFormModal}
	title={isEditing
		? 'Ubah Kategori Produk'
		: formParentId
			? 'Tambah Sub-Kategori Produk'
			: 'Tambah Kategori Produk Baru'}
	size="lg"
>
	<form onsubmit={handleSubmitForm} class="space-y-4.5" id="category-form">
		{#if formError}
			<Alert variant="error" dismissible>{formError}</Alert>
		{/if}

		<!-- Penanda Visual jika menambah Sub-Kategori langsung dari pohon -->
		{#if formParentId && !isEditing}
			<div
				class="flex items-center gap-2 rounded-lg border border-primary-200 bg-primary-50/70 p-3 text-xs text-primary-900"
			>
				<svg
					class="h-4 w-4 shrink-0 text-primary-600"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
				</svg>
				<span>
					Membuat sub-kategori di bawah: <strong
						>{categoryMap.get(formParentId) || formParentId}</strong
					>
				</span>
			</div>
		{/if}

		<Input
			id="category-name-input"
			label="Nama Kategori"
			placeholder="Contoh: Smartphone, Laptop Gaming, Aksesoris"
			bind:value={formName}
			required
			showRequiredAsterisk={true}
		/>

		<Select2
			id="category-parent-select"
			label="Kategori Induk (Parent)"
			options={parentOptions()}
			bind:value={formParentId}
			searchPlaceholder="Cari kategori induk..."
			clearable={false}
		/>

		<!-- Unggah Gambar / Ikon Kategori -->
		<div class="space-y-1.5">
			<span class="block text-xs font-medium text-neutral-700">
				Gambar / Ikon Kategori <span class="font-normal text-neutral-400">(Opsional)</span>
			</span>

			{#if uploadError}
				<Alert variant="error" dismissible>{uploadError}</Alert>
			{/if}

			{#if formImageUrl}
				<!-- Kartu Pratinjau Gambar yang Terpasang -->
				<div
					class="flex items-center gap-3.5 rounded-xl border border-neutral-200 bg-neutral-50/70 p-3"
				>
					<img
						src={getImageUrl(formImageUrl)}
						alt="Pratinjau Kategori"
						class="h-16 w-16 shrink-0 rounded-lg border border-neutral-200 bg-white object-cover shadow-2xs"
					/>
					<div class="min-w-0 flex-1">
						<p class="truncate text-xs font-semibold text-neutral-900">
							{formImageUrl.split('/').pop() || 'Gambar Kategori'}
						</p>
						<p class="mt-0.5 truncate text-[11px] text-neutral-400">
							{formImageUrl}
						</p>
						<div class="mt-2 flex items-center gap-2">
							<Button
								variant="secondary"
								size="sm"
								onclick={() => fileInputElement?.click()}
								disabled={uploadingImage}
							>
								{uploadingImage ? 'Mengunggah...' : 'Ganti Gambar'}
							</Button>
							<Button variant="danger" size="sm" onclick={removeImage} disabled={uploadingImage}>
								Hapus
							</Button>
						</div>
					</div>
				</div>
			{:else}
				<!-- Dropzone Unggah Gambar Baru -->
				<div
					role="button"
					tabindex="0"
					onclick={() => fileInputElement?.click()}
					onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && fileInputElement?.click()}
					ondragover={handleDragOver}
					ondragenter={handleDragOver}
					ondragleave={handleDragLeave}
					ondrop={handleDrop}
					class="flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed p-6 text-center transition-all focus:ring-2 focus:ring-primary-500/20 focus:outline-hidden {isDragging
						? 'border-primary-500 bg-primary-50/40 ring-4 ring-primary-500/10 scale-[1.005]'
						: 'border-neutral-300 bg-neutral-50/50 hover:border-neutral-400 hover:bg-neutral-50'}"
				>
					{#if uploadingImage}
						<svg class="h-8 w-8 animate-spin text-primary-600" fill="none" viewBox="0 0 24 24">
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
						<p class="mt-2 text-xs font-medium text-neutral-700">Mengunggah gambar ke server...</p>
					{:else}
						<div class="pointer-events-none flex flex-col items-center justify-center">
							<div
								class="flex h-10 w-10 items-center justify-center rounded-xl border border-neutral-200 bg-white {isDragging
									? 'text-primary-600 scale-110'
									: 'text-neutral-500'} shadow-2xs transition-all"
							>
								<!-- Heroicons Photo 20x20 -->
								<svg
									class="h-5 w-5"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="1.75"
									stroke-linecap="round"
									stroke-linejoin="round"
								>
									<path
										d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z"
									/>
								</svg>
							</div>
							<p class="mt-2.5 text-xs font-semibold {isDragging ? 'text-primary-700' : 'text-neutral-800'}">
								{isDragging ? 'Lepaskan berkas gambar di sini...' : 'Klik atau seret gambar ke sini untuk mengunggah'}
							</p>
							<p class="mt-0.5 text-[11px] text-neutral-400">
								Format didukung: JPG, PNG, WebP (Maksimal 5 MB)
							</p>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Input Berkas Tersembunyi -->
			<input
				type="file"
				bind:this={fileInputElement}
				onchange={handleFileChange}
				accept=".jpg,.jpeg,.png,.webp"
				class="hidden"
			/>

			<!-- Toggle Opsi Masukkan URL Manual -->
			<div class="pt-1">
				<button
					type="button"
					onclick={() => (showManualUrlInput = !showManualUrlInput)}
					class="text-[11px] font-medium text-neutral-500 transition-colors hover:text-neutral-800"
				>
					{showManualUrlInput
						? 'Sembunyikan input URL manual'
						: 'Atau gunakan tautan URL eksternal'}
				</button>
				{#if showManualUrlInput}
					<div class="mt-2">
						<Input
							id="category-image-url-input"
							placeholder="https://example.com/kategori.jpg"
							bind:value={formImageUrl}
						/>
					</div>
				{/if}
			</div>
		</div>
	</form>

	{#snippet footer()}
		<Button
			variant="secondary"
			size="sm"
			onclick={() => (showFormModal = false)}
			disabled={formSubmitting}
		>
			Batal
		</Button>
		<Button variant="primary" size="sm" loading={formSubmitting} onclick={handleSubmitForm}>
			{isEditing ? 'Simpan Perubahan' : 'Buat Kategori'}
		</Button>
	{/snippet}
</Modal>

<!-- Modal Konfirmasi Hapus Kategori -->
<Modal bind:open={showDeleteModal} title="Konfirmasi Hapus Kategori" size="sm">
	<div class="space-y-3">
		{#if deleteError}
			<Alert variant="error" dismissible>{deleteError}</Alert>
		{/if}

		<p class="text-xs leading-relaxed text-neutral-600">
			Apakah Anda yakin ingin menghapus kategori
			<span class="font-bold text-neutral-900">{deletingCategory?.name}</span>? Tindakan ini tidak
			dapat dibatalkan.
		</p>
	</div>

	{#snippet footer()}
		<Button
			variant="secondary"
			size="sm"
			onclick={() => (showDeleteModal = false)}
			disabled={deleteLoading}
		>
			Batal
		</Button>
		<Button variant="danger" size="sm" loading={deleteLoading} onclick={handleConfirmDelete}>
			Hapus Kategori
		</Button>
	{/snippet}
</Modal>
