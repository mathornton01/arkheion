<script>
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { getBook, deleteBook, uploadBookFile, getDownloadURL } from '$lib/api.js';
  import { notify } from '$lib/stores.js';

  let book = null;
  let loading = true;
  let error = null;

  // Reader state
  let showReader = false;
  let readerContainer;
  let pdfViewer = null;

  // Upload state
  let uploading = false;
  let uploadProgress = 0;
  let fileInput;

  const id = $page.params.id;

  onMount(async () => {
    try {
      book = await getBook(id);
    } catch (err) {
      error = err.message;
    } finally {
      loading = false;
    }
  });

  onDestroy(() => {
    if (pdfViewer) {
      pdfViewer = null;
    }
  });

  async function handleDelete() {
    if (!confirm(`Delete "${book.title}"? This cannot be undone.`)) return;
    try {
      await deleteBook(id);
      notify.success(`"${book.title}" deleted`);
      goto('/library');
    } catch (err) {
      notify.error('Delete failed: ' + err.message);
    }
  }

  async function handleFileUpload(event) {
    const file = event.target.files?.[0];
    if (!file) return;

    uploading = true;
    uploadProgress = 0;
    try {
      const result = await uploadBookFile(id, file, (p) => { uploadProgress = p; });
      notify.success(`File uploaded. Text extraction started.`);
      book = { ...book, file_path: result.file_path, file_type: result.file_type, file_size_bytes: result.file_size_bytes };
    } catch (err) {
      notify.error('Upload failed: ' + err.message);
    } finally {
      uploading = false;
      uploadProgress = 0;
    }
  }

  async function openReader() {
    if (!book.file_path) return;
    showReader = true;

    if (book.file_type === 'pdf') {
      await loadPDF();
    } else if (book.file_type === 'epub') {
      await loadEPUB();
    }
  }

  async function loadPDF() {
    const { GlobalWorkerOptions, getDocument } = await import('pdfjs-dist');
    GlobalWorkerOptions.workerSrc = new URL(
      'pdfjs-dist/build/pdf.worker.min.mjs', import.meta.url
    ).toString();

    const downloadUrl = getDownloadURL(id);
    const loadingTask = getDocument(downloadUrl);
    const pdf = await loadingTask.promise;

    const canvas = document.createElement('canvas');
    readerContainer.appendChild(canvas);
    const ctx = canvas.getContext('2d');

    // Render page 1 as demonstration; a full reader would paginate
    const page = await pdf.getPage(1);
    const viewport = page.getViewport({ scale: 1.5 });
    canvas.height = viewport.height;
    canvas.width = viewport.width;
    canvas.style.maxWidth = '100%';
    await page.render({ canvasContext: ctx, viewport }).promise;
  }

  async function loadEPUB() {
    const ePub = (await import('epubjs')).default;
    const downloadUrl = getDownloadURL(id);
    const epub = ePub(downloadUrl);
    epub.renderTo(readerContainer, { width: '100%', height: '600px' });
    await epub.ready;
    epub.display();
  }

  function formatFileSize(bytes) {
    if (!bytes) return '';
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  function formatDate(d) {
    if (!d) return '';
    return new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' });
  }
</script>

<svelte:head>
  <title>{book ? book.title : 'Book'} — Arkheion</title>
</svelte:head>

{#if loading}
  <div style="display:flex; justify-content:center; padding:4rem">
    <div class="spinner"></div>
  </div>
{:else if error}
  <div class="alert alert-error">{error}</div>
  <a href="/library" class="btn btn-secondary mt-4">← Back to Library</a>
{:else if book}
  <div class="book-detail">
    <!-- Back nav -->
    <a href="/library" class="back-link">← Library</a>

    {#if !showReader}
      <div class="book-layout">
        <!-- Cover + actions -->
        <aside class="book-sidebar">
          {#if book.cover_url}
            <img src={book.cover_url} alt={book.title} class="book-cover" />
          {:else}
            <div class="book-cover-placeholder">📖</div>
          {/if}

          <!-- File actions -->
          <div class="file-actions">
            {#if book.file_path}
              <button class="btn btn-primary w-full" on:click={openReader}>
                📖 Read {book.file_type?.toUpperCase() || 'Book'}
              </button>
              <a href={getDownloadURL(id)} class="btn btn-secondary w-full">
                ⬇ Download
              </a>
            {:else}
              <label class="btn btn-secondary w-full" style="cursor:pointer">
                📤 Upload File
                <input type="file" accept=".pdf,.epub,.txt,.docx"
                  bind:this={fileInput}
                  on:change={handleFileUpload}
                  style="display:none" />
              </label>
            {/if}

            {#if uploading}
              <div class="progress-bar">
                <div class="progress-fill" style="width:{uploadProgress}%"></div>
              </div>
              <p class="text-xs text-muted text-center">{uploadProgress}% uploaded</p>
            {/if}

            {#if book.file_path && !uploading}
              <label class="btn btn-secondary w-full" style="cursor:pointer; font-size:0.8rem">
                ↩ Replace File
                <input type="file" accept=".pdf,.epub,.txt,.docx"
                  on:change={handleFileUpload}
                  style="display:none" />
              </label>
            {/if}
          </div>

          <!-- File metadata -->
          {#if book.file_path}
            <div class="file-meta card">
              <div class="meta-item">
                <span class="meta-label">Format</span>
                <span>{book.file_type?.toUpperCase()}</span>
              </div>
              {#if book.file_size_bytes}
                <div class="meta-item">
                  <span class="meta-label">Size</span>
                  <span>{formatFileSize(book.file_size_bytes)}</span>
                </div>
              {/if}
              <div class="meta-item">
                <span class="meta-label">Searchable</span>
                {#if book.text_extracted}
                  <span class="badge badge-success">Yes</span>
                {:else}
                  <span class="badge badge-warning">Pending</span>
                {/if}
              </div>
            </div>
          {/if}

          <button class="btn btn-danger w-full" style="margin-top:auto" on:click={handleDelete}>
            🗑 Delete Book
          </button>
        </aside>

        <!-- Main content -->
        <article class="book-main">
          <h1 class="book-title">{book.title}</h1>
          {#if book.subtitle}
            <h2 class="book-subtitle">{book.subtitle}</h2>
          {/if}

          {#if book.authors?.length > 0}
            <p class="book-authors">
              by {book.authors.map(a => a.name).join(', ')}
            </p>
          {/if}

          <!-- Metadata chips -->
          <div class="chips">
            {#if book.publisher}
              <span class="chip">{book.publisher}</span>
            {/if}
            {#if book.published_date}
              <span class="chip">{new Date(book.published_date).getFullYear()}</span>
            {/if}
            {#if book.page_count}
              <span class="chip">{book.page_count} pages</span>
            {/if}
            {#if book.language}
              <span class="chip">{book.language.toUpperCase()}</span>
            {/if}
          </div>

          {#if book.isbn}
            <p class="text-muted text-sm">ISBN: {book.isbn}</p>
          {/if}

          {#if book.description}
            <hr class="divider" />
            <h3>Description</h3>
            <p class="book-description">{book.description}</p>
          {/if}

          {#if book.categories?.length > 0}
            <hr class="divider" />
            <h3>Categories</h3>
            <div class="chips">
              {#each book.categories as cat}
                <span class="chip">{cat}</span>
              {/each}
            </div>
          {/if}

          {#if book.tags?.length > 0}
            <hr class="divider" />
            <h3>Tags</h3>
            <div class="chips">
              {#each book.tags as tag}
                <span class="tag">{tag.name}</span>
              {/each}
            </div>
          {/if}

          {#if book.physical_location}
            <hr class="divider" />
            <h3>Physical Location</h3>
            <p>{book.physical_location}</p>
          {/if}

          {#if book.notes}
            <hr class="divider" />
            <h3>Notes</h3>
            <p class="book-notes">{book.notes}</p>
          {/if}

          <hr class="divider" />
          <p class="text-xs text-dim">
            Added {formatDate(book.created_at)} · Updated {formatDate(book.updated_at)}
          </p>
        </article>
      </div>
    {:else}
      <!-- In-browser reader -->
      <div class="reader-header">
        <button class="btn btn-secondary" on:click={() => showReader = false}>← Back</button>
        <h2>{book.title}</h2>
        <a href={getDownloadURL(id)} class="btn btn-secondary">⬇ Download</a>
      </div>
      <div class="reader-container" bind:this={readerContainer}></div>
    {/if}
  </div>
{/if}

<style>
  .book-detail { max-width: 1000px; }
  .back-link {
    display: inline-block;
    color: var(--color-text-muted);
    text-decoration: none;
    font-size: 0.875rem;
    margin-bottom: 1.5rem;
    transition: color var(--transition);
  }
  .back-link:hover { color: var(--color-primary); }

  .book-layout {
    display: grid;
    grid-template-columns: 220px 1fr;
    gap: 2rem;
    align-items: start;
  }

  .book-sidebar {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    position: sticky;
    top: 2rem;
  }

  .book-cover {
    width: 100%;
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
  }
  .book-cover-placeholder {
    width: 100%;
    aspect-ratio: 2/3;
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 3rem;
  }

  .file-actions { display: flex; flex-direction: column; gap: 0.5rem; }

  .progress-bar {
    height: 4px;
    background: var(--color-border);
    border-radius: 2px;
    overflow: hidden;
  }
  .progress-fill {
    height: 100%;
    background: var(--color-primary);
    transition: width 0.2s ease;
  }

  .file-meta.card { padding: 0.875rem; }
  .meta-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 0.8rem;
    padding: 0.25rem 0;
  }
  .meta-label { color: var(--color-text-muted); }

  .book-main {}
  .book-title {
    font-size: 2rem;
    font-weight: 700;
    line-height: 1.2;
    margin-bottom: 0.25rem;
  }
  .book-subtitle {
    font-size: 1.25rem;
    font-weight: 400;
    color: var(--color-text-muted);
    margin-bottom: 0.75rem;
  }
  .book-authors {
    font-size: 1rem;
    color: var(--color-accent);
    margin-bottom: 1rem;
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }
  .chip {
    padding: 0.2rem 0.625rem;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-full);
    font-size: 0.75rem;
    color: var(--color-text-muted);
  }

  .book-description {
    line-height: 1.8;
    color: var(--color-text);
    white-space: pre-line;
  }

  .book-notes {
    background: var(--color-bg-elevated);
    border-left: 3px solid var(--color-primary);
    padding: 0.75rem 1rem;
    border-radius: 0 var(--radius) var(--radius) 0;
    font-style: italic;
    color: var(--color-text-muted);
    white-space: pre-line;
  }

  /* Reader */
  .reader-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--color-border);
  }
  .reader-container {
    background: white;
    border-radius: var(--radius);
    overflow: auto;
    min-height: 600px;
  }

  @media (max-width: 640px) {
    .book-layout {
      grid-template-columns: 1fr;
    }
    .book-sidebar {
      position: static;
    }
    .book-cover, .book-cover-placeholder {
      max-width: 180px;
      margin: 0 auto;
    }
  }
</style>
