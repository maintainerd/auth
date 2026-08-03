import { useState } from "react"
import { useNavigate } from "react-router-dom"
import {
  CalendarDays,
  Edit,
  FileText,
  MoreVertical,
  Pause,
  Play,
  Shield,
  Trash2,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useDeletePolicy, useUpdatePolicyStatus } from "@/hooks/usePolicies"
import { useToast } from "@/hooks/useToast"
import { ConfirmationDialog, DeleteConfirmationDialog } from "@/components/dialog"
import { DetailHeaderCard, StatusBadge, type DetailAttribute } from "@/components/details"
import { SystemBadge } from "@/components/badges"
import { safeFormat } from "@/lib/formatDate"
import type { PolicyDetail, PolicyStatus } from "@/services/api/policies/types"

interface PolicyHeaderProps {
  policy: PolicyDetail
  policyId: string
  afterDeleteTo?: string
}

interface StatusAction {
  status: PolicyStatus
  label: string
  title: string
  description: string
  icon: typeof Play
}

const STATUS_ACTIONS: Record<PolicyStatus, StatusAction> = {
  active: {
    status: "active",
    label: "Activate Policy",
    title: "Activate Policy",
    description: "Are you sure you want to activate this policy?",
    icon: Play,
  },
  inactive: {
    status: "inactive",
    label: "Deactivate Policy",
    title: "Deactivate Policy",
    description:
      "Are you sure you want to deactivate this policy? Services using it will no longer be governed by its statements.",
    icon: Pause,
  },
}

export function PolicyHeader({ policy, policyId, afterDeleteTo }: PolicyHeaderProps) {
  const navigate = useNavigate()
  const { showSuccess, showError } = useToast()
  const deletePolicyMutation = useDeletePolicy()
  const updateStatusMutation = useUpdatePolicyStatus()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [statusAction, setStatusAction] = useState<StatusAction | null>(null)

  const handleDelete = async () => {
    try {
      await deletePolicyMutation.mutateAsync(policyId)
      showSuccess("Policy deleted successfully")
      navigate(afterDeleteTo ?? `/policies`)
    } catch (error) {
      showError(error)
    }
  }

  const handleStatusChange = async () => {
    if (!statusAction) return
    try {
      await updateStatusMutation.mutateAsync({ policyId, data: { status: statusAction.status } })
      showSuccess(`Policy status updated to ${statusAction.status}`)
    } catch (error) {
      showError(error)
    } finally {
      setStatusAction(null)
    }
  }

  // Availability mirrors the backend rules: system policies cannot be edited,
  // change status, or be deleted.
  const statusActions = policy.is_system
    ? []
    : Object.values(STATUS_ACTIONS).filter((action) => action.status !== policy.status)

  const attributes: DetailAttribute[] = [
    {
      icon: Shield,
      label: "Type",
      value: policy.is_system ? "System Policy" : "Custom Policy",
    },
    {
      icon: FileText,
      label: "Policy Version",
      value: <span className="font-mono text-xs">{policy.version}</span>,
    },
    {
      icon: FileText,
      label: "Document Version",
      value: <span className="font-mono text-xs">{policy.document.version}</span>,
    },
    {
      icon: CalendarDays,
      label: "Created",
      value: safeFormat(policy.created_at, "PP"),
    },
    {
      icon: CalendarDays,
      label: "Last updated",
      value: safeFormat(policy.updated_at, "PP"),
    },
  ]

  return (
    <>
      <DetailHeaderCard
        leading={
          <div className="flex size-14 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <Shield className="size-6" />
          </div>
        }
        title={policy.name}
        badge={
          <div className="flex items-center gap-2">
            <StatusBadge status={policy.status} />
            <SystemBadge isSystem={policy.is_system} />
          </div>
        }
        subtitle={policy.description}
        attributes={attributes}
        actions={
          !policy.is_system && (
            <>
              <Button
                data-md-details-edit-button
                variant="outline"
                size="sm"
                className="h-9 gap-2"
                onClick={() =>
                  navigate(`/policies/${policyId}/edit`, {
                    state: { from: `/policies/${policyId}`, backLabel: "Back to Policy Details" },
                  })
                }
              >
                <Edit className="size-4" />
                Edit
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button data-md-details-menu-button variant="outline" size="sm" className="h-9 w-9 p-0">
                    <span className="sr-only">Open actions</span>
                    <MoreVertical className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  {statusActions.map((action) => (
                    <DropdownMenuItem
                      key={action.status}
                      onClick={() => setStatusAction(action)}
                      className={
                        action.status !== "active"
                          ? "text-destructive focus:text-destructive"
                          : undefined
                      }
                    >
                      <action.icon className="mr-2 size-4" />
                      {action.label}
                    </DropdownMenuItem>
                  ))}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => setShowDeleteDialog(true)}
                    className="text-destructive focus:text-destructive"
                  >
                    <Trash2 className="mr-2 size-4" />
                    Delete Policy
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          )
        }
      />

      <ConfirmationDialog
        open={!!statusAction}
        onOpenChange={(open) => { if (!open) setStatusAction(null) }}
        onConfirm={handleStatusChange}
        title={statusAction?.title ?? ""}
        description={statusAction?.description ?? ""}
        // Deactivate stops the policy from governing services → red confirm.
        // Activate is restorative → normal confirm.
        variant={statusAction && statusAction.status !== "active" ? "destructive" : "default"}
        confirmText={statusAction?.label}
        isLoading={updateStatusMutation.isPending}
      />

      <DeleteConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={handleDelete}
        title="Delete Policy"
        description="This action cannot be undone. This will permanently delete the policy and all associated data."
        itemName={policy.name}
        isDeleting={deletePolicyMutation.isPending}
      />
    </>
  )
}
