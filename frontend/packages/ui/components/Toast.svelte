<script lang="ts">
	/**
	 * Toast — Komponen kartu notifikasi melayang individual.
	 *
	 * Props:
	 * - item: Objek ToastItem (id, type, message, title, duration)
	 * - onclose: Callback opsional saat tombol silang diklik
	 */
	import { fly } from 'svelte/transition';
	import type { ToastItem, ToastType } from './toast.svelte.ts';

	interface Props {
		item: ToastItem;
		onclose?: () => void;
	}

	let { item, onclose }: Props = $props();

	const variantStyles: Record<
		ToastType,
		{ iconBg: string; iconColor: string; borderAccent: string }
	> = {
		success: {
			iconBg: 'bg-emerald-50',
			iconColor: 'text-emerald-600',
			borderAccent: 'border-l-emerald-500'
		},
		error: {
			iconBg: 'bg-rose-50',
			iconColor: 'text-rose-600',
			borderAccent: 'border-l-rose-500'
		},
		warning: {
			iconBg: 'bg-amber-50',
			iconColor: 'text-amber-600',
			borderAccent: 'border-l-amber-500'
		},
		info: {
			iconBg: 'bg-sky-50',
			iconColor: 'text-sky-600',
			borderAccent: 'border-l-sky-500'
		}
	};

	const currentStyle = $derived(variantStyles[item.type] ?? variantStyles.info);
</script>

<div
	role="status"
	aria-live="polite"
	transition:fly={{ y: -12, duration: 200 }}
	class="pointer-events-auto flex w-full items-start gap-3 rounded-xl border border-neutral-200/90 border-l-4 bg-white p-3.5 shadow-lg ring-1 ring-black/5 transition-all {currentStyle.borderAccent}"
>
	<!-- Icon Status Semantik (Heroicons) -->
	<div
		class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg {currentStyle.iconBg} {currentStyle.iconColor}"
	>
		{#if item.type === 'success'}
			<!-- Heroicons CheckCircle 20x20 -->
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
				<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
			</svg>
		{:else if item.type === 'error'}
			<!-- Heroicons XCircle 20x20 -->
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
				<path stroke-linecap="round" stroke-linejoin="round" d="m9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
			</svg>
		{:else if item.type === 'warning'}
			<!-- Heroicons ExclamationTriangle 20x20 -->
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
				/>
			</svg>
		{:else}
			<!-- Heroicons InformationCircle 20x20 -->
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.75">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z"
				/>
			</svg>
		{/if}
	</div>

	<!-- Konten Teks Pesan -->
	<div class="flex-1 pt-0.5">
		{#if item.title}
			<h4 class="text-sm font-semibold text-neutral-900 leading-tight mb-0.5">{item.title}</h4>
		{/if}
		<p class="text-sm text-neutral-700 leading-snug">{item.message}</p>
	</div>

	<!-- Tombol Tutup Manual (Silang) -->
	{#if onclose}
		<button
			type="button"
			onclick={onclose}
			class="-mr-1 -mt-1 rounded-lg p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600 focus:outline-hidden"
			aria-label="Tutup notifikasi"
		>
			<!-- Heroicons XMark 16x16 -->
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
			</svg>
		</button>
	{/if}
</div>
