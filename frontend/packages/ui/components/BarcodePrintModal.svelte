<script lang="ts">
  /**
   * BarcodePrintModal — Modal konfigurasi & cetak massal barcode label IMEI / Serial Number.
   *
   * Fitur:
   * - Pratinjau visual barcode Code-128 berbasis SVG tajam.
   * - Opsi format kertas: Stiker Thermal Roll (50x35mm POS) vs Kertas Stiker A4 Grid Sheet.
   * - Salinan per unit (1x, 2x, 3x, 5x, atau kustom hingga 100x).
   * - Toggle informasi label: Nama Toko, Nama Produk, SKU/Brand, Lokasi Gudang.
   * - Eksekusi window.print() otomatis dengan CSS @page standar industri.
   */
  import Modal from './Modal.svelte';
  import Button from './Button.svelte';
  import Badge from './Badge.svelte';
  import Barcode from './Barcode.svelte';
  import { generateCode128Svg } from './barcode-utils.js';

  export interface PrintSerialItem {
    serial_number: string;
    product_name?: string;
    product_sku?: string;
    product_brand?: string;
    location_name?: string;
  }

  interface Props {
    open?: boolean;
    items: PrintSerialItem[];
    onclose?: () => void;
  }

  let { open = $bindable(false), items = [], onclose }: Props = $props();

  let printCopies = $state(1);
  let printFormat = $state<'single' | 'sheet'>('single');
  let printIncludeStoreName = $state(true);
  let printIncludeProductName = $state(true);
  let printIncludeSku = $state(true);
  let printIncludeLocation = $state(false);
  let activePreviewIndex = $state(0);

  // Reset index preview saat daftar item berubah
  $effect(() => {
    if (items.length > 0 && activePreviewIndex >= items.length) {
      activePreviewIndex = 0;
    }
  });

  const currentPreviewItem = $derived(
    items.length > 0 ? (items[activePreviewIndex] ?? items[0]) : null,
  );

  const totalLabels = $derived(
    items.length * Math.max(1, Math.min(100, Number(printCopies) || 1)),
  );

  function handleClose() {
    open = false;
    onclose?.();
  }

  function executePrint() {
    if (items.length === 0) return;

    const copies = Math.max(1, Math.min(100, Number(printCopies) || 1));
    const storeName = 'GEN-E RETAIL';
    const isThermal = printFormat === 'single';

    const printWindow = window.open('', '_blank', 'width=800,height=600');
    if (!printWindow) {
      alert('Harap izinkan pop-up browser untuk mencetak barcode label.');
      return;
    }

    let labelsHtml = '';
    for (const item of items) {
      // Format barcode SVG terkalibrasi:
      // - Thermal Roll: 50x35mm
      // - Kertas HVS A4: Garis bar ditinggikan (height 100) dan modul tebal (2.0) agar tidak blur di serat kertas HVS
      const svgString = generateCode128Svg(item.serial_number, {
        height: isThermal ? 65 : 100,
        moduleWidth: isThermal ? 1.5 : 2.0,
        fontSize: isThermal ? 11 : 13,
        showText: true,
      });

      for (let i = 0; i < copies; i++) {
        labelsHtml += `
          <div class="label-card">
            ${printIncludeStoreName ? `<div class="store-name">${storeName}</div>` : ''}
            ${printIncludeProductName && item.product_name ? `<div class="product-name">${item.product_name}</div>` : ''}
            ${printIncludeSku && item.product_sku ? `<div class="sku-text">SKU: ${item.product_sku} ${item.product_brand ? `(${item.product_brand})` : ''}</div>` : ''}
            <div class="barcode-wrapper">${svgString}</div>
            ${printIncludeLocation && item.location_name ? `<div class="location-text">Lokasi: ${item.location_name}</div>` : ''}
          </div>
        `;
      }
    }

    printWindow.document.write(`
      <!DOCTYPE html>
      <html lang="id">
      <head>
        <meta charset="UTF-8">
        <title>Cetak Barcode Label IMEI & S/N (${totalLabels} Label)</title>
        <style>
          * { box-sizing: border-box; margin: 0; padding: 0; }
          body {
            font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: white;
            color: #000000;
            padding: ${isThermal ? '0' : '8mm 5mm'};
            -webkit-print-color-adjust: exact;
            print-color-adjust: exact;
          }
          @page {
            size: ${isThermal ? '50mm 35mm' : 'A4 portrait'};
            margin: ${isThermal ? '2mm' : '8mm 5mm'};
          }
          .container {
            display: flex;
            flex-wrap: wrap;
            gap: ${isThermal ? '0' : '4mm'};
            justify-content: ${isThermal ? 'center' : 'flex-start'};
            align-content: flex-start;
          }
          .label-card {
            width: ${isThermal ? '48mm' : '64mm'};
            height: ${isThermal ? '32mm' : '40mm'};
            padding: ${isThermal ? '2mm 2.5mm' : '2.5mm 3.5mm'};
            border: ${isThermal ? 'none' : '1px dashed #94a3b8'};
            page-break-inside: avoid;
            break-inside: avoid;
            ${isThermal ? 'page-break-after: always; break-after: page;' : ''}
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: space-between;
            text-align: center;
            overflow: hidden;
            background: #ffffff;
          }
          .store-name {
            font-size: ${isThermal ? '7.5px' : '8.5px'};
            font-weight: 800;
            letter-spacing: 1.2px;
            text-transform: uppercase;
            color: #000000;
            line-height: 1;
            margin-bottom: 0.5mm;
          }
          .product-name {
            font-size: ${isThermal ? '8.5px' : '10px'};
            font-weight: 700;
            line-height: 1.15;
            max-height: ${isThermal ? '18px' : '22px'};
            overflow: hidden;
            color: #000000;
            margin-bottom: 0.5mm;
          }
          .sku-text {
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
            font-size: ${isThermal ? '7.5px' : '8.5px'};
            color: #000000;
            margin-bottom: 0.5mm;
          }
          .barcode-wrapper {
            display: flex;
            justify-content: center;
            align-items: center;
            width: 100%;
            flex: 1;
            margin: 0.5mm 0;
          }
          .barcode-wrapper svg {
            max-width: ${isThermal ? '44mm' : '58mm'};
            max-height: ${isThermal ? '16mm' : '23mm'};
            min-height: ${isThermal ? '12mm' : '18mm'};
            width: 100%;
            height: auto;
          }
          .location-text {
            font-size: ${isThermal ? '7px' : '8px'};
            color: #404040;
            margin-top: 0.5mm;
          }
          @media screen {
            body { background: #f3f4f6; padding: 20px; }
            .label-card { background: white; margin-bottom: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
          }
        </style>
      </head>
      <body>
        <div class="container">
          ${labelsHtml}
        </div>
        <script>
          window.onload = function() {
            window.focus();
            window.print();
            window.onafterprint = function() { window.close(); };
          };
        <\/script>
      </body>
      </html>
    `);

    printWindow.document.close();
    open = false;
  }
</script>

<Modal bind:open title="Cetak Barcode Label IMEI / S/N" size="xl" onclose={handleClose}>
  {#if items.length > 0 && currentPreviewItem}
    <div class="space-y-5">
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Kolom Kiri: Konfigurasi & Opsi Cetak -->
        <div class="space-y-4">
          <!-- Ringkasan Unit -->
          <div class="rounded-xl border border-neutral-200 bg-neutral-50/75 p-3.5">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-neutral-800">Unit Siap Cetak</span>
              <Badge variant="indigo" size="sm">
                {items.length} Unit
              </Badge>
            </div>
            <div class="mt-2 text-xs text-neutral-600">
              {#if items.length === 1}
                <div class="font-mono text-sm font-bold text-neutral-900">
                  {currentPreviewItem.serial_number}
                </div>
                {#if currentPreviewItem.product_name}
                  <div class="mt-0.5 line-clamp-1 text-neutral-700">
                    {currentPreviewItem.product_name}
                  </div>
                {/if}
              {:else}
                <div class="text-neutral-700">
                  Mencetak label stiker serial batch ({items.length} unit unik).
                </div>
                <div
                  class="mt-1 flex items-center justify-between border-t border-neutral-200 pt-2 text-[11px] text-neutral-500"
                >
                  <span
                    >Pratinjau: <strong class="font-mono text-neutral-800"
                      >{currentPreviewItem.serial_number}</strong
                    ></span
                  >
                  <span>({activePreviewIndex + 1} dari {items.length})</span>
                </div>
                <div class="mt-2 flex items-center gap-1.5">
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() => (activePreviewIndex = Math.max(0, activePreviewIndex - 1))}
                    disabled={activePreviewIndex === 0}
                  >
                    Sebelumnya
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() =>
                      (activePreviewIndex = Math.min(items.length - 1, activePreviewIndex + 1))}
                    disabled={activePreviewIndex >= items.length - 1}
                  >
                    Berikutnya
                  </Button>
                </div>
              {/if}
            </div>
          </div>

          <!-- Salinan per Unit -->
          <div>
            <div class="mb-1.5 flex items-center justify-between">
              <span class="text-xs font-semibold text-neutral-900">Salinan per Unit</span>
              <span class="text-[11px] text-neutral-500">
                Total keluaran: <strong class="font-bold text-neutral-900"
                  >{totalLabels} label</strong
                >
              </span>
            </div>
            <div class="flex items-center gap-2">
              <input
                type="number"
                min="1"
                max="100"
                bind:value={printCopies}
                class="w-24 rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-sm font-semibold text-neutral-900 transition focus:border-neutral-900 focus:ring-1 focus:ring-neutral-900 focus:outline-hidden"
              />
              {#each [1, 2, 3, 5] as qty (qty)}
                <button
                  type="button"
                  onclick={() => (printCopies = qty)}
                  class="rounded-md border px-2.5 py-1 text-xs font-medium transition {printCopies ===
                  qty
                    ? 'border-neutral-900 bg-neutral-900 text-white shadow-2xs'
                    : 'border-neutral-200 bg-white text-neutral-600 hover:border-neutral-300'}"
                >
                  {qty}x
                </button>
              {/each}
            </div>
          </div>

          <!-- Format Kertas / Mesin Cetak -->
          <div>
            <span class="mb-1.5 block text-xs font-semibold text-neutral-900"
              >Format Kertas / Printer</span
            >
            <div class="grid grid-cols-2 gap-2">
              <button
                type="button"
                onclick={() => (printFormat = 'single')}
                class="flex flex-col items-start rounded-xl border p-3 text-left transition {printFormat ===
                'single'
                  ? 'border-neutral-900 bg-neutral-50/80 ring-2 ring-neutral-900/10'
                  : 'border-neutral-200 bg-white hover:border-neutral-300'}"
              >
                <span class="text-xs font-bold text-neutral-900">Stiker Thermal Roll</span>
                <span class="mt-0.5 text-[11px] text-neutral-500">Printer label POS (50x35mm)</span>
              </button>

              <button
                type="button"
                onclick={() => (printFormat = 'sheet')}
                class="flex flex-col items-start rounded-xl border p-3 text-left transition {printFormat ===
                'sheet'
                  ? 'border-neutral-900 bg-neutral-50/80 ring-2 ring-neutral-900/10'
                  : 'border-neutral-200 bg-white hover:border-neutral-300'}"
              >
                <span class="text-xs font-bold text-neutral-900">Lembar Kertas A4</span>
                <span class="mt-0.5 text-[11px] text-neutral-500">Grid multi-label laser/inkjet</span
                >
              </button>
            </div>
          </div>

          <!-- Toggle Konten Label -->
          <div>
            <span class="mb-1.5 block text-xs font-semibold text-neutral-900"
              >Informasi pada Label</span
            >
            <div class="space-y-1.5 rounded-xl border border-neutral-200 bg-white p-3 text-xs">
              <label class="flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  bind:checked={printIncludeStoreName}
                  class="rounded text-neutral-900 focus:ring-neutral-900"
                />
                <span>Nama Brand Toko (GEN-E RETAIL)</span>
              </label>
              <label class="flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  bind:checked={printIncludeProductName}
                  class="rounded text-neutral-900 focus:ring-neutral-900"
                />
                <span>Nama Produk Fisik</span>
              </label>
              <label class="flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  bind:checked={printIncludeSku}
                  class="rounded text-neutral-900 focus:ring-neutral-900"
                />
                <span>Kode SKU & Brand Produk</span>
              </label>
              <label class="flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  bind:checked={printIncludeLocation}
                  class="rounded text-neutral-900 focus:ring-neutral-900"
                />
                <span>Lokasi Gudang / Cabang</span>
              </label>
            </div>
          </div>
        </div>

        <!-- Kolom Kanan: Pratinjau Stiker -->
        <div
          class="flex flex-col items-center justify-center rounded-xl border border-neutral-200/80 bg-neutral-100/70 p-6"
        >
          <div class="mb-3 flex w-full items-center justify-between px-1">
            <span class="text-[11px] font-bold tracking-wider text-neutral-500 uppercase">
              Pratinjau Fisik Label
            </span>
            <span class="text-[11px] font-medium text-neutral-600">
              {printFormat === 'single' ? 'Thermal (50x35mm)' : 'Lembar A4 HVS (64x40mm)'}
            </span>
          </div>

          <!-- Kotak Label Fisik Simulasi -->
          <div
            class="relative flex flex-col items-center justify-between rounded-lg border-2 border-dashed border-neutral-300 bg-white p-3 shadow-md transition-all {printFormat === 'single' ? 'w-64 min-h-[160px]' : 'w-80 min-h-[190px]'}"
          >
            {#if printIncludeStoreName}
              <div
                class="mb-0.5 text-center text-[10px] font-extrabold tracking-widest text-neutral-800 uppercase"
              >
                GEN-E RETAIL
              </div>
            {/if}

            {#if printIncludeProductName && currentPreviewItem.product_name}
              <div
                class="mb-0.5 max-h-8 text-center font-bold leading-tight text-neutral-900 line-clamp-2 {printFormat === 'single' ? 'text-[11px]' : 'text-xs'}"
              >
                {currentPreviewItem.product_name}
              </div>
            {/if}

            {#if printIncludeSku && currentPreviewItem.product_sku}
              <div class="mb-1 text-center font-mono text-[9px] text-neutral-600">
                SKU: {currentPreviewItem.product_sku}
                {currentPreviewItem.product_brand ? `(${currentPreviewItem.product_brand})` : ''}
              </div>
            {/if}

            <div class="my-1 flex w-full justify-center">
              <Barcode
                value={currentPreviewItem.serial_number}
                height={printFormat === 'single' ? 55 : 85}
                moduleWidth={printFormat === 'single' ? 1.5 : 2.0}
                fontSize={printFormat === 'single' ? 11 : 13}
                showText={true}
                class={printFormat === 'single' ? 'max-w-[220px]' : 'max-w-[270px]'}
              />
            </div>

            {#if printIncludeLocation && currentPreviewItem.location_name}
              <div class="mt-1 text-center text-[9px] text-neutral-600">
                Lokasi: {currentPreviewItem.location_name}
              </div>
            {/if}
          </div>

          <p class="mt-4 max-w-xs text-center text-[11px] text-neutral-500">
            {printFormat === 'single'
              ? 'Tata letak dan ukuran garis barcode terkalibrasi otomatis untuk printer roll thermal POS.'
              : 'Format A4 HVS diperbesar dengan garis lebih tebal dan tinggi agar barcode terbaca jelas oleh pemindai barcode di kertas HVS.'}
          </p>
        </div>
      </div>
    </div>
  {:else}
    <div class="py-8 text-center text-xs text-neutral-500">
      Tidak ada unit nomor seri yang dipilih untuk dicetak.
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2.5">
      <Button variant="outline" size="sm" onclick={handleClose}>Batal</Button>
      <Button
        variant="primary"
        size="sm"
        disabled={items.length === 0}
        onclick={executePrint}
      >
        <svg class="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.75"
            d="M6.72 13.829c-.24.03-.48.062-.72.096m.72-.096a42.415 42.415 0 0110.56 0m-10.56 0L6.34 18m10.94-4.171c.24.03.48.062.72.096m-.72-.096L17.66 18m0 0l.229 2.523a1.125 1.125 0 01-1.12 1.227H7.231c-.662 0-1.18-.568-1.12-1.227L6.34 18m11.318 0h1.091A2.25 2.25 0 0021 15.75V9.456c0-1.081-.768-2.015-1.837-2.175a48.055 48.055 0 00-1.913-.247M6.34 18H5.25A2.25 2.25 0 013 15.75V9.456c0-1.081.768-2.015 1.837-2.175a48.041 48.041 0 011.913-.247m10.5 0a48.536 48.536 0 00-10.5 0m10.5 0V3.375c0-.621-.504-1.125-1.125-1.125h-8.25c-.621 0-1.125.504-1.125 1.125v3.656l10.5 0z"
          />
        </svg>
        Cetak {totalLabels} Label
      </Button>
    </div>
  {/snippet}
</Modal>
