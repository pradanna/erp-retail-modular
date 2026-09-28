<script lang="ts">
  /**
   * Barcode — Komponen SVG Barcode Code-128 standar industri ritel & logistik.
   *
   * Fitur:
   * - Zero external dependency, berbasis standar resmi ISO/IEC 15417 Code 128-B
   * - Vektor SVG tajam dan jernih pada semua resolusi cetak (Thermal 203 DPI, 300 DPI, hingga Laserjet)
   * - Mendukung label teks monospace di bawah garis barcode
   */
  import { computeCode128 } from './barcode-utils.js';

  interface Props {
    value: string;
    height?: number;
    moduleWidth?: number;
    showText?: boolean;
    fontSize?: number;
    class?: string;
  }

  let {
    value = '',
    height = 48,
    moduleWidth = 2,
    showText = true,
    fontSize = 11,
    class: className = '',
  }: Props = $props();

  const barcodeData = $derived.by(() => computeCode128(value, moduleWidth));
  const textPadding = $derived(Math.round(fontSize * 1.35));
  const totalHeight = $derived(height + (showText ? textPadding : 0));
</script>

{#if barcodeData.bars.length > 0}
  <svg
    viewBox="0 0 {barcodeData.totalWidth} {totalHeight}"
    width={barcodeData.totalWidth}
    height={totalHeight}
    class="select-none {className}"
    style="max-width: 100%; height: auto;"
    aria-label="Barcode {value}"
  >
    <!-- Garis Barcode Hitam Pekat Murni -->
    {#each barcodeData.bars as bar}
      <rect x={bar.x} y={0} width={bar.width} {height} fill="#000000" />
    {/each}

    <!-- Teks Human Readable di Bawah Barcode -->
    {#if showText}
      <text
        x={barcodeData.totalWidth / 2}
        y={height + fontSize}
        text-anchor="middle"
        font-family="ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
        font-size="{fontSize}"
        font-weight="600"
        letter-spacing="1.5"
        fill="#000000"
      >
        {value}
      </text>
    {/if}
  </svg>
{/if}
