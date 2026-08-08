import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { TenantSwitcher } from "@/components/navigation/TenantSwitcher"
import { FeatureSearch } from "@/components/navigation/FeatureSearch"
import { CreateMenu } from "@/components/navigation/CreateMenu"
import { useAppSelector } from "@/store/hooks"
import { logout } from "@/services/api/auth"
import { resolveBrandingLogoUrl } from "@/utils/branding"
import { openIdentityAccount } from "@/lib/identityAccountUrl"
import {
  BookOpen,
  ChevronDown,
  Code2,
  ExternalLink,
  HelpCircle,
  LogOut,
  MessageSquare,
  User,
} from "lucide-react"

const resourceLinks = [
  { title: "Documentation", icon: BookOpen, href: "#" },
  { title: "API Reference", icon: Code2, href: "#" },
  { title: "Community", icon: MessageSquare, href: "#" },
]

export function AppTopNav() {
  const profile = useAppSelector((state) => state.auth.profile)
  const branding = useAppSelector((state) => state.tenant.currentTenant?.branding)
  const tenantIdentityUrl = useAppSelector((state) => state.tenant.identityUrl)

  const displayName = profile?.display_name || profile?.email || "User"
  const initials = displayName.slice(0, 2).toUpperCase()
  const logoLabel = branding?.logo_label || branding?.company_name || "Maintainerd-IAM"
  const logoDetail = branding?.logo_detail
  const showLogoLabel = branding?.show_logo_label ?? true
  const logoSrc = resolveBrandingLogoUrl(branding?.logo_url) ?? "/logo.png"

  return (
    <header
      data-console-top-panel
      className="fixed inset-x-0 top-0 z-30 flex h-14 items-center border-b border-[#1e293b] bg-[#0f172a] px-4 text-white sm:px-6"
    >
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <SidebarTrigger
          data-console-top-control
          className="size-10 bg-white/5 text-slate-300 hover:bg-white/10 hover:text-white active:!bg-white/15 active:!text-white"
        />
        <img src={logoSrc} alt={logoLabel} className="h-7 w-auto shrink-0" />
        {showLogoLabel && (
          <span className="min-w-0 hidden md:inline">
            <div
              data-console-top-logo-label
              className={`font-semibold leading-none text-white ${logoDetail ? "text-sm" : "text-lg"}`}
            >
              {logoLabel}
            </div>
            {logoDetail && (
              <div className="mt-1 truncate text-[11px] leading-none text-slate-400">
                {logoDetail}
              </div>
            )}
          </span>
        )}
        <div className="hidden items-center gap-2 sm:flex ml-6">
          <TenantSwitcher />
          <FeatureSearch className="w-80" />
        </div>
      </div>

      <div className="ml-3 flex shrink-0 items-center gap-1.5">
        <CreateMenu />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              data-console-top-control
              variant="ghost"
              size="icon"
              aria-label="Help & resources"
              className="bg-white/5 text-slate-300 hover:bg-white/10 hover:text-white active:!bg-white/15 active:!text-white data-[state=open]:!bg-white/15 data-[state=open]:!text-white"
            >
              <HelpCircle className="size-5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-48" align="end">
            <DropdownMenuLabel className="font-normal text-muted-foreground">
              Resources
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            {resourceLinks.map((link) => (
              <DropdownMenuItem key={link.title} asChild className="cursor-pointer">
                <a href={link.href} target="_blank" rel="noopener noreferrer">
                  <link.icon className="mr-2 h-4 w-4" />
                  {link.title}
                </a>
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              data-console-top-profile-trigger
              variant="ghost"
              className="flex items-center gap-2 bg-white/5 px-2 text-white hover:bg-white/10 hover:text-white active:!bg-white/15 active:!text-white data-[state=open]:!bg-white/15 data-[state=open]:!text-white"
            >
              <Avatar className="h-8 w-8 shrink-0">
                <AvatarImage src={undefined} alt={displayName} />
                <AvatarFallback className="bg-slate-700 text-xs text-white">
                  {initials}
                </AvatarFallback>
              </Avatar>
              <span className="hidden max-w-40 truncate text-sm font-medium lg:inline">{displayName}</span>
              <ChevronDown className="hidden h-4 w-4 text-slate-400 sm:block" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-56" align="end">
            <DropdownMenuLabel className="font-normal">
              <div className="flex flex-col space-y-1">
                <p className="text-sm font-medium leading-none">{displayName}</p>
                <p className="text-xs leading-none text-muted-foreground">
                  {profile?.email || ""}
                </p>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            {/* Self-service lives in the identity app, not here. One
                implementation instead of two that drift. */}
            <DropdownMenuItem
              className="cursor-pointer"
              onClick={() => openIdentityAccount(tenantIdentityUrl)}
            >
              <User className="mr-2 h-4 w-4" />
              My account
              <ExternalLink className="ml-auto h-3.5 w-3.5 text-muted-foreground" />
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="cursor-pointer" onClick={() => logout()}>
              <LogOut className="mr-2 h-4 w-4" />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
