<script lang="ts">
	/**
	 * Topbar — Header bagian atas Gen-E Backoffice.
	 *
	 * Fitur:
	 * - Tombol hamburger menu mobile untuk memunculkan sidebar
	 * - Breadcrumb / konteks navigasi halaman aktif
	 * - Indikator lokasi aktif staf / cabang
	 * - Dropdown Profil Staf:
	 *   1. Lihat Akun (Modal popup detail akun & peran)
	 *   2. Ganti Password (Modal popup ganti kata sandi mandiri)
	 *   3. Keluar Sesi (Logout & redirect ke /login)
	 */
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getUser, getToken, logout } from '$lib/stores/auth.svelte';
	import { changePassword } from '@erp/api-client';
	import { ApiError, USER_ROLE_LABELS } from '@erp/types';
	import { Button, Input, Modal, Alert, Badge } from '@erp/ui';

	interface Props {
		onOpenMobile?: () => void;
	}

	let { onOpenMobile }: Props = $props();

	const user = $derived(getUser());

	// State untuk dropdown & modal
	let dropdownOpen = $state(false);
	let showProfileModal = $state(false);
	let showPasswordModal = $state(false);

	// State untuk form ganti password
	let oldPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let passwordLoading = $state(false);
	let passwordError = $state<string | null>(null);
	let passwordSuccess = $state<string | null>(null);

	// Penentuan label halaman berdasarkan URL saat ini
	function getPageContext(pathname: string): { module: string; pageTitle: string } {
		if (pathname === '/dashboard') {
			return { module: 'Ikhtisar', pageTitle: 'Dashboard Utama' };
		}
		// Data Master
		if (pathname.startsWith('/master/products') || pathname.startsWith('/inventory/products')) {
			return { module: 'Data Master', pageTitle: 'Katalog Produk' };
		}
		if (pathname.startsWith('/master/categories') || pathname.startsWith('/inventory/categories')) {
			return { module: 'Data Master', pageTitle: 'Kategori Produk' };
		}
		if (pathname.startsWith('/master/locations') || pathname.startsWith('/inventory/locations')) {
			return { module: 'Data Master', pageTitle: 'Cabang & Lokasi' };
		}
		if (pathname.startsWith('/master/barcodes') || pathname.startsWith('/inventory/barcodes')) {
			return { module: 'Data Master', pageTitle: 'Barcode Produk' };
		}
		if (pathname.startsWith('/master/warranties') || pathname.startsWith('/inventory/warranties')) {
			return { module: 'Data Master', pageTitle: 'Garansi Produk' };
		}
		// Inventaris & Stok
		if (pathname.startsWith('/inventory/stocks')) {
			return { module: 'Inventaris & Stok', pageTitle: 'Stok Cabang & Opname' };
		}
		if (pathname.startsWith('/inventory/serials')) {
			return { module: 'Inventaris & Stok', pageTitle: 'Serial & IMEI' };
		}
		if (pathname.startsWith('/inventory/price-overrides')) {
			return { module: 'Inventaris & Stok', pageTitle: 'Promo Cabang' };
		}
		if (pathname.startsWith('/inventory/transfers')) {
			return { module: 'Inventaris & Stok', pageTitle: 'Mutasi Stok Antar Cabang' };
		}
		// Sistem & Otorisasi
		if (pathname.startsWith('/users')) {
			return { module: 'Sistem & Otorisasi', pageTitle: 'Staf & Pengguna' };
		}
		if (pathname.startsWith('/roles')) {
			return { module: 'Sistem & Otorisasi', pageTitle: 'Hak Akses Peran' };
		}
		return { module: 'Backoffice', pageTitle: 'Sistem ERP Retail' };
	}

	const context = $derived(getPageContext(page.url.pathname));

	function resetPasswordForm() {
		oldPassword = '';
		newPassword = '';
		confirmPassword = '';
		passwordError = null;
		passwordSuccess = null;
		passwordLoading = false;
	}

	async function handleLogout() {
		dropdownOpen = false;
		logout();
		await goto('/login');
	}

	async function handleSubmitPassword(e?: Event) {
		e?.preventDefault();
		passwordError = null;
		passwordSuccess = null;

		if (!oldPassword) {
			passwordError = 'Kata sandi saat ini wajib diisi';
			return;
		}
		if (newPassword.length < 6) {
			passwordError = 'Kata sandi baru minimal 6 karakter';
			return;
		}
		if (newPassword !== confirmPassword) {
			passwordError = 'Konfirmasi kata sandi baru tidak sesuai';
			return;
		}

		const token = getToken();
		if (!token) {
			passwordError = 'Sesi telah berakhir, silakan login kembali';
			return;
		}

		passwordLoading = true;
		try {
			await changePassword(token, {
				old_password: oldPassword,
				new_password: newPassword
			});
			passwordSuccess = 'Kata sandi Anda berhasil diperbarui';
			oldPassword = '';
			newPassword = '';
			confirmPassword = '';
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				passwordError = err.message;
			} else if (err instanceof Error) {
				passwordError = err.message;
			} else {
				passwordError = 'Gagal mengganti kata sandi, silakan coba lagi';
			}
		} finally {
			passwordLoading = false;
		}
	}

	function getRoleBadgeVariant(role?: string): 'primary' | 'success' | 'warning' | 'default' {
		switch (role) {
			case 'owner':
			case 'superadmin':
				return 'primary';
			case 'admin':
				return 'primary';
			case 'warehouse':
				return 'warning';
			case 'cashier':
				return 'success';
			default:
				return 'default';
		}
	}
</script>

<header
	class="sticky top-0 z-30 flex h-16 shrink-0 items-center justify-between border-b border-neutral-200 bg-white/95 px-4 backdrop-blur-xs sm:px-6 lg:px-8"
>
	<!-- Bagian Kiri: Tombol Hamburger & Breadcrumbs -->
	<div class="flex items-center gap-3">
		{#if onOpenMobile}
			<button
				type="button"
				class="rounded-lg p-2 text-neutral-500 hover:bg-neutral-100 hover:text-neutral-700 focus:outline-hidden lg:hidden"
				onclick={onOpenMobile}
				aria-label="Buka menu navigasi"
			>
				<svg
					class="h-6 w-6"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
				</svg>
			</button>
		{/if}

		<div class="flex items-center gap-2 text-xs text-neutral-500">
			<span class="font-medium text-neutral-400">{context.module}</span>
			<span class="text-neutral-300">/</span>
			<h1 class="text-sm font-semibold text-neutral-900">{context.pageTitle}</h1>
		</div>
	</div>

	<!-- Bagian Kanan: Indikator Cabang, Status & Menu Profil User -->
	<div class="flex items-center gap-3">
		<!-- Indikator Cabang / Lokasi Staf -->
		<div
			class="hidden items-center gap-2 rounded-lg border border-neutral-200 bg-neutral-50 px-3 py-1.5 text-xs text-neutral-600 sm:flex"
		>
			<svg
				class="h-4 w-4 text-primary-600"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<path
					d="M13.5 21v-7.5a.75.75 0 01.75-.75h3a.75.75 0 01.75.75V21m-4.5 0H2.36m11.14 0H18m0 0h3.64m-1.39 0V9.349m-16.5 11.65V9.35m0 0a3.001 3.001 0 003.75-.615A2.993 2.993 0 009 9.75c.696 0 1.345-.237 1.865-.635A2.991 2.991 0 0012 9.75c.696 0 1.345-.237 1.865-.635A2.991 2.991 0 0015 9.75c.696 0 1.345-.237 1.865-.635A3.001 3.001 0 0020.625 9.35M3.75 9.35l.937-4.685A1.5 1.5 0 016.155 3.39h11.69a1.5 1.5 0 011.468 1.275l.937 4.685"
				/>
			</svg>
			<span class="font-medium text-neutral-800">
				{user?.location_id ? `Lokasi: ${user.location_id}` : 'Semua Cabang (Global)'}
			</span>
		</div>

		<!-- Pemisah Vertikal -->
		<div class="h-6 w-px bg-neutral-200"></div>

		<!-- Dropdown Profil Pengguna -->
		<div class="relative">
			<button
				type="button"
				class="flex items-center gap-2.5 rounded-xl p-1.5 transition-colors hover:bg-neutral-100 focus:outline-hidden"
				onclick={() => (dropdownOpen = !dropdownOpen)}
				aria-expanded={dropdownOpen}
				aria-haspopup="true"
			>
				<!-- Avatar Pengguna -->
				<div
					class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-600 text-xs font-bold text-white shadow-2xs"
				>
					{user?.name?.charAt(0).toUpperCase() || 'U'}
				</div>

				<!-- Nama & Peran (Desktop) -->
				<div class="hidden text-left sm:block">
					<p class="max-w-[120px] truncate text-xs leading-tight font-semibold text-neutral-900">
						{user?.name || 'Pengguna'}
					</p>
					<p class="text-[10px] font-medium text-neutral-500">
						{user?.role ? (USER_ROLE_LABELS[user.role] ?? user.role) : 'Staf'}
					</p>
				</div>

				<!-- Chevron Panah -->
				<svg
					class="h-4 w-4 text-neutral-400 transition-transform duration-200 {dropdownOpen
						? 'rotate-180'
						: ''}"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
				</svg>
			</button>

			<!-- Backdrop Penutup Dropdown saat Klik di Luar -->
			{#if dropdownOpen}
				<div
					class="fixed inset-0 z-30"
					onclick={() => (dropdownOpen = false)}
					onkeydown={(e) => e.key === 'Escape' && (dropdownOpen = false)}
					role="button"
					tabindex="0"
					aria-label="Tutup menu profil"
				></div>

				<!-- Popover Dropdown Card -->
				<div
					class="animate-in fade-in slide-in-from-top-1 absolute top-full right-0 z-40 mt-2 w-64 rounded-2xl border border-neutral-200 bg-white p-2 shadow-xl"
					role="menu"
				>
					<!-- Header Informasi Staf -->
					<div class="mb-1 rounded-xl bg-neutral-50 p-3">
						<div class="flex items-center gap-2.5">
							<div
								class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-600 text-sm font-bold text-white shadow-xs"
							>
								{user?.name?.charAt(0).toUpperCase() || 'U'}
							</div>
							<div class="min-w-0 flex-1">
								<p class="truncate text-xs font-bold text-neutral-900">
									{user?.name || 'Pengguna'}
								</p>
								<p class="truncate text-[11px] text-neutral-500">
									@{user?.username || 'user'}
								</p>
							</div>
						</div>
						<div class="mt-2 flex items-center justify-between border-t border-neutral-200/60 pt-2">
							<span class="text-[10px] text-neutral-500">Hak Akses:</span>
							<Badge variant={getRoleBadgeVariant(user?.role)} size="sm">
								{user?.role ? (USER_ROLE_LABELS[user.role] ?? user.role) : 'Staf'}
							</Badge>
						</div>
					</div>

					<div class="my-1 border-t border-neutral-100"></div>

					<!-- Menu Item 1: Lihat Akun -->
					<button
						type="button"
						class="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-xs font-medium text-neutral-700 transition-colors hover:bg-primary-50 hover:text-primary-700 focus:outline-hidden"
						onclick={() => {
							showProfileModal = true;
							dropdownOpen = false;
						}}
					>
						<svg
							class="h-4.5 w-4.5 text-neutral-400 group-hover:text-primary-600"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="1.75"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path
								d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z"
							/>
						</svg>
						<div class="flex-1">
							<p class="font-medium text-neutral-800">Lihat Akun</p>
							<p class="text-[10px] text-neutral-400">Detail identitas & informasi staf</p>
						</div>
					</button>

					<!-- Menu Item 2: Ganti Password -->
					<button
						type="button"
						class="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-xs font-medium text-neutral-700 transition-colors hover:bg-primary-50 hover:text-primary-700 focus:outline-hidden"
						onclick={() => {
							resetPasswordForm();
							showPasswordModal = true;
							dropdownOpen = false;
						}}
					>
						<svg
							class="h-4.5 w-4.5 text-neutral-400 group-hover:text-primary-600"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="1.75"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path
								d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"
							/>
						</svg>
						<div class="flex-1">
							<p class="font-medium text-neutral-800">Ganti Password</p>
							<p class="text-[10px] text-neutral-400">Perbarui kata sandi akun Anda</p>
						</div>
					</button>

					<div class="my-1 border-t border-neutral-100"></div>

					<!-- Menu Item 3: Keluar Sesi (Logout) -->
					<button
						type="button"
						class="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-xs font-medium text-rose-700 transition-colors hover:bg-rose-50 focus:outline-hidden"
						onclick={handleLogout}
					>
						<svg
							class="h-4.5 w-4.5 text-rose-500"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="1.75"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path
								d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15m3 0 3-3m0 0-3-3m3 3H9"
							/>
						</svg>
						<div class="flex-1">
							<p class="font-semibold text-rose-700">Keluar Sesi</p>
							<p class="text-[10px] text-rose-500">Akhiri sesi penggunaan backoffice</p>
						</div>
					</button>
				</div>
			{/if}
		</div>
	</div>
</header>

<!-- Modal: Lihat Akun Staf -->
<Modal bind:open={showProfileModal} title="Informasi Akun Pengguna" size="md">
	<div class="space-y-4">
		<!-- Header Ringkasan Profil -->
		<div
			class="flex items-center gap-3.5 rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5"
		>
			<div
				class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-primary-600 text-base font-bold text-white shadow-xs"
			>
				{user?.name?.charAt(0).toUpperCase() || 'U'}
			</div>
			<div class="min-w-0 flex-1">
				<h4 class="truncate text-sm font-bold text-neutral-900">{user?.name || '-'}</h4>
				<p class="truncate text-xs text-neutral-500">@{user?.username || '-'}</p>
				<div class="mt-1.5 flex items-center gap-1.5">
					<Badge variant={getRoleBadgeVariant(user?.role)} size="sm">
						{user?.role ? (USER_ROLE_LABELS[user.role] ?? user.role) : 'Staf'}
					</Badge>
					<Badge variant={user?.is_active ? 'success' : 'danger'} size="sm">
						{user?.is_active ? 'Aktif' : 'Non-Aktif'}
					</Badge>
				</div>
			</div>
		</div>

		<!-- Grid Rincian Identitas -->
		<div class="divide-y divide-neutral-100 rounded-xl border border-neutral-200 bg-white">
			<div class="flex items-center justify-between px-3.5 py-2.5 text-xs">
				<span class="font-medium text-neutral-500">Nama Lengkap</span>
				<span class="font-semibold text-neutral-900">{user?.name || '-'}</span>
			</div>
			<div class="flex items-center justify-between px-3.5 py-2.5 text-xs">
				<span class="font-medium text-neutral-500">Username</span>
				<span class="font-semibold text-neutral-900">@{user?.username || '-'}</span>
			</div>
			<div class="flex items-center justify-between px-3.5 py-2.5 text-xs">
				<span class="font-medium text-neutral-500">Alamat Email</span>
				<span class="font-semibold text-neutral-900">{user?.email || '-'}</span>
			</div>
			<div class="flex items-center justify-between px-3.5 py-2.5 text-xs">
				<span class="font-medium text-neutral-500">Peran / Hak Akses</span>
				<span class="font-semibold text-neutral-900">
					{user?.role ? (USER_ROLE_LABELS[user.role] ?? user.role) : '-'}
				</span>
			</div>
			<div class="flex items-center justify-between px-3.5 py-2.5 text-xs">
				<span class="font-medium text-neutral-500">Penempatan Cabang</span>
				<span class="font-semibold text-neutral-900">
					{user?.location_id ? user.location_id : 'Semua Cabang (Global)'}
				</span>
			</div>
			<div class="flex items-center justify-between px-3.5 py-2.5 text-xs">
				<span class="font-medium text-neutral-500">ID Pengguna</span>
				<span class="font-mono text-[11px] text-neutral-600">{user?.id || '-'}</span>
			</div>
		</div>
	</div>

	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={() => (showProfileModal = false)}>Tutup</Button>
	{/snippet}
</Modal>

<!-- Modal: Ganti Password Mandiri -->
<Modal bind:open={showPasswordModal} title="Ganti Kata Sandi" size="md">
	<form onsubmit={handleSubmitPassword} class="space-y-4" id="form-change-password">
		{#if passwordSuccess}
			<Alert variant="success" dismissible>
				{passwordSuccess}
			</Alert>
		{/if}

		{#if passwordError}
			<Alert variant="error" dismissible>
				{passwordError}
			</Alert>
		{/if}

		<p class="text-xs text-neutral-500">
			Masukkan kata sandi lama Anda dan buat kata sandi baru yang kuat (minimal 6 karakter).
		</p>

		<Input
			id="input-old-password"
			label="Kata Sandi Saat Ini"
			type="password"
			placeholder="Masukkan kata sandi saat ini"
			bind:value={oldPassword}
			showPasswordToggle={true}
			required
		/>

		<Input
			id="input-new-password"
			label="Kata Sandi Baru"
			type="password"
			placeholder="Minimal 6 karakter"
			bind:value={newPassword}
			showPasswordToggle={true}
			required
		/>

		<Input
			id="input-confirm-password"
			label="Konfirmasi Kata Sandi Baru"
			type="password"
			placeholder="Ulangi kata sandi baru"
			bind:value={confirmPassword}
			showPasswordToggle={true}
			required
		/>
	</form>

	{#snippet footer()}
		<Button
			variant="secondary"
			size="sm"
			onclick={() => {
				showPasswordModal = false;
				resetPasswordForm();
			}}
		>
			Batal
		</Button>
		<Button variant="primary" size="sm" loading={passwordLoading} onclick={handleSubmitPassword}>
			Simpan Kata Sandi
		</Button>
	{/snippet}
</Modal>
