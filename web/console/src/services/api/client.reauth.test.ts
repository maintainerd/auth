import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'

/**
 * Regression cover for the cold-start redirect loop.
 *
 * Visiting the console logged-out made the URL flicker between / and /login a
 * dozen-plus times before settling. Bootstrap issues two calls in parallel — the
 * tenant lookup (200) and the session probe (401) — and the interceptor acted on
 * the 401 immediately, reloading the page. Any 200 cleared the reload guard, so
 * whichever response landed last decided whether the guard survived; when it did
 * not, the reload repeated.
 *
 * The rule these tests pin down: nothing navigates until bootstrap says the
 * session verdict is in.
 */
describe('reauthentication gating', () => {
  const replace = vi.fn()
  const reload = vi.fn()

  beforeEach(() => {
    vi.resetModules()
    vi.stubGlobal('location', {
      pathname: '/dashboard',
      replace,
      reload,
      origin: 'https://console.test',
      href: 'https://console.test/dashboard',
    })
    window.sessionStorage.clear()
    replace.mockClear()
    reload.mockClear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('exports a bootstrap-settled signal for AppBootstrap to call', async () => {
    const mod = await import('./client')
    expect(typeof mod.setBootstrapSettled).toBe('function')
  })

  it('treats the landing route as exact — "/" must not prefix-match every path', async () => {
    const src = (await import('fs')).readFileSync('src/services/api/client.ts', 'utf-8')
    // '/' inside a startsWith() list would exempt the whole app and silently
    // disable re-authentication everywhere.
    const listMatch = src.match(/const UNAUTHENTICATED_ROUTES = \[(.*?)\]/s)
    expect(listMatch).toBeTruthy()
    expect(listMatch![1]).not.toMatch(/(^|[^/\w])'\/'/)
    expect(src).toContain('const LANDING_ROUTE')
  })

  it('does not clear the reload guard on an arbitrary successful response', async () => {
    const src = (await import('fs')).readFileSync('src/services/api/client.ts', 'utf-8')
    const successHandler = src.slice(
      src.indexOf('axiosInstance.interceptors.response.use'),
      src.indexOf('async (error: AxiosError)'),
    )
    // A parallel 200 wiping the guard set by a parallel 401 is precisely the race
    // that made the reload unbounded.
    expect(successHandler).not.toContain('removeItem')
  })

  it('only clears the reload guard on an authenticated bootstrap', async () => {
    const mod = await import('./client')
    window.sessionStorage.setItem('maintainerd.console.reauth-attempted', '/dashboard')

    mod.setBootstrapSettled(false)
    expect(window.sessionStorage.getItem('maintainerd.console.reauth-attempted')).toBe('/dashboard')

    mod.setBootstrapSettled(true)
    expect(window.sessionStorage.getItem('maintainerd.console.reauth-attempted')).toBeNull()
  })

  it('exempts the session probe so a logged-out boot is not a recovery attempt', async () => {
    const src = (await import('fs')).readFileSync('src/services/api/client.ts', 'utf-8')
    const list = src.slice(src.indexOf('const NO_REAUTH_ENDPOINTS'), src.indexOf('// Routes that are allowed'))
    expect(list).toContain('AUTH.ACCOUNT')
  })

  it('gates every navigation behind the bootstrap verdict', async () => {
    const src = (await import('fs')).readFileSync('src/services/api/client.ts', 'utf-8')
    const fn = src.slice(src.indexOf('function reauthenticate()'), src.indexOf('type RetriableRequestConfig'))
    // The guard must be the FIRST thing checked, before any navigation path.
    expect(fn.indexOf('bootstrapSettled')).toBeGreaterThan(-1)
    expect(fn.indexOf('bootstrapSettled')).toBeLessThan(fn.indexOf('location.reload'))
    expect(fn.indexOf('bootstrapSettled')).toBeLessThan(fn.indexOf('location.replace'))
  })
})
