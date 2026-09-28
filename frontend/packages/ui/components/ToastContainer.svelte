<script lang="ts">
	/**
	 * ToastContainer — Wadah penampung notifikasi melayang di pojok layar.
	 *
	 * Diletakkan satu kali di root layout aplikasi.
	 * Menampilkan daftar toast secara vertikal dengan urutan alami.
	 */
	import { toast } from './toast.svelte.ts';
	import Toast from './Toast.svelte';
	import { portal } from '../actions/portal.js';
</script>

<div
	use:portal
	class="pointer-events-none fixed top-4 right-4 z-toast z-[99999] flex w-full max-w-sm flex-col gap-2.5 sm:top-6 sm:right-6"
	style="z-index: 99999 !important; isolation: isolate;"
	aria-live="polite"
>
	{#each toast.items as item (item.id)}
		<Toast {item} onclose={() => toast.remove(item.id)} />
	{/each}
</div>
