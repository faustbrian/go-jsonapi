# Compatibility policy

Resolved choices in the
[specification decision register](specification-decisions.md) are part of this
policy. Changing one requires specification review and a changelog entry even
when exported Go signatures do not change.

## Versioning

The project follows Semantic Versioning.

- Incompatible exported API or wire-format changes after `v1.0.0` require a
  new major version.
- Backward-compatible additions use minor releases.
- Patch releases fix defects without intentionally changing valid behavior.

## Governed surface

Compatibility includes:

- exported Go names, method signatures, interfaces, and constants;
- JSON member presence, null/empty behavior, and canonical encoding;
- accepted and rejected document shapes;
- typed error fields, stable codes, and JSON pointer paths;
- query parsing and negotiation selection rules;
- official extension/profile URI constants and semantics;
- transaction callback ordering and rollback behavior;
- typed panic conversion and redacted execution error text;
- callback phases, cause unwrapping, and redacted extension/profile/cursor/sort
  error text;
- profile-validator purity and mutation rejection;
- constructor input-copying and concurrent-use guarantees;
- constructed recursive-link depth and cycle rejection;
- default resource limits and stable limit error classification;
- HTTP quality-value parsing and negotiation selection rules;
- supported Go version policy.

JSON object order is not semantically significant, but deterministic output is
still treated as a tested compatibility property.

## Deprecation

When practical, an obsolete exported API is deprecated in documentation and
kept through at least the next minor release before removal in a major release.
Immediate removal is reserved for severe security or correctness defects and
must include explicit release notes.

## Go support

Stable v1 requires Go 1.27.0 or later. The exact minimum is declared in
`go.mod`, `.go-version`, and `modules.json`. Raising it requires release notes
and a new release; consumers on older toolchains remain on the last compatible
release.

## Specification evolution

JSON:API 1.1 follows an additive compatibility model. New specification
members are reviewed before support is claimed. Until implemented, strict core
decoding may reject a newly defined member except where `@`-member or registered
extension/profile mechanisms apply. Such updates are tracked as protocol
compatibility work, not silently inferred.
