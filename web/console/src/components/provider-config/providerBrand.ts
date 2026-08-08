/**
 * Provider brand marks — maps each provider to a brand glyph + colour.
 *
 * Uses the brand icons shipped with @tabler/icons-react (already a project
 * dependency) so no extra icon library is needed. Monochrome brands (GitHub, X)
 * use `text-foreground` so they stay legible in light and dark themes.
 * Internal/unmapped providers fall back to a neutral shield.
 */

import {
  IconBrandAuth0,
  IconBrandAws,
  IconBrandAzure,
  IconBrandFacebook,
  IconBrandGithub,
  IconBrandGitlab,
  IconBrandGoogle,
  IconBrandLinkedin,
  IconBrandX,
} from "@tabler/icons-react"
import { ShieldCheck } from "lucide-react"

export type GlyphComponent = React.ComponentType<{ className?: string }>

export interface ProviderBrand {
  Icon: GlyphComponent
  /** Tailwind text-color class for the glyph (brand colour). */
  color: string
}

const PROVIDER_BRAND: Record<string, ProviderBrand> = {
  maintainerd: { Icon: ShieldCheck, color: "text-foreground" },
  cognito: { Icon: IconBrandAws, color: "text-[#FF9900]" },
  auth0: { Icon: IconBrandAuth0, color: "text-[#EB5424]" },
  microsoft: { Icon: IconBrandAzure, color: "text-[#0078D4]" },
  google: { Icon: IconBrandGoogle, color: "text-[#4285F4]" },
  github: { Icon: IconBrandGithub, color: "text-foreground" },
  gitlab: { Icon: IconBrandGitlab, color: "text-[#FC6D26]" },
  facebook: { Icon: IconBrandFacebook, color: "text-[#1877F2]" },
  linkedin: { Icon: IconBrandLinkedin, color: "text-[#0A66C2]" },
  twitter: { Icon: IconBrandX, color: "text-foreground" },
}

const FALLBACK_BRAND: ProviderBrand = { Icon: ShieldCheck, color: "text-muted-foreground" }

export function getProviderBrand(provider: string): ProviderBrand {
  return PROVIDER_BRAND[provider] ?? FALLBACK_BRAND
}
