# Conformance matrix

This matrix maps required protocol behavior to implementation evidence. A row
is complete only when behavior, malformed input, and stable errors are covered;
statement coverage alone is not treated as conformance proof.

Normative references:

- [JSON:API 1.1 specification](https://jsonapi.org/format/)
- [Atomic Operations extension](https://jsonapi.org/ext/atomic/)
- [Cursor Pagination profile](https://jsonapi.org/profiles/ethanresnick/cursor-pagination/)

## Core document structure

| Requirement group | Implementation | Primary evidence |
| --- | --- | --- |
| Root object and required top-level content | `decodeDocument`, document validator | `codec_test.go`, `validation_test.go`, malformed fixtures |
| `data`/`errors` exclusivity and `included` dependency | document validator | `validation_test.go`, `validation_edge_test.go` |
| Resource `type`, `id`/`lid`, field namespaces | resource validator | `document_test.go`, `identity_validation_test.go` |
| Resource identifier restrictions | identifier validator | `identity_validation_test.go`, `validation_edge_test.go` |
| Relationship object and linkage shapes | relationship decoder/validator | `document_test.go`, `validation_test.go` |
| Link string/object/null forms and members | link codec/validator | `link_test.go`, `codec_defense_test.go` |
| Error object/source shapes | error codec/validator | `document_test.go`, `error_contract_test.go` |
| JSON:API object versions/ext/profile URIs | JSON:API validator | `identity_validation_test.go`, `context_validation_test.go` |
| Unknown members and additive `@` members | strict object decoder | `codec_test.go`, `codec_defense_test.go` |
| Duplicate JSON object members | streaming duplicate scanner | `duplicate_member_test.go`, `codec_defense_test.go` |
| Exact number and presence round trips | exact-value decoder and custom marshalers | `codec_test.go`, `presence_test.go` |

## Compound documents

| Requirement group | Implementation | Primary evidence |
| --- | --- | --- |
| Included resource uniqueness | identity index | `validation_test.go` |
| Primary/included identity conflict detection | identity index | `validation_test.go`, `validation_edge_test.go` |
| Full linkage from primary data | relationship traversal | `validation_test.go` |
| Cycles and repeated linkage | visited identity traversal | `validation_test.go` |
| Local identity linkage | identity rules | `identity_validation_test.go` |
| Canonical compound fixture | core codec | `testdata/valid/compound-document.json`, `codec_test.go` |

## Requests, query parameters, and negotiation

| Requirement group | Implementation | Primary evidence |
| --- | --- | --- |
| Create/update/relationship document contexts | `ValidationOptions` | `context_validation_test.go` |
| include paths | `parseInclude` | `query_test.go` |
| sparse fieldsets | `parseFields` | `query_test.go` |
| sorting | `parseSort` | `query_test.go` |
| page/filter family preservation | `ParameterFamily` | `query_test.go` |
| custom and extension family registration | `QueryParser` | `query_test.go` |
| request content type parameter rules | `CheckContentType` | `negotiation_test.go` |
| response `Accept` selection and quality | `NegotiateAccept` | `negotiation_test.go` |
| deterministic media type formatting | `MediaType.String` | `negotiation_test.go` |

## Registered extensions and profiles

| Requirement group | Implementation | Primary evidence |
| --- | --- | --- |
| Absolute unique extension/profile URIs | `NewCodec` | `member_codec_defense_test.go`, `profile_codec_test.go` |
| Extension namespace/name rules | `NewCodec` | `member_codec_test.go` |
| Top-level and all object member scopes | scoped registry | `member_codec_test.go` |
| Nested link-object members | recursive sanitizer | `member_codec_test.go` |
| Member value semantics | per-member validators | `member_codec_test.go`, `member_codec_defense_test.go` |
| Profile document semantics | profile callbacks | `profile_codec_test.go` |
| Core rejection without registration | core codec | `member_codec_test.go` |

## Atomic Operations extension

| Requirement group | Implementation | Primary evidence |
| --- | --- | --- |
| Top-level operations/results/errors rules | Atomic validator | `atomic_test.go`, `atomic_codec_test.go` |
| Operation code and target rules | operation validator | `atomic_test.go` |
| Add/update/remove resource shapes | contextual Atomic validation | `atomic_test.go`, `context_validation_test.go` |
| Relationship operation shapes | contextual Atomic validation | `atomic_test.go` |
| `ref` identity and relationship constraints | reference validator | `atomic_test.go` |
| Local identity ordering and resolution | operation sequence validation | `atomic_test.go` |
| Results and positional correspondence | result validator/executor | `atomic_test.go`, `atomic_execute_test.go` |
| Ordered transactional application | `ExecuteAtomic` | `atomic_execute_test.go` |
| Rollback on operation/commit failure | `ExecuteAtomic` | `atomic_execute_test.go` |
| Strict codec and malformed documents | Atomic codec | `atomic_codec_test.go` |

## Cursor Pagination profile

| Requirement group | Implementation | Primary evidence |
| --- | --- | --- |
| Page size syntax/default/maximum | `CursorPagination.Parse` | `cursor_test.go` |
| Before/after cursor validation | cursor callback | `cursor_test.go` |
| Range policy and max default | pagination policy | `cursor_test.go` |
| Stable/unsupported sorting | sort callback | `cursor_test.go` |
| Required prev/next links | link validator | `cursor_test.go`, `cursor_page_test.go` |
| Page instance validation | `CursorPage` | `cursor_page_test.go` |
| Item cursor metadata | metadata helpers | `cursor_meta_test.go` |
| Total and estimated total metadata | `CursorPageMeta` | `cursor_meta_test.go` |
| Required error type links/meta | error conversion | `cursor_test.go` |

## Resilience and coverage evidence

- `go test ./... -coverprofile=coverage.out` and `go tool cover -func` report
  100.0% of production statements.
- The raw coverage profile is checked in CI for any zero-count production
  statement, avoiding rounded totals.
- `fuzz_test.go` covers core decoding, Atomic decoding, query parsing, Cursor
  parsing, and media negotiation.
- `benchmark_test.go` covers single resources, collections, compound
  documents, Atomic operations, queries, and negotiation.
- Every accepted codec fuzz input must marshal and decode canonically again.

Application-owned HTTP routing, persistence, filtering semantics, cursor
encoding, and authorization cannot be proven by this package's conformance
suite. The [adoption guide](adoption.md) identifies those integration duties.
