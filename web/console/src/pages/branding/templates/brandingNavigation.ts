export const BRANDING_THEMES_LIST_URL = "/branding?tab=themes"

export const BRANDING_THEMES_BACK_STATE = {
  from: BRANDING_THEMES_LIST_URL,
  backLabel: "Back to Themes",
}

export const brandingDetailState = (brandingId: string) => ({
  from: `/branding/templates/${brandingId}`,
  backLabel: "Back to Branding Details",
})
