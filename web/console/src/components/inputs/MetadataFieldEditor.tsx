import { X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import type { MetadataField } from "@/hooks/useMetadataFields"

export interface MetadataFieldEditorProps {
  fields: MetadataField[]
  error?: string
  disabled?: boolean
  onAdd: () => void
  onUpdate: (id: string, key: string, value: string) => void
  onRemove: (id: string) => void
}

export function MetadataFieldEditor({
  fields,
  error,
  disabled = false,
  onAdd,
  onUpdate,
  onRemove,
}: MetadataFieldEditorProps) {
  return (
    <div className="space-y-4">
      {error && (
        <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {fields.length === 0 ? (
        <div className="rounded-lg border border-dashed py-10 text-center">
          <p className="text-sm text-muted-foreground">No custom metadata yet.</p>
          <Button
            type="button"
            variant="link"
            onClick={onAdd}
            disabled={disabled}
            className="h-auto p-0 text-sm"
          >
            Add your first field
          </Button>
        </div>
      ) : (
        <div className="space-y-2">
          {fields.map((field) => (
            <div key={field.id} className="flex items-center gap-2">
              <Input
                aria-label="Metadata key"
                placeholder="Key (e.g., department)"
                value={field.key}
                onChange={(e) => onUpdate(field.id, e.target.value, field.value)}
                disabled={disabled}
              />
              <Input
                aria-label="Metadata value"
                placeholder="Value (e.g., Engineering)"
                value={field.value}
                onChange={(e) => onUpdate(field.id, field.key, e.target.value)}
                disabled={disabled}
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => onRemove(field.id)}
                disabled={disabled}
                className="size-9 shrink-0 p-0 text-muted-foreground"
              >
                <span className="sr-only">Remove field</span>
                <X className="size-4" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
