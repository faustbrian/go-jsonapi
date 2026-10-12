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

Do not fork shared verification workflows, add permanent replacements, machine-specific
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

## Selected Monthly Assurance

The owner-level [selected-assurance caller](.github/workflows/assurance.yml) is
configured for main only, on day 11 of each month at 05:41 UTC. It selects one
of the 9 current declared root-module fuzz targets using the shared UTC-week
rotation, with 30 seconds of active exploration. The selected risks are:
canonical and Atomic document round trips, query determinism, cursor bounds,
media negotiation, registered members and cursor metadata.
The root targets are in-process codec and protocol checks. The owned nested
interoperability module and historical manual diagnostics retain their separate
full contracts; this lane does not qualify maintained peers or Atomic execution.

Base-free ordinary checks execute deterministic seeds, not active exploration;
committed-range CI selects affected active smoke. Manual assurance requires an
exact anchored selector from the root fuzz operation in `.golib.yaml`, for
example `^FuzzUnmarshal$`.
Empty, unknown and malformed selections refuse. The shared job bounds setup
and execution to ten minutes, active workers and CPU to two, owned memory to
2 GiB, native output to 1 MiB and private-cache lifetime. Failure, cancellation,
exhausted budgets and cleanup refusal remain attributable failures. This lane
does not run mutation or replace Required, original full fuzz budgets,
explicit `make check-full`, or relevant release qualification.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.

## Source Documentation

Install the checksum-pinned public verifier declared in `.golib.yaml`.
`make source-docs-check` validates objective Go comment, marker, link and
generated-producer structure; `make check`, `make check-full` and `make ci`
require it before their existing checks. Shared CI runs the same strict
contract and includes its result in Required. This is structural enforcement,
not a substitute for source-truth review or behavioral verification.
