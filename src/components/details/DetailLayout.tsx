import type { ReactNode } from "react"
import { ArrowLeft, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DetailsContainer } from "@/components/container"

interface DetailLayoutProps {
  backLabel: string
  onBack: () => void
  isLoading?: boolean
  isError?: boolean
  notFoundTitle?: string
  notFoundDescription?: string
  /** Rendered only once the entity has loaded successfully. */
  children: ReactNode
}

/**
 * The shared detail-page shell: a back link plus the standard loading skeleton
 * and not-found states. `children` (header + content) renders only on success,
 * so it can safely assume the entity exists.
 */
export function DetailLayout({
  backLabel,
  onBack,
  isLoading,
  isError,
  notFoundTitle = "Not found",
  notFoundDescription = "The item you're looking for doesn't exist or may have been removed.",
  children,
}: DetailLayoutProps) {
  return (
    <DetailsContainer>
      <div className="flex flex-col gap-4">
        <Button
          data-md-back-button
          variant="ghost"
          size="sm"
          onClick={onBack}
          className="-ml-2 w-fit gap-2 text-muted-foreground"
        >
          <ArrowLeft className="size-4" />
          {backLabel}
        </Button>

        {isLoading ? (
          <DetailSkeleton />
        ) : isError ? (
          <DetailError
            title={notFoundTitle}
            description={notFoundDescription}
            backLabel={backLabel}
            onBack={onBack}
          />
        ) : (
          children
        )}
      </div>
    </DetailsContainer>
  )
}

function DetailSkeleton() {
  return (
    <>
      <Card>
        <CardContent className="space-y-5">
          <div className="flex items-center gap-4">
            <Skeleton className="size-14 rounded-xl" />
            <div className="space-y-2">
              <Skeleton className="h-5 w-48" />
              <Skeleton className="h-4 w-28" />
            </div>
          </div>
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="space-y-2">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-4 w-32" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
      <Skeleton className="h-64 w-full rounded-xl" />
    </>
  )
}

function DetailError({
  title,
  description,
  backLabel,
  onBack,
}: {
  title: string
  description: string
  backLabel: string
  onBack: () => void
}) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center justify-center gap-4 py-12 text-center">
        <div className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <AlertCircle className="size-6" />
        </div>
        <div className="space-y-1">
          <h2 className="text-lg font-semibold">{title}</h2>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
        <Button data-md-back-button variant="outline" onClick={onBack}>
          <ArrowLeft className="mr-2 size-4" />
          {backLabel}
        </Button>
      </CardContent>
    </Card>
  )
}
