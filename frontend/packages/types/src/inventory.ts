/**
 * Inventory Types — Single Source of Truth untuk tipe data modul Inventory.
 *
 * Seluruh interface di sini adalah cerminan 1:1 dari DTO backend Go
 * (lihat: backend/internal/modules/inventory/interfaces/dto.go).
 */

// ── Enums & Const Types ──────────────────────────────────────────────────

export type ProductStatus = 'draft' | 'active' | 'archived';

export const PRODUCT_STATUS_LABELS: Record<ProductStatus, string> = {
  draft: 'Draft',
  active: 'Aktif',
  archived: 'Diarsipkan',
};

export type LocationType = 'physical' | 'online';

export const LOCATION_TYPE_LABELS: Record<LocationType, string> = {
  physical: 'Toko / Cabang Fisik',
  online: 'Gudang Online Storefront',
};

export type SerialStatus =
  | 'available'
  | 'reserved'
  | 'sold'
  | 'defective'
  | 'returned'
  | 'tersedia'
  | 'terjual'
  | 'retur';

export const SERIAL_STATUS_LABELS: Record<SerialStatus, string> = {
  available: 'Tersedia',
  reserved: 'Dipesan',
  sold: 'Terjual',
  defective: 'Rusak / Cacat',
  returned: 'Retur',
  tersedia: 'Tersedia',
  terjual: 'Terjual',
  retur: 'Retur',
};

export type TransferStatus =
  'pending_approval' | 'approved' | 'rejected' | 'in_transit' | 'received';

export const TRANSFER_STATUS_LABELS: Record<TransferStatus, string> = {
  pending_approval: 'Menunggu Persetujuan',
  approved: 'Disetujui',
  rejected: 'Ditolak',
  in_transit: 'Dalam Pengiriman',
  received: 'Diterima Lengkap',
};

export type WarrantyType = 'toko' | 'pabrik';

export const WARRANTY_TYPE_LABELS: Record<WarrantyType, string> = {
  toko: 'Garansi Toko',
  pabrik: 'Garansi Resmi Pabrik',
};

// ── Product DTOs ──────────────────────────────────────────────────────────

export interface CreateProductRequest {
  sku: string;
  category_id: string;
  name: string;
  brand?: string | undefined;
  description?: string | undefined;
  unit: string;
  purchase_price: number;
  selling_price: number;
  is_ppn: boolean;
  flag_serial_tracking: boolean;
  weight_gram?: number | undefined;
  atribut_varian?: Record<string, unknown> | undefined;
}

export interface UpdateProductRequest {
  name: string;
  brand?: string | undefined;
  description?: string | undefined;
  unit: string;
  purchase_price: number;
  selling_price: number;
  is_ppn: boolean;
  flag_serial_tracking: boolean;
  weight_gram?: number | undefined;
  atribut_varian?: Record<string, unknown> | undefined;
}

export interface SetProductStatusRequest {
  status: ProductStatus;
}

export interface ListProductParams {
  status?: string | undefined;
  category_id?: string | undefined;
  search?: string | undefined;
  page?: number | undefined;
  limit?: number | undefined;
}

export interface ProductResponse {
  id: string;
  sku: string;
  category_id: string;
  name: string;
  brand: string;
  description: string;
  unit: string;
  purchase_price?: number | null | undefined;
  selling_price: number;
  status: ProductStatus;
  is_ppn: boolean;
  flag_serial_tracking: boolean;
  weight_gram: number;
  atribut_varian: Record<string, unknown>;
  primary_image_url?: string | null | undefined;
  created_at: string;
  updated_at: string;
}

export interface ProductImageResponse {
  id: string;
  product_id: string;
  url: string;
  is_primary: boolean;
  sort_order: number;
  created_at: string;
}

export interface ListProductResponse {
  data: ProductResponse[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

// ── Category DTOs ─────────────────────────────────────────────────────────

export interface CreateCategoryRequest {
  name: string;
  parent_id?: string | undefined;
  image_url?: string | undefined;
}

export interface UpdateCategoryRequest {
  name: string;
  parent_id?: string | undefined;
  image_url?: string | undefined;
}

export interface CategoryResponse {
  id: string;
  name: string;
  parent_id?: string | null | undefined;
  image_url?: string | null | undefined;
  created_at: string;
  updated_at: string;
}

// ── Location DTOs ─────────────────────────────────────────────────────────

export interface CreateLocationRequest {
  code: string;
  name: string;
  type: LocationType;
  address?: string | undefined;
  latitude?: number | null | undefined;
  longitude?: number | null | undefined;
}

export interface UpdateLocationRequest {
  code: string;
  name: string;
  type: LocationType;
  address?: string | undefined;
  latitude?: number | null | undefined;
  longitude?: number | null | undefined;
}

export interface SetLocationStatusRequest {
  is_active: boolean;
}

export interface LocationResponse {
  id: string;
  code: string;
  name: string;
  type: LocationType;
  address: string;
  latitude?: number | null | undefined;
  longitude?: number | null | undefined;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// ── Stock DTOs ────────────────────────────────────────────────────────────

export interface AdjustStockRequest {
  product_id: string;
  location_id: string;
  new_quantity: number;
  reason?: string | undefined;
}

export interface UpdateMinStockRequest {
  product_id: string;
  location_id: string;
  min_stock: number;
}

export interface StockResponse {
  id: string;
  product_id: string;
  location_id: string;
  quantity: number;
  reserved_quantity: number;
  available_quantity: number;
  min_stock: number;
  is_low_stock: boolean;
  updated_at: string;
}

export interface StockAdjustmentResponse {
  id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  location_id: string;
  location_name: string;
  previous_quantity: number;
  new_quantity: number;
  difference: number;
  reason: string;
  adjusted_by: string;
  adjusted_by_name: string;
  created_at: string;
}

export interface StockAdjustmentListParams {
  location_id?: string | undefined;
  product_id?: string | undefined;
  page?: number | undefined;
  limit?: number | undefined;
}

// ── Barcode DTOs ──────────────────────────────────────────────────────────

export interface AddBarcodeRequest {
  barcode: string;
  is_primary: boolean;
}

export interface BarcodeResponse {
  id: string;
  product_id: string;
  barcode: string;
  is_primary: boolean;
  created_at: string;
}

export interface ProductLookupResponse {
  product: ProductResponse;
  scanned_barcode: BarcodeResponse;
}

// ── Serial Unit DTOs ──────────────────────────────────────────────────────

export interface RegisterSerialUnitsRequest {
  location_id: string;
  serial_numbers: string[];
}

export interface SerialUnitResponse {
  id: string;
  product_id: string;
  location_id: string;
  serial_number: string;
  status: SerialStatus;
  created_at: string;
  updated_at: string;
  product_name?: string;
  product_sku?: string;
  product_brand?: string;
  location_name?: string;
  location_code?: string;
}

export interface SerialUnitLookupResponse {
  serial_unit: SerialUnitResponse;
  product_name: string;
  product_sku: string;
  product_brand: string;
  location_name: string;
  location_code: string;
}

export interface UpdateSerialStatusRequest {
  status: SerialStatus;
}

// ── Price Override DTOs ───────────────────────────────────────────────────

export interface CreatePriceOverrideRequest {
  location_id: string;
  promotional_price: number;
  max_quantity?: number | undefined;
  start_date: string;
  end_date: string;
  reason?: string | undefined;
}

export interface PriceOverrideResponse {
  id: string;
  product_id: string;
  location_id: string;
  promotional_price: number;
  max_quantity?: number | null | undefined;
  claimed_quantity: number;
  remaining_quantity?: number | null | undefined;
  start_date: string;
  end_date: string;
  reason: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  product_name?: string;
  product_sku?: string;
  product_brand?: string;
  base_price?: number;
  location_name?: string;
  location_code?: string;
}

export interface EffectivePriceResponse {
  product_id: string;
  location_id: string;
  base_price: number;
  effective_price: number;
  has_discount: boolean;
  discount_amount: number;
  remaining_quota?: number | null | undefined;
  promo_id?: string | null | undefined;
  promo_reason?: string | null | undefined;
}

// ── Stock Transfer DTOs ───────────────────────────────────────────────────

export interface StockTransferItemRequest {
  product_id: string;
  quantity: number;
  serial_unit_ids?: string[] | undefined;
}

export interface CreateStockTransferRequest {
  from_location_id: string;
  to_location_id: string;
  notes?: string | undefined;
  items: StockTransferItemRequest[];
}

export interface RejectStockTransferRequest {
  reason: string;
}

export interface StockTransferItemResponse {
  id: string;
  product_id: string;
  quantity: number;
  received_quantity: number;
  serial_unit_ids: string[];
  created_at: string;
}

export interface StockTransferResponse {
  id: string;
  transfer_number: string;
  from_location_id: string;
  to_location_id: string;
  status: TransferStatus;
  notes: string;
  rejection_reason?: string | undefined;
  requested_by: string;
  requested_by_name?: string | undefined;
  approved_by?: string | null | undefined;
  approved_by_name?: string | null | undefined;
  received_by?: string | null | undefined;
  received_by_name?: string | null | undefined;
  items: StockTransferItemResponse[];
  created_at: string;
  updated_at: string;
}

// ── Warranty DTOs ─────────────────────────────────────────────────────────

export interface CreateWarrantyPolicyRequest {
  name: string;
  type: WarrantyType;
  duration_months: number;
  duration_days: number;
  coverage?: string | undefined;
  claim_instructions?: string | undefined;
}

export interface WarrantyPolicyResponse {
  id: string;
  name: string;
  type: WarrantyType;
  duration_months: number;
  duration_days: number;
  coverage: string;
  claim_instructions: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface AssignProductWarrantyRequest {
  warranty_policy_id: string;
}

export interface ProductWarrantyResponse {
  id: string;
  product_id: string;
  warranty_policy_id: string;
  type: WarrantyType;
  is_active: boolean;
  policy?: WarrantyPolicyResponse | null | undefined;
  created_at: string;
  updated_at: string;
}
