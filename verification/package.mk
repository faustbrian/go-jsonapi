.PHONY: conformance docs

conformance:
	go test . -count=1

docs:
	./scripts/check-docs.sh
