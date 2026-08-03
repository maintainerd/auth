const CLIENT_STORAGE_KEY = 'maintainerd_auth_client_id'

export interface PublicAuthContext {
  clientId?: string
  tenantId?: string
}

/**
 * Snapshot of the domain-bootstrap result the app needs to attach tenant/client
 * context to public API calls. It is populated once the tenant bootstrap
 * resolves (see store/tenant) and read synchronously here, so the service layer
 * can resolve the right context without threading React/Redux state through
 * every call.
 *
 *  - `tenantSlug`      — the resolved tenant's `name` (slug), sent as `tenant_id`.
 *  - `defaultClientId` — the tenant's default identity client for this surface,
 *                        used for first-party direct navigation so the caller
 *                        need not supply any client_id/tenant_id.
 */
interface TenantBootstrapContext {
  tenantSlug?: string
  defaultClientId?: string
}

let bootstrapContext: TenantBootstrapContext = {}

/** Populate the bootstrap context once the domain bootstrap has resolved. */
export function setTenantBootstrapContext(context: TenantBootstrapContext): void {
  bootstrapContext = {
    tenantSlug: context.tenantSlug?.trim() || undefined,
    defaultClientId: context.defaultClientId?.trim() || undefined,
  }
}

/** Clear the bootstrap context (e.g. on logout / teardown). */
export function clearTenantBootstrapContext(): void {
  bootstrapContext = {}
}

function trimParam(params: URLSearchParams, key: string): string | undefined {
  return params.get(key)?.trim() || undefined
}

function storedClientId(): string | undefined {
  return sessionStorage.getItem(CLIENT_STORAGE_KEY)?.trim() || undefined
}

/**
 * The tenant is resolved by the backend from the full host (see the domain
 * bootstrap). Returns the tenant slug (`name`) from that response, or undefined
 * before the bootstrap has resolved / for the system tenant. A user-supplied
 * `tenant_id` query param is deliberately ignored so a caller can never point
 * the app at another tenant.
 */
function bootstrapTenantId(): string | undefined {
  return bootstrapContext.tenantSlug
}

/**
 * Persist the OAuth `client_id` from the current URL (when present) and return
 * the active public-auth context. Only an explicit `client_id` is read from /
 * remembered across the query string; the tenant comes from the domain
 * bootstrap. The tenant's default identity client is NOT baked into `clientId`
 * here — it is applied as a fallback in resolvePublicAuthContext() so the
 * branding overlay and connection lookups only react to an explicit client.
 */
export function rememberPublicAuthContext(search: string): PublicAuthContext {
  const params = new URLSearchParams(search)
  const clientId = trimParam(params, 'client_id')
  if (clientId) {
    sessionStorage.setItem(CLIENT_STORAGE_KEY, clientId)
  }
  return {
    clientId: clientId ?? storedClientId(),
    tenantId: bootstrapTenantId(),
  }
}

export function rememberClientId(search: string): string | null {
  return rememberPublicAuthContext(search).clientId ?? null
}

export function currentPublicAuthContext(): PublicAuthContext {
  return rememberPublicAuthContext(typeof window !== 'undefined' ? window.location.search : '')
}

/**
 * Resolve the single context value the backend should receive, in priority order:
 *   1. an explicit OAuth `client_id` (external-client / persisted flow) wins;
 *   2. otherwise the tenant's default identity `client_id` from the domain
 *      bootstrap — the client this surface *is*;
 *   3. otherwise a caller-supplied tenant slug as `tenant_id`;
 *   4. otherwise nothing — the system tenant, where the backend defaults.
 *
 * Step 2 applies even when the caller passed a `tenantId`. Every public auth
 * endpoint is client-scoped: it requires `client_id` and rejects `tenant_id`
 * outright ("Public registration requires client_id and does not accept
 * tenant_id"). So preferring a caller's tenant slug over the known surface
 * client cannot produce a working request — only a 400. Callers that read the
 * tenant slug off the bootstrap (register, magic link, MFA continuation) were
 * doing exactly that on any first-party visit with no `client_id` in the URL.
 *
 * A `client_id` and a `tenant_id` are never sent together.
 */
export function resolvePublicAuthContext(context?: PublicAuthContext): PublicAuthContext {
  const ctx = context ?? currentPublicAuthContext()

  const clientId = ctx.clientId?.trim() || undefined
  if (clientId) return { clientId }

  if (bootstrapContext.defaultClientId) {
    return { clientId: bootstrapContext.defaultClientId }
  }

  const tenantId = ctx.tenantId?.trim() || undefined
  return tenantId ? { tenantId } : {}
}

export function appendPublicAuthContext(params: URLSearchParams, context?: PublicAuthContext): URLSearchParams {
  const hasExplicit = Boolean(context?.clientId || context?.tenantId)
  const resolved = hasExplicit ? resolvePublicAuthContext(context) : resolvePublicAuthContext()
  if (resolved.clientId) params.set('client_id', resolved.clientId)
  else if (resolved.tenantId) params.set('tenant_id', resolved.tenantId)
  return params
}

export function publicAuthQuery(context?: PublicAuthContext): string {
  return appendPublicAuthContext(new URLSearchParams(), context).toString()
}

export function requireClientId(): string {
  const clientId = resolvePublicAuthContext().clientId
  if (!clientId) throw new Error('client_id is required')
  return clientId
}
