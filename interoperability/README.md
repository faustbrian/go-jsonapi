# JSON:API interoperability harness

This internal, non-releasable module compares selected JSON:API decisions with
the pinned `github.com/DataDog/jsonapi` v0.13.0 maintained peer. It exists only
as attributable conformance evidence for the public
`github.com/faustbrian/go-jsonapi/v2` module, published as v2.0.0.

The harness keeps its existing internal module identity. Standalone execution
with `GOWORK=off` selects the published v2.0.0 release without replacements and
reports its actual selected module version. The repository workspace instead
selects the current root source and reports that distinct workspace identity.

The harness is not an installable library and does not define application
compatibility policy. Run it from the repository root with
`make interoperability`.

See the versioned [Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/README.md)
for package-family and ownership guidance.
