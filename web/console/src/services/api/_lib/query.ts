/**
 * Builds a `?a=1&b=2` query string from a params object, skipping
 * undefined / null / empty-string values. Centralizes the URLSearchParams loop
 * that every service used to repeat.
 */
export function buildQuery(params?: Record<string, unknown>): string {
  if (!params) return ''

  const search = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      search.append(key, String(value))
    }
  })

  const queryString = search.toString()
  return queryString ? `?${queryString}` : ''
}
