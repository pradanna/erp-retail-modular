<script lang="ts">
	/**
	 * Halaman Matriks Hak Akses Peran (PBAC - Permission-Based Access Control).
	 *
	 * Fitur Kunci:
	 * - Mengambil dan menyajikan matriks kapabilitas izin granular per modul.
	 * - Pengelompokan izin berbasis Bounded Context (Inventory: Produk, Stok, Barcode, Serial, Transfer, Garansi; Shared: User & Roles).
	 * - Checkbox interaktif untuk mengaktifkan/menonaktifkan hak akses peran (Admin, Gudang, Kasir).
	 * - Proteksi keamanan: Peran Owner & Superadmin berstatus bypass penuh permanen.
	 * - Pelacakan perubahan (dirty state detection) & tombol simpan per peran atau sekaligus.
	 * - Sinkronisasi langsung ke in-memory cache backend Go (Zero Latency O(1)).
	 * - Zero-Warning TypeScript & Svelte 5 Runes.
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import { getRolePermissionsMatrix, updateRolePermissions } from '@erp/api-client';
	import type { RolePermissionsMatrix, Permission, RoleInfo, UserRole } from '@erp/types';
	import { ApiError, USER_ROLE_LABELS } from '@erp/types';
	import {
		Button,
		Table,
		Alert,
		Badge,
		SearchInput,
		toast
	} from '@erp/ui';

	// ── 1. State Utama ────────────────────────────────────────────────────────
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);

	let permissions = $state<Permission[]>([]);
	let roles = $state<RoleInfo[]>([]);
	// Matrix state lokal yang dapat diedit: Key: role_name, Value: Set of permission_names
	let editableMatrix = $state<Record<string, string[]>>({});
	// Original snapshot untuk dirty checking
	let originalMatrix = $state<Record<string, string[]>>({});

	// Filter pencarian & modul
	let searchQuery = $state('');
	let selectedModuleFilter = $state<string>('all');

	// ── 2. Lifecycle & Data Fetching ──────────────────────────────────────────
	onMount(() => {
		void loadMatrix();
	});

	async function loadMatrix() {
		const token = getToken();
		if (!token) return;

		loading = true;
		error = null;

		try {
			const data: RolePermissionsMatrix = await getRolePermissionsMatrix(token);
			permissions = data.permissions;
			roles = data.roles;

			// Salin matriks ke state lokal yang reaktif
			const clone: Record<string, string[]> = {};
			const orig: Record<string, string[]> = {};
			for (const r of data.roles) {
				const assigned = data.matrix[r.name] || [];
				clone[r.name] = [...assigned];
				orig[r.name] = [...assigned];
			}
			editableMatrix = clone;
			originalMatrix = orig;
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Gagal memuat matriks hak akses peran';
			}
		} finally {
			loading = false;
		}
	}

	// ── 3. Modul & Filter ─────────────────────────────────────────────────────
	const availableModules = $derived.by(() => {
		const set = new Set<string>();
		for (const p of permissions) {
			if (p.module) set.add(p.module);
		}
		return Array.from(set);
	});

	// Peran yang dapat dikonfigurasi (bukan owner / superadmin karena mereka bypass)
	const configurableRoles = $derived(
		roles.filter((r) => r.name !== 'owner' && r.name !== 'superadmin' && r.name !== 'customer')
	);

	const filteredPermissions = $derived.by(() => {
		let list = permissions;

		// Filter modul
		if (selectedModuleFilter !== 'all') {
			list = list.filter((p) => p.module === selectedModuleFilter);
		}

		// Filter pencarian
		if (searchQuery.trim()) {
			const q = searchQuery.toLowerCase().trim();
			list = list.filter(
				(p) =>
					p.name.toLowerCase().includes(q) ||
					p.description.toLowerCase().includes(q) ||
					p.module.toLowerCase().includes(q)
			);
		}

		return list;
	});

	// Pengelompokan izin per modul untuk tabel terstruktur
	const groupedPermissions = $derived.by(() => {
		const groups: Record<string, Permission[]> = {};
		for (const p of filteredPermissions) {
			if (!groups[p.module]) {
				groups[p.module] = [];
			}
			groups[p.module].push(p);
		}
		return groups;
	});

	// ── 4. Handler Interaksi Checkbox ─────────────────────────────────────────
	function hasPermission(role: string, permName: string): boolean {
		const perms = editableMatrix[role];
		if (!perms) return false;
		return perms.includes(permName);
	}

	function togglePermission(role: string, permName: string) {
		const current = editableMatrix[role] ? [...editableMatrix[role]] : [];
		const index = current.indexOf(permName);

		if (index > -1) {
			// Cabut izin
			current.splice(index, 1);
		} else {
			// Berikan izin
			current.push(permName);
		}

		editableMatrix[role] = current;
	}

	function isRoleDirty(role: string): boolean {
		const current = editableMatrix[role] || [];
		const orig = originalMatrix[role] || [];
		if (current.length !== orig.length) return true;
		const set = new Set(orig);
		return current.some((p) => !set.has(p));
	}

	const hasAnyChanges = $derived.by(() => {
		return configurableRoles.some((r) => isRoleDirty(r.name));
	});

	// ── 5. Handler Simpan Peran ───────────────────────────────────────────────
	async function saveRolePermissions(role: string) {
		const token = getToken();
		if (!token) return;

		saving = true;
		const permsToSave = editableMatrix[role] || [];

		try {
			await updateRolePermissions(token, role, permsToSave);
			// Update original snapshot
			originalMatrix[role] = [...permsToSave];
			const roleDisplayName = USER_ROLE_LABELS[role as UserRole] || role;
			toast.success(`Hak akses peran '${roleDisplayName}' berhasil disinkronkan ke sistem`);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				toast.error(err.message);
			} else {
				toast.error('Gagal memperbarui hak akses peran');
			}
		} finally {
			saving = false;
		}
	}

	async function saveAllRoles() {
		const dirtyRoles = configurableRoles.filter((r) => isRoleDirty(r.name));
		if (dirtyRoles.length === 0) {
			toast.info('Tidak ada perubahan izin yang perlu disimpan');
			return;
		}

		saving = true;
		try {
			for (const r of dirtyRoles) {
				const token = getToken();
				if (!token) break;
				const perms = editableMatrix[r.name] || [];
				await updateRolePermissions(token, r.name, perms);
				originalMatrix[r.name] = [...perms];
			}
			toast.success('Seluruh konfigurasi hak akses peran berhasil diperbarui');
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				toast.error(err.message);
			} else {
				toast.error('Gagal memperbarui izin peran');
			}
		} finally {
			saving = false;
		}
	}

	function resetChanges() {
		const clone: Record<string, string[]> = {};
		for (const key of Object.keys(originalMatrix)) {
			clone[key] = [...originalMatrix[key]];
		}
		editableMatrix = clone;
		toast.info('Perubahan centang izin telah direset ke kondisi tersimpan');
	}

	function formatModuleName(mod: string): string {
		switch (mod) {
			case 'inventory':
				return 'Modul Inventaris & Stok (Inventory)';
			case 'shared':
				return 'Modul Sistem & Pengguna (Shared Context)';
			case 'purchasing':
				return 'Modul Pembelian (Purchasing)';
			case 'sales':
				return 'Modul Penjualan & Kasir (Sales)';
			case 'finance':
				return 'Modul Keuangan (Finance)';
			case 'commission':
				return 'Modul Komisi Salesman (Commission)';
			default:
				return `Modul ${mod.toUpperCase()}`;
		}
	}
</script>

<div class="space-y-6">
	<!-- Page Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h1 class="text-xl font-bold tracking-tight text-neutral-900 sm:text-2xl">
				Hak Akses Peran & Otorisasi PBAC
			</h1>
			<p class="mt-1 text-xs text-neutral-500">
				Atur matriks izin kapabilitas granular untuk setiap peran operasional (Admin, Gudang, Kasir).
			</p>
		</div>

		<div class="flex items-center gap-2">
			{#if hasAnyChanges}
				<Button
					variant="outline"
					size="sm"
					disabled={saving}
					onclick={resetChanges}
				>
					Batal Ubah
				</Button>

				<Button
					variant="primary"
					size="sm"
					loading={saving}
					onclick={saveAllRoles}
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
					</svg>
					Simpan Semua Perubahan
				</Button>
			{:else}
				<Button variant="outline" size="sm" onclick={loadMatrix}>
					<svg class="h-4 w-4 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
						<path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
					</svg>
					Segarkan
				</Button>
			{/if}
		</div>
	</div>

	<!-- Alert Peringatan & Edukasi Keamanan PBAC -->
	<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-4 text-xs text-neutral-700 shadow-2xs">
		<div class="flex items-start gap-3">
			<div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-neutral-900 text-white">
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
				</svg>
			</div>
			<div class="space-y-1">
				<div class="font-bold text-neutral-900">Prinsip Keamanan Hak Akses Peran (PBAC)</div>
				<p class="text-neutral-600 leading-relaxed">
					Peran <strong>Owner</strong> dan <strong>Super Administrator</strong> memiliki wewenang <em>bypass</em> sistem penuh ke seluruh modul secara otomatis demi menjaga stabilitas instalasi.
					Perubahan centang izin di bawah ini berlaku langsung untuk akun dengan peran <strong>Admin</strong>, <strong>Gudang</strong>, dan <strong>Kasir</strong> seketika setelah disimpan (in-memory cached O(1)).
				</p>
			</div>
		</div>
	</div>

	<!-- Alert Error Global -->
	{#if error}
		<Alert variant="error" title="Gagal Memuat Matriks">{error}</Alert>
	{/if}

	<!-- Filter & Search Toolbar -->
	<div class="flex flex-col gap-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs sm:flex-row sm:items-center sm:justify-between">
		<!-- Quick Filter Modul Pills -->
		<div class="flex flex-wrap items-center gap-1.5">
			<button
				type="button"
				onclick={() => { selectedModuleFilter = 'all'; }}
				class="inline-flex items-center rounded-full border px-3 py-1.5 text-xs font-medium transition-colors {selectedModuleFilter === 'all'
					? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
					: 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50 shadow-2xs'}"
			>
				Semua Modul ({permissions.length})
			</button>

			{#each availableModules as mod}
				<button
					type="button"
					onclick={() => { selectedModuleFilter = mod; }}
					class="inline-flex items-center rounded-full border px-3 py-1.5 text-xs font-medium capitalize transition-colors {selectedModuleFilter === mod
						? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
						: 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50 shadow-2xs'}"
				>
					{mod}
				</button>
			{/each}
		</div>

		<!-- Search Input -->
		<div class="w-full sm:w-64">
			<SearchInput
				bind:value={searchQuery}
				placeholder="Cari izin atau aksi..."
			/>
		</div>
	</div>

	<!-- Matriks Tabel PBAC -->
	<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
		<Table empty={filteredPermissions.length === 0} emptyMessage={loading ? 'Memuat matriks otorisasi...' : 'Tidak ada izin yang sesuai dengan pencarian.'}>
			<thead>
				<tr class="border-b border-neutral-200 bg-neutral-50/80 text-left text-2xs font-semibold tracking-wider text-neutral-600 uppercase">
					<th class="px-4 py-3 min-w-[280px]">Kapabilitas Izin (Resource & Aksi)</th>
					{#each configurableRoles as r (r.name)}
						<th class="px-4 py-3 text-center min-w-[130px]">
							<div class="flex flex-col items-center gap-1">
								<span class="font-bold text-neutral-900">{r.display_name}</span>
								{#if isRoleDirty(r.name)}
									<button
										type="button"
										disabled={saving}
										onclick={() => saveRolePermissions(r.name)}
										class="rounded bg-primary-600 px-2 py-0.5 text-[9.5px] font-semibold text-white transition-opacity hover:opacity-90 shadow-2xs"
										title="Simpan perubahan peran ini"
									>
										Simpan
									</button>
								{/if}
							</div>
						</th>
					{/each}
					<th class="px-4 py-3 text-center min-w-[120px] bg-neutral-100/50">Superadmin</th>
					<th class="px-4 py-3 text-center min-w-[100px] bg-neutral-100/50">Owner</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-100 text-xs">
				{#each Object.entries(groupedPermissions) as [mod, perms] (mod)}
					<!-- Header Grup Modul -->
					<tr class="bg-neutral-100/70 border-y border-neutral-200">
						<td colspan={configurableRoles.length + 3} class="px-4 py-2 font-bold tracking-wider text-neutral-800 uppercase text-2xs">
							<div class="flex items-center gap-2">
								<span class="h-2 w-2 rounded-full bg-neutral-900"></span>
								{formatModuleName(mod)}
								<span class="font-normal text-neutral-500">({perms.length} izin)</span>
							</div>
						</td>
					</tr>

					<!-- Baris Izin Granular -->
					{#each perms as p (p.name)}
						<tr class="transition-colors hover:bg-neutral-50/60 {p.name === 'inventory.stocks.adjust' ? 'bg-amber-50/30' : ''}">
							<!-- Info Izin -->
							<td class="px-4 py-2.5">
								<div class="font-mono text-2xs font-semibold text-neutral-900">
									{p.name}
									{#if p.name === 'inventory.stocks.adjust'}
										<span class="ml-1.5 rounded bg-amber-100 px-1.5 py-0.2 text-[9px] font-bold text-amber-800 border border-amber-200">
											Stock Opname
										</span>
									{/if}
								</div>
								<div class="mt-0.5 text-2xs text-neutral-500 line-clamp-1">{p.description}</div>
							</td>

							<!-- Checkbox per Configurable Role -->
							{#each configurableRoles as r (r.name)}
								{@const checked = hasPermission(r.name, p.name)}
								<td class="px-4 py-2.5 text-center">
									<label class="inline-flex cursor-pointer items-center justify-center p-1">
										<input
											type="checkbox"
											{checked}
											onchange={() => togglePermission(r.name, p.name)}
											class="h-4 w-4 rounded border-neutral-300 text-neutral-900 focus:ring-neutral-900 focus:ring-offset-0 transition-colors cursor-pointer"
										/>
									</label>
								</td>
							{/each}

							<!-- Superadmin Bypass Badge -->
							<td class="px-4 py-2.5 text-center bg-neutral-100/30">
								<Badge variant="primary" size="sm">Bypass</Badge>
							</td>

							<!-- Owner Bypass Badge -->
							<td class="px-4 py-2.5 text-center bg-neutral-100/30">
								<Badge variant="primary" size="sm">Bypass</Badge>
							</td>
						</tr>
					{/each}
				{/each}
			</tbody>
		</Table>
	</div>
</div>
