# Migration notes

## From direct `encoding/json`

1. replace ad hoc response maps with `Document` and `ResourceObject`;
2. choose explicit primary/relationship data constructors;
3. map domain values to attributes and relationships;
4. replace `json.Marshal`/`json.Unmarshal` at protocol boundaries with the
   package codec;
5. map typed errors to your HTTP error policy;
6. add canonical fixtures before switching production responses.

Expect stricter behavior: duplicate members, unknown members, invalid links,
identity conflicts, and compound-document linkage errors will no longer pass
silently.

## From reflection/tag-based JSON:API libraries

Create a small adapter per resource type rather than porting tags mechanically.
Audit:

- resource type and ID formatting;
- absent versus null versus empty collections;
- relationship linkage and links;
- included-resource deduplication;
- attribute naming and number representation;
- error pointer construction;
- query and media negotiation previously handled by middleware.

Explicit adapters are more code than tags but make protocol compatibility
visible and testable.

## Wire-format comparison

Do not compare only decoded domain values. Capture and compare:

- member presence;
- null versus omitted values;
- relationship linkage;
- included identities;
- link string versus object forms;
- exact numbers;
- extension/profile media type parameters;
- deterministic bytes where clients or caches depend on them.

## Incremental adoption

Use the package first at one endpoint boundary. Keep the old mapping behind a
feature flag or shadow comparison, build golden fixtures from accepted cases,
and monitor rejection codes. Expand only after mismatches are classified as an
old bug, a new bug, or an intentional compatibility change.

## Future release migrations

Breaking changes before v1 and all migrations after v1 are recorded in
`CHANGELOG.md`. A stable release will not remove or redefine exported APIs or
wire behavior without a new major version.
