<script lang="ts">
	/**
	 * Halaman Manajemen Lokasi & Cabang Toko (Data Master).
	 *
	 * Fitur:
	 * - Daftar cabang fisik & gudang online storefront
	 * - Filter tipe lokasi (Semua, Fisik, Online)
	 * - Filter pencarian nama/kode lokasi seketika (SearchInput)
	 * - Modal Tambah & Ubah Lokasi
	 * - Toggle status aktif/non-aktif lokasi
	 * - Modal Konfirmasi Hapus Lokasi
	 * - Umpan balik notifikasi Alert
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import {
		listLocations,
		createLocation,
		updateLocation,
		setLocationStatus,
		deleteLocation
	} from '@erp/api-client';
	import type { LocationResponse, LocationType } from '@erp/types';
	import { LOCATION_TYPE_LABELS, ApiError } from '@erp/types';
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
		MapPicker,
		ActionMenu,
		toast
	} from '@erp/ui';

	let locations = $state<LocationResponse[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Filter & Pencarian
	let searchQuery = $state('');
	let selectedTypeFilter = $state<string>('all');
	let selectedStatusFilter = $state<string>('all');

	// Modal Tambah / Edit
	let showFormModal = $state(false);
	let isEditing = $state(false);
	let editingId = $state<string | null>(null);
	let formCode = $state('');
	let formName = $state('');
	let formType = $state<LocationType>('physical');
	let formAddress = $state('');
	let formLatitude = $state<number | null>(null);
	let formLongitude = $state<number | null>(null);
	let formSubmitting = $state(false);
	let formError = $state<string | null>(null);

	// Modal Hapus
	let showDeleteModal = $state(false);
	let deletingLocation = $state<LocationResponse | null>(null);
	let deleteLoading = $state(false);
	let deleteError = $state<string | null>(null);

	// Modal Lihat Peta
	let showMapModal = $state(false);
	let viewingMapLocation = $state<LocationResponse | null>(null);

	function openMapModal(loc: LocationResponse) {
		viewingMapLocation = loc;
		showMapModal = true;
	}

	const locationTypeOptions: Select2Option[] = [
		{ value: 'physical', label: 'Toko / Cabang Fisik', subtext: 'Operasional Kasir & POS Offline' },
		{ value: 'online', label: 'Gudang Online Storefront', subtext: 'Fulfillment Pesanan Online' }
	];

	const locationTypeFilterOptions: Select2Option[] = [
		{ value: 'all', label: 'Semua Tipe Lokasi' },
		{ value: 'physical', label: 'Toko Fisik', subtext: 'Operasional Kasir & POS' },
		{ value: 'online', label: 'Gudang Online', subtext: 'Fulfillment E-Commerce' }
	];

	const locationStatusFilterOptions: Select2Option[] = [
		{ value: 'all', label: 'Semua Status' },
		{ value: 'active', label: 'Aktif', subtext: 'Dapat bertransaksi' },
		{ value: 'inactive', label: 'Nonaktif', subtext: 'Operasional ditutup' }
	];

	async function loadLocations() {
		const token = getToken();
		if (!token) return;

		loading = true;
		error = null;
		try {
			const data = await listLocations(token);
			locations = data;
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat data lokasi cabang';
			}
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadLocations();
	});

	// Filter data berdasarkan tipe, status, dan pencarian
	const filteredLocations = $derived(
		locations.filter((loc) => {
			const matchesType = selectedTypeFilter === 'all' || loc.type === selectedTypeFilter;
			const matchesStatus =
				selectedStatusFilter === 'all' ||
				(selectedStatusFilter === 'active' && loc.is_active) ||
				(selectedStatusFilter === 'inactive' && !loc.is_active);
			const q = searchQuery.toLowerCase().trim();
			const matchesSearch =
				!q ||
				loc.name.toLowerCase().includes(q) ||
				loc.code.toLowerCase().includes(q) ||
				loc.address.toLowerCase().includes(q);
			return matchesType && matchesStatus && matchesSearch;
		})
	);

	// Statistik Ringkasan
	const totalLocations = $derived(locations.length);
	const physicalCount = $derived(locations.filter((l) => l.type === 'physical').length);
	const onlineCount = $derived(locations.filter((l) => l.type === 'online').length);
	const activeCount = $derived(locations.filter((l) => l.is_active).length);

	function openAddModal() {
		isEditing = false;
		editingId = null;
		formCode = '';
		formName = '';
		formType = 'physical';
		formAddress = '';
		formLatitude = null;
		formLongitude = null;
		formError = null;
		showFormModal = true;
	}

	function openEditModal(loc: LocationResponse) {
		isEditing = true;
		editingId = loc.id;
		formCode = loc.code;
		formName = loc.name;
		formType = loc.type;
		formAddress = loc.address;
		formLatitude = loc.latitude ?? null;
		formLongitude = loc.longitude ?? null;
		formError = null;
		showFormModal = true;
	}

	function openDeleteModal(loc: LocationResponse) {
		deletingLocation = loc;
		deleteError = null;
		showDeleteModal = true;
	}

	async function handleSubmitForm(e?: Event) {
		e?.preventDefault();
		formError = null;

		if (!formCode.trim()) {
			formError = 'Kode lokasi wajib diisi';
			return;
		}
		if (!formName.trim()) {
			formError = 'Nama lokasi wajib diisi';
			return;
		}

		const token = getToken();
		if (!token) return;

		formSubmitting = true;
		try {
			if (isEditing && editingId) {
				await updateLocation(token, editingId, {
					code: formCode.trim().toUpperCase(),
					name: formName.trim(),
					type: formType,
					address: formAddress.trim() || undefined,
					latitude: formLatitude,
					longitude: formLongitude
				});
				toast.success(`Lokasi "${formName.trim()}" berhasil diperbarui`);
			} else {
				await createLocation(token, {
					code: formCode.trim().toUpperCase(),
					name: formName.trim(),
					type: formType,
					address: formAddress.trim() || undefined,
					latitude: formLatitude,
					longitude: formLongitude
				});
				toast.success(`Lokasi baru "${formName.trim()}" berhasil ditambahkan`);
			}

			showFormModal = false;
			await loadLocations();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				formError = err.message;
			} else if (err instanceof Error) {
				formError = err.message;
			} else {
				formError = 'Terjadi kesalahan saat menyimpan data lokasi';
			}
		} finally {
			formSubmitting = false;
		}
	}

	async function handleToggleStatus(loc: LocationResponse) {
		const token = getToken();
		if (!token) return;

		try {
			const newStatus = !loc.is_active;
			await setLocationStatus(token, loc.id, newStatus);
			toast.success(
				`Status lokasi "${loc.name}" diubah menjadi ${newStatus ? 'Aktif' : 'Non-Aktif'}`
			);
			await loadLocations();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal mengubah status lokasi';
			}
		}
	}

	async function handleConfirmDelete() {
		if (!deletingLocation) return;

		const token = getToken();
		if (!token) return;

		deleteLoading = true;
		deleteError = null;
		try {
			await deleteLocation(token, deletingLocation.id);
			toast.success(`Lokasi "${deletingLocation.name}" berhasil dihapus`);
			showDeleteModal = false;
			deletingLocation = null;
			await loadLocations();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				deleteError = err.message;
			} else if (err instanceof Error) {
				deleteError = err.message;
			} else {
				deleteError = 'Gagal menghapus lokasi';
			}
		} finally {
			deleteLoading = false;
		}
	}
</script>

<div class="space-y-6">
	<!-- Page Header -->
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
		<div>
			<h2 class="text-xl font-bold tracking-tight text-neutral-900 sm:text-2xl">
				Lokasi & Cabang Toko
			</h2>
			<p class="text-xs text-neutral-500 sm:text-sm">
				Kelola toko cabang fisik, outlet kasir, dan gudang online storefront.
			</p>
		</div>

		<Button variant="primary" onclick={openAddModal}>
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
			<span>Tambah Lokasi</span>
		</Button>
	</div>

	<!-- Notifikasi Umpan Balik -->
	{#if error}
		<Alert variant="error" dismissible title="Terjadi Kesalahan">
			{error}
		</Alert>
	{/if}

	<!-- Statistik Ringkas -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
		<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs">
			<span class="text-xs font-medium text-neutral-500">Total Lokasi</span>
			<p class="mt-1 text-2xl font-bold text-neutral-900">{totalLocations}</p>
		</div>
		<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs">
			<span class="text-xs font-medium text-neutral-500">Toko / Cabang Fisik</span>
			<p class="mt-1 text-2xl font-bold text-primary-600">{physicalCount}</p>
		</div>
		<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs">
			<span class="text-xs font-medium text-neutral-500">Gudang Online</span>
			<p class="mt-1 text-2xl font-bold text-emerald-600">{onlineCount}</p>
		</div>
		<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs">
			<span class="text-xs font-medium text-neutral-500">Aktif Operasional</span>
			<p class="mt-1 text-2xl font-bold text-neutral-800">{activeCount}</p>
		</div>
	</div>

	<!-- Bar Filter & Pencarian (1 ROW Flex Sejajar) -->
	<div
		class="flex flex-col gap-3 rounded-xl border border-neutral-200/80 bg-white p-4 shadow-xs md:flex-row md:items-center md:justify-between"
	>
		<div class="flex flex-1 flex-wrap items-center gap-3">
			<div class="w-full sm:w-72">
				<SearchInput
					bind:value={searchQuery}
					placeholder="Cari kode, nama, atau alamat..."
					debounceMs={200}
					class="w-full max-w-none"
				/>
			</div>

			<!-- Filter Tipe Lokasi via Select2 -->
			<div class="w-full sm:w-56">
				<Select2
					options={locationTypeFilterOptions}
					bind:value={selectedTypeFilter}
					placeholder="Semua Tipe Lokasi"
					clearable={false}
				/>
			</div>

			<!-- Filter Status Lokasi via Select2 -->
			<div class="w-full sm:w-48">
				<Select2
					options={locationStatusFilterOptions}
					bind:value={selectedStatusFilter}
					placeholder="Semua Status"
					clearable={false}
				/>
			</div>
		</div>

		{#if searchQuery || selectedTypeFilter !== 'all' || selectedStatusFilter !== 'all'}
			<p class="text-xs whitespace-nowrap text-neutral-500">
				Menampilkan <span class="font-semibold text-neutral-800">{filteredLocations.length}</span>
				dari {totalLocations} lokasi
			</p>
		{/if}
	</div>

	<!-- Tabel Lokasi Cabang -->
	<Table
		empty={!loading && filteredLocations.length === 0}
		emptyTitle="Belum Ada Lokasi"
		emptyMessage={searchQuery
			? 'Tidak ada lokasi yang sesuai dengan pencarian.'
			: 'Silakan klik tombol Tambah Lokasi untuk mendaftarkan toko cabang atau gudang online.'}
		{loading}
	>
		<thead>
			<tr
				class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-bold tracking-wider text-neutral-500 uppercase"
			>
				<th class="px-4 py-3">Kode</th>
				<th class="px-4 py-3">Nama Lokasi</th>
				<th class="px-4 py-3">Tipe</th>
				<th class="px-4 py-3">Alamat</th>
				<th class="px-4 py-3">Koordinat / Peta</th>
				<th class="px-4 py-3">Status</th>
				<th class="px-4 py-3 text-right">Aksi</th>
			</tr>
		</thead>
		<tbody class="divide-y divide-neutral-100">
			{#each filteredLocations as loc (loc.id)}
				<tr class="transition-colors hover:bg-neutral-50/70">
					<!-- Kode Lokasi -->
					<td class="px-4 py-3">
						<span
							class="rounded-md border border-neutral-200 bg-neutral-100 px-2 py-1 font-mono text-xs font-bold text-neutral-800"
						>
							{loc.code}
						</span>
					</td>

					<!-- Nama Lokasi -->
					<td class="px-4 py-3">
						<div class="flex items-center gap-2">
							<span class="font-semibold text-neutral-900">{loc.name}</span>
						</div>
					</td>

					<!-- Tipe Lokasi -->
					<td class="px-4 py-3">
						{#if loc.type === 'physical'}
							<Badge variant="primary" size="sm">
								{LOCATION_TYPE_LABELS[loc.type]}
							</Badge>
						{:else}
							<Badge variant="success" size="sm">
								{LOCATION_TYPE_LABELS[loc.type]}
							</Badge>
						{/if}
					</td>

					<!-- Alamat -->
					<td class="max-w-xs truncate px-4 py-3 text-neutral-600">
						{loc.address || '-'}
					</td>

					<!-- Koordinat & Peta -->
					<td class="px-4 py-3">
						{#if loc.latitude !== null && loc.latitude !== undefined && loc.longitude !== null && loc.longitude !== undefined}
							<button
								type="button"
								onclick={() => openMapModal(loc)}
								class="group/pin inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-2 py-1 text-2xs font-medium text-neutral-700 shadow-2xs transition-colors hover:border-neutral-900 hover:bg-neutral-50"
								title="Klik untuk melihat pratinjau peta"
							>
								<!-- Map Pin Heroicon -->
								<svg class="h-3.5 w-3.5 text-neutral-600 transition-colors group-hover/pin:text-neutral-900" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M15 10.5a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1115 0z" />
								</svg>
								<span class="font-mono text-neutral-800 font-semibold">{loc.latitude.toFixed(4)}, {loc.longitude.toFixed(4)}</span>
							</button>
						{:else}
							<span class="text-xs text-neutral-400 italic">Belum diatur</span>
						{/if}
					</td>

					<!-- Status Aktif / Non-Aktif -->
					<td class="px-4 py-3">
						<button
							type="button"
							onclick={() => handleToggleStatus(loc)}
							class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors {loc.is_active
								? 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100'
								: 'bg-neutral-100 text-neutral-500 hover:bg-neutral-200'}"
							title="Klik untuk mengubah status aktif"
						>
							<span
								class="h-1.5 w-1.5 rounded-full {loc.is_active
									? 'bg-emerald-500'
									: 'bg-neutral-400'}"
							></span>
							<span>{loc.is_active ? 'Aktif' : 'Non-Aktif'}</span>
						</button>
					</td>

					<!-- Aksi (Titik Tiga Dropdown) -->
					<td class="px-4 py-3 text-right whitespace-nowrap">
						<div class="flex items-center justify-end">
							<ActionMenu
								items={[
									...(loc.latitude !== null && loc.latitude !== undefined && loc.longitude !== null && loc.longitude !== undefined
										? [
												{
													label: 'Lihat Peta Lokasi',
													iconSvg:
														'M15 10.5a3 3 0 11-6 0 3 3 0 016 0zM19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1115 0z',
													onclick: () => openMapModal(loc)
												}
											]
										: []),
									{
										label: loc.is_active ? 'Nonaktifkan Cabang' : 'Aktifkan Cabang',
										iconSvg:
											'M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99',
										onclick: () => handleToggleStatus(loc)
									},
									{
										label: 'Ubah Lokasi',
										iconSvg:
											'M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10',
										onclick: () => openEditModal(loc)
									},
									{
										label: 'Hapus Lokasi',
										variant: 'danger',
										divider: true,
										iconSvg:
											'M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0',
										onclick: () => openDeleteModal(loc)
									}
								]}
							/>
						</div>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>
</div>

<!-- Modal Form Tambah / Edit Lokasi -->
<Modal
	bind:open={showFormModal}
	title={isEditing ? 'Ubah Informasi Lokasi Cabang' : 'Tambah Lokasi / Cabang Baru'}
	size="lg"
>
	<form onsubmit={handleSubmitForm} class="space-y-4" id="location-form">
		{#if formError}
			<Alert variant="error" dismissible>{formError}</Alert>
		{/if}

		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
			<Input
				id="location-code-input"
				label="Kode Lokasi"
				placeholder="Contoh: CKR-01, WH-ONL"
				bind:value={formCode}
				required
				showRequiredAsterisk={true}
			/>

			<Select2
				id="location-type-select"
				label="Tipe Lokasi"
				options={locationTypeOptions}
				bind:value={formType}
				clearable={false}
			/>
		</div>

		<Input
			id="location-name-input"
			label="Nama Lokasi / Cabang"
			placeholder="Contoh: Cabang Cikarang Pusat, Gudang Utama"
			bind:value={formName}
			required
			showRequiredAsterisk={true}
		/>

		<Input
			id="location-address-input"
			label="Alamat Lengkap"
			placeholder="Jl. Raya Industri No. 12, Cikarang"
			bind:value={formAddress}
		/>

		<!-- Map & GPS Coordinates Picker -->
		<div class="pt-2 border-t border-neutral-100">
			<MapPicker
				bind:latitude={formLatitude}
				bind:longitude={formLongitude}
				label="Titik Koordinat & Peta Lokasi (OpenStreetMap)"
				height="260px"
			/>
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
			{isEditing ? 'Simpan Perubahan' : 'Buat Lokasi'}
		</Button>
	{/snippet}
</Modal>

<!-- Modal Pratinjau Peta Lokasi -->
<Modal
	bind:open={showMapModal}
	title="Peta Lokasi Cabang: {viewingMapLocation?.name || ''}"
	size="lg"
>
	<div class="space-y-3">
		<div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between rounded-lg bg-neutral-50 p-3 border border-neutral-200/80">
			<div>
				<div class="text-xs font-bold text-neutral-800">{viewingMapLocation?.name} ({viewingMapLocation?.code})</div>
				<div class="text-2xs text-neutral-600 mt-0.5">{viewingMapLocation?.address || 'Alamat fisik belum diisi'}</div>
			</div>
			{#if viewingMapLocation?.type}
				<div class="shrink-0 mt-1 sm:mt-0">
					<Badge variant={viewingMapLocation.type === 'physical' ? 'primary' : 'success'} size="sm">
						{LOCATION_TYPE_LABELS[viewingMapLocation.type]}
					</Badge>
				</div>
			{/if}
		</div>

		{#if viewingMapLocation?.latitude !== null && viewingMapLocation?.latitude !== undefined && viewingMapLocation?.longitude !== null && viewingMapLocation?.longitude !== undefined}
			<MapPicker
				latitude={viewingMapLocation.latitude}
				longitude={viewingMapLocation.longitude}
				readonly={true}
				height="360px"
				zoom={14}
			/>
		{:else}
			<Alert variant="info">Lokasi cabang ini belum memiliki titik koordinat GPS.</Alert>
		{/if}
	</div>

	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={() => (showMapModal = false)}>
			Tutup
		</Button>
	{/snippet}
</Modal>

<!-- Modal Konfirmasi Hapus Lokasi -->
<Modal bind:open={showDeleteModal} title="Konfirmasi Hapus Lokasi" size="sm">
	<div class="space-y-3">
		{#if deleteError}
			<Alert variant="error" dismissible>{deleteError}</Alert>
		{/if}

		<p class="text-xs leading-relaxed text-neutral-600">
			Apakah Anda yakin ingin menghapus lokasi
			<span class="font-bold text-neutral-900"
				>{deletingLocation?.name} ({deletingLocation?.code})</span
			>? Tindakan ini tidak dapat dibatalkan.
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
			Hapus Lokasi
		</Button>
	{/snippet}
</Modal>
