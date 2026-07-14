# Performance and resilience

Correctness is the primary performance constraint: the package does not trade
strict validation or deterministic output for speculative speed.

## Representative benchmarks

`benchmark_test.go` covers:

- marshal/unmarshal of one resource;
- marshal/unmarshal of 100-resource collections;
- marshal/unmarshal of a 100-resource compound document;
- marshal/unmarshal of 100 Atomic operations;
- query parsing;
- `Accept` negotiation.

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

Run one target with an anchored name:

```sh
go test ./... -run '^$' -fuzz '^FuzzNegotiation$' -fuzztime=30s
```

CI uses short deterministic fuzz smoke runs. Longer fuzzing belongs in a
scheduled workflow because fuzzing every target on each pull request has
unbounded cost.

## Operational limits

The package accepts complete byte slices and has no hidden global limits. HTTP
applications should bound request bodies, query/header sizes, collection page
sizes, and included-resource expansion before invoking the codec. Cursor and
Atomic adapters should honor context cancellation in application callbacks.
