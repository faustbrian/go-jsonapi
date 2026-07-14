# go-jsonapi

`go-jsonapi` is a strict, framework-agnostic implementation of JSON:API 1.1
for Go. It models and validates core documents, implements the official Atomic
Operations extension and Cursor Pagination profile, and provides explicit
query and content-negotiation primitives for `net/http` applications.

The package is transport-agnostic: it does not choose a router, persistence
layer, filtering language, cursor encoding, or domain-error policy.

## Status

The repository is preparing its first public release. Production package code
has meaningful 100% statement coverage, but the API remains pre-v1 until a
`v1.0.0` tag is published. See the [compatibility policy](docs/compatibility.md)
before adopting an unreleased revision.

Implemented and verified:

- JSON:API 1.1 document modeling, strict decoding, validation, and stable
  serialization
- compound documents, relationships, links, meta, errors, local identifiers,
  and request/response validation contexts
- sparse fieldsets, inclusion paths, sorting, page/filter hooks, and
  explicitly registered query families
- JSON:API media type negotiation, extension parameters, profile parameters,
  quality values, and wildcard `Accept` candidates
- the official Atomic Operations extension, including transaction execution
- the official Cursor Pagination profile's parsing, metadata, links, and error
  contracts
- explicitly registered extension members and profile validators
- UTF-8 enforcement and bounded decoding before semantic validation

The detailed evidence is in the [feature matrix](docs/features.md) and
[conformance matrix](docs/conformance.md). Recommendations are tracked
separately because they are not normative requirements.

## Requirements

- Go 1.24 or later

## Install

```sh
go get github.com/faustbrian/go-jsonapi
```

## Quickstart

```go
package main

import (
	"fmt"
	"log"

	jsonapi "github.com/faustbrian/go-jsonapi"
)

func main() {
	document := jsonapi.Document{Data: jsonapi.ResourceData(
		jsonapi.ResourceObject{
			Type: "articles",
			ID:   "1",
			Attributes: jsonapi.Attributes{
				"title": "JSON:API in Go",
			},
		},
	)}

	payload, err := jsonapi.Marshal(document)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(payload))
}
```

Output:

```json
{"data":{"type":"articles","id":"1","attributes":{"title":"JSON:API in Go"}}}
```

Decode untrusted input with the same strict boundary:

```go
document, err := jsonapi.Unmarshal(requestBody)
if err != nil {
	var decodeError *jsonapi.DecodeError
	var validationError *jsonapi.ValidationError
	// Map these typed errors to the HTTP response policy of your service.
}
```

Use `UnmarshalWith` and `MarshalWith` when the endpoint has a request-specific
validation context. Use a configured `Codec` when registering extension
members or profile semantics.

## HTTP integration

The package deliberately returns protocol data instead of writing responses.
A plain `net/http` handler typically:

1. validates `Content-Type` with `Negotiator.CheckContentType`;
2. selects the response representation with `Negotiator.NegotiateAccept`;
3. parses `r.URL.Query()` with `ParseQuery` or a configured `QueryParser`;
4. decodes and validates the document;
5. maps application data to `Document` or `AtomicDocument`;
6. marshals the result and writes the negotiated content type.

See the [adoption guide](docs/adoption.md) for a complete handler and the
[cookbook](docs/cookbook.md) for common integration patterns.

## Design boundaries

- no ORM or database assumptions
- no router or dependency-injection integration
- no reflection-based domain model mapping
- no built-in filter language or cursor encoding
- no hidden HTTP status selection
- no unregistered custom members in JSON:API-defined objects

These boundaries keep wire behavior explicit and let services compose their
own application semantics without forking the protocol layer.

## Documentation

- [Documentation index](docs/README.md)
- [Quickstart](docs/quickstart.md)
- [Architecture](docs/architecture.md)
- [Public API reference](docs/api-reference.md)
- [Supported features](docs/features.md)
- [Conformance evidence](docs/conformance.md)
- [Extensions and profiles](docs/extensions-and-profiles.md)
- [Recommendations](docs/recommendations.md)
- [Adoption guide](docs/adoption.md)
- [End-to-end examples](docs/examples.md)
- [Cookbook](docs/cookbook.md)
- [FAQ](docs/faq.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Migration notes](docs/migration.md)
- [Compatibility policy](docs/compatibility.md)
- [Release guide](docs/releasing.md)
- [Performance](docs/performance.md)
- [Hardening report](docs/hardening-report.md)
- [Threat model](docs/threat-model.md)
- [Roadmap](ROADMAP.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)

## Verification

```sh
gofmt -l .
go vet ./...
go test ./...
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go test ./... -run '^$' -bench . -benchmem
```

Fuzz targets and CI commands are documented in
[performance and resilience testing](docs/performance.md).
Application-supplied extension/profile validators and cursor/sort hooks are
contained and redacted as typed `CallbackError` values; retained causes and
panic values are for trusted diagnostics only.

## Contributing

Issues and pull requests are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md)
before proposing protocol behavior: normative JSON:API requirements,
recommendations, extensions, profiles, and application conventions are
reviewed as distinct categories.

## License

`go-jsonapi` is available under the [MIT License](LICENSE).
