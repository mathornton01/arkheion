<script>
  import { onMount } from 'svelte';
  import { listBooks } from '$lib/api.js';
  import { totalBooks } from '$lib/stores.js';

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
    <a href="/library" class="btn btn-primary">+ Add Book</a>
  </div>

  <!-- Stats cards -->
  <div class="stats-grid">
    <div class="stat-card">
      <div class="stat-value">{loading ? '…' : stats.total.toLocaleString()}</div>
      <div class="stat-label">Total Books</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">{loading ? '…' : stats.withFiles.toLocaleString()}</div>
      <div class="stat-label">With Digital Files</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">{loading ? '…' : stats.extracted.toLocaleString()}</div>
      <div class="stat-label">Text Extracted</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">
        {loading ? '…' : stats.total > 0 ? Math.round((stats.extracted / stats.total) * 100) + '%' : '0%'}
      </div>
      <div class="stat-label">Search Coverage</div>
    </div>
  </div>

  <!-- Quick actions -->
  <div class="section">
    <h2>Quick Actions</h2>
    <div class="quick-actions">
      <a href="/scan" class="action-card">
        <span class="action-icon">📷</span>
        <span class="action-title">Scan Barcode</span>
        <span class="action-desc">Add a book by scanning its ISBN barcode with your camera</span>
      </a>
      <a href="/search" class="action-card">
        <span class="action-icon">🔍</span>
        <span class="action-title">Search Library</span>
        <span class="action-desc">Full-text search across all book content</span>
      </a>
      <a href="/admin" class="action-card">
        <span class="action-icon">⚡</span>
        <span class="action-title">Export for AI</span>
        <span class="action-desc">Download JSONL training data for Golem / LLM pipelines</span>
      </a>
      <a href="/admin" class="action-card">
        <span class="action-icon">🔗</span>
        <span class="action-title">Manage Webhooks</span>
        <span class="action-desc">Connect Grimoire and other external tools</span>
      </a>
    </div>
  </div>

  <!-- Recent books -->
  {#if !loading && recentBooks.length > 0}
    <div class="section">
      <div class="section-header">
        <h2>Recently Added</h2>
        <a href="/library" class="text-sm" style="color: var(--color-primary)">View all →</a>
      </div>
      <div class="grid-books">
        {#each recentBooks as book}
          <a href="/library/{book.id}" class="book-card">
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
            </div>
          </a>
        {/each}
      </div>
    </div>
  {:else if !loading}
    <div class="empty-state">
      <div class="empty-icon">📚</div>
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
  .dashboard { max-width: 1100px; }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 2rem;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 1rem;
    margin-bottom: 2.5rem;
  }

  .stat-card {
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: 1.5rem;
    text-align: center;
  }
  .stat-value {
    font-size: 2.25rem;
    font-weight: 700;
    color: var(--color-primary);
    line-height: 1;
    margin-bottom: 0.5rem;
  }
  .stat-label {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .section { margin-bottom: 2.5rem; }
  .section h2 { margin-bottom: 1rem; }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
  }

  .quick-actions {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 1rem;
  }

  .action-card {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 1.5rem;
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    text-decoration: none;
    transition: all var(--transition);
  }
  .action-card:hover {
    border-color: var(--color-primary);
    transform: translateY(-2px);
    box-shadow: var(--shadow-md);
  }
  .action-icon { font-size: 1.75rem; }
  .action-title { font-weight: 600; color: var(--color-text); font-size: 0.95rem; }
  .action-desc  { font-size: 0.8rem; color: var(--color-text-muted); line-height: 1.4; }

  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
  }
  .empty-icon { font-size: 4rem; margin-bottom: 1rem; }
  .empty-state h2 { margin-bottom: 0.5rem; }
  .empty-state p { margin-bottom: 2rem; }
  .empty-actions { display: flex; gap: 1rem; justify-content: center; flex-wrap: wrap; }
</style>
