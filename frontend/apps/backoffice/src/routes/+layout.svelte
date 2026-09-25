<script lang="ts">
	/**
	 * Root Layout — Inisialisasi auth dan render children.
	 * Layout ini adalah "pembungkus terluar" yang dijalankan untuk SEMUA halaman.
	 */
	import './layout.css';
	import { initAuth, isLoading } from '$lib/stores/auth.svelte';
	import { ToastContainer } from '@erp/ui';
	import { onMount } from 'svelte';

	let { children } = $props();

	let initialized = $state(false);

	onMount(async () => {
		await initAuth();
		initialized = true;
	});
</script>

<svelte:head>
	<title>ERP Retail — Backoffice</title>
</svelte:head>

<!-- Wadah Notifikasi Melayang Global (Toast) -->
<ToastContainer />

{#if !initialized || isLoading()}
	<!-- Loading screen saat inisialisasi auth -->
	<div class="flex h-screen items-center justify-center bg-neutral-50">
		<div class="flex flex-col items-center gap-4">
			<div
				class="h-10 w-10 animate-spin rounded-full border-4 border-primary-200 border-t-primary-600"
			></div>
			<p class="text-sm font-medium text-neutral-500">Memuat sistem...</p>
		</div>
	</div>
{:else}
	{@render children()}
{/if}
