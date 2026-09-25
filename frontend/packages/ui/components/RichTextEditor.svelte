<script lang="ts">
  /**
   * RichTextEditor — Komponen WYSIWYG Editor Reusable (Gen-E Enterprise).
   *
   * Fitur:
   * - Zero external dependency (berbasis browser selection & contenteditable API)
   * - Toolbar kaya fitur: Bold, Italic, Underline, Strikethrough, H2, H3, Bullet & Numbered List, Quote, Link, Divider
   * - Mode ganda: Visual WYSIWYG & Kode Sumber HTML (<>)
   * - Standar icon Heroicons SVG outline (tanpa emoticon & tanpa sparkle)
   * - Styling tipografi modern yang serasi dengan identitas Tailwind CSS v4 @theme
   */
  interface Props {
    id?: string;
    label?: string;
    value?: string;
    placeholder?: string;
    minHeight?: string;
    disabled?: boolean;
    error?: string;
  }

  let {
    id = `rte-${Math.random().toString(36).slice(2, 9)}`,
    label,
    value = $bindable(''),
    placeholder = 'Tuliskan deskripsi lengkap produk di sini...',
    minHeight = '180px',
    disabled = false,
    error,
  }: Props = $props();

  let editorElement: HTMLDivElement | null = $state(null);
  let isSourceMode = $state(false);

  // Sinkronisasi nilai luar ke innerHTML saat elemen dimount atau berubah secara eksternal
  $effect(() => {
    if (editorElement && !isSourceMode) {
      const currentContent = editorElement.innerHTML;
      const incomingContent = value || '';
      if (currentContent !== incomingContent) {
        editorElement.innerHTML = incomingContent;
      }
    }
  });

  function handleInput() {
    if (editorElement && !isSourceMode) {
      value = editorElement.innerHTML;
    }
  }

  function handleSourceInput(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    value = target.value;
  }

  function toggleSourceMode() {
    if (isSourceMode) {
      // Kembali ke mode visual
      isSourceMode = false;
      // Di next tick setelah DOM terupdate
      setTimeout(() => {
        if (editorElement) {
          editorElement.innerHTML = value || '';
          editorElement.focus();
        }
      }, 0);
    } else {
      // Pindah ke mode kode sumber HTML
      if (editorElement) {
        value = editorElement.innerHTML;
      }
      isSourceMode = true;
    }
  }

  function exec(command: string, arg?: string) {
    if (disabled || isSourceMode) return;
    editorElement?.focus();
    document.execCommand(command, false, arg);
    handleInput();
  }

  function insertLink() {
    if (disabled || isSourceMode) return;
    const selection = window.getSelection();
    const selectedText = selection?.toString() || '';
    const url = prompt('Masukkan tautan URL (contoh: https://domain.com):', 'https://');
    if (url && url.trim() !== '' && url !== 'https://') {
      if (!selectedText && selection && selection.rangeCount > 0) {
        // Jika tidak ada teks terpilih, sisipkan teks link
        exec('insertHTML', `<a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a>`);
      } else {
        exec('createLink', url);
      }
    }
  }

  function insertDivider() {
    exec('insertHorizontalRule');
  }
</script>

<div class="w-full">
  {#if label}
    <label for={id} class="mb-1.5 block text-xs font-medium text-neutral-700">
      {label}
    </label>
  {/if}

  <div
    class="overflow-hidden rounded-xl border transition-colors {error
      ? 'border-rose-400 focus-within:border-rose-500 focus-within:ring-2 focus-within:ring-rose-500/20'
      : 'focus-within:border-primary-500 focus-within:ring-primary-500/20 border-neutral-300 focus-within:ring-2'} {disabled
      ? 'bg-neutral-100 opacity-60'
      : 'bg-white'}"
  >
    <!-- Toolbar Editor -->
    <div
      class="flex flex-wrap items-center gap-0.5 border-b border-neutral-200 bg-neutral-50/90 px-2.5 py-1.5 text-neutral-600 select-none"
    >
      <!-- Grup 1: Format Teks (Bold, Italic, Underline, Strike) -->
      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Tebal (Ctrl+B)"
        onclick={() => exec('bold')}
        {disabled}
      >
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
          <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
        </svg>
      </button>

      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Miring (Ctrl+I)"
        onclick={() => exec('italic')}
        {disabled}
      >
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <line x1="19" y1="4" x2="10" y2="4" />
          <line x1="14" y1="20" x2="5" y2="20" />
          <line x1="15" y1="4" x2="9" y2="20" />
        </svg>
      </button>

      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Garis Bawah (Ctrl+U)"
        onclick={() => exec('underline')}
        {disabled}
      >
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M6 3v7a6 6 0 0 0 6 6 6 6 0 0 0 6-6V3" />
          <line x1="4" y1="21" x2="20" y2="21" />
        </svg>
      </button>

      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Coret Teks"
        onclick={() => exec('strikeThrough')}
        {disabled}
      >
        <svg
          class="h-4 w-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M16 4H9a3 3 0 0 0-2.83 4" />
          <path d="M14 12a4 4 0 0 1 0 8H6" />
          <line x1="4" y1="12" x2="20" y2="12" />
        </svg>
      </button>

      <!-- Divider -->
      <div class="mx-1 h-4 w-px bg-neutral-200"></div>

      <!-- Grup 2: Struktur Heading & Paragraf -->
      <button
        type="button"
        class="rounded-md px-1.5 py-1 text-xs font-bold transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Judul Bagian (Heading 2)"
        onclick={() => exec('formatBlock', '<h2>')}
        {disabled}
      >
        H2
      </button>

      <button
        type="button"
        class="rounded-md px-1.5 py-1 text-xs font-semibold transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Sub-Judul (Heading 3)"
        onclick={() => exec('formatBlock', '<h3>')}
        {disabled}
      >
        H3
      </button>

      <button
        type="button"
        class="rounded-md px-1.5 py-1 text-xs font-medium transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Paragraf Normal"
        onclick={() => exec('formatBlock', '<p>')}
        {disabled}
      >
        P
      </button>

      <!-- Divider -->
      <div class="mx-1 h-4 w-px bg-neutral-200"></div>

      <!-- Grup 3: Daftar (Lists) -->
      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Daftar Poin (Bulleted List)"
        onclick={() => exec('insertUnorderedList')}
        {disabled}
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
          <line x1="8" y1="6" x2="21" y2="6" />
          <line x1="8" y1="12" x2="21" y2="12" />
          <line x1="8" y1="18" x2="21" y2="18" />
          <line x1="3" y1="6" x2="3.01" y2="6" stroke-width="3" />
          <line x1="3" y1="12" x2="3.01" y2="12" stroke-width="3" />
          <line x1="3" y1="18" x2="3.01" y2="18" stroke-width="3" />
        </svg>
      </button>

      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Daftar Angka (Numbered List)"
        onclick={() => exec('insertOrderedList')}
        {disabled}
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
          <line x1="10" y1="6" x2="21" y2="6" />
          <line x1="10" y1="12" x2="21" y2="12" />
          <line x1="10" y1="18" x2="21" y2="18" />
          <path d="M4 6h1v4" />
          <path d="M4 10h2" />
          <path d="M6 18H4c0-1 2-2 2-3s-1-1.5-2-1" />
        </svg>
      </button>

      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Kutipan (Blockquote)"
        onclick={() => exec('formatBlock', '<blockquote>')}
        {disabled}
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
          <path
            d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V20c0 1 0 1 1 1z"
          />
          <path
            d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V20c0 1 0 1 1 1z"
          />
        </svg>
      </button>

      <!-- Divider -->
      <div class="mx-1 h-4 w-px bg-neutral-200"></div>

      <!-- Grup 4: Tautan & Garis Pembatas -->
      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Sisipkan Tautan URL"
        onclick={insertLink}
        {disabled}
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
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
        </svg>
      </button>

      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Garis Pemisah (Horizontal Rule)"
        onclick={insertDivider}
        {disabled}
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
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
      </button>

      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Hapus Format (Clear Formatting)"
        onclick={() => exec('removeFormat')}
        {disabled}
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
          <path d="M18 6L6 18" />
          <path d="M4 6h16" />
        </svg>
      </button>

      <!-- Divider -->
      <div class="mx-1 h-4 w-px bg-neutral-200"></div>

      <!-- Grup 5: Undo / Redo -->
      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Undo (Ctrl+Z)"
        onclick={() => exec('undo')}
        {disabled}
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
          <path d="M3 7v6h6" />
          <path d="M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13" />
        </svg>
      </button>

      <button
        type="button"
        class="rounded-md p-1.5 transition-colors hover:bg-neutral-200/70 hover:text-neutral-900 focus:outline-hidden disabled:opacity-40"
        title="Redo (Ctrl+Y)"
        onclick={() => exec('redo')}
        {disabled}
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
          <path d="M21 7v6h-6" />
          <path d="M3 17a9 9 0 0 1 9-9 9 9 0 0 1 6 2.3l3 2.7" />
        </svg>
      </button>

      <!-- Tombol Toggle Mode Kode Sumber (HTML Source Code) -->
      <button
        type="button"
        class="ml-auto flex items-center gap-1 rounded-md px-2 py-1 text-xs font-semibold transition-colors {isSourceMode
          ? 'bg-neutral-900 text-white'
          : 'text-neutral-500 hover:bg-neutral-200/70 hover:text-neutral-900'} focus:outline-hidden"
        title={isSourceMode
          ? 'Kembali ke Editor Visual (WYSIWYG)'
          : 'Lihat / Edit Kode Sumber HTML (<>)'}
        onclick={toggleSourceMode}
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
          <polyline points="16 18 22 12 16 6" />
          <polyline points="8 6 2 12 8 18" />
        </svg>
        <span>{isSourceMode ? 'Visual' : 'HTML'}</span>
      </button>
    </div>

    <!-- Area Konten Editor (Visual WYSIWYG vs Source Code) -->
    {#if isSourceMode}
      <textarea
        {id}
        {value}
        oninput={handleSourceInput}
        style="min-height: {minHeight};"
        class="w-full resize-y bg-neutral-900 p-4 font-mono text-xs leading-relaxed text-emerald-400 focus:outline-hidden"
        placeholder="<!-- Ketik atau tempel tag HTML di sini -->"
        {disabled}></textarea>
    {:else}
      <div
        bind:this={editorElement}
        {id}
        contenteditable={!disabled}
        oninput={handleInput}
        style="min-height: {minHeight};"
        class="prose-editor w-full overflow-y-auto px-4 py-3 text-xs leading-relaxed text-neutral-800 focus:outline-hidden"
        role="textbox"
        tabindex="0"
        aria-multiline="true"
        data-placeholder={placeholder}
      ></div>
    {/if}

    <!-- Footer Editor: Info Format & Mode -->
    <div
      class="flex items-center justify-between border-t border-neutral-100 bg-neutral-50/50 px-3 py-1 text-[11px] text-neutral-400"
    >
      <span>{isSourceMode ? 'Mode Kode Sumber HTML Mentah' : 'Mode Editor Visual (WYSIWYG)'}</span>
      <span>Format aman disimpan sebagai HTML</span>
    </div>
  </div>

  {#if error}
    <p class="mt-1 text-xs text-rose-600">{error}</p>
  {/if}
</div>

<style>
  /* Tipografi di dalam area editor agar serasi dengan tampilan storefront */
  .prose-editor :global(h2) {
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--color-neutral-900, #0f172a);
    margin-top: 0.75rem;
    margin-bottom: 0.35rem;
  }

  .prose-editor :global(h3) {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--color-neutral-900, #0f172a);
    margin-top: 0.5rem;
    margin-bottom: 0.25rem;
  }

  .prose-editor :global(p) {
    margin-bottom: 0.5rem;
    line-height: 1.6;
  }

  .prose-editor :global(ul) {
    list-style-type: disc;
    padding-left: 1.25rem;
    margin-bottom: 0.5rem;
  }

  .prose-editor :global(ol) {
    list-style-type: decimal;
    padding-left: 1.25rem;
    margin-bottom: 0.5rem;
  }

  .prose-editor :global(li) {
    margin-bottom: 0.2rem;
  }

  .prose-editor :global(blockquote) {
    border-left: 3px solid var(--color-neutral-300, #cbd5e1);
    padding-left: 0.75rem;
    margin-top: 0.5rem;
    margin-bottom: 0.5rem;
    font-style: italic;
    color: var(--color-neutral-600, #475569);
  }

  .prose-editor :global(a) {
    color: var(--color-indigo-600, #4f46e5);
    text-decoration: underline;
    font-weight: 500;
  }

  .prose-editor :global(hr) {
    border: 0;
    border-top: 1px solid var(--color-neutral-200, #e2e8f0);
    margin-top: 0.75rem;
    margin-bottom: 0.75rem;
  }

  /* Placeholder saat kosong */
  .prose-editor:empty::before {
    content: attr(data-placeholder);
    color: var(--color-neutral-400, #94a3b8);
    pointer-events: none;
  }
</style>
