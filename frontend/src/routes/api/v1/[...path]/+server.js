/**
 * Catch-all proxy for /api/v1/* -> backend
 *
 * Forwards all browser API requests to the real backend with the API key.
 * The API key is only available server-side via FRONTEND_API_KEY.
 */

const BACKEND = process.env.INTERNAL_API_BASE_URL || 'http://localhost:7477/api/v1';
const API_KEY = process.env.FRONTEND_API_KEY || '';

async function proxy({ request, params }) {
  const path = params.path;
  const url = new URL(request.url);
  const backendUrl = `${BACKEND}/${path}${url.search}`;

  const headers = new Headers(request.headers);
  headers.set('X-API-Key', API_KEY);
  // Remove headers that shouldn't be forwarded
  headers.delete('host');

  const res = await fetch(backendUrl, {
    method: request.method,
    headers,
    body: ['GET', 'HEAD'].includes(request.method) ? undefined : await request.arrayBuffer()
  });

  const resHeaders = new Headers(res.headers);
  resHeaders.delete('transfer-encoding');

  return new Response(res.body, {
    status: res.status,
    headers: resHeaders
  });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
