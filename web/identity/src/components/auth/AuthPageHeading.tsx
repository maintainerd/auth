/**
 * The title/subtitle block at the top of every hosted auth page.
 *
 * Geometry mirrors the console's login-template preview (`space-y-1 text-center`,
 * `text-2xl font-semibold tracking-tight` over muted `text-sm`) so the branding
 * editor's preview and the live page agree.
 */
type Props = {
  title: string
  subtitle?: string
}

export default function AuthPageHeading({ title, subtitle }: Props) {
  return (
    <div className="space-y-1 text-center">
      <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
      {subtitle && <p className="text-sm text-muted-foreground">{subtitle}</p>}
    </div>
  )
}
