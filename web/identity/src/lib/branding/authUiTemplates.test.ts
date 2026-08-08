import { describe, expect, it } from 'vitest'
import {
  authUiTemplateIdFromMetadata,
  authUiTemplatePresentationFromMetadata,
  safeAuthTemplateImageUrl,
} from './authUiTemplates'

describe('authUiTemplates', () => {
  it('uses an explicit auth UI template before layout fallbacks', () => {
    expect(authUiTemplateIdFromMetadata({ auth_ui_template: 'editorial-cover' }, 'split')).toBe('editorial-cover')
    expect(authUiTemplateIdFromMetadata(undefined, 'split')).toBe('split-showcase')
    expect(authUiTemplateIdFromMetadata(undefined, 'full_page')).toBe('stepper-flow')
    expect(authUiTemplateIdFromMetadata({ auth_ui_template: 'unknown' }, 'centered')).toBe('centered-card')
  })

  it('reads login template presentation metadata with defaults', () => {
    expect(authUiTemplatePresentationFromMetadata({
      login_form_logo_placement: 'above-form',
      split_showcase_visual_style: 'security-radar',
      split_showcase_panel_title: 'Workspace access',
      split_showcase_panel_subtitle: 'Use your verified identity.',
      split_showcase_image_url: 'https://example.test/cover.jpg',
      login_form_logo_detail: 'Identity access',
    })).toEqual({
      logoPlacement: 'above-form',
      logoDetail: 'Identity access',
      splitShowcaseVisualStyle: 'security-radar',
      splitShowcaseTitle: 'Workspace access',
      splitShowcaseSubtitle: 'Use your verified identity.',
      splitShowcaseImageUrl: 'https://example.test/cover.jpg',
    })
  })

  it('limits visual image URLs to simple app-relative or web URLs', () => {
    expect(safeAuthTemplateImageUrl('/assets/cover.jpg')).toBe('/assets/cover.jpg')
    expect(safeAuthTemplateImageUrl('https://example.test/cover.jpg')).toBe('https://example.test/cover.jpg')
    expect(safeAuthTemplateImageUrl('javascript:alert(1)')).toBeUndefined()
    expect(safeAuthTemplateImageUrl('https://example.test/<bad>.jpg')).toBeUndefined()
  })
})
