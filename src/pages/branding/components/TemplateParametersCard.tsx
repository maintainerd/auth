import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

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
        <div data-md-table-shell className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader className="[&_tr]:border-b [&_tr]:bg-muted">
              <TableRow>
                <TableHead className="h-9 text-xs">Parameter</TableHead>
                <TableHead className="h-9 text-xs">Description</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.parameter}>
                  <TableCell className="font-mono text-xs">{row.parameter}</TableCell>
                  <TableCell className="text-muted-foreground">{row.description}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
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
