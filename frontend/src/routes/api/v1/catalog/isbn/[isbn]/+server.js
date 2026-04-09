/**
 * SvelteKit server proxy for GET /api/v1/catalog/isbn/:isbn
 *
 * Forwards browser requests to the real backend with the server-side API key.
 */

import { json, error } from '@sveltejs/kit';

const BACKEND = process.env.INTERNAL_API_BASE_URL || process.env.PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1';
const API_KEY = process.env.FRONTEND_API_KEY || '';

export async function GET({ params }) {
  const { isbn } = params;
  if (!isbn) throw error(400, 'ISBN required');

  const res = await fetch(`${BACKEND}/catalog/isbn/${isbn}`, {
    headers: { 'X-API-Key': API_KEY }
  });

  const data = await res.json().catch(() => null);
  return json(data, { status: res.status });
}
