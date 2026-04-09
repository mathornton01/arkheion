/**
 * Arkheion — Svelte stores
 *
 * Global reactive state shared across the application.
 * All stores are writable and exported for use in any component.
 */

import { writable, derived } from 'svelte/store';

// =============================================================================
// Notification / Toast store
// =============================================================================

/**
 * @typedef {object} Toast
 * @property {string} id
 * @property {'success'|'error'|'info'|'warning'} type
 * @property {string} message
 * @property {number} duration - Auto-dismiss duration in ms (0 = no auto-dismiss)
 */

/** @type {import('svelte/store').Writable<Toast[]>} */
export const toasts = writable([]);

let toastCounter = 0;

/**
 * Add a toast notification.
 * @param {'success'|'error'|'info'|'warning'} type
 * @param {string} message
 * @param {number} [duration=4000]
 */
export function addToast(type, message, duration = 4000) {
  const id = String(++toastCounter);
  toasts.update((t) => [...t, { id, type, message, duration }]);
  if (duration > 0) {
    setTimeout(() => dismissToast(id), duration);
  }
  return id;
}

export function dismissToast(id) {
  toasts.update((t) => t.filter((toast) => toast.id !== id));
}

export const notify = {
  success: (msg, dur) => addToast('success', msg, dur),
  error: (msg, dur) => addToast('error', msg, dur),
  info: (msg, dur) => addToast('info', msg, dur),
  warning: (msg, dur) => addToast('warning', msg, dur)
};

// =============================================================================
// Library / Books store
// =============================================================================

/**
 * @typedef {object} Book
 * @property {string} id
 * @property {string} title
 * @property {string} [subtitle]
 * @property {Array<{name: string}>} [authors]
 * @property {string} [cover_url]
 * @property {string} [file_type]
 * @property {boolean} text_extracted
 * @property {string[]} [categories]
 * @property {Array<{name: string, slug: string}>} [tags]
 */

/** Currently loaded books (from last list call) */
export const books = writable(/** @type {Book[]} */ ([]));

/** Pagination state from last list call */
export const bookPagination = writable({
  page: 1,
  per_page: 20,
  total: 0,
  total_pages: 1
});

/** Active filters on the library page */
export const bookFilters = writable({
  tag: '',
  category: '',
  language: '',
  text_extracted: null,
  q: ''
});

/** Currently viewing/editing book */
export const currentBook = writable(/** @type {Book|null} */ (null));

// Derived store: total book count
export const totalBooks = derived(bookPagination, ($p) => $p.total);

// =============================================================================
// Search store
// =============================================================================

/** @type {import('svelte/store').Writable<{query: string, results: any[], processing_time_ms: number}|null>} */
export const searchState = writable(null);

export const searchQuery = writable('');

// =============================================================================
// UI State
// =============================================================================

/** Whether the "Add Book" modal is open */
export const addBookModalOpen = writable(false);

/** Global loading indicator */
export const globalLoading = writable(false);

/** Sidebar open state (for mobile) */
export const sidebarOpen = writable(false);

/** View mode: 'grid' | 'list' */
export const libraryViewMode = writable(/** @type {'grid'|'list'} */ ('grid'));

// =============================================================================
// Scanner store
// =============================================================================

/** Last scanned barcode/ISBN */
export const lastScannedISBN = writable(/** @type {string|null} */ (null));

/** Whether the scanner is active */
export const scannerActive = writable(false);
