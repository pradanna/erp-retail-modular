<script lang="ts">
  /**
   * Button — Komponen tombol reusable dengan variasi visual dan loading state.
   *
   * Props:
   * - variant: 'primary' | 'secondary' | 'danger' | 'ghost' | 'outline' (default: 'primary')
   * - size: 'sm' | 'md' | 'lg' (default: 'md')
   * - loading: boolean — tampilkan spinner dan disable klik
   * - disabled: boolean
   * - type: 'button' | 'submit' | 'reset' (default: 'button')
   * - form: ID form HTML yang ditargetkan (opsional)
   * - fullWidth: boolean — tombol 100% lebar container
   *
   * Catatan Svelte 5:
   * - Menggunakan $props() rune untuk deklarasi props
   * - Snippet {@render children()} menggantikan <slot />
   */
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost' | 'outline';
    size?: 'sm' | 'md' | 'lg';
    loading?: boolean;
    disabled?: boolean;
    type?: 'button' | 'submit' | 'reset';
    form?: string;
    fullWidth?: boolean;
    onclick?: (e: MouseEvent) => void;
    children: Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    loading = false,
    disabled = false,
    type = 'button',
    form,
    fullWidth = false,
    onclick,
    children,
  }: Props = $props();

  // Mapping variant ke class Tailwind v4
  const variantClasses: Record<string, string> = {
    primary:
      'bg-primary-600 text-white hover:bg-primary-500 focus:ring-primary-500 shadow-sm active:scale-[0.99]',
    secondary:
      'bg-neutral-100 text-neutral-800 hover:bg-neutral-200 focus:ring-neutral-400 active:scale-[0.99]',
    outline:
      'bg-white text-neutral-700 border border-neutral-300 hover:bg-neutral-50 hover:border-neutral-400 focus:ring-primary-500 shadow-xs active:scale-[0.99]',
    danger:
      'bg-danger-600 text-white hover:bg-danger-700 focus:ring-danger-500 shadow-sm active:scale-[0.99]',
    ghost:
      'bg-transparent text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900 focus:ring-primary-500',
  };

  const sizeClasses: Record<string, string> = {
    sm: 'h-9 px-3.5 text-xs',
    md: 'h-10 px-4 text-sm',
    lg: 'h-12 px-6 text-sm font-semibold',
  };
</script>

<button
  {type}
  {form}
  {onclick}
  disabled={disabled || loading}
  class="inline-flex items-center justify-center gap-2 rounded-xl font-medium
    transition-all duration-150 ease-in-out
    focus:outline-none focus:ring-2 focus:ring-offset-1
    disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none
    {variantClasses[variant]}
    {sizeClasses[size]}
    {fullWidth ? 'w-full' : ''}"
>
  {#if loading}
    <svg
      class="h-4 w-4 animate-spin"
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
  {/if}
  {@render children()}
</button>
