<script lang="ts">
  /**
   * Input — Komponen input form reusable dengan label, leading icon, password toggle, dan error state.
   *
   * Sesuai design system:
   * - Mendukung leadingIcon via Svelte 5 snippet
   * - Password visibility toggle otomatis
   * - Zero emoji, zero sparkle
   * - Tailwind v4 @theme tokens
   * - Robust h-12 padding & vertical centering
   */
  import type { Snippet } from 'svelte';
  import type { HTMLInputAttributes } from 'svelte/elements';

  interface Props {
    label?: string;
    type?: 'text' | 'password' | 'email' | 'number' | 'search';
    placeholder?: string;
    value?: string | number;
    error?: string;
    disabled?: boolean;
    required?: boolean;
    showRequiredAsterisk?: boolean;
    id?: string;
    name?: string;
    autocomplete?: HTMLInputAttributes['autocomplete'];
    leadingIcon?: Snippet;
    showPasswordToggle?: boolean;
  }

  let {
    label = '',
    type = 'text',
    placeholder = '',
    value = $bindable(''),
    error = '',
    disabled = false,
    required = false,
    showRequiredAsterisk = false,
    id = '',
    name = '',
    autocomplete = undefined,
    leadingIcon,
    showPasswordToggle = false,
  }: Props = $props();

  let passwordVisible = $state(false);
  let effectiveType = $derived(
    type === 'password' ? (passwordVisible ? 'text' : 'password') : type,
  );
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
    {#if leadingIcon}
      <div
        class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5 text-neutral-400"
      >
        {@render leadingIcon()}
      </div>
    {/if}

    <input
      {id}
      {name}
      type={effectiveType}
      {placeholder}
      {disabled}
      {required}
      {autocomplete}
      bind:value
      class="h-12 w-full rounded-xl border bg-white px-3.5 text-sm text-neutral-900 shadow-xs
        transition-all duration-150
        placeholder:text-neutral-400
        focus:ring-4 focus:outline-none
        disabled:cursor-not-allowed disabled:bg-neutral-50 disabled:text-neutral-400
        {leadingIcon ? 'pl-11' : 'pl-3.5'}
        {type === 'password' || showPasswordToggle ? 'pr-11' : 'pr-3.5'}
        {error
        ? 'border-danger-400 focus:border-danger-500 focus:ring-danger-500/10'
        : 'focus:border-primary-500 focus:ring-primary-500/10 border-neutral-200 hover:border-neutral-300'}"
    />

    {#if type === 'password' || showPasswordToggle}
      <button
        type="button"
        tabindex="-1"
        onclick={() => (passwordVisible = !passwordVisible)}
        class="absolute inset-y-0 right-0 flex items-center pr-3.5 text-neutral-400 transition-colors hover:text-neutral-600 focus:outline-none"
        aria-label={passwordVisible ? 'Sembunyikan password' : 'Lihat password'}
      >
        {#if passwordVisible}
          <!-- Heroicons EyeSlash 20x20 -->
          <svg
            class="h-5 w-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M3.98 8.223A10.477 10.477 0 0 0 1.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.451 10.451 0 0 1 12 4.5c4.756 0 8.773 3.162 10.065 7.498a10.522 10.522 0 0 1-4.293 5.774M6.228 6.228 3 3m3.228 3.228 3.65 3.65m7.894 7.894L21 21m-3.228-3.228-3.65-3.65m0 0a3 3 0 1 0-4.243-4.243m4.242 4.242L9.88 9.88"
            />
          </svg>
        {:else}
          <!-- Heroicons Eye 20x20 -->
          <svg
            class="h-5 w-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z"
            />
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
            />
          </svg>
        {/if}
      </button>
    {/if}
  </div>

  {#if error}
    <p class="text-danger-600 text-xs font-medium">{error}</p>
  {/if}
</div>
