# Specification provenance

`manifest.tsv` pins the specification-derived valid documents exercised from
`testdata/valid`. The digest and byte count detect accidental fixture drift;
the fragment-free source URL identifies the authority and the section column
identifies the normative section represented by each fixture. These are
maintained transcriptions, not an official JSON:API test suite.

The [conformance matrix](../docs/conformance.md) maps JSON:API 1.1, Atomic
Operations, Cursor Pagination, and referenced-standard requirements to
executable evidence. The
[specification decision register](../docs/specification-decisions.md) records
ambiguities, authority boundaries, and package policy that fixtures alone
cannot determine.

`monitoring.json` pins the base format, Atomic Operations, Cursor Pagination,
and recommendations to upstream commit
`52fc018adcc15358cb0f9d3de119f7e0fbe34ea6`. Separate mutable `gh-pages`
authorities detect upstream publication changes without silently changing the
selected behavior. RFC source publications and RFC errata feeds remain
separate authorities for the same reason.

`differential.tsv` is reproduced by the non-releasable `interoperability`
module against the current workspace and maintained peer
DataDog/jsonapi v0.13.0. It covers overlapping unknown-member, query-family,
relationship-linkage, resource-uniqueness, and sparse-field behavior. Atomic
Operations, Cursor Pagination, extension registration, recommendations, and
authority precedence have no materially overlapping maintained peer in that
library and remain explicitly unassessed rather than inferred.

## Decision evidence

| Decision | Conformance boundary |
| --- | --- |
| JSONAPI-DEC-001 | Non-compliant and unrecognized members |
| JSONAPI-DEC-002 | Extension namespaces and profile authority |
| JSONAPI-DEC-003 | Query parameter families |
| JSONAPI-DEC-004 | Relationship objects and linkage |
| JSONAPI-DEC-005 | Included-resource uniqueness and full linkage |
| JSONAPI-DEC-006 | Sparse fieldsets |
| JSONAPI-DEC-007 | Pagination parameters and links |
| JSONAPI-DEC-008 | Atomic operation order and rollback |
| JSONAPI-DEC-009 | Recommendation status |
| JSONAPI-DEC-010 | Conflicts between authorities |
| JSONAPI-DEC-011 | Duplicate JSON object members |
| JSONAPI-DEC-012 | Link relation type validation |
| JSONAPI-DEC-013 | HTTP media range precedence and ties |

When a governing specification or accepted erratum changes a represented
document, retain the old fixture in history, update the fixture from the
authoritative section, validate it with `jq -e .`, refresh its digest and byte
count with `shasum -a 256` and `wc -c`, and rerun the conformance and
specification gates.

When a pinned peer changes, update the nested module dependency, reproduce the
matrix with `go -C interoperability run .`, classify each changed result, and
review the affected decision before updating `differential.tsv` and its current
decision digest.
