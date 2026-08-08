import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Search } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import featureIndex from "@/data/features.json"

interface ScoredFeature {
  label: string
  path: string
  keywords: string[]
  score: number
  category: string
}

function scoreFeature(feat: (typeof featureIndex.categories)[number]["features"][number], query: string): number {
  const q = query.toLowerCase()
  const label = feat.label.toLowerCase()
  let score = 0
  if (label === q) score += 200
  else if (label.startsWith(q)) score += 100
  else if (label.includes(q)) score += 50
  for (const kw of feat.keywords) {
    if (kw.includes(q)) score += 20
  }
  return score
}

// Feature search is a static, client-side index (src/data/features.json): the
// routes an operator can reach, with labels and keywords to match against. It
// never calls the API — there is nothing to load, and the index is the single
// place new pages register themselves for search.
export function FeatureSearch({ className }: { className?: string }) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const navigate = useNavigate()

  useEffect(() => {
    if (!open) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") { setOpen(false); setQuery("") }
    }
    document.addEventListener("keydown", onKeyDown)
    return () => document.removeEventListener("keydown", onKeyDown)
  }, [open])

  useEffect(() => {
    const onGlobalKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault()
        setQuery("")
        setOpen(true)
      }
    }
    document.addEventListener("keydown", onGlobalKeyDown)
    return () => document.removeEventListener("keydown", onGlobalKeyDown)
  }, [])

  const results = useMemo(() => {
    const q = query.trim()
    if (!q) return { scored: [] as ScoredFeature[], grouped: new Map<string, ScoredFeature[]>() }

    const scored: ScoredFeature[] = []
    for (const cat of featureIndex.categories) {
      for (const feat of cat.features) {
        const score = scoreFeature(feat, q)
        if (score > 0) {
          scored.push({ ...feat, score, category: cat.name })
        }
      }
    }
    scored.sort((a, b) => b.score - a.score)

    const grouped = new Map<string, ScoredFeature[]>()
    for (const item of scored) {
      const list = grouped.get(item.category)
      if (list) list.push(item)
      else grouped.set(item.category, [item])
    }
    return { scored, grouped }
  }, [query])

  const select = (path: string) => {
    setOpen(false)
    setQuery("")
    navigate(path)
  }

  return (
    <div className={cn("min-w-0", className)}>
      <Popover open={open} onOpenChange={(v) => { setOpen(v); if (!v) setQuery("") }}>
        <PopoverTrigger asChild>
          <Button
            data-console-top-search-trigger
            variant="ghost"
            role="combobox"
            aria-expanded={open}
            aria-label="Search features"
            className="h-10 w-full gap-2 border border-[var(--md-top-search-border)] bg-[var(--md-top-search-bg)] px-3 text-sm text-[var(--md-top-search-text)] hover:bg-[var(--md-top-search-hover)] hover:text-[var(--md-top-search-text)] data-[state=open]:bg-[var(--md-top-search-hover)] data-[state=open]:text-[var(--md-top-search-text)]"
          >
            <Search className="size-4 shrink-0 opacity-60" />
            <span className="hidden min-w-0 flex-1 truncate text-left font-medium md:inline">
              Search features
            </span>
            <kbd className="hidden h-5 items-center gap-0.5 rounded border border-[var(--md-top-search-border)] bg-[var(--md-top-search-hover)] px-1.5 font-sans text-[10px] opacity-60 md:inline-flex">
              ⌘K
            </kbd>
          </Button>
        </PopoverTrigger>

        <PopoverContent className="w-96 p-0" align="start">
          <Command shouldFilter={false}>
            <CommandInput
              value={query}
              onValueChange={setQuery}
              placeholder="Search features..."
              autoFocus
            />
            <CommandList>
              {!query ? (
                <p className="py-6 text-center text-sm text-muted-foreground">
                  Type to search features.
                </p>
              ) : results.scored.length === 0 ? (
                <CommandEmpty>No feature found.</CommandEmpty>
              ) : (
                [...results.grouped.entries()].map(([category, items]) => (
                  <CommandGroup key={category} heading={category}>
                    {items.map((item) => (
                      <CommandItem
                        key={item.path}
                        value={item.label}
                        onSelect={() => select(item.path)}
                        className="cursor-pointer gap-2"
                      >
                        <Search className="size-3.5 shrink-0 text-muted-foreground" />
                        <span className="truncate">{item.label}</span>
                      </CommandItem>
                    ))}
                  </CommandGroup>
                ))
              )}
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  )
}
