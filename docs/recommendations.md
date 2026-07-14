# JSON:API recommendations

The [official recommendations](https://jsonapi.org/recommendations/) are
useful consistency guidance but are not normative JSON:API requirements. This
matrix deliberately keeps them separate from conformance claims.

| Recommendation area | Package position | Responsibility |
| --- | --- | --- |
| URL design rooted in resource collections | Adopt in examples; not enforced | application router |
| Resource and relationship `self` links | Supported by model; not auto-generated | application |
| Related links for available relationships | Supported by model | application |
| Relationship linkage in resource documents | Fully modeled and validated when present | shared |
| Conventional `filter[...]` family | Preserved as an explicit hook | application defines operators |
| Sparse fieldsets with `fields[type]` | Parsed and validated | application applies selection |
| Inclusion with `include` | Parsed and validated | application loads and builds linkage |
| Sorting with `sort` | Parsed into ordered fields | application maps allowed fields |
| Pagination links | Link model supports all relations | pagination adapter builds URLs |
| `first` and `last` when inexpensive | Supported, not required by core | application |
| Profile/extension-aware self link type | Full link-object type support | application builds value |
| Asynchronous processing with 202 responses | No queue assumptions | application HTTP workflow |
| Method override for clients unable to PATCH | Not implemented in core | optional middleware |
| Error detail consistency | Typed error primitives provided | application presentation policy |

## Adopted conventions in project examples

Examples use plural collection URLs, stable resource type names, relationship
links, bracketed filter families, comma-separated include/field/sort values,
and explicit pagination links. These choices improve consistency but the core
does not reject alternative valid URL designs or application semantics.

## Intentionally not enforced

- URL path naming and pluralization
- router method selection
- filter operator vocabulary
- default include or sparse-fieldset policy
- asynchronous job resources
- method-override headers
- domain error titles and localization

Enforcing these in a transport-neutral document package would blur
recommendations into normative rules and couple unrelated applications.
