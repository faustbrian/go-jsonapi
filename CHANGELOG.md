# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and releases follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

The [specification decision register](docs/specification-decisions.md) records
the observable protocol choices covered by these entries.

### Added

- Add machine-auditable JSON:API decision, conformance, authority-monitoring,
  interoperability, and decision-history records.
- JSONAPI-DEC-001 sha256:1c992612f6fdf57e5858753747bcb57464f149b517084813681e7029f31186f0
- JSONAPI-DEC-002 sha256:7d6658ae3e8b8176dffdb80996a5439f55b30d8d3c1e5a2a3d1c9f285f2d2338
- JSONAPI-DEC-003 sha256:86ae12b7a1ba561cf2f694756d980ed6eec544b3007f47058e87f8c51023115e
- JSONAPI-DEC-004 sha256:be285995517629345b407e0b93744ada5cc85d54b72092b38310d9ff16b795f5
- JSONAPI-DEC-005 sha256:db4077bb84f3aadc3f3b68c307029f01c2f49ae45a8f17d34e0d4945a03f31d3
- JSONAPI-DEC-006 sha256:526de3bc95e292ca970e560bd06783a9ce70a8ef656dfd8e7ef51151a00339c5
- JSONAPI-DEC-007 sha256:ad587c6d3623a7964b3047f742d2b9c191b9b979984171857c7277092d110501
- JSONAPI-DEC-008 sha256:98b181e40f339a4a11b389a6bfe2ed0ca28ac1225fabfdaad62f5430bb5363b7
- JSONAPI-DEC-009 sha256:47736a0649bf48e19fed8cc75bb8308ac1790699487a4e4055a8e2a4d486ec2b
- JSONAPI-DEC-010 sha256:c7a9604e5ef3e66a7576826b8ab08b6e58d5de1bd4d9545184b4ff0c66d8c2fa
- Add explicit RFC decisions for duplicate JSON members, link relation types,
  and HTTP Accept precedence and deterministic ties.
- JSONAPI-DEC-011 sha256:d234cd9a4e40ce09214a023d09c11af6e7077c1ba5a39f8c01a9dc6cbbb98095
- JSONAPI-DEC-012 sha256:85161bb47428b3aead1e8930b318ccd9a81468001e190378cdb7b621eb3022ef
- JSONAPI-DEC-013 sha256:9b156b1aaf44e12eb6f10ee74a737c981aec3f898b94fad1f547ac2e68c57e04

### Changed

- Pin JSON:API source text to an immutable upstream revision, monitor mutable
  update documents separately, and add an executable DataDog/jsonapi v0.13.0
  differential harness outside the public module.
- JSONAPI-DEC-001 sha256:7ab9e941c1eed52abff8d37ed674e466d75682bfc9dc3427242089692c7fadd4
- JSONAPI-DEC-002 sha256:92b24e46b91b172239d6c7ef6afeab3028f6d3d883f6c66e641b951574811cb3
- JSONAPI-DEC-003 sha256:b894d768df3cdfa96914f4b8805c3dfdfe3bf3988b7281e48f73113c7abff5c1
- JSONAPI-DEC-004 sha256:eefa54a58ed63f71bbaa3b6d2d8be4cabcaef92ee05d1d9c7e7014aebde062fd
- JSONAPI-DEC-005 sha256:8c5339c0c40d1ca9ecbc7db8fe8ecb133d597b6dfa4d535f9140e7a803751050
- JSONAPI-DEC-006 sha256:91709eb6d82599863d8e03759d1cd9f7268d26e8cfb3b0dd652b7e271873cb97
- JSONAPI-DEC-007 sha256:17f841abf7b9511169d916412f03e7b513bfde2b93b7642266a23d92e5ee99f0
- JSONAPI-DEC-008 sha256:383a65c3c89cae68d2d614993caab29da026c9a6a2c057e570b6bbf652c9a7db
- JSONAPI-DEC-009 sha256:ee3532533128439a7617e7da04fbbda13f30b14b7eb301190fd5712a1d2c0625
- JSONAPI-DEC-010 sha256:6721f9d3ea09062d9665828d3e1f2127d3bd3d1d967d6214e01a164bc7b8b1f2

- Use the centrally maintained Go library verification and CI contract at
  `go-library-tools` v1.0.13.

### Documentation

- Clarify how shared safety-policy updates are coordinated across standalone
  repositories.

- Replace archived monorepo and AI-generated documentation entry points with
  a standalone, human-oriented documentation structure.

## [1.0.0] - 2026-08-25

### Fixed

- Preserve Accept-header order while evaluating unique representations so
  quality selection and canonical equal-quality tie-breaking are deterministic.

- Make metadata at-member filtering deterministic under mutation verification
  while preserving all ordinary metadata members.

### Changed

- Validate action pinning from the standalone repository root and leave
  repository-foundation policy to the authoritative repository contract.

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Replace obsolete standalone-repository links and workflow claims with
  monorepo-canonical targets and current release guidance.

### Compatibility

- Added a pinned module export baseline so incompatible public API changes
  fail the canonical repository gate.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-jsonapi` identity while preserving its documented API and behavior.
- Upgrade `golang.org/x/text` to v0.41.0 so the dependency graph no longer
  contains GO-2026-5970.
- Pin specification-derived JSON:API fixtures and make security, resource,
  compatibility, and wire consequences explicit for every protocol decision.
- Ignore non-compliant members during core and Atomic decoding as required by
  JSON:API 1.1 while retaining strict validation for recognized members.
- Avoid speculative capacity arithmetic when encoding registered additional
  members; the buffer now grows from the already bounded core document size.
- Regenerated the complete machine-readable documentation bundle from the
  current user, adoption, compatibility, and release documentation.
- Expose JSON:API specification verification as an explicit conformance gate.
- Added the `GO-SAFETY-1` ownership, concurrency, race, fuzz, resource, and
  benchmark standard with an executable `make safety` gate.
- Moved AI planning and hardening briefs into `.ai/` and clarified the
  separate purposes of project and third-party notice files.

### Added

- An auditable specification-decision register covering JSON:API core,
  extensions, profiles, recommendations, and application-policy boundaries.
- A standardized OSS repository skeleton covering policy, documentation,
  legal notices, Go tooling, pinned CI, security, and release automation.
- Evidence-driven audit and hardening goal covering JSON:API 1.1, Atomic
  Operations, Cursor Pagination, negotiation, queries, and resource safety.
- Strict JSON:API 1.1 document models, codecs, validation, links, errors,
  compound-document support, and deterministic serialization.
- Query parsing primitives for inclusion, sparse fieldsets, sorting,
  pagination, filtering, implementation families, and extension namespaces.
- Content negotiation for the JSON:API media type, extensions, profiles,
  quality values, and wildcard candidates.
- Complete Atomic Operations document validation and transaction-oriented
  execution interfaces.
- Cursor Pagination query, link, metadata, item-cursor, total, estimated-total,
  and profile error helpers.
- Extension-member registries across JSON:API-defined object scopes and
  document-level profile validation hooks.
- Golden fixtures, malformed-input regressions, round-trip tests, fuzz targets,
  and representative benchmarks.
- Project documentation, conformance matrices, adoption guidance, and
  contribution and security policies.
- GitHub Actions quality, compatibility, fuzzing, security, benchmark,
  documentation, and tagged-release automation.
- MIT licensing for public use, modification, and distribution.
- Bounded decoding with configurable byte, nesting, object-member, array-item,
  and total-value limits shared by core, Atomic, and configured codecs.
- Bounded query and media negotiation APIs with production defaults for
  decoded parameters, header candidates, and extension/profile URI lists.
- Distinct registered scopes for extension members inside `links` objects and
  individual link objects, including opaque links-object value helpers.
- A finding ledger, verification-backed hardening report, and package threat
  model for the pre-v1 audit.
- Constructed-validation, member-registry, cursor-metadata, and canonical
  round-trip fuzz targets plus adversarial compound and pagination benchmarks.
- `CallbackError` for redacted, inspectable extension/profile/cursor/sort
  callback failures and panic values.

### Fixed

- Reject cursor metadata numbers that round to or beyond the signed 64-bit
  boundary before conversion, preventing implementation-defined overflow.
- Keep module-archive tests scoped to files shipped with the JSON:API module;
  repository-root workflow policy remains owned by the root verification gate.
- Bound fuzz-smoke concurrency to avoid deadline flakes on high-core hosts.
- Preserve large JSON numbers in attributes without `float64` precision loss.
- Invoke registered member validators only once during configured decoding.
- Preserve explicitly empty string members, including empty resource IDs and
  empty URI-references, without confusing presence with a Go zero value.
- Allow update-request endpoint identity checks to target a valid empty ID
  through explicit expected-ID presence state.
- Preserve empty `id` and `lid` presence when Atomic relationship data is
  validated as resource identifier objects.
- Support href-targeted Atomic relationship add, update, and remove data
  shapes while rejecting relationship-shaped operations without a target.
- Validate Atomic results against their request operations before commit,
  including data-free removal/relationship results and singular resource data.
- Require lid-only Atomic resource targets and relationship linkage to resolve
  to the current or a prior resource add operation.
- Reject directly comparable type, ID, or LID mismatches between an Atomic
  resource update's `ref` target and `data` representation.
- Resolve compound-document linkage through either `id` or `lid` when an
  included resource carries both identities.
- Detect duplicate canonical resource objects across `id` and `lid` aliases.
- Reject pagination links beside non-collection top-level data and known
  to-one relationship linkage.
- Require optional `jsonapi.ext` and `jsonapi.profile` arrays to include all
  configured applied URIs while continuing to ignore unknown profiles.
- Ignore unrecognized profiles when calculating `Accept` media-range
  specificity so they cannot override an otherwise acceptable base range.
- Separate forward-compatible @-Member names from semantic member names and
  ignore constructed @ members in relationships and links.
- Require error-object `status` strings to contain a valid HTTP status code in
  the inclusive range from 100 through 599.
- Reject core link members used in the wrong links-object scope while retaining
  registered extension members and the core pagination names where permitted.
- Validate URI-references and absolute registration URIs using RFC 3986 wire
  characters, requiring spaces, Unicode, and reserved path data to be escaped.
- Reject underscores in registered link relation types as prohibited by the
  Web Linking grammar.
- Reserve only `type` and `id` in resource field namespaces, allowing `lid` as
  an attribute or relationship name as required by JSON:API 1.1.
- Reject U+007F DELETE in member names while retaining the specification's
  globally allowed U+0080-and-above Unicode range.
- Reject invalid UTF-8 in constructed member names before encoding can replace
  it and change the registered or validated name.
- Bound constructed `describedby` link chains and reject pointer cycles before
  recursive validation can exhaust the stack.
- Reject profile validators that mutate their document view so callbacks
  cannot invalidate already-checked core or extension semantics.
- Reject extension member names that place `@` after the namespace separator;
  the @-Member exception applies only at the start of the complete name.
- Accept valid constructed @-Members without extension registration while
  preventing them from satisfying required document or relationship content.
- Require relationship-only links to include `self` or `related` rather than
  accepting an empty or unrelated links object as relationship content.
- Enforce Cursor Pagination boundary-link nullability and support aliases for
  the profile's `page` metadata element across pages, items, and errors.
- Accept mathematically integral JSON number forms in Cursor Pagination totals
  without binary floating-point conversion or precision loss.
- Reject malformed UTF-8 instead of accepting Go's replacement-character
  decoding at JSON:API boundaries.
- Enforce the HTTP `qvalue` grammar instead of accepting floating-point forms
  such as `NaN`, exponents, signs, or excess fractional precision.
- Enforce literal U+0020 separators in media-type extension and profile URI
  lists instead of normalizing tabs or Unicode whitespace.
- Preserve cursor/sort validator causes through `errors.Is` without copying
  their potentially sensitive text into public pagination errors.
- Contain extension/profile/cursor/sort callback panics and redact returned
  callback error text while preserving trusted `errors.Is`/`errors.As` access.
- Allow a registered extension member to provide the required content of an
  error object while keeping ignored @-Members structurally invisible.
- Require Atomic create results to return server-generated resource data before
  commit while retaining the extension's client-generated-ID omission rule.
- Convert panics from Atomic transaction callbacks into typed, redacted
  `AtomicPanicError` failures and attempt rollback exactly once.
- Reject canceled or nil contexts before beginning an Atomic transaction.
- Apply HTTP media-range specificity before quality when selecting a JSON:API
  representation, so a wildcard cannot override a more specific rejection.
- Reject unknown core or Atomic validation contexts and negative Atomic
  expected-result counts instead of silently weakening contextual checks.
- Enforce canonical SemVer release tags, including leading-zero and prerelease
  identifier rules, before publishing artifacts.
- Decouple validated changelog release dates from the wall-clock day on which
  a prepared tag is published.

[Unreleased]: https://github.com/faustbrian/go-jsonapi/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/faustbrian/go-jsonapi/releases/tag/v1.0.0
