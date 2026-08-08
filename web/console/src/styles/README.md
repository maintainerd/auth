# Style And Theme Organization

`index.css` owns global Tailwind/shadcn setup, base CSS variables, dark-mode
fallback values, and browser-level defaults such as autofill and native color
input rendering.

`console-theme.css` owns runtime branding only. Keep selectors scoped under
`html[data-console-theme="active"]` and group them by surface:

- app shell and guard/auth pages
- top panel and side navigation
- buttons and page actions
- tables, cards, and notices
- forms, overlays, toggles, and option groups
- listing cards, icon containers, and status badges

`toast.css` is intentionally separate because Sonner exposes its own CSS
variables and is not part of the component token map.

When adding a themeable component, prefer a stable `data-md-*` or
`data-console-*` attribute on the component over styling Tailwind utility
classes directly. Then add the token defaults in `themeTokens.ts`, map them in
`consoleTheme.ts`, and wire backend seed defaults so new tenants receive the same
configurable surface.
