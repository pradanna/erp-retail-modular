import { SvelteSet, SvelteDate } from 'svelte/reactivity';
import {
	createStockOut,
	listSerials,
	getStockMovementDetail,
	lookupBarcode,
	lookupSerialUnit
} from '@erp/api-client';
import type {
	ProductResponse,
	SerialUnitResponse,
	CreateStockMovementItemRequest,
	StockMovementResponse
} from '@erp/types';
import { ApiError } from '@erp/types';
import { toast, type Select2Option } from '@erp/ui';

export interface FormItemRow {
	id: string;
	productId: string;
	quantity: number;
	notes: string;
	selectedSerials: string[];
	availableSerials: SerialUnitResponse[];
	loadingSerials: boolean;
	barcodeInput: string;
	select2Value: string;
}

export function createEmptyItemRow(): FormItemRow {
	return {
		id: crypto.randomUUID(),
		productId: '',
		quantity: 1,
		notes: '',
		selectedSerials: [],
		availableSerials: [],
		loadingSerials: false,
		barcodeInput: '',
		select2Value: ''
	};
}

export const stockOutReasonOptions: { value: string; label: string }[] = [
	{ value: 'rusak_afkir', label: 'Barang Rusak / Pecah / Cacat (Scrap / Afkir)' },
	{ value: 'pemakaian_internal', label: 'Pemakaian Sendiri / Operasional Toko' },
	{ value: 'sampel_display', label: 'Sampel Pajangan Display Toko' },
	{ value: 'penjualan_non_pos', label: 'Penjualan Bebas / Grosir Manual (Non-POS)' },
	{ value: 'retur_supplier', label: 'Retur Pengembalian ke Vendor / Distributor' },
	{ value: 'pengeluaran_lain', label: 'Pengeluaran Lainnya' }
];

export function getReasonLabel(reason: string): string {
	const found = stockOutReasonOptions.find((r) => r.value === reason);
	return found?.label ?? reason;
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
 * State Controller untuk Formulir Pencatatan Barang Keluar (Stock Out)
 */
export class StockOutFormState {
	showModal = $state(false);
	movementDate = $state<string>(new SvelteDate().toISOString().slice(0, 10));
	locationId = $state<string>('');
	categoryReason = $state<string>('rusak_afkir');
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
		this.categoryReason = 'rusak_afkir';
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

	async handleFastScan(
		token: string,
		products: ProductResponse[],
		getProductById: (id: string) => ProductResponse | undefined
	) {
		const code = this.fastScanCode.trim();
		if (!code) return;
		if (!this.locationId) {
			toast.warning('Pilih lokasi / cabang terlebih dahulu sebelum memindai barang.');
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
					// Lanjutkan ke serial lookup
				}
			}

			// 3. Jika belum, coba lookup serial unit / IMEI
			if (!matchedProd) {
				try {
					const snRes = await lookupSerialUnit(token, code);
					if (snRes && snRes.serial_unit?.product_id) {
						matchedProd = products.find((p) => p.id === snRes.serial_unit.product_id);
						if (matchedProd) {
							// Cari baris yang sudah berisi produk ini atau buat baris baru
							let rowIdx = this.items.findIndex((it) => it.productId === matchedProd!.id);
							if (rowIdx === -1) {
								const emptyIdx = this.items.findIndex((it) => !it.productId);
								if (emptyIdx !== -1) {
									rowIdx = emptyIdx;
								} else {
									this.addItemRow();
									rowIdx = this.items.length - 1;
								}
								await this.handleProductChange(rowIdx, matchedProd.id, token, getProductById);
							}

							const item = this.items[rowIdx];
							if (item.selectedSerials.includes(snRes.serial_unit.serial_number)) {
								toast.warning(`Nomor seri ${snRes.serial_unit.serial_number} sudah dipilih`);
								this.fastScanCode = '';
								return;
							}

							// Auto tambah kuantitas jika kuantitas kurang dari jumlah serial
							if (item.selectedSerials.length >= Number(item.quantity)) {
								item.quantity = Number(item.quantity) + 1;
							}

							item.selectedSerials = [...item.selectedSerials, snRes.serial_unit.serial_number];
							this.fastScanCode = '';
							toast.success(
								`Nomor seri ${snRes.serial_unit.serial_number} (${matchedProd.name}) berhasil dipilih!`
							);
							return;
						}
					}
				} catch {
					// Not found as serial
				}
			}

			if (matchedProd) {
				// Cari apakah produk ini sudah ada dalam baris
				const existingIdx = this.items.findIndex((it) => it.productId === matchedProd!.id);
				if (existingIdx !== -1) {
					this.items[existingIdx].quantity = Number(this.items[existingIdx].quantity) + 1;
					toast.success(
						`Kuantitas "${matchedProd.name}" bertambah menjadi ${this.items[existingIdx].quantity}`
					);
				} else {
					const emptyIdx = this.items.findIndex((it) => !it.productId);
					let targetIdx = emptyIdx;
					if (targetIdx === -1) {
						this.addItemRow();
						targetIdx = this.items.length - 1;
					}
					await this.handleProductChange(targetIdx, matchedProd.id, token, getProductById);
					toast.success(`Produk "${matchedProd.name}" berhasil ditambahkan!`);
				}
				this.fastScanCode = '';
			} else {
				toast.error(`Barcode / SKU / IMEI "${code}" tidak ditemukan dalam sistem.`);
			}
		} finally {
			this.fastScanLoading = false;
		}
	}

	async loadAvailableSerials(
		idx: number,
		token: string,
		getProductById: (id: string) => ProductResponse | undefined
	) {
		const item = this.items[idx];
		if (!item || !item.productId || !this.locationId) return;

		const prod = getProductById(item.productId);
		if (!prod?.flag_serial_tracking) return;

		try {
			item.loadingSerials = true;
			const res = await listSerials(token, {
				product_id: item.productId,
				location_id: this.locationId,
				status: 'tersedia'
			});
			item.availableSerials = res ?? [];
		} catch (err: unknown) {
			console.error('Gagal mengambil daftar nomor seri:', err);
			item.availableSerials = [];
		} finally {
			item.loadingSerials = false;
		}
	}

	async handleProductChange(
		idx: number,
		newProductId: string,
		token: string,
		getProductById: (id: string) => ProductResponse | undefined
	) {
		const item = this.items[idx];
		if (!item) return;
		item.productId = newProductId;
		item.selectedSerials = [];
		item.barcodeInput = '';
		item.select2Value = '';
		item.availableSerials = [];

		const prod = getProductById(newProductId);
		if (prod?.flag_serial_tracking && this.locationId) {
			await this.loadAvailableSerials(idx, token, getProductById);
		}
	}

	async handleLocationChange(
		newLocationId: string,
		token: string,
		getProductById: (id: string) => ProductResponse | undefined
	) {
		this.locationId = newLocationId;
		for (let i = 0; i < this.items.length; i++) {
			this.items[i].selectedSerials = [];
			this.items[i].barcodeInput = '';
			this.items[i].select2Value = '';
			if (this.items[i].productId) {
				await this.loadAvailableSerials(i, token, getProductById);
			}
		}
	}

	handleAddSerialByBarcode(idx: number) {
		const item = this.items[idx];
		if (!item) return;
		const sn = item.barcodeInput.trim();
		if (!sn) return;

		if (item.selectedSerials.includes(sn)) {
			toast.warning(`Nomor seri ${sn} sudah ada dalam daftar pilihan`);
			item.barcodeInput = '';
			return;
		}

		if (item.selectedSerials.length >= Number(item.quantity)) {
			toast.warning(
				`Kuantitas barang keluar (${item.quantity}) sudah terpenuhi. Tambah jumlah jika ingin mengeluarkan lebih banyak unit.`
			);
			return;
		}

		const match = item.availableSerials.find(
			(u) => u.serial_number.toLowerCase() === sn.toLowerCase()
		);

		if (!match) {
			toast.error(`Nomor seri "${sn}" tidak ditemukan atau tidak berstatus tersedia di gudang ini`);
			return;
		}

		item.selectedSerials = [...item.selectedSerials, match.serial_number];
		item.barcodeInput = '';
		toast.success(`Nomor seri ${match.serial_number} berhasil dipilih`);
	}

	handleSelectSerialFromDropdown(idx: number, serialNumber: string) {
		if (!serialNumber) return;
		const item = this.items[idx];
		if (!item) return;

		if (item.selectedSerials.includes(serialNumber)) {
			item.select2Value = '';
			return;
		}

		if (item.selectedSerials.length >= Number(item.quantity)) {
			toast.warning(
				`Kuantitas barang keluar (${item.quantity}) sudah terpenuhi. Tambah jumlah jika ingin mengeluarkan lebih banyak unit.`
			);
			item.select2Value = '';
			return;
		}

		item.selectedSerials = [...item.selectedSerials, serialNumber];
		item.select2Value = '';
		toast.success(`Nomor seri ${serialNumber} berhasil dipilih`);
	}

	handleRemoveSerial(idx: number, serialIndex: number) {
		const item = this.items[idx];
		if (!item) return;
		item.selectedSerials = item.selectedSerials.filter((_, i) => i !== serialIndex);
	}

	handleAutoFillSerials(idx: number) {
		const item = this.items[idx];
		if (!item) return;
		const needed = Number(item.quantity) - item.selectedSerials.length;
		if (needed <= 0) return;

		const unselected = item.availableSerials.filter(
			(u) => !item.selectedSerials.includes(u.serial_number)
		);
		const toAdd = unselected.slice(0, needed).map((u) => u.serial_number);
		item.selectedSerials = [...item.selectedSerials, ...toAdd];
	}

	handleClearSerials(idx: number) {
		const item = this.items[idx];
		if (!item) return;
		item.selectedSerials = [];
		item.barcodeInput = '';
		item.select2Value = '';
	}

	getSerialSelect2Options(item: FormItemRow): Select2Option[] {
		const unselected = item.availableSerials.filter(
			(u) => !item.selectedSerials.includes(u.serial_number)
		);
		return unselected.map((u) => ({
			value: u.serial_number,
			label: u.serial_number,
			subtext: 'Tersedia di Gudang'
		}));
	}

	async submit(
		token: string,
		getProductById: (id: string) => ProductResponse | undefined,
		onSuccess: () => Promise<void>
	) {
		if (!this.locationId) {
			this.formError = 'Silakan pilih cabang/gudang asal';
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

			if (prod?.flag_serial_tracking) {
				if (item.selectedSerials.length !== Number(item.quantity)) {
					this.formError = `Baris ke-${i + 1} (${prod.name}): Kuantitas ${item.quantity}, tetapi Anda baru memilih ${item.selectedSerials.length} nomor seri/IMEI.`;
					return;
				}
				serials = item.selectedSerials;
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

			const created = await createStockOut(token, {
				movement_date: this.movementDate || undefined,
				location_id: this.locationId,
				category_reason: this.categoryReason,
				reference_number: this.refNumber.trim() || undefined,
				notes: this.notes.trim() || undefined,
				items: requestItems
			});

			toast.success(`Transaksi Barang Keluar ${created.movement_number} berhasil disimpan!`);
			this.showModal = false;
			await onSuccess();
		} catch (err: unknown) {
			this.formError =
				err instanceof ApiError ? err.message : 'Gagal menyimpan transaksi barang keluar';
		} finally {
			this.submitting = false;
		}
	}
}

/**
 * Controller untuk Detail Dokumen Barang Keluar
 */
export class StockOutDetailState {
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
