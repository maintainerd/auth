/**
 * ProviderConfigSection
 *
 * Renders the dynamic, provider-aware configuration UI shared by the identity
 * and social provider forms: a "{Provider} Configuration" card whose fields
 * change with the selected provider (driven by the provider config schemas).
 *
 * There is no free-form key/value section: the backend validates config
 * strictly and rejects unknown keys, so only the schema's known fields are
 * editable. All values write into the same `config` JSON via the controller.
 */

import { ExternalLink, Info } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { FormInputField, FormPasswordField, FormSwitchField, FormTextareaField } from "@/components/form"
import { FormUrlField, FormScopeField } from "@/components/inputs"
import { PROVIDER_LABELS } from "./providerConfigSchemas"
import { ProviderLogo } from "./ProviderLogo"
import type { ProviderConfigController } from "./useProviderConfig"

export interface ProviderConfigSectionProps {
  provider: string
  controller: ProviderConfigController
  disabled?: boolean
}

export function ProviderConfigSection({ provider, controller, disabled }: ProviderConfigSectionProps) {
  const { schema, values, presetSecrets, errors, setFieldValue } = controller

  const providerLabel = PROVIDER_LABELS[provider] ?? "Provider"
  const hasFields = Boolean(schema && schema.groups.length > 0)

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <ProviderLogo provider={provider} className="size-9" iconClassName="size-5" />
          <CardTitle className="text-base">{providerLabel} Configuration</CardTitle>
        </div>
        {schema?.summary && (
          <p className="text-sm text-muted-foreground">{schema.summary}</p>
        )}
        {schema?.docsUrl && (
          <a
            href={schema.docsUrl}
            target="_blank"
            rel="noreferrer noopener"
            className="inline-flex w-fit items-center gap-1.5 text-sm font-medium text-primary hover:underline"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            {schema.docsLabel ?? "Setup documentation"}
          </a>
        )}
      </CardHeader>
      <CardContent className="space-y-6">
        {hasFields ? (
          schema!.groups.map((group, groupIndex) => (
            <div key={group.title} className="space-y-4">
              {groupIndex > 0 && <div className="border-t" />}
              <div className="space-y-0.5">
                <h4 className="text-sm font-semibold">{group.title}</h4>
                {group.description && (
                  <p className="text-sm text-muted-foreground">{group.description}</p>
                )}
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                {group.fields.map((field) => {
                  const commonProps = {
                    label: field.label,
                    placeholder: field.placeholder,
                    description: field.description,
                    required: field.required,
                    error: errors[field.key],
                    disabled,
                    value: values[field.key] ?? "",
                  }

                  if (field.type === "switch") {
                    return (
                      <FormSwitchField
                        key={field.key}
                        id={`pc-${field.key}`}
                        label={field.label}
                        description={field.description}
                        checked={values[field.key] === "true"}
                        onCheckedChange={(checked) => setFieldValue(field.key, String(checked))}
                        disabled={disabled}
                      />
                    )
                  }

                  if (field.type === "password") {
                    return (
                      <FormPasswordField
                        key={field.key}
                        {...commonProps}
                        // Required is satisfied by an already-stored secret, so
                        // don't show the asterisk once one exists.
                        required={field.required && !presetSecrets[field.key]}
                        description={
                          presetSecrets[field.key]
                            ? "A secret is stored. Leave blank to keep it, or enter a new value to replace it."
                            : field.description
                        }
                        placeholder={presetSecrets[field.key] ? "••••••••" : field.placeholder}
                        onChange={(e) => setFieldValue(field.key, e.target.value)}
                      />
                    )
                  }

                  if (field.type === "url") {
                    return (
                      <FormUrlField
                        key={field.key}
                        {...commonProps}
                        onChange={(e) => setFieldValue(field.key, e.target.value)}
                      />
                    )
                  }

                  if (field.type === "scopes") {
                    return (
                      <FormScopeField
                        key={field.key}
                        {...commonProps}
                        // Scopes read best full-width (values can be long URLs).
                        containerClassName="md:col-span-2"
                        suggestions={field.suggestions}
                        onValueChange={(v) => setFieldValue(field.key, v)}
                      />
                    )
                  }

                  if (field.type === "textarea") {
                    return (
                      <FormTextareaField
                        key={field.key}
                        {...commonProps}
                        containerClassName="md:col-span-2"
                        rows={4}
                        onChange={(e) => setFieldValue(field.key, e.target.value)}
                      />
                    )
                  }

                  return (
                    <FormInputField
                      key={field.key}
                      {...commonProps}
                      type="text"
                      onChange={(e) => setFieldValue(field.key, e.target.value)}
                    />
                  )
                })}
              </div>
            </div>
          ))
        ) : (
          <Alert>
            <Info className="h-4 w-4" />
            <AlertDescription>
              {schema?.summary ?? "This provider has no configuration fields."}
            </AlertDescription>
          </Alert>
        )}
      </CardContent>
    </Card>
  )
}
