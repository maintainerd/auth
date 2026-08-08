import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { TemplateParametersCard } from "@/pages/branding/components/TemplateParametersCard"
import type { SmsTemplate } from "@/services/api/sms-templates/types"

interface SmsTemplateContentProps {
  template: SmsTemplate
}

export function SmsTemplateContent({ template }: SmsTemplateContentProps) {
  return (
    <div className="space-y-6">
      <TemplateParametersCard
        parametersDoc={template.parametersDoc}
        description="Variables available in the message content. They are replaced with actual values when the SMS is sent."
      />

      <Card>
        <CardHeader>
          <CardTitle>Message Content</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <h3 className="text-sm font-medium text-muted-foreground mb-2">Message</h3>
              <div className="rounded-md border bg-muted/50 p-4">
                <p className="text-sm whitespace-pre-wrap">{template.message}</p>
              </div>
            </div>

            <div>
              <h3 className="text-sm font-medium text-muted-foreground mb-2">Character Count</h3>
              <p className="text-sm">{template.message.length} characters</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
