import type { AuditLogEntry } from "@/services/api/audit-log/types"

export function formatAuditActor(entry: AuditLogEntry) {
  if (entry.actor_user_id != null) {
    return {
      label: entry.actor_user_name || `User #${entry.actor_user_id}`,
      context: entry.actor_user_name ? `User #${entry.actor_user_id}` : "User",
    }
  }

  if (entry.actor_client_id != null) {
    return {
      label: entry.actor_client_name || `Client #${entry.actor_client_id}`,
      context: entry.actor_client_name ? `Client #${entry.actor_client_id}` : "Client",
    }
  }

  return { label: "System", context: "No actor" }
}
