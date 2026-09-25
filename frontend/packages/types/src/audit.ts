/**
 * Tipe data untuk shared context: Audit Log
 */

export interface AuditLogResponse {
  id: string;
  user_id?: string | null | undefined;
  user_name: string;
  user_role: string;
  action: string;
  module: string;
  target_type?: string | null | undefined;
  target_id?: string | null | undefined;
  summary: string;
  details?: string | null | undefined;
  ip_address?: string | null | undefined;
  created_at: string;
}

export interface AuditLogListParams {
  module?: string | undefined;
  action?: string | undefined;
  user_id?: string | undefined;
  start_date?: string | undefined;
  end_date?: string | undefined;
  page?: number | undefined;
  limit?: number | undefined;
}
