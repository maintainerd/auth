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
import { CreateMenu } from "@/components/navigation/CreateMenu"
import { useAppSelector } from "@/store/hooks"
import { logout } from "@/services/api/auth"
import {
  BookOpen,
  ChevronDown,
  Code2,
  HelpCircle,
  LogOut,
  MessageSquare,
  Settings,
  User,
} from "lucide-react"
import { useNavigate } from "react-router-dom"

const resourceLinks = [
  { title: "Documentation", icon: BookOpen, href: "#" },
  { title: "API Reference", icon: Code2, href: "#" },
  { title: "Community", icon: MessageSquare, href: "#" },
]

export function AppTopNav() {
  const navigate = useNavigate()
  const profile = useAppSelector((state) => state.auth.profile)

  const displayName = profile?.display_name || profile?.email || "User"
  const initials = displayName.slice(0, 2).toUpperCase()

  return (
    <header className="fixed inset-x-0 top-0 z-30 flex h-14 items-center border-b border-[#1e293b] bg-[#0f172a] px-4 text-white sm:px-6">
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <SidebarTrigger className="size-10 text-slate-300 hover:bg-white/10 hover:text-white active:!bg-white/15 active:!text-white" />
        <img src="/logo.png" alt="Maintainerd" className="h-7 w-auto shrink-0" />
        <span className="min-w-0">
          <div className="text-base font-semibold leading-none text-white" style={{ fontFamily: "system-ui, sans-serif" }}>
            Maintainerd-IAM
          </div>
        </span>
        <TenantSwitcher className="ml-4 hidden w-48 sm:block" size="compact" />
      </div>

      <div className="ml-3 flex shrink-0 items-center gap-1.5">
        <CreateMenu />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Help & resources"
              className="text-slate-300 hover:bg-white/10 hover:text-white active:!bg-white/15 active:!text-white data-[state=open]:!bg-white/15 data-[state=open]:!text-white"
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
              variant="ghost"
              className="flex items-center gap-2 px-2 text-white hover:bg-white/10 hover:text-white active:!bg-white/15 active:!text-white data-[state=open]:!bg-white/15 data-[state=open]:!text-white"
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
            <DropdownMenuItem
              className="cursor-pointer"
              onClick={() => navigate(`/account/profile`)}
            >
              <User className="mr-2 h-4 w-4" />
              Profile
            </DropdownMenuItem>
            <DropdownMenuItem
              className="cursor-pointer"
              onClick={() => navigate(`/account/settings`)}
            >
              <Settings className="mr-2 h-4 w-4" />
              Settings
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
