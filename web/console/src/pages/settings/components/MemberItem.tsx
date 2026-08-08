import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { User, Shield, Trash2, MoreHorizontal, ArrowRightLeft } from "lucide-react"
import { ListingItemCard, ListingItemMeta } from "@/components/details"
import { format } from "date-fns"
import type { TenantMember } from "@/services/api/tenants/members"

interface MemberItemProps {
  member: TenantMember
  onUpdateRole?: () => void
  onDelete?: (memberId: string, memberName: string) => void
  onTransferOwnership?: () => void
}

export function MemberItem({ member, onUpdateRole, onDelete, onTransferOwnership }: MemberItemProps) {
  return (
    <ListingItemCard
      icon={User}
      action={
        (onUpdateRole || onDelete || onTransferOwnership) && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button data-md-icon-action-button variant="ghost" size="icon-sm" className="p-0">
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {onTransferOwnership && (
                <DropdownMenuItem onClick={onTransferOwnership}>
                  <ArrowRightLeft className="mr-2 h-4 w-4" />
                  Transfer Ownership
                </DropdownMenuItem>
              )}
              {onTransferOwnership && (onUpdateRole || onDelete) && <DropdownMenuSeparator />}
              {onUpdateRole && (
                <DropdownMenuItem onClick={onUpdateRole}>
                  <Shield className="mr-2 h-4 w-4" />
                  Update Role
                </DropdownMenuItem>
              )}
              {onUpdateRole && onDelete && <DropdownMenuSeparator />}
              {onDelete && (
                <DropdownMenuItem
                  onClick={() => onDelete(member.tenant_member_id, member.user.fullname)}
                  className="text-destructive"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Remove Member
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )
      }
    >
      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-sm font-medium">{member.user.fullname}</p>
          <Badge
            variant={member.role === 'owner' ? 'default' : 'secondary'}
            className="text-xs capitalize"
          >
            {member.role}
          </Badge>
          {member.user.status === 'active' && (
            <Badge variant="outline" className="text-xs">
              Active
            </Badge>
          )}
        </div>
        <p className="text-sm text-muted-foreground">{member.user.email}</p>
        <ListingItemMeta>
          {member.user.username && (
            <span>@{member.user.username}</span>
          )}
          {member.user.phone && (
            <span>{member.user.phone}</span>
          )}
          <span>Added: {format(new Date(member.created_at), "MMM d, yyyy")}</span>
        </ListingItemMeta>
      </div>
    </ListingItemCard>
  )
}
