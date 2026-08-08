/**
 * useProviderConfig
 *
 * Owns the editable state behind a provider's `config` JSON. It keeps the
 * provider's well-known fields (scopes, OAuth2 endpoints) in sync as the
 * operator switches providers and recombines them into a single object on save.
 *
 * The backend validates config strictly and REJECTS unknown keys, so this hook
 * only ever reads and emits the keys declared by the provider's config schema.
 * Any stray key found in a stored config is dropped on load rather than
 * surfaced for editing.
 *
 * Field values are held as display strings regardless of their JSON type; the
 * field's declared `type` decides how they are parsed back out in `buildConfig`:
 *   - list   → comma/newline separated string ⇄ string[]
 *   - switch → "true"/"false" ⇄ boolean
 *   - password → write-only config fields, if a provider ever declares one.
 *
 * Shared broker connection fields such as issuer/client ID/secret/JIT/domains
 * are top-level provider columns and are handled by the identity provider form,
 * not by this config JSON controller.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { SCOPE_TOKEN_REGEX, isHttpsUrl, isPemCertificate } from "@/lib/validations/regex"
import {
  getProviderConfigSchema,
  getProviderFields,
  isOAuth2OnlyProvider,
  isAllowedProviderHost,
  type ProviderConfigField,
  type ProviderConfigSchema,
} from "./providerConfigSchemas"

/**
 * Parse a whitespace/comma/newline separated string into a trimmed, non-empty
 * list. The separator set (/[\s,\n]+/) is kept identical to the form's submit
 * tokenizer and the yup emailDomains tokenizer so what validates is exactly
 * what is sent to the backend.
 */
function parseList(value: string): string[] {
  return value
    .split(/[\s,\n]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function stringifyValue(value: unknown): string {
  if (value === null || value === undefined) return ""
  if (Array.isArray(value)) return value.map(String).join(", ")
  if (typeof value === "object") return JSON.stringify(value)
  return String(value)
}

function fieldByKey(provider: string): Map<string, ProviderConfigField> {
  return new Map(getProviderFields(provider).map((field) => [field.key, field]))
}

/** Seed values for fields that declare a `default` (used on create). */
function seedDefaults(schema: ProviderConfigSchema | undefined): Record<string, string> {
  const seeded: Record<string, string> = {}
  schema?.groups.forEach((group) =>
    group.fields.forEach((field) => {
      if (field.default !== undefined) seeded[field.key] = field.default
    })
  )
  return seeded
}

interface SplitResult {
  known: Record<string, string>
  /** Password keys that already have a stored secret (shown as "leave blank to keep"). */
  presetSecrets: Record<string, boolean>
}

/**
 * Split a raw config object into the provider's known fields. Keys the provider
 * does not declare are dropped — the backend rejects unknown keys, so there is
 * nothing useful to do with them.
 */
function splitConfig(config: Record<string, unknown> | null | undefined, provider: string): SplitResult {
  const fields = fieldByKey(provider)
  const known: Record<string, string> = {}
  const presetSecrets: Record<string, boolean> = {}

  Object.entries(config || {}).forEach(([key, value]) => {
    const field = fields.get(key)
    if (!field) return

    if (field.type === "password") {
      // Write-only: never surface the stored/redacted secret in the input.
      known[key] = ""
      presetSecrets[key] = stringifyValue(value).trim() !== ""
      return
    }

    if (field.type === "switch") {
      known[key] = value === true || value === "true" ? "true" : "false"
      return
    }

    known[key] = stringifyValue(value)
  })

  return { known, presetSecrets }
}

export function useProviderConfig(provider: string) {
  const schema = useMemo(() => getProviderConfigSchema(provider), [provider])

  // On create there is no stored config, so pre-fill fields that declare a
  // default (e.g. scopes, OAuth2 endpoints). On edit, load() replaces this with
  // the saved config.
  const [values, setValues] = useState<Record<string, string>>(() => seedDefaults(getProviderConfigSchema(provider)))
  const [presetSecrets, setPresetSecrets] = useState<Record<string, boolean>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})

  const prevProviderRef = useRef(provider)

  /** Hydrate from an existing config (edit mode). */
  const load = useCallback(
    (config: Record<string, unknown> | null | undefined, providerForSplit: string = provider) => {
      const { known, presetSecrets: secrets } = splitConfig(config, providerForSplit)
      setValues(known)
      setPresetSecrets(secrets)
      setErrors({})
      prevProviderRef.current = providerForSplit
    },
    [provider]
  )

  // When the provider changes, re-bucket the current values under the new
  // provider's field set. Shared structured keys (scopes, endpoint overrides, …)
  // carry over between providers that both define them; the previous provider's
  // keys that the new one does NOT define are dropped — so e.g. switching to the
  // built-in provider (which has no config) doesn't leave stray scopes behind.
  useEffect(() => {
    if (prevProviderRef.current === provider) return
    prevProviderRef.current = provider

    const fields = fieldByKey(provider)

    setValues((prevValues) => {
      const nextKnown: Record<string, string> = {}
      // Carry over prior structured values only for keys the new provider defines.
      Object.entries(prevValues).forEach(([key, value]) => {
        if (fields.has(key)) nextKnown[key] = value
      })
      // Seed defaults for the new provider's fields that ended up empty.
      fields.forEach((field, key) => {
        if (field.default !== undefined && !(nextKnown[key] ?? "").trim()) {
          nextKnown[key] = field.default
        }
      })
      return nextKnown
    })

    setPresetSecrets({})
    setErrors({})
  }, [provider])

  const setFieldValue = useCallback((key: string, value: string) => {
    setValues((prev) => ({ ...prev, [key]: value }))
    setErrors((prev) => (prev[key] ? { ...prev, [key]: "" } : prev))
  }, [])

  /**
   * Validate the provider's config fields.
   *
   * FORMAT checks (scope-token shape, https:// URLs, PEM certificate shape) run
   * whenever a value is present, regardless of status — rubbish must never reach
   * the backend. PRESENCE ("required") checks only run when the provider is
   * being saved active, mirroring the backend, which only requires config fields
   * when status === "active". A draft (inactive) provider may be saved with the
   * required fields left blank.
   */
  const validate = useCallback(
    (status: string = "active"): boolean => {
      const nextErrors: Record<string, string> = {}
      const requirePresence = status === "active"

      const requireIfActive = (key: string, label: string, empty: boolean) => {
        if (requirePresence && empty) nextErrors[key] = `${label} is required`
      }

      if (schema) {
        schema.groups.forEach((group) =>
          group.fields.forEach((field) => {
            const raw = values[field.key] ?? ""
            const value = raw.trim()

            switch (field.type) {
              case "scopes": {
                // Format-check every token (even when optional); required only
                // when active.
                const scopes = parseList(raw)
                const invalid = scopes.filter((s) => !SCOPE_TOKEN_REGEX.test(s))
                if (invalid.length > 0) {
                  nextErrors[field.key] = `Invalid scope${invalid.length > 1 ? "s" : ""}: ${invalid.join(", ")}`
                } else if (field.required) {
                  requireIfActive(field.key, field.label, scopes.length === 0)
                }
                break
              }
              case "url": {
                // HTTPS enforcement on every URL field (OAuth2-only endpoints,
                // SAML sso_url/slo_url), then — for OAuth2-only providers whose
                // endpoints have fixed official hosts (github/facebook/x) — bind
                // the host so a bogus endpoint can't exfiltrate the client secret.
                // Mirrors backend requireHTTPSURL + requireAllowedHost.
                if (value !== "" && !isHttpsUrl(value)) {
                  nextErrors[field.key] = `${field.label} must be a valid https:// URL`
                } else if (value !== "" && isOAuth2OnlyProvider(provider) && !isAllowedProviderHost(provider, value)) {
                  nextErrors[field.key] = `${field.label} host is not permitted for the ${provider} provider`
                } else if (field.required) {
                  requireIfActive(field.key, field.label, value === "")
                }
                break
              }
              case "textarea": {
                // SAML certificate: shape-check for a PEM X.509 block when set.
                if (field.key === "certificate" && value !== "" && !isPemCertificate(value)) {
                  nextErrors[field.key] = `${field.label} must be a PEM-encoded X.509 certificate`
                } else if (field.required) {
                  requireIfActive(field.key, field.label, value === "")
                }
                break
              }
              case "password": {
                // Satisfied by a newly typed secret or an already-stored one.
                if (field.required) {
                  requireIfActive(field.key, field.label, value === "" && !presetSecrets[field.key])
                }
                break
              }
              case "list": {
                if (field.required) {
                  requireIfActive(field.key, field.label, parseList(raw).length === 0)
                }
                break
              }
              case "switch":
                break
              default: {
                if (field.required) {
                  requireIfActive(field.key, field.label, value === "")
                }
              }
            }
          })
        )
      }
      setErrors(nextErrors)

      return Object.keys(nextErrors).length === 0
    },
    [schema, values, presetSecrets, provider]
  )

  /** Build the config object to persist from the provider's known fields only. */
  const buildConfig = useCallback((): Record<string, unknown> => {
    const config: Record<string, unknown> = {}

    if (schema) {
      schema.groups.forEach((group) =>
        group.fields.forEach((field) => {
          const raw = values[field.key] ?? ""

          switch (field.type) {
            case "password": {
              // Write-only: only send when a new secret was typed; otherwise the
              // backend preserves the stored value.
              const trimmed = raw.trim()
              if (trimmed !== "") config[field.key] = trimmed
              break
            }
            case "list":
            case "scopes": {
              const list = parseList(raw)
              if (list.length > 0) config[field.key] = list
              break
            }
            case "switch": {
              config[field.key] = raw === "true"
              break
            }
            default: {
              const trimmed = raw.trim()
              if (trimmed !== "") config[field.key] = trimmed
            }
          }
        })
      )
    }

    return config
  }, [schema, values])

  return {
    schema,
    values,
    presetSecrets,
    errors,
    setFieldValue,
    load,
    validate,
    buildConfig,
  }
}

export type ProviderConfigController = ReturnType<typeof useProviderConfig>
