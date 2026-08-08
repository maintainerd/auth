import { Plus, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { FormUrlField } from "@/components/inputs"

/** A single URI row, tracking its server-side id when it already exists. */
export interface UriEntry {
  uri: string
  id?: string
}

interface UriListFieldProps {
  label: string
  description: string
  placeholder: string
  addLabel: string
  items: UriEntry[]
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, value: string) => void
  disabled?: boolean
}

/**
 * A repeatable list of URI inputs with add/remove controls. Used for redirect
 * URIs, allowed origins, logout URLs, and CORS origins on the client form so
 * each list shares one consistent, accessible layout.
 */
export function UriListField({
  label,
  description,
  placeholder,
  addLabel,
  items,
  onAdd,
  onRemove,
  onChange,
  disabled,
}: UriListFieldProps) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <p className="text-xs text-muted-foreground">{description}</p>
      {items.map((item, index) => (
        <div key={index} className="flex gap-2">
          <FormUrlField
            label={`${label} ${index + 1}`}
            labelClassName="sr-only"
            containerClassName="flex-1"
            value={item.uri}
            onChange={(event) => onChange(index, event.target.value)}
            placeholder={placeholder}
            disabled={disabled}
          />
          {items.length > 1 && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onRemove(index)}
              disabled={disabled}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={onAdd}
        disabled={disabled}
        className="gap-2"
      >
        <Plus className="h-4 w-4" />
        {addLabel}
      </Button>
    </div>
  )
}
