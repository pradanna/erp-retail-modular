/**
 * Root Layout Load — SPA Mode Configuration.
 *
 * export const ssr = false → Membuat SELURUH aplikasi backoffice menjadi SPA (Single Page Application).
 * Artinya: semua rendering terjadi di browser (client-side), bukan di server.
 *
 * Mengapa SPA untuk Backoffice?
 * - Backoffice adalah aplikasi internal (bukan publik), SEO tidak diperlukan.
 * - SPA memberikan navigasi lebih cepat (tanpa full page reload).
 * - Lebih sederhana untuk mengelola auth state di client.
 *
 * Analoginya: SSR = restoran mengantarkan makanan jadi ke meja pelanggan.
 * SPA = restoran memberikan bahan dan resep, pelanggan memasak sendiri di meja.
 * Untuk dashboard internal, "memasak sendiri" lebih efisien.
 */
export const ssr = false;
