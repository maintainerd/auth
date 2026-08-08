# Performance — bundle budget

This app is a React 19 + Vite SPA. Route components are lazily loaded
(route-based code-splitting, F4), so only the shell + the visited route's chunk
load on first paint. Heavy third-party libraries are split into dedicated vendor
chunks (`react-vendor`, `data-vendor`, `ui-vendor`, `form-vendor`,
`util-vendor`) so they cache independently of app code.

## Bundle budget

**Budget: no single emitted chunk may exceed 500 KB raw (uncompressed).**

This matches Vite's default `chunkSizeWarningLimit` (500 KB). A build that emits
a chunk larger than 500 KB is a signal that a heavy dependency needs its own
manual chunk or a route needs further splitting. The 500 KB ceiling is on the
raw byte size; gzipped transfer sizes are far smaller (see below).

## Current measured chunk sizes

Measured from `npm run build` (production, no source maps). Largest chunks:

| Chunk | Raw | Gzip |
|-------|-----|------|
| `index-*.js` (app entry) | 347.6 KB | 105.2 KB |
| `ui-vendor-*.js` (Radix / icons / cmdk) | 201.3 KB | 66.5 KB |
| `data-vendor-*.js` (react-query / table / redux / axios) | 154.6 KB | 50.3 KB |
| `index-*.css` (styles) | 113.0 KB | — |
| `util-vendor-*.js` (date-fns / qrcode / clsx / tw-merge / cva) | 76.6 KB | 25.8 KB |
| `form-vendor-*.js` (react-hook-form / resolvers / yup) | 73.9 KB | 25.5 KB |
| `react-vendor-*.js` | 48.3 KB | 17.5 KB |

All other chunks are lazily-loaded route/component chunks under ~42 KB each.

**No chunk exceeds 500 KB raw** — the largest is the app entry at ~348 KB raw
(~105 KB gzip). Budget is met.

## Lighthouse

Automated Lighthouse auditing is **deferred to the upcoming Playwright/browser
test suite**, to be added by a specialist. For now the bundle budget above is the
tracked performance gate; Lighthouse can be run manually against the production
preview (`npm run build && npm run preview`, then audit `/login`).
