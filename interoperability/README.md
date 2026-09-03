# JSON:API interoperability harness

This internal, non-releasable module compares selected JSON:API decisions with
the pinned `github.com/DataDog/jsonapi` v0.13.0 maintained peer. It exists only
as attributable conformance evidence for the public
[`jsonapi`](https://pkg.go.dev/github.com/faustbrian/go-jsonapi) module.

The harness is not an installable library and does not define application
compatibility policy. Run it from the repository root with
`make interoperability`.

See the versioned [Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/README.md)
for package-family and ownership guidance.
