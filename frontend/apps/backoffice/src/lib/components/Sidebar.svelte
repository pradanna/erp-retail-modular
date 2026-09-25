<script lang="ts">
	/**
	 * Sidebar — Komponen navigasi vertikal Gen-E Backoffice.
	 *
	 * Fitur:
	 * - Identitas brand Gen-E & logo resmi GEN ENTERPRISE
	 * - Tipografi kompak (text-xs) bernuansa modern enterprise
	 * - Aksen warna primary (Monochrome Obsidian Gen-E) pada status aktif & hover
	 * - Tooltip animasi slide dari kiri ke kanan saat sidebar menciut (collapsed)
	 * - Profil user
	 * - Standar icon Heroicons SVG outline (tanpa emoticon & tanpa sparkle)
	 */
	import { page } from '$app/state';
	import { slide } from 'svelte/transition';
	import { getUser } from '$lib/stores/auth.svelte';
	import { Badge } from '@erp/ui';

	interface Props {
		collapsed?: boolean;
		mobileOpen?: boolean;
		onToggleCollapse?: () => void;
		onCloseMobile?: () => void;
	}

	let { collapsed = false, mobileOpen = false, onToggleCollapse, onCloseMobile }: Props = $props();

	const user = $derived(getUser());

	interface NavItem {
		label: string;
		href: string;
		exact?: boolean;
		badge?: string;
		iconSvg: string;
	}

	interface NavGroup {
		title: string;
		items: NavItem[];
	}

	const navGroups: NavGroup[] = [
		{
			title: 'Menu Utama',
			items: [
				{
					label: 'Dashboard',
					href: '/dashboard',
					exact: true,
					iconSvg:
						'M2.25 12l8.954-8.955c.44-.439 1.152-.439 1.591 0L21.75 12M4.5 9.75v10.125c0 .621.504 1.125 1.125 1.125H9.75v-4.875c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21h4.125c.621 0 1.125-.504 1.125-1.125V9.75M8.25 21h8.25'
				}
			]
		},
		{
			title: 'Data Master',
			items: [
				{
					label: 'Kategori Produk',
					href: '/master/categories',
					iconSvg:
						'M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z'
				},
				{
					label: 'Katalog Produk',
					href: '/master/products',
					iconSvg:
						'M21 7.5l-9-5.25L3 7.5m18 0l-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9'
				},
				{
					label: 'Barcode Produk',
					href: '/master/barcodes',
					iconSvg:
						'M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5zM6.75 6.75h.75v.75h-.75v-.75zM6.75 16.5h.75v.75h-.75v-.75zM16.5 6.75h.75v.75h-.75v-.75zM13.5 13.5h3.75v3.75H13.5V13.5zM13.5 19.5h3.75V21H13.5v-1.5zM19.5 13.5H21v3.75h-1.5V13.5zM19.5 19.5H21V21h-1.5v-1.5zM16.5 16.5h3v3h-3v-3z'
				},
				{
					label: 'Garansi Produk',
					href: '/master/warranties',
					iconSvg:
						'M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z'
				},
				{
					label: 'Cabang & Lokasi',
					href: '/master/locations',
					iconSvg:
						'M13.5 21v-7.5a.75.75 0 01.75-.75h3a.75.75 0 01.75.75V21m-4.5 0H2.36m11.14 0H18m0 0h3.64m-1.39 0V9.349m-16.5 11.65V9.35m0 0a3.001 3.001 0 003.75-.615A2.993 2.993 0 009 9.75c.696 0 1.345-.237 1.865-.635A2.991 2.991 0 0012 9.75c.696 0 1.345-.237 1.865-.635A2.991 2.991 0 0015 9.75c.696 0 1.345-.237 1.865-.635A3.001 3.001 0 0020.625 9.35M3.75 9.35l.937-4.685A1.5 1.5 0 016.155 3.39h11.69a1.5 1.5 0 011.468 1.275l.937 4.685'
				}
			]
		},
		{
			title: 'Inventaris & Stok',
			items: [
				{
					label: 'Stok Cabang',
					href: '/inventory/stocks',
					iconSvg:
						'M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 5.625c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125m16.5 5.625c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125'
				},
				{
					label: 'Mutasi Stok',
					href: '/inventory/transfers',
					iconSvg: 'M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5'
				},
				{
					label: 'Serial & IMEI',
					href: '/inventory/serials',
					iconSvg:
						'M10.5 1.5H8.25A2.25 2.25 0 006 3.75v16.5a2.25 2.25 0 002.25 2.25h7.5A2.25 2.25 0 0018 20.25V3.75a2.25 2.25 0 00-2.25-2.25H13.5m-3 0V3h3V1.5m-3 0h3m-3 18.75h3'
				},
				{
					label: 'Promo Cabang',
					href: '/inventory/price-overrides',
					iconSvg:
						'M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3zM6 6h.008v.008H6V6z'
				}
			]
		},
		{
			title: 'Sistem & Otorisasi',
			items: [
				{
					label: 'Staf & Pengguna',
					href: '/users',
					iconSvg:
						'M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z'
				},
				{
					label: 'Hak Akses Peran',
					href: '/roles',
					iconSvg:
						'M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z'
				},
				{
					label: 'Log Audit Sistem',
					href: '/audit-logs',
					iconSvg:
						'M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z'
				}
			]
		}
	];

	function isItemActive(href: string, exact: boolean = false): boolean {
		const current = page.url.pathname;
		if (exact) {
			return current === href;
		}
		return current === href || current.startsWith(href + '/');
	}

	// State kelompok navigasi yang diciutkan (accordion)
	let collapsedGroups = $state<Record<string, boolean>>({});

	function toggleGroup(title: string) {
		collapsedGroups[title] = !isGroupCollapsed(title);
	}

	function isGroupCollapsed(title: string): boolean {
		if (collapsedGroups[title] !== undefined) {
			return collapsedGroups[title];
		}
		return false;
	}

	function hasActiveItem(items: NavItem[]): boolean {
		return items.some((item) => isItemActive(item.href, item.exact));
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

<!-- Mobile Overlay Backdrop -->
{#if mobileOpen}
	<div
		class="fixed inset-0 z-40 bg-neutral-900/50 backdrop-blur-xs transition-opacity lg:hidden"
		onclick={onCloseMobile}
		onkeydown={(e) => e.key === 'Escape' && onCloseMobile?.()}
		role="button"
		tabindex="0"
		aria-label="Tutup menu navigasi"
	></div>
{/if}

<!-- Sidebar Container -->
<aside
	class="fixed inset-y-0 left-0 z-50 flex flex-col border-r border-neutral-200 bg-white transition-all duration-300 ease-in-out lg:static lg:z-auto {collapsed
		? 'w-18 overflow-visible'
		: 'w-64'} {mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}"
>
	<!-- Brand Header: Logo Resmi Gen-E & GEN ENTERPRISE -->
	<div class="flex h-16 shrink-0 items-center justify-between border-b border-neutral-200 px-3.5">
		<a
			href="/dashboard"
			class="text-decoration-none flex items-center gap-2.5 overflow-hidden focus:outline-hidden"
			onclick={onCloseMobile}
		>
			<img
				src="/brand/gene_mark_black.png"
				alt="Gen-E"
				class="h-8.5 w-8.5 shrink-0 object-contain"
			/>

			{#if !collapsed}
				<div class="min-w-0 flex-1">
					<div class="text-base font-extrabold tracking-tight text-neutral-900">Gen-E</div>
					<p class="truncate text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
						GEN ENTERPRISE
					</p>
				</div>
			{/if}
		</a>

		<!-- Desktop Collapse Toggle Button -->
		{#if onToggleCollapse}
			<button
				type="button"
				class="hidden rounded-lg p-1.5 text-neutral-400 transition-colors hover:bg-primary-50 hover:text-primary-600 focus:outline-hidden lg:block"
				onclick={onToggleCollapse}
				title={collapsed ? 'Perluas Sidebar' : 'Ciutkan Sidebar'}
			>
				<svg
					class="h-4.5 w-4.5 transition-transform duration-200 {collapsed ? 'rotate-180' : ''}"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path d="M15.75 19.5L8.25 12l7.5-7.5" />
				</svg>
			</button>
		{/if}

		<!-- Mobile Close Button -->
		{#if onCloseMobile}
			<button
				type="button"
				class="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 focus:outline-hidden lg:hidden"
				onclick={onCloseMobile}
				aria-label="Tutup sidebar"
			>
				<svg
					class="h-5 w-5"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		{/if}
	</div>

	<!-- Navigation Items -->
	<nav class="flex-1 space-y-4 px-2.5 py-4 {collapsed ? 'overflow-visible' : 'overflow-y-auto'}">
		{#each navGroups as group (group.title)}
			{@const isCollapsed = !collapsed && isGroupCollapsed(group.title)}
			{@const hasActive = hasActiveItem(group.items)}
			<div class="space-y-1">
				{#if !collapsed}
					<button
						type="button"
						onclick={() => toggleGroup(group.title)}
						class="flex w-full items-center justify-between rounded-md px-2.5 py-1 text-[10px] font-bold tracking-wider text-neutral-400 uppercase transition-colors hover:bg-neutral-100/60 hover:text-neutral-700 focus:outline-hidden"
						aria-expanded={!isCollapsed}
					>
						<span
							class="flex items-center gap-1.5 {hasActive ? 'font-extrabold text-neutral-800' : ''}"
						>
							{group.title}
							{#if isCollapsed && hasActive}
								<span class="h-1.5 w-1.5 rounded-full bg-primary-600"></span>
							{/if}
						</span>
						<svg
							class="h-3.5 w-3.5 text-neutral-400 transition-transform duration-200 {isCollapsed
								? '-rotate-90'
								: 'rotate-0'}"
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
				{:else}
					<div class="mx-auto my-1.5 h-px w-6 bg-neutral-200"></div>
				{/if}

				{#if !isCollapsed}
					<div class="space-y-0.5" transition:slide={{ duration: 150 }}>
						{#each group.items as item (item.href)}
							{@const active = isItemActive(item.href, item.exact)}
							<div class="group relative">
								<a
									href={item.href}
									onclick={onCloseMobile}
									class="flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-xs transition-all duration-150 {active
										? 'bg-primary-600 font-semibold text-white shadow-xs'
										: 'font-medium text-neutral-600 hover:bg-primary-50/70 hover:text-primary-700'} {collapsed
										? 'justify-center px-0'
										: ''}"
								>
									<!-- Nav Icon -->
									<svg
										class="h-4.5 w-4.5 shrink-0 transition-colors {active
											? 'text-white'
											: 'text-neutral-400 group-hover:text-primary-600'}"
										viewBox="0 0 24 24"
										fill="none"
										stroke="currentColor"
										stroke-width={active ? '2.25' : '1.75'}
										stroke-linecap="round"
										stroke-linejoin="round"
									>
										<path d={item.iconSvg} />
									</svg>

									{#if !collapsed}
										<span class="flex-1 truncate">{item.label}</span>
										{#if item.badge}
											<span
												class="py-0.2 rounded-full bg-primary-100 px-1.5 text-[10px] font-bold text-primary-700"
											>
												{item.badge}
											</span>
										{/if}
									{/if}
								</a>

								<!-- Tooltip Animasi Kiri ke Kanan saat Collapsed -->
								{#if collapsed}
									<div
										class="pointer-events-none absolute top-1/2 left-full z-50 ml-3.5 -translate-x-2.5 -translate-y-1/2 opacity-0 transition-all duration-200 ease-out group-hover:translate-x-0 group-hover:opacity-100"
									>
										<div
											class="relative flex items-center gap-1.5 rounded-md border border-neutral-800 bg-neutral-900 px-2.5 py-1.5 text-xs font-medium whitespace-nowrap text-white shadow-xl"
										>
											<!-- Caret Triangle pointing left -->
											<div
												class="absolute top-1/2 -left-1 h-2 w-2 -translate-y-1/2 rotate-45 border-b border-l border-neutral-800 bg-neutral-900"
											></div>
											<span class="relative z-10">{item.label}</span>
											{#if item.badge}
												<span
													class="py-0.2 relative z-10 rounded bg-primary-500/30 px-1 text-[10px] font-bold text-primary-300"
												>
													{item.badge}
												</span>
											{/if}
										</div>
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/each}
	</nav>

	<!-- Footer Profile -->
	<div class="shrink-0 border-t border-neutral-200 bg-neutral-50/70 p-2.5">
		{#if !collapsed}
			<div class="flex items-center gap-2.5 px-1">
				<div
					class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-600 text-xs font-bold text-white shadow-2xs"
				>
					{user?.name?.charAt(0).toUpperCase() || 'U'}
				</div>
				<div class="min-w-0 flex-1">
					<p class="truncate text-xs font-semibold text-neutral-900">
						{user?.name || 'Pengguna'}
					</p>
					<div class="mt-0.5 flex items-center gap-1">
						<Badge variant={getRoleBadgeVariant(user?.role)} size="sm">
							{user?.role || 'Staf'}
						</Badge>
					</div>
				</div>
			</div>
		{:else}
			<div class="group relative flex justify-center">
				<div
					class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-600 text-xs font-bold text-white shadow-2xs"
				>
					{user?.name?.charAt(0).toUpperCase() || 'U'}
				</div>
				<!-- Tooltip User saat Collapsed -->
				<div
					class="pointer-events-none absolute top-1/2 left-full z-50 ml-3.5 -translate-x-2.5 -translate-y-1/2 opacity-0 transition-all duration-200 ease-out group-hover:translate-x-0 group-hover:opacity-100"
				>
					<div
						class="relative flex flex-col rounded-md border border-neutral-800 bg-neutral-900 px-2.5 py-1.5 text-xs font-medium whitespace-nowrap text-white shadow-xl"
					>
						<div
							class="absolute top-1/2 -left-1 h-2 w-2 -translate-y-1/2 rotate-45 border-b border-l border-neutral-800 bg-neutral-900"
						></div>
						<span class="relative z-10 font-semibold">{user?.name || 'Pengguna'}</span>
						<span class="relative z-10 text-[10px] text-neutral-400 capitalize"
							>{user?.role || 'Staf'}</span
						>
					</div>
				</div>
			</div>
		{/if}
	</div>
</aside>
