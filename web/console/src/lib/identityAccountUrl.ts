import { API_CONFIG } from "@/services/api/config"

/**
 * Builds a link into the identity app's self-service account area.
 *
 * Self-service — profile, avatar, MFA, sessions, devices, linked accounts,
 * preferences — lives ONLY in the identity app. The console used to carry its
 * own copy of each, which meant every change had to be made twice and the two
 * drifted: identity's backup-codes dialog was missing the reveal gate and the
 * download button the console had, and neither filtered MFA methods by tenant
 * policy until both were fixed separately.
 *
 * This is the split Keycloak draws between its Admin Console and its Account
 * Console, and Okta between its admin and end-user dashboards: the admin app
 * manages the tenant, and you manage yourself in one place.
 *
 * No re-authentication happens on the hop. The two apps share a registrable
 * domain, so the session cookie travels with the navigation.
 */
export function identityAccountUrl(tenantIdentityUrl: string | null | undefined, path = ""): string {
  // The per-tenant host from the tenant bootstrap is authoritative; the
  // env-derived origin is only the last-resort fallback.
  const base = (tenantIdentityUrl || API_CONFIG.IDENTITY_BASE_URL).replace(/\/$/, "")
  const suffix = path.replace(/^\//, "")
  return suffix ? `${base}/account/${suffix}` : `${base}/account`
}

/**
 * Opens the identity account area in a NEW TAB, leaving the console where it is.
 *
 * An admin is usually mid-task here; navigating away would lose that context
 * for what is a brief errand.
 *
 * noopener is not optional. Without it the opened tab receives a window.opener
 * handle back to the console and can navigate this tab elsewhere — reverse
 * tabnabbing, which on an admin console means being redirected to a
 * convincing fake of it. noreferrer additionally withholds the console URL.
 */
export function openIdentityAccount(tenantIdentityUrl: string | null | undefined, path = ""): void {
  window.open(identityAccountUrl(tenantIdentityUrl, path), "_blank", "noopener,noreferrer")
}
