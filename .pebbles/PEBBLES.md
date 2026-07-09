# Pebbles — project knowledge graph

This `.pebbles/` folder is the machine-readable spec for this repository,
maintained with the Pebbles app. It is the **entry point for AI agents**:
read this file first, then explore the sections below as they bear on your
task. A developer prompting about one pebble expects you to also honor the
global context here — especially `rules/` and `architecture/`.

A **pebble** is one small unit of spec: a requirement, a rule, an API
contract, a data model, a DTO/structure, a workflow, a skill, or a piece of
knowledge. One pebble = one `.pebble` file (plain JSON inside; legacy
`.json` files are read too). **Never edit `.pebble` files directly** —
all mutations go through the `pebbles-cli` tool (path provided in your
session context; `pebbles-cli help` shows usage). It validates every
write and keeps the graph consistent; direct editor writes are denied.
Reading pebble files directly is fine and encouraged. A `.pebblesignore`
file at the PROJECT root (gitignore-style: `dir/`, `*.ext`, or substring
lines) tells the map what to skip. Folders inside a section are
free-form organization.

## Code graph — find code without grepping

`pebbles-cli` maintains a dependency graph of the whole project (imports +
symbols resolved to real files). Prefer it over grep — it is far cheaper
in tokens and points you straight at the right file:

- `map-symbol <name> [--kind class]` — where a symbol is DEFINED (file:line)
- `map-search <term>` — files/symbols matching a term, ranked by importance
- `map-file <path>` — a file's symbols, what it imports, what imports it
- `map-neighbors <path> --depth 2` — the connected file cluster
- `map-callers <symbol>` — every call site of a function/class, with the
  CALLING function attributed (disambiguated via each caller's imports)
- `refs <path>` — a pebble's connections BOTH ways: its outgoing refs and
  codeLinks, and every pebble/unit that references it (backlinks)

A pebble's `codeLinks[].path` doubles as a **map node id**: feed it to
`map-file` for instant symbols/imports/importers/calls — the fastest
analysis entry point for any linked spec.

Edges are TYPED: `import`, `dynamic-import` (lazy/code-split),
`reexport` (barrel files), and class-hierarchy `extends` /
`implements` / `mixin` — so the graph answers "what subclasses X"
as well as "what imports X". Then read only the files the graph names.

## Reverse mode — extracting spec FROM existing code

On an existing app the code is the source of truth and the pebbles start
empty: EXTRACT instead of implement. **A project map is required first**
(`pebbles-cli map`, or `--fresh` when `mapGeneratedAt` is stale). Then:
locate the implementation via the graph, read it, and fill pebbles via
`update`/`create` — models get their real fields (type/required/
primaryKey/refKey/default), structures their real shapes, workflows their real
flows (unit ids and node positions are auto-filled), contracts their
real endpoints. `link-code` every source file. Extracted spec gets
status `implemented` (it exists in code) — reserve `verified` for
behavior you actually proved. Works per-pebble or project-wide (start
from the map's `hubs` — the most-imported files are the domain core).

## codeLinks — binding spec to implementation

A pebble may carry `codeLinks: [{ id, path, symbol?, line?, note? }]` — the
actual files that implement it. Set them with
`pebbles-cli link-code <pebble> --file <rel> [--symbol NAME] [--line N]
[--note "why"]` (a feature often spans several files — link each). Next
time you work the pebble, its codeLinks tell you exactly where the code is.
Common flows:

- *Attach code to a spec*: read the pebble → locate its files via the graph
  → `link-code` each one.
- *Fill a model from code*: `map-symbol <Model> --kind class` → read that
  one file → `update` the model pebble's fields → `link-code` the file.
- *Make code follow the spec*: read the pebble → edit the real source files
  (never the `.pebble`) so they match.

## Sections

| Folder | Holds | Read it when… |
|---|---|---|
| `pebbles/` | requirements, tasks, checklists (any kind allowed); acceptance criteria conventionally live in `pebbles/scenarios/` | you need to know what is being built |
| `rules/` | prescriptive constraints — coding standards, architecture rules. These are **enforceable**: code that violates them is wrong | before writing or reviewing any code |
| `contracts/` | API endpoints, events, webhooks; payload shapes reference `structures/` rather than duplicating them | touching any API surface |
| `architecture/` | system topology — services, apps, databases, queues and their **allowed dependencies** | before any cross-module change |
| `findings/` | the APPEND-ONLY evidence log — everything with an outcome, not only bugs: bug, verified-working (pass evidence), spec-violation, security, performance, regression, tech-debt, ux, data, research, decision, assumption, measurement, incomplete, other. Create with `pebbles-cli add-finding <target> --title … [--unit <id>] --category … --observation … --evidence "file:line"` — it cross-links the report to the target pebble AND the specific unit automatically. NEVER update or delete a finding: same question with a new answer gets a NEW finding with `--supersedes <old>` (old file untouched); the chain's latest entry is current truth — check chains before trusting an old verdict | record what you discover (passes AND failures); read before re-investigating |
| `tests/` | test PLANS: one pebble per feature/suite, one checklist item per test case. Item text = the behavior verified ("rejects an expired token"); item `description` = Given/When/Then + what requirement it covers; item `refs` → the requirement/AC it proves; the pebble's `codeLinks` → the real test files. Statuses: `planned`=to write, `verified`=passing, `broken`=failing, `drift`=outdated — update them (set-status --unit) whenever you run or write tests | know what's covered; update statuses as tests run |
| `models/` | schemas / entities (tables) | touching persistence |
| `structures/` | DTOs, return shapes, objects/classes | shaping data |
| `workflows/` | application flow logic as flowcharts | changing how a flow behaves |
| `skills/` | structured definitions of what an agent can do | orchestrating agent work |
| `agents/` | agent definitions — role, behavior instructions, skills used, hard constraints | acting as (or spawning) a specific agent |
| `memory/` | durable memory from past sessions — context, summary, distilled facts, optional transcript | starting any session: load relevant facts before working |
| `knowledge/` | shared context and background; decision records (ADRs) in `knowledge/decisions/` | you need the "why" |

(`.chats/` holds per-pebble AI conversation history — not spec; ignore it.)

## File format

Every pebble file is JSON:

```json
{
  "version": 1,
  "id": "uuid",
  "name": "…",
  "type": "pebble | rule | contract | architecture | skill | knowledge | model | structure | workflow",
  "description": "…",
  "tags": [],
  "createdAt": "ISO-8601",
  "updatedAt": "ISO-8601",
  "content": { "kind": "…", … }
}
```

The pebble root may also carry `status` (overall implementation state),
`priority` (`critical | high | medium | low` — **order your work by it**),
`owner` (person/team responsible), `tags` (cross-cutting labels), and
`refs` (see **References** below). List items and flow nodes may carry
`priority` too.

**Not everything is a task.** Pebbles have natures, and the fields adapt:

- **Buildable** (requirements, tests, models, structures, contracts,
  workflows, architecture) — carry `priority`; statuses follow the
  implementation lifecycle below.
- **Normative** (rules) — carry `enforcement` instead of priority:
  `strict` (MUST — a violation marks the violating unit `broken` and
  warrants a finding) · `recommended` (SHOULD — violations surface as
  drift/warnings) · `optional` (MAY — advisory only). A rule's own status
  is just its lifecycle: `planned` = Draft, `verified` = Active,
  `deprecated` = Inactive. Violations are NEVER recorded on the rule —
  they go on the violating unit's status and into `findings/`. Rules are
  attached to other pebbles via refs.
- **Reference** (knowledge, memory) — a knowledge base, not work: no
  priority; status is freshness (`planned` = draft, `verified` = current,
  `drift` = outdated, `deprecated` = archived).
- **Operational** (skills, agents) — definitions that are usable or not;
  no spec↔code alignment: `planned` = draft, `verified` = active,
  `drift` = needs update, `deprecated` = retired.
- **Reports** (findings) — a case lifecycle: `broken` = open (red in the
  explorer until dealt with), `in-progress` = investigating, `verified` =
  resolved, `deprecated` = dismissed.
- **Tests** — verification: `planned` = to write, `verified` = passing,
  `broken` = failing, `drift` = outdated.

Each nature exposes only its own minimal status set — when setting a
status, use the values listed for that nature above.

Statuses live at BOTH levels — the pebble and each unit (checklist
item, table field, workflow node) — and they are the product: the user
reads them to know what is implemented and what is not. After touching
anything, set-status the units you touched, not just the pebble.
`query --unit-status <s>` finds pebbles by unit-level status and
returns the matching units.

On a model (DB table), `required: true` means NOT NULL — the app
displays this column inverted as a "Nullable" checkbox.

On `update`, `content` is replaced whole BUT unit state survives the
rewrite: units matched by id (or, when the patch has no id, by
name/text/label) keep their existing `id`, `status` and `description`
when the patch omits them. Explicit values always win. Still set
statuses explicitly when you know them — extraction from real code
should mark each unit `implemented`.

Table completeness is ENFORCED: `create`/`update` reject `fields`
content where any field lacks a `description` (what it is used for —
derive it from the code: comments, validations, call sites) or a
`status`. A table without field context is not a spec.

Foreign keys are FIRST-CLASS graph edges. When a field references
another model and that model's pebble exists, set
`refKey: { "path": "models/<target>.pebble", "field": "<field name>" }`
— the CLI resolves the field name to its id, fills `pebbleId`/`label`,
and auto-wires the graph refs on both the field and the pebble (visible
in Properties → Reference). Check for the target with `query`/`list`
before writing; an FK column without its refKey when the target exists
is an incomplete extraction. Nested record types work the same via
`typeRef: { "path": ... }`.

`content.kind` is one of:

- `"checklist"` — `{ "items": [{ "id", "text", "status", "description"?, "refs"? }] }` —
  `text` is the one-line title; `description` holds the full detail
- `"text"` — `{ "body": "markdown text" }`
- `"fields"` — `{ "fields": [{ "name", "type", "required", "default"?, "description", "status"?, "typeRef"?, "primaryKey"?, "refKey"?, "refs"? }] }` —
  a field's type may be ANOTHER table (nested record): `typeRef:
  { pebbleId, path }` links it and `type` carries that table's name —
  follow the path to expand the shape. MODEL tables are schemas: fields
  may carry `primaryKey: true` and `refKey: { pebbleId, path, fieldId,
  label }` — a reference (foreign) key to another model's field; follow
  it for join semantics. Every typeRef/refKey target is mirrored into the
  pebble's `refs` graph —
  fields carry a status too: a field defined here but absent from the real
  schema is `missing`; a mismatched one is `drift`
- `"flow"` — a workflow flowchart. Node fields: `id` (any stable
  string), `label`, `status`, `nodeType`, `description`? and `refs`?.
  **OMIT `x`/`y`** — positions are auto-filled and you MUST run
  `arrange` after writing (never hand-place). Edge fields: `id`,
  `source`, `target`, `label`? (the arm ANSWER shown on the edge —
  "yes"/"no"/a status code — this is where a condition's branch text
  goes), `sourceHandle`? (the physical outlet id `"l"`/`"r"`/`"b"` ONLY;
  NEVER put the answer here — it is a handle id, and a wrong value makes
  the edge vanish). Prefer `label` and OMIT `sourceHandle`. `nodeType`: `start | end | step | condition |
  subprocess | parallel | success | error | external | exception | global-exception |
  try | catch`.

  `try` `{` / `catch` `}` = a try/catch drawn as a bracket pair. A `try` has
  MANY inputs and exactly TWO outputs: one into the guarded BODY (any node,
  whose path must RETURN to the matching catch) and one straight to that
  `catch` (the spine). A `catch` has exactly TWO inputs (that spine + the
  body's last node) and exactly TWO outputs: the EXCEPTION line (always
  labeled "Exceptions", fixed — non-editable) and the CONTINUATION line (no
  label) to the next code. CLI-enforced; in the app a brace whose logic isn't
  met is outlined dashed-red.

  `external` = an EXTERNAL SYSTEM (another backend/service, a frontend, a
  queue/message bus, a partner/AI API, a webhook peer) — a process (many
  inputs, ONE output) that marks a crossing of the app boundary. Mark
  EVERY crossing, never buried in a step, and name the DIRECTION +
  COMMUNICATION TYPE (REST/webhook/gRPC/GraphQL/queue/WebSocket/…) + DATA
  in its description (ref the contract/structure). INBOUND (we receive — a
  request/webhook/gRPC/consumed queue message): put it at the entry, right
  after `start`, detailing the received payload — or, for a continuous
  injector, as a SOURCE with no incoming line pointing at the listener.
  OUTBOUND (we send — an HTTP client call, a queue publish, a frontend
  notify, a partner call): put it where it happens; if it's the last thing
  (push out, then done) connect it to an `end`.

  ERROR HANDLING by SCOPE — match the code's shape. A single-purpose guard
  (auth, input) stays inline (check → its own `error` → `end`).

  A GUARDED try/catch — the error-handling construct in ANY language, which
  the AI identifies itself (try/catch, try/except, begin/rescue, do/catch,
  defer+recover, an err-return guarded region, …), not by matching a keyword
  → the dedicated `try` `{` / `catch` `}` nodes (see the try/catch entry
  above). The `try` opens with TWO outputs (the BODY, whose
  path returns to the catch, + a SPINE straight to the catch); the BODY is
  modeled in full (an inner guard that throws keeps its OWN `condition` and a
  throw arm → an `error` "<what> — error thrown" → its own `end`; never
  collapse/drop inner guards). The `catch` closes with TWO inputs (the spine +
  the body's return) and TWO outputs: the EXCEPTION line labeled "Exceptions"
  (fixed) → the failure handling, and the CONTINUATION line (no label) → the
  code after the try/catch. Auto-arrange aligns the FIRST body node
  horizontally with the try (a step to the right, not a diagonal); the rest
  flows down between the braces.

  CAUGHT vs UNCAUGHT decides where the Exceptions line goes: a catch that
  HANDLES the error locally (rollback, log, recover, set-status, return a
  default) → those REAL processes as EXPLICIT nodes → `end`, self-contained,
  NO global-exception; a catch that RE-THROWS or delegates to the app's global
  handler → its local step(s), then an `exception` hand-off → `end` → a
  `global-exception` flow. Never one generic box; never invent a
  global-exception for a catch that fully handles its own error.

  `exception` / `global-exception` are the UNCAUGHT / ESCAPED path — an error
  NOT bounded by a local catch (a `throw`/error-return with no enclosing
  try/catch, an error propagating past all catches, a rethrow, or a
  middleware/framework filter). An `exception` node is a HAND-OFF marker → its
  OWN `end`; it is DUPLICATED like success/error (never shared, in-degree 1)
  and ALWAYS requires a `global-exception` flow (CLI-enforced) that holds the
  handler's REAL processes as EXPLICIT nodes (rollback, log, notify,
  mark-failed → usually a `condition` on error type → error terminals),
  laid out to the right of the main flow, NOT a separate pebble/subprocess.

  COMPLETENESS (any language): capture EVERY branch and exit — each
  conditional/switch-case (incl. default)/loop-exit/throw/return, each
  catch/except/rescue CLAUSE (a multi-clause handler is several handlers,
  each modeled distinctly — never merged), and each outcome. Before
  finishing, list the code's decision + exit points and confirm each maps to
  the graph; leave no path behind.

  DESCRIPTIONS (every node/unit, every pebble): write in TWO TIERS — (1)
  plain language first (what it is + how it works, readable by product owner/
  QA/non-engineer), then (2) a separate final paragraph of technical detail
  (file:line, symbols, types, rules). A description states ONLY what the
  thing IS + its detail — NEVER an issue, bug or verdict. Anything wrong is a
  FINDING, never a description. Be detailed: an external node carries its
  full contract (direction + comm type + complete payload shape). Findings
  follow the same two tiers (plain problem first, technical evidence last).

  RULES (all CLI-ENFORCED — a violating write is rejected): exactly ONE
  `start` (one output, no input) · `global-exception` is also a root (no
  input, one output) beside the main flow · `step`/`subprocess`/`external` have ≤1 output
  (branch with `condition`, fan out with `parallel`) · `condition`/
  `parallel` need ≥2 outputs · every path ends at an `end` (the sole
  sink) · every node reachable from a root (start, an external source, or
  a global-exception) · a `subprocess` node carries
  `subprocess: { pebbleId, path }` to a child workflow · an `exception` node
  REQUIRES a `global-exception` flow in the same pebble (the catch's processes
  live under it).
  OUTCOMES: `success`/`error`/`exception` are markers — success/error used
  ONLY where the code has a real success/failure signal (2xx/Ok/resolve;
  throw/Err/4xx-5xx), `exception` where a region is handed to a global
  handler; each has ONE input + ONE output into its OWN dedicated `end`,
  is NEVER shared (duplicate them, even same type), and a plain return is
  a plain `end`
  with no marker.
  FLOW FORWARD: the only backward/loop edge is a genuine PROCESS retry (a
  `step`/`subprocess`/`parallel` or a retry-`condition` looping back to
  re-process). Outcomes and ends (`success`/`error`/`exception`/rollback/
  `end`) ALWAYS move forward — never a back-edge to reuse a shared node.

  WORKED EXAMPLE — a full create (copy the shape; omit x/y; then arrange):
  ```
  pebbles-cli create workflows --title "Reset Password" --content '{
    "kind": "flow",
    "nodes": [
      {"id":"start","label":"POST /reset","status":"implemented","nodeType":"start","description":"AuthController::reset (app/Http/Controllers/AuthController.php:88)"},
      {"id":"valid","label":"Token valid?","status":"implemented","nodeType":"condition","description":"PasswordBroker::validateToken (…:41)"},
      {"id":"err","label":"422 Invalid token","status":"implemented","nodeType":"error","description":"throw ValidationException (…:44)"},
      {"id":"errEnd","label":"End","status":"implemented","nodeType":"end"},
      {"id":"reset","label":"Update password","status":"implemented","nodeType":"step","description":"User::forceFill+save (…:52)"},
      {"id":"ok","label":"200 OK","status":"implemented","nodeType":"success","description":"return response 200 (…:57)"},
      {"id":"okEnd","label":"End","status":"implemented","nodeType":"end"}
    ],
    "edges": [
      {"id":"e1","source":"start","target":"valid"},
      {"id":"e2","source":"valid","target":"err","label":"no"},
      {"id":"e3","source":"valid","target":"reset","label":"yes"},
      {"id":"e4","source":"err","target":"errEnd"},
      {"id":"e5","source":"reset","target":"ok"},
      {"id":"e6","source":"ok","target":"okEnd"}
    ]
  }'
  ```
  Then: `pebbles-cli arrange workflows/reset-password.pebble`. Update
  uses the same content shape via `update <path> --patch '{"content": …}'`.
- `"skill"` — a machine-first skill definition. Read the fields directly,
  no prose parsing needed:

Reference integrity is ENFORCED on create/update: a `refs`/`typeRef`/
`refKey`/`subprocess` entry whose target pebble doesn't exist (or whose
`refKey.fieldId` isn't a real field there) is dropped before the write
and reported in the result's `droppedRefs`. Create the target first,
then link.

  ```json
  {
    "kind": "skill",
    "purpose": "one line: what this skill does",
    "whenToUse": "trigger conditions — when an agent should apply it",
    "arguments": "what the invocation accepts, e.g. \"<ticket-id> [--draft]\"",
    "prerequisites": [{ "id", "text" }], // must be true BEFORE running
    "tools": "allowed tools, e.g. \"Read, Edit, Bash(git *)\" — empty = any",
    "steps": [{ "id", "text" }],        // ordered procedure
    "constraints": [{ "id", "text" }],  // hard rules: must / never
    "output": "what it produces + how to know it worked (definition of done)",
    "examples": [{ "id", "input", "output" }]  // few-shot pairs
  }
  ```

  When executing a skill: check `whenToUse` matches the situation, verify
  every `prerequisites` entry holds (stop and say so if one doesn't), stay
  within `tools` when set, follow `steps` in order, treat every
  `constraints` entry as non-negotiable, judge completion against
  `output`, and use `examples` as ground truth for the expected shape of
  the result.

- `"agent"` — an agent definition:

  ```json
  {
    "kind": "agent",
    "role": "one line: who this agent is",
    "instructions": "system-prompt-style behavior",
    "skills": [{ "id", "text" }],       // names of skill pebbles it uses
    "constraints": [{ "id", "text" }]   // hard behavioral rules
  }
  ```

- `"finding"` — an investigation result, with its resolution:

  ```json
  {
    "kind": "finding",
    "severity": "critical | high | medium | low | info",
    "category": "bug | spec-violation | security | performance | tech-debt | …",
    "state": "open | investigating | resolved | wont-fix | false-positive",
    "location": "where it lives — files, module, endpoint",
    "observation": "what was found (the problem)",
    "evidence": [{ "id", "text" }],       // file:line refs, logs, repro steps
    "impact": "why it matters — blast radius",
    "rootCause": "the underlying cause",
    "solution": "recommended resolution (prose)",
    "solutionSteps": [{ "id", "text" }]   // ordered actions to resolve it
  }
  ```

  When you investigate something, record the result as a finding here —
  don't let it evaporate into chat. Fill `evidence` with concrete proof
  and always propose a `solution`. Set `state` to `resolved` once fixed.

- `"memory"` — durable memory, portable between sessions and agents:

  ```json
  {
    "kind": "memory",
    "context": "where it came from: agent, session, date",
    "summary": "short digest",
    "facts": [{ "id", "text" }],   // distilled durable facts — load these first
    "transcript": "optional raw conversation (markdown)"
  }
  ```

  Read `facts` first; open `transcript` only when a fact needs its source.
  When you learn something durable during a session, offer to record it as
  a new memory pebble rather than letting it evaporate.

## Status — the spec↔code alignment scale

This graph is the **desired state**; the codebase is the **actual state**.
A `status` (on a pebble, list item, flow node, or table field) names the
relationship between the two:

| status | means | typically set by |
|---|---|---|
| `planned` | defined in the spec, not built yet | the author |
| `in-progress` | actively being implemented | the agent starting work |
| `implemented` | built, **not yet verified** against this spec | the agent after writing code |
| `verified` | confirmed matching the spec, **with evidence** | verification (agent or human) |
| `drift` | exists in code but no longer matches this spec — a reconciliation question | verification |
| `broken` | has a known bug — violates its spec; fix the code | verification (paired with a finding) |
| `missing` | expected in code but not found | verification |
| `deprecated` | no longer applies; kept for history | the author |

Rules for agents:

- **Never set `verified` without evidence** in the actual code — a file:line
  reference or a passing test. `implemented` is the honest state after
  writing code you haven't re-checked.
- When implementing a unit, walk it forward: `planned` → `in-progress` →
  `implemented`, and only → `verified` after checking the result.
- When **reconciling** spec vs code (audits, reviews), assign `drift` when
  the code differs from the spec and `missing` when the code lacks it —
  e.g. a model field defined here that doesn't exist in the real schema
  is `missing`; one whose type changed is `drift`.
- `drift` vs `broken`: drift asks "which side is right?" (the spec may be
  stale); broken means the code is unambiguously defective — fix the code.
- **When you find a bug or violation in a unit**: create a finding in
  `findings/` (with a reference back to the unit), set the unit's status
  to `broken` (bug) / `drift` (mismatch) / `missing` (absent), and when
  the finding is resolved, re-verify the unit before restoring `verified`.
- On `drift`, don't silently "fix" either side: surface it — the spec may
  be stale or the code may be wrong.
- Test items use the same scale with test vocabulary: `implemented` =
  written but not run, `verified` = passing, `broken` = failing,
  `drift` = asserts an outdated spec, `missing` = the test disappeared.

(Legacy values `pending/ok/attention/warning/error` map to
`planned/verified/implemented/drift/broken` and are migrated on read.)

## References — the spec graph's edges

Pebbles, checklist items, and flow nodes may carry `refs`: typed links to
any other part of the spec. This is a pre-built dependency graph — the
interconnections are already recorded so you don't have to rediscover them.

```json
{
  "id": "uuid of this ref entry",
  "targetKind": "pebble | item | node | field",
  "pebbleId": "id of the target pebble (canonical — survives moves)",
  "targetId": "id of the item/node/field inside it (absent = whole pebble)",
  "path": "project-relative path of the target file at link time (hint)",
  "label": "human-readable target name"
}
```

When asked to implement or verify something, ALWAYS follow its `refs`
first: a checklist item may reference the workflow it changes, the model
it touches, or the contract it must satisfy. Resolve by `path` when the
file still exists, else search sections for the `pebbleId`. After
implementing and verifying, update the item's `status` accordingly —
statuses are how progress is tracked here.

## For AI agents

- Treat `rules/` as hard constraints and `architecture/` as the dependency
  map. Flag violations you notice, even when unasked.
- A pebble rarely stands alone: a contract's payloads live in
  `structures/`, a requirement's behavior in `workflows/`, its background
  in `knowledge/`. Follow those links before answering.
- Statuses reflect the last verification against the codebase. Do not call
  something `ok` without evidence in the actual code.
