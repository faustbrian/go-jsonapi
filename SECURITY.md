# Security policy

## Supported versions

Before `v1.0.0`, security fixes are applied to the latest revision of `main`.
After the first stable release, the project will document supported release
lines here and provide fixes for the latest stable minor line.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting for this repository. Include:

- the affected API or parsing surface;
- a minimal reproducer or malformed payload;
- expected and observed behavior;
- potential availability, integrity, or confidentiality impact;
- any suggested mitigation.

Do not include secrets or production data. Please allow maintainers reasonable
time to confirm and coordinate a fix before public disclosure.

## Security posture

The package processes untrusted JSON, query strings, and media type headers.
Its CI therefore includes static analysis, dependency vulnerability scanning,
fuzz-target smoke runs, strict decoding tests, and deterministic re-encoding
checks. The package does not perform authentication, authorization, network
I/O, or persistence.

Core, Atomic, and configured codecs reject malformed UTF-8 and apply bounded
defaults before semantic decoding: 16 MiB per document, 64 composite levels,
10,000 members per object, 100,000 items per array, and 1,000,000 total JSON
values. Applications should still cap HTTP request bodies before allocating a
byte slice. Use `DecodeLimits` only to lower or deliberately raise a boundary
after considering endpoint memory and latency budgets.

Query parsing and media negotiation also use bounded defaults for decoded
names/values, selector and list counts, header bytes/candidates, and
extension/profile URI counts and lengths. Transport servers should retain
their own request-target and header limits so oversized input is rejected
before decoding or allocation.

Application extension/profile validators and cursor/sort hooks run behind a
panic boundary. Failures are available through `errors.Is`/`errors.As` and the
typed `CallbackError`, while public error strings remain generic. Panic values
are retained for trusted inspection only. Do not pass callback causes or panic
values into JSON:API error titles or details. Successful profile validators
must also leave their document view unchanged; mutation is detected and
rejected so a callback cannot invalidate already-checked protocol semantics.

Atomic transaction callback panics are recovered as `AtomicPanicError` values.
The panic value is retained for explicit inspection and error unwrapping but
is not formatted into the default error string. After a transaction begins,
apply, result-validation, commit, cancellation, and panic failures each cause
one rollback attempt using a non-canceled context. A faulty transaction may
still have committed before returning or panicking; the package cannot provide
stronger atomicity than the application implementation.
