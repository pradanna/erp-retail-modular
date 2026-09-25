<script lang="ts">
  /**
   * ActionMenu — Komponen menu aksi titik tiga (...) dropdown reusable (Gen-E Enterprise).
   *
   * Fitur:
   * - Trigger ringkas tombol titik tiga vertikal (Heroicons EllipsisVertical)
   * - Smart Floating Placement (position: fixed) agar tidak terpotong oleh overflow-x tabel
   * - Auto-dropup otomatis jika berada dekat dasar layar
   * - Mendukung varian item: 'default' | 'danger' | 'primary'
   * - Pemisah item (divider) dan status disabled
   * - Deteksi klik-luar (click-outside) dan tombol Escape
   */
  import type { Snippet } from 'svelte';

  export interface ActionMenuItem {
    id?: string;
    label: string;
    icon?: string; // Raw SVG HTML snippet (<svg ...>)
    iconSvg?: string; // string path SVG Heroicons (viewBox 0 0 24 24)
    onclick?: () => void;
    onClick?: () => void;
    variant?: 'default' | 'danger' | 'primary';
    disabled?: boolean;
    divider?: boolean;
  }

  interface Props {
    items?: ActionMenuItem[];
    align?: 'right' | 'left';
    title?: string;
    class?: string;
    trigger?: Snippet;
    children?: Snippet;
  }

  let {
    items = [],
    align = 'right',
    title = 'Pilihan Aksi',
    class: className = '',
    trigger,
    children,
  }: Props = $props();

  let isOpen = $state(false);
  let triggerElement: HTMLButtonElement | null = $state(null);
  let menuElement: HTMLDivElement | null = $state(null);
  let menuStyle = $state('');

  function updatePosition() {
    if (!isOpen || !triggerElement) return;
    const rect = triggerElement.getBoundingClientRect();
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    // Jika trigger sudah keluar layar akibat scroll jauh, tutup menu
    if (rect.bottom < 0 || rect.top > viewportHeight) {
      closeMenu();
      return;
    }

    const spaceBelow = viewportHeight - rect.bottom;
    const spaceAbove = rect.top;

    // Perkiraan tinggi menu (36px per item + padding)
    const estimatedHeight = Math.max(120, items.length * 36 + 16);

    let verticalPos = '';
    if (spaceBelow < estimatedHeight && spaceAbove > spaceBelow) {
      // Buka ke atas (Dropup)
      const bottom = viewportHeight - rect.top + 4;
      verticalPos = `bottom: ${bottom}px;`;
    } else {
      // Buka ke bawah
      const top = rect.bottom + 4;
      verticalPos = `top: ${top}px;`;
    }

    let horizontalPos = '';
    if (align === 'right') {
      const right = Math.max(8, viewportWidth - rect.right);
      horizontalPos = `right: ${right}px;`;
    } else {
      const left = Math.max(8, rect.left);
      horizontalPos = `left: ${left}px;`;
    }

    menuStyle = `position: fixed; ${horizontalPos} ${verticalPos} z-index: 50;`;
  }

  function toggleMenu(e: MouseEvent) {
    e.stopPropagation();
    if (isOpen) {
      closeMenu();
    } else {
      openMenu();
    }
  }

  function openMenu() {
    isOpen = true;
    updatePosition();
    setTimeout(() => {
      updatePosition();
    }, 10);
  }

  function closeMenu() {
    isOpen = false;
  }

  function handleItemClick(item: ActionMenuItem) {
    if (item.disabled) return;
    closeMenu();
    if (item.onclick) {
      item.onclick();
    } else if (item.onClick) {
      item.onClick();
    }
  }

  function handleWindowClick(e: MouseEvent) {
    if (!isOpen) return;
    const target = e.target as Node;
    if (
      triggerElement &&
      !triggerElement.contains(target) &&
      (!menuElement || !menuElement.contains(target))
    ) {
      closeMenu();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (!isOpen) return;
    if (e.key === 'Escape' || e.key === 'Tab') {
      closeMenu();
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
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleKeydown} />

<div class="relative inline-block text-left {className}">
  <!-- Tombol Trigger Titik Tiga -->
  {#if trigger}
    {@render trigger()}
  {:else}
    <button
      type="button"
      bind:this={triggerElement}
      onclick={toggleMenu}
      aria-haspopup="menu"
      aria-expanded={isOpen}
      {title}
      class="flex h-8 w-8 items-center justify-center rounded-lg text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-900 focus:outline-hidden {isOpen
        ? 'bg-neutral-100 text-neutral-900 ring-2 ring-neutral-200'
        : ''}"
    >
      <!-- Heroicons EllipsisVertical 18x18 -->
      <svg
        class="h-4 w-4"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <circle cx="12" cy="12" r="1" />
        <circle cx="12" cy="5" r="1" />
        <circle cx="12" cy="19" r="1" />
      </svg>
    </button>
  {/if}

  <!-- Popover Menu Melayang (Fixed Viewport Coordinates) -->
  {#if isOpen}
    <div
      bind:this={menuElement}
      style={menuStyle}
      role="menu"
      tabindex="-1"
      class="min-w-[170px] overflow-hidden rounded-xl border border-neutral-200 bg-white py-1 shadow-xl transition-all"
    >
      {#if items.length > 0}
        {#each items as item (item.label)}
          {#if item.divider}
            <div class="my-1 border-t border-neutral-100"></div>
          {/if}
          <button
            type="button"
            role="menuitem"
            disabled={item.disabled}
            onclick={() => handleItemClick(item)}
            class="flex w-full items-center gap-2.5 px-3.5 py-2 text-left text-xs transition-colors {item.disabled
              ? 'cursor-not-allowed opacity-40'
              : item.variant === 'danger'
                ? 'text-rose-600 hover:bg-rose-50 hover:text-rose-700'
                : item.variant === 'primary'
                  ? 'font-medium text-primary-600 hover:bg-primary-50 hover:text-primary-700'
                  : 'text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900'}"
          >
            {#if item.icon}
              <span class="inline-flex h-4 w-4 shrink-0 items-center justify-center">
                <!-- eslint-disable-next-line svelte/no-at-html-tags -->
                {@html item.icon}
              </span>
            {:else if item.iconSvg}
              <svg
                class="h-4 w-4 shrink-0"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d={item.iconSvg} />
              </svg>
            {/if}
            <span class="truncate">{item.label}</span>
          </button>
        {/each}
      {/if}

      {#if children}
        {@render children()}
      {/if}
    </div>
  {/if}
</div>
