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

<svelte:head><title>Search — Arkheion</title></svelte:head>

<div class="search-page">
  <h1>Search</h1>

  <form class="search-form" on:submit={handleSubmit}>
    <div class="search-input-wrap">
      <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
      </svg>
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
        <button type="button" class="clear-btn" on:click={() => { searchQuery.set(''); searchState.set(null); }}>&#x2715;</button>
      {/if}
    </div>
    <button type="submit" class="btn btn-primary" disabled={loading}>
      {loading ? 'Searching…' : 'Search'}
    </button>
  </form>

  <div class="filter-bar">
    <span class="text-xs text-dim filter-label">Filter</span>
    <input class="filter-input" type="text" placeholder='e.g. language="en" AND text_extracted=true'
      bind:value={filter} on:change={() => $searchQuery && performSearch($searchQuery)} />
    <button class="btn btn-secondary" style="font-size:0.75rem; padding:0.35rem 0.65rem"
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
    </div>
  {:else if $searchState}
    <div class="results-header">
      <p class="text-muted text-sm">
        {($searchState.total_hits ?? $searchState.estimated_total_hits ?? 0).toLocaleString()} results
        for <strong>"{$searchState.query}"</strong>
        <span class="text-dim"> &middot; {$searchState.processing_time_ms}ms</span>
      </p>
    </div>

    {#if $searchState.hits?.length === 0}
      <div class="empty-results">
        <p class="text-muted">No results for "<strong>{$searchState.query}</strong>".</p>
        <p class="text-dim text-sm" style="margin-top:0.5rem">
          Full-text search requires uploaded and extracted files.
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
                <div class="result-thumb-placeholder">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.25" width="22" height="22">
                    <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
                  </svg>
                </div>
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
                    <span class="badge badge-success">Indexed</span>
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
      <p class="tips-heading">Tips</p>
      <ul class="tips-list">
        <li>Search by title, author, description, category, or tag</li>
        <li>Books with uploaded files include full-text results</li>
        <li>Use Meilisearch filters: <code>language="en"</code>, <code>text_extracted=true</code></li>
        <li>Meilisearch handles typos automatically</li>
      </ul>
    </div>
  {/if}
</div>

<style>
  .search-page { max-width: 760px; }
  h1 { margin-bottom: 1.375rem; }

  .search-form {
    display: flex;
    gap: 0.625rem;
    margin-bottom: 0.625rem;
  }

  .search-input-wrap {
    position: relative;
    flex: 1;
    display: flex;
    align-items: center;
  }
  .search-icon {
    position: absolute;
    left: 0.75rem;
    width: 15px;
    height: 15px;
    color: var(--color-text-dim);
    pointer-events: none;
  }
  .search-input {
    width: 100%;
    padding: 0.625rem 2.25rem 0.625rem 2.375rem;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    color: var(--color-text);
    font-size: 0.9rem;
    outline: none;
    transition: border-color var(--transition);
    font-family: var(--font-sans);
  }
  .search-input:focus { border-color: var(--color-primary); }
  .clear-btn {
    position: absolute;
    right: 0.625rem;
    background: none;
    border: none;
    color: var(--color-text-dim);
    cursor: pointer;
    font-size: 0.75rem;
    line-height: 1;
    padding: 0.125rem;
  }
  .clear-btn:hover { color: var(--color-text-muted); }

  .filter-bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1.375rem;
  }
  .filter-label { flex-shrink: 0; }
  .filter-input {
    flex: 1;
    padding: 0.35rem 0.575rem;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    color: var(--color-text);
    font-size: 0.775rem;
    font-family: var(--font-mono);
    outline: none;
    transition: border-color var(--transition);
  }
  .filter-input:focus { border-color: var(--color-border-strong); }

  .loading-state {
    display: flex;
    justify-content: center;
    padding: 3rem;
  }

  .results-header { margin-bottom: 0.875rem; }
  .results-list { display: flex; flex-direction: column; gap: 0.5rem; }

  .result-item.card {
    padding: 0.875rem 1rem;
    text-decoration: none;
    display: block;
    transition: border-color var(--transition);
  }
  .result-item.card:hover {
    border-color: var(--color-border-strong);
  }
  .result-layout { display: flex; gap: 0.875rem; }
  .result-thumb {
    width: 48px;
    height: 70px;
    object-fit: cover;
    border-radius: 3px;
    flex-shrink: 0;
  }
  .result-thumb-placeholder {
    width: 48px;
    height: 70px;
    background: var(--color-bg-elevated);
    border-radius: 3px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--color-text-dim);
    flex-shrink: 0;
  }
  .result-body { flex: 1; min-width: 0; }
  .result-title {
    font-size: 0.9rem;
    font-weight: 600;
    margin-bottom: 0.2rem;
    color: var(--color-text);
  }
  .result-author { margin-bottom: 0.375rem; }
  .result-excerpt { line-height: 1.55; margin-bottom: 0.4rem; }
  .result-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
    align-items: center;
  }

  :global(.result-title mark),
  :global(.result-author mark),
  :global(.result-excerpt mark) {
    background: rgba(192, 57, 43, 0.2);
    color: var(--color-accent);
    border-radius: 2px;
    padding: 0 1px;
  }

  .empty-results { padding: 1.5rem 0; }

  .search-tips {
    margin-top: 1.5rem;
    padding: 1.25rem;
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
  }
  .tips-heading {
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--color-text-dim);
    text-transform: uppercase;
    letter-spacing: 0.07em;
    margin-bottom: 0.625rem;
  }
  .tips-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .tips-list li {
    font-size: 0.825rem;
    color: var(--color-text-muted);
    padding-left: 1rem;
    position: relative;
  }
  .tips-list li::before {
    content: '';
    position: absolute;
    left: 0;
    top: 0.55em;
    width: 4px;
    height: 4px;
    border-radius: 50%;
    background: var(--color-primary);
  }
  code {
    background: var(--color-bg-elevated);
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
    font-family: var(--font-mono);
    font-size: 0.775rem;
    color: var(--color-text-muted);
    border: 1px solid var(--color-border);
  }

  .chip {
    padding: 0.1rem 0.45rem;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-full);
    font-size: 0.675rem;
    color: var(--color-text-muted);
  }
</style>
