<script lang="ts">
	/**
	 * Layout induk untuk seluruh rute terproteksi (app).
	 * Mengintegrasikan Sidebar navigasi vertikal, Topbar header, dan area konten utama.
	 */
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Topbar from '$lib/components/Topbar.svelte';

	let { children } = $props();

	let collapsed = $state(false);
	let mobileOpen = $state(false);

	function toggleCollapse() {
		collapsed = !collapsed;
	}

	function openMobile() {
		mobileOpen = true;
	}

	function closeMobile() {
		mobileOpen = false;
	}
</script>

<div class="flex h-screen overflow-hidden bg-neutral-50 font-sans text-neutral-900 antialiased">
	<!-- Sidebar Navigasi Utama -->
	<Sidebar {collapsed} {mobileOpen} onToggleCollapse={toggleCollapse} onCloseMobile={closeMobile} />

	<!-- Area Konten Utama & Header -->
	<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
		<!-- Topbar Header -->
		<Topbar onOpenMobile={openMobile} />

		<!-- Konten Halaman (Scrollable) -->
		<main class="flex-1 overflow-y-auto p-4 sm:p-6 lg:p-8">
			{@render children()}
		</main>
	</div>
</div>
