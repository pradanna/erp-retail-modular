import { SvelteSet, SvelteDate } from 'svelte/reactivity';
import { createStockIn, getStockMovementDetail, lookupBarcode } from '@erp/api-client';
import type {
	ProductResponse,
	CreateStockMovementItemRequest,
	StockMovementResponse
} from '@erp/types';
import { ApiError } from '@erp/types';
import { toast } from '@erp/ui';

export interface FormItemRow {
	id: string;
	productId: string;
	quantity: number;
	notes: string;
	serialInput: string;
	barcodeScanInput: string;
	showBulkTextarea: boolean;
}

export function createEmptyItemRow(): FormItemRow {
	return {
		id: crypto.randomUUID(),
		productId: '',
		quantity: 1,
		notes: '',
		serialInput: '',
		barcodeScanInput: '',
		showBulkTextarea: false
	};
}

export const stockInReasonOptions: { value: string; label: string }[] = [
	{ value: 'pembelian_non_po', label: 'Pembelian Langsung / Bebas (Non-PO)' },
	{ value: 'saldo_awal', label: 'Saldo Awal Migrasi Persediaan Toko' },
	{ value: 'bonus_supplier', label: 'Bonus / Hadiah Cuma-cuma dari Supplier' },
	{ value: 'retur_konsumen', label: 'Penerimaan Retur dari Konsumen' },
	{ value: 'penerimaan_lain', label: 'Penerimaan Lainnya / Hibah' }
];

export function getReasonLabel(code: string): string {
	const found = stockInReasonOptions.find((r) => r.value === code);
	return found?.label ?? code;
}

export function formatDate(dtStr: string): string {
	if (!dtStr) return '-';
	try {
		return new SvelteDate(dtStr).toLocaleDateString('id-ID', {
			day: '2-digit',
			month: 'short',
			year: 'numeric'
		});
	} catch {
		return dtStr;
	}
}

export function formatDateTime(dtStr: string): string {
	try {
		return new SvelteDate(dtStr).toLocaleString('id-ID', {
			day: '2-digit',
			month: 'short',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	} catch {
		return dtStr;
	}
}

export function formatRupiah(amount?: number): string {
	if (amount === undefined || amount === null) return '-';
	return new Intl.NumberFormat('id-ID', {
		style: 'currency',
		currency: 'IDR',
		minimumFractionDigits: 0
	}).format(amount);
}

export function getImageUrl(url?: string | null): string {
	if (!url) return '';
	if (url.startsWith('http://') || url.startsWith('https://')) return url;
	const cleanPath = url.startsWith('/') ? url : `/${url}`;
	return `http://localhost:8088${cleanPath}`;
}

export function copySerials(serials: string[]) {
	if (!serials || serials.length === 0) return;
	navigator.clipboard.writeText(serials.join('\n'));
	toast.success(`${serials.length} nomor seri berhasil disalin ke clipboard!`);
}

/**
 * Controller untuk Formulir Pencatatan Barang Masuk (Stock In)
 */
export class StockInFormState {
	showModal = $state(false);
	movementDate = $state<string>(new SvelteDate().toISOString().slice(0, 10));
	locationId = $state<string>('');
	categoryReason = $state<string>('pembelian_non_po');
	refNumber = $state<string>('');
	notes = $state<string>('');
	items = $state<FormItemRow[]>([createEmptyItemRow()]);
	submitting = $state(false);
	formError = $state<string | null>(null);
	fastScanCode = $state<string>('');
	fastScanLoading = $state<boolean>(false);

	openModal(defaultLocationId: string) {
		this.locationId = defaultLocationId;
		this.movementDate = new SvelteDate().toISOString().slice(0, 10);
		this.categoryReason = 'pembelian_non_po';
		this.refNumber = '';
		this.notes = '';
		this.items = [createEmptyItemRow()];
		this.formError = null;
		this.fastScanCode = '';
		this.fastScanLoading = false;
		this.showModal = true;
	}

	closeModal() {
		this.showModal = false;
	}

	addItemRow() {
		this.items = [...this.items, createEmptyItemRow()];
	}

	removeItemRow(index: number) {
		if (this.items.length <= 1) {
			toast.warning('Dokumen harus memiliki minimal 1 barang');
			return;
		}
		this.items = this.items.filter((_, i) => i !== index);
	}

	async handleFastScan(token: string, products: ProductResponse[]) {
		const code = this.fastScanCode.trim();
		if (!code) return;
		if (!this.locationId) {
			toast.warning('Pilih cabang penerima terlebih dahulu sebelum scan barcode.');
			return;
		}

		this.fastScanLoading = true;
		try {
			// 1. Cek apakah cocok dengan SKU produk
			let matchedProd = products.find((p) => p.sku.toLowerCase() === code.toLowerCase());

			// 2. Jika belum, coba lookup barcode produk
			if (!matchedProd) {
				try {
					const bcRes = await lookupBarcode(token, code);
					if (bcRes && bcRes.product) {
						matchedProd = products.find((p) => p.id === bcRes.product.id) || bcRes.product;
					}
				} catch {
					// Fallthrough
				}
			}

			if (matchedProd) {
				const existingIdx = this.items.findIndex((it) => it.productId === matchedProd!.id);
				if (existingIdx !== -1) {
					this.items[existingIdx].quantity = Number(this.items[existingIdx].quantity) + 1;
					toast.success(
						`Kuantitas "${matchedProd.name}" bertambah menjadi ${this.items[existingIdx].quantity}`
					);
				} else {
					const emptyIdx = this.items.findIndex((it) => !it.productId);
					if (emptyIdx !== -1) {
						this.items[emptyIdx].productId = matchedProd.id;
					} else {
						const newRow = createEmptyItemRow();
						newRow.productId = matchedProd.id;
						this.items = [...this.items, newRow];
					}
					toast.success(`Produk "${matchedProd.name}" berhasil ditambahkan!`);
				}
				this.fastScanCode = '';
			} else {
				toast.error(`Barcode / SKU "${code}" tidak ditemukan dalam sistem katalog.`);
			}
		} finally {
			this.fastScanLoading = false;
		}
	}

	getSerialsArray(item: FormItemRow): string[] {
		return (item.serialInput ?? '')
			.split('\n')
			.map((s) => s.trim())
			.filter((s) => s.length > 0);
	}

	handleAddSerialByBarcode(idx: number) {
		const item = this.items[idx];
		if (!item) return;
		const sn = item.barcodeScanInput.trim();
		if (!sn) return;

		const serials = this.getSerialsArray(item);
		if (serials.some((s) => s.toLowerCase() === sn.toLowerCase())) {
			toast.warning(`Nomor seri / IMEI ${sn} sudah ada dalam daftar`);
			item.barcodeScanInput = '';
			return;
		}

		if (serials.length >= Number(item.quantity)) {
			item.quantity = Number(item.quantity) + 1;
		}

		serials.push(sn);
		item.serialInput = serials.join('\n');
		item.barcodeScanInput = '';
		toast.success(`Nomor seri ${sn} berhasil ditambahkan!`);
	}

	handleRemoveSerial(idx: number, snToRemove: string) {
		const item = this.items[idx];
		if (!item) return;
		const serials = this.getSerialsArray(item).filter(
			(s) => s.toLowerCase() !== snToRemove.toLowerCase()
		);
		item.serialInput = serials.join('\n');
	}

	handleClearSerials(idx: number) {
		const item = this.items[idx];
		if (!item) return;
		item.serialInput = '';
		toast.info('Daftar nomor seri telah dikosongkan');
	}

	async submit(
		token: string,
		getProductById: (id: string) => ProductResponse | undefined,
		onSuccess: () => Promise<void>
	) {
		if (!this.locationId) {
			this.formError = 'Silakan pilih cabang/gudang penerima';
			return;
		}

		// Validasi item
		const requestItems: CreateStockMovementItemRequest[] = [];
		const seenProducts = new SvelteSet<string>();

		for (let i = 0; i < this.items.length; i++) {
			const item = this.items[i];
			if (!item || !item.productId) {
				this.formError = `Baris ke-${i + 1}: Silakan pilih produk`;
				return;
			}
			if (seenProducts.has(item.productId)) {
				this.formError = `Produk pada baris ke-${i + 1} duplikat. Gabungkan kuantitas dalam 1 baris.`;
				return;
			}
			seenProducts.add(item.productId);

			if (item.quantity <= 0) {
				this.formError = `Baris ke-${i + 1}: Jumlah kuantitas harus lebih dari 0`;
				return;
			}

			const prod = getProductById(item.productId);
			let serials: string[] | undefined = undefined;

			if (prod?.flag_serial_tracking && item.serialInput.trim()) {
				const splitSerials = item.serialInput
					.split('\n')
					.map((s) => s.trim())
					.filter(Boolean);

				if (splitSerials.length !== item.quantity) {
					this.formError = `Baris ke-${i + 1} (${prod.name}): Kuantitas ${item.quantity}, tetapi Anda memasukkan ${splitSerials.length} nomor seri/IMEI.`;
					return;
				}
				serials = splitSerials;
			}

			requestItems.push({
				product_id: item.productId,
				quantity: item.quantity,
				notes: item.notes.trim() || undefined,
				serial_numbers: serials
			});
		}

		try {
			this.submitting = true;
			this.formError = null;

			const created = await createStockIn(token, {
				movement_date: this.movementDate || undefined,
				location_id: this.locationId,
				category_reason: this.categoryReason,
				reference_number: this.refNumber.trim() || undefined,
				notes: this.notes.trim() || undefined,
				items: requestItems
			});

			toast.success(`Transaksi Barang Masuk ${created.movement_number} berhasil disimpan!`);
			this.showModal = false;
			await onSuccess();
		} catch (err: unknown) {
			this.formError =
				err instanceof ApiError ? err.message : 'Gagal menyimpan transaksi barang masuk';
		} finally {
			this.submitting = false;
		}
	}
}

/**
 * Controller untuk Detail Dokumen Barang Masuk
 */
export class StockInDetailState {
	showModal = $state(false);
	selectedMovement = $state<StockMovementResponse | null>(null);
	loading = $state(false);

	async open(token: string, movementId: string) {
		try {
			this.loading = true;
			this.showModal = true;
			this.selectedMovement = await getStockMovementDetail(token, movementId);
		} catch {
			toast.error('Gagal mengambil detail dokumen');
			this.showModal = false;
		} finally {
			this.loading = false;
		}
	}

	close() {
		this.showModal = false;
	}
}
