package jsonapi

import (
	"errors"
	"testing"
)

func TestUnmarshalProducesCanonicalValidatedDocument(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"meta":{"requestId":"abc"},
		"included":[{
			"attributes":{"name":"Jane"},
			"id":"9",
			"type":"people"
		}],
		"data":{
			"relationships":{"author":{"data":{"id":"9","type":"people"}}},
			"attributes":{"title":"JSON:API"},
			"id":"1",
			"type":"articles"
		},
		"jsonapi":{"version":"1.1"}
	}`)

	document, err := Unmarshal(payload)
	if err != nil {
		t.Fatalf("unmarshal document: %v", err)
	}
	got, err := Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}

	want := `{"jsonapi":{"version":"1.1"},"data":{"type":"articles","id":"1","attributes":{"title":"JSON:API"},"relationships":{"author":{"data":{"type":"people","id":"9"}}}},"included":[{"type":"people","id":"9","attributes":{"name":"Jane"}}],"meta":{"requestId":"abc"}}`
	if string(got) != want {
		t.Fatalf("unexpected canonical JSON:\n got: %s\nwant: %s", got, want)
	}
}

func TestUnmarshalPreservesNullAndEmptyData(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"null primary data":       `{"data":null}`,
		"empty primary data":      `{"data":[]}`,
		"null relationship data":  `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":null}}}}`,
		"empty relationship data": `{"data":{"type":"articles","id":"1","relationships":{"tags":{"data":[]}}}}`,
	}

	for name, payload := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			document, err := Unmarshal([]byte(payload))
			if err != nil {
				t.Fatalf("unmarshal document: %v", err)
			}
			got, err := Marshal(document)
			if err != nil {
				t.Fatalf("marshal document: %v", err)
			}
			if string(got) != payload {
				t.Fatalf("unexpected round trip: got %s, want %s", got, payload)
			}
		})
	}
}

func TestUnmarshalIgnoresAtMembers(t *testing.T) {
	t.Parallel()

	document, err := Unmarshal([]byte(`{
		"@context":"https://example.com/context",
		"data":{
			"type":"articles",
			"id":"1",
			"@annotation":{"internal":true},
			"attributes":{"title":"JSON:API","@language":"en"}
		}
	}`))
	if err != nil {
		t.Fatalf("unmarshal document: %v", err)
	}
	got, err := Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}
	want := `{"data":{"type":"articles","id":"1","attributes":{"title":"JSON:API"}}}`
	if string(got) != want {
		t.Fatalf("unexpected JSON: got %s, want %s", got, want)
	}
}

func TestUnmarshalRejectsMalformedDocuments(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		payload string
		path    string
		code    string
	}{
		"invalid JSON": {
			payload: `{"data":`,
			path:    "",
			code:    "syntax",
		},
		"root is not object": {
			payload: `[]`,
			path:    "",
			code:    "type",
		},
		"unknown top-level member": {
			payload: `{"data":null,"unknown":true}`,
			path:    "/unknown",
			code:    "unknown-member",
		},
		"unknown resource member": {
			payload: `{"data":{"type":"articles","id":"1","unknown":true}}`,
			path:    "/data/unknown",
			code:    "unknown-member",
		},
		"primary data has scalar shape": {
			payload: `{"data":"articles"}`,
			path:    "/data",
			code:    "type",
		},
		"relationship data has scalar shape": {
			payload: `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":"9"}}}}`,
			path:    "/data/relationships/author/data",
			code:    "type",
		},
		"attributes is not object": {
			payload: `{"data":{"type":"articles","id":"1","attributes":[]}}`,
			path:    "/data/attributes",
			code:    "type",
		},
		"link has invalid shape": {
			payload: `{"data":null,"links":{"self":42}}`,
			path:    "/links/self",
			code:    "type",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := Unmarshal([]byte(test.payload))
			if err == nil {
				t.Fatal("expected decode error")
			}
			var decodeError *DecodeError
			if !errors.As(err, &decodeError) {
				t.Fatalf("expected DecodeError, got %T: %v", err, err)
			}
			if decodeError.Path != test.path || decodeError.Code != test.code {
				t.Fatalf(
					"unexpected error: got path %q code %q, want path %q code %q",
					decodeError.Path,
					decodeError.Code,
					test.path,
					test.code,
				)
			}
		})
	}
}

func TestMarshalRejectsInvalidDocument(t *testing.T) {
	t.Parallel()

	_, err := Marshal(Document{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUnmarshalReportsErrorSourceFieldsDeterministically(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"errors":[{"source":{
		"pointer":1,
		"parameter":2,
		"header":3
	}}]}`)
	for range 100 {
		_, err := Unmarshal(payload)
		var decodeError *DecodeError
		if !errors.As(err, &decodeError) {
			t.Fatalf("expected DecodeError, got %T: %v", err, err)
		}
		if decodeError.Path != "/errors/0/source/pointer" {
			t.Fatalf("unexpected first error source path: %q", decodeError.Path)
		}
	}
}
