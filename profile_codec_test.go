package jsonapi

import (
	"errors"
	"testing"
)

func TestCodecAppliesRegisteredProfileDocumentValidation(t *testing.T) {
	t.Parallel()

	profileError := errors.New("timestamps attribute is required")
	codec, err := NewCodec(CodecOptions{Profiles: []ProfileDefinition{{
		URI: "https://example.com/profiles/timestamps",
		ValidateDocument: func(document Document) error {
			if document.Data == nil || document.Data.one == nil ||
				document.Data.one.Attributes["timestamps"] == nil {
				return profileError
			}
			return nil
		},
	}}})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}

	payload := []byte(`{"data":{"type":"articles","id":"1"}}`)
	if _, err := codec.Unmarshal(payload); !errors.Is(err, profileError) {
		t.Fatalf("unexpected profile decode error: %v", err)
	}
	if _, err := codec.Marshal(Document{Data: ResourceData(ResourceObject{
		Type: "articles",
		ID:   "1",
	})}); !errors.Is(err, profileError) {
		t.Fatalf("unexpected profile encode error: %v", err)
	}

	valid := Document{Data: ResourceData(ResourceObject{
		Type:       "articles",
		ID:         "1",
		Attributes: Attributes{"timestamps": map[string]any{"created": "now"}},
	})}
	if _, err := codec.Marshal(valid); err != nil {
		t.Fatalf("validate profile document: %v", err)
	}
}

func TestNewCodecRejectsInvalidProfileDefinitions(t *testing.T) {
	t.Parallel()

	tests := []CodecOptions{
		{Profiles: []ProfileDefinition{{URI: "/relative"}}},
		{Profiles: []ProfileDefinition{
			{URI: "https://example.com/profiles/one"},
			{URI: "https://example.com/profiles/one"},
		}},
	}
	for _, options := range tests {
		if _, err := NewCodec(options); err == nil {
			t.Fatalf("expected invalid codec options: %#v", options)
		}
	}
}
