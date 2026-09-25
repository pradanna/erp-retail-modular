<script lang="ts">
	/**
	 * Halaman Log Audit Sistem (System Audit Trail)
	 *
	 * Menampilkan rekam jejak seluruh aksi penting yang dilakukan oleh staf, admin,
	 * dan proses otomatis sistem demi akuntabilitas dan transparansi operasional.
	 */
	import { onMount } from 'svelte';
	import { getToken } from '$lib/stores/auth.svelte';
	import { listAuditLogs } from '@erp/api-client';
	import type { AuditLogResponse } from '@erp/types';
	import {
		Button,
		Table,
		Pagination,
		Modal,
		Badge,
		Select2,
		type Select2Option,
		toast
	} from '@erp/ui';

	// ── State Utama ───────────────────────────────────────────────────────────
	let logs = $state<AuditLogResponse[]>([]);
	let loading = $state(false);
	let currentPage = $state(1);
	let totalPages = $state(1);
	let totalItems = $state(0);
	const pageSize = 15;

	// ── Filter State ──────────────────────────────────────────────────────────
	let selectedModule = $state<string>('');
	let startDate = $state<string>('');
	let endDate = $state<string>('');

	const moduleFilterOptions: Select2Option[] = [
		{ value: '', label: 'Semua Modul' },
		{ value: 'inventory', label: 'Inventory (Stok & Opname)' },
		{ value: 'purchasing', label: 'Purchasing (Penerimaan Barang)' },
		{ value: 'sales', label: 'Sales (Penjualan Kasir)' },
		{ value: 'auth', label: 'Auth & Pengguna' }
	];

	// ── Modal Detail JSON ─────────────────────────────────────────────────────
	let showDetailModal = $state(false);
	let selectedLog = $state<AuditLogResponse | null>(null);

	async function loadLogs(page: number = 1) {
		loading = true;
		currentPage = page;
		try {
			const token = getToken();
			if (!token) return;

			const res = await listAuditLogs(token, {
				module: selectedModule || undefined,
				start_date: startDate || undefined,
				end_date: endDate || undefined,
				page,
				limit: pageSize
			});

			logs = res.data;
			totalItems = res.meta.total_items;
			totalPages = res.meta.total_pages;
		} catch (err: unknown) {
			console.error(err);
			toast.error('Gagal memuat log audit sistem.');
		} finally {
			loading = false;
		}
	}

	function openDetail(item: AuditLogResponse) {
		selectedLog = item;
		showDetailModal = true;
	}

	function formatDateTime(iso: string): string {
		try {
			const d = new Date(iso);
			return d.toLocaleString('id-ID', {
				day: 'numeric',
				month: 'short',
				year: 'numeric',
				hour: '2-digit',
				minute: '2-digit',
				second: '2-digit'
			});
		} catch {
			return iso;
		}
	}

	function formatJsonDetails(raw?: string | null): string {
		if (!raw) return 'Tidak ada data rinci tambahan.';
		try {
			const parsed = JSON.parse(raw);
			return JSON.stringify(parsed, null, 2);
		} catch {
			return raw;
		}
	}

	function getModuleBadgeVariant(mod: string): 'indigo' | 'warning' | 'success' | 'default' {
		switch (mod) {
			case 'inventory':
				return 'indigo';
			case 'purchasing':
				return 'warning';
			case 'sales':
				return 'success';
			default:
				return 'default';
		}
	}

	onMount(() => {
		loadLogs(1);
	});
</script>

<svelte:head>
	<title>Log Audit Sistem — ERP Retail Modular</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header Halaman -->
	<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h1 class="text-xl font-bold tracking-tight text-neutral-900 sm:text-2xl">
				Log Audit & Jejak Aktivitas
			</h1>
			<p class="text-xs text-neutral-500 sm:text-sm">
				Rekam jejak seluruh aksi penting yang dilakukan staf dan admin untuk memastikan akuntabilitas operasional toko.
			</p>
		</div>

		<Button variant="outline" size="sm" onclick={() => loadLogs(currentPage)} loading={loading}>
			<!-- Refresh Icon -->
			<svg class="h-4 w-4 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
				<path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
			</svg>
			Segarkan Log
		</Button>
	</div>

	<!-- Toolbar Filter -->
	<div class="flex flex-col gap-3 rounded-xl border border-neutral-200 bg-white p-3.5 shadow-2xs sm:flex-row sm:items-center sm:justify-between">
		<div class="flex flex-wrap items-center gap-3">
			<div class="w-full sm:w-60">
				<Select2
					options={moduleFilterOptions}
					value={selectedModule}
					placeholder="Semua Modul"
					onchange={(val) => { selectedModule = String(val); loadLogs(1); }}
				/>
			</div>

			<!-- Rentang Tanggal Filter -->
			<div class="flex items-center gap-2 text-xs">
				<span class="text-neutral-400">Dari:</span>
				<input
					type="date"
					bind:value={startDate}
					onchange={() => loadLogs(1)}
					class="rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:outline-hidden"
				/>
				<span class="text-neutral-400">Sampai:</span>
				<input
					type="date"
					bind:value={endDate}
					onchange={() => loadLogs(1)}
					class="rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs text-neutral-800 shadow-2xs focus:border-neutral-900 focus:outline-hidden"
				/>
			</div>
		</div>

		{#if selectedModule || startDate || endDate}
			<Button
				variant="outline"
				size="sm"
				onclick={() => { selectedModule = ''; startDate = ''; endDate = ''; loadLogs(1); }}
			>
				Reset Filter
			</Button>
		{/if}
	</div>

	<!-- Tabel Log Audit -->
	<div class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
		<Table empty={logs.length === 0} emptyMessage={loading ? 'Memuat log audit sistem...' : 'Tidak ada aktivitas tercatat yang sesuai filter.'}>
			<thead>
				<tr class="border-b border-neutral-200 bg-neutral-50/80 text-left text-2xs font-semibold tracking-wider text-neutral-600 uppercase">
					<th class="px-4 py-3">Waktu</th>
					<th class="px-4 py-3">Pengguna / Staf</th>
					<th class="px-4 py-3">Modul & Aksi</th>
					<th class="px-4 py-3">Ringkasan Aktivitas</th>
					<th class="px-4 py-3 text-right">Rincian</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-neutral-100">
				{#each logs as logItem}
					<tr class="hover:bg-neutral-50/60 transition-colors">
						<td class="px-4 py-3 text-2xs text-neutral-500 whitespace-nowrap font-mono">
							{formatDateTime(logItem.created_at)}
						</td>
						<td class="px-4 py-3 whitespace-nowrap">
							<div class="font-medium text-neutral-900 text-xs">{logItem.user_name}</div>
							<span class="rounded bg-neutral-100 px-1.5 py-0.2 text-3xs font-mono text-neutral-600 uppercase">
								{logItem.user_role}
							</span>
						</td>
						<td class="px-4 py-3 whitespace-nowrap">
							<Badge variant={getModuleBadgeVariant(logItem.module)} size="sm">
								{logItem.module}
							</Badge>
							<div class="font-mono text-3xs text-neutral-400 mt-1">{logItem.action}</div>
						</td>
						<td class="px-4 py-3 text-xs text-neutral-700">
							{logItem.summary}
						</td>
						<td class="px-4 py-3 text-right whitespace-nowrap">
							<Button
								variant="outline"
								size="sm"
								onclick={() => openDetail(logItem)}
								disabled={!logItem.details}
							>
								Lihat JSON
							</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	</div>

	<!-- Pagination -->
	{#if totalPages > 1}
		<div class="flex items-center justify-between">
			<span class="text-xs text-neutral-500">
				Menampilkan {(currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, totalItems)} dari {totalItems} log aktivitas
			</span>
			<Pagination
				page={currentPage}
				{totalPages}
				{totalItems}
				limit={pageSize}
				onPageChange={(p) => loadLogs(p)}
			/>
		</div>
	{/if}
</div>

<!-- Modal Rincian JSON Audit Log -->
{#if showDetailModal && selectedLog}
	<Modal bind:open={showDetailModal} title="Rincian Payload Audit Log" size="xl">
		<div class="space-y-3">
			<div class="rounded-xl border border-neutral-200 bg-neutral-50/70 p-3 text-xs">
				<div class="flex items-center justify-between">
					<span class="font-semibold text-neutral-900">{selectedLog.action}</span>
					<span class="font-mono text-neutral-500">{formatDateTime(selectedLog.created_at)}</span>
				</div>
				<p class="mt-1 text-neutral-600">{selectedLog.summary}</p>
			</div>

			<div>
				<span class="text-2xs font-semibold uppercase tracking-wider text-neutral-400">Data Rinci (JSON Payload):</span>
				<pre class="mt-1.5 max-h-96 overflow-y-auto rounded-xl border border-neutral-800 bg-neutral-950 p-4 font-mono text-2xs text-emerald-400 shadow-inner">
{formatJsonDetails(selectedLog.details)}
				</pre>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex items-center justify-end">
				<Button variant="primary" size="sm" onclick={() => (showDetailModal = false)}>
					Tutup
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}
