import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"

interface DataTableActiveFiltersProps {
  activeFilters: string[]
  onClearAll: () => void
}

export function DataTableActiveFilters({
  activeFilters,
  onClearAll,
}: DataTableActiveFiltersProps) {
  if (activeFilters.length === 0) {
    return null
  }

  return (
    <div className="flex flex-wrap items-center gap-2 p-3 bg-muted/50 rounded-lg">
      <span className="text-sm font-medium text-muted-foreground">Active filters:</span>
      {activeFilters.map((filter, index) => (
        <Badge key={index} variant="secondary" className="text-xs">
          {filter}
        </Badge>
      ))}
      <Button
        variant="ghost"
        size="sm"
        onClick={onClearAll}
        className="h-6 px-2 text-xs"
      >
        Clear all
      </Button>
    </div>
  )
}

