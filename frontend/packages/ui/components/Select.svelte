<script lang="ts">
  /**
   * Select — Komponen pilihan dropdown form reusable.
   *
   * Fitur:
   * - Mendukung label, placeholder opsi kosong, error state
   * - Desain serasi dengan Input.svelte (h-12, rounded-xl, Tailwind v4 @theme)
   * - Heroicons chevron down SVG
   * - Bebas emoji & bebas icon sparkle
   */
  import type { SelectOption } from '@erp/types';

  interface Props {
    label?: string;
    value?: string | number;
    options: SelectOption[];
    placeholder?: string;
    error?: string;
    disabled?: boolean;
    required?: boolean;
    showRequiredAsterisk?: boolean;
    id?: string;
    name?: string;
    onchange?: (e: Event) => void;
  }

  let {
    label = '',
    value = $bindable(''),
    options = [],
    placeholder = 'Pilih salah satu...',
    error = '',
    disabled = false,
    required = false,
    showRequiredAsterisk = false,
    id = '',
    name = '',
    onchange,
  }: Props = $props();
</script>

<div class="flex flex-col gap-1.5">
  {#if label}
    <label for={id} class="text-sm font-medium text-neutral-800">
      {label}
      {#if required && showRequiredAsterisk}
        <span class="text-danger-500">*</span>
      {/if}
    </label>
  {/if}

  <div class="relative flex items-center">
    <select
      {id}
      {name}
      {disabled}
      {required}
      bind:value
      {onchange}
      class="h-12 w-full appearance-none rounded-xl border bg-white px-3.5 pr-10 text-sm text-neutral-900 shadow-xs
        transition-all duration-150
        focus:ring-4 focus:outline-hidden
        disabled:cursor-not-allowed disabled:bg-neutral-50 disabled:text-neutral-400
        {error
        ? 'border-danger-400 focus:border-danger-500 focus:ring-danger-500/10'
        : 'focus:border-primary-500 focus:ring-primary-500/10 border-neutral-200 hover:border-neutral-300'}"
    >
      {#if placeholder}
        <option value="" disabled selected={!value}>{placeholder}</option>
      {/if}
      {#each options as opt (opt.value)}
        <option value={opt.value} disabled={opt.disabled}>{opt.label}</option>
      {/each}
    </select>

    <!-- Caret Icon -->
    <div
      class="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3.5 text-neutral-400"
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
        <path d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
      </svg>
    </div>
  </div>

  {#if error}
    <p class="text-danger-600 text-xs font-medium">{error}</p>
  {/if}
</div>
