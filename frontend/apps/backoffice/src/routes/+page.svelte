<script lang="ts">
	/**
	 * Root Page — Redirect handler.
	 * Halaman "/" langsung redirect ke dashboard atau login tergantung auth state.
	 */
	import { isAuthenticated } from '$lib/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';

	onMount(() => {
		if (isAuthenticated()) {
			goto(resolve('/dashboard'));
		} else {
			goto(resolve('/login'));
		}
	});
</script>

<div class="flex h-screen items-center justify-center bg-neutral-50">
	<div
		class="h-10 w-10 animate-spin rounded-full border-4 border-primary-200 border-t-primary-600"
	></div>
</div>
