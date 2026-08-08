import { Plus, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export interface AttributeMappingRow {
  id: string
  externalClaim: string
  tokenClaim: string
}

interface AttributeMappingEditorProps {
  rows: AttributeMappingRow[]
  error?: string
  disabled?: boolean
  onAdd: () => void
  onUpdate: (id: string, patch: Partial<Omit<AttributeMappingRow, "id">>) => void
  onRemove: (id: string) => void
}

/**
 * Structured editor for the external-claim → token-claim mapping.
 *
 * Deliberately NOT the shared MetadataFieldEditor: that component sanitizes keys
 * against `[a-z0-9_-]`, which silently strips the dots out of a nested claim path
 * like `github.repository`. The exchange path resolves dotted paths, so using it
 * here would quietly corrupt a legitimate mapping.
 *
 * A structured editor also makes the previous failure mode impossible: the mapping
 * used to be a raw JSON textarea whose parse failure was swallowed, so a typo
 * silently erased a saved mapping and still reported success.
 */
export function AttributeMappingEditor({
  rows,
  error,
  disabled = false,
  onAdd,
  onUpdate,
  onRemove,
}: AttributeMappingEditorProps) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <Label>Attribute mapping</Label>
          <p className="text-sm text-muted-foreground">
            Copy claims from the workload&apos;s token into the issued access token. The
            source may be a nested path, e.g. <code className="text-xs">github.repository</code>.
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onAdd}
          disabled={disabled}
        >
          <Plus className="size-4 mr-1" />
          Add
        </Button>
      </div>

      {rows.length === 0 ? (
        <p className="text-sm text-muted-foreground rounded-md border border-dashed p-4">
          No claims are copied. The issued token will carry only the standard claims.
        </p>
      ) : (
        <div className="space-y-2">
          {rows.map((row) => (
            <div key={row.id} className="flex items-start gap-2">
              <Input
                aria-label="Source claim in the workload token"
                placeholder="repository"
                value={row.externalClaim}
                onChange={(e) => onUpdate(row.id, { externalClaim: e.target.value })}
                disabled={disabled}
                className="font-mono text-sm"
              />
              <span className="pt-2 text-muted-foreground" aria-hidden="true">
                &rarr;
              </span>
              <Input
                aria-label="Claim name to write in the issued token"
                placeholder="repository"
                value={row.tokenClaim}
                onChange={(e) => onUpdate(row.id, { tokenClaim: e.target.value })}
                disabled={disabled}
                className="font-mono text-sm"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Remove mapping"
                onClick={() => onRemove(row.id)}
                disabled={disabled}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          ))}
        </div>
      )}

      {error && (
        <p role="alert" className="text-sm text-red-600">
          {error}
        </p>
      )}
    </div>
  )
}
