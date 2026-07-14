package jsonapi

import (
	"errors"
	"testing"
)

func TestAtomicCodecCanonicalRequestRoundTrip(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"meta":{"requestId":"abc"},
		"atomic:operations":[{
			"data":{"attributes":{"title":"New"},"type":"articles","lid":"article-1"},
			"href":"/articles",
			"op":"add",
			"meta":{}
		},{
			"data":{"attributes":{"title":"Updated"},"type":"articles","lid":"article-1"},
			"ref":{"lid":"article-1","type":"articles"},
			"op":"update"
		}],
		"jsonapi":{"version":"1.1","ext":["https://jsonapi.org/ext/atomic"]}
	}`)

	document, err := UnmarshalAtomic(payload)
	if err != nil {
		t.Fatalf("unmarshal atomic document: %v", err)
	}
	encoded, err := MarshalAtomic(document)
	if err != nil {
		t.Fatalf("marshal atomic document: %v", err)
	}

	want := `{"jsonapi":{"version":"1.1","ext":["https://jsonapi.org/ext/atomic"]},"atomic:operations":[{"op":"add","href":"/articles","data":{"type":"articles","lid":"article-1","attributes":{"title":"New"}},"meta":{}},{"op":"update","ref":{"type":"articles","lid":"article-1"},"data":{"type":"articles","lid":"article-1","attributes":{"title":"Updated"}}}],"meta":{"requestId":"abc"}}`
	if string(encoded) != want {
		t.Fatalf("unexpected canonical JSON:\n got: %s\nwant: %s", encoded, want)
	}
}

func TestAtomicCodecPreservesEmptyResults(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"atomic:results":[{}, {"meta":{}}]}`)
	want := `{"atomic:results":[{},{"meta":{}}]}`
	document, err := UnmarshalAtomic(payload)
	if err != nil {
		t.Fatalf("unmarshal atomic results: %v", err)
	}
	encoded, err := MarshalAtomic(document)
	if err != nil {
		t.Fatalf("marshal atomic results: %v", err)
	}
	if string(encoded) != want {
		t.Fatalf("unexpected round trip: got %s, want %s", encoded, want)
	}
}

func TestUnmarshalAtomicRejectsForbiddenAndUnknownMembers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		payload string
		path    string
		code    string
	}{
		"invalid JSON": {
			payload: `{"atomic:operations":`,
			path:    "",
			code:    "syntax",
		},
		"root is not object": {
			payload: `[]`,
			path:    "",
			code:    "type",
		},
		"core data": {
			payload: `{"data":null,"atomic:operations":[{"op":"remove","href":"/articles/1"}]}`,
			path:    "/data",
			code:    "forbidden",
		},
		"core included": {
			payload: `{"included":[],"meta":{}}`,
			path:    "/included",
			code:    "forbidden",
		},
		"unknown top-level member": {
			payload: `{"meta":{},"unknown":true}`,
			path:    "/unknown",
			code:    "unknown-member",
		},
		"jsonapi is not object": {
			payload: `{"jsonapi":[],"meta":{}}`,
			path:    "/jsonapi",
			code:    "type",
		},
		"links is not object": {
			payload: `{"links":[],"meta":{}}`,
			path:    "/links",
			code:    "type",
		},
		"operations is not array": {
			payload: `{"atomic:operations":null}`,
			path:    "/atomic:operations",
			code:    "type",
		},
		"operation is not object": {
			payload: `{"atomic:operations":[null]}`,
			path:    "/atomic:operations/0",
			code:    "type",
		},
		"unknown operation member": {
			payload: `{"atomic:operations":[{"op":"remove","href":"/articles/1","unknown":true}]}`,
			path:    "/atomic:operations/0/unknown",
			code:    "unknown-member",
		},
		"duplicate operation member": {
			payload: `{"atomic:operations":[{"op":"remove","op":"add","href":"/articles/1"}]}`,
			path:    "/atomic:operations/0/op",
			code:    "duplicate-member",
		},
		"operation code is not string": {
			payload: `{"atomic:operations":[{"op":1}]}`,
			path:    "/atomic:operations/0/op",
			code:    "type",
		},
		"operation ref is not object": {
			payload: `{"atomic:operations":[{"op":"remove","ref":null}]}`,
			path:    "/atomic:operations/0/ref",
			code:    "type",
		},
		"unknown reference member": {
			payload: `{"atomic:operations":[{"op":"remove","ref":{"type":"articles","id":"1","unknown":true}}]}`,
			path:    "/atomic:operations/0/ref/unknown",
			code:    "unknown-member",
		},
		"operation href is not string": {
			payload: `{"atomic:operations":[{"op":"remove","href":1}]}`,
			path:    "/atomic:operations/0/href",
			code:    "type",
		},
		"operation data has scalar shape": {
			payload: `{"atomic:operations":[{"op":"add","data":1}]}`,
			path:    "/atomic:operations/0/data",
			code:    "type",
		},
		"operation meta is not object": {
			payload: `{"atomic:operations":[{"op":"remove","href":"/articles/1","meta":[]}]}`,
			path:    "/atomic:operations/0/meta",
			code:    "type",
		},
		"results is not array": {
			payload: `{"atomic:results":null}`,
			path:    "/atomic:results",
			code:    "type",
		},
		"result is not object": {
			payload: `{"atomic:results":[null]}`,
			path:    "/atomic:results/0",
			code:    "type",
		},
		"unknown result member": {
			payload: `{"atomic:results":[{"unknown":true}]}`,
			path:    "/atomic:results/0/unknown",
			code:    "unknown-member",
		},
		"result data has scalar shape": {
			payload: `{"atomic:results":[{"data":1}]}`,
			path:    "/atomic:results/0/data",
			code:    "type",
		},
		"result meta is not object": {
			payload: `{"atomic:results":[{"meta":[]}]}`,
			path:    "/atomic:results/0/meta",
			code:    "type",
		},
		"errors is not array": {
			payload: `{"errors":null}`,
			path:    "/errors",
			code:    "type",
		},
		"top-level meta is not object": {
			payload: `{"meta":[]}`,
			path:    "/meta",
			code:    "type",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := UnmarshalAtomic([]byte(test.payload))
			if err == nil {
				t.Fatal("expected decode error")
			}
			var decodeError *DecodeError
			if !errors.As(err, &decodeError) {
				t.Fatalf("expected DecodeError, got %T: %v", err, err)
			}
			if decodeError.Path != test.path || decodeError.Code != test.code {
				t.Fatalf("unexpected decode error: %#v", decodeError)
			}
		})
	}
}

func TestMarshalAtomicRejectsInvalidDocument(t *testing.T) {
	t.Parallel()

	_, err := MarshalAtomic(AtomicDocument{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUnmarshalAtomicReportsReferenceFieldsInDocumentOrder(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"atomic:operations":[{
		"op":"remove",
		"ref":{"type":1,"id":2,"lid":3,"relationship":4}
	}]}`)
	for range 100 {
		_, err := UnmarshalAtomic(payload)
		var decodeError *DecodeError
		if !errors.As(err, &decodeError) {
			t.Fatalf("expected DecodeError, got %T: %v", err, err)
		}
		if decodeError.Path != "/atomic:operations/0/ref/type" {
			t.Fatalf("unexpected first path: %q", decodeError.Path)
		}
	}
}

func TestAtomicCodecAppliesProtocolContext(t *testing.T) {
	t.Parallel()

	_, err := UnmarshalAtomicWith(
		[]byte(`{"atomic:results":[{}]}`),
		AtomicValidationOptions{Context: AtomicRequestContext},
	)
	if err == nil {
		t.Fatal("expected request context violation")
	}

	_, err = MarshalAtomicWith(
		AtomicDocument{Results: []AtomicResult{{}}},
		AtomicValidationOptions{
			Context:             AtomicResponseContext,
			ExpectedResultCount: 2,
		},
	)
	if err == nil {
		t.Fatal("expected response result count violation")
	}
}
