import type { AuditLogEntry } from "@/services/api/audit-log/types"

export function formatAuditActor(entry: AuditLogEntry) {
  // Actors are identified by their resolved (denormalized) name only — internal
  // integer PKs are never sent to the client, so the presence of a name is the
  // user-vs-client-vs-system discriminator.
  if (entry.actor_user_name != null) {
    return { label: entry.actor_user_name, context: "User" }
  }

  if (entry.actor_client_name != null) {
    return { label: entry.actor_client_name, context: "Client" }
  }

  return { label: "System", context: "No actor" }
}
