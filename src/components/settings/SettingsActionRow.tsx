import type { ComponentProps, ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ListingItemCard } from '@/components/details'
import { cn } from '@/lib/utils'

interface SettingsActionRowProps {
  icon: LucideIcon
  title: string
  description: string
  action?: ReactNode
  actionLabel?: string
  actionVariant?: ComponentProps<typeof Button>['variant']
  className?: string
  disabled?: boolean
  hideChevron?: boolean
  onAction?: () => void
}

export function SettingsActionRow({
  icon,
  title,
  description,
  action,
  actionLabel,
  actionVariant = 'ghost',
  className,
  disabled,
  hideChevron,
  onAction,
}: SettingsActionRowProps) {
  const actionNode = action ?? (
    <Button
      type="button"
      variant={actionVariant}
      size="sm"
      className="shrink-0"
      onClick={onAction}
      disabled={disabled}
    >
      {actionLabel}
      {!hideChevron && <ChevronRight className="ml-1 size-4" />}
    </Button>
  )

  return (
    <ListingItemCard
      icon={icon}
      action={actionNode}
      iconClassName="size-10"
      contentClassName="items-center gap-4"
      className={cn('items-center rounded-none border-0 px-4 py-4 first:rounded-t-xl last:rounded-b-xl', className)}
    >
      <p className="text-sm font-medium">{title}</p>
      <p className="text-sm text-muted-foreground">{description}</p>
    </ListingItemCard>
  )
}
