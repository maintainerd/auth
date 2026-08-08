/**
 * Form Page Header Component
 * Reusable header for form pages with back button, title, description, badges, and warnings
 */

import type { ReactNode } from "react"
import { useNavigate } from "react-router-dom"
import { ArrowLeft, Shield } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription } from "@/components/ui/alert"

interface FormPageHeaderProps {
  // Navigation
  backUrl: string
  backLabel?: string
  // Optional override for the back action (e.g. an unsaved-changes guard).
  // When provided it replaces the default navigate(backUrl).
  onBack?: () => void

  // Header content
  title: string
  description: string
  
  // Optional badges
  showSystemBadge?: boolean
  customBadge?: ReactNode
  
  // Optional warning
  showWarning?: boolean
  warningIcon?: ReactNode
  warningMessage?: string
  
  // Additional content
  headerActions?: ReactNode
}

export function FormPageHeader({
  backUrl,
  backLabel = "Back",
  onBack,
  title,
  description,
  showSystemBadge = false,
  customBadge,
  showWarning = false,
  warningIcon,
  warningMessage,
  headerActions,
}: FormPageHeaderProps) {
  const navigate = useNavigate()

  return (
    <>
      {/* Back Button */}
      <div>
        <Button
          data-md-back-button
          variant="ghost"
          size="sm"
          onClick={onBack ?? (() => navigate(backUrl))}
          className="-ml-2 w-fit gap-2 text-muted-foreground"
        >
          <ArrowLeft className="size-4" />
          {backLabel}
        </Button>
      </div>

      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold tracking-tight">{title}</h1>
            
            {/* System Badge */}
            {showSystemBadge && (
              <Badge variant="secondary" className="gap-1">
                <Shield className="h-3 w-3" />
                System
              </Badge>
            )}
            
            {/* Custom Badge */}
            {customBadge}
          </div>
          
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
        
        {/* Optional header actions (e.g., delete button) */}
        {headerActions}
      </div>

      {/* Warning Alert */}
      {showWarning && warningMessage && (
        <Alert>
          {warningIcon || <Shield className="h-4 w-4" />}
          <AlertDescription>{warningMessage}</AlertDescription>
        </Alert>
      )}
    </>
  )
}
