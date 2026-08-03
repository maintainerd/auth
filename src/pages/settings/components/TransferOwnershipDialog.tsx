import { useState, useEffect, useMemo } from "react"
import { Search, ArrowRightLeft, User, AlertTriangle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useTransferTenantOwnership, useTenantMembers } from "@/hooks/useTenantMembers"
import { useToast } from "@/hooks/useToast"
import type { TenantMember } from "@/services/api/tenants/members"

interface TransferOwnershipDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  tenantId: string
  currentOwner: TenantMember
}

export function TransferOwnershipDialog({ open, onOpenChange, tenantId, currentOwner }: TransferOwnershipDialogProps) {
  const { showSuccess, showError } = useToast()
  const [selectedMemberId, setSelectedMemberId] = useState<string>("")
  const [searchQuery, setSearchQuery] = useState("")

  const transferMutation = useTransferTenantOwnership(tenantId)

  const { data: membersData } = useTenantMembers(tenantId, {
    page: 1,
    limit: 1000,
  })

  useEffect(() => {
    if (!open) {
      setSelectedMemberId("")
      setSearchQuery("")
    }
  }, [open])

  const handleSubmit = async () => {
    if (!selectedMemberId) {
      showError("Please select a member to transfer ownership to")
      return
    }

    try {
      await transferMutation.mutateAsync(selectedMemberId)
      showSuccess("Ownership transferred successfully")
      onOpenChange(false)
    } catch (error) {
      showError(error)
    }
  }

  const isLoading = transferMutation.isPending

  const eligibleMembers = useMemo(() => {
    const rows = membersData?.data?.rows ?? []
    return rows.filter((m: TenantMember) => m.role !== 'owner')
  }, [membersData?.data?.rows])

  const filteredMembers = useMemo(() => {
    if (!searchQuery) return eligibleMembers
    return eligibleMembers.filter((member: TenantMember) =>
      member.user.fullname.toLowerCase().includes(searchQuery.toLowerCase()) ||
      member.user.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
      member.user.username.toLowerCase().includes(searchQuery.toLowerCase())
    )
  }, [eligibleMembers, searchQuery])

  const selectedMember = eligibleMembers.find((m: TenantMember) => m.tenant_member_id === selectedMemberId)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Transfer Ownership</DialogTitle>
          <DialogDescription>
            Transfer ownership of this tenant to another member. The current owner will be demoted to a regular member.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>This action cannot be easily undone</AlertTitle>
            <AlertDescription>
              <strong>{currentOwner.user.fullname}</strong> will lose owner privileges and become a regular member.
              The new owner will receive full administrative access including the super-admin role.
            </AlertDescription>
          </Alert>

          <div className="space-y-3">
            <Label>
              Select New Owner <span className="text-destructive">*</span>
            </Label>

            <div className="relative">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search members by name or email..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-8"
              />
            </div>

            <div data-md-checkbox-sub-container className="border rounded-lg max-h-[300px] overflow-y-auto">
              {eligibleMembers.length === 0 && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  No other members available. Add members to the tenant before transferring ownership.
                </div>
              )}

              {eligibleMembers.length > 0 && filteredMembers.length === 0 && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  No members found matching your search
                </div>
              )}

              {filteredMembers.length > 0 && (
                <div className="divide-y">
                  {filteredMembers.map((member: TenantMember) => (
                    <div
                      key={member.tenant_member_id}
                      className={`flex items-start gap-3 p-3 hover:bg-accent/50 transition-colors cursor-pointer ${
                        selectedMemberId === member.tenant_member_id ? 'bg-accent' : ''
                      }`}
                      onClick={() => setSelectedMemberId(member.tenant_member_id)}
                    >
                      <Checkbox
                        id={member.tenant_member_id}
                        checked={selectedMemberId === member.tenant_member_id}
                        onCheckedChange={(checked) => {
                          setSelectedMemberId(checked ? member.tenant_member_id : "")
                        }}
                        disabled={isLoading}
                      />
                      <label
                        htmlFor={member.tenant_member_id}
                        className="flex-1 cursor-pointer"
                      >
                        <div className="flex items-center gap-2 mb-1">
                          <User className="h-4 w-4 text-muted-foreground" />
                          <span className="font-medium">{member.user.fullname}</span>
                          <Badge variant="secondary" className="text-xs capitalize">
                            {member.role}
                          </Badge>
                        </div>
                        <div className="text-sm text-muted-foreground">{member.user.email}</div>
                        {member.user.username && member.user.username !== member.user.email && (
                          <div className="text-xs text-muted-foreground mt-0.5">
                            @{member.user.username}
                          </div>
                        )}
                      </label>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {selectedMember && (
              <p className="text-sm text-muted-foreground">
                Ownership will be transferred to <strong>{selectedMember.user.fullname}</strong>
              </p>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleSubmit}
            disabled={!selectedMemberId || isLoading}
            className="gap-2"
          >
            {isLoading ? (
              <>Transferring...</>
            ) : (
              <>
                <ArrowRightLeft className="h-4 w-4" />
                Transfer Ownership
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
