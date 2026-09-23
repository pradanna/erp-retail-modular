/**
 * Auth Store — State management autentikasi untuk Backoffice SPA.
 *
 * Menggunakan Svelte 5 runes ($state) untuk reactive state.
 * Token JWT disimpan di localStorage agar session tetap terjaga saat refresh browser.
 *
 * Alur:
 * 1. User login → token dan data user disimpan di store + localStorage
 * 2. Setiap navigasi → auth guard cek apakah token ada
 * 3. User logout → hapus token dari store + localStorage → redirect ke /login
 *
 * Analoginya: Auth store seperti "kartu identitas" yang selalu dibawa user.
 * Setiap pintu (halaman) mengecek kartu ini sebelum membuka akses.
 */

import { login as apiLogin, getMe } from '@erp/api-client';
import type { UserResponse } from '@erp/types';
import { ApiError } from '@erp/types';

const TOKEN_KEY = 'erp_auth_token';

/** Reactive auth state menggunakan Svelte 5 runes. */
let token = $state<string | null>(null);
let user = $state<UserResponse | null>(null);
let loading = $state(false);
let error = $state<string | null>(null);

/**
 * Inisialisasi auth state dari localStorage saat app pertama kali dimuat.
 * Dipanggil sekali di root layout.
 */
export async function initAuth(): Promise<void> {
	if (typeof window === 'undefined') return;

	const savedToken = localStorage.getItem(TOKEN_KEY);
	if (!savedToken) return;

	token = savedToken;
	loading = true;

	try {
		const profile = await getMe(savedToken);
		user = profile;
	} catch {
		// Token expired atau invalid → bersihkan
		token = null;
		user = null;
		localStorage.removeItem(TOKEN_KEY);
	} finally {
		loading = false;
	}
}

/**
 * Login ke sistem ERP.
 * @returns true jika berhasil, false jika gagal
 */
export async function login(username: string, password: string): Promise<boolean> {
	loading = true;
	error = null;

	try {
		const response = await apiLogin({ username, password });
		token = response.token;
		user = response.user;
		localStorage.setItem(TOKEN_KEY, response.token);
		return true;
	} catch (err: unknown) {
		if (err instanceof ApiError) {
			error = err.message;
		} else if (err instanceof Error) {
			error = err.message;
		} else {
			error = 'Terjadi kesalahan yang tidak terduga';
		}
		return false;
	} finally {
		loading = false;
	}
}

/**
 * Logout — bersihkan semua state autentikasi.
 */
export function logout(): void {
	token = null;
	user = null;
	error = null;
	localStorage.removeItem(TOKEN_KEY);
}

/**
 * Getter functions — karena $state tidak bisa di-export langsung sebagai reactive,
 * kita ekspos via getter functions yang akan di-subscribe oleh komponen.
 */
export function getToken(): string | null {
	return token;
}

export function getUser(): UserResponse | null {
	return user;
}

export function isAuthenticated(): boolean {
	return token !== null && user !== null;
}

export function isLoading(): boolean {
	return loading;
}

export function getError(): string | null {
	return error;
}

export function clearError(): void {
	error = null;
}
