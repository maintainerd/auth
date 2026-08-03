import { describe, expect, it } from "vitest"

import {
  AUTH_UI_TEMPLATE_IDS,
  AUTH_UI_TEMPLATES,
  authUiTemplatePresentationFromMetadata,
  authUiTemplatePresentationMetadata,
  authUiTemplateOptions,
} from "./authUiTemplates"

describe("auth UI templates", () => {
  it("exposes every configured template as a selectable option", () => {
    expect(AUTH_UI_TEMPLATES.map((template) => template.id)).toEqual(AUTH_UI_TEMPLATE_IDS)
    expect(authUiTemplateOptions().map((option) => option.value)).toEqual(AUTH_UI_TEMPLATE_IDS)
  })

  it("keeps template ids unique", () => {
    const ids = AUTH_UI_TEMPLATES.map((template) => template.id)

    expect(new Set(ids).size).toBe(ids.length)
  })

  it("round-trips hosted auth logo detail metadata", () => {
    const presentation = authUiTemplatePresentationFromMetadata({
      login_form_logo_detail: "  Identity access  ",
    })

    expect(presentation.logoDetail).toBe("Identity access")
    expect(authUiTemplatePresentationMetadata({}, presentation)).toMatchObject({
      login_form_logo_detail: "Identity access",
    })
  })
})
