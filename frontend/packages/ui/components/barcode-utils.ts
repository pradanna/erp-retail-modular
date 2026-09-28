/**
 * barcode-utils.ts — Utility murni ISO/IEC 15417 Code 128-B Barcode Generator.
 *
 * Mendukung kalkulasi koordinat garis dan pembuatan string SVG utuh
 * tanpa ketergantungan library pihak ketiga (Zero Dependency).
 */

export interface Bar {
	x: number;
	width: number;
}

export interface BarcodeRenderData {
	bars: Bar[];
	totalWidth: number;
}

export interface Code128SvgOptions {
	height?: number;
	moduleWidth?: number;
	showText?: boolean;
	fontSize?: number;
	className?: string;
}

// Pola garis Code-128 (0 hingga 106)
// Setiap digit melambangkan ketebalan bar hitam dan spasi putih secara bergantian
export const CODE128_PATTERNS: readonly string[] = [
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
	'114131', '311141', '411131', '211412', '211214', '211232', '2331112' // 100-106 (106 = Stop Code)
];

/**
 * Menghitung posisi koordinat batang hitam (bars) dan total lebar barcode.
 */
export function computeCode128(value: string, moduleWidth = 2): BarcodeRenderData {
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
	// Quiet zone minimal 10x module width sesuai standar ISO/IEC 15417
	const quietZone = Math.max(12, Math.round(moduleWidth * 8));
	const bars: Bar[] = [];
	let currentX = quietZone;

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

	currentX += quietZone;

	return {
		bars,
		totalWidth: currentX
	};
}

/**
 * Menghasilkan markup string SVG utuh dari teks barcode.
 * Sangat berguna untuk print preview dan dokumen window.print() tanpa manipulasi DOM.
 */
export function generateCode128Svg(value: string, options: Code128SvgOptions = {}): string {
	const height = options.height ?? 48;
	const moduleWidth = options.moduleWidth ?? 2;
	const showText = options.showText ?? true;
	const fontSize = options.fontSize ?? 12;
	const className = options.className ?? '';

	const data = computeCode128(value, moduleWidth);
	if (data.bars.length === 0) return '';

	const textPadding = Math.round(fontSize * 1.35);
	const totalHeight = height + (showText ? textPadding : 0);
	const barsSvg = data.bars
		.map((bar) => `<rect x="${bar.x}" y="0" width="${bar.width}" height="${height}" fill="#000000" />`)
		.join('');

	const textSvg = showText
		? `<text x="${data.totalWidth / 2}" y="${height + fontSize}" text-anchor="middle" font-family="ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace" font-size="${fontSize}" font-weight="600" letter-spacing="1.5" fill="#000000">${value}</text>`
		: '';

	return `<svg viewBox="0 0 ${data.totalWidth} ${totalHeight}" width="${data.totalWidth}" height="${totalHeight}" class="${className}" style="max-width: 100%; height: auto;" aria-label="Barcode ${value}">${barsSvg}${textSvg}</svg>`;
}
