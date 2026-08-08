import { Button } from '@/components/ui/button'
import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

interface FormSubmitButtonProps {
  isSubmitting: boolean
  submitText: string
  submittingText?: string
  disabled?: boolean
  className?: string
}

export default function FormSubmitButton({
  isSubmitting,
  submitText,
  submittingText,
  disabled = false,
  className
}: FormSubmitButtonProps) {
  const defaultSubmittingText = `${submitText.replace(/^(Create|Update|Add|Save|Delete)/, '$1ing')}...`
  const displaySubmittingText = submittingText || defaultSubmittingText

  return (
    <Button
      type="submit"
      disabled={isSubmitting || disabled}
      className={cn("gap-2", className)}
    >
      {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
      {isSubmitting ? displaySubmittingText : submitText}
    </Button>
  )
}
