<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { listBooks, createBook, lookupISBN } from '$lib/api.js';
  import { books, bookPagination, bookFilters, libraryViewMode, notify } from '$lib/stores.js';

  let loading = true;
  let showAddModal = false;
  let addLoading = false;
  let isbnLoading = false;

  // Add Book form state
  let form = {
    isbn: '', title: '', subtitle: '', authors: '', publisher: '',
    published_date: '', description: '', page_count: '',
    categories: '', language: 'en', cover_url: '', tags: '',
    physical_location: '', notes: ''
  };
  let formError = '';

  async function loadBooks() {
    loading = true;
    try {
      const filters = Object.fromEntries(
        Object.entries($bookFilters).filter(([, v]) => v !== '' && v !== null)
      );
      const result = await listBooks({
        page: $bookPagination.page,
        per_page: $bookPagination.per_page,
        ...filters
      });
      books.set(result.data);
      bookPagination.set(result.pagination);
    } catch (err) {
      notify.error('Failed to load books: ' + err.message);
    } finally {
      loading = false;
    }
  }

  onMount(loadBooks);

  async function handleFilterChange() {
    bookPagination.update(p => ({ ...p, page: 1 }));
    await loadBooks();
  }

  async function goToPage(page) {
    bookPagination.update(p => ({ ...p, page }));
    await loadBooks();
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  async function lookupIsbn() {
    if (!form.isbn.trim()) return;
    isbnLoading = true;
    formError = '';
    try {
      const result = await lookupISBN(form.isbn);
      form.title = result.title || '';
      form.subtitle = result.subtitle || '';
      form.authors = (result.authors || []).join(', ');
      form.publisher = result.publisher || '';
      form.published_date = result.published_date || '';
      form.description = result.description || '';
      form.page_count = result.page_count ? String(result.page_count) : '';
      form.categories = (result.categories || []).join(', ');
      form.language = result.language || 'en';
      form.cover_url = result.cover_url || '';
      notify.success(`Metadata found via ${result.source}`);
    } catch (err) {
      formError = err.code === 'ISBN_NOT_FOUND'
        ? 'ISBN not found in OpenLibrary or Google Books'
        : err.message;
    } finally {
      isbnLoading = false;
    }
  }

  async function handleAddBook() {
    if (!form.title.trim()) { formError = 'Title is required'; return; }
    addLoading = true;
    formError = '';
    try {
      const data = {
        isbn: form.isbn,
        title: form.title,
        subtitle: form.subtitle,
        authors: form.authors.split(',').map(a => a.trim()).filter(Boolean),
        publisher: form.publisher,
        published_date: form.published_date,
        description: form.description,
        page_count: form.page_count ? parseInt(form.page_count) : 0,
        categories: form.categories.split(',').map(c => c.trim()).filter(Boolean),
        language: form.language || 'en',
        cover_url: form.cover_url,
        tags: form.tags.split(',').map(t => t.trim()).filter(Boolean),
        physical_location: form.physical_location,
        notes: form.notes
      };
      const book = await createBook(data);
      notify.success(`"${book.title}" added to library`);
      showAddModal = false;
      resetForm();
      await loadBooks();
    } catch (err) {
      formError = err.message;
    } finally {
      addLoading = false;
    }
  }

  function resetForm() {
    form = {
      isbn: '', title: '', subtitle: '', authors: '', publisher: '',
      published_date: '', description: '', page_count: '',
      categories: '', language: 'en', cover_url: '', tags: '',
      physical_location: '', notes: ''
    };
    formError = '';
  }

  function openBook(id) {
    goto(`/library/${id}`);
  }
</script>

<svelte:head><title>Library — Arkheion</title></svelte:head>

<div class="library-page">
  <!-- Header -->
  <div class="page-header">
    <div>
      <h1>Library</h1>
      <p class="text-muted text-sm">{$bookPagination.total.toLocaleString()} books</p>
    </div>
    <div class="header-actions">
      <div class="view-toggle">
        <button class:active={$libraryViewMode === 'grid'} on:click={() => libraryViewMode.set('grid')}>⊞</button>
        <button class:active={$libraryViewMode === 'list'} on:click={() => libraryViewMode.set('list')}>☰</button>
      </div>
      <button class="btn btn-primary" on:click={() => { showAddModal = true; resetForm(); }}>+ Add Book</button>
    </div>
  </div>

  <!-- Filters -->
  <div class="filters card mb-6">
    <div class="filters-row">
      <div class="field" style="flex:2">
        <label class="label" for="q">Search</label>
        <input id="q" class="input" type="text" placeholder="Title, author, description…"
          bind:value={$bookFilters.q} on:input={handleFilterChange} />
      </div>
      <div class="field" style="flex:1">
        <label class="label" for="tag">Tag</label>
        <input id="tag" class="input" type="text" placeholder="e.g. science"
          bind:value={$bookFilters.tag} on:input={handleFilterChange} />
      </div>
      <div class="field" style="flex:1">
        <label class="label" for="category">Category</label>
        <input id="category" class="input" type="text" placeholder="e.g. Fiction"
          bind:value={$bookFilters.category} on:input={handleFilterChange} />
      </div>
      <div class="field" style="flex:1">
        <label class="label" for="lang">Language</label>
        <input id="lang" class="input" type="text" placeholder="e.g. en"
          bind:value={$bookFilters.language} on:input={handleFilterChange} />
      </div>
    </div>
  </div>

  <!-- Book grid / list -->
  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <p class="text-muted">Loading books…</p>
    </div>
  {:else if $books.length === 0}
    <div class="empty-state">
      <div class="empty-icon">📭</div>
      <h2>No books found</h2>
      <p class="text-muted">Try adjusting your filters or add some books.</p>
    </div>
  {:else if $libraryViewMode === 'grid'}
    <div class="grid-books">
      {#each $books as book}
        <button class="book-card" on:click={() => openBook(book.id)}>
          {#if book.cover_url}
            <img src={book.cover_url} alt={book.title} class="cover" loading="lazy" />
          {:else}
            <div class="cover-placeholder">📖</div>
          {/if}
          <div class="info">
            <div class="title">{book.title}</div>
            {#if book.authors?.length > 0}
              <div class="author">{book.authors[0].name}</div>
            {/if}
            {#if book.file_type}
              <span class="badge badge-muted" style="margin-top:0.25rem;font-size:0.65rem">
                {book.file_type.toUpperCase()}
              </span>
            {/if}
          </div>
        </button>
      {/each}
    </div>
  {:else}
    <!-- List view -->
    <div class="book-list">
      {#each $books as book}
        <button class="book-list-item" on:click={() => openBook(book.id)}>
          {#if book.cover_url}
            <img src={book.cover_url} alt="" class="list-cover" />
          {:else}
            <div class="list-cover-placeholder">📖</div>
          {/if}
          <div class="list-info">
            <div class="list-title">{book.title}</div>
            {#if book.authors?.length > 0}
              <div class="list-author text-muted text-sm">
                {book.authors.map(a => a.name).join(', ')}
              </div>
            {/if}
            <div class="list-meta text-xs text-dim">
              {#if book.publisher}{book.publisher} · {/if}
              {#if book.language}{book.language.toUpperCase()} · {/if}
              {#if book.text_extracted}
                <span style="color: var(--color-success)">✓ Searchable</span>
              {:else if book.file_type}
                <span style="color: var(--color-warning)">⏳ Extracting…</span>
              {:else}
                <span class="text-dim">No file</span>
              {/if}
            </div>
          </div>
          <div class="list-tags">
            {#each (book.tags || []).slice(0, 3) as tag}
              <span class="tag">{tag.name}</span>
            {/each}
          </div>
        </button>
      {/each}
    </div>
  {/if}

  <!-- Pagination -->
  {#if $bookPagination.total_pages > 1}
    <div class="pagination">
      <button class="btn btn-secondary" disabled={$bookPagination.page <= 1}
        on:click={() => goToPage($bookPagination.page - 1)}>← Previous</button>
      <span class="text-muted text-sm">
        Page {$bookPagination.page} of {$bookPagination.total_pages}
      </span>
      <button class="btn btn-secondary" disabled={$bookPagination.page >= $bookPagination.total_pages}
        on:click={() => goToPage($bookPagination.page + 1)}>Next →</button>
    </div>
  {/if}
</div>

<!-- Add Book Modal -->
{#if showAddModal}
  <div class="modal-overlay" on:click|self={() => showAddModal = false}>
    <div class="modal">
      <div class="modal-header">
        <h2>Add Book</h2>
        <button class="modal-close" on:click={() => showAddModal = false}>✕</button>
      </div>

      <div class="modal-body">
        {#if formError}
          <div class="alert alert-error">{formError}</div>
        {/if}

        <!-- ISBN lookup -->
        <div class="isbn-row">
          <div class="field" style="flex:1; margin-bottom:0">
            <label class="label" for="isbn">ISBN (optional — auto-fills metadata)</label>
            <input id="isbn" class="input" type="text" placeholder="9780345539434"
              bind:value={form.isbn} />
          </div>
          <button class="btn btn-secondary" on:click={lookupIsbn} disabled={isbnLoading}>
            {isbnLoading ? 'Looking up…' : '🔍 Lookup'}
          </button>
          <a href="/scan" class="btn btn-secondary" title="Scan barcode with camera">📷</a>
        </div>

        <hr class="divider" />

        <div class="form-grid">
          <div class="field" style="grid-column:1/-1">
            <label class="label" for="f-title">Title *</label>
            <input id="f-title" class="input" type="text" bind:value={form.title} required />
          </div>
          <div class="field">
            <label class="label" for="f-subtitle">Subtitle</label>
            <input id="f-subtitle" class="input" type="text" bind:value={form.subtitle} />
          </div>
          <div class="field">
            <label class="label" for="f-authors">Authors (comma-separated)</label>
            <input id="f-authors" class="input" type="text" placeholder="Carl Sagan, Ann Druyan" bind:value={form.authors} />
          </div>
          <div class="field">
            <label class="label" for="f-publisher">Publisher</label>
            <input id="f-publisher" class="input" type="text" bind:value={form.publisher} />
          </div>
          <div class="field">
            <label class="label" for="f-date">Published Date</label>
            <input id="f-date" class="input" type="date" bind:value={form.published_date} />
          </div>
          <div class="field">
            <label class="label" for="f-pages">Page Count</label>
            <input id="f-pages" class="input" type="number" min="0" bind:value={form.page_count} />
          </div>
          <div class="field">
            <label class="label" for="f-lang">Language</label>
            <input id="f-lang" class="input" type="text" placeholder="en" bind:value={form.language} />
          </div>
          <div class="field" style="grid-column:1/-1">
            <label class="label" for="f-desc">Description</label>
            <textarea id="f-desc" class="input" rows="3" bind:value={form.description}></textarea>
          </div>
          <div class="field">
            <label class="label" for="f-cats">Categories (comma-separated)</label>
            <input id="f-cats" class="input" type="text" placeholder="Science, Astronomy" bind:value={form.categories} />
          </div>
          <div class="field">
            <label class="label" for="f-tags">Tags (comma-separated)</label>
            <input id="f-tags" class="input" type="text" placeholder="classic, must-read" bind:value={form.tags} />
          </div>
          <div class="field">
            <label class="label" for="f-loc">Physical Location</label>
            <input id="f-loc" class="input" type="text" placeholder="Shelf A3, Box 2" bind:value={form.physical_location} />
          </div>
          <div class="field">
            <label class="label" for="f-cover">Cover URL</label>
            <input id="f-cover" class="input" type="url" placeholder="https://…" bind:value={form.cover_url} />
          </div>
          <div class="field" style="grid-column:1/-1">
            <label class="label" for="f-notes">Notes</label>
            <textarea id="f-notes" class="input" rows="2" bind:value={form.notes}></textarea>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn btn-secondary" on:click={() => showAddModal = false}>Cancel</button>
        <button class="btn btn-primary" on:click={handleAddBook} disabled={addLoading}>
          {addLoading ? 'Adding…' : 'Add Book'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .library-page { max-width: 1200px; }

  .page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 1.5rem;
  }
  .header-actions { display: flex; gap: 0.75rem; align-items: center; }

  .view-toggle {
    display: flex;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .view-toggle button {
    background: none;
    border: none;
    color: var(--color-text-muted);
    padding: 0.4rem 0.7rem;
    cursor: pointer;
    font-size: 1rem;
    transition: all var(--transition);
  }
  .view-toggle button.active {
    background: var(--color-primary);
    color: white;
  }

  .filters.card { padding: 1rem 1.25rem; }
  .filters-row {
    display: flex;
    gap: 1rem;
    align-items: flex-end;
    flex-wrap: wrap;
  }
  .filters-row .field { margin-bottom: 0; }

  .loading-state, .empty-state {
    text-align: center;
    padding: 4rem 2rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
  }
  .empty-icon { font-size: 3rem; }

  /* List view */
  .book-list { display: flex; flex-direction: column; gap: 0.5rem; }
  .book-list-item {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.875rem 1rem;
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    cursor: pointer;
    text-align: left;
    width: 100%;
    transition: all var(--transition);
  }
  .book-list-item:hover {
    border-color: var(--color-primary-dim);
    box-shadow: var(--shadow-sm);
  }
  .list-cover {
    width: 40px;
    height: 56px;
    object-fit: cover;
    border-radius: 4px;
    flex-shrink: 0;
  }
  .list-cover-placeholder {
    width: 40px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-elevated);
    border-radius: 4px;
    font-size: 1.25rem;
    flex-shrink: 0;
  }
  .list-info { flex: 1; min-width: 0; }
  .list-title { font-weight: 600; font-size: 0.9rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .list-author { margin-top: 0.125rem; }
  .list-meta { margin-top: 0.25rem; }
  .list-tags { display: flex; gap: 0.25rem; flex-wrap: wrap; flex-shrink: 0; }

  /* Pagination */
  .pagination {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1.5rem;
    margin-top: 2rem;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.75);
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
  }
  .modal {
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 680px;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    box-shadow: var(--shadow-lg);
  }
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid var(--color-border);
  }
  .modal-close {
    background: none;
    border: none;
    color: var(--color-text-muted);
    font-size: 1.1rem;
    cursor: pointer;
    padding: 0.25rem;
  }
  .modal-body {
    flex: 1;
    overflow-y: auto;
    padding: 1.5rem;
  }
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    padding: 1rem 1.5rem;
    border-top: 1px solid var(--color-border);
  }

  .isbn-row {
    display: flex;
    gap: 0.75rem;
    align-items: flex-end;
    margin-bottom: 1rem;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0 1rem;
  }

  textarea.input { resize: vertical; min-height: 80px; }
</style>
