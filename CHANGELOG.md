# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and releases follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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

### Fixed

- Preserve large JSON numbers in attributes without `float64` precision loss.
- Invoke registered member validators only once during configured decoding.

[Unreleased]: https://github.com/faustbrian/go-jsonapi/commits/main
