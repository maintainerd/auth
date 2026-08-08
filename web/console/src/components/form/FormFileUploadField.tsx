/**
 * Reusable Form File Upload Field Component
 * A file upload field with drag & drop, preview, and validation
 */

import { forwardRef, useState, useRef, useCallback } from "react"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { cn } from "@/lib/utils"
import { Upload, X, User } from "lucide-react"

export interface FormFileUploadFieldProps {
  label: string
  error?: string
  description?: string
  required?: boolean
  containerClassName?: string
  labelClassName?: string
  errorClassName?: string
  descriptionClassName?: string
  accept?: string
  maxSize?: number // in bytes
  preview?: boolean
  value?: string // URL of current image
  onChange?: (file: File | null) => void
  onUpload?: (file: File) => Promise<string> // Returns uploaded file URL
  uploading?: boolean
}

const FormFileUploadField = forwardRef<HTMLInputElement, FormFileUploadFieldProps>(
  (
    {
      label,
      error,
      description,
      required = false,
      containerClassName,
      labelClassName,
      errorClassName,
      descriptionClassName,
      accept = "image/*",
      maxSize = 5 * 1024 * 1024, // 5MB default
      preview = true,
      value,
      onChange,
      onUpload,
      uploading = false,
      ...props
    },
    ref
  ) => {
    const [dragActive, setDragActive] = useState(false)
    const [previewUrl, setPreviewUrl] = useState<string | null>(value || null)
    // Local rejection message (e.g. oversize file) surfaced through FieldError,
    // separate from the parent-controlled `error` prop.
    const [rejectionError, setRejectionError] = useState<string | null>(null)
    const inputRef = useRef<HTMLInputElement>(null)

    // Generate ID if not provided so the label/input/error are properly linked.
    const fieldId = label.toLowerCase().replace(/\s+/g, '-')

    // Parent-provided validation error takes precedence over a local rejection.
    const displayError = error || rejectionError

    const setInputRef = useCallback((node: HTMLInputElement | null) => {
      inputRef.current = node
      if (typeof ref === "function") {
        ref(node)
      } else if (ref) {
        ref.current = node
      }
    }, [ref])

    const handleFiles = useCallback(async (files: FileList | null) => {
      if (!files || files.length === 0) return

      const file = files[0]

      // Validate file size — surface a rejection message instead of failing silently.
      if (file.size > maxSize) {
        setRejectionError(
          `File is too large. Maximum size is ${Math.round(maxSize / (1024 * 1024))}MB.`
        )
        return
      }

      // Clear any previous rejection now that a valid file was chosen.
      setRejectionError(null)

      // Create preview URL
      if (preview) {
        const url = URL.createObjectURL(file)
        setPreviewUrl(url)
      }

      // Call onChange
      onChange?.(file)

      // Upload file if onUpload is provided
      if (onUpload) {
        try {
          const uploadedUrl = await onUpload(file)
          setPreviewUrl(uploadedUrl)
        } catch (error) {
          console.error('Upload failed:', error)
          // Reset preview on upload failure
          setPreviewUrl(value || null)
        }
      }
    }, [maxSize, preview, onChange, onUpload, value])

    const handleDrag = useCallback((e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.type === "dragenter" || e.type === "dragover") {
        setDragActive(true)
      } else if (e.type === "dragleave") {
        setDragActive(false)
      }
    }, [])

    const handleDrop = useCallback((e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setDragActive(false)
      
      if (e.dataTransfer.files && e.dataTransfer.files[0]) {
        handleFiles(e.dataTransfer.files)
      }
    }, [handleFiles])

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
      e.preventDefault()
      if (e.target.files && e.target.files[0]) {
        handleFiles(e.target.files)
      }
    }, [handleFiles])

    const handleButtonClick = useCallback(() => {
      inputRef.current?.click()
    }, [])

    const handleRemove = useCallback(() => {
      setPreviewUrl(null)
      onChange?.(null)
      if (inputRef.current) {
        inputRef.current.value = ''
      }
    }, [onChange])

    return (
      <Field className={cn(containerClassName)}>
        <FieldLabel htmlFor={fieldId} className={cn(labelClassName)}>
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </FieldLabel>
        
        <div className="space-y-4">
          {/* Preview */}
          {preview && previewUrl && (
            <div className="flex items-center gap-4">
              <Avatar className="h-20 w-20">
                <AvatarImage src={previewUrl} alt="Profile preview" />
                <AvatarFallback>
                  <User className="h-8 w-8" />
                </AvatarFallback>
              </Avatar>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleRemove}
                disabled={uploading}
              >
                <X className="h-4 w-4 mr-2" />
                Remove
              </Button>
            </div>
          )}

          {/* Upload Area */}
          <div
            className={cn(
              "relative border border-dashed rounded-lg p-6 transition-colors",
              dragActive ? "border-primary bg-primary/5" : "border-muted-foreground/25",
              displayError && "border-red-500",
              uploading && "opacity-50 pointer-events-none"
            )}
            onDragEnter={handleDrag}
            onDragLeave={handleDrag}
            onDragOver={handleDrag}
            onDrop={handleDrop}
          >
            <input
              ref={setInputRef}
              id={fieldId}
              type="file"
              accept={accept}
              onChange={handleChange}
              className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
              disabled={uploading}
              aria-invalid={displayError ? "true" : "false"}
              aria-describedby={
                displayError ? `${fieldId}-error` :
                description ? `${fieldId}-description` :
                undefined
              }
              {...props}
            />
            
            <div className="flex flex-col items-center justify-center text-center">
              <Upload className="h-10 w-10 text-muted-foreground mb-4" />
              <div className="space-y-2">
                <p className="text-sm font-medium">
                  {uploading ? "Uploading..." : "Drop your image here, or click to browse"}
                </p>
                <p className="text-xs text-muted-foreground">
                  PNG, JPG, GIF up to {Math.round(maxSize / (1024 * 1024))}MB
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="mt-4"
                onClick={handleButtonClick}
                disabled={uploading}
              >
                Choose File
              </Button>
            </div>
          </div>
        </div>

        {description && !displayError && (
          <FieldDescription id={`${fieldId}-description`} className={cn("text-muted-foreground", descriptionClassName)}>
            {description}
          </FieldDescription>
        )}

        {displayError && (
          <FieldError id={`${fieldId}-error`} className={cn(errorClassName)}>
            {displayError}
          </FieldError>
        )}
      </Field>
    )
  }
)

FormFileUploadField.displayName = "FormFileUploadField"

export { FormFileUploadField }
