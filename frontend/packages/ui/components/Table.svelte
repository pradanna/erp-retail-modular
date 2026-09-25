<script lang="ts">
  /**
   * Table — Wrapper tabel responsif reusable untuk Backoffice.
   *
   * Fitur:
   * - Overflow horizontal otomatis untuk layar sempit
   * - Empty state ramah pengguna dengan SVG Heroicons
   * - Loading skeleton state saat data sedang diambil
   * - Rounded corner & border standar enterprise
   */
  import type { Snippet } from 'svelte';

  interface Props {
    empty?: boolean;
    emptyTitle?: string;
    emptyMessage?: string;
    loading?: boolean;
    children: Snippet;
  }

  let {
    empty = false,
    emptyTitle = 'Belum Ada Data',
    emptyMessage = 'Tidak ada catatan yang tersedia untuk ditampilkan.',
    loading = false,
    children,
  }: Props = $props();
</script>

<div class="w-full overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xs">
  <div class="overflow-x-auto">
    <table class="w-full text-left text-xs text-neutral-600">
      {@render children()}
    </table>
  </div>

  {#if loading}
    <div class="flex flex-col items-center justify-center gap-2.5 py-12 text-neutral-400">
      <svg
        class="text-primary-600 h-6 w-6 animate-spin"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path
          class="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
        />
      </svg>
      <span class="text-xs font-medium text-neutral-500">Memuat data...</span>
    </div>
  {:else if empty}
    <div class="flex flex-col items-center justify-center px-4 py-12 text-center">
      <div
        class="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-neutral-100 text-neutral-400"
      >
        <svg
          class="h-6 w-6"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path
            d="M2.25 13.5h3.86a2.25 2.25 0 012.012 1.244l.256.512a2.25 2.25 0 002.013 1.244h3.218a2.25 2.25 0 002.013-1.244l.256-.512a2.25 2.25 0 012.013-1.244h3.859m-19.5.375v4.875A2.25 2.25 0 004.5 21h15a2.25 2.25 0 002.25-2.25v-4.875M2.25 13.5l3.86-7.72A2.25 2.25 0 018.122 4.5h7.756a2.25 2.25 0 012.012 1.28l3.86 7.72"
          />
        </svg>
      </div>
      <h4 class="text-sm font-semibold text-neutral-800">{emptyTitle}</h4>
      <p class="mt-1 max-w-sm text-xs text-neutral-500">{emptyMessage}</p>
    </div>
  {/if}
</div>
