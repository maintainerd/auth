/**
 * Notifier (email/SMS delivery) hooks
 */

import { useQuery } from "@tanstack/react-query"
import { fetchEmailConfigStatus } from "@/services/api/notifier"

export const notifierKeys = {
  all: ["notifier"] as const,
  emailStatus: () => [...notifierKeys.all, "email", "status"] as const,
}

/** Whether email delivery is configured. `retry: false` so a permission/no-config
 *  case resolves fast and callers can treat it as "unknown" without blocking. */
export function useEmailConfigStatus() {
  return useQuery({
    queryKey: notifierKeys.emailStatus(),
    queryFn: fetchEmailConfigStatus,
    retry: false,
  })
}
