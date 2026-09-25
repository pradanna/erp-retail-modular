<script lang="ts">
  /**
   * Select2 — Komponen Searchable Dropdown / Combobox Reusable (Gen-E Enterprise).
   *
   * Fitur:
   * - Filter pencarian teks langsung (real-time search) di dalam popover
   * - Navigasi keyboard penuh (Arrow Up/Down, Enter untuk memilih, Escape untuk menutup)
   * - Tombol bersihkan pilihan (clearable button)
   * - Mendukung label hierarkis / subtext (misal: "Induk > Sub-kategori")
   * - Deteksi klik luar (click-outside) otomatis
   * - Standar icon Heroicons SVG outline (tanpa emoticon & tanpa sparkle)
   */
  import type { SelectOption } from '@erp/types';

  export interface Select2Option extends SelectOption {
    subtext?: string;
  }

  interface Props {
    id?: string;
    label?: string;
    value?: string | number;
    options: (SelectOption | Select2Option)[];
    placeholder?: string;
    searchPlaceholder?: string;
    error?: string;
    disabled?: boolean;
    required?: boolean;
    showRequiredAsterisk?: boolean;
    clearable?: boolean;
    name?: string;
    class?: string;
    placement?: 'auto' | 'bottom' | 'top';
    onchange?: (val: string | number) => void;
  }

  let {
    id = `select2-${Math.random().toString(36).slice(2, 9)}`,
    label = '',
    value = $bindable(''),
    options = [],
    placeholder = 'Pilih salah satu...',
    searchPlaceholder = 'Ketik untuk mencari...',
    error = '',
    disabled = false,
    required = false,
    showRequiredAsterisk = false,
    clearable = true,
    name = '',
    class: className = '',
    placement = 'auto',
    onchange,
  }: Props = $props();

  let isOpen = $state(false);
  let searchQuery = $state('');
  let highlightedIndex = $state(0);
  let containerElement: HTMLDivElement | null = $state(null);
  let searchInputElement: HTMLInputElement | null = $state(null);
  let popoverElement: HTMLDivElement | null = $state(null);

  let effectivePlacement = $state<'bottom' | 'top'>('bottom');
  let popoverStyle = $state('');
  let popoverMaxHeight = $state(224);

  // Mencari opsi yang sedang dipilih
  const selectedOption = $derived(
    options.find((opt) => String(opt.value) === String(value) && !opt.label.startsWith('--')),
  );

  // Filter opsi berdasarkan kata kunci pencarian
  const filteredOptions = $derived(
    options.filter((opt) => {
      // Lewati jika hanya dummy placeholder berawalan '--'
      if (String(opt.value) === '' && opt.label.startsWith('--')) return false;

      const query = searchQuery.trim().toLowerCase();
      if (!query) return true;

      const labelMatch = opt.label.toLowerCase().includes(query);
      const subtextMatch = (opt as Select2Option).subtext?.toLowerCase().includes(query) ?? false;
      return labelMatch || subtextMatch;
    }),
  );

  function toggleDropdown() {
    if (disabled) return;
    if (isOpen) {
      closeDropdown();
    } else {
      openDropdown();
    }
  }

  function updatePosition() {
    if (!isOpen || !containerElement) return;
    const rect = containerElement.getBoundingClientRect();
    const viewportHeight = window.innerHeight;
    const spaceBelow = viewportHeight - rect.bottom;
    const spaceAbove = rect.top;

    // Jika elemen trigger keluar dari viewport, tutup dropdown
    if (rect.bottom < 0 || rect.top > viewportHeight) {
      closeDropdown();
      return;
    }

    // Deteksi scroll parent terdekat (seperti modal body atau scroll container)
    const scrollParent = containerElement.closest(
      '.overflow-y-auto, .overflow-auto, [role="dialog"]',
    );
    let parentSpaceBelow = spaceBelow;
    let parentSpaceAbove = spaceAbove;

    if (scrollParent) {
      const parentRect = scrollParent.getBoundingClientRect();
      parentSpaceBelow = parentRect.bottom - rect.bottom;
      parentSpaceAbove = rect.top - parentRect.top;
    }

    // Kebutuhan tinggi menu dropdown (input cari 44px + footer info 24px + min 3 opsi 110px = ~178px)
    const minRequiredSpace = 180;
    const idealHeight = 240;

    let chosenPlacement: 'bottom' | 'top' = 'bottom';

    if (placement === 'top') {
      chosenPlacement = 'top';
    } else if (placement === 'bottom') {
      chosenPlacement = 'bottom';
    } else {
      // Mode 'auto':
      // Jika ruang di dalam parent sempit (< minRequiredSpace) dan ruang atas lebih luas,
      // ATAU jika ruang viewport di bawah sempit dan ruang atas lebih luas:
      if (parentSpaceBelow < minRequiredSpace && parentSpaceAbove > parentSpaceBelow) {
        chosenPlacement = 'top';
      } else if (spaceBelow < minRequiredSpace && spaceAbove > spaceBelow) {
        chosenPlacement = 'top';
      } else {
        chosenPlacement = 'bottom';
      }
    }

    effectivePlacement = chosenPlacement;

    if (chosenPlacement === 'top') {
      const availableHeight = Math.max(120, Math.min(idealHeight, spaceAbove - 20));
      popoverMaxHeight = availableHeight;
      const bottomPx = viewportHeight - rect.top + 6;
      popoverStyle = `position: fixed; left: ${rect.left}px; width: ${rect.width}px; bottom: ${bottomPx}px; z-index: 100;`;
    } else {
      const availableHeight = Math.max(120, Math.min(idealHeight, spaceBelow - 20));
      popoverMaxHeight = availableHeight;
      const topPx = rect.bottom + 6;
      popoverStyle = `position: fixed; left: ${rect.left}px; width: ${rect.width}px; top: ${topPx}px; z-index: 100;`;
    }
  }

  $effect(() => {
    if (isOpen) {
      updatePosition();
      const handleScrollOrResize = () => {
        updatePosition();
      };
      window.addEventListener('scroll', handleScrollOrResize, true);
      window.addEventListener('resize', handleScrollOrResize);
      return () => {
        window.removeEventListener('scroll', handleScrollOrResize, true);
        window.removeEventListener('resize', handleScrollOrResize);
      };
    }
  });

  function openDropdown() {
    if (disabled) return;
    isOpen = true;
    searchQuery = '';
    highlightedIndex = 0;
    updatePosition();
    setTimeout(() => {
      updatePosition();
      searchInputElement?.focus();
    }, 20);
  }

  function closeDropdown() {
    isOpen = false;
    searchQuery = '';
    highlightedIndex = 0;
  }

  function selectOption(opt: SelectOption | Select2Option) {
    if (opt.disabled) return;
    value = opt.value;
    onchange?.(opt.value);
    closeDropdown();
  }

  function clearSelection(e: MouseEvent) {
    e.stopPropagation();
    if (disabled) return;
    value = '';
    onchange?.('');
  }

  function handleWindowClick(e: MouseEvent) {
    if (!isOpen) return;
    const target = e.target as Node;
    if (
      containerElement &&
      !containerElement.contains(target) &&
      (!popoverElement || !popoverElement.contains(target))
    ) {
      closeDropdown();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (!isOpen) {
      if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
        e.preventDefault();
        openDropdown();
      }
      return;
    }

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        if (filteredOptions.length > 0) {
          highlightedIndex = (highlightedIndex + 1) % filteredOptions.length;
        }
        break;
      case 'ArrowUp':
        e.preventDefault();
        if (filteredOptions.length > 0) {
          highlightedIndex =
            (highlightedIndex - 1 + filteredOptions.length) % filteredOptions.length;
        }
        break;
      case 'Enter':
        e.preventDefault();
        if (filteredOptions.length > 0 && filteredOptions[highlightedIndex]) {
          selectOption(filteredOptions[highlightedIndex]);
        }
        break;
      case 'Escape':
      case 'Tab':
        closeDropdown();
        break;
    }
  }
</script>

<svelte:window onclick={handleWindowClick} />

<div class="relative w-full {className}" bind:this={containerElement}>
  <!-- Input Tersembunyi untuk Form Submit HTML Biasa -->
  {#if name}
    <input type="hidden" {name} {value} />
  {/if}

  {#if label}
    <label for={id} class="mb-1.5 block text-xs font-medium text-neutral-700">
      {label}
      {#if required && showRequiredAsterisk}
        <span class="text-rose-500">*</span>
      {/if}
    </label>
  {/if}

  <!-- Kotak Trigger Utama -->
  <button
    type="button"
    {id}
    {disabled}
    onclick={toggleDropdown}
    onkeydown={handleKeydown}
    aria-haspopup="listbox"
    aria-expanded={isOpen}
    class="flex h-10 w-full items-center justify-between rounded-xl border bg-white px-3.5 text-xs text-neutral-900 shadow-2xs transition-all focus:outline-hidden disabled:cursor-not-allowed disabled:bg-neutral-100 disabled:text-neutral-400 {error
      ? 'border-rose-400 focus:border-rose-500 focus:ring-2 focus:ring-rose-500/20'
      : isOpen
        ? 'border-primary-500 ring-primary-500/20 ring-2'
        : 'border-neutral-300 hover:border-neutral-400'}"
  >
    <!-- Teks Terpilih / Placeholder -->
    <div class="flex min-w-0 flex-1 items-center gap-2 overflow-hidden text-left">
      {#if selectedOption}
        <span class="truncate font-medium text-neutral-900">
          {selectedOption.label}
        </span>
        {#if (selectedOption as Select2Option).subtext}
          <span class="truncate text-[10px] text-neutral-400">
            ({(selectedOption as Select2Option).subtext})
          </span>
        {/if}
      {:else}
        <span class="truncate text-neutral-400">{placeholder}</span>
      {/if}
    </div>

    <!-- Aksi Kanan (Clear & Caret) -->
    <div class="ml-2 flex items-center gap-1">
      {#if clearable && selectedOption && String(selectedOption.value) !== '' && !disabled}
        <span
          role="button"
          tabindex="0"
          class="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 focus:outline-hidden"
          title="Kosongkan pilihan"
          onclick={clearSelection}
          onkeydown={(e) =>
            (e.key === 'Enter' || e.key === ' ') && clearSelection(e as unknown as MouseEvent)}
        >
          <svg
            class="h-3.5 w-3.5"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </span>
      {/if}

      <span class="text-neutral-400 transition-transform duration-200 {isOpen ? 'rotate-180' : ''}">
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
        </svg>
      </span>
    </div>
  </button>

  <!-- Popover Menu Pencarian & Daftar Opsi (Floating Fixed Coordinates) -->
  {#if isOpen}
    <div
      bind:this={popoverElement}
      style={popoverStyle}
      class="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-2xl transition-all"
    >
      <!-- Bar Pencarian Real-Time -->
      <div class="border-b border-neutral-100 bg-neutral-50/70 p-2">
        <div class="relative flex items-center">
          <span class="pointer-events-none absolute left-2.5 text-neutral-400">
            <svg
              class="h-3.5 w-3.5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          </span>

          <input
            type="text"
            bind:this={searchInputElement}
            bind:value={searchQuery}
            onkeydown={handleKeydown}
            placeholder={searchPlaceholder}
            class="focus:border-primary-500 focus:ring-primary-500/20 w-full rounded-lg border border-neutral-200 bg-white py-1.5 pr-2.5 pl-8 text-xs text-neutral-900 placeholder-neutral-400 focus:ring-1 focus:outline-hidden"
          />

          {#if searchQuery}
            <button
              type="button"
              aria-label="Bersihkan pencarian"
              title="Bersihkan pencarian"
              class="absolute right-2 text-neutral-400 hover:text-neutral-600 focus:outline-hidden"
              onclick={() => {
                searchQuery = '';
                searchInputElement?.focus();
              }}
            >
              <svg
                class="h-3 w-3"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          {/if}
        </div>
      </div>

      <!-- Daftar Opsi -->
      <div
        style="max-height: {popoverMaxHeight}px;"
        class="overflow-y-auto p-1 text-xs"
        role="listbox"
        aria-label={label || 'Daftar pilihan'}
      >
        {#if filteredOptions.length > 0}
          {#each filteredOptions as opt, idx (opt.value)}
            {@const isSelected = String(opt.value) === String(value)}
            {@const isHighlighted = idx === highlightedIndex}
            <button
              type="button"
              role="option"
              aria-selected={isSelected}
              disabled={opt.disabled}
              onclick={() => selectOption(opt)}
              onmouseenter={() => (highlightedIndex = idx)}
              class="flex w-full items-center justify-between rounded-lg px-2.5 py-2 text-left transition-colors {opt.disabled
                ? 'cursor-not-allowed opacity-40'
                : isSelected
                  ? 'bg-primary-50 text-primary-900 font-semibold'
                  : isHighlighted
                    ? 'bg-neutral-100 text-neutral-900'
                    : 'text-neutral-700 hover:bg-neutral-50'}"
            >
              <div class="min-w-0 flex-1">
                <div class="truncate">{opt.label}</div>
                {#if (opt as Select2Option).subtext}
                  <div class="mt-0.5 truncate text-[10px] text-neutral-400">
                    {(opt as Select2Option).subtext}
                  </div>
                {/if}
              </div>

              {#if isSelected}
                <span class="text-primary-600 ml-2 shrink-0">
                  <svg
                    class="h-4 w-4"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </span>
              {/if}
            </button>
          {/each}
        {:else}
          <div class="px-3 py-6 text-center text-xs text-neutral-400">
            Tidak ada hasil yang cocok dengan "{searchQuery}"
          </div>
        {/if}
      </div>

      <!-- Footer Info Jumlah Opsi -->
      <div
        class="flex items-center justify-between border-t border-neutral-100 bg-neutral-50/60 px-3 py-1 text-[10px] text-neutral-400"
      >
        <span>{filteredOptions.length} opsi tersedia</span>
        <span class="hidden sm:inline">Gunakan ↑↓ dan Enter</span>
      </div>
    </div>
  {/if}

  {#if error}
    <p class="mt-1 text-xs text-rose-600">{error}</p>
  {/if}
</div>
