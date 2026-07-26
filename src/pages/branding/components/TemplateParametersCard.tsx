import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

interface TemplateParametersCardProps {
  description: string
  parametersDoc?: string | null
}

export function TemplateParametersCard({ description, parametersDoc }: TemplateParametersCardProps) {
  const rows = parseParameterRows(parametersDoc)

  if (rows.length === 0) return null

  return (
    <Card>
      <CardHeader>
        <CardTitle>Template Parameters</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto rounded-md border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-2 text-left font-medium">Parameter</th>
                <th className="px-4 py-2 text-left font-medium">Description</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.parameter} className="border-b last:border-0">
                  <td className="px-4 py-2 font-mono text-xs">{row.parameter}</td>
                  <td className="px-4 py-2 text-muted-foreground">{row.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}

function parseParameterRows(parametersDoc?: string | null) {
  if (!parametersDoc) return []

  return parametersDoc
    .split("\n")
    .filter((line) => line.startsWith("|") && !line.includes("---"))
    .slice(1)
    .map((line) => line.split("|").map((cell) => cell.trim()).filter(Boolean))
    .filter((cells) => cells.length >= 2)
    .map(([parameter, description]) => ({
      parameter: parameter.replace(/`/g, ""),
      description,
    }))
}
