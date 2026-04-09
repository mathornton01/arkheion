<script>
  import { onMount, onDestroy } from 'svelte';
  import { goto } from '$app/navigation';
  import { BarcodeScanner, isCameraSupported, requestCameraPermission } from '$lib/scanner.js';
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

  // Tags to add to scanned books (user preference)
  let scanTags = 'scanned';

  const supported = isCameraSupported();

  onDestroy(() => {
    scanner?.stop();
  });

  async function startScanner() {
    cameraError = '';
    const granted = await requestCameraPermission();
    if (!granted) {
      cameraError = 'Camera permission denied. Please allow camera access in your browser settings.';
      return;
    }

    scanner = new BarcodeScanner(videoEl);
    scanning = true;
    scannerActive.set(true);

    scanner.onResult = async (barcode) => {
      // Immediately show the scanned value
      lastScannedISBN.set(barcode);
      // Pause scanning while we look up
      await performLookup(barcode);
    };

    scanner.onError = (err) => {
      if (err.name !== 'NotFoundException') {
        cameraError = `Scanner error: ${err.message}`;
      }
    };

    try {
      await scanner.start();
    } catch (err) {
      scanning = false;
      scannerActive.set(false);
      cameraError = `Failed to start camera: ${err.message}`;
    }
  }

  function stopScanner() {
    scanner?.stop();
    scanning = false;
    scannerActive.set(false);
  }

  async function performLookup(barcode) {
    // Pause scanner while looking up
    scanner?.stop();
    scanning = false;
    lookupLoading = true;
    lookupError = '';
    lookupResult = null;

    try {
      lookupResult = await scanBarcode(barcode);
    } catch (err) {
      if (err.code === 'ISBN_NOT_FOUND') {
        lookupError = `No book found for barcode: ${barcode}. You can add it manually.`;
        // Pre-fill ISBN at least
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
      notify.success(`"${book.title}" added to library!`);
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
    <p class="text-muted">Point your camera at a book's ISBN barcode to add it to your library.</p>
  </div>

  {#if !supported}
    <div class="alert alert-error">
      Your browser does not support camera access. Try using a modern mobile browser (Chrome or Safari).
    </div>
  {:else}

    {#if !lookupResult}
      <!-- Camera view -->
      <div class="scanner-section">
        <div class="video-container" class:active={scanning}>
          <!-- svelte-ignore a11y-media-has-caption -->
          <video bind:this={videoEl} class="video-feed" playsinline autoplay muted></video>
          {#if scanning}
            <div class="scan-overlay">
              <div class="scan-frame"></div>
            </div>
          {:else}
            <div class="video-placeholder">
              <span class="placeholder-icon">📷</span>
              <span>Camera feed will appear here</span>
            </div>
          {/if}
        </div>

        {#if cameraError}
          <div class="alert alert-error">{cameraError}</div>
        {/if}

        <div class="scanner-controls">
          {#if !scanning}
            <button class="btn btn-primary" on:click={startScanner}>
              📷 Start Camera
            </button>
          {:else}
            <button class="btn btn-secondary" on:click={stopScanner}>
              ⏹ Stop Camera
            </button>
          {/if}
        </div>

        {#if $lastScannedISBN && lookupLoading}
          <div class="scanning-status">
            <div class="spinner"></div>
            <p>Looking up: <strong>{$lastScannedISBN}</strong></p>
          </div>
        {/if}
      </div>

      <!-- Manual ISBN entry -->
      <div class="manual-section">
        <hr class="divider" />
        <h3>Or enter ISBN manually</h3>
        <form class="manual-form" on:submit={handleManualISBN}>
          <input class="input" name="isbn" type="text"
            placeholder="ISBN-10 or ISBN-13 (e.g. 9780345539434)"
            pattern="[\d\-X]+" />
          <button type="submit" class="btn btn-primary">🔍 Lookup</button>
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
              <div class="result-cover-placeholder">📖</div>
            {/if}

            <div class="result-info">
              <div class="result-source text-xs text-dim mb-2">
                Found via {lookupResult.source === 'openlibrary' ? 'OpenLibrary' : lookupResult.source === 'google_books' ? 'Google Books' : 'Manual entry'}
              </div>

              {#if lookupResult.title}
                <h2 class="result-title">{lookupResult.title}</h2>
              {:else}
                <p class="text-muted">No title found — please edit after adding</p>
              {/if}

              {#if lookupResult.authors?.length > 0}
                <p class="result-authors">by {lookupResult.authors.join(', ')}</p>
              {/if}
              {#if lookupResult.publisher}
                <p class="text-muted text-sm">{lookupResult.publisher}
                  {#if lookupResult.published_date} · {new Date(lookupResult.published_date).getFullYear()}{/if}
                </p>
              {/if}
              {#if lookupResult.isbn}
                <p class="text-dim text-xs">ISBN: {lookupResult.isbn}</p>
              {/if}
            </div>
          </div>

          {#if lookupResult.description}
            <p class="result-description text-muted text-sm">{lookupResult.description.slice(0, 300)}
              {#if lookupResult.description.length > 300}…{/if}
            </p>
          {/if}

          <!-- Tags for this scan -->
          <div class="field mt-4">
            <label class="label" for="scan-tags">Tags to add (comma-separated)</label>
            <input id="scan-tags" class="input" type="text" bind:value={scanTags}
              placeholder="scanned, to-read" />
          </div>

          <div class="result-actions">
            <button class="btn btn-secondary" on:click={scanAnother}>
              📷 Scan Another
            </button>
            <a href="/library" class="btn btn-secondary">
              Add Manually
            </a>
            <button class="btn btn-primary" on:click={addBook} disabled={addingBook}>
              {addingBook ? 'Adding…' : '+ Add to Library'}
            </button>
          </div>
        </div>
      </div>
    {/if}
  {/if}

  <!-- Tips -->
  <div class="tips card mt-4">
    <h3 class="mb-2">Tips for scanning</h3>
    <ul class="tips-list">
      <li>Hold the book's barcode in good lighting</li>
      <li>Most ISBNs are on the back cover, starting with 978 or 979</li>
      <li>Keep the camera 15–25 cm from the barcode</li>
      <li>If scanning fails, use the manual ISBN entry below</li>
    </ul>
  </div>
</div>

<style>
  .scan-page { max-width: 680px; }

  .page-header { margin-bottom: 2rem; }
  .page-header h1 { margin-bottom: 0.25rem; }

  .scanner-section { margin-bottom: 1.5rem; }

  .video-container {
    position: relative;
    width: 100%;
    background: var(--color-bg-card);
    border: 2px solid var(--color-border);
    border-radius: var(--radius-lg);
    overflow: hidden;
    aspect-ratio: 4/3;
    max-height: 360px;
    margin-bottom: 1rem;
    transition: border-color var(--transition);
  }
  .video-container.active {
    border-color: var(--color-primary);
  }

  .video-feed {
    width: 100%;
    height: 100%;
    object-fit: cover;
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
    font-size: 0.875rem;
  }
  .placeholder-icon { font-size: 2.5rem; }

  /* Scan overlay — animated frame */
  .scan-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: none;
  }
  .scan-frame {
    width: 240px;
    height: 100px;
    border: 2px solid var(--color-accent);
    border-radius: 4px;
    box-shadow: 0 0 0 4000px rgba(0,0,0,0.35);
    animation: pulse 1.5s ease-in-out infinite;
  }
  @keyframes pulse {
    0%, 100% { border-color: var(--color-accent); }
    50%       { border-color: var(--color-primary); }
  }

  .scanner-controls { display: flex; gap: 0.75rem; justify-content: center; }

  .scanning-status {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    justify-content: center;
    margin-top: 1rem;
    color: var(--color-text-muted);
    font-size: 0.875rem;
  }

  .manual-section { margin-top: 1.5rem; }
  .manual-section h3 { margin-bottom: 0.75rem; }
  .manual-form {
    display: flex;
    gap: 0.75rem;
  }
  .manual-form .input { flex: 1; }

  /* Result */
  .result-section {}
  .result-card { padding: 1.5rem; }
  .result-layout {
    display: flex;
    gap: 1.5rem;
    margin-bottom: 1rem;
  }
  .result-cover {
    width: 100px;
    height: 140px;
    object-fit: cover;
    border-radius: var(--radius);
    flex-shrink: 0;
  }
  .result-cover-placeholder {
    width: 100px;
    height: 140px;
    background: var(--color-bg-elevated);
    border-radius: var(--radius);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 2rem;
    flex-shrink: 0;
  }
  .result-info { flex: 1; }
  .result-title { font-size: 1.25rem; margin-bottom: 0.25rem; }
  .result-authors { color: var(--color-accent); font-size: 0.9rem; margin-bottom: 0.25rem; }
  .result-description { margin-top: 0.75rem; line-height: 1.6; }
  .result-actions {
    display: flex;
    gap: 0.75rem;
    margin-top: 1.25rem;
    flex-wrap: wrap;
  }

  /* Tips */
  .tips h3 { color: var(--color-text-muted); font-size: 0.875rem; }
  .tips-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }
  .tips-list li {
    font-size: 0.8rem;
    color: var(--color-text-dim);
    padding-left: 1.25rem;
    position: relative;
  }
  .tips-list li::before {
    content: '•';
    position: absolute;
    left: 0.375rem;
    color: var(--color-primary);
  }
</style>
