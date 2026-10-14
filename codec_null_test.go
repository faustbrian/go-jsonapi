package jsonapi

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestUnmarshalRejectsNullStringMembers(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, payload, path string }{
		{"version", `{"jsonapi":{"version":null},"data":null}`, "/jsonapi/version"},
		{"resource type", `{"data":{"type":null,"id":"1"}}`, "/data/type"},
		{"resource id", `{"data":{"type":"articles","id":null}}`, "/data/id"},
		{"resource lid", `{"data":{"type":"articles","lid":null}}`, "/data/lid"},
		{"relationship type", `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":{"type":null,"id":"2"}}}}}`, "/data/relationships/author/data/type"},
		{"relationship id", `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":{"type":"people","id":null}}}}}`, "/data/relationships/author/data/id"},
		{"relationship lid", `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":{"type":"people","lid":null}}}}}`, "/data/relationships/author/data/lid"},
		{"link href", `{"data":null,"links":{"self":{"href":null}}}`, "/links/self/href"},
		{"link rel", `{"data":null,"links":{"self":{"href":"/articles","rel":null}}}`, "/links/self/rel"},
		{"link title", `{"data":null,"links":{"self":{"href":"/articles","title":null}}}`, "/links/self/title"},
		{"link type", `{"data":null,"links":{"self":{"href":"/articles","type":null}}}`, "/links/self/type"},
		{"error id", `{"errors":[{"id":null}]}`, "/errors/0/id"},
		{"error status", `{"errors":[{"status":null}]}`, "/errors/0/status"},
		{"error code", `{"errors":[{"code":null}]}`, "/errors/0/code"},
		{"error title", `{"errors":[{"title":null}]}`, "/errors/0/title"},
		{"error detail", `{"errors":[{"detail":null}]}`, "/errors/0/detail"},
		{"source pointer", `{"errors":[{"source":{"pointer":null}}]}`, "/errors/0/source/pointer"},
		{"source parameter", `{"errors":[{"source":{"parameter":null}}]}`, "/errors/0/source/parameter"},
		{"source header", `{"errors":[{"source":{"header":null}}]}`, "/errors/0/source/header"},
	}
	codec, err := NewCodec(CodecOptions{})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}
	decoders := []struct {
		name   string
		decode func([]byte) (Document, error)
	}{
		{"core", Unmarshal},
		{"registered codec", codec.Unmarshal},
	}
	for _, decoder := range decoders {
		t.Run(decoder.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					_, err := decoder.decode([]byte(tt.payload))
					var decoded *DecodeError
					if !errors.As(err, &decoded) || decoded.Path != tt.path || decoded.Code != "type" {
						t.Fatalf("got %v; want type DecodeError at %s", err, tt.path)
					}
				})
			}
		})
	}
}

func TestUnmarshalAtomicRejectsNullStringMembers(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, payload, path string }{
		{"op", `{"atomic:operations":[{"op":null,"ref":{"type":"articles","id":"1"}}]}`, "/atomic:operations/0/op"},
		{"href", `{"atomic:operations":[{"op":"remove","href":null}]}`, "/atomic:operations/0/href"},
		{"ref type", `{"atomic:operations":[{"op":"remove","ref":{"type":null,"id":"1"}}]}`, "/atomic:operations/0/ref/type"},
		{"ref id", `{"atomic:operations":[{"op":"remove","ref":{"type":"articles","id":null}}]}`, "/atomic:operations/0/ref/id"},
		{"ref lid", `{"atomic:operations":[{"op":"remove","ref":{"type":"articles","lid":null}}]}`, "/atomic:operations/0/ref/lid"},
		{"ref relationship", `{"atomic:operations":[{"op":"remove","ref":{"type":"articles","id":"1","relationship":null}}]}`, "/atomic:operations/0/ref/relationship"},
		{"data id", `{"atomic:operations":[{"op":"add","data":{"type":"articles","id":null}}]}`, "/atomic:operations/0/data/id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalAtomic([]byte(tt.payload))
			var decoded *DecodeError
			if !errors.As(err, &decoded) || decoded.Path != tt.path || decoded.Code != "type" {
				t.Fatalf("got %v; want type DecodeError at %s", err, tt.path)
			}
		})
	}
}

func TestNullStringRefusalPreservesValidPresence(t *testing.T) {
	t.Parallel()
	for _, payload := range []string{
		`{"data":null}`,
		`{"data":{"type":"articles","id":""}}`,
		`{"data":{"type":"articles","lid":""}}`,
		`{"data":{"type":"articles","id":"1","attributes":{"value":null},"relationships":{"author":{"data":null}}},"links":{"self":null}}`,
	} {
		doc, err := Unmarshal([]byte(payload))
		if err != nil {
			t.Fatalf("valid payload rejected: %s: %v", payload, err)
		}
		encoded, err := Marshal(doc)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, err := Unmarshal(encoded); err != nil {
			t.Fatalf("round trip: %v", err)
		}
		assertNullStringControlMembers(t, payload, encoded)
	}
	for _, payload := range []string{
		`{"atomic:operations":[{"op":"remove","ref":{"type":"articles","id":""}}]}`,
		`{"atomic:operations":[{"op":"add","data":{"type":"articles","lid":""}},{"op":"remove","ref":{"type":"articles","lid":""}}]}`,
	} {
		doc, err := UnmarshalAtomic([]byte(payload))
		if err != nil {
			t.Fatalf("valid Atomic payload rejected: %s: %v", payload, err)
		}
		encoded, err := MarshalAtomic(doc)
		if err != nil {
			t.Fatalf("marshal Atomic: %v", err)
		}
		assertNullStringControlMembers(t, payload, encoded)
	}
}

func assertNullStringControlMembers(t *testing.T, original string, encoded []byte) {
	t.Helper()
	var before, after any
	if err := json.Unmarshal([]byte(original), &before); err != nil {
		t.Fatalf("parse control: %v", err)
	}
	if err := json.Unmarshal(encoded, &after); err != nil {
		t.Fatalf("parse encoded control: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("valid member presence changed: got %s, want %s", encoded, original)
	}
}
