export const WEBHOOKS_LIST_URL = "/events?tab=webhooks"

export const WEBHOOKS_BACK_STATE = {
  from: WEBHOOKS_LIST_URL,
  backLabel: "Back to Webhooks",
}

export const webhookDetailState = (webhookId: string) => ({
  from: `/webhooks/${webhookId}`,
  backLabel: "Back to Webhook Details",
})
