import * as React from "react"
import type { ReactNode } from "react"
import type { Table } from "@tanstack/react-table"
import { Filter, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { useDebouncedSearch } from "@/hooks/useDebouncedSearch"
import { DataTableViewOptions } from "./DataTableViewOptions"
import type { FilterGroup, ListingFilters } from "./useServerDataTable"

interface ListingToolbarProps<TRow> {
  table: Table<TRow>
  search: string
  onSearchChange: (value: string) => void
  searchPlaceholder: string
  filterGroups?: readonly FilterGroup[]
  filters: ListingFilters
  onFiltersChange: (filters: ListingFilters) => void
  /** Renders the primary "New …" button when provided. */
  onCreate?: () => void
  createLabel?: string
  /** Extra action elements rendered after the column-toggle and before create. */
  extraActions?: ReactNode
}

/**
 * The shared listing toolbar: white search input + checkbox Filters popover +
 * column visibility + an optional create action — all sized/coloured to the
 * standard so no listing page re-implements it.
 */
export function ListingToolbar<TRow>({
  table,
  search,
  onSearchChange,
  searchPlaceholder,
  filterGroups = [],
  filters,
  onFiltersChange,
  onCreate,
  createLabel = "New",
  extraActions,
}: ListingToolbarProps<TRow>) {
  const [isFilterOpen, setIsFilterOpen] = React.useState(false)
  const { searchInput, handleSearchChange, handleKeyDown } = useDebouncedSearch({
    initialValue: search,
    delay: 500,
    onDebouncedChange: onSearchChange,
  })

  const toggleValue = (key: string, value: string, checked: boolean) => {
    const current = filters[key] ?? []
    onFiltersChange({
      ...filters,
      [key]: checked ? [...current, value] : current.filter((v) => v !== value),
    })
  }

  const clearAll = () => {
    const cleared: ListingFilters = {}
    for (const group of filterGroups) cleared[group.key] = []
    onFiltersChange(cleared)
  }

  const activeFilterCount = filterGroups.reduce(
    (count, group) => count + (filters[group.key]?.length ? 1 : 0),
    0,
  )

  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex flex-1 items-center gap-2">
        <Input
          placeholder={searchPlaceholder}
          value={searchInput}
          onChange={(e) => handleSearchChange(e.target.value)}
          onKeyDown={handleKeyDown}
          className="w-full bg-background sm:w-80"
        />

        {filterGroups.length > 0 && (
          <Popover open={isFilterOpen} onOpenChange={setIsFilterOpen}>
            <PopoverTrigger asChild>
              <Button data-md-filter-button variant="outline" className="relative bg-background">
                <Filter className="mr-2 h-4 w-4" />
                <span className="hidden sm:inline">Filters</span>
                {activeFilterCount > 0 && (
                  <Badge variant="destructive" className="ml-2 h-5 w-5 rounded-full p-0 text-xs">
                    {activeFilterCount}
                  </Badge>
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-80" align="start">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h4 className="font-medium">Filters</h4>
                  {activeFilterCount > 0 && (
                    <Button variant="ghost" size="sm" onClick={clearAll}>
                      Clear All
                    </Button>
                  )}
                </div>

                {filterGroups.map((group) => (
                  <div key={group.key} className="space-y-2">
                    <Label className="text-sm font-medium">{group.label}</Label>
                    <div className="grid grid-cols-2 gap-2">
                      {group.options.map((option) => (
                        <div key={option} className="flex items-center space-x-2">
                          <Checkbox
                            id={`${group.key}-${option}`}
                            checked={filters[group.key]?.includes(option) ?? false}
                            onCheckedChange={(checked) => toggleValue(group.key, option, checked === true)}
                          />
                          <Label htmlFor={`${group.key}-${option}`} className="text-sm capitalize">
                            {option}
                          </Label>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </PopoverContent>
          </Popover>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <DataTableViewOptions table={table} />
        {extraActions}
        {onCreate && (
          <Button onClick={onCreate}>
            <Plus className="mr-2 h-4 w-4" />
            <span className="hidden sm:inline">{createLabel}</span>
          </Button>
        )}
      </div>
    </div>
  )
}
