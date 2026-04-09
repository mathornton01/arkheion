<script>
  import { onMount } from 'svelte';
  import { listBooks } from '$lib/api.js';

  let stats = { total: 0, withFiles: 0, extracted: 0 };
  let recentBooks = [];
  let loading = true;

  onMount(async () => {
    try {
      const [allBooks, withFiles, extracted] = await Promise.all([
        listBooks({ per_page: 1 }),
        listBooks({ per_page: 1, text_extracted: false }),
        listBooks({ per_page: 1, text_extracted: true })
      ]);

      stats.total = allBooks.pagination.total;
      stats.extracted = extracted.pagination.total;
      stats.withFiles = stats.total - withFiles.pagination.total;

      const recent = await listBooks({ per_page: 6 });
      recentBooks = recent.data;
    } catch (err) {
      console.error('Dashboard load error:', err);
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head><title>Dashboard — Arkheion</title></svelte:head>

<div class="dashboard">
  <div class="page-header">
    <h1>Dashboard</h1>
    <a href="/library" class="btn btn-primary">Add Book</a>
  </div>

  <!-- Stats -->
  <div class="stats-grid">
    <div class="stat-card">
      <div class="stat-value">{loading ? '—' : stats.total.toLocaleString()}</div>
      <div class="stat-label">Total Books</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">{loading ? '—' : stats.withFiles.toLocaleString()}</div>
      <div class="stat-label">With Files</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">{loading ? '—' : stats.extracted.toLocaleString()}</div>
      <div class="stat-label">Text Extracted</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">
        {loading ? '—' : stats.total > 0 ? Math.round((stats.extracted / stats.total) * 100) + '%' : '0%'}
      </div>
      <div class="stat-label">Search Coverage</div>
    </div>
  </div>

  <!-- Quick actions -->
  <div class="section">
    <h2 class="section-title">Quick Actions</h2>
    <div class="quick-actions">
      <a href="/scan" class="action-card">
        <div class="action-icon-wrap">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 9V6a2 2 0 0 1 2-2h3"/><path d="M21 9V6a2 2 0 0 0-2-2h-3"/>
            <path d="M3 15v3a2 2 0 0 0 2 2h3"/><path d="M21 15v3a2 2 0 0 1-2 2h-3"/>
            <line x1="7" y1="8" x2="7" y2="16"/><line x1="11" y1="8" x2="11" y2="16"/>
            <line x1="15" y1="8" x2="15" y2="16"/><line x1="17" y1="8" x2="17" y2="16"/>
          </svg>
        </div>
        <span class="action-title">Scan Barcode</span>
        <span class="action-desc">Add a book by scanning its ISBN barcode</span>
      </a>
      <a href="/search" class="action-card">
        <div class="action-icon-wrap">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
        </div>
        <span class="action-title">Search Library</span>
        <span class="action-desc">Full-text search across all book content</span>
      </a>
      <a href="/admin" class="action-card">
        <div class="action-icon-wrap">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
        </div>
        <span class="action-title">Export for AI</span>
        <span class="action-desc">Download JSONL for Golem / LLM pipelines</span>
      </a>
      <a href="/admin" class="action-card">
        <div class="action-icon-wrap">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
          </svg>
        </div>
        <span class="action-title">Webhooks</span>
        <span class="action-desc">Connect Grimoire and external integrations</span>
      </a>
    </div>
  </div>

  <!-- Recent books -->
  {#if !loading && recentBooks.length > 0}
    <div class="section">
      <div class="section-header">
        <h2 class="section-title">Recently Added</h2>
        <a href="/library" class="view-all">View all</a>
      </div>
      <div class="grid-books">
        {#each recentBooks as book}
          <a href="/library/{book.id}" class="book-card">
            {#if book.cover_url}
              <img src={book.cover_url} alt={book.title} class="cover" loading="lazy" />
            {:else}
              <div class="cover-placeholder">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.25" width="32" height="32">
                  <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
                </svg>
              </div>
            {/if}
            <div class="info">
              <div class="title">{book.title}</div>
              {#if book.authors?.length > 0}
                <div class="author">{book.authors[0].name}</div>
              {/if}
            </div>
          </a>
        {/each}
      </div>
    </div>
  {:else if !loading}
    <div class="empty-state">
      <div class="empty-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.25" width="48" height="48">
          <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
        </svg>
      </div>
      <h2>Your library is empty</h2>
      <p class="text-muted">Add your first book to get started.</p>
      <div class="empty-actions">
        <a href="/scan" class="btn btn-primary">Scan Barcode</a>
        <a href="/library" class="btn btn-secondary">Add Manually</a>
      </div>
    </div>
  {/if}
</div>

<style>
  .dashboard { max-width: 1050px; }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1.75rem;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    gap: 0.875rem;
    margin-bottom: 2.25rem;
  }

  .stat-card {
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: 1.25rem 1.5rem;
  }
  .stat-value {
    font-size: 2rem;
    font-weight: 700;
    color: var(--color-primary);
    line-height: 1;
    margin-bottom: 0.4rem;
    letter-spacing: -0.02em;
  }
  .stat-label {
    font-size: 0.7rem;
    color: var(--color-text-dim);
    text-transform: uppercase;
    letter-spacing: 0.07em;
    font-weight: 600;
  }

  .section { margin-bottom: 2.25rem; }
  .section-title { margin-bottom: 0.875rem; font-size: 1rem; color: var(--color-text-muted); font-weight: 500; letter-spacing: 0.01em; }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.875rem;
  }
  .view-all {
    font-size: 0.775rem;
    color: var(--color-text-dim);
    text-decoration: none;
    transition: color var(--transition);
  }
  .view-all:hover { color: var(--color-accent); }

  .quick-actions {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 0.875rem;
  }

  .action-card {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 1.25rem;
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    text-decoration: none;
    transition: all var(--transition);
    border-bottom: 2px solid transparent;
  }
  .action-card:hover {
    border-color: var(--color-border-strong);
    border-bottom-color: var(--color-primary);
    box-shadow: var(--shadow-md);
  }
  .action-icon-wrap {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--color-primary);
    margin-bottom: 0.25rem;
  }
  .action-icon-wrap svg { width: 20px; height: 20px; }
  .action-title { font-weight: 600; color: var(--color-text); font-size: 0.875rem; }
  .action-desc  { font-size: 0.75rem; color: var(--color-text-muted); line-height: 1.45; }

  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.75rem;
  }
  .empty-icon { color: var(--color-text-dim); margin-bottom: 0.5rem; }
  .empty-state p { margin-bottom: 1rem; }
  .empty-actions { display: flex; gap: 0.75rem; justify-content: center; flex-wrap: wrap; }
</style>
