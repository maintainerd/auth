import { useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Palette, Plus, Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/layout"
import { EmptyState } from "@/components/details"
import { useBrandings } from "@/hooks/useBranding"
import { BrandingCard } from "./components/BrandingCard"

export default function BrandingTemplatesPage({ standalone = true }: { standalone?: boolean }) {
  const navigate = useNavigate()
  const { data, isLoading, isError } = useBrandings()
  const [search, setSearch] = useState("")

  const brandings = useMemo(() => {
    const all = data ?? []
    const term = search.trim().toLowerCase()
    if (!term) return all
    return all.filter(
      (b) =>
        b.name?.toLowerCase().includes(term) || b.company_name?.toLowerCase().includes(term),
    )
  }, [data, search])

  const content = (
    <div className="flex flex-col gap-4">
      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search templates by name or company..."
          className="pl-9"
        />
      </div>

      {isLoading && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i} className="overflow-hidden py-0">
              <Skeleton className="h-20 w-full rounded-none" />
              <CardContent className="space-y-2 p-4">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-5 w-16" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {isError && (
        <p className="py-8 text-center text-sm text-destructive">Failed to load branding</p>
      )}

      {data && brandings.length === 0 && (
        <EmptyState
          icon={Palette}
          title={search ? "No matching templates" : "No branding templates"}
          description={
            search
              ? "No templates match your search. Try a different name or company."
              : "Create a branding template to customize the look of your auth experience."
          }
        />
      )}

      {brandings.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {brandings.map((b) => (
            <BrandingCard key={b.branding_id} branding={b} />
          ))}
        </div>
      )}
    </div>
  )

  if (!standalone) return content

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <PageHeader
          title="Themes"
          description="Create and manage branding themes. The active template is the style loaded across the auth experience."
        />
        <Button
          variant="outline"
          size="sm"
          className="h-9 shrink-0 gap-2"
          onClick={() => navigate(`/branding/templates/create`)}
        >
          <Plus className="size-4" />
          New Template
        </Button>
      </div>
      {content}
    </div>
  )
}
