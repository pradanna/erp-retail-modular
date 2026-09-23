<script lang="ts">
  /**
   * Alert — Komponen notifikasi inline dengan variasi semantik.
   *
   * Props:
   * - variant: 'info' | 'success' | 'warning' | 'error' (default: 'info')
   * - title: Judul alert (opsional)
   * - dismissible: boolean — tampilkan tombol tutup
   */
  import type { Snippet } from 'svelte';

  type AlertVariant = 'info' | 'success' | 'warning' | 'error';

  interface Props {
    variant?: AlertVariant;
    title?: string;
    dismissible?: boolean;
    children: Snippet;
  }

  let {
    variant = 'info',
    title = '',
    dismissible = false,
    children,
  }: Props = $props();

  let visible = $state(true);

  const defaultStyles = {
    bg: 'bg-primary-50',
    border: 'border-primary-200',
    text: 'text-primary-800',
    iconColor: 'text-primary-600',
  };

  const variantStyles: Record<AlertVariant, { bg: string; border: string; text: string; iconColor: string }> = {
    info: defaultStyles,
    success: {
      bg: 'bg-success-50',
      border: 'border-success-200',
      text: 'text-success-800',
      iconColor: 'text-success-600',
    },
    warning: {
      bg: 'bg-warning-50',
      border: 'border-warning-200',
      text: 'text-warning-800',
      iconColor: 'text-warning-600',
    },
    error: {
      bg: 'bg-danger-50',
      border: 'border-danger-200',
      text: 'text-danger-800',
      iconColor: 'text-danger-600',
    },
  };

  const styles = $derived(variantStyles[variant] ?? defaultStyles);
</script>

{#if visible}
  <div
    class="flex items-start gap-3 rounded-lg border p-4
      {styles.bg} {styles.border} {styles.text}"
    role="alert"
  >
    <div class="mt-0.5 shrink-0 {styles.iconColor}">
      {#if variant === 'info'}
        <!-- Heroicons InformationCircle -->
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z" />
        </svg>
      {:else if variant === 'success'}
        <!-- Heroicons CheckCircle -->
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
      {:else if variant === 'warning'}
        <!-- Heroicons ExclamationTriangle -->
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
        </svg>
      {:else}
        <!-- Heroicons XCircle -->
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="m9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
      {/if}
    </div>

    <div class="flex-1">
      {#if title}
        <p class="mb-1 font-semibold">{title}</p>
      {/if}
      <div class="text-sm leading-relaxed">
        {@render children()}
      </div>
    </div>

    {#if dismissible}
      <button
        type="button"
        class="shrink-0 rounded p-1 opacity-60 transition-opacity hover:opacity-100"
        onclick={() => (visible = false)}
        aria-label="Tutup alert"
      >
        <!-- Heroicons XMark -->
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
        </svg>
      </button>
    {/if}
  </div>
{/if}
