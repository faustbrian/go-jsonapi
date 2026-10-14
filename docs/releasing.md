# Versioning and release guide

[Documentation index](README.md)

Releases are tag-driven and reproducible from a clean `main` commit.

## Prepare a release

1. choose the SemVer version and confirm compatibility impact;
2. move relevant `CHANGELOG.md` entries from `Unreleased` to a dated version;
3. update migration notes for every breaking change;
4. confirm documentation and executable examples reflect the release;
5. complete the applicable root release qualification and rehearsal, using
   authorized hosted execution for system-risk checks;
6. deliver the reviewed preparation to `main` with its required exact-source
   hosted CI green;
7. create a signed annotated `vMAJOR.MINOR.PATCH` tag on that verified source
   only with publication authority;
8. verify the remote tag, prepare the deterministic source archive/checksums,
   and publish the reviewed release notes and assets after qualification.

## Required verification

Ordinary `make check` and `make ci` select bounded local verification, not release
qualification. `make check-full` preserves all configured full operations;
`golib release dry-run --module .` and the existing selected full root release
job retain the explicit release boundary. See
[contributor verification](../CONTRIBUTING.md#verification) for isolated
development, maintained-peer integration and focused active fuzzing.

```sh
test -z "$(gofmt -l .)"
go mod tidy -diff
go vet ./...
go test ./...
go test ./... -race
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go test ./... -run '^Example'
go test ./... -run '^$' -bench . -benchtime=100ms
govulncheck ./...
```

Run all fuzz targets for at least one minute each before a stable release.

## Tag rules

- tags must use canonical `vMAJOR.MINOR.PATCH` numbers, without leading zeros,
  and may include a SemVer prerelease;
- build metadata is not used in release tags;
- the module version and exactly one valid, dated changelog heading must agree;
- the changelog date records release preparation and need not equal the later
  tag-push date;
- tags are never force-updated or reused;
- release artifacts are built from the verified tag commit, not a mutable
  branch;
- release notes are generated from the matching changelog section and commit
  history, then reviewed before publication when practical.

## Release artifacts

Go modules are consumed from the Git tag. The current CI workflows qualify
source and scheduled assurance; they do not publish tags or release entries.
Publication is a separate authorized operation. Build the deterministic source
archive and checksum from the verified tag, retain its source identity, and
upload the reviewed notes and assets to a draft release. Verify the assets and
checksums before making that release public. Publish no library binaries.
Finally verify the remote tag, public release assets and an actual clean
`GOWORK=off` public consumer without replacements before reporting publication
complete. A successful CI status alone does not prove those boundaries.

## Rollback

Published module versions are immutable. Fix a bad release with a new patch
version. Retracting a module version requires a documented reason in `go.mod`
and release notes; do not delete or rewrite the original tag.
