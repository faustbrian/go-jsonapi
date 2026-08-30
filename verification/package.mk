.PHONY: conformance docs interoperability

conformance:
	go test . -count=1

docs:
	./scripts/check-docs.sh

interoperability:
	GOWORK=auto go -C interoperability test ./...
