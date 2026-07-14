# Public API reference

This guide groups the complete exported surface by purpose. Symbol comments in
the source are the canonical field-level reference and are rendered by
`go doc github.com/faustbrian/go-jsonapi`.

## Core documents

| API | Purpose |
| --- | --- |
| `Document` | Top-level core document with `jsonapi`, links, data, included, errors, meta, and registered members |
| `JSONAPI` | Version plus applied extension/profile URIs and meta |
| `ResourceObject` | Resource identity, attributes, relationships, links, meta, and registered members |
| `Identifier` | Resource linkage identity using `type` plus `id` or `lid` |
| `Relationship`, `Relationships` | Relationship links, data, meta, and registered members |
| `Attributes`, `Meta`, `Members` | Application values and registered semantic values |
| `ErrorObject`, `ErrorSource` | JSON:API error representation and pointer/parameter/header source |

Primary-data constructors:

- `NullData()` creates an explicit top-level `data: null`;
- `ResourceData(resource)` creates to-one primary data;
- `ResourceCollection(resources...)` creates collection primary data;
- `NullRelationship()` creates explicit null linkage;
- `ToOne(identifier)` and `ToMany(identifiers...)` create relationship linkage.

## Core codec and validation

| API | Purpose |
| --- | --- |
| `Marshal`, `Unmarshal` | Strict generic document boundary |
| `MarshalWith`, `UnmarshalWith` | Boundary with `ValidationOptions` |
| `Document.Validate`, `Document.ValidateWith` | Validate a constructed document without encoding |
| `ValidationContext` | Generic, response, create, update, or relationship mutation context |
| `DecodeError` | One parsing/shape failure with JSON pointer and code |
| `ValidationError` | Ordered collection of `Violation` values |
| `Violation` | Path, stable machine code, and human-readable message |

All marshal entry points validate before emitting bytes. All unmarshal entry
points reject duplicate JSON members and validate before returning a document.

## Links

`Links` maps relation names to the opaque `Link` sum type. Construct links with:

- `URI(href)` for a JSON string;
- `NullLink()` for JSON `null`;
- `ObjectLink(href, meta)` for the common object form;
- `LinkFromObject(LinkObject{...})` for the full JSON:API 1.1 link object;
- `LanguageTag(tag)` or `LanguageTags(tags...)` for `hreflang`.

`LinkObject` supports `href`, `rel`, `describedby`, `title`, `type`, `hreflang`,
`meta`, and registered extension members.

## Query parsing

| API | Purpose |
| --- | --- |
| `ParseQuery(url.Values)` | Parse core query parameters with page/filter hooks |
| `NewQueryParser(custom, namespaces)` | Register implementation and extension families |
| `QueryParser.Parse` | Parse with the configured family registry |
| `Query` | Parsed include paths, fieldsets, sort, page, filter, custom, and extension families |
| `SortField` | One ordered field with descending flag |
| `ParameterFamily` | Original decoded names and values for application processing |
| `QueryError` | HTTP 400 query error with parameter and stable code |

## Content negotiation

| API | Purpose |
| --- | --- |
| `MediaTypeJSONAPI` | `application/vnd.api+json` constant |
| `MediaType` | Canonical extension and profile parameter representation |
| `MediaType.String` | Stable formatted content type |
| `NewNegotiator` | Register supported extension/profile URIs |
| `Negotiator.CheckContentType` | Validate request content type; returns 415 failures |
| `Negotiator.NegotiateAccept` | Select a response representation; returns 406 failures |
| `NegotiatedMedia` | Selected media type, content type, and `Vary: Accept` requirement |
| `NegotiationError` | Status, code, and message for protocol mapping |

## Registered extensions and profiles

| API | Purpose |
| --- | --- |
| `CodecOptions` | Validation plus extension/profile definitions |
| `NewCodec` | Validate registrations and construct a strict configured codec |
| `Codec.Marshal`, `Codec.Unmarshal` | Encode/decode registered semantic members |
| `ExtensionDefinition` | Absolute URI, namespace, and member definitions |
| `MemberDefinition` | Object scope, namespaced name, and optional value validator |
| `MemberScope` | Top-level, resource, relationship, identifier, JSON:API, error, error source, or link object |
| `ProfileDefinition` | Absolute URI and optional document semantic validator |

## Atomic Operations

| API | Purpose |
| --- | --- |
| `AtomicExtensionURI` | Official extension URI |
| `AtomicDocument` | Operations, results, errors, links, meta, and JSON:API object |
| `AtomicOperation`, `AtomicOperationCode` | Add, update, or remove operation |
| `AtomicReference` | Resource or relationship target by `id`/`lid` |
| `AtomicResult` | Positional operation result |
| `MarshalAtomic`, `UnmarshalAtomic` | Strict generic Atomic codec |
| `MarshalAtomicWith`, `UnmarshalAtomicWith` | Codec with `AtomicValidationOptions` |
| `AtomicDocument.Validate`, `ValidateWith` | Validate constructed Atomic documents |
| `AtomicValidationContext` | Generic, operations request, or results response |
| `ExecuteAtomic` | Ordered transaction execution with rollback/commit guarantees |
| `AtomicTransactionBeginner` | Begins an application transaction |
| `AtomicTransaction` | Applies operations and commits or rolls back |
| `AtomicExecutionError` | Operation and rollback failure context |

## Cursor Pagination

| API | Purpose |
| --- | --- |
| `CursorPaginationProfileURI` | Official profile URI |
| `NewCursorPagination` | Validate endpoint pagination policy |
| `CursorPagination.Parse`, `ParseQuery` | Validate page family and optional sort policy |
| `CursorPaginationConfig` | Default/max size, range support, cursor/sort validators |
| `CursorPageRequest` | Parsed size, before/after values, presence, and range flag |
| `CursorPaginationError` | Profile error with `ErrorObject` conversion |
| `ValidateCursorPaginationLinks` | Require and validate `prev` and `next` links |
| `CursorPage` | Data/links/meta instance validation |
| `CursorPageMeta`, `CursorEstimatedTotal` | Exact/estimated total metadata representation |
| `CursorPageMeta.Meta`, `ParseCursorPageMeta` | Encode/decode pagination metadata |
| `CursorItemMeta`, `ParseCursorItemMeta` | Encode/decode optional per-item cursor metadata |

The exported error type-link URI constants are
`CursorUnsupportedSortTypeURI`, `CursorMaxSizeExceededTypeURI`, and
`CursorRangeNotSupportedTypeURI`.
