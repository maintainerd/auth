/**
 * SAML Service Provider metadata parsing.
 *
 * When a tenant adds a SAML 2.0 identity provider, Maintainerd is the SERVICE
 * PROVIDER. Finishing the integration requires handing three values to the
 * upstream IdP admin — the SP entity ID, the Assertion Consumer Service (ACS)
 * URL, and the SP metadata URL. Those values are composed by the backend from
 * its own public hostname, so the console must never invent them: a URL that is
 * one host or one path segment away from what the backend actually advertises
 * produces an IdP that posts assertions into the void, and the failure only
 * surfaces at the end of a user's login. Everything below is therefore read out
 * of the SP metadata document the backend itself serves.
 */

/** The SAML 2.0 metadata namespace, kept for documentation of what we parse. */
export const SAML_METADATA_NAMESPACE = 'urn:oasis:names:tc:SAML:2.0:metadata'

/**
 * Maintainerd's ACS route accepts POST only, so the HTTP-POST binding is the
 * only endpoint an IdP may be pointed at. The SP metadata also advertises an
 * HTTP-Artifact endpoint at the same location; picking whichever came first
 * would sometimes hand an admin a binding the SP cannot service.
 */
const HTTP_POST_BINDING = 'urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST'

const ACS_PATH_SEGMENT = '/saml/acs/'
const METADATA_PATH_SEGMENT = '/saml/metadata/'

export interface SamlServiceProviderMetadata {
  /** SP entity ID (`EntityDescriptor/@entityID`) — the IdP's audience value. */
  entityId: string
  /** Assertion Consumer Service URL for the HTTP-POST binding. */
  acsUrl: string
  /**
   * Absolute URL the IdP can fetch this metadata from, or `null` when it cannot
   * be established from the document. Never a fabricated value.
   */
  metadataUrl: string | null
  /** Single Logout endpoint, when the SP advertises one. */
  sloUrl: string | null
  /** NameID formats the SP requests, in document order. */
  nameIdFormats: string[]
  /** True when the SP signs its AuthnRequests and the IdP must verify them. */
  authnRequestsSigned: boolean
  /** True when the IdP must sign the assertions it returns. */
  wantAssertionsSigned: boolean
}

/** Raised when the metadata document cannot be trusted to configure an IdP. */
export class SamlMetadataParseError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'SamlMetadataParseError'
  }
}

function attr(element: Element | undefined | null, name: string): string {
  return (element?.getAttribute(name) ?? '').trim()
}

/** `xs:boolean` allows `true`/`false` and `1`/`0`; anything else is not true. */
function xmlBoolean(value: string): boolean {
  return value === 'true' || value === '1'
}

function isAbsoluteHttpUrl(value: string): boolean {
  try {
    const { protocol } = new URL(value)
    return protocol === 'https:' || protocol === 'http:'
  } catch {
    return false
  }
}

/**
 * The metadata URL is either the entity ID (Maintainerd publishes its metadata
 * endpoint as the SP entity ID) or the ACS URL with its route segment swapped —
 * both of which come from the backend's own document. When neither shape
 * matches we return `null` so the UI can say "unavailable" rather than print a
 * guess an admin would paste into a production IdP.
 */
function resolveMetadataUrl(entityId: string, acsUrl: string): string | null {
  if (isAbsoluteHttpUrl(entityId) && entityId.includes(METADATA_PATH_SEGMENT)) {
    return entityId
  }
  if (isAbsoluteHttpUrl(acsUrl)) {
    const at = acsUrl.lastIndexOf(ACS_PATH_SEGMENT)
    if (at !== -1) {
      return (
        acsUrl.slice(0, at) + METADATA_PATH_SEGMENT + acsUrl.slice(at + ACS_PATH_SEGMENT.length)
      )
    }
  }
  return null
}

/**
 * Parses the SP metadata XML served by the backend into the handful of values
 * an IdP administrator needs.
 *
 * Throws `SamlMetadataParseError` rather than returning partial data: a card
 * showing an entity ID but a blank ACS URL reads as "configure the rest by
 * hand" and is exactly how a half-configured SSO connection ships.
 */
export function parseSamlServiceProviderMetadata(xml: string): SamlServiceProviderMetadata {
  if (typeof DOMParser === 'undefined') {
    throw new SamlMetadataParseError('XML parsing is not available in this environment')
  }

  const doc = new DOMParser().parseFromString(xml, 'application/xml')

  // Browsers do not throw on malformed XML; they return a document whose body
  // is a <parsererror> element. Missing this check would silently produce an
  // "entity ID unavailable" card instead of an honest parse failure.
  if (doc.getElementsByTagNameNS('*', 'parsererror').length > 0) {
    throw new SamlMetadataParseError('Service provider metadata is not valid XML')
  }

  const root = doc.documentElement
  if (!root || root.localName !== 'EntityDescriptor') {
    throw new SamlMetadataParseError('Service provider metadata has no EntityDescriptor element')
  }

  const entityId = attr(root, 'entityID')
  if (!entityId) {
    throw new SamlMetadataParseError('Service provider metadata has no entityID')
  }

  const acsEndpoints = Array.from(doc.getElementsByTagNameNS('*', 'AssertionConsumerService'))
  const postEndpoint = acsEndpoints.find((node) => attr(node, 'Binding') === HTTP_POST_BINDING)
  const acsUrl = attr(postEndpoint, 'Location')
  if (!acsUrl) {
    throw new SamlMetadataParseError(
      'Service provider metadata has no HTTP-POST assertion consumer service',
    )
  }

  const sloUrl =
    Array.from(doc.getElementsByTagNameNS('*', 'SingleLogoutService'))
      .map((node) => attr(node, 'Location'))
      .find((location) => location !== '') ?? null

  const nameIdFormats = Array.from(doc.getElementsByTagNameNS('*', 'NameIDFormat'))
    .map((node) => (node.textContent ?? '').trim())
    .filter((format, index, all) => format !== '' && all.indexOf(format) === index)

  const descriptor = doc.getElementsByTagNameNS('*', 'SPSSODescriptor')[0]

  return {
    entityId,
    acsUrl,
    metadataUrl: resolveMetadataUrl(entityId, acsUrl),
    sloUrl,
    nameIdFormats,
    // Absent attributes mean "not signed" / "signature not required" per the
    // SAML metadata schema defaults, which is also the safe reading here: we
    // only ever tell an admin that signing IS required when the SP says so.
    authnRequestsSigned: xmlBoolean(attr(descriptor, 'AuthnRequestsSigned')),
    wantAssertionsSigned: xmlBoolean(attr(descriptor, 'WantAssertionsSigned')),
  }
}
