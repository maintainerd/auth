import { useEffect, useRef, useState } from "react"
import { Link as LinkIcon, Trash2, Upload, User } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { cn } from "@/lib/utils"
import { compressImage } from "@/lib/imageCompression"

/**
 * Accepted upload types, matching what the server will decode. SVG is absent
 * deliberately — it is a document that can carry script, and the API rejects it.
 * Offering it here would only produce a file picker whose result is refused.
 */
export const AVATAR_ACCEPT = "image/png,image/jpeg,image/webp,image/gif"

/** Mirrors MaxProfilePictureBytes on the server. */
export const AVATAR_MAX_BYTES = 2 * 1024 * 1024

export type AvatarMode = "url" | "upload"

interface AvatarFieldProps {
  mode: AvatarMode
  onModeChange: (mode: AvatarMode) => void
  /** The saved avatar, whatever its source. */
  previewUrl?: string | null
  /** Chosen but not yet uploaded — previewed from an object URL. */
  pendingFile: File | null
  onFileChange: (file: File | null) => void
  onRemove?: () => void
  /** Rendered in URL mode; the parent owns the form registration. */
  urlField: React.ReactNode
  disabled?: boolean
  error?: string
}

/**
 * Avatar picker: link an image or upload one.
 *
 * The two are mutually exclusive — the server stores one source and clears the
 * other — so they are presented as a choice rather than two fields that could
 * disagree about which image wins.
 */
export function AvatarField({
  mode,
  onModeChange,
  previewUrl,
  pendingFile,
  onFileChange,
  onRemove,
  urlField,
  disabled,
  error,
}: AvatarFieldProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [localError, setLocalError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [objectUrl, setObjectUrl] = useState<string | null>(null)

  // Revoked on replace and unmount. Without this every re-render of a picked
  // file leaks another blob URL for the lifetime of the page.
  useEffect(() => {
    if (!pendingFile) {
      setObjectUrl(null)
      return
    }
    const url = URL.createObjectURL(pendingFile)
    setObjectUrl(url)
    return () => URL.revokeObjectURL(url)
  }, [pendingFile])

  const shownUrl = objectUrl ?? previewUrl ?? null
  const hasImage = Boolean(shownUrl)

  const pickFile = async (file: File | null) => {
    setLocalError(null)
    if (!file) {
      onFileChange(null)
      return
    }
    if (!AVATAR_ACCEPT.split(",").includes(file.type)) {
      setLocalError("Use a PNG, JPEG, WebP or GIF image.")
      return
    }

    setBusy(true)
    try {
      // Compress before the size check, not after: a phone photo is routinely
      // 4–8 MB, and rejecting it outright asks the user to do something most
      // people cannot.
      const { file: prepared } = await compressImage(file, AVATAR_MAX_BYTES)
      if (prepared.size > AVATAR_MAX_BYTES) {
        setLocalError("That image is too large to compress under 2 MB. Try a smaller one.")
        onFileChange(null)
        return
      }
      onFileChange(prepared)
    } catch {
      setLocalError("That image could not be read.")
      onFileChange(null)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col items-center gap-4 sm:flex-row sm:items-start">
        <div className="relative">
          <div className="flex size-20 items-center justify-center overflow-hidden rounded-full border bg-muted">
            {shownUrl ? (
              <img src={shownUrl} alt="" className="size-full object-cover" />
            ) : (
              <User className="size-8 text-muted-foreground" />
            )}
          </div>
          {busy && (
            <div className="absolute inset-0 flex items-center justify-center rounded-full bg-background/70 text-xs">
              …
            </div>
          )}
        </div>

        <div className="w-full min-w-0 flex-1 space-y-3">
          {/* Segmented control. gap-1 inside the track and generous horizontal
              padding on each option — the earlier version butted the icon
              against the label and the two options against each other. */}
          <div className="inline-flex w-full gap-1 rounded-lg border bg-muted/40 p-1 sm:w-auto">
            {([
              { value: "url", label: "Use a link", icon: LinkIcon },
              { value: "upload", label: "Upload", icon: Upload },
            ] as const).map(({ value, label, icon: Icon }) => (
              <button
                key={value}
                type="button"
                disabled={disabled || busy}
                aria-pressed={mode === value}
                onClick={() => onModeChange(value)}
                className={cn(
                  "inline-flex flex-1 items-center justify-center gap-2 rounded-md px-4 py-2 text-sm transition-colors sm:flex-none",
                  mode === value
                    ? "bg-background font-medium shadow-sm"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                <Icon className="size-4 shrink-0" />
                <span className="whitespace-nowrap">{label}</span>
              </button>
            ))}
          </div>

          {mode === "url" ? (
            urlField
          ) : (
            <div className="space-y-2">
              <input
                ref={inputRef}
                type="file"
                accept={AVATAR_ACCEPT}
                className="hidden"
                onChange={(e) => {
                  void pickFile(e.target.files?.[0] ?? null)
                  // Reset so re-picking the same file fires change again.
                  e.target.value = ""
                }}
              />
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={disabled || busy}
                  onClick={() => inputRef.current?.click()}
                >
                  <Upload className="mr-2 size-4" />
                  {busy ? "Preparing…" : hasImage ? "Change image" : "Choose image"}
                </Button>
                {hasImage && onRemove && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    disabled={disabled || busy}
                    onClick={() => {
                      onFileChange(null)
                      setLocalError(null)
                      onRemove()
                    }}
                  >
                    <Trash2 className="mr-2 size-4" />
                    Remove
                  </Button>
                )}
              </div>
              <p className="text-xs text-muted-foreground">
                {pendingFile
                  ? `${pendingFile.name} · ${formatBytes(pendingFile.size)}`
                  : "PNG, JPEG, WebP or GIF. Large images are resized automatically."}
              </p>
            </div>
          )}

          {(localError || error) && <p className="text-sm text-destructive">{localError ?? error}</p>}
        </div>
      </div>

      <Separator />
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
