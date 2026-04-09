/**
 * SvelteKit server proxy for POST /api/v1/catalog/scan
 *
 * The browser-side api.js sends requests to /api/v1 (relative URL fallback).
 * This endpoint forwards those requests to the real backend with the API key,
 * which is only available server-side via process.env.FRONTEND_API_KEY.
 */

import { json, error } from '@sveltejs/kit';

const BACKEND = process.env.INTERNAL_API_BASE_URL || process.env.PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1';
const API_KEY = process.env.FRONTEND_API_KEY || '';

export async function POST({ request }) {
  let body;
  try {
    body = await request.json();
  } catch {
    throw error(400, 'Invalid JSON body');
  }

  const res = await fetch(`${BACKEND}/catalog/scan`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY
    },
    body: JSON.stringify(body)
  });

  const data = await res.json().catch(() => null);
  return json(data, { status: res.status });
}
