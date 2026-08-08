import { EmailTemplateListing } from "./components/EmailTemplateListing"
import { PageHeader } from "@/components/layout"

export default function EmailTemplatesPage({ standalone = true }: { standalone?: boolean }) {
  if (!standalone) return <EmailTemplateListing tableInCard />

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Email Templates"
        description="Review and edit the system-managed email templates used for authentication, notifications, and platform communications."
      />
      <EmailTemplateListing tableInCard />
    </div>
  )
}
