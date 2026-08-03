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
    <div data-md-listing-item className="flex items-start gap-3 py-3 border-b last:border-0">
      <div data-md-listing-icon className="mt-1">
        <User className="h-4 w-4 text-muted-foreground" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-medium">{member.user.fullname}</span>
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
        <p className="text-sm text-muted-foreground mb-2">{member.user.email}</p>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          {member.user.username && (
            <>
              <span>@{member.user.username}</span>
              <span>•</span>
            </>
          )}
          {member.user.phone && (
            <>
              <span>{member.user.phone}</span>
              <span>•</span>
            </>
          )}
          <span>Added: {format(new Date(member.created_at), "MMM d, yyyy")}</span>
        </div>
      </div>
      {(onUpdateRole || onDelete || onTransferOwnership) && (
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
      )}
    </div>
  )
}
