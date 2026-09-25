<script lang="ts">
  /**
   * SearchInput — Input pencarian cepat dengan debouncing otomatis.
   *
   * Props:
   * - value: string ($bindable) — Kata kunci pencarian
   * - placeholder: string — Teks petunjuk (default: 'Cari data...')
   * - debounceMs: number — Waktu jeda debouncing dalam milidetik (default: 300)
   * - disabled: boolean — Status dinonaktifkan
   * - onsearch?: (query: string) => void — Callback saat pencarian dipicu
   */
  interface Props {
    value?: string;
    placeholder?: string;
    debounceMs?: number;
    disabled?: boolean;
    onsearch?: (query: string) => void;
    class?: string;
  }

  let {
    value = $bindable(''),
    placeholder = 'Cari data...',
    debounceMs = 300,
    disabled = false,
    onsearch,
    class: className = '',
  }: Props = $props();

  let timer: ReturnType<typeof setTimeout> | null = null;

  function handleInput(e: Event) {
    const target = e.target as HTMLInputElement;
    value = target.value;

    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      onsearch?.(value);
    }, debounceMs);
  }

  function handleClear() {
    value = '';
    if (timer) clearTimeout(timer);
    onsearch?.('');
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      if (timer) clearTimeout(timer);
      onsearch?.(value);
    }
  }
</script>

<div class="relative flex w-full {className || 'max-w-sm'} items-center">
  <!-- Search Icon -->
  <div
    class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5 text-neutral-400"
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
      <path d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
    </svg>
  </div>

  <input
    type="text"
    {placeholder}
    {disabled}
    {value}
    oninput={handleInput}
    onkeydown={handleKeydown}
    class="focus:border-primary-500 focus:ring-primary-500/10 h-10 w-full rounded-xl border border-neutral-200 bg-white pr-8 pl-9 text-xs text-neutral-900 shadow-2xs transition-all duration-150 placeholder:text-neutral-400 hover:border-neutral-300 focus:ring-4 focus:outline-hidden disabled:cursor-not-allowed disabled:bg-neutral-50"
  />

  <!-- Clear Button saat ada teks -->
  {#if value}
    <button
      type="button"
      onclick={handleClear}
      class="absolute inset-y-0 right-0 flex items-center pr-2.5 text-neutral-400 hover:text-neutral-600 focus:outline-hidden"
      aria-label="Bersihkan pencarian"
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
        <path d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  {/if}
</div>
