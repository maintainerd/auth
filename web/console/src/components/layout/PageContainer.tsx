import type { ReactNode } from "react"
import { Card, CardContent } from "@/components/ui/card"

interface PageContainerProps {
  children: ReactNode
}

// The standard listing-page shell: the whole page (header → toolbar → table →
// pagination) lives inside a single encapsulating card, centered and width-capped
// to line up with the detail pages.
export function PageContainer({ children }: PageContainerProps) {
  return (
    <Card className="mx-auto w-full max-w-6xl py-6">
      <CardContent className="flex flex-col gap-6 px-6">
        {children}
      </CardContent>
    </Card>
  )
}
