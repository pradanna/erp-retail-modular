<script lang="ts">
	/**
	 * Halaman Manajemen Garansi Produk — Data Master (Gen-E Enterprise).
	 *
	 * Fitur:
	 * - Tab 1: Master Template Kebijakan Garansi (Garansi Toko & Garansi Resmi Pabrik)
	 * - Tab 2: Penetapan Garansi ke Katalog Produk (Enforcing aturan: maks 1 toko & 1 pabrik aktif)
	 * - Modal Buat Template Garansi Baru
	 * - Modal Pasang Garansi ke Produk Terpilih
	 * - Nonaktifkan garansi produk
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listWarrantyPolicies,
		createWarrantyPolicy,
		listProducts,
		getProductWarranties,
		assignProductWarranty,
		deactivateProductWarranty
	} from '@erp/api-client';
	import type {
		WarrantyPolicyResponse,
		ProductResponse,
		ProductWarrantyResponse,
		WarrantyType,
		CreateWarrantyPolicyRequest
	} from '@erp/types';
	import { WARRANTY_TYPE_LABELS, ApiError } from '@erp/types';
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
		type ActionMenuItem,
		toast
	} from '@erp/ui';

	// Tab aktif
	let activeTab = $state<'policies' | 'assignment'>('policies');

	// State Master Kebijakan
	let policies = $state<WarrantyPolicyResponse[]>([]);
	let loadingPolicies = $state(false);
	let policyError = $state<string | null>(null);

	// Filter & Pencarian Tab 1 (Template Kebijakan)
	let policySearchQuery = $state('');
	let selectedWarrantyTypeFilter = $state<string>('all');

	const warrantyTypeFilterOptions: Select2Option[] = [
		{ value: 'all', label: 'Semua Tipe Garansi' },
		{ value: 'toko', label: 'Garansi Toko', subtext: 'Servis / replace toko' },
		{ value: 'pabrik', label: 'Garansi Resmi Pabrik', subtext: 'Service center resmi' }
	];

	// Filter kebijakan garansi
	const filteredPolicies = $derived(
		policies.filter((p) => {
			const matchesSearch =
				!policySearchQuery.trim() ||
				p.name.toLowerCase().includes(policySearchQuery.toLowerCase().trim()) ||
				(p.coverage && p.coverage.toLowerCase().includes(policySearchQuery.toLowerCase().trim()));
			const matchesType =
				selectedWarrantyTypeFilter === 'all' || p.type === selectedWarrantyTypeFilter;
			return matchesSearch && matchesType;
		})
	);

	// State Modal Buat Kebijakan
	let showPolicyModal = $state(false);
	let formPolicyName = $state('');
	let formPolicyType = $state<WarrantyType>('toko');
	let formDurationMonths = $state(12);
	let formDurationDays = $state(0);
	let formCoverage = $state('');
	let formInstructions = $state('');
	let policySubmitting = $state(false);
	let formPolicyError = $state<string | null>(null);

	// State Penetapan Garansi Produk
	let products = $state<ProductResponse[]>([]);
	let selectedProductId = $state('');
	let assignedWarranties = $state<ProductWarrantyResponse[]>([]);
	let loadingAssigned = $state(false);
	let productSearchQuery = $state('');

	// Modal Pasang Garansi
	let showAssignModal = $state(false);
	let assignPolicyId = $state('');
	let assignSubmitting = $state(false);
	let assignError = $state<string | null>(null);

	// Opsi tipe garansi untuk Select2
	const warrantyTypeOptions: Select2Option[] = [
		{ value: 'toko', label: 'Garansi Toko', subtext: 'Replace / Servis Toko' },
		{ value: 'pabrik', label: 'Garansi Resmi Pabrik', subtext: 'Service Center Resmi' }
	];

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

	// Opsi kebijakan yang aktif untuk modal assign (Select2)
	const activePolicyOptions = $derived<Select2Option[]>(
		policies
			.filter((p) => p.is_active)
			.map((p) => ({
				value: p.id,
				label: p.name,
				subtext: `[${WARRANTY_TYPE_LABELS[p.type]}] Masa aktif: ${p.duration_months} Bulan`
			}))
	);

	const currentProduct = $derived(products.find((p) => p.id === selectedProductId) ?? null);

	async function loadPolicies() {
		const token = getToken();
		if (!token) return;

		loadingPolicies = true;
		policyError = null;
		try {
			policies = await listWarrantyPolicies(token);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				policyError = err.message;
			} else {
				policyError = 'Gagal memuat master template garansi';
			}
		} finally {
			loadingPolicies = false;
		}
	}

	async function loadProductsForAssignment() {
		const token = getToken();
		if (!token) return;

		try {
			const res = await listProducts(token, { limit: 100 });
			products = res.data ?? [];
			if (products.length > 0 && !selectedProductId) {
				selectedProductId = products[0].id;
				await loadProductWarrantiesData(products[0].id);
			}
		} catch (err: unknown) {
			console.error('Gagal memuat list produk:', err);
		}
	}

	async function loadProductWarrantiesData(productId: string) {
		if (!productId) {
			assignedWarranties = [];
			return;
		}

		const token = getToken();
		if (!token) return;

		loadingAssigned = true;
		try {
			assignedWarranties = await getProductWarranties(token, productId);
		} catch (err: unknown) {
			console.error('Gagal memuat garansi produk:', err);
		} finally {
			loadingAssigned = false;
		}
	}

	async function selectProduct(productId: string) {
		if (selectedProductId === productId) return;
		selectedProductId = productId;
		await loadProductWarrantiesData(productId);
	}

	function openCreatePolicyModal() {
		formPolicyName = '';
		formPolicyType = 'toko';
		formDurationMonths = 12;
		formDurationDays = 0;
		formCoverage = '';
		formInstructions = '';
		formPolicyError = null;
		showPolicyModal = true;
	}

	async function handleSubmitPolicy(e: SubmitEvent) {
		e.preventDefault();
		const token = getToken();
		if (!token) return;

		if (!formPolicyName.trim()) {
			formPolicyError = 'Nama template garansi wajib diisi';
			return;
		}

		if (formDurationMonths <= 0 && formDurationDays <= 0) {
			formPolicyError = 'Durasi garansi harus lebih dari 0 hari / bulan';
			return;
		}

		policySubmitting = true;
		formPolicyError = null;
		try {
			const payload: CreateWarrantyPolicyRequest = {
				name: formPolicyName.trim(),
				type: formPolicyType,
				duration_months: formDurationMonths,
				duration_days: formDurationDays,
				coverage: formCoverage.trim() || undefined,
				claim_instructions: formInstructions.trim() || undefined
			};

			await createWarrantyPolicy(token, payload);
			toast.success(`Kebijakan garansi "${formPolicyName.trim()}" berhasil dibuat.`);
			showPolicyModal = false;
			await loadPolicies();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				formPolicyError = err.message;
			} else {
				formPolicyError = 'Gagal menyimpan template garansi';
			}
		} finally {
			policySubmitting = false;
		}
	}

	function openAssignModal() {
		if (!selectedProductId) return;
		assignPolicyId = '';
		assignError = null;
		showAssignModal = true;
	}

	async function handleSubmitAssign(e: SubmitEvent) {
		e.preventDefault();
		const token = getToken();
		if (!token || !selectedProductId) return;

		if (!assignPolicyId) {
			assignError = 'Pilih salah satu template kebijakan garansi';
			return;
		}

		assignSubmitting = true;
		assignError = null;
		try {
			await assignProductWarranty(token, selectedProductId, {
				warranty_policy_id: assignPolicyId
			});

			toast.success('Kebijakan garansi berhasil dipasangkan ke produk.');
			showAssignModal = false;
			await loadProductWarrantiesData(selectedProductId);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				assignError = err.message;
			} else {
				assignError = 'Gagal memasang garansi ke produk';
			}
		} finally {
			assignSubmitting = false;
		}
	}

	async function handleDeactivateWarranty(pw: ProductWarrantyResponse) {
		const token = getToken();
		if (!token || !selectedProductId) return;

		try {
			await deactivateProductWarranty(token, selectedProductId, pw.id);
			toast.success('Garansi produk berhasil dinonaktifkan.');
			await loadProductWarrantiesData(selectedProductId);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				policyError = err.message;
			} else {
				policyError = 'Gagal menonaktifkan garansi';
			}
		}
	}

	onMount(async () => {
		await Promise.all([loadPolicies(), loadProductsForAssignment()]);
	});
</script>

<svelte:head>
	<title>Garansi Produk — Data Master Gen-E</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header Halaman -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-neutral-900">Garansi Produk</h1>
			<p class="mt-1 text-sm text-neutral-500">
				Kelola template kebijakan purna jual (Garansi Toko & Garansi Resmi Pabrik) serta
				penetapannya.
			</p>
		</div>

		<div>
			{#if activeTab === 'policies'}
				<Button variant="primary" onclick={openCreatePolicyModal}>
					<svg
						class="mr-2 h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					Buat Template Garansi
				</Button>
			{/if}
		</div>
	</div>

	<!-- Notifikasi Global -->
	{#if policyError}
		<Alert variant="error" dismissible title="Terjadi Kesalahan">
			{policyError}
		</Alert>
	{/if}

	<!-- Tab Switcher -->
	<div class="border-b border-neutral-200">
		<nav class="-mb-px flex space-x-8" aria-label="Tabs Garansi">
			<button
				type="button"
				onclick={() => (activeTab = 'policies')}
				class="border-b-2 py-3.5 text-xs font-semibold whitespace-nowrap transition-colors {activeTab ===
				'policies'
					? 'border-primary-600 text-primary-600'
					: 'border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700'}"
			>
				Template Kebijakan ({policies.length})
			</button>

			<button
				type="button"
				onclick={() => (activeTab = 'assignment')}
				class="border-b-2 py-3.5 text-xs font-semibold whitespace-nowrap transition-colors {activeTab ===
				'assignment'
					? 'border-primary-600 text-primary-600'
					: 'border-transparent text-neutral-500 hover:border-neutral-300 hover:text-neutral-700'}"
			>
				Penetapan ke Produk
			</button>
		</nav>
	</div>

	<!-- Konten Tab 1: Template Kebijakan -->
	{#if activeTab === 'policies'}
		<!-- Bar Filter & Pencarian Template Garansi via Select2 -->
		<div class="mb-4 flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
			<div class="flex flex-wrap items-center gap-3">
				<SearchInput
					bind:value={policySearchQuery}
					placeholder="Cari nama template garansi atau cakupan..."
					debounceMs={200}
				/>

				<div class="w-60">
					<Select2
						options={warrantyTypeFilterOptions}
						bind:value={selectedWarrantyTypeFilter}
						placeholder="Semua Tipe Garansi"
						clearable={false}
					/>
				</div>
			</div>

			{#if policySearchQuery || selectedWarrantyTypeFilter !== 'all'}
				<p class="text-xs text-neutral-500">
					Ditemukan <span class="font-semibold text-neutral-800">{filteredPolicies.length}</span>
					dari {policies.length} template
				</p>
			{/if}
		</div>

		<Table
			empty={!loadingPolicies && filteredPolicies.length === 0}
			emptyTitle="Belum Ada Template Garansi"
			emptyMessage={policySearchQuery || selectedWarrantyTypeFilter !== 'all'
				? 'Tidak ada template garansi yang cocok dengan filter Anda.'
				: 'Buat template garansi toko (misal: Garansi Tukar Baru 7 Hari) atau garansi pabrik (1 Tahun).'}
			loading={loadingPolicies}
		>
			<thead>
				<tr
					class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-bold tracking-wider text-neutral-500 uppercase"
				>
					<th class="px-4 py-3 text-left">Nama Kebijakan</th>
					<th class="px-4 py-3 text-left">Tipe Garansi</th>
					<th class="px-4 py-3 text-left">Masa Berlaku</th>
					<th class="px-4 py-3 text-left">Cakupan & Panduan Klaim</th>
					<th class="px-4 py-3 text-center">Status</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-100">
				{#each filteredPolicies as pol (pol.id)}
					<tr class="transition-colors hover:bg-neutral-50/70">
						<td class="px-4 py-3 font-medium whitespace-nowrap text-neutral-900">
							{pol.name}
						</td>
						<td class="px-4 py-3 whitespace-nowrap">
							<Badge variant={pol.type === 'toko' ? 'primary' : 'warning'} size="sm">
								{WARRANTY_TYPE_LABELS[pol.type]}
							</Badge>
						</td>
						<td class="px-4 py-3 text-xs whitespace-nowrap text-neutral-700">
							{#if pol.duration_months > 0}
								<strong class="text-neutral-900">{pol.duration_months}</strong> Bulan
							{/if}
							{#if pol.duration_days > 0}
								<span class="text-neutral-500">({pol.duration_days} Hari)</span>
							{/if}
						</td>
						<td class="px-4 py-3 text-xs text-neutral-600">
							<div class="max-w-md">
								{#if pol.coverage}
									<div class="line-clamp-1"><strong>Cakupan:</strong> {pol.coverage}</div>
								{/if}
								{#if pol.claim_instructions}
									<div class="line-clamp-1 text-neutral-400">
										<strong>Klaim:</strong>
										{pol.claim_instructions}
									</div>
								{/if}
							</div>
						</td>
						<td class="px-4 py-3 text-center whitespace-nowrap">
							{#if pol.is_active}
								<Badge variant="success" size="sm">Aktif</Badge>
							{:else}
								<Badge variant="default" size="sm">Non-aktif</Badge>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{:else}
		<!-- Konten Tab 2: Master-Detail Katalog Produk (Kiri) & Penetapan Garansi (Kanan) -->
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
							<div
								class="mt-2.5 rounded-lg border border-primary-100 bg-primary-50/80 p-2 text-[11px] text-primary-800"
							>
								<strong>Aturan Bisnis:</strong> Maksimal 1 garansi toko aktif dan 1 garansi resmi pabrik
								aktif per produk.
							</div>
						</div>
					{/if}
				</div>
			</div>

			<!-- Sisi Kanan: Tabel Garansi Aktif Produk Ini (Detail) -->
			<div class="lg:col-span-7">
				<div class="rounded-xl border border-neutral-200/80 bg-white shadow-xs">
					<div class="border-b border-neutral-200/80 px-5 py-4">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
							<div>
								<div class="flex items-center gap-2">
									<h2 class="text-sm font-semibold text-neutral-900">Garansi Aktif Produk Ini</h2>
									<span
										class="inline-flex items-center rounded-full bg-neutral-100 px-2 py-0.5 text-[11px] font-medium text-neutral-600"
									>
										{assignedWarranties.length} garansi terpasang
									</span>
								</div>
								{#if currentProduct}
									<p class="mt-0.5 text-xs text-neutral-500">
										Produk: <span class="font-medium text-neutral-800">{currentProduct.name}</span>
									</p>
								{/if}
							</div>

							<!-- Tombol Pasang Garansi dipindahkan ke panel ini -->
							<div>
								<Button
									variant="primary"
									size="sm"
									onclick={openAssignModal}
									disabled={!selectedProductId}
								>
									<svg
										class="mr-1.5 h-3.5 w-3.5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M12 4.5v15m7.5-7.5h-15"
										/>
									</svg>
									Pasang Garansi ke Produk
								</Button>
							</div>
						</div>
					</div>

					<Table
						empty={!loadingAssigned && assignedWarranties.length === 0}
						emptyTitle="Belum Ada Garansi yang Terpasang"
						emptyMessage={selectedProductId
							? 'Produk ini belum memiliki garansi aktif. Klik "+ Pasang Garansi ke Produk" di atas.'
							: 'Pilih salah satu produk di katalog kiri terlebih dahulu.'}
						loading={loadingAssigned}
					>
						<thead>
							<tr
								class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-bold tracking-wider text-neutral-500 uppercase"
							>
								<th class="px-4 py-3 text-left">Tipe Garansi</th>
								<th class="px-4 py-3 text-left">Template Kebijakan</th>
								<th class="px-4 py-3 text-center">Status</th>
								<th class="px-4 py-3 text-right">Aksi</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-100">
							{#each assignedWarranties as pw (pw.id)}
								<tr class="transition-colors hover:bg-neutral-50/70">
									<td class="px-4 py-3 whitespace-nowrap">
										<Badge variant={pw.type === 'toko' ? 'primary' : 'warning'} size="sm">
											{WARRANTY_TYPE_LABELS[pw.type]}
										</Badge>
									</td>
									<td class="px-4 py-3">
										<span class="text-sm font-medium text-neutral-900">
											{pw.policy?.name ?? 'Kebijakan Garansi'}
										</span>
										{#if pw.policy}
											<div class="text-xs text-neutral-500">
												Durasi: {pw.policy.duration_months} Bulan
												{#if pw.policy.duration_days > 0}
													({pw.policy.duration_days} Hari)
												{/if}
											</div>
										{/if}
									</td>
									<td class="px-4 py-3 text-center whitespace-nowrap">
										{#if pw.is_active}
											<Badge variant="success" size="sm">Aktif</Badge>
										{:else}
											<Badge variant="default" size="sm">Non-aktif</Badge>
										{/if}
									</td>
									<td class="px-4 py-3 text-right whitespace-nowrap">
										{#if pw.is_active}
											<ActionMenu
												items={[
													{
														label: 'Nonaktifkan Garansi',
														icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" /></svg>`,
														variant: 'danger',
														onClick: () => handleDeactivateWarranty(pw)
													}
												]}
											/>
										{:else}
											<span class="text-xs text-neutral-400">—</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</Table>
				</div>
			</div>
		</div>
	{/if}
</div>

<!-- Modal Buat Template Kebijakan Baru -->
<Modal
	open={showPolicyModal}
	title="Buat Template Kebijakan Garansi Baru"
	size="md"
	onclose={() => (showPolicyModal = false)}
>
	<form onsubmit={handleSubmitPolicy} class="space-y-4">
		{#if formPolicyError}
			<Alert variant="error" title="Gagal">{formPolicyError}</Alert>
		{/if}

		<div>
			<label for="policy-name" class="mb-1.5 block text-xs font-medium text-neutral-700">
				Nama Kebijakan Garansi <span class="text-rose-500">*</span>
			</label>
			<Input
				id="policy-name"
				placeholder="Contoh: Garansi Resmi TAM 1 Tahun"
				bind:value={formPolicyName}
				required
			/>
		</div>

		<div>
			<Select2
				id="policy-type"
				label="Tipe Tanggungan"
				options={warrantyTypeOptions}
				bind:value={formPolicyType}
				required
				showRequiredAsterisk={true}
				clearable={false}
			/>
		</div>

		<div class="grid grid-cols-2 gap-4">
			<div>
				<label for="policy-months" class="mb-1.5 block text-xs font-medium text-neutral-700">
					Durasi (Bulan)
				</label>
				<Input id="policy-months" type="number" placeholder="12" bind:value={formDurationMonths} />
			</div>

			<div>
				<label for="policy-days" class="mb-1.5 block text-xs font-medium text-neutral-700">
					Durasi Tambahan (Hari)
				</label>
				<Input id="policy-days" type="number" placeholder="0" bind:value={formDurationDays} />
			</div>
		</div>

		<div>
			<label for="policy-coverage" class="mb-1.5 block text-xs font-medium text-neutral-700">
				Cakupan Kerusakan yang Ditanggung
			</label>
			<textarea
				id="policy-coverage"
				rows="2"
				placeholder="Contoh: Sparepart, LCD, Mesin (Kerusakan akibat cacat pabrik)..."
				bind:value={formCoverage}
				class="w-full rounded-xl border border-neutral-300 px-3.5 py-2.5 text-xs text-neutral-900 transition-colors focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 focus:outline-hidden"
			></textarea>
		</div>

		<div>
			<label for="policy-instructions" class="mb-1.5 block text-xs font-medium text-neutral-700">
				Petunjuk Alur Klaim bagi Pelanggan
			</label>
			<textarea
				id="policy-instructions"
				rows="2"
				placeholder="Contoh: Membawa unit beserta nota pembelian ke Service Center terdekat..."
				bind:value={formInstructions}
				class="w-full rounded-xl border border-neutral-300 px-3.5 py-2.5 text-xs text-neutral-900 transition-colors focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 focus:outline-hidden"
			></textarea>
		</div>

		<div class="flex items-center justify-end gap-3 border-t border-neutral-200/80 pt-4">
			<Button
				variant="secondary"
				type="button"
				onclick={() => (showPolicyModal = false)}
				disabled={policySubmitting}
			>
				Batal
			</Button>
			<Button variant="primary" type="submit" loading={policySubmitting}>
				Simpan Template Garansi
			</Button>
		</div>
	</form>
</Modal>

<!-- Modal Pasang Garansi ke Produk -->
<Modal
	open={showAssignModal}
	title="Pasang Garansi ke Produk"
	size="md"
	onclose={() => (showAssignModal = false)}
>
	<form onsubmit={handleSubmitAssign} class="space-y-4">
		{#if assignError}
			<Alert variant="error" title="Gagal">{assignError}</Alert>
		{/if}

		<p class="text-xs text-neutral-600">
			Pasangkan template kebijakan garansi aktif ke produk <strong class="text-neutral-900"
				>{currentProduct?.name}</strong
			>:
		</p>

		<div>
			<Select2
				id="assign-warranty-policy"
				label="Template Kebijakan Garansi"
				options={activePolicyOptions}
				bind:value={assignPolicyId}
				placeholder="Pilih template kebijakan garansi aktif..."
				searchPlaceholder="Ketik nama atau tipe garansi..."
				required
				showRequiredAsterisk={true}
				clearable={true}
			/>
		</div>

		<div class="flex items-center justify-end gap-3 border-t border-neutral-200/80 pt-4">
			<Button
				variant="secondary"
				type="button"
				onclick={() => (showAssignModal = false)}
				disabled={assignSubmitting}
			>
				Batal
			</Button>
			<Button variant="primary" type="submit" loading={assignSubmitting}>Pasang Garansi</Button>
		</div>
	</form>
</Modal>
