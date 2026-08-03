import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { User, Plus, Star, Trash2, Pencil, MoreHorizontal } from 'lucide-react'
import AccountLayout from '@/components/layout/AccountLayout'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useToast } from '@/hooks/useToast'
import {
  fetchProfiles,
  setDefaultProfile,
  deleteProfile,
  type UserProfile,
} from '@/services/api/account'

function profileName(p: UserProfile): string {
  return (
    p.display_name ||
    [p.first_name, p.last_name].filter(Boolean).join(' ') ||
    'Unnamed profile'
  )
}

function fmtDate(value?: string | null): string | undefined {
  if (!value) return undefined
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? undefined : d.toLocaleDateString()
}

function ViewField({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="flex flex-col gap-1">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="text-sm">{value || <span className="text-muted-foreground">Not set</span>}</p>
    </div>
  )
}

function ProfileActions({
  profile, onEdit, onSetDefault, onDelete,
}: {
  profile: UserProfile
  onEdit: () => void
  onSetDefault: () => void
  onDelete: () => void
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" className="size-8 p-0">
          <span className="sr-only">Open menu</span>
          <MoreHorizontal className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onEdit}>
          <Pencil className="mr-2 size-4" /> Edit
        </DropdownMenuItem>
        {!profile.is_default && (
          <DropdownMenuItem onClick={onSetDefault}>
            <Star className="mr-2 size-4" /> Set as default
          </DropdownMenuItem>
        )}
        {!profile.is_default && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={onDelete}>
              <Trash2 className="mr-2 size-4" /> Delete
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export default function AccountProfilesPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const { showError, showSuccess } = useToast()
  const [pendingDelete, setPendingDelete] = useState<UserProfile | null>(null)

  const { data: profiles = [], isLoading } = useQuery({
    queryKey: ['account', 'profiles'],
    queryFn: fetchProfiles,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['account', 'profiles'] })

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => setDefaultProfile(id),
    onSuccess: () => { showSuccess('Default profile updated'); invalidate() },
    onError: (err) => showError(err, 'Could not set default profile'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteProfile(id),
    onSuccess: () => {
      showSuccess('Profile deleted')
      setPendingDelete(null)
      invalidate()
    },
    onError: (err) => { showError(err, 'Could not delete profile'); setPendingDelete(null) },
  })

  // Feature the default profile on top; if the account has no profile flagged
  // as default yet, fall back to the first one so the summary card still shows.
  const defaultProfile = profiles.find((p: UserProfile) => p.is_default) ?? profiles[0]

  return (
    <AccountLayout title="Profile">
      <div className="space-y-6">
        {/* Default profile — full details */}
        {!isLoading && defaultProfile && (
          <Card>
            <CardHeader>
              <div className="flex items-center gap-4">
                <div className="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted">
                  {defaultProfile.profile_url ? (
                    <img src={defaultProfile.profile_url} alt="" className="size-16 rounded-full object-cover" />
                  ) : (
                    <span className="text-lg font-semibold text-muted-foreground">
                      {profileName(defaultProfile).slice(0, 2).toUpperCase()}
                    </span>
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <CardTitle className="truncate text-lg">{profileName(defaultProfile)}</CardTitle>
                    {defaultProfile.is_default && (
                      <span className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                        Default
                      </span>
                    )}
                  </div>
                  <p className="mt-0.5 text-sm text-muted-foreground">
                    {defaultProfile.is_default ? 'Your default profile' : 'Your profile'}
                  </p>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  className="shrink-0 gap-1.5"
                  onClick={() => navigate(`/account/profile/${defaultProfile.profile_id}/edit`)}
                >
                  <Pencil className="size-3.5" /> Edit
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
                <ViewField label="Display name" value={defaultProfile.display_name} />
                <ViewField label="First name" value={defaultProfile.first_name} />
                <ViewField label="Middle name" value={defaultProfile.middle_name} />
                <ViewField label="Last name" value={defaultProfile.last_name} />
                <ViewField label="Email" value={defaultProfile.email} />
                <ViewField label="Gender" value={defaultProfile.gender} />
                <ViewField label="Date of birth" value={fmtDate(defaultProfile.birthdate)} />
                <ViewField label="Timezone" value={defaultProfile.timezone} />
                <ViewField label="Language" value={defaultProfile.language} />
                <ViewField label="Added" value={fmtDate(defaultProfile.created_at)} />
              </div>
            </CardContent>
          </Card>
        )}

        {/* All profiles */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-base">Profiles</CardTitle>
            <Button
              size="sm"
              variant="outline"
              className="gap-1.5"
              onClick={() => navigate('/account/profile/new')}
            >
              <Plus className="size-3.5" /> Add profile
            </Button>
          </CardHeader>
          <CardContent>
            {isLoading && <p className="text-sm text-muted-foreground">Loading profiles…</p>}
            {!isLoading && profiles.length === 0 && (
              <div className="flex flex-col items-center gap-3 py-8 text-center">
                <User className="size-8 text-muted-foreground/50" />
                <p className="text-sm text-muted-foreground">No profiles yet</p>
              </div>
            )}

            <div className="space-y-2">
              {profiles.map((profile: UserProfile) => (
                <div
                  key={profile.profile_id}
                  data-md-listing-item
                  className="flex items-center justify-between gap-3 rounded-lg border p-3"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <div data-md-listing-icon className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted">
                      {profile.profile_url ? (
                        <img src={profile.profile_url} alt="" className="size-9 rounded-full object-cover" />
                      ) : (
                        <User className="size-4 text-muted-foreground" />
                      )}
                    </div>
                    <div className="flex min-w-0 items-center gap-2">
                      <p className="truncate text-sm font-medium">{profileName(profile)}</p>
                      {profile.is_default && (
                        <span className="rounded-full bg-primary/10 px-1.5 py-0.5 text-xs font-medium text-primary">
                          Default
                        </span>
                      )}
                    </div>
                  </div>
                  <ProfileActions
                    profile={profile}
                    onEdit={() => navigate(`/account/profile/${profile.profile_id}/edit`)}
                    onSetDefault={() => setDefaultMutation.mutate(profile.profile_id)}
                    onDelete={() => setPendingDelete(profile)}
                  />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      <Dialog open={pendingDelete !== null} onOpenChange={(open) => { if (!open) setPendingDelete(null) }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete profile</DialogTitle>
            <DialogDescription>
              {pendingDelete
                ? `"${profileName(pendingDelete)}" will be permanently removed. This can't be undone.`
                : ''}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setPendingDelete(null)}>Cancel</Button>
            <Button
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => pendingDelete && deleteMutation.mutate(pendingDelete.profile_id)}
            >
              {deleteMutation.isPending ? 'Deleting…' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </AccountLayout>
  )
}
