import { redirect } from '@sveltejs/kit';
import { isAuthenticated, getUser } from '$lib/stores/auth.svelte';

/**
 * Route Guard untuk seluruh rute internal di bawah (app).
 * 1. Jika user belum login, otomatis dialihkan ke /login.
 * 2. Jika akun ber-role 'customer', dilarang masuk ke Backoffice staf internal.
 */
export function load() {
	if (!isAuthenticated()) {
		redirect(307, '/login');
	}

	const user = getUser();
	if (user?.role === 'customer') {
		redirect(307, '/login?error=forbidden');
	}
}
