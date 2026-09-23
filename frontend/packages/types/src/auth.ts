/**
 * Auth Types — Single Source of Truth untuk tipe data autentikasi.
 *
 * Tipe-tipe di sini adalah mirror 1:1 dari backend Go DTO
 * (lihat: backend/internal/shared/auth/handler.go).
 * Frontend TIDAK BOLEH mendefinisikan tipe auth di tempat lain.
 */

// ── Roles ──────────────────────────────────────────────────────────────────
/** Peran yang tersedia di sistem ERP. */
export type UserRole = 'owner' | 'superadmin' | 'admin' | 'cashier' | 'warehouse' | 'customer';

/** Label display ramah untuk setiap role. */
export const USER_ROLE_LABELS: Record<UserRole, string> = {
  owner: 'Owner',
  superadmin: 'Super Admin',
  admin: 'Admin',
  cashier: 'Kasir',
  warehouse: 'Gudang',
  customer: 'Customer',
};

// ── Request DTOs ───────────────────────────────────────────────────────────

/** Body untuk POST /api/v1/auth/login */
export interface LoginRequest {
  username: string;
  password: string;
}

/** Body untuk POST /api/v1/auth/verify-password (Step-up Auth) */
export interface VerifyPasswordRequest {
  password: string;
}

/** Body untuk POST /api/v1/auth/change-password */
export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

/** Body untuk POST /api/v1/users */
export interface CreateUserRequest {
  name: string;
  username: string;
  email: string;
  password: string;
  role: UserRole;
  location_id?: string | undefined;
}

/** Body untuk PATCH /api/v1/users/:id/status */
export interface SetUserStatusRequest {
  is_active: boolean;
}

// ── Response DTOs ──────────────────────────────────────────────────────────

/** Representasi publik entitas User (tanpa password hash). */
export interface UserResponse {
  id: string;
  name: string;
  username: string;
  email: string;
  role: UserRole;
  location_id?: string | undefined;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

/** Response setelah login berhasil. */
export interface LoginResponse {
  token: string;
  user: UserResponse;
}

/** Response Step-up Authentication. */
export interface VerifyPasswordResponse {
  verified: boolean;
}
