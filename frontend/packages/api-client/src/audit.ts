/**
 * Audit API Client — Type-safe HTTP functions untuk modul Shared Audit Log.
 */

import { fetchApi } from './client.js';
import type { AuditLogResponse, AuditLogListParams } from '@erp/types';

export async function listAuditLogs(
  token: string,
  params?: AuditLogListParams,
): Promise<{ data: AuditLogResponse[]; meta: { page: number; limit: number; total_items: number; total_pages: number } }> {
  const query = new URLSearchParams();
  if (params?.module) query.set('module', params.module);
  if (params?.action) query.set('action', params.action);
  if (params?.user_id) query.set('user_id', params.user_id);
  if (params?.start_date) query.set('start_date', params.start_date);
  if (params?.end_date) query.set('end_date', params.end_date);
  if (params?.page) query.set('page', String(params.page));
  if (params?.limit) query.set('limit', String(params.limit));

  const qs = query.toString() ? `?${query.toString()}` : '';
  return fetchApi<{ data: AuditLogResponse[]; meta: { page: number; limit: number; total_items: number; total_pages: number } }>(
    `/api/shared/audit-logs${qs}`,
    { token },
  );
}
