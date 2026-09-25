<script lang="ts">
  /**
   * Card — Container card dengan padding, shadow, dan rounded corners.
   *
   * Props:
   * - padding: 'none' | 'sm' | 'md' | 'lg' (default: 'md')
   * - hover: boolean — efek shadow bertambah saat hover
   */
  import type { Snippet } from 'svelte';

  interface Props {
    padding?: 'none' | 'sm' | 'md' | 'lg';
    hover?: boolean;
    children: Snippet;
  }

  let { padding = 'md', hover = false, children }: Props = $props();

  const paddingClasses: Record<string, string> = {
    none: '',
    sm: 'p-4',
    md: 'p-6',
    lg: 'p-8',
  };
</script>

<div
  class="shadow-card rounded-xl border border-neutral-200 bg-white
    {paddingClasses[padding]}
    {hover ? 'hover:shadow-card-hover transition-shadow duration-200' : ''}"
>
  {@render children()}
</div>
