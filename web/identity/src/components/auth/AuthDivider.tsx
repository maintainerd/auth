/**
 * Labelled rule separating groups of sign-in options.
 *
 * Matches the console's login-template preview: two hairlines flanking a small
 * muted label with `gap-3`, and deliberately NOT uppercased — the preview
 * renders the label verbatim, so upper-casing here would drift from it.
 */
type Props = {
  label: string
}

export default function AuthDivider({ label }: Props) {
  return (
    <div className="flex items-center gap-3">
      <span className="h-px flex-1 bg-border" />
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="h-px flex-1 bg-border" />
    </div>
  )
}
