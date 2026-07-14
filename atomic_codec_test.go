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
