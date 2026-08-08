import * as React from "react"

import { cn } from "@/lib/utils"
import { Tabs } from "@/components/ui/tabs"

/**
 * Detail-page tabs: the generic `Tabs` primitive plus the shared detail-page
 * layout spacing — a gap above the tab bar (separating it from the header card)
 * and a comfortable gap between the bar and its content, so every detail page
 * lines up without repeating margins per page.
 *
 * The spacing lives here, not in the `ui/tabs` primitive, so that primitive
 * stays layout-neutral and reusable elsewhere (settings, config pages, etc.).
 */
export function DetailTabs({
  className,
  ...props
}: React.ComponentProps<typeof Tabs>) {
  return <Tabs className={cn("mt-2 gap-6", className)} {...props} />
}
