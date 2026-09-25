/**
 * Base HTTP Client — Wrapper fetch() yang menstandarkan komunikasi ke backend REST API.
 *
 * Konsep:
 * Daripada setiap halaman menulis fetch() sendiri-sendiri (duplikasi kode),
 * kita buat satu fungsi pusat yang menangani:
 * 1. Base URL backend
 * 2. Header Authorization otomatis (jika ada token)
 * 3. Parsing JSON response
 * 4. Error handling terpusat (lempar ApiError jika non-2xx)
 *
 * Analoginya: fetchApi() adalah "resepsionis" yang mengirimkan surat ke backend.
 * Dia tahu alamat backend, selalu melampirkan kartu identitas (token),
 * dan memberi tahu jika surat balasan berisi berita buruk (error).
 */

import { ApiError } from '@erp/types';
import type { ApiErrorBody } from '@erp/types';

/** Konfigurasi base URL backend. Bisa di-override via environment variable. */
const API_BASE_URL = 'http://localhost:8088';

/** Opsi tambahan untuk fetchApi. */
interface FetchOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  token?: string | null;
  headers?: Record<string, string>;
}

/**
 * Fungsi utama untuk memanggil backend REST API.
 *
 * @param path — Endpoint relatif (contoh: '/api/v1/auth/login')
 * @param options — Method, body, token, dan header tambahan
 * @returns Parsed JSON response bertipe T
 * @throws ApiError jika response bukan 2xx
 *
 * @example
 * ```ts
 * const data = await fetchApi<LoginResponse>('/api/v1/auth/login', {
 *   method: 'POST',
 *   body: { username: 'admin', password: '123456' },
 * });
 * ```
 */
export async function fetchApi<T>(path: string, options: FetchOptions = {}): Promise<T> {
  const { method = 'GET', body, token, headers = {} } = options;

  const isFormData = typeof FormData !== 'undefined' && body instanceof FormData;

  // Susun header request
  const requestHeaders: Record<string, string> = {
    ...(!isFormData ? { 'Content-Type': 'application/json' } : {}),
    ...headers,
  };

  // Lampirkan token JWT di header Authorization jika tersedia
  if (token) {
    requestHeaders['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers: requestHeaders,
    // Hanya sertakan body untuk method non-GET
    body: body !== undefined ? (isFormData ? (body as BodyInit) : JSON.stringify(body)) : undefined,
  });

  // Jika response bukan 2xx, parse error body dan lempar ApiError
  if (!response.ok) {
    let errorMessage = `HTTP ${response.status}`;
    try {
      const errorBody = (await response.json()) as ApiErrorBody;
      errorMessage = errorBody.error;
    } catch {
      // Jika body bukan JSON, gunakan status text
      errorMessage = response.statusText || errorMessage;
    }
    throw new ApiError(response.status, errorMessage);
  }

  // Parse dan kembalikan response body sebagai tipe T
  return (await response.json()) as T;
}
