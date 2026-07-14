# Contributing

Thank you for helping make `go-jsonapi` a dependable protocol foundation.

## Before opening a change

Use an issue for behavior changes that affect public APIs, wire output,
validation, extensions, profiles, or compatibility. State which category the
proposal belongs to:

- normative JSON:API requirement;
- official extension or profile requirement;
- non-normative JSON:API recommendation;
- application extension point;
- implementation, performance, or documentation improvement.

Do not present application conventions as JSON:API requirements.

## Development setup

Requirements:

- Go 1.24 or later;
- Git;
- `golangci-lint` for the same lint gate used by CI.

Clone the repository and run:

```sh
go mod download
go test ./...
go vet ./...
```

## Change requirements

- Add a regression test before fixing a defect.
- Keep meaningful 100% production-code statement coverage.
- Preserve deterministic wire output unless a documented breaking change is
  intentional.
- Update the conformance matrix for normative behavior changes.
- Update examples and user documentation for public behavior changes.
- Add an entry under `Unreleased` in `CHANGELOG.md`.
- Keep dependencies minimal and explain every addition.

## Local verification

```sh
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go test ./... -run '^Example'
go test ./... -run '^$' -bench . -benchtime=100ms
```

Run each fuzz target for changes near its boundary:

```sh
go test ./... -run '^$' -fuzz '^FuzzUnmarshal$' -fuzztime=30s
go test ./... -run '^$' -fuzz '^FuzzUnmarshalAtomic$' -fuzztime=30s
go test ./... -run '^$' -fuzz '^FuzzParseQuery$' -fuzztime=30s
go test ./... -run '^$' -fuzz '^FuzzCursorPaginationQuery$' -fuzztime=30s
go test ./... -run '^$' -fuzz '^FuzzNegotiation$' -fuzztime=30s
```

## Commit and pull request style

Use focused conventional commits with a body explaining why the change is
needed. Pull requests should include:

- the protocol requirement or problem being solved;
- compatibility and wire-format impact;
- tests and fixtures added;
- verification commands and results;
- documentation and changelog updates.

## Reporting security issues

Do not open a public issue for a suspected vulnerability. Follow
[SECURITY.md](SECURITY.md).
