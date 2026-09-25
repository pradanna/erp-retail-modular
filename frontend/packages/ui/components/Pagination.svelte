<script lang="ts">
  /**
   * Pagination — Kontrol navigasi nomor halaman untuk tabel data.
   *
   * Props:
   * - page: number — Halaman saat ini (1-indexed)
   * - totalPages: number — Total jumlah halaman
   * - totalItems?: number — Total jumlah baris data keseluruhan
   * - limit?: number — Jumlah baris per halaman (default: 10)
   * - onPageChange: (newPage: number) => void — Callback saat pengguna berpindah halaman
   */
  interface Props {
    page: number;
    totalPages: number;
    totalItems?: number;
    limit?: number;
    onPageChange: (newPage: number) => void;
  }

  let { page = 1, totalPages = 1, totalItems, limit = 10, onPageChange }: Props = $props();

  const startItem = $derived(totalItems ? (page - 1) * limit + 1 : 0);
  const endItem = $derived(totalItems ? Math.min(page * limit, totalItems) : 0);

  // Buat array daftar tombol nomor halaman
  const pagesList = $derived((): (number | '...')[] => {
    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, i) => i + 1);
    }
    if (page <= 3) {
      return [1, 2, 3, 4, '...', totalPages];
    }
    if (page >= totalPages - 2) {
      return [1, '...', totalPages - 3, totalPages - 2, totalPages - 1, totalPages];
    }
    return [1, '...', page - 1, page, page + 1, '...', totalPages];
  });
</script>

{#if totalPages > 0}
  <div
    class="flex flex-col items-center justify-between gap-3 border-t border-neutral-200 bg-white px-4 py-3 sm:flex-row sm:px-6"
  >
    <!-- Informasi Jumlah Data -->
    <div class="text-xs text-neutral-500">
      {#if totalItems !== undefined}
        Menampilkan <span class="font-semibold text-neutral-800">{startItem}</span> -
        <span class="font-semibold text-neutral-800">{endItem}</span> dari
        <span class="font-semibold text-neutral-800">{totalItems}</span> data
      {:else}
        Halaman <span class="font-semibold text-neutral-800">{page}</span> dari
        <span class="font-semibold text-neutral-800">{totalPages}</span>
      {/if}
    </div>

    <!-- Tombol Navigasi Halaman -->
    <div class="flex items-center gap-1">
      <!-- Tombol Prev -->
      <button
        type="button"
        onclick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-600 transition-colors hover:bg-neutral-50 hover:text-neutral-900 focus:outline-hidden disabled:cursor-not-allowed disabled:opacity-40"
        aria-label="Halaman sebelumnya"
      >
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M15.75 19.5L8.25 12l7.5-7.5" />
        </svg>
      </button>

      <!-- Nomor Halaman -->
      {#each pagesList() as p}
        {#if p === '...'}
          <span class="inline-flex h-8 w-8 items-center justify-center text-xs text-neutral-400"
            >...</span
          >
        {:else}
          <button
            type="button"
            onclick={() => onPageChange(p)}
            class="inline-flex h-8 min-w-[2rem] items-center justify-center rounded-lg px-2 text-xs font-semibold transition-colors focus:outline-hidden {p ===
            page
              ? 'bg-primary-600 text-white shadow-2xs'
              : 'border border-neutral-200 bg-white text-neutral-700 hover:bg-neutral-50'}"
          >
            {p}
          </button>
        {/if}
      {/each}

      <!-- Tombol Next -->
      <button
        type="button"
        onclick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-600 transition-colors hover:bg-neutral-50 hover:text-neutral-900 focus:outline-hidden disabled:cursor-not-allowed disabled:opacity-40"
        aria-label="Halaman berikutnya"
      >
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M8.25 4.5l7.5 7.5-7.5 7.5" />
        </svg>
      </button>
    </div>
  </div>
{/if}
