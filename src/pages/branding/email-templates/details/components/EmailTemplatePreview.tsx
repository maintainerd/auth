import DOMPurify from "dompurify"
import { useMemo } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { EmailTemplate } from "@/services/api/email-templates/types"

interface EmailTemplatePreviewProps {
  template: EmailTemplate
}

export function EmailTemplatePreview({ template }: EmailTemplatePreviewProps) {
  // Template bodies are operator-authored HTML but are still untrusted at render
  // time (a lower-privileged author, or a tampered value, could inject script).
  // Sanitise before injecting so the preview cannot execute arbitrary markup.
  const safeHtml = useMemo(
    () => DOMPurify.sanitize(template.bodyHtml ?? "", { USE_PROFILES: { html: true } }),
    [template.bodyHtml],
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle>HTML Preview</CardTitle>
        <CardDescription>Preview of the email template as it will appear to recipients</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="border rounded-md p-4 bg-white">
          <div
            className="prose max-w-none"
            // nosemgrep: react-dangerouslysetinnerhtml -- content sanitized with DOMPurify above
            dangerouslySetInnerHTML={{ __html: safeHtml }}
          />
        </div>
      </CardContent>
    </Card>
  )
}
