<script lang="ts">
  /**
   * Badge — Komponen penanda status / label kategori / role.
   *
   * Props:
   * - variant: 'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'purple' | 'indigo' | 'cyan' (default: 'default')
   * - size: 'sm' | 'md' (default: 'md')
   * - children: Snippet konten teks label
   */
  import type { Snippet } from 'svelte';

  interface Props {
    variant?:
      | 'default'
      | 'primary'
      | 'success'
      | 'warning'
      | 'danger'
      | 'info'
      | 'purple'
      | 'indigo'
      | 'cyan';
    size?: 'sm' | 'md';
    children: Snippet;
  }

  let { variant = 'default', size = 'md', children }: Props = $props();

  const variantClasses: Record<string, string> = {
    default: 'bg-neutral-100 text-neutral-700 border-neutral-200',
    primary: 'bg-primary-50 text-primary-700 border-primary-200',
    success: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    warning: 'bg-amber-50 text-amber-700 border-amber-200',
    danger: 'bg-rose-50 text-rose-700 border-rose-200',
    info: 'bg-info-50 text-info-700 border-info-200',
    purple: 'bg-purple-50 text-purple-700 border-purple-200',
    indigo: 'bg-indigo-50 text-indigo-700 border-indigo-200',
    cyan: 'bg-cyan-50 text-cyan-700 border-cyan-200',
  };

  const sizeClasses: Record<string, string> = {
    sm: 'px-2 py-0.5 text-xs font-medium',
    md: 'px-2.5 py-1 text-xs font-semibold',
  };
</script>

<span
  class="inline-flex items-center rounded-md border transition-colors {variantClasses[variant] ??
    variantClasses.default} {sizeClasses[size] ?? sizeClasses.md}"
>
  {@render children()}
</span>
