import { Skeleton } from "@/components/ui/skeleton"

/** Skeleton placeholder rows for a loading list inside a detail tab. */
export function ListSkeleton({ rows = 3 }: { rows?: number }) {
  return (
    <div className="space-y-3">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 rounded-lg border p-4">
          <Skeleton className="size-9 shrink-0 rounded-lg" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-3 w-56" />
          </div>
        </div>
      ))}
    </div>
  )
}
