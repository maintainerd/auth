import { Controller, type Control, type FieldErrors } from "react-hook-form"
import { X, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { PolicyFormData } from "@/lib/validations"

interface StatementFieldProps {
  statementIndex: number
  control: Control<PolicyFormData>
  errors: FieldErrors<PolicyFormData>
  isLoading: boolean
  canRemove: boolean
  onRemove: () => void
}

export function StatementField({
  statementIndex,
  control,
  errors,
  isLoading,
  canRemove,
  onRemove
}: StatementFieldProps) {
  const statementErrors = errors.document?.statement?.[statementIndex]

  return (
    <div className="space-y-4 rounded-lg border bg-background p-4">
      <div className="flex items-center justify-between">
        <div>
          <h4 className="text-sm font-medium">Statement {statementIndex + 1}</h4>
          <p className="text-sm text-muted-foreground">
            Match resources first, then define which actions are allowed or denied.
          </p>
        </div>
        {canRemove && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onRemove}
            className="gap-2 text-destructive hover:text-destructive"
            disabled={isLoading}
          >
            <X className="h-4 w-4" />
            Remove
          </Button>
        )}
      </div>

      <div className="space-y-2">
        <Label>Effect *</Label>
        <Controller
          name={`document.statement.${statementIndex}.effect` as const}
          control={control}
          render={({ field }) => (
            <Select
              value={field.value}
              onValueChange={field.onChange}
              disabled={isLoading}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="allow">Allow</SelectItem>
                <SelectItem value="deny">Deny</SelectItem>
              </SelectContent>
            </Select>
          )}
        />
        {statementErrors?.effect && (
          <p className="text-sm text-destructive">{statementErrors.effect.message}</p>
        )}
      </div>

      <Controller
        name={`document.statement.${statementIndex}.resource` as const}
        control={control}
        render={({ field }) => (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Resources *</Label>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => field.onChange([...field.value, ""])}
                className="gap-1 text-xs"
                disabled={isLoading}
              >
                <Plus className="h-3 w-3" />
                Add Resource
              </Button>
            </div>
            <div className="space-y-2">
              {field.value.map((resource, resourceIndex) => (
                <div key={resourceIndex} className="flex gap-2">
                  <Input
                    value={resource}
                    onChange={(e) => {
                      const newResources = [...field.value]
                      newResources[resourceIndex] = e.target.value
                      field.onChange(newResources)
                    }}
                    placeholder="e.g., auth:users, auth:*, *"
                    className="flex-1"
                    disabled={isLoading}
                  />
                  {field.value.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        const newResources = field.value.filter((_, i) => i !== resourceIndex)
                        field.onChange(newResources)
                      }}
                      className="px-2"
                      disabled={isLoading}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">
              Format: service-name:api-name (e.g., "auth:users", "auth:*" for all APIs, or "*" for all resources)
            </p>
            {statementErrors?.resource && (
              <p className="text-sm text-destructive">{statementErrors.resource.message}</p>
            )}
          </div>
        )}
      />

      <Controller
        name={`document.statement.${statementIndex}.action` as const}
        control={control}
        render={({ field }) => (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Actions *</Label>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => field.onChange([...field.value, ""])}
                className="gap-1 text-xs"
                disabled={isLoading}
              >
                <Plus className="h-3 w-3" />
                Add Action
              </Button>
            </div>
            <div className="space-y-2">
              {field.value.map((action, actionIndex) => (
                <div key={actionIndex} className="flex gap-2">
                  <Input
                    value={action}
                    onChange={(e) => {
                      const newActions = [...field.value]
                      newActions[actionIndex] = e.target.value
                      field.onChange(newActions)
                    }}
                    placeholder="e.g., users:read, users:write, users:*"
                    className="flex-1"
                    disabled={isLoading}
                  />
                  {field.value.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        const newActions = field.value.filter((_, i) => i !== actionIndex)
                        field.onChange(newActions)
                      }}
                      className="px-2"
                      disabled={isLoading}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">
              Format: permission (e.g., "users:read", "users:write", "users:*")
            </p>
            {statementErrors?.action && (
              <p className="text-sm text-destructive">{statementErrors.action.message}</p>
            )}
          </div>
        )}
      />
    </div>
  )
}
