import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { TemplateParametersCard } from "@/pages/branding/components/TemplateParametersCard"
import type { EmailTemplate } from "@/services/api/email-templates/types"

interface EmailTemplateContentProps {
  template: EmailTemplate
}

export function EmailTemplateContent({ template }: EmailTemplateContentProps) {
  return (
    <div className="space-y-6">
      <TemplateParametersCard
        parametersDoc={template.parametersDoc}
        description="Variables available in the HTML and plain text content. They are replaced with actual values when the email is sent."
      />

      <Card>
        <CardHeader>
          <CardTitle>Email Content</CardTitle>
          <CardDescription>Subject line, HTML body, and plain text fallback</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-1">
            <p className="text-sm font-medium">Subject</p>
            <p className="text-sm text-muted-foreground">{template.subject}</p>
          </div>

          <div className="space-y-1">
            <p className="text-sm font-medium">HTML Content</p>
            <pre className="p-4 bg-muted rounded-md overflow-x-auto text-xs">
              <code>{template.bodyHtml}</code>
            </pre>
          </div>

          <div className="space-y-1">
            <p className="text-sm font-medium">Plain Text Content</p>
            <pre className="p-4 bg-muted rounded-md overflow-x-auto text-xs whitespace-pre-wrap">
              <code>{template.bodyPlain}</code>
            </pre>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
