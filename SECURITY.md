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
