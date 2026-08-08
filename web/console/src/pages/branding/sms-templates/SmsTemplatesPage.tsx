import { SmsTemplateListing } from "./components/SmsTemplateListing"
import { PageHeader } from "@/components/layout"

export default function SmsTemplatesPage({ standalone = true }: { standalone?: boolean }) {
  if (!standalone) return <SmsTemplateListing tableInCard />

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="SMS Templates"
        description="Review and edit the system-managed SMS templates used for authentication, notifications, and platform communications."
      />
      <SmsTemplateListing tableInCard />
    </div>
  )
}
