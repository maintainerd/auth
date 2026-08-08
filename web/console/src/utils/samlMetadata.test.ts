import { describe, it, expect } from "vitest"
import {
  parseSamlServiceProviderMetadata,
  SamlMetadataParseError,
} from "./samlMetadata"

const BASE = "https://identity-api.auth.maintainerd.local"

/**
 * Mirrors what the backend actually serves: crewjam/saml marshals the SP
 * EntityDescriptor with a default xmlns, two AssertionConsumerService entries
 * (HTTP-POST index 1, HTTP-Artifact index 2) pointing at the same location, and
 * WantAssertionsSigned always true.
 */
function samlSpMetadata(
  options: {
    entityId?: string
    acsUrl?: string
    nameIdFormat?: string
    sloUrl?: string
    authnRequestsSigned?: boolean
  } = {},
): string {
  const {
    entityId = `${BASE}/federation/saml/metadata/okta-sso`,
    acsUrl = `${BASE}/federation/saml/acs/okta-sso`,
    nameIdFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    sloUrl,
    authnRequestsSigned = false,
  } = options

  return `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="${entityId}" validUntil="2026-08-06T00:00:00Z" cacheDuration="PT24H0M0S">
  <SPSSODescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" validUntil="2026-08-06T00:00:00Z" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol" AuthnRequestsSigned="${authnRequestsSigned}" WantAssertionsSigned="true">
    ${sloUrl ? `<SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="${sloUrl}"></SingleLogoutService>` : ""}
    <NameIDFormat>${nameIdFormat}</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="${acsUrl}" index="1"></AssertionConsumerService>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact" Location="${acsUrl}" index="2"></AssertionConsumerService>
  </SPSSODescriptor>
</EntityDescriptor>`
}

describe("parseSamlServiceProviderMetadata", () => {
  it("extracts the entity ID, ACS URL and metadata URL an IdP admin needs", () => {
    const result = parseSamlServiceProviderMetadata(samlSpMetadata())

    expect(result.entityId).toBe(`${BASE}/federation/saml/metadata/okta-sso`)
    expect(result.acsUrl).toBe(`${BASE}/federation/saml/acs/okta-sso`)
    expect(result.metadataUrl).toBe(`${BASE}/federation/saml/metadata/okta-sso`)
    expect(result.nameIdFormats).toEqual([
      "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    ])
    expect(result.wantAssertionsSigned).toBe(true)
    expect(result.authnRequestsSigned).toBe(false)
    expect(result.sloUrl).toBeNull()
  })

  it("picks the HTTP-POST binding, not whichever endpoint came first", () => {
    // The SP route accepts POST only, so an admin handed the HTTP-Artifact
    // endpoint would configure a flow Maintainerd cannot service.
    const xml = samlSpMetadata().replace(
      '<AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="' +
        `${BASE}/federation/saml/acs/okta-sso" index="1"></AssertionConsumerService>`,
      '<AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="' +
        `${BASE}/federation/saml/acs/post-only" index="3"></AssertionConsumerService>`,
    )

    expect(parseSamlServiceProviderMetadata(xml).acsUrl).toBe(
      `${BASE}/federation/saml/acs/post-only`,
    )
  })

  it("throws when no HTTP-POST assertion consumer service is advertised", () => {
    const xml = samlSpMetadata().replace(
      "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
      "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
    )

    expect(() => parseSamlServiceProviderMetadata(xml)).toThrow(SamlMetadataParseError)
  })

  it("derives the metadata URL from the ACS URL when the entity ID is a bare URN", () => {
    // Both values come from the backend's own document; nothing is invented.
    const result = parseSamlServiceProviderMetadata(
      samlSpMetadata({ entityId: "urn:maintainerd:sp:okta-sso" }),
    )

    expect(result.entityId).toBe("urn:maintainerd:sp:okta-sso")
    expect(result.metadataUrl).toBe(`${BASE}/federation/saml/metadata/okta-sso`)
  })

  it("reports the metadata URL as unavailable rather than fabricating one", () => {
    const result = parseSamlServiceProviderMetadata(
      samlSpMetadata({
        entityId: "urn:maintainerd:sp:okta-sso",
        acsUrl: "https://sp.example.com/custom/consume",
      }),
    )

    expect(result.metadataUrl).toBeNull()
  })

  it("surfaces the Single Logout endpoint when the SP publishes one", () => {
    const result = parseSamlServiceProviderMetadata(
      samlSpMetadata({ sloUrl: `${BASE}/federation/saml/slo/okta-sso` }),
    )

    expect(result.sloUrl).toBe(`${BASE}/federation/saml/slo/okta-sso`)
  })

  it("reads AuthnRequestsSigned when the SP signs its requests", () => {
    const result = parseSamlServiceProviderMetadata(samlSpMetadata({ authnRequestsSigned: true }))

    expect(result.authnRequestsSigned).toBe(true)
  })

  it("treats an absent signature attribute as not required", () => {
    const xml = samlSpMetadata().replace(' WantAssertionsSigned="true"', "")

    expect(parseSamlServiceProviderMetadata(xml).wantAssertionsSigned).toBe(false)
  })

  it("rejects malformed XML instead of returning an empty record", () => {
    expect(() => parseSamlServiceProviderMetadata("<EntityDescriptor")).toThrow(
      SamlMetadataParseError,
    )
  })

  it("rejects a well-formed document that is not an EntityDescriptor", () => {
    // An HTML error page proxied back as 200 must not read as "no values yet".
    expect(() => parseSamlServiceProviderMetadata("<error>not found</error>")).toThrow(
      SamlMetadataParseError,
    )
  })

  it("rejects an EntityDescriptor with no entityID", () => {
    const xml = samlSpMetadata().replace(
      `entityID="${BASE}/federation/saml/metadata/okta-sso" `,
      "",
    )

    expect(() => parseSamlServiceProviderMetadata(xml)).toThrow(SamlMetadataParseError)
  })

  it("parses metadata that uses a namespace prefix", () => {
    // Not every IdP-facing serializer emits a default xmlns; the parser must be
    // namespace-agnostic or a prefixed document would look like it had no ACS.
    const xml = `<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="${BASE}/federation/saml/metadata/prefixed">
  <md:SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol" WantAssertionsSigned="true">
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="${BASE}/federation/saml/acs/prefixed" index="1"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>`

    const result = parseSamlServiceProviderMetadata(xml)

    expect(result.acsUrl).toBe(`${BASE}/federation/saml/acs/prefixed`)
    expect(result.wantAssertionsSigned).toBe(true)
  })
})
