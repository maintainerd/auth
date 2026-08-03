import { useState } from "react"
import { Ban, CheckCircle2, ChevronDown, ChevronRight, FileText, KeyRound, Target } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { InformationCard } from "@/components/card"
import { EmptyState } from "@/components/details"
import { cn } from "@/lib/utils"
import type { PolicyStatement } from "@/services/api/policies/types"

interface PolicyStatementsTabProps {
  documentVersion: string
  statements: PolicyStatement[]
}

type TokenKind = "action" | "resource"

export function PolicyStatementsTab({ documentVersion, statements }: PolicyStatementsTabProps) {
  const [expandedStatements, setExpandedStatements] = useState<Set<number>>(() => new Set())

  const toggleStatement = (index: number) => {
    const nextExpanded = new Set(expandedStatements)
    if (nextExpanded.has(index)) nextExpanded.delete(index)
    else nextExpanded.add(index)
    setExpandedStatements(nextExpanded)
  }

  if (statements.length === 0) {
    return (
      <InformationCard
        title="Policy Statements"
        description="Statements define which actions are allowed or denied against matching resources."
        icon={FileText}
        action={<DocumentBadge version={documentVersion} />}
      >
        <EmptyState
          icon={FileText}
          title="No statements"
          description="This policy document does not define any allow or deny statements."
        />
      </InformationCard>
    )
  }

  return (
    <InformationCard
      title="Policy Statements"
      description="Review the access rules in evaluation order. A deny statement overrides matching allows; otherwise access requires a matching allow."
      icon={FileText}
      action={<DocumentBadge version={documentVersion} />}
    >
      <div className="space-y-3">
        {statements.map((statement, index) => (
          <PolicyStatementItem
            key={`${statement.effect}-${index}`}
            statement={statement}
            index={index}
            isExpanded={expandedStatements.has(index)}
            onToggle={() => toggleStatement(index)}
          />
        ))}
      </div>
    </InformationCard>
  )
}

function DocumentBadge({ version }: { version: string }) {
  return (
    <Badge variant="outline" className="font-mono text-xs">
      Document {version}
    </Badge>
  )
}

interface PolicyStatementItemProps {
  statement: PolicyStatement
  index: number
  isExpanded: boolean
  onToggle: () => void
}

function PolicyStatementItem({ statement, index, isExpanded, onToggle }: PolicyStatementItemProps) {
  const isAllow = statement.effect === "allow"
  const EffectIcon = isAllow ? CheckCircle2 : Ban

  return (
    <div data-md-listing-item className="rounded-lg border p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <div
            className={cn(
              "flex size-10 shrink-0 items-center justify-center rounded-lg",
              isAllow ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-700",
            )}
          >
            <EffectIcon className="size-5" />
          </div>
          <div className="min-w-0 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-semibold">Statement {String(index + 1).padStart(2, "0")}</span>
              <EffectBadge effect={statement.effect} />
              {!isAllow && (
                <Badge variant="outline" className="border-red-200 bg-red-50 text-xs text-red-700">
                  Overrides allows
                </Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              {isAllow ? "Grants" : "Blocks"} {statement.action.length} action
              {statement.action.length === 1 ? "" : "s"} against {statement.resource.length} resource pattern
              {statement.resource.length === 1 ? "" : "s"}.
            </p>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
              <span>{countWildcards(statement.action)} wildcard actions</span>
              <span>{countWildcards(statement.resource)} wildcard resources</span>
            </div>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          <Badge variant="secondary" className="text-xs">
            {statement.action.length + statement.resource.length} patterns
          </Badge>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-8 p-0"
            aria-expanded={isExpanded}
            onClick={onToggle}
          >
            <span className="sr-only">{isExpanded ? "Collapse statement" : "Expand statement"}</span>
            {isExpanded ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
          </Button>
        </div>
      </div>

      {isExpanded && (
        <div className="mt-4 border-t pt-4">
          <div className="mb-3">
            <h4 className="text-sm font-semibold">Matched Patterns</h4>
            <p className="text-sm text-muted-foreground">
              Both an action and a resource pattern must match for this statement to apply.
            </p>
          </div>

          <div className="space-y-4">
            <PatternGroup
              icon={Target}
              title="Resources"
              description="Resource patterns the operations apply to."
              tokens={statement.resource}
              kind="resource"
            />
            <PatternGroup
              icon={KeyRound}
              title="Actions"
              description="Operations matched by this statement."
              tokens={statement.action}
              kind="action"
            />
          </div>
        </div>
      )}
    </div>
  )
}

function EffectBadge({ effect }: { effect: PolicyStatement["effect"] }) {
  const isAllow = effect === "allow"

  return (
    <Badge
      variant="outline"
      className={cn(
        "gap-1 capitalize",
        isAllow ? "border-emerald-200 bg-emerald-50 text-emerald-700" : "border-red-200 bg-red-50 text-red-700",
      )}
    >
      {isAllow ? <CheckCircle2 className="size-3" /> : <Ban className="size-3" />}
      {effect}
    </Badge>
  )
}

function PatternGroup({
  icon: Icon,
  title,
  description,
  tokens,
  kind,
}: {
  icon: typeof KeyRound
  title: string
  description: string
  tokens: string[]
  kind: TokenKind
}) {
  return (
    <div className="space-y-3">
      <div>
        <div className="flex items-center gap-2">
          <Icon className="size-4 text-muted-foreground" />
          <h5 className="text-sm font-semibold">{title}</h5>
          <Badge variant="secondary" className="text-xs">
            {tokens.length}
          </Badge>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      </div>

      <div className="space-y-2">
        {tokens.map((token, index) => (
          <PatternRow key={`${token}-${index}`} token={token} kind={kind} />
        ))}
      </div>
    </div>
  )
}

function PatternRow({ token, kind }: { token: string; kind: TokenKind }) {
  const value = token.trim()
  const isGlobalWildcard = value === "*"
  const isWildcard = value.includes("*")

  return (
    <div data-md-listing-nested className="flex items-start justify-between gap-3 rounded-md border bg-background p-3">
      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="break-all font-mono text-sm font-medium">{value}</span>
          {isWildcard && (
            <Badge
              variant="outline"
              className={cn(
                "text-xs",
                isGlobalWildcard
                  ? "border-amber-200 bg-amber-50 text-amber-700"
                  : "border-sky-200 bg-sky-50 text-sky-700",
              )}
            >
              {isGlobalWildcard ? "Global wildcard" : "Wildcard"}
            </Badge>
          )}
        </div>
        <p className="text-sm text-muted-foreground">{describePattern(value, kind)}</p>
      </div>
    </div>
  )
}

function describePattern(value: string, kind: TokenKind): string {
  if (value === "*") return kind === "action" ? "Matches every action." : "Matches every resource."
  if (value.includes("*")) return "Wildcard pattern. Literal text around the wildcard must still match."

  const separator = value.indexOf(":")
  if (separator === -1) return kind === "action" ? "Exact action match." : "Exact resource match."

  const namespace = value.slice(0, separator)
  const target = value.slice(separator + 1)
  return kind === "action"
    ? `Namespace ${namespace}; operation ${target}.`
    : `Namespace ${namespace}; resource ${target}.`
}

function countWildcards(tokens: string[]) {
  return tokens.filter((token) => token.includes("*")).length
}
