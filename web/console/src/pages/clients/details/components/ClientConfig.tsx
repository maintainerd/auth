import { Settings } from "lucide-react"
import { InformationCard } from "@/components/card"
import { EmptyState, ListSkeleton } from "@/components/details"
import { useClientConfig } from "@/hooks/useClients"
import { formatClientConfigValue, getClientConfigSections } from "../../clientConfig"

interface ClientConfigProps {
  clientId: string
}

export function ClientConfig({ clientId }: ClientConfigProps) {
  const { data: configData, isLoading, isError } = useClientConfig(clientId)

  const config = configData ?? {}
  const sections = getClientConfigSections(config)

  return (
    <InformationCard
      title="OAuth & Token Configuration"
      description="Standard OAuth, OIDC, token, URI, CORS, and client security settings."
      icon={Settings}
    >
      {isLoading && <ListSkeleton />}

      {isError && (
        <p className="py-8 text-center text-sm text-destructive">Failed to load configuration</p>
      )}

      {configData && sections.length > 0 && (
        <div className="space-y-6">
          {sections.map((section, sectionIndex) => (
            <section key={section.title} className="space-y-4">
              {sectionIndex > 0 && <div className="border-t" />}
              <div className="space-y-0.5">
                <h4 className="text-sm font-semibold">{section.title}</h4>
                <p className="text-sm text-muted-foreground">{section.description}</p>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                {section.entries.map(([key, value]) => (
                  <div key={key} className="space-y-1">
                    <p className="font-mono text-sm font-medium text-muted-foreground">{key}</p>
                    <p className="break-all rounded bg-muted px-2 py-1.5 font-mono text-sm">
                      {formatClientConfigValue(value) || "—"}
                    </p>
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>
      )}

      {configData && sections.length === 0 && (
        <EmptyState
          icon={Settings}
          title="No configuration"
          description="This client has no standard OAuth, token, URI, CORS, or security configuration stored yet."
        />
      )}
    </InformationCard>
  )
}
