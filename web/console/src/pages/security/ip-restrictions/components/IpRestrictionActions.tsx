import { Edit, Trash2, Play, Pause, type LucideIcon } from "lucide-react"
import { RowActions, type RowActionItem } from "@/components/data-table"
import {
  useDeleteIpRestrictionRule,
  useUpdateIpRestrictionRuleStatus,
} from "@/hooks/useIpRestrictionRules"
import { useToast } from "@/hooks/useToast"
import type { IpRestrictionRule } from "@/services/api/ip-restriction-rules/types"

interface IpRestrictionActionsProps {
  rule: IpRestrictionRule
  onEdit: (rule: IpRestrictionRule) => void
}

interface StatusAction {
  status: "active" | "inactive"
  label: string
  title: string
  description: string
  icon: LucideIcon
}

const STATUS_TRANSITIONS: Record<"active" | "inactive", StatusAction[]> = {
  active: [
    {
      status: "inactive",
      label: "Deactivate Rule",
      title: "Deactivate IP Rule",
      description:
        "Are you sure you want to deactivate this IP rule? It will no longer be enforced.",
      icon: Pause,
    },
  ],
  inactive: [
    {
      status: "active",
      label: "Activate Rule",
      title: "Activate IP Rule",
      description:
        "Are you sure you want to activate this IP rule? It will be enforced immediately.",
      icon: Play,
    },
  ],
}

export function IpRestrictionActions({ rule, onEdit }: IpRestrictionActionsProps) {
  const { showSuccess, showError } = useToast()
  const updateStatusMutation = useUpdateIpRestrictionRuleStatus()
  const deleteRuleMutation = useDeleteIpRestrictionRule()

  const changeStatus = async (status: "active" | "inactive") => {
    try {
      await updateStatusMutation.mutateAsync({
        id: rule.ipRestrictionRuleId,
        data: { status },
      })
      showSuccess(`IP rule ${status === "active" ? "activated" : "deactivated"} successfully`)
    } catch (error) {
      showError(error)
    }
  }

  const items: RowActionItem[] = [
    {
      key: "edit",
      label: "Edit Rule",
      icon: Edit,
      onSelect: () => onEdit(rule),
    },
    ...STATUS_TRANSITIONS[rule.status].map(
      (action): RowActionItem => ({
        key: `status-${action.status}`,
        label: action.label,
        icon: action.icon,
        // Deactivate takes the rule out of enforcement → destructive (red
        // confirm). Activate is restorative → default.
        destructive: action.status !== "active",
        onSelect: () => changeStatus(action.status),
        confirm: {
          title: action.title,
          description: action.description,
          confirmText: action.label,
        },
      }),
    ),
    {
      key: "delete",
      label: "Delete Rule",
      icon: Trash2,
      destructive: true,
      separatorBefore: true,
      onSelect: async () => {
        try {
          await deleteRuleMutation.mutateAsync(rule.ipRestrictionRuleId)
          showSuccess("IP rule deleted successfully")
        } catch (error) {
          showError(error)
        }
      },
      confirm: {
        title: "Delete IP Rule",
        description:
          "This will permanently delete this IP restriction rule. This action cannot be undone.",
        destructive: true,
        itemName: rule.ipAddress,
      },
    },
  ]

  return <RowActions items={items} />
}
