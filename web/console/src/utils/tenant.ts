/**
 * Tenant Utilities
 *
 * The console is served per-tenant on a subdomain, and the tenant is now
 * resolved from the FULL host by the backend tenant-bootstrap endpoint — the
 * client no longer parses a tenant slug out of the hostname.
 *
 * What remains here is a minimal, TLD-agnostic host-builder used only by the
 * TenantSwitcher: switching tenants crosses a subdomain boundary, so it needs to
 * build an absolute URL to a *different* tenant's console origin. It anchors on
 * the fixed `console` label instead of the domain suffix, so it works for the
 * `.local` dev hosts and the production hosts alike.
 */

/** The fixed host label that anchors the console, shared by every tenant. */
const CONSOLE_LABEL = 'console'

/** Split a hostname (defaulting to the current one) into its DNS labels. */
function hostLabels(hostname?: string): string[] {
  const host = (hostname ?? window.location.hostname).split(':')[0]
  return host.split('.').filter(Boolean)
}

/**
 * The base console host shared by every tenant — the current host with any
 * leading tenant slug stripped. Both `acme.console.auth.x` and `console.auth.x`
 * yield `console.auth.x`.
 */
export function baseConsoleHost(hostname?: string): string {
  const labels = hostLabels(hostname)
  if (labels[1] === CONSOLE_LABEL) return labels.slice(1).join('.')
  return labels.join('.')
}

/**
 * Build an absolute URL to a (possibly different) tenant's console. Switching
 * tenants crosses a subdomain boundary, so an in-app path is not enough — we need
 * the full origin for the target tenant.
 *
 * @param slug - target tenant slug, or `null`/empty for the system tenant
 * @param path - in-console path (defaults to `/`)
 */
export function tenantConsoleUrl(slug: string | null | undefined, path = '/'): string {
  const { protocol, port } = window.location
  const base = baseConsoleHost()
  const host = slug ? `${slug}.${base}` : base
  const portSuffix = port ? `:${port}` : ''
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${protocol}//${host}${portSuffix}${normalizedPath}`
}
