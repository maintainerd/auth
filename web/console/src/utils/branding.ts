import { API_CONFIG } from '@/services/api/config'

/**
 * Resolves backend-relative branding logo paths to full public API URLs.
 * When the backend stores a logo via file upload, it returns a path like
 * `/public/branding/{uuid}/logo`. This must be prefixed with the public
 * base URL (stripped of /api/v1) to be usable in an <img> tag.
 */
export function resolveBrandingLogoUrl(logoUrl: string | null | undefined): string | null {
  if (!logoUrl) return null
  if (logoUrl.startsWith('/public/branding/')) {
    const origin = API_CONFIG.PUBLIC_BASE_URL.replace('/api/v1', '').replace('/public-api/api/v1', '')
    // In dev we use a proxy path, so just return the logoUrl as-is for relative paths
    if (API_CONFIG.PUBLIC_BASE_URL.startsWith('/')) {
      return logoUrl
    }
    return `${origin}${logoUrl}`
  }
  return logoUrl
}
