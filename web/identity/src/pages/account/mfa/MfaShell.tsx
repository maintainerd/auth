import { Link } from "react-router-dom"
import { ArrowLeft } from "lucide-react"
import AccountLayout from "@/components/layout/AccountLayout"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"

export const MFA_HUB_ROUTE = "/account/mfa"

// Shared chrome for the per-method setup pages: the account layout (sidenav +
// topnav) plus a "Back" link to the MFA hub, matching the console's dedicated
// setup pages.
export function MFASetupShell({ children }: { children: React.ReactNode }) {
  return (
    <AccountLayout title="Two-Factor Authentication">
      <Link
        to={MFA_HUB_ROUTE}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> Back to Two-Factor Authentication
      </Link>
      {children}
    </AccountLayout>
  )
}

export function ConfirmRemoveDialog({
  open, onOpenChange, onConfirm, title, description, confirmText = "Remove", loadingText = "Removing…", isLoading,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  title: string
  description: string
  confirmText?: string
  loadingText?: string
  isLoading?: boolean
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isLoading}>
            {isLoading ? loadingText : confirmText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function MfaSetupSkeleton() {
  return (
    <Card>
      <CardContent className="space-y-4 py-6">
        <div className="h-5 w-56 animate-pulse rounded bg-muted" />
        <div className="h-4 w-72 animate-pulse rounded bg-muted" />
        <div className="h-40 w-full animate-pulse rounded-lg bg-muted" />
      </CardContent>
    </Card>
  )
}
