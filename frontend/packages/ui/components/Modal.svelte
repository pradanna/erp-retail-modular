<script lang="ts">
  /**
   * Modal — Komponen dialog popup reusable dengan overlay backdrop.
   *
   * Props:
   * - open: boolean (bindable) — status tampil/sembunyi modal
   * - title: string — judul header modal
   * - size: 'sm' | 'md' | 'lg' | 'xl' (default: 'md')
   * - onclose: () => void — callback saat modal ditutup
   * - children: Snippet konten utama modal
   * - footer: Snippet tombol aksi di bagian bawah (opsional)
   */
  import type { Snippet } from 'svelte';

  interface Props {
    open?: boolean;
    title: string;
    size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | '5xl';
    onclose?: () => void;
    children: Snippet;
    footer?: Snippet;
  }

  let { open = $bindable(false), title, size = 'md', onclose, children, footer }: Props = $props();

  const sizeClasses: Record<string, string> = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-2xl',
    '2xl': 'max-w-4xl',
    '3xl': 'max-w-5xl',
    '4xl': 'max-w-6xl',
    '5xl': 'max-w-7xl',
  };

  function handleClose() {
    open = false;
    onclose?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      handleClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <!-- Backdrop Blur Overlay -->
    <div
      class="fixed inset-0 bg-neutral-900/50 backdrop-blur-xs transition-opacity"
      onclick={handleClose}
      onkeydown={(e) => e.key === 'Escape' && handleClose()}
      role="button"
      tabindex="0"
      aria-label="Tutup dialog modal"
    ></div>

    <!-- Modal Dialog Window -->
    <div
      class="relative z-10 w-full overflow-hidden rounded-2xl bg-white shadow-2xl transition-all {sizeClasses[
        size
      ] ?? sizeClasses.md}"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-neutral-200 px-6 py-4.5">
        <h3 id="modal-title" class="text-base font-semibold text-neutral-900">
          {title}
        </h3>
        <button
          type="button"
          class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 focus:outline-hidden"
          onclick={handleClose}
          aria-label="Tutup modal"
        >
          <svg
            class="h-5 w-5"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Content Body -->
      <div class="max-h-[calc(100vh-10rem)] overflow-y-auto px-6 py-5 text-sm text-neutral-600">
        {@render children()}
      </div>

      <!-- Footer Actions -->
      {#if footer}
        <div
          class="flex items-center justify-end gap-3 border-t border-neutral-200 bg-neutral-50 px-6 py-3.5"
        >
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}
