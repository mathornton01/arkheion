/**
 * Arkheion API client
 *
 * Wraps all backend API calls. All functions return the parsed JSON response
 * or throw an Error with a descriptive message.
 *
 * The API key and base URL are sourced from environment variables.
 * In the browser: PUBLIC_API_BASE_URL
 * In SvelteKit server hooks/load functions: INTERNAL_API_BASE_URL
 */

// In the browser, use PUBLIC_API_BASE_URL.
// In SSR context, use INTERNAL_API_BASE_URL (set by the server +layout.server.js).
const BASE_URL =
  (typeof window === 'undefined'
    ? process.env.INTERNAL_API_BASE_URL
    : undefined) ||
  import.meta.env.PUBLIC_API_BASE_URL ||
  '/api/v1';

const API_KEY =
  (typeof window === 'undefined' ? process.env.FRONTEND_API_KEY : undefined) || '';

/**
 * Core fetch wrapper. Adds auth header and handles error responses.
 * @param {string} path - API path relative to base URL (e.g. "/books")
 * @param {RequestInit} options - Fetch options
 * @returns {Promise<any>} Parsed JSON response
 */
async function apiFetch(path, options = {}) {
  const url = `${BASE_URL}${path}`;
  const headers = {
    'Content-Type': 'application/json',
    'X-API-Key': API_KEY,
    ...options.headers
  };

  // Don't set Content-Type for FormData (browser sets it with boundary)
  if (options.body instanceof FormData) {
    delete headers['Content-Type'];
  }

  const response = await fetch(url, { ...options, headers });

  if (response.status === 204) return null;

  const data = await response.json().catch(() => null);

  if (!response.ok) {
    const message =
      data?.error?.message ||
      data?.message ||
      `API error: ${response.status} ${response.statusText}`;
    const error = new Error(message);
    error.status = response.status;
    error.code = data?.error?.code;
    throw error;
  }

  return data;
}

// =============================================================================
// Books
// =============================================================================

/**
 * List books with optional filters and pagination.
 * @param {object} params
 * @param {number} [params.page]
 * @param {number} [params.per_page]
 * @param {string} [params.tag]
 * @param {string} [params.category]
 * @param {string} [params.language]
 * @param {boolean} [params.text_extracted]
 * @param {string} [params.q]
 * @returns {Promise<{data: Book[], pagination: Pagination}>}
 */
export async function listBooks(params = {}) {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') {
      qs.append(k, String(v));
    }
  });
  return apiFetch(`/books${qs.toString() ? '?' + qs : ''}`);
}

/**
 * Get a single book by ID.
 * @param {string} id - Book UUID
 * @returns {Promise<Book>}
 */
export async function getBook(id) {
  return apiFetch(`/books/${id}`);
}

/**
 * Create a new book.
 * @param {CreateBookRequest} data
 * @returns {Promise<Book>}
 */
export async function createBook(data) {
  return apiFetch('/books', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/**
 * Update a book (partial update — only provided fields change).
 * @param {string} id
 * @param {Partial<CreateBookRequest>} data
 * @returns {Promise<Book>}
 */
export async function updateBook(id, data) {
  return apiFetch(`/books/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/**
 * Delete a book.
 * @param {string} id
 * @returns {Promise<null>}
 */
export async function deleteBook(id) {
  return apiFetch(`/books/${id}`, { method: 'DELETE' });
}

/**
 * Upload a book file (PDF, EPUB, etc.).
 * @param {string} id - Book UUID
 * @param {File} file - File object from input element
 * @param {(progress: number) => void} [onProgress] - Upload progress callback (0-100)
 * @returns {Promise<{file_path: string, file_type: string, file_size_bytes: number}>}
 */
export async function uploadBookFile(id, file, onProgress) {
  return new Promise((resolve, reject) => {
    const formData = new FormData();
    formData.append('file', file);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', `${BASE_URL}/books/${id}/upload`);
    xhr.setRequestHeader('X-API-Key', API_KEY);

    if (onProgress) {
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          onProgress(Math.round((e.loaded / e.total) * 100));
        }
      });
    }

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(JSON.parse(xhr.responseText));
      } else {
        const err = JSON.parse(xhr.responseText || '{}');
        reject(new Error(err?.error?.message || `Upload failed: ${xhr.status}`));
      }
    };

    xhr.onerror = () => reject(new Error('Network error during upload'));
    xhr.send(formData);
  });
}

/**
 * Get the download URL for a book file (proxied through backend).
 * @param {string} id - Book UUID
 * @returns {string} URL to use in an <a href> or window.open()
 */
export function getDownloadURL(id) {
  return `${BASE_URL}/books/${id}/download`;
}

// =============================================================================
// Catalog (ISBN lookup)
// =============================================================================

/**
 * Look up book metadata by ISBN.
 * @param {string} isbn - ISBN-10 or ISBN-13 (dashes optional)
 * @returns {Promise<ISBNLookupResult>}
 */
export async function lookupISBN(isbn) {
  const normalized = isbn.replace(/[-\s]/g, '');
  return apiFetch(`/catalog/isbn/${normalized}`);
}

/**
 * Look up book metadata from a scanned barcode string.
 * @param {string} barcode - Raw barcode string from ZXing
 * @returns {Promise<ISBNLookupResult>}
 */
export async function scanBarcode(barcode) {
  return apiFetch('/catalog/scan', {
    method: 'POST',
    body: JSON.stringify({ barcode })
  });
}

// =============================================================================
// Search
// =============================================================================

/**
 * Full-text search via Meilisearch.
 * @param {string} query
 * @param {object} [params]
 * @param {number} [params.page]
 * @param {number} [params.per_page]
 * @param {string} [params.filter] - Meilisearch filter expression
 * @returns {Promise<SearchResult>}
 */
export async function search(query, params = {}) {
  const qs = new URLSearchParams({ q: query });
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') qs.append(k, String(v));
  });
  return apiFetch(`/search?${qs}`);
}

// =============================================================================
// Webhooks
// =============================================================================

/**
 * List all registered webhooks.
 * @returns {Promise<{data: Webhook[]}>}
 */
export async function listWebhooks() {
  return apiFetch('/webhooks');
}

/**
 * Create a new webhook.
 * @param {CreateWebhookRequest} data
 * @returns {Promise<Webhook>}
 */
export async function createWebhook(data) {
  return apiFetch('/webhooks', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/**
 * Delete a webhook by ID.
 * @param {string} id
 * @returns {Promise<null>}
 */
export async function deleteWebhook(id) {
  return apiFetch(`/webhooks/${id}`, { method: 'DELETE' });
}

/**
 * Activate a webhook.
 * @param {string} id
 * @returns {Promise<Webhook>}
 */
export async function activateWebhook(id) {
  return apiFetch(`/webhooks/${id}/activate`, { method: 'PUT' });
}

/**
 * Deactivate a webhook.
 * @param {string} id
 * @returns {Promise<Webhook>}
 */
export async function deactivateWebhook(id) {
  return apiFetch(`/webhooks/${id}/deactivate`, { method: 'PUT' });
}

// =============================================================================
// Export
// =============================================================================

/**
 * Get the JSONL export URL with optional filters.
 * Use this URL in an <a download> tag or window.open().
 * @param {object} [params]
 * @param {string} [params.tag]
 * @param {string} [params.category]
 * @param {string} [params.language]
 * @returns {string}
 */
export function getExportURL(params = {}) {
  const qs = new URLSearchParams({ format: 'jsonl' });
  Object.entries(params).forEach(([k, v]) => {
    if (v) qs.append(k, v);
  });
  // Note: the export endpoint requires auth. The frontend constructs a URL
  // but the actual request must include the API key header.
  return `${BASE_URL}/export?${qs}`;
}

/**
 * Trigger a JSONL export and return the raw text.
 * For large libraries, use getExportURL() with window.open() instead.
 * @param {object} [params]
 * @returns {Promise<string>} Raw JSONL text
 */
export async function exportBooks(params = {}) {
  const qs = new URLSearchParams({ format: 'jsonl' });
  Object.entries(params).forEach(([k, v]) => {
    if (v) qs.append(k, v);
  });

  const response = await fetch(`${BASE_URL}/export?${qs}`, {
    headers: { 'X-API-Key': API_KEY }
  });
  if (!response.ok) throw new Error(`Export failed: ${response.status}`);
  return response.text();
}
