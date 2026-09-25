<script lang="ts">
	/**
	 * Halaman Manajemen Staf & Pengguna (Sistem & Otorisasi).
	 *
	 * Fitur Kunci:
	 * - Tinjauan metrik total staf, staf aktif, dan distribusi peran (Admin, Gudang, Kasir).
	 * - Filter peran, cabang/lokasi penugasan, dan pencarian cepat nama/username/email.
	 * - Tabel data staf dengan avatar inisial, badge peran semantik, dan status akun.
	 * - Modal registrasi staf baru dengan penugasan cabang terintegrasi.
	 * - Modal peninjauan detail akun staf dan tanggal audit bergabung.
	 * - Toggle status akun aktif / nonaktif.
	 * - Type-safe, Zero-Warning TypeScript & Svelte 5 Runes.
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import { listUsers, createUser, setUserStatus, listLocations } from '@erp/api-client';
	import type { UserResponse, LocationResponse, UserRole, CreateUserRequest } from '@erp/types';
	import { ApiError, USER_ROLE_LABELS } from '@erp/types';
	import {
		Button,
		Input,
		Select,
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

	// ── 1. State Utama ────────────────────────────────────────────────────────
	let users = $state<UserResponse[]>([]);
	let locations = $state<LocationResponse[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Filter & Pencarian
	let searchQuery = $state('');
	let selectedRoleFilter = $state<string>('');
	let selectedLocationFilter = $state<string>('');
	let activeOnlyFilter = $state(false);

	// Pagination
	let currentPage = $state(1);
	let pageSize = $state(10);

	// ── 2. State Modal Tambah Staf ────────────────────────────────────────────
	let showCreateModal = $state(false);
	let formName = $state('');
	let formUsername = $state('');
	let formEmail = $state('');
	let formPassword = $state('');
	let formRole = $state<UserRole>('cashier');
	let formLocationId = $state<string>('');
	let formSubmitting = $state(false);
	let formError = $state<string | null>(null);

	// ── 3. State Modal Detail Staf ────────────────────────────────────────────
	let showDetailModal = $state(false);
	let detailUser = $state<UserResponse | null>(null);

	// ── 4. State Modal Konfirmasi Status ──────────────────────────────────────
	let showStatusModal = $state(false);
	let targetStatusUser = $state<UserResponse | null>(null);
	let statusSubmitting = $state(false);

	// ── 5. Lifecycle & Data Fetching ──────────────────────────────────────────
	onMount(() => {
		void loadInitialData();
	});

	async function loadInitialData() {
		const token = getToken();
		if (!token) return;

		loading = true;
		error = null;

		try {
			const [userList, locList] = await Promise.all([
				listUsers(token),
				listLocations(token)
			]);
			users = userList;
			locations = locList;
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat data staf pengguna';
			}
		} finally {
			loading = false;
		}
	}

	async function refreshUsers() {
		const token = getToken();
		if (!token) return;

		try {
			users = await listUsers(token);
		} catch {
			toast.error('Gagal memperbarui daftar pengguna');
		}
	}

	// ── 6. Opsi Form & Filter ─────────────────────────────────────────────────
	const roleSelectOptions = [
		{ value: 'cashier', label: 'Kasir (Operator POS Penjualan)' },
		{ value: 'warehouse', label: 'Admin Gudang (Kelola Stok Fisik & Mutasi)' },
		{ value: 'admin', label: 'Admin Operasional (Kepala Cabang / Supervisor)' },
		{ value: 'superadmin', label: 'Super Administrator (Akses Teknis Penuh)' },
		{ value: 'owner', label: 'Owner (Pemilik Bisnis)' }
	];

	const locationSelect2Options = $derived<Select2Option[]>([
		{ value: '', label: 'Semua Cabang / Kantor Pusat' },
		...locations.map((loc) => ({
			value: loc.id,
			label: `${loc.name} (${loc.code})`,
			subtext: loc.address || loc.type
		}))
	]);

	const locationFormOptions = $derived<Select2Option[]>([
		{ value: '', label: 'Kantor Pusat / Akses Seluruh Lokasi' },
		...locations.map((loc) => ({
			value: loc.id,
			label: `${loc.name} (${loc.code})`,
			subtext: loc.address || loc.type
		}))
	]);

	// ── 7. Filter & Kalkulasi Data Tampilan ───────────────────────────────────
	const filteredUsers = $derived.by(() => {
		let list = users;

		// Filter pencarian
		if (searchQuery.trim()) {
			const q = searchQuery.toLowerCase().trim();
			list = list.filter(
				(u) =>
					u.name.toLowerCase().includes(q) ||
					u.username.toLowerCase().includes(q) ||
					u.email.toLowerCase().includes(q)
			);
		}

		// Filter peran
		if (selectedRoleFilter) {
			list = list.filter((u) => u.role === selectedRoleFilter);
		}

		// Filter lokasi cabang
		if (selectedLocationFilter) {
			list = list.filter((u) => u.location_id === selectedLocationFilter);
		}

		// Filter hanya aktif
		if (activeOnlyFilter) {
			list = list.filter((u) => u.is_active);
		}

		return list;
	});

	const totalItems = $derived(filteredUsers.length);
	const totalPages = $derived(Math.max(1, Math.ceil(totalItems / pageSize)));

	const paginatedUsers = $derived.by(() => {
		const start = (currentPage - 1) * pageSize;
		return filteredUsers.slice(start, start + pageSize);
	});

	// Statistik Ringkasan
	const stats = $derived.by(() => {
		const total = users.length;
		const active = users.filter((u) => u.is_active).length;
		const adminCount = users.filter((u) => u.role === 'admin' || u.role === 'superadmin' || u.role === 'owner').length;
		const warehouseCount = users.filter((u) => u.role === 'warehouse').length;
		const cashierCount = users.filter((u) => u.role === 'cashier').length;
		return { total, active, adminCount, warehouseCount, cashierCount };
	});

	function getLocationName(locId?: string): string {
		if (!locId) return 'Kantor Pusat / Global';
		const loc = locations.find((l) => l.id === locId);
		return loc ? loc.name : 'Lokasi Tidak Dikenal';
	}

	function getRoleBadgeVariant(role: string): 'primary' | 'success' | 'warning' | 'info' | 'default' {
		switch (role) {
			case 'owner':
			case 'superadmin':
				return 'primary';
			case 'admin':
				return 'info';
			case 'warehouse':
				return 'warning';
			case 'cashier':
				return 'success';
			default:
				return 'default';
		}
	}

	function formatDateTime(isoString: string): string {
		try {
			const d = new Date(isoString);
			return d.toLocaleDateString('id-ID', {
				day: 'numeric',
				month: 'short',
				year: 'numeric'
			});
		} catch {
			return isoString;
		}
	}

	// ── 8. Handler Aksi Form Staf Baru ────────────────────────────────────────
	function openCreateModal() {
		formName = '';
		formUsername = '';
		formEmail = '';
		formPassword = '';
		formRole = 'cashier';
		formLocationId = '';
		formError = null;
		showCreateModal = true;
	}

	async function handleCreateUser(e: SubmitEvent) {
		e.preventDefault();
		const token = getToken();
		if (!token) return;

		if (!formName.trim() || !formUsername.trim() || !formEmail.trim() || !formPassword) {
			formError = 'Semua kolom bertanda bintang wajib diisi';
			return;
		}

		if (formPassword.length < 6) {
			formError = 'Kata sandi minimal 6 karakter';
			return;
		}

		formSubmitting = true;
		formError = null;

		const payload: CreateUserRequest = {
			name: formName.trim(),
			username: formUsername.trim().toLowerCase(),
			email: formEmail.trim().toLowerCase(),
			password: formPassword,
			role: formRole,
			location_id: formLocationId ? formLocationId : undefined
		};

		try {
			const created = await createUser(token, payload);
			toast.success(`Staf ${created.name} (@${created.username}) berhasil didaftarkan`);
			showCreateModal = false;
			await refreshUsers();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				formError = err.message;
			} else if (err instanceof Error) {
				formError = err.message;
			} else {
				formError = 'Gagal mendaftarkan akun staf baru';
			}
		} finally {
			formSubmitting = false;
		}
	}

	// ── 9. Handler Toggle Status Staf ─────────────────────────────────────────
	function promptToggleStatus(u: UserResponse) {
		targetStatusUser = u;
		showStatusModal = true;
	}

	async function confirmToggleStatus() {
		if (!targetStatusUser) return;
		const token = getToken();
		if (!token) return;

		statusSubmitting = true;
		const nextState = !targetStatusUser.is_active;

		try {
			await setUserStatus(token, targetStatusUser.id, nextState);
			toast.success(
				nextState
					? `Akun staf ${targetStatusUser.name} berhasil diaktifkan kembali`
					: `Akun staf ${targetStatusUser.name} telah dinonaktifkan`
			);
			showStatusModal = false;
			targetStatusUser = null;
			await refreshUsers();
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				toast.error(err.message);
			} else {
				toast.error('Gagal memperbarui status akun staf');
			}
		} finally {
			statusSubmitting = false;
		}
	}

	function openDetail(u: UserResponse) {
		detailUser = u;
		showDetailModal = true;
	}
</script>

<div class="space-y-6">
	<!-- Page Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h1 class="text-xl font-bold tracking-tight text-neutral-900 sm:text-2xl">
				Staf & Pengguna Sistem
			</h1>
			<p class="mt-1 text-xs text-neutral-500">
				Kelola akun staf internal toko, penugasan lokasi cabang, dan otorisasi operasional kasir serta gudang.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" onclick={refreshUsers}>
				<svg class="h-4 w-4 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
					<path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
				</svg>
				Segarkan
			</Button>

			<Button variant="primary" size="sm" onclick={openCreateModal}>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M19 7.5v3m0 0v3m0-3h3m-3 0h-3m-2.25-4.125a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zM4 19.235v-.11a6.375 6.375 0 0112.75 0v.109A12.318 12.318 0 0110.374 21c-2.331 0-4.512-.645-6.374-1.765z" />
				</svg>
				Tambah Staf
			</Button>
		</div>
	</div>

	<!-- Alert Error Global -->
	{#if error}
		<Alert variant="error" title="Gagal Memuat Data">{error}</Alert>
	{/if}

	<!-- Kartu Ringkasan Metrik Staf (5 Kolom Bersih) -->
	<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
		<!-- Total Staf -->
		<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
			<div class="text-[9.5px] font-semibold tracking-wider text-neutral-400 uppercase">Total Staf</div>
			<div class="mt-1 text-2xl font-normal text-neutral-900 sm:text-3xl">{stats.total}</div>
			<div class="mt-0.5 text-[10px] text-neutral-400/80">Akun terdaftar</div>
		</div>

		<!-- Staf Aktif -->
		<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
			<div class="text-[9.5px] font-semibold tracking-wider text-emerald-600 uppercase">Status Aktif</div>
			<div class="mt-1 text-2xl font-normal text-emerald-700 sm:text-3xl">{stats.active}</div>
			<div class="mt-0.5 text-[10px] text-emerald-600/80">Bisa login ke sistem</div>
		</div>

		<!-- Pimpinan & Admin -->
		<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
			<div class="text-[9.5px] font-semibold tracking-wider text-sky-600 uppercase">Pimpinan & Admin</div>
			<div class="mt-1 text-2xl font-normal text-sky-700 sm:text-3xl">{stats.adminCount}</div>
			<div class="mt-0.5 text-[10px] text-neutral-400/80">Owner, Superadmin, Admin</div>
		</div>

		<!-- Admin Gudang -->
		<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
			<div class="text-[9.5px] font-semibold tracking-wider text-amber-600 uppercase">Admin Gudang</div>
			<div class="mt-1 text-2xl font-normal text-amber-700 sm:text-3xl">{stats.warehouseCount}</div>
			<div class="mt-0.5 text-[10px] text-neutral-400/80">Pengelola stok & mutasi</div>
		</div>

		<!-- Kasir Penjualan -->
		<div class="rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs">
			<div class="text-[9.5px] font-semibold tracking-wider text-emerald-600 uppercase">Kasir Toko</div>
			<div class="mt-1 text-2xl font-normal text-emerald-700 sm:text-3xl">{stats.cashierCount}</div>
			<div class="mt-0.5 text-[10px] text-neutral-400/80">Operator transaksi POS</div>
		</div>
	</div>

	<!-- Filter & Search Toolbar -->
	<div class="flex flex-col gap-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs lg:flex-row lg:items-center lg:justify-between">
		<!-- Quick Filter Role Pills -->
		<div class="flex flex-wrap items-center gap-1.5">
			<button
				type="button"
				onclick={() => { selectedRoleFilter = ''; currentPage = 1; }}
				class="inline-flex items-center rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {selectedRoleFilter === ''
					? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50 shadow-2xs'}"
			>
				Semua Peran ({users.length})
			</button>

			<button
				type="button"
				onclick={() => { selectedRoleFilter = 'admin'; currentPage = 1; }}
				class="inline-flex items-center rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {selectedRoleFilter === 'admin'
					? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50 shadow-2xs'}"
			>
				Admin
			</button>

			<button
				type="button"
				onclick={() => { selectedRoleFilter = 'warehouse'; currentPage = 1; }}
				class="inline-flex items-center rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {selectedRoleFilter === 'warehouse'
					? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50 shadow-2xs'}"
			>
				Gudang
			</button>

			<button
				type="button"
				onclick={() => { selectedRoleFilter = 'cashier'; currentPage = 1; }}
				class="inline-flex items-center rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {selectedRoleFilter === 'cashier'
					? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50 shadow-2xs'}"
			>
				Kasir
			</button>
		</div>

		<!-- Filter Lokasi Cabang & Search Input -->
		<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
			<div class="w-full sm:w-56">
				<Select2
					options={locationSelect2Options}
					value={selectedLocationFilter}
					placeholder="Semua Lokasi / Cabang"
					searchPlaceholder="Cari cabang..."
					clearable={true}
					onchange={(val) => { selectedLocationFilter = String(val); currentPage = 1; }}
				/>
			</div>

			<div class="w-full sm:w-64">
				<SearchInput
					bind:value={searchQuery}
					placeholder="Cari nama, username, email..."
					onsearch={() => { currentPage = 1; }}
				/>
			</div>
		</div>
	</div>

	<!-- Tabel Daftar Staf Pengguna -->
	<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
		<Table empty={paginatedUsers.length === 0} emptyMessage={loading ? 'Memuat data staf pengguna...' : 'Tidak ada staf yang sesuai kriteria pencarian.'}>
			<thead>
				<tr class="border-b border-neutral-200 bg-neutral-50/80 text-left text-2xs font-semibold tracking-wider text-neutral-600 uppercase">
					<th class="px-4 py-3">Staf Pengguna</th>
					<th class="px-4 py-3">Email</th>
					<th class="px-4 py-3 text-center">Peran Sistem</th>
					<th class="px-4 py-3">Penugasan Lokasi</th>
					<th class="px-4 py-3 text-center">Status</th>
					<th class="px-4 py-3 text-center">Terdaftar</th>
					<th class="px-4 py-3 text-right">Aksi</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-100 text-xs">
				{#each paginatedUsers as u (u.id)}
					<tr class="transition-colors hover:bg-neutral-50/60 {!u.is_active ? 'bg-neutral-50/40 opacity-75' : ''}">
						<!-- Staf Profil & Avatar -->
						<td class="px-4 py-3">
							<div class="flex items-center gap-2.5">
								<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-neutral-900 text-2xs font-bold text-white shadow-2xs">
									{u.name.charAt(0).toUpperCase()}
								</div>
								<div class="min-w-0">
									<div class="font-semibold text-neutral-900 truncate">{u.name}</div>
									<div class="font-mono text-2xs text-neutral-400">@{u.username}</div>
								</div>
							</div>
						</td>

						<!-- Email -->
						<td class="px-4 py-3 text-neutral-600 font-mono text-xs">
							{u.email}
						</td>

						<!-- Peran Badge -->
						<td class="px-4 py-3 text-center whitespace-nowrap">
							<Badge variant={getRoleBadgeVariant(u.role)} size="sm">
								{USER_ROLE_LABELS[u.role] || u.role}
							</Badge>
						</td>

						<!-- Lokasi Cabang -->
						<td class="px-4 py-3 text-neutral-700">
							<div class="flex items-center gap-1.5">
								<svg class="h-3.5 w-3.5 shrink-0 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
									<path stroke-linecap="round" stroke-linejoin="round" d="M15 10.5a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1115 0z" />
								</svg>
								<span class="truncate">{getLocationName(u.location_id)}</span>
							</div>
						</td>

						<!-- Status Badge -->
						<td class="px-4 py-3 text-center whitespace-nowrap">
							{#if u.is_active}
								<Badge variant="success" size="sm">Aktif</Badge>
							{:else}
								<Badge variant="default" size="sm">Nonaktif</Badge>
							{/if}
						</td>

						<!-- Tanggal Terdaftar -->
						<td class="px-4 py-3 text-center font-mono text-xs text-neutral-500 whitespace-nowrap">
							{formatDateTime(u.created_at)}
						</td>

						<!-- Aksi Menu -->
						<td class="px-4 py-3 text-right whitespace-nowrap">
							<ActionMenu
								items={[
									{
										label: 'Lihat Detail Staf',
										icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M15 9h3.75M15 12h3.75M15 15h3.75M4.5 19.5h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5zm6-10.125a1.875 1.875 0 11-3.75 0 1.875 1.875 0 013.75 0zm1.294 6.336a6.721 6.721 0 01-3.17.789 6.721 6.721 0 01-3.168-.789 3.376 3.376 0 016.338 0z" /></svg>`,
										onClick: () => openDetail(u)
									},
									...(u.role !== 'owner'
										? [
												{
													label: u.is_active ? 'Nonaktifkan Akun' : 'Aktifkan Akun',
													variant: (u.is_active ? 'danger' : 'default') as 'danger' | 'default',
													divider: true,
													icon: `<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" /></svg>`,
													onClick: () => promptToggleStatus(u)
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
	<Pagination
		page={currentPage}
		{totalPages}
		{totalItems}
		limit={pageSize}
		onPageChange={(p) => { currentPage = p; }}
	/>
</div>

<!-- Modal 1: Tambah Staf Baru -->
<Modal bind:open={showCreateModal} title="Tambah Staf Pengguna Baru" size="xl">
	<form onsubmit={handleCreateUser} class="space-y-4">
		{#if formError}
			<Alert variant="error" title="Gagal Menyimpan">{formError}</Alert>
		{/if}

		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
			<!-- Nama Lengkap -->
			<div class="sm:col-span-2">
				<Input
					label="Nama Lengkap Staf *"
					placeholder="Contoh: Budi Santoso"
					bind:value={formName}
					required
				/>
			</div>

			<!-- Username -->
			<div>
				<Input
					label="Username Login *"
					placeholder="contoh: budi_gudang"
					bind:value={formUsername}
					required
				/>
			</div>

			<!-- Email -->
			<div>
				<Input
					type="email"
					label="Alamat Email *"
					placeholder="contoh: budi@tokoretail.com"
					bind:value={formEmail}
					required
				/>
			</div>

			<!-- Password -->
			<div class="sm:col-span-2">
				<Input
					type="password"
					label="Kata Sandi Awal *"
					placeholder="Minimal 6 karakter"
					bind:value={formPassword}
					showPasswordToggle={true}
					required
				/>
				<p class="mt-1 text-2xs text-neutral-400">
					Staf dapat mengganti kata sandi ini secara mandiri melalui pengaturan profil akun.
				</p>
			</div>

			<!-- Role / Peran -->
			<div>
				<Select
					label="Peran Akun (Role) *"
					options={roleSelectOptions}
					bind:value={formRole}
					required
				/>
			</div>

			<!-- Lokasi Penugasan -->
			<div>
				<label for="form-user-location" class="mb-1 block text-xs font-semibold text-neutral-700">
					Penugasan Cabang / Lokasi
				</label>
				<Select2
					id="form-user-location"
					options={locationFormOptions}
					bind:value={formLocationId}
					placeholder="Pilih lokasi kerja..."
					searchPlaceholder="Cari cabang..."
					clearable={true}
				/>
				<p class="mt-1 text-2xs text-neutral-400">
					Kosongkan jika staf berstatus kantor pusat / pengawas seluruh cabang.
				</p>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end gap-2">
				<Button
					type="button"
					variant="outline"
					disabled={formSubmitting}
					onclick={() => { showCreateModal = false; }}
				>
					Batal
				</Button>
				<Button
					type="submit"
					variant="primary"
					loading={formSubmitting}
				>
					Daftarkan Staf
				</Button>
			</div>
		{/snippet}
	</form>
</Modal>

<!-- Modal 2: Detail Staf Pengguna -->
<Modal bind:open={showDetailModal} title="Detail Akun Staf" size="md">
	{#if detailUser}
		<div class="space-y-4 text-xs">
			<div class="flex items-center gap-3 rounded-lg border border-neutral-200 bg-neutral-50/70 p-3">
				<div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-neutral-900 text-sm font-bold text-white shadow-2xs">
					{detailUser.name.charAt(0).toUpperCase()}
				</div>
				<div class="min-w-0 flex-1">
					<div class="text-sm font-bold text-neutral-900 truncate">{detailUser.name}</div>
					<div class="font-mono text-2xs text-neutral-400">@{detailUser.username}</div>
					<div class="mt-1 flex items-center gap-1.5">
						<Badge variant={getRoleBadgeVariant(detailUser.role)} size="sm">
							{USER_ROLE_LABELS[detailUser.role] || detailUser.role}
						</Badge>
						{#if detailUser.is_active}
							<Badge variant="success" size="sm">Aktif</Badge>
						{:else}
							<Badge variant="default" size="sm">Nonaktif</Badge>
						{/if}
					</div>
				</div>
			</div>

			<div class="divide-y divide-neutral-100 rounded-lg border border-neutral-200 bg-white">
				<div class="flex justify-between px-3.5 py-2.5">
					<span class="text-neutral-500">ID Pengguna</span>
					<span class="font-mono text-neutral-700">{detailUser.id}</span>
				</div>
				<div class="flex justify-between px-3.5 py-2.5">
					<span class="text-neutral-500">Alamat Email</span>
					<span class="font-mono text-neutral-900">{detailUser.email}</span>
				</div>
				<div class="flex justify-between px-3.5 py-2.5">
					<span class="text-neutral-500">Lokasi Penugasan</span>
					<span class="font-semibold text-neutral-900">{getLocationName(detailUser.location_id)}</span>
				</div>
				<div class="flex justify-between px-3.5 py-2.5">
					<span class="text-neutral-500">Tanggal Terdaftar</span>
					<span class="font-mono text-neutral-700">{formatDateTime(detailUser.created_at)}</span>
				</div>
				<div class="flex justify-between px-3.5 py-2.5">
					<span class="text-neutral-500">Terakhir Diperbarui</span>
					<span class="font-mono text-neutral-700">{formatDateTime(detailUser.updated_at)}</span>
				</div>
			</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end">
			<Button variant="outline" onclick={() => { showDetailModal = false; }}>Tutup</Button>
		</div>
	{/snippet}
</Modal>

<!-- Modal 3: Konfirmasi Ubah Status Akun -->
<Modal bind:open={showStatusModal} title="Konfirmasi Perubahan Status Akun" size="sm">
	{#if targetStatusUser}
		<div class="space-y-3 text-xs text-neutral-600">
			<p>
				Apakah Anda yakin ingin
				<strong class="font-bold {targetStatusUser.is_active ? 'text-rose-600' : 'text-emerald-700'}">
					{targetStatusUser.is_active ? 'menonaktifkan' : 'mengaktifkan kembali'}
				</strong>
				akun staf <strong class="font-semibold text-neutral-900">{targetStatusUser.name}</strong> (@{targetStatusUser.username})?
			</p>
			{#if targetStatusUser.is_active}
				<div class="rounded-lg border border-amber-200 bg-amber-50 p-2.5 text-2xs text-amber-800">
					Setelah dinonaktifkan, staf ini tidak akan dapat login ke sistem Backoffice maupun POS kasir toko.
				</div>
			{/if}
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex items-center justify-end gap-2">
			<Button
				type="button"
				variant="outline"
				disabled={statusSubmitting}
				onclick={() => { showStatusModal = false; targetStatusUser = null; }}
			>
				Batal
			</Button>
			<Button
				variant={targetStatusUser?.is_active ? 'danger' : 'primary'}
				loading={statusSubmitting}
				onclick={confirmToggleStatus}
			>
				{targetStatusUser?.is_active ? 'Ya, Nonaktifkan' : 'Ya, Aktifkan'}
			</Button>
		</div>
	{/snippet}
</Modal>
