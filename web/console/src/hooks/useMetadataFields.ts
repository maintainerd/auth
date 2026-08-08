import { useState, useEffect, useCallback, useMemo } from "react"

export interface MetadataField {
  id: string
  key: string
  value: string
}

// Monotonic id source — Date.now() alone can collide when two fields are
// added within the same millisecond, which would make updates hit both rows.
let fieldSeq = 0
const nextFieldId = () => `field-${++fieldSeq}`

export interface UseMetadataFieldsOptions {
  /** Keys owned by first-class form controls; using them as metadata is an error. */
  reservedKeys?: ReadonlySet<string>
  /** Error prefix shown before the offending reserved keys. */
  reservedKeysMessage?: string
  /** Also allow underscores in keys (provider-style snake_case, e.g. `cognito_region`). */
  allowUnderscore?: boolean
  /** Maximum key length (default 25). */
  maxKeyLength?: number
}

export function useMetadataFields({
  reservedKeys,
  reservedKeysMessage = "Reserved keys",
  allowUnderscore = false,
  maxKeyLength = 25,
}: UseMetadataFieldsOptions = {}) {
  const [customFields, setCustomFields] = useState<MetadataField[]>([])
  const [metadataError, setMetadataError] = useState<string>("")

  const keyPattern = useMemo(
    () => (allowUnderscore ? /^[a-z0-9_-]+$/ : /^[a-z0-9-]+$/),
    [allowUnderscore],
  )
  const charsetMessage = allowUnderscore
    ? "only lowercase letters, numbers, hyphens, and underscores allowed"
    : "only lowercase letters, numbers, and hyphens allowed"

  useEffect(() => {
    const errors: string[] = []
    const keys: string[] = []

    for (const field of customFields) {
      const key = field.key.trim()
      if (key === "") continue

      if (key.length > maxKeyLength) {
        errors.push(`"${key}" — key must not exceed ${maxKeyLength} characters`)
      } else if (!keyPattern.test(key)) {
        errors.push(`"${key}" — ${charsetMessage}`)
      }
      keys.push(key)
    }

    const usedReservedKeys = reservedKeys ? keys.filter((key) => reservedKeys.has(key)) : []
    if (usedReservedKeys.length > 0) {
      const uniqueReserved = [...new Set(usedReservedKeys)]
      errors.push(`${reservedKeysMessage}: ${uniqueReserved.join(", ")}`)
    }

    const duplicateKeys = keys.filter((key, index) => keys.indexOf(key) !== index)
    if (duplicateKeys.length > 0) {
      const uniqueDuplicates = [...new Set(duplicateKeys)]
      errors.push(`Duplicate keys: ${uniqueDuplicates.join(", ")}`)
    }

    setMetadataError(errors.length > 0 ? errors[0] : "")
  }, [customFields, keyPattern, charsetMessage, maxKeyLength, reservedKeys, reservedKeysMessage])

  const addCustomField = useCallback(() => {
    setCustomFields((prev) => [
      ...prev,
      { id: nextFieldId(), key: "", value: "" },
    ])
  }, [])

  const removeCustomField = useCallback((id: string) => {
    setCustomFields((prev) => prev.filter((field) => field.id !== id))
  }, [])

  const sanitizeKey = useCallback(
    (raw: string) => {
      const stripPattern = allowUnderscore ? /[^a-z0-9_-]/g : /[^a-z0-9-]/g
      return raw.replace(stripPattern, '').toLowerCase().slice(0, maxKeyLength)
    },
    [allowUnderscore, maxKeyLength],
  )

  const updateCustomField = useCallback(
    (id: string, key: string, value: string) => {
      setCustomFields((prev) =>
        prev.map((field) => (field.id === id ? { ...field, key: sanitizeKey(key), value } : field)),
      )
    },
    [sanitizeKey],
  )

  const buildPayload = useCallback((): Record<string, string> | undefined => {
    const metadata: Record<string, string> = {}
    customFields.forEach((field) => {
      if (field.key.trim() && field.value.trim()) {
        metadata[field.key.trim()] = field.value.trim()
      }
    })
    return Object.keys(metadata).length > 0 ? metadata : undefined
  }, [customFields])

  const resetFields = useCallback(
    (metadata?: Record<string, unknown> | null) => {
      if (metadata) {
        const fields = Object.entries(metadata).map(([key, value], index) => ({
          id: `metadata-${index}`,
          key,
          // Objects would render as "[object Object]" through String(); keep
          // their JSON shape so a round-trip edit doesn't corrupt them.
          value: typeof value === "object" && value !== null ? JSON.stringify(value) : String(value),
        }))
        setCustomFields(fields)
      } else {
        setCustomFields([])
      }
      setMetadataError("")
    },
    [],
  )

  return {
    customFields,
    metadataError,
    addCustomField,
    removeCustomField,
    updateCustomField,
    buildPayload,
    resetFields,
  }
}
