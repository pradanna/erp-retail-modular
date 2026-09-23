<script lang="ts">
	/**
	 * Dashboard — Halaman utama setelah user berhasil login.
	 * Menggunakan komponen @erp/ui (Card, Button).
	 */
	import { getUser, logout } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { Button, Card } from '@erp/ui';

	const user = getUser();

	async function handleLogout() {
		logout();
		await goto(resolve('/login'));
	}
</script>

<svelte:head>
	<title>Dashboard — ERP Retail</title>
</svelte:head>

<div class="min-h-screen bg-neutral-50 p-8">
	<div class="mx-auto max-w-4xl space-y-6">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-bold tracking-tight text-neutral-900">Dashboard</h1>
				<p class="text-sm text-neutral-500">Selamat datang di sistem ERP Retail Backoffice</p>
			</div>
			<Button variant="secondary" onclick={handleLogout}>Keluar (Logout)</Button>
		</div>

		<Card padding="lg">
			<h2 class="text-lg font-semibold text-neutral-800">Informasi Sesi Login</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 text-sm sm:grid-cols-2">
				<div>
					<span class="text-neutral-500">Nama Pengguna:</span>
					<p class="font-medium text-neutral-900">{user?.name || '-'}</p>
				</div>
				<div>
					<span class="text-neutral-500">Username:</span>
					<p class="font-mono font-medium text-neutral-900">{user?.username || '-'}</p>
				</div>
				<div>
					<span class="text-neutral-500">Email:</span>
					<p class="font-medium text-neutral-900">{user?.email || '-'}</p>
				</div>
				<div>
					<span class="text-neutral-500">Peran (Role):</span>
					<p class="font-semibold text-primary-600 uppercase">{user?.role || '-'}</p>
				</div>
			</div>
		</Card>
	</div>
</div>
