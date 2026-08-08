# Contributing to Maintainerd Auth Console

Thanks for contributing! This is the admin console (internal management UI) for
Maintainerd Auth — a React 19 + Vite + Redux Toolkit single-page app.

## DCO

All contributions are accepted under the Apache-2.0 license. You certify you
have the right to submit your contribution by including a `Signed-off-by:` line
in your commits (`git commit -s`).

## Branch & commit conventions

- Branch from `main` with a descriptive name (e.g. `feat/tenant-settings`, `fix/role-form`).
- Use [conventional commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `chore:`, `test:`, `perf:`).
- Keep commits small and focused — one logical change per commit.

## Requirements

- Node.js `>=22 <23` (see `.nvmrc`).

## Quality gates

Before opening a PR, run:

```bash
npm run lint
npm run test:cov
npm run build
```

CI blocks merge if lint, tests, or the type-check/build fail.

## Running locally

```bash
npm ci
npm run dev
```

The dev server runs on port 5173 and proxies `/api` to the internal management
auth API (`private-api.auth.maintainerd.local`) and `/public-api` to the public
auth API (`public-api.auth.maintainerd.local`) for OAuth bootstrap. For the full
stack, use the `maintainerd-dev/` Docker Compose environment.

## Code conventions

- TypeScript strict — no `any` and no `@ts-ignore`.
- Prefer the shared form components in `src/components/form` and the validation
  builders in `src/lib/validations`.
- Never surface raw `HTTP <status>` strings to users; use the client's
  user-facing error messages.

## Getting help

Open an issue or start a discussion on the GitHub repository.
