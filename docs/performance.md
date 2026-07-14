# Performance and resilience

Correctness is the primary performance constraint: the package does not trade
strict validation or deterministic output for speculative speed.

## Representative benchmarks

`benchmark_test.go` covers:

- marshal/unmarshal of one resource;
- marshal/unmarshal of 100-resource collections;
- marshal/unmarshal of a 100-resource compound document;
- unmarshal of a 1,000-resource adversarial compound graph;
- marshal/unmarshal of 100 Atomic operations;
- query parsing;
- `Accept` negotiation;
- Cursor Pagination metadata construction and parsing.

Run:

```sh
go test ./... -run '^$' -bench . -benchmem
```

Baseline observed on 2026-07-14 with Go 1.24, macOS arm64, Apple M4 Max
(`-benchtime=100ms`):

| Benchmark | Time | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: |
| Marshal single resource | 2.16 us | 1,394 | 20 |
| Unmarshal single resource | 7.52 us | 10,679 | 151 |
| Marshal 100 resources | 183 us | 173,782 | 1,334 |
| Unmarshal 100 resources | 693 us | 967,758 | 12,842 |
| Marshal compound document | 540 us | 477,315 | 4,355 |
| Unmarshal compound document | 1.83 ms | 2,152,765 | 34,389 |
| Marshal 100 Atomic operations | 50.2 us | 43,774 | 505 |
| Unmarshal 100 Atomic operations | 240 us | 213,430 | 5,539 |
| Parse representative query | 918 ns | 1,712 | 26 |
| Negotiate representative Accept | 2.30 us | 2,288 | 45 |

The adversarial compound and pagination metadata benchmarks are intentionally
listed without frozen numbers until repeated `benchstat` samples are captured
on the release commit. They remain part of every benchmark smoke run.

These numbers are a local reference, not a universal SLA. CI runs benchmarks
as a smoke strategy; release comparisons should use `benchstat` over repeated
runs on comparable hardware before declaring a regression.

## Fuzz targets

| Target | Boundary | Invariant |
| --- | --- | --- |
| `FuzzUnmarshal` | core JSON codec | accepted input marshals and decodes canonically |
| `FuzzUnmarshalAtomic` | Atomic codec | accepted input marshals and decodes canonically |
| `FuzzParseQuery` | query parser | accepted parsing is deterministic |
| `FuzzCursorPaginationQuery` | Cursor profile | accepted size remains within configured bounds |
| `FuzzNegotiation` | media types | canonical/selected content types remain acceptable |
| `FuzzConstructedDocumentValidation` | constructed core documents | repeated validation has a stable outcome |
| `FuzzMemberRegistry` | extension registrations and values | accepted registrations round trip through their codec |
| `FuzzCursorMetadata` | page and item metadata | accepted metadata round trips exactly |
| `FuzzMarshalUnmarshalRoundTrip` | constructed documents | valid UTF-8 models retain canonical bytes across a codec round trip |

Run one target with an anchored name:

```sh
go test ./... -run '^$' -fuzz '^FuzzNegotiation$' -fuzztime=30s
```

CI uses short deterministic fuzz smoke runs. Longer fuzzing belongs in a
scheduled workflow because fuzzing every target on each pull request has
unbounded cost.

## Operational limits

Core, Atomic, and configured document codecs apply `DefaultDecodeLimits`:

| Limit | Default |
| --- | ---: |
| encoded document bytes | 16 MiB |
| nested arrays/objects | 64 |
| members in one object | 10,000 |
| items in one array | 100,000 |
| total JSON values | 1,000,000 |

`DecodeLimits` can lower or raise individual limits; zero selects the default.
HTTP applications must still limit bodies before reading them into memory and
must independently bound query/header sizes, collection page sizes, and
included-resource expansion. Cursor and Atomic adapters should honor context
cancellation in application callbacks.

`DefaultQueryLimits` bounds 100 distinct parameters, 200 values, 1,024 bytes
per decoded name, 8,192 bytes per value, 64 KiB in aggregate, 32 selectors,
and 1,000 comma-list entries. `DefaultNegotiationLimits` bounds headers at
32 KiB, Accept candidates and parameter URI lists at 100, individual URIs at
2,048 bytes, and configured URIs at 1,000.
