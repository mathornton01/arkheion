<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { search } from '$lib/api.js';
  import { searchState, searchQuery } from '$lib/stores.js';

  let inputEl;
  let loading = false;
  let error = '';
  let filter = '';

  // Restore query from URL on mount
  onMount(() => {
    const q = $page.url.searchParams.get('q');
    if (q) {
      searchQuery.set(q);
      performSearch(q);
    }
    inputEl?.focus();
  });

  async function performSearch(q) {
    if (!q?.trim()) return;
    loading = true;
    error = '';
    try {
      const result = await search(q, { per_page: 30, filter: filter || undefined });
      searchState.set({ query: q, ...result });
      // Update URL without reload
      const url = new URL(window.location.href);
      url.searchParams.set('q', q);
      goto(`?q=${encodeURIComponent(q)}`, { replaceState: true, noScroll: true });
    } catch (err) {
      error = err.message;
      searchState.set(null);
    } finally {
      loading = false;
    }
  }

  function handleSubmit(e) {
    e.preventDefault();
    performSearch($searchQuery);
  }

  function highlightMatch(text, query) {
    if (!text || !query) return text;
    const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    return text.replace(new RegExp(`(${escaped})`, 'gi'), '<mark>$1</mark>');
  }
</script>

<svelte:head>
  <title>Search — Arkheion</title>
</svelte:head>

<div class="search-page">
  <h1>Search Library</h1>

  <!-- Search form -->
  <form class="search-form" on:submit={handleSubmit}>
    <div class="search-input-wrap">
      <span class="search-icon">🔍</span>
      <input
        bind:this={inputEl}
        bind:value={$searchQuery}
        class="search-input"
        type="text"
        placeholder="Search books, authors, content…"
        autocomplete="off"
        spellcheck="false"
      />
      {#if $searchQuery}
        <button type="button" class="clear-btn" on:click={() => { searchQuery.set(''); searchState.set(null); }}>✕</button>
      {/if}
    </div>
    <button type="submit" class="btn btn-primary" disabled={loading}>
      {loading ? 'Searching…' : 'Search'}
    </button>
  </form>

  <!-- Filter bar -->
  <div class="filter-bar">
    <span class="text-xs text-muted">Filter:</span>
    <input class="filter-input" type="text" placeholder='e.g. language="en" AND text_extracted=true'
      bind:value={filter} on:change={() => $searchQuery && performSearch($searchQuery)} />
    <button class="btn btn-secondary" style="font-size:0.75rem; padding:0.375rem 0.75rem"
      on:click={() => { filter = ''; $searchQuery && performSearch($searchQuery); }}>
      Clear
    </button>
  </div>

  {#if error}
    <div class="alert alert-error">{error}</div>
  {/if}

  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <p class="text-muted">Searching…</p>
    </div>
  {:else if $searchState}
    <!-- Results header -->
    <div class="results-header">
      <p class="text-muted text-sm">
        {$searchState.total_hits?.toLocaleString() ?? $searchState.estimated_total_hits?.toLocaleString() ?? 0} results
        for "<strong>{$searchState.query}</strong>"
        · {$searchState.processing_time_ms}ms
      </p>
    </div>

    {#if $searchState.hits?.length === 0}
      <div class="empty-results">
        <p class="text-muted">No books found for "<strong>{$searchState.query}</strong>".</p>
        <p class="text-dim text-sm mt-4">
          Tip: Full-text search only returns books with extracted text.
          Upload PDF/EPUB files to enable full-text search.
        </p>
      </div>
    {:else}
      <div class="results-list">
        {#each $searchState.hits as hit}
          <a href="/library/{hit.id}" class="result-item card">
            <div class="result-layout">
              {#if hit.cover_url}
                <img src={hit.cover_url} alt="" class="result-thumb" />
              {:else}
                <div class="result-thumb-placeholder">📖</div>
              {/if}
              <div class="result-body">
                <h3 class="result-title">
                  {@html highlightMatch(hit.title, $searchState.query)}
                </h3>
                {#if hit.authors?.length > 0}
                  <p class="result-author text-muted text-sm">
                    {@html highlightMatch(hit.authors.join(', '), $searchState.query)}
                  </p>
                {/if}
                {#if hit.description}
                  <p class="result-excerpt text-sm text-muted">
                    {@html highlightMatch(hit.description.slice(0, 200), $searchState.query)}…
                  </p>
                {:else if hit.extracted_text_snippet}
                  <p class="result-excerpt text-sm text-muted">
                    …{@html highlightMatch(hit.extracted_text_snippet.slice(0, 200), $searchState.query)}…
                  </p>
                {/if}
                <div class="result-meta">
                  {#if hit.categories?.length > 0}
                    <span class="chip">{hit.categories[0]}</span>
                  {/if}
                  {#if hit.language}
                    <span class="chip">{hit.language.toUpperCase()}</span>
                  {/if}
                  {#if hit.text_extracted}
                    <span class="badge badge-success">Full Text</span>
                  {/if}
                  {#each (hit.tags || []).slice(0, 3) as tag}
                    <span class="tag">{tag}</span>
                  {/each}
                </div>
              </div>
            </div>
          </a>
        {/each}
      </div>
    {/if}
  {:else if !loading}
    <div class="search-tips">
      <h3>Search tips</h3>
      <ul class="tips-list">
        <li>Search by title, author, description, category, or tag</li>
        <li>For books with uploaded files, full text is searched too</li>
        <li>Use Meilisearch filters: <code>language="en"</code>, <code>text_extracted=true</code></li>
        <li>Meilisearch handles typos automatically</li>
      </ul>
    </div>
  {/if}
</div>

<style>
  .search-page { max-width: 800px; }
  h1 { margin-bottom: 1.5rem; }

  .search-form {
    display: flex;
    gap: 0.75rem;
    margin-bottom: 0.75rem;
  }

  .search-input-wrap {
    position: relative;
    flex: 1;
  }
  .search-icon {
    position: absolute;
    left: 0.875rem;
    top: 50%;
    transform: translateY(-50%);
    font-size: 1rem;
  }
  .search-input {
    width: 100%;
    padding: 0.75rem 2.5rem 0.75rem 2.625rem;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    color: var(--color-text);
    font-size: 1rem;
    outline: none;
    transition: border-color var(--transition);
  }
  .search-input:focus { border-color: var(--color-primary); }
  .clear-btn {
    position: absolute;
    right: 0.75rem;
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    color: var(--color-text-muted);
    cursor: pointer;
    font-size: 0.875rem;
  }

  .filter-bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
  }
  .filter-input {
    flex: 1;
    padding: 0.375rem 0.625rem;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    color: var(--color-text);
    font-size: 0.8rem;
    font-family: var(--font-mono);
    outline: none;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 3rem;
  }

  .results-header { margin-bottom: 1rem; }

  .results-list { display: flex; flex-direction: column; gap: 0.75rem; }

  .result-item.card {
    padding: 1rem 1.25rem;
    text-decoration: none;
    display: block;
  }
  .result-layout {
    display: flex;
    gap: 1rem;
  }
  .result-thumb {
    width: 56px;
    height: 80px;
    object-fit: cover;
    border-radius: 4px;
    flex-shrink: 0;
  }
  .result-thumb-placeholder {
    width: 56px;
    height: 80px;
    background: var(--color-bg-elevated);
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.5rem;
    flex-shrink: 0;
  }
  .result-body { flex: 1; min-width: 0; }
  .result-title {
    font-size: 1rem;
    font-weight: 600;
    margin-bottom: 0.25rem;
    color: var(--color-text);
  }
  .result-author { margin-bottom: 0.5rem; }
  .result-excerpt { line-height: 1.5; margin-bottom: 0.5rem; }
  .result-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.375rem;
    align-items: center;
  }

  :global(.result-title mark),
  :global(.result-author mark),
  :global(.result-excerpt mark) {
    background: rgba(247, 162, 58, 0.3);
    color: var(--color-accent);
    border-radius: 2px;
    padding: 0 1px;
  }

  .empty-results { padding: 2rem 0; }

  .search-tips {
    margin-top: 2rem;
    padding: 1.5rem;
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
  }
  .search-tips h3 { margin-bottom: 0.75rem; color: var(--color-text-muted); font-size: 0.875rem; }
  .tips-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .tips-list li {
    font-size: 0.875rem;
    color: var(--color-text-dim);
    padding-left: 1.25rem;
    position: relative;
  }
  .tips-list li::before {
    content: '▸';
    position: absolute;
    left: 0;
    color: var(--color-primary);
  }
  code {
    background: var(--color-bg-elevated);
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--color-accent);
  }

  .chip {
    padding: 0.15rem 0.5rem;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-full);
    font-size: 0.7rem;
    color: var(--color-text-muted);
  }
</style>
