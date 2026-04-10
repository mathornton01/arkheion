<script>
  import { onMount, onDestroy } from 'svelte';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import { scanBarcode, createBook } from '$lib/api.js';
  import { lastScannedISBN, scannerActive, notify } from '$lib/stores.js';

  let videoEl;
  let scanner = null;
  let scanning = false;
  let cameraError = '';
  let lookupResult = null;
  let lookupLoading = false;
  let lookupError = '';
  let addingBook = false;
  let scanTags = 'scanned';
  let supported = false;

  // Dynamically import scanner only in browser (ZXing uses browser APIs, breaks SSR)
  let BarcodeScanner = null;

  onMount(async () => {
    if (!browser) return;
    const mod = await import('$lib/scanner.js');
    BarcodeScanner = mod.BarcodeScanner;
    supported = mod.isCameraSupported();
  });

  onDestroy(() => { scanner?.stop(); });

  async function startScanner() {
    if (!browser || !BarcodeScanner) return;
    cameraError = '';

    scanner = new BarcodeScanner(videoEl);
    scanning = true;
    scannerActive.set(true);

    scanner.onResult = async (barcode) => {
      lastScannedISBN.set(barcode);
      await performLookup(barcode);
    };

    scanner.onError = (err) => {
      // Suppress "not found" noise — only show real errors
      const msg = err.message || '';
      const isNotFound =
        err.name === 'NotFoundException' ||
        msg.includes('No MultiFormat') ||
        msg.includes('not found');
      if (!isNotFound) {
        cameraError = `Scanner error: ${err.message}`;
      }
    };

    try {
      await scanner.start();
    } catch (err) {
      scanning = false;
      scannerActive.set(false);
      if (err.name === 'NotAllowedError' || err.name === 'PermissionDeniedError') {
        cameraError = 'Camera permission denied. Allow camera access in your browser settings then reload.';
      } else {
        cameraError = `Failed to start camera: ${err.message}`;
      }
    }
  }

  function stopScanner() {
    scanner?.stop();
    scanning = false;
    scannerActive.set(false);
  }

  async function performLookup(barcode) {
    scanner?.stop();
    scanning = false;
    scannerActive.set(false);
    lookupLoading = true;
    lookupError = '';
    lookupResult = null;

    try {
      lookupResult = await scanBarcode(barcode);
    } catch (err) {
      if (err.code === 'ISBN_NOT_FOUND') {
        lookupError = `No match found for barcode: ${barcode}.`;
        lookupResult = { isbn: barcode, title: '', authors: [], source: 'manual' };
      } else {
        lookupError = `Lookup failed: ${err.message}`;
      }
    } finally {
      lookupLoading = false;
    }
  }

  async function handleManualISBN(event) {
    event.preventDefault();
    const isbn = event.target.elements.isbn.value.trim();
    if (!isbn) return;
    lastScannedISBN.set(isbn);
    await performLookup(isbn);
  }

  async function addBook() {
    if (!lookupResult) return;
    addingBook = true;
    try {
      const book = await createBook({
        isbn: lookupResult.isbn,
        title: lookupResult.title,
        authors: lookupResult.authors || [],
        publisher: lookupResult.publisher,
        published_date: lookupResult.published_date,
        description: lookupResult.description,
        page_count: lookupResult.page_count,
        categories: lookupResult.categories || [],
        language: lookupResult.language || 'en',
        cover_url: lookupResult.cover_url,
        tags: scanTags.split(',').map(t => t.trim()).filter(Boolean)
      });
      notify.success(`"${book.title}" added to library`);
      goto(`/library/${book.id}`);
    } catch (err) {
      notify.error('Failed to add book: ' + err.message);
      addingBook = false;
    }
  }

  function scanAnother() {
    lookupResult = null;
    lookupError = '';
    lastScannedISBN.set(null);
    startScanner();
  }
</script>

<svelte:head><title>Scan — Arkheion</title></svelte:head>

<div class="scan-page">
  <div class="page-header">
    <h1>Scan Barcode</h1>
    <p class="text-muted text-sm">Point your webcam at a book's ISBN barcode to add it to your library.</p>
  </div>

  {#if !supported && browser}
    <div class="alert alert-error">
      Camera access is not supported in this browser. Try Chrome, Edge, or Firefox with camera permissions enabled.
    </div>
  {:else}

    {#if !lookupResult}
      <div class="scanner-section">
        <div class="video-container" class:active={scanning}>
          <!-- svelte-ignore a11y-media-has-caption -->
          <video bind:this={videoEl} class="video-feed" playsinline muted></video>
          {#if scanning}
            <div class="scan-overlay">
              <div class="scan-frame">
                <div class="scan-line"></div>
              </div>
            </div>
          {:else}
            <div class="video-placeholder">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.25" width="40" height="40">
                <path d="M3 9V6a2 2 0 0 1 2-2h3"/><path d="M21 9V6a2 2 0 0 0-2-2h-3"/>
                <path d="M3 15v3a2 2 0 0 0 2 2h3"/><path d="M21 15v3a2 2 0 0 1-2 2h-3"/>
                <line x1="7" y1="8" x2="7" y2="16"/><line x1="11" y1="8" x2="11" y2="16"/>
                <line x1="15" y1="8" x2="15" y2="16"/><line x1="17" y1="8" x2="17" y2="16"/>
              </svg>
              <span>Camera inactive</span>
              <span class="text-xs" style="color: var(--color-text-dim)">Click Start Camera to begin</span>
            </div>
          {/if}
        </div>

        {#if cameraError}
          <div class="alert alert-error">{cameraError}</div>
        {/if}

        <div class="scanner-controls">
          {#if !scanning}
            <button class="btn btn-primary" on:click={startScanner}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="15" height="15">
                <circle cx="12" cy="12" r="3"/><path d="M3 7c0-1.1.9-2 2-2h2l2-3h6l2 3h2a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z"/>
              </svg>
              Start Camera
            </button>
          {:else}
            <button class="btn btn-secondary" on:click={stopScanner}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="15" height="15">
                <rect x="3" y="3" width="18" height="18" rx="2"/>
              </svg>
              Stop Camera
            </button>
          {/if}
        </div>

        {#if $lastScannedISBN && lookupLoading}
          <div class="scanning-status">
            <div class="spinner"></div>
            <span>Looking up <strong>{$lastScannedISBN}</strong></span>
          </div>
        {/if}
      </div>

      <div class="manual-section">
        <hr class="divider" />
        <h3 class="manual-heading">Enter ISBN manually</h3>
        <form class="manual-form" on:submit={handleManualISBN}>
          <input class="input" name="isbn" type="text"
            placeholder="ISBN-10 or ISBN-13 (e.g. 9780345539434)"
            pattern="[\d\-X]+" />
          <button type="submit" class="btn btn-primary">Lookup</button>
        </form>
      </div>
    {:else}
      <!-- Lookup result -->
      <div class="result-section">
        {#if lookupError}
          <div class="alert alert-error">{lookupError}</div>
        {/if}

        <div class="result-card card">
          <div class="result-layout">
            {#if lookupResult.cover_url}
              <img src={lookupResult.cover_url} alt={lookupResult.title} class="result-cover" />
            {:else}
              <div class="result-cover-placeholder">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" width="36" height="36">
                  <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
                </svg>
              </div>
            {/if}

            <div class="result-info">
              <div class="result-source">
                {lookupResult.source === 'openlibrary' ? 'OpenLibrary' : lookupResult.source === 'google_books' ? 'Google Books' : 'Manual entry'}
              </div>

              <div class="field">
                <label class="label" for="scan-title">Title</label>
                <input id="scan-title" class="input" type="text" bind:value={lookupResult.title}
                  placeholder="Book title (required)" required />
              </div>

              {#if lookupResult.authors?.length > 0}
                <p class="result-authors">{lookupResult.authors.join(', ')}</p>
              {/if}
              {#if lookupResult.publisher}
                <p class="text-muted text-sm">
                  {lookupResult.publisher}
                  {#if lookupResult.published_date} &middot; {new Date(lookupResult.published_date).getFullYear()}{/if}
                </p>
              {/if}
              {#if lookupResult.isbn}
                <p class="text-dim text-xs">ISBN: {lookupResult.isbn}</p>
              {/if}
            </div>
          </div>

          {#if lookupResult.description}
            <p class="result-description text-muted text-sm">{lookupResult.description.slice(0, 300)}{#if lookupResult.description.length > 300}…{/if}</p>
          {/if}

          <div class="field mt-4">
            <label class="label" for="scan-tags">Tags (comma-separated)</label>
            <input id="scan-tags" class="input" type="text" bind:value={scanTags}
              placeholder="scanned, to-read" />
          </div>

          <div class="result-actions">
            <button class="btn btn-secondary" on:click={scanAnother}>Scan Another</button>
            <a href="/library" class="btn btn-secondary">Add Manually</a>
            <button class="btn btn-primary" on:click={addBook}
              disabled={addingBook || !lookupResult?.title?.trim()}>
              {addingBook ? 'Adding…' : 'Add to Library'}
            </button>
          </div>
        </div>
      </div>
    {/if}
  {/if}

  <div class="tips card mt-4">
    <p class="tips-heading">Tips</p>
    <ul class="tips-list">
      <li>Good lighting significantly improves scan accuracy</li>
      <li>ISBNs are on the back cover, usually starting with 978 or 979</li>
      <li>Hold the barcode 15–25 cm from the camera</li>
      <li>Keep the barcode centered in the red frame</li>
      <li>Use manual entry below if the camera scan fails</li>
    </ul>
  </div>
</div>

<style>
  .scan-page { max-width: 640px; }

  .page-header { margin-bottom: 1.75rem; }
  .page-header h1 { margin-bottom: 0.25rem; }

  .scanner-section { margin-bottom: 1.25rem; }

  .video-container {
    position: relative;
    width: 100%;
    background: #0a0a0a;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    overflow: hidden;
    aspect-ratio: 4/3;
    max-height: 360px;
    margin-bottom: 0.875rem;
    transition: border-color var(--transition);
  }
  .video-container.active {
    border-color: var(--color-primary);
    box-shadow: 0 0 0 1px var(--color-primary);
  }

  .video-feed {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border: 3px solid #2ecc71;
    box-sizing: border-box;
  }

  .video-placeholder {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    color: var(--color-text-dim);
    font-size: 0.8rem;
  }

  .scan-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: none;
  }

  .scan-frame {
    position: relative;
    width: 240px;
    height: 100px;
    border: 2px solid var(--color-primary);
    border-radius: 4px;
    box-shadow: 0 0 0 4000px rgba(0,0,0,0.45);
    overflow: hidden;
  }

  .scan-line {
    position: absolute;
    left: 0;
    right: 0;
    height: 2px;
    background: linear-gradient(90deg, transparent, var(--color-primary), transparent);
    animation: scan 2s ease-in-out infinite;
    top: 0;
  }

  @keyframes scan {
    0%   { top: 0; opacity: 1; }
    50%  { top: calc(100% - 2px); opacity: 1; }
    100% { top: 0; opacity: 1; }
  }

  .scanner-controls {
    display: flex;
    justify-content: center;
    gap: 0.625rem;
  }

  .scanner-controls .btn {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .scanning-status {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    justify-content: center;
    margin-top: 0.875rem;
    color: var(--color-text-muted);
    font-size: 0.825rem;
  }

  .manual-section { margin-top: 1.25rem; }
  .manual-heading {
    font-size: 0.825rem;
    font-weight: 500;
    color: var(--color-text-muted);
    margin-bottom: 0.625rem;
  }
  .manual-form { display: flex; gap: 0.625rem; }
  .manual-form .input { flex: 1; }

  /* Result */
  .result-card { padding: 1.375rem; }
  .result-layout { display: flex; gap: 1.25rem; margin-bottom: 0.875rem; }
  .result-cover {
    width: 88px;
    height: 124px;
    object-fit: cover;
    border-radius: var(--radius);
    flex-shrink: 0;
  }
  .result-cover-placeholder {
    width: 88px;
    height: 124px;
    background: var(--color-bg-elevated);
    border-radius: var(--radius);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--color-text-dim);
    flex-shrink: 0;
  }
  .result-info { flex: 1; }
  .result-source {
    font-size: 0.675rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--color-text-dim);
    margin-bottom: 0.4rem;
  }
  .result-title { font-size: 1.1rem; margin-bottom: 0.2rem; }
  .result-authors { color: var(--color-accent); font-size: 0.85rem; margin-bottom: 0.2rem; font-weight: 500; }
  .result-description { margin-top: 0.625rem; line-height: 1.6; }
  .result-actions {
    display: flex;
    gap: 0.625rem;
    margin-top: 1rem;
    flex-wrap: wrap;
  }

  /* Tips */
  .tips { padding: 1rem 1.25rem; }
  .tips-heading {
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--color-text-dim);
    margin-bottom: 0.5rem;
  }
  .tips-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  .tips-list li {
    font-size: 0.775rem;
    color: var(--color-text-muted);
    padding-left: 1rem;
    position: relative;
  }
  .tips-list li::before {
    content: '';
    position: absolute;
    left: 0.25rem;
    top: 0.55em;
    width: 3px;
    height: 3px;
    border-radius: 50%;
    background: var(--color-primary);
  }
</style>
