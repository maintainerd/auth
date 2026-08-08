import { MoreHorizontal, Edit, Trash2, CheckCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import type { UserProfile } from "@/services/api/users/types"

interface ProfileActionsProps {
  profile: UserProfile
  onEdit: (profile: UserProfile) => void
  onDelete: (profile: UserProfile) => void
  onSetAsDefault: (profile: UserProfile) => void
}

export function ProfileActions({ profile, onEdit, onDelete, onSetAsDefault }: ProfileActionsProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button data-md-icon-action-button variant="ghost" className="h-8 w-8 p-0">
          <span className="sr-only">Open menu</span>
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => onEdit(profile)}>
          <Edit className="mr-2 h-4 w-4" />
          Edit Profile
        </DropdownMenuItem>
        
        {!profile.is_default && (
          <DropdownMenuItem onClick={() => onSetAsDefault(profile)}>
            <CheckCircle className="mr-2 h-4 w-4" />
            Set As Default
          </DropdownMenuItem>
        )}

        {!profile.is_default && <DropdownMenuSeparator />}
        
        {!profile.is_default && (
          <DropdownMenuItem
            onClick={() => onDelete(profile)}
            className="text-destructive focus:text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete Profile
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
