# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected package's docs.
2. Run `GOWORK=off go test ./...` or the affected package's narrower tests.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected stable entries in the
[specification decision register](docs/specification-decisions.md), and complete
the Specification Decisions section of the pull request template. An
unresolved interpretation or stale source pin is release-blocking; peer
behavior cannot silently select policy.

Mutation is optional local or scheduled assurance, not a required PR status.
Confirmed behavioral or security defects still require a focused regression
and repair; retain reviewed narrow equivalent/invalid-mutant dispositions.

Do not add package-local workflows, permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Standalone `GOWORK=off go test ./...` runs deterministic tests, fuzz seeds and
saved corpus regressions without sibling checkouts or ecosystem ledgers. The
optional checked-in workspace contains only this repository and its intentional
interoperability module. Neither module declares required services, build tags
or test tags.

With the checksum-pinned public verifier selected by `.golib.yaml`, use:

```bash
make inventory
make check
make ci
```

`check` and `ci` select bounded local verification. Ordinary CI consumes the
public verifier without source bootstrapping. Existing manual diagnostic and
full root release selectors remain separate, with their source-development
version semantics, refusal, evidence and cleanup contracts intact.

Select one target for short active fuzz exploration. Optional local mutation
selects the root package; neither command is an all-change prerequisite:

```bash
GOWORK=off GOMAXPROCS=2 go test -run '^$' -fuzz '^FuzzUnmarshal$' -fuzztime=2s -parallel=1 -timeout=2m .
golib mutation --module . --package .
```

Protocol, codec ownership, callback and Atomic transaction changes retain their
affected conformance, concurrency, cancellation/rollback, compatibility and
direct-consumer checks. Use `make interoperability` for relevant maintained-peer
decisions; it tests the owned nested module without adding peer dependencies to
the public module. Existing fixtures and independent specification policy remain
authoritative.

`make check-full` retains full operations, all nine native 2s fuzz budgets,
conformance, interoperability, documentation and benchmarks. Relevant release
qualification remains `golib release dry-run --module .` and the selected full
root release job; see [releasing](docs/releasing.md). Select full/release checks
for a named risk or release boundary, not every commit. Report unavailable,
skipped and failed boundaries honestly; local checks are not hosted-CI or
public-release qualification.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.
