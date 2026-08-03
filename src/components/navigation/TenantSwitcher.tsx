import { useState } from "react"
import { Check, ChevronsUpDown, Plus, Shield } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
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
import { CreateTenantDialog, ConfirmationDialog } from "@/components/dialog"
import { useTenant } from "@/hooks/useTenant"
import { useTenantsList } from "@/hooks/useTenants"
import { logoutAndRedirect } from "@/services/api/auth"
import { tenantConsoleUrl } from "@/utils/tenant"
import type { TenantEntity } from "@/services/api/tenants/types"

export function TenantSwitcher({
  className,
  size = "default",
}: {
  className?: string
  size?: "default" | "compact"
}) {
  const [open, setOpen] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [switchTarget, setSwitchTarget] = useState<TenantEntity | null>(null)

  // The current tenant comes from the store (resolved from the host subdomain on
  // app init), so the label is always correct — even if the active tenant is
  // beyond the list cap.
  const { currentTenant, isLoading: tenantLoading } = useTenant()
  const { data: tenantsData, isLoading: tenantsLoading } = useTenantsList({ limit: 100 })
  const tenants = (tenantsData?.data?.rows as TenantEntity[]) || []

  // The active tenant is the one the backend resolved for this host (stored in
  // `currentTenant`). Never fall back to "the first tenant" — that would show
  // the wrong tenant.
  const active = currentTenant ?? null
  const label = active?.name ?? "Select tenant"
  const showSkeleton = !active && (tenantLoading || tenantsLoading)

  const handleSelect = (tenant: TenantEntity) => {
    setOpen(false)
    if (tenant.tenant_id !== active?.tenant_id) {
      setSwitchTarget(tenant)
    }
  }

  const handleConfirmSwitch = () => {
    if (!switchTarget) return
    // Switching tenants re-authenticates into the target tenant, which lives on a
    // different console subdomain. Clear the local session and hard-navigate to
    // the target tenant's dashboard (a protected route), which starts a fresh
    // OAuth flow there. The system tenant has no slug (null). Doing it
    // synchronously avoids the SPA render race that could otherwise fire the
    // OAuth redirect for the *current* route mid-switch.
    const targetSlug = switchTarget.is_system ? null : switchTarget.name
    logoutAndRedirect(tenantConsoleUrl(targetSlug, "/dashboard"))
  }

  const handleCreate = () => {
    setOpen(false)
    setCreateOpen(true)
  }

  return (
    <div className={cn("min-w-0", className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            data-console-top-dropdown-trigger
            variant="ghost"
            role="combobox"
            aria-expanded={open}
            aria-label="Switch tenant"
            className={cn(
              size === "compact"
                ? "h-9 gap-1.5 border border-slate-700 bg-white/5 px-2 text-xs text-slate-300 hover:bg-white/10 hover:text-white active:!bg-white/15 active:!text-white data-[state=open]:!bg-white/15 data-[state=open]:!text-white"
                : "h-10 gap-2 border border-slate-700 bg-white/5 px-2 text-sm text-slate-300 hover:bg-white/10 hover:text-white active:!bg-white/15 active:!text-white data-[state=open]:!bg-white/15 data-[state=open]:!text-white",
            )}
          >
            {showSkeleton ? (
              <Skeleton
                className={cn(
                  size === "compact" ? "h-3.5 w-20" : "h-4 w-24",
                )}
              />
            ) : (
              <span className="min-w-0 flex-1 truncate text-left font-medium">{label}</span>
            )}
            <ChevronsUpDown
              className={cn(
                "ml-auto shrink-0",
                size === "compact" ? "size-3.5 text-slate-400" : "size-4 text-slate-400",
              )}
            />
          </Button>
        </PopoverTrigger>

        <PopoverContent className="w-72 p-0" align="start">
          <Command>
            <CommandInput placeholder="Search tenants..." />
            <CommandList>
              {tenantsLoading ? (
                <div className="space-y-2 p-2">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="flex items-center gap-2 px-1 py-1">
                      <Skeleton className="size-6 rounded-md" />
                      <Skeleton className="h-4 w-32" />
                    </div>
                  ))}
                </div>
              ) : (
                <>
                  <CommandEmpty>No tenants found.</CommandEmpty>
                  <CommandGroup heading="Switch tenant">
                    {tenants.map((tenant) => {
                      const isActive = tenant.tenant_id === active?.tenant_id
                      return (
                        <CommandItem
                          key={tenant.tenant_id}
                          value={`${tenant.name}`}
                          onSelect={() => handleSelect(tenant)}
                          className="cursor-pointer gap-2"
                        >
                          <div className="flex min-w-0 flex-col">
                            <div className="flex items-center gap-1.5">
                              <span className="truncate font-medium">{tenant.name}</span>
                              {tenant.is_system && (
                                <Badge variant="secondary" className="h-4 gap-0.5 px-1 text-[10px]">
                                  <Shield className="size-2.5" />
                                  System
                                </Badge>
                              )}
                            </div>
                            {tenant.description && (
                              <span className="truncate text-xs text-muted-foreground">
                                {tenant.description}
                              </span>
                            )}
                          </div>
                          <Check
                            className={cn(
                              "ml-auto size-4 text-primary",
                              isActive ? "opacity-100" : "opacity-0",
                            )}
                          />
                        </CommandItem>
                      )
                    })}
                  </CommandGroup>
                </>
              )}
            </CommandList>

            {active?.is_system && (
            <div className="border-t p-1">
              <button
                type="button"
                onClick={handleCreate}
                className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm font-medium text-primary transition-colors hover:bg-accent"
              >
                <Plus className="size-4" />
                Create New Tenant
              </button>
            </div>
            )}
          </Command>
        </PopoverContent>
      </Popover>

      <CreateTenantDialog open={createOpen} onOpenChange={setCreateOpen} />

      <ConfirmationDialog
        open={switchTarget !== null}
        onOpenChange={(open) => { if (!open) setSwitchTarget(null) }}
        onConfirm={handleConfirmSwitch}
        title="Switch tenant"
        description={
          switchTarget
            ? `You will be logged out and redirected to the login page for "${switchTarget.name}".`
            : "You will be logged out and redirected to the tenant login page."
        }
        confirmText="Switch & Logout"
        variant="destructive"
        showWarning
        warningMessage="Switching tenants will log you out of your current session."
      />
    </div>
  )
}
