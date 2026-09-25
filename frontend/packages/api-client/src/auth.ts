/**
 * Auth API Client — Fungsi type-safe untuk memanggil endpoint autentikasi backend.
 *
 * Setiap fungsi di sini merepresentasikan satu endpoint di backend Go.
 * Tipe request/response diimpor dari @erp/types (single source of truth).
 */

import { fetchApi } from './client.js';
import type {
  LoginRequest,
  LoginResponse,
  UserResponse,
  VerifyPasswordResponse,
  ChangePasswordRequest,
  CreateUserRequest,
} from '@erp/types';

/**
 * Login ke sistem ERP.
 * Endpoint: POST /api/v1/auth/login
 */
export async function login(req: LoginRequest): Promise<LoginResponse> {
  return fetchApi<LoginResponse>('/api/v1/auth/login', {
    method: 'POST',
    body: req,
  });
}

/**
 * Ambil profil user yang sedang login berdasarkan JWT token.
 * Endpoint: GET /api/v1/auth/me
 */
export async function getMe(token: string): Promise<UserResponse> {
  return fetchApi<UserResponse>('/api/v1/auth/me', { token });
}

/**
 * Verifikasi password untuk Step-up Authentication.
 * Digunakan sebelum aksi kritikal (approve PO, transfer stok, dll).
 * Endpoint: POST /api/v1/auth/verify-password
 */
export async function verifyPassword(
  token: string,
  password: string,
): Promise<VerifyPasswordResponse> {
  return fetchApi<VerifyPasswordResponse>('/api/v1/auth/verify-password', {
    method: 'POST',
    body: { password },
    token,
  });
}

/**
 * Ganti password mandiri user.
 * Endpoint: POST /api/v1/auth/change-password
 */
export async function changePassword(
  token: string,
  req: ChangePasswordRequest,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>('/api/v1/auth/change-password', {
    method: 'POST',
    body: req,
    token,
  });
}

/**
 * Daftar semua staf (hanya superadmin/owner).
 * Endpoint: GET /api/v1/users
 */
export async function listUsers(
  token: string,
  filters?: { role?: string; location_id?: string; active_only?: boolean },
): Promise<UserResponse[]> {
  const params = new URLSearchParams();
  if (filters?.role) params.set('role', filters.role);
  if (filters?.location_id) params.set('location_id', filters.location_id);
  if (filters?.active_only) params.set('active_only', 'true');

  const query = params.toString();
  const path = `/api/v1/users${query ? `?${query}` : ''}`;

  return fetchApi<UserResponse[]>(path, { token });
}

/**
 * Buat staf baru (hanya superadmin/owner).
 * Endpoint: POST /api/v1/users
 */
export async function createUser(token: string, req: CreateUserRequest): Promise<UserResponse> {
  return fetchApi<UserResponse>('/api/v1/users', {
    method: 'POST',
    body: req,
    token,
  });
}

/**
 * Ambil detail staf berdasarkan ID (hanya superadmin/owner).
 * Endpoint: GET /api/v1/users/:id
 */
export async function getUser(token: string, id: string): Promise<UserResponse> {
  return fetchApi<UserResponse>(`/api/v1/users/${id}`, { token });
}

/**
 * Toggle status aktif/nonaktif staf (hanya superadmin/owner).
 * Endpoint: PATCH /api/v1/users/:id/status
 */
export async function setUserStatus(
  token: string,
  id: string,
  isActive: boolean,
): Promise<UserResponse> {
  return fetchApi<UserResponse>(`/api/v1/users/${id}/status`, {
    method: 'PATCH',
    body: { is_active: isActive },
    token,
  });
}

/**
 * Ambil matriks peran dan daftar izin granular (PBAC).
 * Endpoint: GET /api/v1/roles/matrix
 */
export async function getRolePermissionsMatrix(token: string): Promise<import('@erp/types').RolePermissionsMatrix> {
  return fetchApi<import('@erp/types').RolePermissionsMatrix>('/api/v1/roles/matrix', { token });
}

/**
 * Perbarui daftar izin yang dimiliki oleh peran tertentu (PBAC).
 * Endpoint: PUT /api/v1/roles/:role/permissions
 */
export async function updateRolePermissions(
  token: string,
  role: string,
  permissions: string[],
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/api/v1/roles/${role}/permissions`, {
    method: 'PUT',
    body: { permissions },
    token,
  });
}

/**
 * Ambil daftar seluruh izin kapabilitas granular yang tersedia di sistem.
 * Endpoint: GET /api/v1/permissions
 */
export async function listPermissions(token: string): Promise<import('@erp/types').Permission[]> {
  return fetchApi<import('@erp/types').Permission[]>('/api/v1/permissions', { token });
}

