import { useState } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { User, Calendar, Mail, Globe, Clock, Languages, Check, Plus, type LucideIcon } from "lucide-react"
import { safeFormat } from "@/lib/formatDate"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { DataTablePagination, usePaginationTable } from "@/components/data-table"
import { ConfirmationDialog, DeleteConfirmationDialog } from "@/components/dialog"
import { useUserProfiles, useDeleteUserProfile, useSetUserProfileAsDefault } from "@/hooks/useUsers"
import { useToast } from "@/hooks/useToast"
import type { UserProfile } from "@/services/api/users/types"
import { ProfileFormDialog } from "./ProfileFormDialog"
import { ProfileActions } from "./ProfileActions"
import { type PaginationState } from "@tanstack/react-table"

interface UserProfilesProps {
  userId: string
}

export function UserProfiles({ userId }: UserProfilesProps) {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [isDeleteOpen, setIsDeleteOpen] = useState(false)
  const [isSetDefaultOpen, setIsSetDefaultOpen] = useState(false)
  const [selectedProfile, setSelectedProfile] = useState<UserProfile | undefined>()

  const { showSuccess, showError } = useToast()
  const deleteProfileMutation = useDeleteUserProfile()
  const setAsDefaultMutation = useSetUserProfileAsDefault()

  const { data, isLoading, isError } = useUserProfiles(userId, {
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sort_by: 'created_at',
    sort_order: 'desc',
  })

  const handleCreateProfile = () => {
    setSelectedProfile(undefined)
    setIsFormOpen(true)
  }

  const handleEditProfile = (profile: UserProfile) => {
    setSelectedProfile(profile)
    setIsFormOpen(true)
  }

  const handleDeleteProfile = (profile: UserProfile) => {
    setSelectedProfile(profile)
    setIsDeleteOpen(true)
  }

  const handleSetAsDefault = async (profile: UserProfile) => {
    setSelectedProfile(profile)
    setIsSetDefaultOpen(true)
  }

  const handleConfirmSetAsDefault = async () => {
    if (!selectedProfile) return

    try {
      await setAsDefaultMutation.mutateAsync({
        userId,
        profileId: selectedProfile.profile_id,
      })
      showSuccess("Profile set as default successfully")
      setIsSetDefaultOpen(false)
    } catch (error) {
      showError(error)
    }
  }

  const handleConfirmDelete = async () => {
    if (!selectedProfile) return

    try {
      await deleteProfileMutation.mutateAsync({
        userId,
        profileId: selectedProfile.profile_id,
      })
      showSuccess("Profile deleted successfully")
      setIsDeleteOpen(false)
    } catch (error) {
      showError(error)
    }
  }

  const table = usePaginationTable({
    pagination,
    onPaginationChange: setPagination,
    pageCount: data?.total_pages ?? 0,
  })

  const formatDate = (dateString: string) => safeFormat(dateString, 'MMM dd, yyyy')

  // Name fields are optional on the backend, so compose without interpolating
  // "undefined". Prefer the display name; fall back to the composed legal name.
  const legalName = (p: UserProfile) =>
    [p.first_name, p.middle_name, p.last_name].filter(Boolean).join(" ").trim()
  const profileName = (p: UserProfile) => p.display_name?.trim() || legalName(p) || "Unnamed profile"
  const initials = (name: string) =>
    name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]?.toUpperCase()).join("")

  return (
    <>
      <InformationCard
        title="User Profiles"
        description="User profile information including personal details and preferences"
        icon={User}
        action={
          <Button onClick={handleCreateProfile} size="sm" className="gap-2">
            <Plus className="h-4 w-4" />
            Create Profile
          </Button>
        }
      >
      <div className="space-y-4">
        {isLoading && <ListSkeleton />}

          {isError && (
            <p className="py-8 text-center text-sm text-destructive">Failed to load profiles</p>
          )}

          {data && data.rows.length === 0 && (
            <EmptyState
              icon={User}
              title="No profiles"
              description="This user has no profiles yet. Create one to capture personal details and preferences."
            />
          )}

          {data && data.rows.length > 0 && (
            <div className="space-y-4">
              {data.rows.map((profile: UserProfile) => {
                const name = profileName(profile)
                const legal = legalName(profile)
                const subParts = [
                  profile.display_name?.trim() && legal && legal !== name ? legal : null,
                  profile.gender ? profile.gender.charAt(0).toUpperCase() + profile.gender.slice(1) : null,
                ].filter(Boolean)

                const attributes: { icon: LucideIcon; value: string }[] = []
                if (profile.email) attributes.push({ icon: Mail, value: profile.email })
                if (profile.birthdate) attributes.push({ icon: Calendar, value: formatDate(profile.birthdate) })
                if (profile.timezone) attributes.push({ icon: Clock, value: profile.timezone })
                if (profile.language) attributes.push({ icon: Languages, value: profile.language })

                return (
                  <div
                    data-md-listing-item
                    key={profile.profile_id}
                    className={`space-y-4 rounded-lg border p-4 ${
                      profile.is_default
                        ? "border-blue-200 bg-blue-50"
                        : "border-border/60 bg-muted/20"
                    }`}
                  >
                    {/* Header */}
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex items-start gap-3">
                        {profile.profile_url ? (
                          <img
                            src={profile.profile_url}
                            alt={name}
                            className="size-10 shrink-0 rounded-full object-cover ring-2 ring-background"
                          />
                        ) : (
                          <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 text-sm font-semibold text-white shadow-sm shadow-blue-500/20">
                            {initials(name)}
                          </div>
                        )}
                        <div>
                          <div className="flex flex-wrap items-center gap-2">
                            <h4 className="text-sm font-medium">{name}</h4>
                            {profile.is_default && (
                              <Badge variant="secondary" className="gap-1 font-normal">
                                <Check className="size-3" />
                                Default
                              </Badge>
                            )}
                          </div>
                          {subParts.length > 0 && (
                            <p className="text-sm text-muted-foreground">{subParts.join(" · ")}</p>
                          )}
                        </div>
                      </div>
                      <ProfileActions
                        profile={profile}
                        onEdit={handleEditProfile}
                        onDelete={handleDeleteProfile}
                        onSetAsDefault={handleSetAsDefault}
                      />
                    </div>

                    {/* Attributes */}
                    {attributes.length > 0 && (
                      <div className="grid grid-cols-1 gap-x-6 gap-y-2.5 sm:grid-cols-2">
                        {attributes.map(({ icon: Icon, value }) => (
                          <div key={value} className="flex items-center gap-2 text-sm">
                            <Icon className="size-4 shrink-0 text-muted-foreground" />
                            <span className="truncate">{value}</span>
                          </div>
                        ))}
                      </div>
                    )}

                    {/* Metadata */}
                    {profile.metadata && Object.keys(profile.metadata).length > 0 && (
                      <div className="divide-y rounded-md border bg-background text-sm">
                        {Object.entries(profile.metadata).map(([key, value]) => (
                          <div key={key} className="flex items-center justify-between gap-4 px-3 py-2">
                            <span className="font-medium break-all">{key}</span>
                            <span className="font-mono text-muted-foreground break-all">{String(value)}</span>
                          </div>
                        ))}
                      </div>
                    )}

                    {/* Footer */}
                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-border/60 pt-3 text-xs text-muted-foreground">
                      <span className="inline-flex items-center gap-1">
                        <Calendar className="size-3" />
                        Created {formatDate(profile.created_at)}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        <Globe className="size-3" />
                        Updated {formatDate(profile.updated_at)}
                      </span>
                    </div>
                  </div>
                )
              })}
            </div>
          )}

        {/* Pagination */}
        {data && data.total > 0 && (
          <div className="border-t pt-4">
            <DataTablePagination table={table} rowCount={data.total} />
          </div>
        )}
      </div>
    </InformationCard>

    <ProfileFormDialog
      open={isFormOpen}
      onOpenChange={setIsFormOpen}
      userId={userId}
      profile={selectedProfile}
    />

    <ConfirmationDialog
      open={isSetDefaultOpen}
      onOpenChange={setIsSetDefaultOpen}
      onConfirm={handleConfirmSetAsDefault}
      title="Set Profile as Default"
      description={`Are you sure you want to set "${selectedProfile ? profileName(selectedProfile) : ""}" as the default profile?`}
      confirmText="Set as Default"
      isLoading={setAsDefaultMutation.isPending}
    />

    <DeleteConfirmationDialog
      open={isDeleteOpen}
      onOpenChange={setIsDeleteOpen}
      onConfirm={handleConfirmDelete}
      title="Delete Profile"
      description="This permanently deletes this profile. This action cannot be undone."
      itemName={selectedProfile ? profileName(selectedProfile) : ""}
      isDeleting={deleteProfileMutation.isPending}
    />
  </>
  )
}
