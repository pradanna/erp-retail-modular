/**
 * Toast Notification Store — Svelte 5 Runes ($state).
 *
 * Mengelola antrean notifikasi melayang (toasts) di seluruh aplikasi secara reaktif.
 * Menyediakan helper praktis:
 * - toast.success("Pesan berhasil")
 * - toast.error("Pesan error")
 * - toast.warning("Pesan peringatan")
 * - toast.info("Pesan informasi")
 */

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface ToastItem {
	id: string;
	type: ToastType;
	message: string;
	title?: string;
	duration?: number; // ms, default: 4000
}

export interface ToastOptions {
	title?: string;
	duration?: number;
}

let toasts = $state<ToastItem[]>([]);

function addToast(type: ToastType, message: string, options?: ToastOptions): string {
	const id = Math.random().toString(36).substring(2, 9);
	const duration = options?.duration ?? 4000;

	const newItem: ToastItem = {
		id,
		type,
		message,
		title: options?.title,
		duration
	};

	toasts = [...toasts, newItem];

	if (duration > 0 && typeof window !== 'undefined') {
		setTimeout(() => {
			removeToast(id);
		}, duration);
	}

	return id;
}

function removeToast(id: string): void {
	toasts = toasts.filter((t) => t.id !== id);
}

function clearAll(): void {
	toasts = [];
}

export const toast = {
	get items(): ToastItem[] {
		return toasts;
	},
	success(message: string, options?: ToastOptions): string {
		return addToast('success', message, options);
	},
	error(message: string, options?: ToastOptions): string {
		return addToast('error', message, options);
	},
	info(message: string, options?: ToastOptions): string {
		return addToast('info', message, options);
	},
	warning(message: string, options?: ToastOptions): string {
		return addToast('warning', message, options);
	},
	remove(id: string): void {
		removeToast(id);
	},
	clear(): void {
		clearAll();
	}
};
