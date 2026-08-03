/**
 * Branding API types
 *
 * Branding records are per-tenant themes. Exactly one is active (the loaded
 * style). System records (seeded default / light / dark) can be
 * edited and activated but never deleted. Theme tokens (colors, font) live in
 * the freeform `metadata` JSON; only logo/favicon/support/legal URLs and
 * company name are dedicated fields.
 */

export type BrandingLayout = "centered" | "full_page" | "split"

/** A branding theme as returned by the admin endpoints.
 *  Note: `branding_id` carries the record UUID — use it for update/activate/delete. */
export interface Branding {
  branding_id: string
  name: string
  is_system: boolean
  is_active: boolean
  layout: BrandingLayout
  company_name: string
  logo_label: string
  show_logo_label: boolean
  logo_url: string
  favicon_url: string
  support_url: string
  privacy_policy_url: string
  terms_of_service_url: string
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

/** Payload to create or update a branding theme. */
export interface BrandingRequest {
  name: string
  layout: BrandingLayout
  company_name: string
  logo_label: string
  show_logo_label: boolean
  logo_url: string
  favicon_url: string
  support_url: string
  privacy_policy_url: string
  terms_of_service_url: string
  metadata: Record<string, unknown>
  logo_data?: string         // base64-encoded; mutually exclusive with logo_url
  logo_content_type?: string // "image/png" | "image/jpeg" | "image/webp"
}
