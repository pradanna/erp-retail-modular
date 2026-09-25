/**
 * Inventory API Client — Type-safe HTTP functions untuk modul Inventory.
 *
 * Menghubungkan frontend SvelteKit dengan endpoint REST API backend Go.
 * Seluruh tipe data diimpor langsung dari @erp/types (single source of truth).
 */

import { fetchApi } from './client.js';
import type {
  CreateProductRequest,
  UpdateProductRequest,
  ProductResponse,
  ListProductParams,
  ListProductResponse,
  ProductStatus,
  CreateCategoryRequest,
  UpdateCategoryRequest,
  CategoryResponse,
  CreateLocationRequest,
  UpdateLocationRequest,
  LocationResponse,
  AdjustStockRequest,
  UpdateMinStockRequest,
  StockResponse,
  AddBarcodeRequest,
  BarcodeResponse,
  ProductLookupResponse,
  RegisterSerialUnitsRequest,
  SerialUnitResponse,
  SerialUnitLookupResponse,
  SerialStatus,
  CreatePriceOverrideRequest,
  PriceOverrideResponse,
  EffectivePriceResponse,
  CreateStockTransferRequest,
  StockTransferResponse,
  CreateWarrantyPolicyRequest,
  WarrantyPolicyResponse,
  AssignProductWarrantyRequest,
  ProductWarrantyResponse,
  ProductImageResponse,
  StockAdjustmentResponse,
  StockAdjustmentListParams,
} from '@erp/types';

// ── 1. Produk ─────────────────────────────────────────────────────────────

export async function createProduct(
  token: string,
  req: CreateProductRequest,
): Promise<{ id: string; message: string }> {
  return fetchApi<{ id: string; message: string }>('/api/v1/inventory/products', {
    method: 'POST',
    body: req,
    token,
  });
}

export async function listProducts(
  token: string,
  params?: ListProductParams,
): Promise<ListProductResponse> {
  const q = new URLSearchParams();
  if (params?.status) q.set('status', params.status);
  if (params?.category_id) q.set('category_id', params.category_id);
  if (params?.search) q.set('search', params.search);
  if (params?.page) q.set('page', String(params.page));
  if (params?.limit) q.set('limit', String(params.limit));

  const query = q.toString();
  return fetchApi<ListProductResponse>(`/api/v1/inventory/products${query ? `?${query}` : ''}`, {
    token,
  });
}

export async function getProduct(token: string, id: string): Promise<ProductResponse> {
  return fetchApi<ProductResponse>(`/api/v1/inventory/products/${id}`, { token });
}

export async function updateProduct(
  token: string,
  id: string,
  req: UpdateProductRequest,
): Promise<ProductResponse> {
  return fetchApi<ProductResponse>(`/api/v1/inventory/products/${id}`, {
    method: 'PUT',
    body: req,
    token,
  });
}

export async function setProductStatus(
  token: string,
  id: string,
  status: ProductStatus,
): Promise<{ id: string; status: ProductStatus }> {
  return fetchApi<{ id: string; status: ProductStatus }>(
    `/api/v1/inventory/products/${id}/status`,
    {
      method: 'PATCH',
      body: { status },
      token,
    },
  );
}

// ── 2. Kategori ───────────────────────────────────────────────────────────

export async function createCategory(
  token: string,
  req: CreateCategoryRequest,
): Promise<{ id: string; message: string }> {
  return fetchApi<{ id: string; message: string }>('/api/v1/inventory/categories', {
    method: 'POST',
    body: req,
    token,
  });
}

export async function listCategories(token: string): Promise<CategoryResponse[]> {
  return fetchApi<CategoryResponse[]>('/api/v1/inventory/categories', { token });
}

export async function updateCategory(
  token: string,
  id: string,
  req: UpdateCategoryRequest,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/api/v1/inventory/categories/${id}`, {
    method: 'PUT',
    body: req,
    token,
  });
}

export async function deleteCategory(token: string, id: string): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/api/v1/inventory/categories/${id}`, {
    method: 'DELETE',
    token,
  });
}

// ── 3. Lokasi Cabang & Gudang ─────────────────────────────────────────────

export async function createLocation(
  token: string,
  req: CreateLocationRequest,
): Promise<{ id: string; message: string }> {
  return fetchApi<{ id: string; message: string }>('/api/v1/inventory/locations', {
    method: 'POST',
    body: req,
    token,
  });
}

export async function listLocations(
  token: string,
  activeOnly?: boolean,
): Promise<LocationResponse[]> {
  const query = activeOnly ? '?active_only=true' : '';
  return fetchApi<LocationResponse[]>(`/api/v1/inventory/locations${query}`, { token });
}

export async function getLocation(token: string, id: string): Promise<LocationResponse> {
  return fetchApi<LocationResponse>(`/api/v1/inventory/locations/${id}`, { token });
}

export async function updateLocation(
  token: string,
  id: string,
  req: UpdateLocationRequest,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/api/v1/inventory/locations/${id}`, {
    method: 'PUT',
    body: req,
    token,
  });
}

export async function setLocationStatus(
  token: string,
  id: string,
  isActive: boolean,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/api/v1/inventory/locations/${id}/status`, {
    method: 'PATCH',
    body: { is_active: isActive },
    token,
  });
}

export async function deleteLocation(token: string, id: string): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/api/v1/inventory/locations/${id}`, {
    method: 'DELETE',
    token,
  });
}

// ── 4. Stok & Opname ──────────────────────────────────────────────────────

export async function adjustStock(token: string, req: AdjustStockRequest): Promise<StockResponse> {
  return fetchApi<StockResponse>('/api/v1/inventory/stocks/adjust', {
    method: 'POST',
    body: req,
    token,
  });
}

export async function updateMinStock(
  token: string,
  req: UpdateMinStockRequest,
): Promise<StockResponse> {
  return fetchApi<StockResponse>('/api/v1/inventory/stocks/min-stock', {
    method: 'PUT',
    body: req,
    token,
  });
}

export async function getStock(
  token: string,
  productId: string,
  locationId: string,
): Promise<StockResponse> {
  return fetchApi<StockResponse>(
    `/api/v1/inventory/stocks?product_id=${productId}&location_id=${locationId}`,
    { token },
  );
}

export async function listStocks(token: string, locationId: string): Promise<StockResponse[]> {
  return fetchApi<StockResponse[]>(`/api/v1/inventory/stocks?location_id=${locationId}`, { token });
}

export async function listLowStockAlerts(
  token: string,
  locationId?: string,
): Promise<StockResponse[]> {
  const query = locationId ? `?location_id=${locationId}` : '';
  return fetchApi<StockResponse[]>(`/api/v1/inventory/stocks/alerts${query}`, { token });
}

export async function listStockAdjustments(
  token: string,
  params?: StockAdjustmentListParams,
): Promise<{ data: StockAdjustmentResponse[]; meta: { page: number; limit: number; total_items: number; total_pages: number } }> {
  const query = new URLSearchParams();
  if (params?.location_id) query.set('location_id', params.location_id);
  if (params?.product_id) query.set('product_id', params.product_id);
  if (params?.page) query.set('page', String(params.page));
  if (params?.limit) query.set('limit', String(params.limit));

  const qs = query.toString() ? `?${query.toString()}` : '';
  return fetchApi<{ data: StockAdjustmentResponse[]; meta: { page: number; limit: number; total_items: number; total_pages: number } }>(
    `/api/v1/inventory/stocks/adjustments${qs}`,
    { token },
  );
}

// ── 5. Barcode Produk ─────────────────────────────────────────────────────

export async function addBarcode(
  token: string,
  productId: string,
  req: AddBarcodeRequest,
): Promise<BarcodeResponse> {
  return fetchApi<BarcodeResponse>(`/api/v1/inventory/products/${productId}/barcodes`, {
    method: 'POST',
    body: req,
    token,
  });
}

export async function listBarcodes(token: string, productId: string): Promise<BarcodeResponse[]> {
  return fetchApi<BarcodeResponse[]>(`/api/v1/inventory/products/${productId}/barcodes`, { token });
}

export async function deleteBarcode(
  token: string,
  productId: string,
  barcodeId: string,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(
    `/api/v1/inventory/products/${productId}/barcodes/${barcodeId}`,
    {
      method: 'DELETE',
      token,
    },
  );
}

export async function lookupBarcode(
  token: string,
  barcode: string,
): Promise<ProductLookupResponse> {
  return fetchApi<ProductLookupResponse>(
    `/api/v1/inventory/barcodes/lookup?code=${encodeURIComponent(barcode)}&barcode=${encodeURIComponent(barcode)}`,
    { token },
  );
}

// ── 6. Serial Number & IMEI ───────────────────────────────────────────────

export async function registerSerialUnits(
  token: string,
  productId: string,
  req: RegisterSerialUnitsRequest,
): Promise<SerialUnitResponse[]> {
  return fetchApi<SerialUnitResponse[]>(`/api/v1/inventory/products/${productId}/serials`, {
    method: 'POST',
    body: req,
    token,
  });
}

export async function listSerialUnits(
  token: string,
  productId: string,
): Promise<SerialUnitResponse[]> {
  return fetchApi<SerialUnitResponse[]>(`/api/v1/inventory/products/${productId}/serials`, {
    token,
  });
}

export async function listSerials(
  token: string,
  filters?: {
    product_id?: string;
    location_id?: string;
    status?: string;
    search?: string;
  },
): Promise<SerialUnitResponse[]> {
  const q = new URLSearchParams();
  if (filters?.product_id) q.set('product_id', filters.product_id);
  if (filters?.location_id) q.set('location_id', filters.location_id);
  if (filters?.status) q.set('status', filters.status);
  if (filters?.search) q.set('search', filters.search);
  const query = q.toString();
  return fetchApi<SerialUnitResponse[]>(`/api/v1/inventory/serials${query ? `?${query}` : ''}`, {
    token,
  });
}

export async function lookupSerialUnit(
  token: string,
  serialNumber: string,
): Promise<SerialUnitLookupResponse> {
  return fetchApi<SerialUnitLookupResponse>(
    `/api/v1/inventory/serials/lookup?sn=${encodeURIComponent(serialNumber)}&serial=${encodeURIComponent(serialNumber)}`,
    { token },
  );
}

export async function updateSerialStatus(
  token: string,
  serialId: string,
  status: SerialStatus,
): Promise<SerialUnitResponse> {
  return fetchApi<SerialUnitResponse>(`/api/v1/inventory/serials/${serialId}/status`, {
    method: 'PATCH',
    body: { status },
    token,
  });
}

// ── 7. Price Override (Promo Cabang) ──────────────────────────────────────

export async function createPriceOverride(
  token: string,
  productId: string,
  req: CreatePriceOverrideRequest,
): Promise<PriceOverrideResponse> {
  return fetchApi<PriceOverrideResponse>(
    `/api/v1/inventory/products/${productId}/price-overrides`,
    {
      method: 'POST',
      body: req,
      token,
    },
  );
}

export async function listPriceOverrides(
  token: string,
  productId: string,
): Promise<PriceOverrideResponse[]> {
  return fetchApi<PriceOverrideResponse[]>(
    `/api/v1/inventory/products/${productId}/price-overrides`,
    { token },
  );
}

export async function listAllPriceOverrides(
  token: string,
  filters?: {
    product_id?: string;
    location_id?: string;
    is_active?: boolean;
  },
): Promise<PriceOverrideResponse[]> {
  const q = new URLSearchParams();
  if (filters?.product_id) q.set('product_id', filters.product_id);
  if (filters?.location_id) q.set('location_id', filters.location_id);
  if (filters?.is_active !== undefined) q.set('is_active', String(filters.is_active));
  const query = q.toString();
  return fetchApi<PriceOverrideResponse[]>(
    `/api/v1/inventory/price-overrides${query ? `?${query}` : ''}`,
    { token },
  );
}

export async function getEffectivePrice(
  token: string,
  productId: string,
  locationId: string,
): Promise<EffectivePriceResponse> {
  return fetchApi<EffectivePriceResponse>(
    `/api/v1/inventory/price-overrides/effective-price?product_id=${productId}&location_id=${locationId}`,
    { token },
  );
}

export async function deactivatePriceOverride(
  token: string,
  overrideId: string,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(
    `/api/v1/inventory/price-overrides/${overrideId}/deactivate`,
    {
      method: 'PATCH',
      token,
    },
  );
}

export async function claimPromoQuota(
  token: string,
  overrideId: string,
  quantity: number,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(`/api/v1/inventory/price-overrides/${overrideId}/claim`, {
    method: 'POST',
    body: { quantity },
    token,
  });
}

// ── 8. Stock Transfer (Mutasi Stok Antar Cabang) ──────────────────────────

export async function createStockTransfer(
  token: string,
  req: CreateStockTransferRequest,
): Promise<StockTransferResponse> {
  return fetchApi<StockTransferResponse>('/api/v1/inventory/transfers', {
    method: 'POST',
    body: req,
    token,
  });
}

export async function listStockTransfers(
  token: string,
  filters?: {
    status?: string;
    location_id?: string;
    from_location_id?: string;
    to_location_id?: string;
  },
): Promise<StockTransferResponse[]> {
  const q = new URLSearchParams();
  if (filters?.status) q.set('status', filters.status);
  if (filters?.location_id) q.set('location_id', filters.location_id);
  if (filters?.from_location_id) q.set('from_location_id', filters.from_location_id);
  if (filters?.to_location_id) q.set('to_location_id', filters.to_location_id);
  const query = q.toString();

  return fetchApi<StockTransferResponse[]>(
    `/api/v1/inventory/transfers${query ? `?${query}` : ''}`,
    { token },
  );
}

export async function getStockTransfer(token: string, id: string): Promise<StockTransferResponse> {
  return fetchApi<StockTransferResponse>(`/api/v1/inventory/transfers/${id}`, { token });
}

export async function approveStockTransfer(
  token: string,
  id: string,
): Promise<StockTransferResponse> {
  return fetchApi<StockTransferResponse>(`/api/v1/inventory/transfers/${id}/approve`, {
    method: 'POST',
    token,
  });
}

export async function rejectStockTransfer(
  token: string,
  id: string,
  reason: string,
): Promise<StockTransferResponse> {
  return fetchApi<StockTransferResponse>(`/api/v1/inventory/transfers/${id}/reject`, {
    method: 'POST',
    body: { reason },
    token,
  });
}

export async function shipStockTransfer(token: string, id: string): Promise<StockTransferResponse> {
  return fetchApi<StockTransferResponse>(`/api/v1/inventory/transfers/${id}/ship`, {
    method: 'POST',
    token,
  });
}

export async function receiveStockTransfer(
  token: string,
  id: string,
): Promise<StockTransferResponse> {
  return fetchApi<StockTransferResponse>(`/api/v1/inventory/transfers/${id}/receive`, {
    method: 'POST',
    token,
  });
}

// ── 9. Garansi (Warranty) ─────────────────────────────────────────────────

export async function createWarrantyPolicy(
  token: string,
  req: CreateWarrantyPolicyRequest,
): Promise<WarrantyPolicyResponse> {
  return fetchApi<WarrantyPolicyResponse>('/api/v1/inventory/warranties/policies', {
    method: 'POST',
    body: req,
    token,
  });
}

export async function listWarrantyPolicies(token: string): Promise<WarrantyPolicyResponse[]> {
  return fetchApi<WarrantyPolicyResponse[]>('/api/v1/inventory/warranties/policies', {
    token,
  });
}

export async function assignProductWarranty(
  token: string,
  productId: string,
  req: AssignProductWarrantyRequest,
): Promise<ProductWarrantyResponse> {
  return fetchApi<ProductWarrantyResponse>(`/api/v1/inventory/products/${productId}/warranties`, {
    method: 'POST',
    body: req,
    token,
  });
}

export async function getProductWarranties(
  token: string,
  productId: string,
): Promise<ProductWarrantyResponse[]> {
  return fetchApi<ProductWarrantyResponse[]>(`/api/v1/inventory/products/${productId}/warranties`, {
    token,
  });
}

export async function deactivateProductWarranty(
  token: string,
  productId: string,
  warrantyId: string,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(
    `/api/v1/inventory/warranties/products/${productId}/deactivate?warranty_id=${warrantyId}`,
    {
      method: 'POST',
      token,
    },
  );
}

// ── 9. Foto Produk (Product Images) ──────────────────────────────────────

export async function uploadProductImage(
  token: string,
  productId: string,
  file: File,
  isPrimary = false,
): Promise<ProductImageResponse> {
  const formData = new FormData();
  formData.append('image', file);
  if (isPrimary) {
    formData.append('is_primary', 'true');
  }

  return fetchApi<ProductImageResponse>(`/api/v1/inventory/products/${productId}/images`, {
    method: 'POST',
    body: formData,
    token,
  });
}

export async function listProductImages(
  token: string,
  productId: string,
): Promise<ProductImageResponse[]> {
  return fetchApi<ProductImageResponse[]>(`/api/v1/inventory/products/${productId}/images`, {
    token,
  });
}

export async function deleteProductImage(
  token: string,
  productId: string,
  imageId: string,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(
    `/api/v1/inventory/products/${productId}/images/${imageId}`,
    {
      method: 'DELETE',
      token,
    },
  );
}

export async function setPrimaryProductImage(
  token: string,
  productId: string,
  imageId: string,
): Promise<{ message: string }> {
  return fetchApi<{ message: string }>(
    `/api/v1/inventory/products/${productId}/images/${imageId}/primary`,
    {
      method: 'PATCH',
      token,
    },
  );
}

// ── 10. Foto Kategori (Category Images) ────────────────────────────────────

export async function uploadCategoryImage(
  token: string,
  file: File,
): Promise<{ url: string }> {
  const formData = new FormData();
  formData.append('image', file);

  return fetchApi<{ url: string }>(`/api/v1/inventory/categories/upload-image`, {
    method: 'POST',
    body: formData,
    token,
  });
}


