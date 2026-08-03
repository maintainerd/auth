import { useNavigate } from "react-router-dom"
import type { LucideIcon } from "lucide-react"
import {
  AppWindow,
  Boxes,
  Building2,
  ChevronDown,
  ClipboardList,
  KeyRound,
  Mail,
  Palette,
  Plus,
  ScrollText,
  Server,
  Shield,
  User,
  Webhook,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type CreateItem = { label: string; route: string; icon: LucideIcon }
type CreateGroup = { label: string; items: CreateItem[] }

// Every "create" form across the app, grouped by feature area (mirrors the
// sidebar taxonomy). This is the single global entry point for creating anything.
const CREATE_GROUPS: CreateGroup[] = [
  {
    label: "Identity & Access",
    items: [
      { label: "User", route: "/users/create", icon: User },
      { label: "Role", route: "/roles/create", icon: Shield },
      { label: "Invitation", route: "/invites/create", icon: Mail },
      { label: "Identity Provider", route: "/providers/identity/create", icon: KeyRound },
      { label: "Registration Flow", route: "/registration-flows/create", icon: ClipboardList },
    ],
  },
  {
    label: "Applications & APIs",
    items: [
      { label: "Client", route: "/clients/create", icon: AppWindow },
      { label: "Service", route: "/services/create", icon: Boxes },
      { label: "API", route: "/apis/create", icon: Server },
      { label: "Policy", route: "/policies/create", icon: ScrollText },
    ],
  },
  {
    label: "Branding",
    items: [{ label: "Theme", route: "/branding/templates/create", icon: Palette }],
  },
  {
    label: "Operations",
    items: [{ label: "Webhook", route: "/webhooks/create", icon: Webhook }],
  },
  {
    label: "Administration",
    items: [{ label: "Tenant", route: "/tenants/create", icon: Building2 }],
  },
]

export function CreateMenu() {
  const navigate = useNavigate()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button data-console-top-create-button>
          <Plus className="h-4 w-4" />
          Create
          <ChevronDown className="h-4 w-4 opacity-80" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="max-h-[75vh] w-56 overflow-y-auto">
        {CREATE_GROUPS.map((group, i) => (
          <div key={group.label}>
            {i > 0 && <DropdownMenuSeparator />}
            <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
              {group.label}
            </DropdownMenuLabel>
            {group.items.map((item) => (
              <DropdownMenuItem
                key={item.route}
                className="cursor-pointer"
                onClick={() => navigate(item.route)}
              >
                <item.icon className="mr-2 h-4 w-4" />
                {item.label}
              </DropdownMenuItem>
            ))}
          </div>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
