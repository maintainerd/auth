/**
 * Details Container Component
 * Provides a consistent max-width centered container for detail pages and forms
 */

import type { ReactNode } from "react"
import { cn } from "@/lib/utils"

interface DetailsContainerProps {
  children: ReactNode
  className?: string
}

export function DetailsContainer({ children, className }: DetailsContainerProps) {
  return (
    <div className={cn("max-w-6xl mx-auto", className)}>
      {children}
    </div>
  )
}

