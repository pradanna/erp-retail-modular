<script lang="ts">
  /**
   * StepUpModal — Modal otorisasi ulang (Step-up Authentication) untuk aksi kritikal.
   *
   * Digunakan sebelum mengeksekusi mutasi sensitif:
   * - Approval transfer stok antar cabang
   * - Approval purchase order
   * - Konfirmasi goods receipt
   *
   * Menuntut staf memasukkan kembali kata sandi mereka demi akuntabilitas audit.
   */
  import Modal from './Modal.svelte';
  import Input from './Input.svelte';
  import Button from './Button.svelte';
  import Alert from './Alert.svelte';

  interface Props {
    open?: boolean;
    title?: string;
    description?: string;
    actionLabel?: string;
    loading?: boolean;
    error?: string;
    onconfirm: (password: string) => void;
    oncancel?: () => void;
  }

  let {
    open = $bindable(false),
    title = 'Verifikasi Keamanan Diperlukan',
    description = 'Tindakan ini memerlukan otorisasi ulang. Masukkan kata sandi akun Anda untuk melanjutkan.',
    actionLabel = 'Konfirmasi Tindakan',
    loading = false,
    error = '',
    onconfirm,
    oncancel,
  }: Props = $props();

  let password = $state('');

  function handleSubmit(e?: Event) {
    e?.preventDefault();
    if (!password) return;
    onconfirm(password);
  }

  function handleCancel() {
    open = false;
    password = '';
    oncancel?.();
  }
</script>

<Modal bind:open {title} size="sm" onclose={handleCancel}>
  <form onsubmit={handleSubmit} class="space-y-4">
    <div
      class="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50/80 p-3 text-xs text-amber-800"
    >
      <!-- Heroicons ShieldExclamation -->
      <svg
        class="mt-0.5 h-5 w-5 shrink-0 text-amber-600"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path
          d="M12 9v3.75m0-10.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z"
        />
        <path d="M12 15.75h.007v.008H12v-.008z" />
      </svg>
      <div>
        <p class="font-semibold text-amber-900">Aksi Kritikal Terproteksi</p>
        <p class="mt-0.5 leading-relaxed text-amber-700">{description}</p>
      </div>
    </div>

    {#if error}
      <Alert variant="error" dismissible>{error}</Alert>
    {/if}

    <Input
      id="step-up-password-input"
      label="Kata Sandi Akun Anda"
      type="password"
      placeholder="Masukkan kata sandi saat ini"
      bind:value={password}
      showPasswordToggle={true}
      required
    />
  </form>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={handleCancel} disabled={loading}>Batal</Button>
    <Button
      variant="danger"
      size="sm"
      {loading}
      disabled={!password || loading}
      onclick={handleSubmit}
    >
      {actionLabel}
    </Button>
  {/snippet}
</Modal>
