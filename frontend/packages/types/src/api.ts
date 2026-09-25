/**
 * API Types — Tipe generik untuk komunikasi HTTP ke backend.
 */

/** Error yang dikembalikan oleh backend (format JSON: { "error": "pesan" }). */
export interface ApiErrorBody {
  error: string;
}

/**
 * Custom Error class untuk error dari API backend.
 * Menyimpan HTTP status code dan pesan error dari server.
 *
 * Penggunaan:
 * ```ts
 * try {
 *   await login(req);
 * } catch (err) {
 *   if (err instanceof ApiError && err.status === 401) {
 *     // Handle unauthorized
 *   }
 * }
 * ```
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

/** Opsi generik untuk komponen form dropdown (Select). */
export interface SelectOption {
  value: string | number;
  label: string;
  disabled?: boolean;
}
