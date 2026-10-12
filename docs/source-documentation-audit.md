# Source documentation audit

## Scope and reproducible inventory

Baseline: `f8d69a11a1e7c49ffa0dd43766d9d8bb9cd955b4`. Tier A source prose:
preserve executable behavior, module inputs, historical authority pins and typed
append-only decision digests. The standalone repository owns this pass.

Reproduce with `git ls-files '*.go'`, `git ls-files '*go.mod'`,
`golib source-docs inventory --json`, and `rg -n 'go:build|go:generate|go:embed|Code generated|Deprecated:' --glob '*.go'`.
Inventory: 52 tracked Go files, two modules and two package directories. The
interoperability harness is independently resolved and non-releasable. Audit
all public declarations, interfaces, fields and methods, private parser/graph/
callback invariants, tests, examples, nine fuzz targets and benchmarks. External
peer source is not rewritten. Generator/platform/marker results are recorded
below after inspection rather than assumed from the default build.

| File | Module directory | Package | Kind |
| --- | --- | --- | --- |
| `atomic.go` | `.` | `jsonapi` | source |
| `atomic_codec.go` | `.` | `jsonapi` | source |
| `atomic_codec_test.go` | `.` | `jsonapi` | test |
| `atomic_execute.go` | `.` | `jsonapi` | source |
| `atomic_execute_test.go` | `.` | `jsonapi` | test |
| `atomic_test.go` | `.` | `jsonapi` | test |
| `automation_test.go` | `.` | `jsonapi` | test |
| `benchmark_test.go` | `.` | `jsonapi` | test |
| `callback.go` | `.` | `jsonapi` | source |
| `callback_test.go` | `.` | `jsonapi` | test |
| `codec.go` | `.` | `jsonapi` | source |
| `codec_defense_test.go` | `.` | `jsonapi` | test |
| `codec_test.go` | `.` | `jsonapi` | test |
| `context_validation_test.go` | `.` | `jsonapi` | test |
| `cursor.go` | `.` | `jsonapi` | source |
| `cursor_admission_security_test.go` | `.` | `jsonapi_test` | test |
| `cursor_budget_boundary_test.go` | `.` | `jsonapi_test` | test |
| `cursor_meta.go` | `.` | `jsonapi` | source |
| `cursor_meta_test.go` | `.` | `jsonapi` | test |
| `cursor_page.go` | `.` | `jsonapi` | source |
| `cursor_page_test.go` | `.` | `jsonapi` | test |
| `cursor_test.go` | `.` | `jsonapi` | test |
| `document.go` | `.` | `jsonapi` | source |
| `document_test.go` | `.` | `jsonapi` | test |
| `duplicate_member_test.go` | `.` | `jsonapi` | test |
| `error_contract_test.go` | `.` | `jsonapi` | test |
| `examples_test.go` | `.` | `jsonapi_test` | test |
| `fuzz_test.go` | `.` | `jsonapi` | test |
| `identity_validation_test.go` | `.` | `jsonapi` | test |
| `interoperability/runner.go` | `interoperability` | `main` | source |
| `interoperability/runner_test.go` | `interoperability` | `main` | test |
| `limits.go` | `.` | `jsonapi` | source |
| `link_test.go` | `.` | `jsonapi` | test |
| `member_codec.go` | `.` | `jsonapi` | source |
| `member_codec_defense_test.go` | `.` | `jsonapi` | test |
| `member_codec_test.go` | `.` | `jsonapi` | test |
| `negotiation.go` | `.` | `jsonapi` | source |
| `negotiation_limits_test.go` | `.` | `jsonapi` | test |
| `negotiation_test.go` | `.` | `jsonapi` | test |
| `ownership_test.go` | `.` | `jsonapi` | test |
| `presence_test.go` | `.` | `jsonapi` | test |
| `profile_codec_test.go` | `.` | `jsonapi` | test |
| `query.go` | `.` | `jsonapi` | source |
| `query_limits_test.go` | `.` | `jsonapi` | test |
| `query_test.go` | `.` | `jsonapi` | test |
| `resource_limits_test.go` | `.` | `jsonapi` | test |
| `unknown_member_test.go` | `.` | `jsonapi` | test |
| `validation.go` | `.` | `jsonapi` | source |
| `validation_edge_test.go` | `.` | `jsonapi` | test |
| `validation_mutation_test.go` | `.` | `jsonapi` | test |
| `validation_scope_boundary_test.go` | `.` | `jsonapi_test` | test |
| `validation_test.go` | `.` | `jsonapi` | test |

## Gaps versus technical questions before editing

The objective scanner identifies 179 missing-comment findings, mostly public
wire fields/options plus the harness package comment. Existing package documentation understates the core,
Atomic Operations, Cursor Pagination, query and negotiation surfaces. Missing
comments must communicate absence/null/empty distinctions, caller ownership,
limits and application obligations, not narrate field spellings.

Resolve these questions against implementation and tests before promising a
contract: which constructors copy only outer containers; whether custom JSON
and validators can block/panic or mutate validation inputs; where byte/depth/
value/resource budgets apply; whether unknown or duplicate members are ignored;
which extension/profile scopes are activated; how atomic executor failure
cleanup and partial results are represented; how cursor errors and requested
sort fields differ from backend policy; and whether negotiation commits profile
semantics or merely advertises registered support.

High-risk files: codec.go, validation.go, member_codec.go, limits.go,
atomic.go/atomic_codec.go/atomic_execute.go, cursor.go/cursor_meta.go/
cursor_page.go, negotiation.go and query.go. Presence bits and shallow value
constructors in document.go are compatibility-sensitive. callback.go owns an
explicit application-callback diagnostic boundary, not general cancellation.

## Selected proof

Read the pinned JSON:API 1.1 source authority and local typed decisions/fixtures;
retain historical pins and append-only metadata. Select native API handling
from `.golib.yaml` and actual scripts/tests, not an assumed documentation
snapshot. Use Go1.27.2 in the granted serialized slot for all-file formatting,
executable-token/comment-free-AST/directive equivalence, both package renders
and native Example checks. Inspect generation, local links, markers, pinned
changed-content secrets and the complete diff. No active fuzz, race, benchmark,
protocol/release campaign or peer qualification is warranted for unchanged
runtime. Own and clean all three Go caches after the bounded session.

## Audited contract surfaces

The native AST pass inventories 393 public declarations/members, including
interface methods and all public fields; private capitalized fixture members
are not a public-API requirement. Reproduce the semantic member inventory with
`go doc -all` for the catalog packages and the source inventory commands above.

| Surface | Contract inventory |
| --- | --- |
| Wire models | Attributes, Meta, Document, JSONAPI, ResourceObject, Identifier, PrimaryData, Relationship/RelationshipData, Links/Link/LinkObject/LinkHreflang, ErrorObject/ErrorSource; constructors, With methods and MarshalJSON presence/aliasing |
| Core validation/codec | ValidationContext/Options, Violation/ValidationError, Validate/ValidateWith, DecodeError, Marshal/Unmarshal variants and DecodeLimits; duplicate/unknown-member scans, pointer paths, URI/link checks and ID/LID reachability |
| Registered semantics | Members, MemberScope constants, Member/Extension/ProfileDefinition, CodecOptions and Codec; registry copying, callback borrowing, optional declarations and profile before/after wire comparison |
| Atomic Operations | Extension/operation/context constants, AtomicValidationOptions/Document/Operation/Reference/Result; codec/validation variants, AtomicTransaction/Beginner interfaces, ExecuteAtomic, AtomicExecutionError/AtomicPanicError |
| Query and negotiation | ParameterFamily, SortField, Query/QueryError, QueryParser and QueryLimits; MediaType, NegotiatedMedia, NegotiationError/Negotiator and NegotiationLimits; syntax versus application projection/filter/ordering policy |
| Cursor Pagination | Profile/error URI constants, CursorPaginationConfig/Request/Error, CursorPagination, CursorPage, CursorPageMeta/EstimatedTotal; parsing, page evidence, alias/presence and exact numeric metadata helpers |
| Independent harness | Five maintained-peer observation pairs, actual build dependency identity, partial writer failure and fixed matrix normalization; no whole-protocol or current-main qualification |

All tracked tests, helpers, five runnable examples, nine fuzz targets and twelve
benchmarks were included in the source audit. Fuzz target comments identify
input boundaries and observed invariants; benchmark comments identify fixture
shape, preparation outside b.Loop and excluded application/I/O work. Atomic
recording fixtures are not storage-atomicity evidence. The peer harness remains
pinned at DataDog/jsonapi v0.13.0 outside the public module.

No generated Go files, generator directives, embedded source, build/platform
constraints, deprecations or actionable debt/note/warning markers were found.
Specification-derived JSON fixtures and typed decision/conformance/history
records are maintained inputs, not generated Go code. No artifact regeneration
or modification of historical authority pins/digests is needed.

## Resolved rationale and existing limitation register

- JSON:API 1.1 is pinned to json-api/json-api revision
  `52fc018adcc15358cb0f9d3de119f7e0fbe34ea6`. The pinned official source was
  read directly, including Document Structure and Resource Object
  Identification. The thirteen existing typed decisions continue to own
  unknown members, registration, query families, presence, alias-aware full
  linkage, sparse fieldsets, pagination, Atomic ordering, recommendations,
  authority conflicts, duplicate members, RFC 8288 relation grammar and
  RFC 9110 Accept precedence/ties. No expected protocol rule was changed.
- Nil/empty/null distinctions are intentional wire presence contracts. Value
  constructors and With methods copy outer values or slices only; nested maps,
  pointers and custom values remain borrowed. Direct MarshalJSON encodes wire
  values without the checked API's validation.
- Decode byte/structural budgets include ignored members. Configurable limits
  do not bound arbitrary caller allocations, custom JSON methods or callbacks.
  Constructed link recursion has its own default-depth and ancestor-cycle checks.
  Query admission bounds decoded values; the HTTP layer still owns encoded
  request-target/body/header admission. Direct cursor input uses fixed default
  query budgets even if an upstream parser is configured more generously.
- Registry constructors copy definitions and URI/name containers, not callback
  state. Registered callbacks are synchronous and may be concurrent. Profile
  before/after JSON comparison detects observable mutation after success but
  neither clones nor restores caller data. Ordinary wrapped callback panics are
  converted; blocking, custom JSON side effects and goroutine termination are
  not isolated. No input budget is a callback deadline.
- Atomic execution validates before beginning, applies in order, validates
  corresponding results before commit and returns no partial response on
  handled failures. Failed partial begin remains implementation-owned. Rollback
  uses context.WithoutCancel without a package deadline; implementations must
  bound cleanup. A rollback attempt following commit error cannot prove
  non-durability or justify retrying external effects. Goroutine termination is
  outside handled failure cleanup; no runtime guarantee was manufactured.
- Query parsing does not authorize/project fields or execute filters and sorts.
  Negotiated URI declarations do not install codec semantics. Cursor helpers
  validate caller/backend evidence, not authenticate cursors or fetch results.
  Caller-owned maps must stay stable during validation; arbitrary map traversal
  does not promise violation order. The inaccurate deterministic-error/source-
  pointer claims in current architecture prose were narrowed.
- Existing **wire-type limitation**, not an intentional interpretation:
  a disposable read-only native probe called Unmarshal with
  `{"data":{"type":"articles","id":null}}` and the explicit-string control
  `{"data":{"type":"articles","id":""}}`. Both succeeded, and Marshal
  produced `{"data":{"type":"articles","id":""}}` without error for each.
  Shared decodeString inherits encoding/json null-to-zero-string normalization;
  the caller sets presence separately. JSON:API 1.1 Resource Object
  Identification requires string identity values, so this observation is not
  conformance success or permission for null IDs. Source API/private-seam and
  current security docs explicitly disclose the limitation. Runtime and test
  code remain unchanged. A separate behavioral repair/acceptance decision is
  outside this source pass; the observation was reported to the coordinator.

## Selected checks and outcomes

- Pinned Go1.27.2 formatting passed on all 52 tracked files. A disposable stdlib
  parser/scanner compared exact executable token/literal/inserted-semicolon
  sequences, normalized comment-free ASTs and compiler/analyzer directives with
  the exact baseline: identical across every file, including tests and harness.
- Native `go doc -all` rendered both packages. Root rendered contracts were
  inspected and final affected documentation reproduced twice byte-stably.
  The harness render used a disposable two-module workspace naming only this
  repository's root/harness; no repository workspace/sum was changed.
- The native documentation selector passed all five existing examples:

  ```sh
  GOWORK=off go test ./... -run '^Example' -count=1 -timeout=3m
  ```

  No example executable statement changed, and no broader runtime suite ran
  solely for comments.
- `.golib.yaml` selects root apidiff against `api/baseline.txt`; there is no
  native prose API snapshot or snapshot-generating test/script. The API type
  baseline, all module/workspace manifests/sums, specification fixtures and
  historical typed records remain unchanged. The export analyzer and peer
  interoperability campaign are not claimed as locally executed.
- Current shared source inventory/check covers all 52 tracked Go files with no
  structural findings. Automation/CI adoption remains coordinator-owned.
- Native required-file/local Markdown-path checks and current Unreleased
  structure passed. Pinned Gitleaks v8.30.1 used the retained policy, inline
  allow suppression disabled, empty ignore input and full redaction.
  The changed-current full-file snapshot **failed** with 23 generic-api-key
  matches, all in CHANGELOG.md on byte-identical historical JSONAPI decision
  SHA256 lines. Source comparison proves these are preserved public decision
  digests, not newly added credentials. No suppression, history rewriting or
  scanner-policy change was made.
- The applicable added-content/new-file scan with the same pinned scanner
  passed across all 22 changed paths with zero findings. This is distinct from
  the failed full-file snapshot; neither result is renamed as the other.
  Private snapshots/reports were deleted after redacted identities/counts were
  captured. Ordinary complete Tier A review found no executable changes or
  unresolved source-documentation finding; the known decoder defect remains
  an explicitly disclosed separate behavior boundary.
- The null-string probe observation is separate from the unchanged-runtime
  proof. Its helper, both rendered outputs, temporary workspace/sum and all
  three exact task-owned Go caches were removed after the bounded session.

No runtime repair, active fuzzing, race/leak/benchmark campaign, maintained-peer
qualification, external CI, publication or release was performed. The existing
null-string defect remains open for a separately scoped behavior change.
