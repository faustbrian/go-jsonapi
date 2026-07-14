package jsonapi

import "testing"

func TestRoundTripPreservesExplicitEmptyMembers(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"empty errors array":         `{"errors":[]}`,
		"empty top-level containers": `{"links":{},"data":[],"included":[],"meta":{}}`,
		"empty resource containers":  `{"data":{"type":"articles","id":"1","attributes":{},"relationships":{},"links":{},"meta":{}}}`,
		"empty relationship meta":    `{"data":{"type":"articles","id":"1","relationships":{"author":{"meta":{}}}}}`,
		"empty identifier meta":      `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":{"type":"people","id":"9","meta":{}}}}}}`,
		"empty JSON API containers":  `{"jsonapi":{"ext":[],"profile":[],"meta":{}},"data":null}`,
		"empty link object meta":     `{"links":{"self":{"href":"/articles","meta":{}}},"data":null}`,
		"empty error meta":           `{"errors":[{"meta":{}}]}`,
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

func TestMarshalPreservesEmptyRequiredTopLevelMember(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		document Document
		want     string
	}{
		"errors": {
			document: Document{Errors: []ErrorObject{}},
			want:     `{"errors":[]}`,
		},
		"meta": {
			document: Document{Meta: Meta{}},
			want:     `{"meta":{}}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := Marshal(test.document)
			if err != nil {
				t.Fatalf("marshal document: %v", err)
			}
			if string(got) != test.want {
				t.Fatalf("unexpected JSON: got %s, want %s", got, test.want)
			}
		})
	}
}
