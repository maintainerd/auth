import { describe, it, expect, vi, beforeEach } from "vitest"

const { axiosGetMock } = vi.hoisted(() => ({ axiosGetMock: vi.fn() }))

vi.mock("axios", () => ({
  default: { get: axiosGetMock },
}))

// The control-plane client is unused by this call, but importing the module
// pulls it in — stub it so no interceptor machinery runs in the test.
vi.mock("../client", () => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  deleteRequest: vi.fn(),
}))

import { fetchSamlServiceProviderMetadata } from "./index"
import { API_CONFIG } from "../config"
import { SamlMetadataParseError } from "@/utils/samlMetadata"

const BASE = "https://identity-api.auth.maintainerd.local"

const METADATA_XML = `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="${BASE}/federation/saml/metadata/okta-sso">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol" WantAssertionsSigned="true">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="${BASE}/federation/saml/acs/okta-sso" index="1"/>
  </SPSSODescriptor>
</EntityDescriptor>`

describe("fetchSamlServiceProviderMetadata", () => {
  beforeEach(() => vi.clearAllMocks())

  it("reads the metadata from the public data plane, not the control plane", async () => {
    // The SP metadata endpoint is unauthenticated and lives on the data plane;
    // hitting the control-plane base would 404 for every tenant.
    axiosGetMock.mockResolvedValue({ data: METADATA_XML })

    await fetchSamlServiceProviderMetadata("okta-sso")

    expect(axiosGetMock).toHaveBeenCalledWith(
      `${API_CONFIG.PUBLIC_BASE_URL}/federation/saml/metadata/okta-sso`,
      expect.objectContaining({ responseType: "text" }),
    )
  })

  it("percent-encodes the identifier so it cannot escape the metadata route", async () => {
    axiosGetMock.mockResolvedValue({ data: METADATA_XML })

    await fetchSamlServiceProviderMetadata("../../clients")

    expect(axiosGetMock).toHaveBeenCalledWith(
      `${API_CONFIG.PUBLIC_BASE_URL}/federation/saml/metadata/..%2F..%2Fclients`,
      expect.anything(),
    )
  })

  it("returns the parsed values alongside the verbatim document", async () => {
    axiosGetMock.mockResolvedValue({ data: METADATA_XML })

    const result = await fetchSamlServiceProviderMetadata("okta-sso")

    expect(result.entityId).toBe(`${BASE}/federation/saml/metadata/okta-sso`)
    expect(result.acsUrl).toBe(`${BASE}/federation/saml/acs/okta-sso`)
    expect(result.xml).toBe(METADATA_XML)
  })

  it("rejects a non-metadata body instead of returning blank fields", async () => {
    // A proxy or SPA fallback can answer 200 with HTML; treating that as an
    // empty service-provider record would render a card of dashes that an
    // admin reads as "not configured yet".
    axiosGetMock.mockResolvedValue({ data: "<!doctype html><html></html>" })

    await expect(fetchSamlServiceProviderMetadata("okta-sso")).rejects.toBeInstanceOf(
      SamlMetadataParseError,
    )
  })

  it("propagates a transport failure rather than swallowing it", async () => {
    axiosGetMock.mockRejectedValue(new Error("Request failed with status code 404"))

    await expect(fetchSamlServiceProviderMetadata("okta-sso")).rejects.toThrow(/404/)
  })
})
