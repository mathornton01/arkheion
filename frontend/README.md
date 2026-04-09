# Arkheion Frontend

SvelteKit application for the Arkheion library management system.

## Stack

- **SvelteKit 2** — SSR + SPA routing
- **PDF.js** — In-browser PDF rendering
- **epub.js** — In-browser EPUB rendering
- **ZXing-js** — Barcode scanning via device camera

## Development

```bash
cp ../.env.example ../.env
cp .env.example .env

npm install
npm run dev
```

Open http://localhost:5173

## Building

```bash
npm run build
npm run preview  # Preview production build
```

## Routes

| Route            | Description                             |
|------------------|-----------------------------------------|
| `/`              | Dashboard with stats and recent books   |
| `/library`       | Book grid/list with filters             |
| `/library/[id]`  | Book detail, reader, upload             |
| `/scan`          | Barcode scanner + ISBN lookup           |
| `/search`        | Full-text search via Meilisearch        |
| `/admin`         | Webhooks, export, API reference         |

## Key Files

- `src/lib/api.js` — All API calls (never call fetch directly from components)
- `src/lib/scanner.js` — ZXing barcode scanner wrapper
- `src/lib/stores.js` — Svelte stores for global state
- `src/app.css` — Global design system
