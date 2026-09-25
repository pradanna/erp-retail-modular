<script lang="ts">
  /**
   * Barcode — Komponen SVG Barcode Code-128 standar industri ritel & logistik.
   *
   * Fitur:
   * - Zero external dependency, berbasis standar resmi ISO/IEC 15417 Code 128-B
   * - Vektor SVG tajam dan jernih pada semua resolusi cetak (Thermal 203 DPI, 300 DPI, hingga Laserjet)
   * - Mendukung label teks monospace di bawah garis barcode
   */
  interface Props {
    value: string;
    height?: number;
    moduleWidth?: number;
    showText?: boolean;
    class?: string;
  }

  let {
    value = '',
    height = 48,
    moduleWidth = 2,
    showText = true,
    class: className = '',
  }: Props = $props();

  // Pola garis Code-128 (0 hingga 106)
  // Setiap digit melambangkan ketebalan garis (bar) dan spasi (space) secara bergantian
  const CODE128_PATTERNS: readonly string[] = [
    '212222', '222122', '222221', '121223', '121322', '131222', '122213', '122312', '132212', '221213', // 0-9
    '221312', '231212', '112232', '122132', '122231', '113222', '123122', '123221', '223211', '221132', // 10-19
    '221231', '213212', '223112', '312131', '311222', '321122', '321221', '312212', '322112', '322211', // 20-29
    '212123', '212321', '232121', '111323', '131123', '131321', '112313', '132113', '132311', '211313', // 30-39
    '231113', '231311', '112133', '112331', '132131', '113123', '113321', '133121', '313121', '211331', // 40-49
    '231131', '213113', '213311', '213131', '311123', '311321', '331121', '312113', '312311', '332111', // 50-59
    '314111', '221411', '431111', '111224', '111422', '121124', '121421', '141122', '141221', '112214', // 60-69
    '112412', '122114', '122411', '142112', '142211', '241211', '221114', '413111', '241112', '134111', // 70-79
    '111242', '121142', '121241', '114212', '124112', '124211', '411212', '421112', '421211', '212141', // 80-89
    '214121', '412121', '111143', '111341', '131141', '114113', '114311', '411113', '411311', '113141', // 90-99
    '114131', '311141', '411131', '211412', '211214', '211232', '2331112' // 100-106 (106 adalah Stop Code)
  ];

  interface Bar {
    x: number;
    width: number;
  }

  const barcodeData = $derived.by(() => {
    const cleanText = value.trim();
    if (!cleanText) {
      return { bars: [], totalWidth: 0 };
    }

    // Menggunakan Start Code B (indeks 104)
    const symbols: number[] = [104];
    let checksum = 104;

    for (let i = 0; i < cleanText.length; i++) {
      const code = cleanText.charCodeAt(i);
      // Validasi karakter dalam rentang ASCII Code-128B (32-126)
      const val = code >= 32 && code <= 126 ? code - 32 : 0;
      symbols.push(val);
      checksum += (i + 1) * val;
    }

    // Tambahkan Checksum dan Stop Code (106)
    symbols.push(checksum % 103);
    symbols.push(106);

    // Hitung posisi horizontal (x) dan lebar setiap garis hitam
    const bars: Bar[] = [];
    let currentX = 10; // Margin kiri (quiet zone)

    for (const sym of symbols) {
      const pattern = CODE128_PATTERNS[sym] || CODE128_PATTERNS[0];
      for (let j = 0; j < pattern.length; j++) {
        const span = parseInt(pattern[j], 10) * moduleWidth;
        if (j % 2 === 0) {
          // Indeks genap melambangkan bar hitam
          bars.push({ x: currentX, width: span });
        }
        currentX += span;
      }
    }

    currentX += 10; // Margin kanan (quiet zone)

    return {
      bars,
      totalWidth: currentX,
    };
  });
</script>

{#if barcodeData.bars.length > 0}
  <svg
    viewBox="0 0 {barcodeData.totalWidth} {height + (showText ? 16 : 0)}"
    width={barcodeData.totalWidth}
    height={height + (showText ? 16 : 0)}
    class="select-none {className}"
    style="max-width: 100%; height: auto;"
    aria-label="Barcode {value}"
  >
    <!-- Garis Barcode Hitam -->
    {#each barcodeData.bars as bar}
      <rect x={bar.x} y={0} width={bar.width} {height} fill="black" />
    {/each}

    <!-- Teks Human Readable di Bawah Barcode -->
    {#if showText}
      <text
        x={barcodeData.totalWidth / 2}
        y={height + 12}
        text-anchor="middle"
        font-family="ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
        font-size="11"
        font-weight="600"
        letter-spacing="1.5"
        fill="#171717"
      >
        {value}
      </text>
    {/if}
  </svg>
{/if}
