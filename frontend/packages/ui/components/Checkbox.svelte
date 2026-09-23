<script lang="ts">
  /**
   * Checkbox — Komponen pilihan biner standar design system.
   *
   * Props:
   * - label: Teks label di samping checkbox
   * - checked: Binding dua arah ($bindable)
   * - disabled: boolean
   * - id: ID unik
   * - name: string
   */
  interface Props {
    label?: string;
    checked?: boolean;
    disabled?: boolean;
    id?: string;
    name?: string;
  }

  let {
    label = '',
    checked = $bindable(false),
    disabled = false,
    id = '',
    name = '',
  }: Props = $props();
</script>

<label for={id} class="inline-flex items-center gap-2.5 cursor-pointer select-none text-sm text-neutral-600 {disabled ? 'cursor-not-allowed opacity-50' : 'hover:text-neutral-900'}">
  <input
    {id}
    {name}
    type="checkbox"
    {disabled}
    bind:checked
    class="h-4.5 w-4.5 rounded border-neutral-300 text-primary-600 accent-primary-600 focus:ring-primary-500 transition-colors cursor-pointer"
  />
  {#if label}
    <span class="text-sm font-normal text-neutral-600 select-none">{label}</span>
  {/if}
</label>
