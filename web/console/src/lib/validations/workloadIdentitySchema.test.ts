import { describe, it, expect } from "vitest"
import {
  workloadIdentitySchema,
  validateIssuerUrl,
  validateSubjectPattern,
  validateAttributeMapping,
  validateAllowedScopes,
  parseAllowedScopes,
  isReservedClaimName,
  MAX_ATTRIBUTE_MAPPING_ENTRIES,
} from "./workloadIdentitySchema"

const validForm = {
  client_uuid: "00000000-0000-0000-0000-0000000000cc",
  name: "github-actions",
  description: "",
  issuer_url: "https://token.actions.githubusercontent.com",
  audience: "https://auth.example.com",
  subject_claim: "sub",
  subject_pattern: "repo:my-org/my-repo:*",
  allowed_scopes: "api:read",
  is_active: true,
}

describe("workloadIdentitySchema", () => {
  it("accepts a valid federation", async () => {
    await expect(workloadIdentitySchema.validate(validForm)).resolves.toBeTruthy()
  })

  it("requires the fields the backend requires", async () => {
    for (const field of ["client_uuid", "name", "issuer_url", "audience", "subject_pattern"]) {
      await expect(
        workloadIdentitySchema.validate({ ...validForm, [field]: "" }),
      ).rejects.toThrow()
    }
  })

  it("enforces the backend length limits", async () => {
    await expect(
      workloadIdentitySchema.validate({ ...validForm, name: "a".repeat(101) }),
    ).rejects.toThrow(/100/)
    await expect(
      workloadIdentitySchema.validate({ ...validForm, audience: "a".repeat(513) }),
    ).rejects.toThrow(/512/)
    await expect(
      workloadIdentitySchema.validate({ ...validForm, subject_claim: "a".repeat(101) }),
    ).rejects.toThrow(/100/)
  })
})

// The backend requires https and rejects a trailing slash; yup's `.url()` accepted
// http:// and everything else, so the console used to send values the server refused.
describe("validateIssuerUrl", () => {
  it("accepts an https issuer", () => {
    expect(validateIssuerUrl("https://token.actions.githubusercontent.com")).toBeNull()
  })

  it("rejects http, including loopback", () => {
    expect(validateIssuerUrl("http://token.actions.githubusercontent.com")).toMatch(/https/i)
    // The shared isHttpsUrl helper permits http://localhost; this backend rule does not.
    expect(validateIssuerUrl("http://localhost:8080")).toMatch(/https/i)
  })

  it("rejects a non-absolute or non-URL value", () => {
    expect(validateIssuerUrl("token.actions.githubusercontent.com")).toMatch(/absolute/i)
    expect(validateIssuerUrl("not a url")).toMatch(/absolute/i)
  })

  // OIDC discovery compares the issuer byte-for-byte, so a trailing slash would
  // produce a federation that can never match.
  it("rejects a trailing slash", () => {
    expect(validateIssuerUrl("https://token.actions.githubusercontent.com/")).toMatch(/slash/i)
  })

  it("rejects an over-long issuer", () => {
    expect(validateIssuerUrl(`https://example.com/${"a".repeat(520)}`)).toMatch(/512/)
  })
})

// subject_pattern is the ONLY thing separating this tenant's workloads from everyone
// else's on a shared public issuer, so breadth is a security rule, not a style rule.
describe("validateSubjectPattern", () => {
  it("rejects a bare wildcard", () => {
    expect(validateSubjectPattern("*")).toMatch(/must not start with a wildcard/i)
  })

  it("rejects a leading wildcard", () => {
    expect(validateSubjectPattern("*:ref:refs/heads/main")).toMatch(/wildcard/i)
    expect(validateSubjectPattern("?repo:org/x")).toMatch(/wildcard/i)
  })

  it("rejects a pattern with too little literal text", () => {
    expect(validateSubjectPattern("rep*")).toMatch(/too broad/i)
  })

  it("accepts an anchored wildcard pattern", () => {
    expect(validateSubjectPattern("repo:my-org/my-repo:*")).toBeNull()
    expect(validateSubjectPattern("system:serviceaccount:production:*")).toBeNull()
  })

  // An exact pattern cannot over-match, however short.
  it("accepts a short literal pattern", () => {
    expect(validateSubjectPattern("ci")).toBeNull()
  })
})

// The map's VALUES are destination claim names in the issued token. ExtraClaims is
// merged last over the standard claims, so a reserved value forges the token's
// identity on an endpoint that takes no client credentials.
describe("validateAttributeMapping", () => {
  it("accepts an empty mapping", () => {
    expect(validateAttributeMapping({})).toBeNull()
  })

  it("accepts a sane mapping, including a nested source path", () => {
    expect(
      validateAttributeMapping({
        repository: "repository",
        "github.workflow": "workflow_name",
      }),
    ).toBeNull()
  })

  it("rejects a reserved destination claim", () => {
    for (const reserved of ["sub", "client_id", "svc", "tenant_id", "permissions", "exp", "act"]) {
      expect(validateAttributeMapping({ repository: reserved })).toMatch(/cannot be overridden/i)
    }
  })

  it("rejects a reserved destination regardless of case", () => {
    expect(validateAttributeMapping({ repository: "SUB" })).toMatch(/cannot be overridden/i)
  })

  it("rejects a malformed destination claim name", () => {
    for (const bad of ["Has-Caps", "has space", "1leading", "has.dot"]) {
      expect(validateAttributeMapping({ repository: bad })).toMatch(/not a valid claim name/i)
    }
  })

  it("rejects an empty source or destination", () => {
    expect(validateAttributeMapping({ "": "repository" })).toMatch(/must not be empty/i)
    expect(validateAttributeMapping({ repository: "" })).toMatch(/no target claim name/i)
  })

  it("bounds the number of entries", () => {
    const mapping: Record<string, string> = {}
    for (let i = 0; i <= MAX_ATTRIBUTE_MAPPING_ENTRIES; i++) {
      mapping[`external_${i}`] = `internal_${i}`
    }
    expect(validateAttributeMapping(mapping)).toMatch(/more than/i)
  })
})

describe("isReservedClaimName", () => {
  it("covers the claims the gRPC and HTTP surfaces authorize on", () => {
    for (const claim of ["sub", "client_id", "svc", "tenant_id", "permissions", "roles", "cnf"]) {
      expect(isReservedClaimName(claim)).toBe(true)
    }
    expect(isReservedClaimName("repository")).toBe(false)
    expect(isReservedClaimName("workflow_name")).toBe(false)
  })
})

describe("scopes", () => {
  it("splits on commas and whitespace, dropping blanks", () => {
    expect(parseAllowedScopes("api:read, api:write   api:admin,")).toEqual([
      "api:read",
      "api:write",
      "api:admin",
    ])
    expect(parseAllowedScopes("")).toEqual([])
    expect(parseAllowedScopes(null)).toEqual([])
  })

  it("rejects an over-long scope", () => {
    expect(validateAllowedScopes(["a".repeat(129)])).toMatch(/128/)
    expect(validateAllowedScopes(["api:read"])).toBeNull()
  })
})
