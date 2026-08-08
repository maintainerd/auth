import { useState, useEffect } from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { TriangleAlert, Loader2 } from "lucide-react"

interface DeleteConfirmationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void | Promise<void>
  title: string
  description: string
  /** Optional extra consequence shown as a destructive callout below the description. */
  confirmationText?: string
  /** The exact text the user must type to enable the destructive action. */
  itemName: string
  isDeleting?: boolean
  /** Label for the destructive button (default "Delete"). */
  confirmLabel?: string
}

export function DeleteConfirmationDialog({
  open,
  onOpenChange,
  onConfirm,
  title,
  description,
  confirmationText,
  itemName,
  isDeleting = false,
  confirmLabel = "Delete",
}: DeleteConfirmationDialogProps) {
  const [inputValue, setInputValue] = useState("")
  const isConfirmEnabled = inputValue.trim() === itemName.trim()

  // Reset input when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setInputValue("")
    }
  }, [open])

  const handleConfirm = async () => {
    if (isConfirmEnabled) {
      await onConfirm()
      onOpenChange(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[460px]">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-destructive/10 text-destructive">
              <TriangleAlert className="size-5" />
            </div>
            <DialogTitle>{title}</DialogTitle>
          </div>
          <DialogDescription className="pt-1 text-left">{description}</DialogDescription>
        </DialogHeader>

        {confirmationText && (
          <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2.5 text-sm text-destructive">
            {confirmationText}
          </div>
        )}

        <div className="space-y-2">
          <Label htmlFor="confirmation-input" className="font-normal text-muted-foreground">
            Type <span className="font-semibold text-foreground">{itemName}</span> to confirm
          </Label>
          <Input
            id="confirmation-input"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            disabled={isDeleting}
            autoComplete="off"
            autoFocus
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isDeleting}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={!isConfirmEnabled || isDeleting}
          >
            {isDeleting && <Loader2 className="size-4 animate-spin" />}
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
