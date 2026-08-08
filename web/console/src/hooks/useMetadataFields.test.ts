import { describe, it, expect } from "vitest"
import { act, renderHook } from "@testing-library/react"
import { useMetadataFields } from "./useMetadataFields"

function addField(result: { current: ReturnType<typeof useMetadataFields> }, key: string, value: string) {
  act(() => result.current.addCustomField())
  const field = result.current.customFields[result.current.customFields.length - 1]
  act(() => result.current.updateCustomField(field.id, key, value))
}

describe("useMetadataFields", () => {
  it("sanitizes keys to kebab-case and caps at 25 chars by default", () => {
    const { result } = renderHook(() => useMetadataFields())
    addField(result, "team_name-1!", "Engineering")
    expect(result.current.customFields[0].key).toBe("teamname-1")

    addField(result, "a".repeat(40), "x")
    expect(result.current.customFields[1].key).toBe("a".repeat(25))
  })

  it("keeps underscores and honours maxKeyLength when configured", () => {
    const { result } = renderHook(() =>
      useMetadataFields({ allowUnderscore: true, maxKeyLength: 50 }),
    )
    addField(result, "cognito_region", "us-east-1")
    expect(result.current.customFields[0].key).toBe("cognito_region")
    expect(result.current.metadataError).toBe("")
  })

  it("flags reserved keys with the configured message", () => {
    const { result } = renderHook(() =>
      useMetadataFields({
        allowUnderscore: true,
        maxKeyLength: 50,
        reservedKeys: new Set(["grant_types"]),
        reservedKeysMessage: "Move standard client configuration to its own controls",
      }),
    )
    addField(result, "grant_types", "authorization_code")
    expect(result.current.metadataError).toBe(
      "Move standard client configuration to its own controls: grant_types",
    )
  })

  it("flags duplicate keys", () => {
    const { result } = renderHook(() => useMetadataFields())
    addField(result, "team", "a")
    addField(result, "team", "b")
    expect(result.current.metadataError).toBe("Duplicate keys: team")
  })

  it("builds a payload from non-empty fields only", () => {
    const { result } = renderHook(() => useMetadataFields())
    addField(result, "team", "core")
    addField(result, "empty", "")
    expect(result.current.buildPayload()).toEqual({ team: "core" })
  })

  it("returns undefined payload when nothing is set", () => {
    const { result } = renderHook(() => useMetadataFields())
    expect(result.current.buildPayload()).toBeUndefined()
  })

  it("resets fields from metadata, JSON-encoding object values", () => {
    const { result } = renderHook(() => useMetadataFields())
    act(() => result.current.resetFields({ tier: "gold", nested: { a: 1 } }))
    expect(result.current.customFields).toHaveLength(2)
    expect(result.current.customFields[0]).toMatchObject({ key: "tier", value: "gold" })
    expect(result.current.customFields[1]).toMatchObject({ key: "nested", value: '{"a":1}' })

    act(() => result.current.resetFields(null))
    expect(result.current.customFields).toHaveLength(0)
  })
})
