package jsonapi

import (
	"errors"
	"testing"
)

func TestCodecRoundTripsRegisteredResourceExtensionMember(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{Extensions: []ExtensionDefinition{{
		URI:       "https://example.com/ext/version",
		Namespace: "version",
		Members: []MemberDefinition{{
			Scope: ResourceMemberScope,
			Name:  "version:id",
			Validate: func(value any) error {
				if _, ok := value.(string); !ok {
					return errors.New("version id must be a string")
				}
				return nil
			},
		}},
	}}})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}

	payload := []byte(`{
		"data":{"type":"articles","id":"1","version:id":"42"},
		"jsonapi":{"version":"1.1","ext":["https://example.com/ext/version"]}
	}`)
	document, err := codec.Unmarshal(payload)
	if err != nil {
		t.Fatalf("decode extension document: %v", err)
	}
	resource := document.Data.one
	if resource == nil || resource.AdditionalMembers["version:id"] != "42" {
		t.Fatalf("extension member was not preserved: %#v", resource)
	}

	encoded, err := codec.Marshal(document)
	if err != nil {
		t.Fatalf("encode extension document: %v", err)
	}
	want := `{"jsonapi":{"version":"1.1","ext":["https://example.com/ext/version"]},"data":{"type":"articles","id":"1","version:id":"42"}}`
	if string(encoded) != want {
		t.Fatalf("unexpected extension document:\n got: %s\nwant: %s", encoded, want)
	}
}

func TestCodecRejectsInvalidRegisteredMemberValue(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{Extensions: []ExtensionDefinition{{
		URI:       "https://example.com/ext/version",
		Namespace: "version",
		Members: []MemberDefinition{{
			Scope: ResourceMemberScope,
			Name:  "version:id",
			Validate: func(value any) error {
				if _, ok := value.(string); !ok {
					return errors.New("version id must be a string")
				}
				return nil
			},
		}},
	}}})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}

	_, err = codec.Unmarshal([]byte(`{
		"data":{"type":"articles","id":"1","version:id":42}
	}`))
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !hasViolation(validationError, "/data/version:id", "member-value") {
		t.Fatalf("unexpected violations: %#v", validationError.Violations)
	}
}

func TestCoreCodecRejectsUnregisteredExtensionMember(t *testing.T) {
	t.Parallel()

	_, err := Unmarshal([]byte(`{
		"data":{"type":"articles","id":"1","version:id":"42"}
	}`))
	var decodeError *DecodeError
	if !errors.As(err, &decodeError) || decodeError.Path != "/data/version:id" ||
		decodeError.Code != "unknown-member" {
		t.Fatalf("unexpected unregistered member error: %T %#v", err, decodeError)
	}
}

func TestCoreMarshalRejectsUnregisteredExtensionMember(t *testing.T) {
	t.Parallel()

	document := Document{Data: ResourceData(ResourceObject{
		Type:              "articles",
		ID:                "1",
		AdditionalMembers: Members{"version:id": "42"},
	})}
	_, err := Marshal(document)
	var validationError *ValidationError
	if !errors.As(err, &validationError) ||
		!hasViolation(validationError, "/data/version:id", "unregistered-member") {
		t.Fatalf("unexpected marshal error: %T %#v", err, validationError)
	}
}

func TestNewCodecRejectsInvalidExtensionDefinitions(t *testing.T) {
	t.Parallel()

	tests := []CodecOptions{
		{Extensions: []ExtensionDefinition{{URI: "/relative", Namespace: "version"}}},
		{Extensions: []ExtensionDefinition{{URI: "https://example.com/ext", Namespace: "bad-name"}}},
		{Extensions: []ExtensionDefinition{{
			URI:       "https://example.com/ext",
			Namespace: "version",
			Members: []MemberDefinition{{
				Scope: ResourceMemberScope,
				Name:  "other:id",
			}},
		}}},
		{Extensions: []ExtensionDefinition{
			{
				URI:       "https://example.com/one",
				Namespace: "one",
				Members:   []MemberDefinition{{Scope: ResourceMemberScope, Name: "one:id"}},
			},
			{
				URI:       "https://example.com/two",
				Namespace: "two",
				Members:   []MemberDefinition{{Scope: ResourceMemberScope, Name: "one:id"}},
			},
		}},
	}
	for _, options := range tests {
		if _, err := NewCodec(options); err == nil {
			t.Fatalf("expected invalid codec options: %#v", options)
		}
	}
}

func TestCodecRejectsUnregisteredApplicationMemberOnMarshal(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}
	document := Document{Data: ResourceData(ResourceObject{
		Type:              "articles",
		ID:                "1",
		AdditionalMembers: Members{"version:id": "42"},
	})}
	_, err = codec.Marshal(document)
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !hasViolation(validationError, "/data/version:id", "unregistered-member") {
		t.Fatalf("unexpected violations: %#v", validationError.Violations)
	}
}

func TestCodecRoundTripsTopLevelExtensionMemberAsSemanticContent(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{Extensions: []ExtensionDefinition{{
		URI:       "https://example.com/ext/version",
		Namespace: "version",
		Members: []MemberDefinition{{
			Scope: TopLevelMemberScope,
			Name:  "version:manifest",
		}},
	}}})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}

	payload := []byte(`{"version:manifest":{"revision":42}}`)
	document, err := codec.Unmarshal(payload)
	if err != nil {
		t.Fatalf("decode top-level extension document: %v", err)
	}
	manifest, ok := document.AdditionalMembers["version:manifest"].(map[string]any)
	if !ok || manifest["revision"] == nil {
		t.Fatalf("top-level member was not preserved: %#v", document.AdditionalMembers)
	}
	encoded, err := codec.Marshal(document)
	if err != nil {
		t.Fatalf("encode top-level extension document: %v", err)
	}
	if string(encoded) != string(payload) {
		t.Fatalf("unexpected round trip: got %s, want %s", encoded, payload)
	}
}

func TestCoreCodecRejectsTopLevelExtensionMemberAsSemanticContent(t *testing.T) {
	t.Parallel()

	_, err := Unmarshal([]byte(`{"version:manifest":{"revision":42}}`))
	var decodeError *DecodeError
	if !errors.As(err, &decodeError) || decodeError.Path != "/version:manifest" ||
		decodeError.Code != "unknown-member" {
		t.Fatalf("unexpected core decode error: %T %#v", err, decodeError)
	}

	_, err = Marshal(Document{
		AdditionalMembers: Members{"version:manifest": map[string]any{"revision": 42}},
	})
	var validationError *ValidationError
	if !errors.As(err, &validationError) ||
		!hasViolation(validationError, "/version:manifest", "unregistered-member") {
		t.Fatalf("unexpected core marshal error: %T %#v", err, validationError)
	}
}

func TestCodecRoundTripsRelationshipExtensionMemberAsSemanticContent(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{Extensions: []ExtensionDefinition{{
		URI:       "https://example.com/ext/version",
		Namespace: "version",
		Members: []MemberDefinition{{
			Scope: RelationshipMemberScope,
			Name:  "version:state",
		}},
	}}})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}

	payload := []byte(`{"data":{"type":"articles","id":"1","relationships":{"history":{"version:state":"archived"}}}}`)
	document, err := codec.Unmarshal(payload)
	if err != nil {
		t.Fatalf("decode relationship extension document: %v", err)
	}
	relationship := document.Data.one.Relationships["history"]
	if relationship.AdditionalMembers["version:state"] != "archived" {
		t.Fatalf("relationship member was not preserved: %#v", relationship)
	}
	encoded, err := codec.Marshal(document)
	if err != nil {
		t.Fatalf("encode relationship extension document: %v", err)
	}
	if string(encoded) != string(payload) {
		t.Fatalf("unexpected round trip: got %s, want %s", encoded, payload)
	}
}

func TestCoreCodecRejectsRelationshipExtensionMember(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"data":{"type":"articles","id":"1","relationships":{"history":{"version:state":"archived"}}}}`)
	_, err := Unmarshal(payload)
	var decodeError *DecodeError
	if !errors.As(err, &decodeError) ||
		decodeError.Path != "/data/relationships/history/version:state" ||
		decodeError.Code != "unknown-member" {
		t.Fatalf("unexpected core decode error: %T %#v", err, decodeError)
	}

	_, err = Marshal(Document{Data: ResourceData(ResourceObject{
		Type: "articles",
		ID:   "1",
		Relationships: Relationships{"history": {
			AdditionalMembers: Members{"version:state": "archived"},
		}},
	})})
	var validationError *ValidationError
	if !errors.As(err, &validationError) ||
		!hasViolation(
			validationError,
			"/data/relationships/history/version:state",
			"unregistered-member",
		) {
		t.Fatalf("unexpected core marshal error: %T %#v", err, validationError)
	}
}

func TestCodecRejectsInvalidRelationshipExtensionMemberValue(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{Extensions: []ExtensionDefinition{{
		URI:       "https://example.com/ext/version",
		Namespace: "version",
		Members: []MemberDefinition{{
			Scope: RelationshipMemberScope,
			Name:  "version:state",
			Validate: func(value any) error {
				if value != "archived" {
					return errors.New("state must be archived")
				}
				return nil
			},
		}},
	}}})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}

	_, err = codec.Unmarshal([]byte(`{"data":{"type":"articles","id":"1","relationships":{"history":{"version:state":"active"}}}}`))
	var validationError *ValidationError
	if !errors.As(err, &validationError) ||
		!hasViolation(
			validationError,
			"/data/relationships/history/version:state",
			"member-value",
		) {
		t.Fatalf("unexpected value error: %T %#v", err, validationError)
	}
}

func TestCodecRoundTripsIdentifierExtensionMember(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{Extensions: []ExtensionDefinition{{
		URI:       "https://example.com/ext/version",
		Namespace: "version",
		Members: []MemberDefinition{{
			Scope: IdentifierMemberScope,
			Name:  "version:etag",
		}},
	}}})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}

	payload := []byte(`{"data":{"type":"articles","id":"1","relationships":{"author":{"data":{"type":"people","id":"9","version:etag":"abc"}}}}}`)
	document, err := codec.Unmarshal(payload)
	if err != nil {
		t.Fatalf("decode identifier extension document: %v", err)
	}
	identifier := document.Data.one.Relationships["author"].Data.one
	if identifier == nil || identifier.AdditionalMembers["version:etag"] != "abc" {
		t.Fatalf("identifier member was not preserved: %#v", identifier)
	}
	encoded, err := codec.Marshal(document)
	if err != nil {
		t.Fatalf("encode identifier extension document: %v", err)
	}
	if string(encoded) != string(payload) {
		t.Fatalf("unexpected round trip: got %s, want %s", encoded, payload)
	}
}

func TestCoreCodecRejectsIdentifierExtensionMember(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"data":{"type":"articles","id":"1","relationships":{"author":{"data":{"type":"people","id":"9","version:etag":"abc"}}}}}`)
	_, err := Unmarshal(payload)
	var decodeError *DecodeError
	if !errors.As(err, &decodeError) ||
		decodeError.Path != "/data/relationships/author/data/version:etag" ||
		decodeError.Code != "unknown-member" {
		t.Fatalf("unexpected core decode error: %T %#v", err, decodeError)
	}

	_, err = Marshal(Document{Data: ResourceData(ResourceObject{
		Type: "articles",
		ID:   "1",
		Relationships: Relationships{"author": {
			Data: ToOne(Identifier{
				Type:              "people",
				ID:                "9",
				AdditionalMembers: Members{"version:etag": "abc"},
			}),
		}},
	})})
	var validationError *ValidationError
	if !errors.As(err, &validationError) ||
		!hasViolation(
			validationError,
			"/data/relationships/author/data/version:etag",
			"unregistered-member",
		) {
		t.Fatalf("unexpected core marshal error: %T %#v", err, validationError)
	}
}

func TestCodecRoundTripsToManyIdentifierExtensionMembers(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{Extensions: []ExtensionDefinition{{
		URI:       "https://example.com/ext/version",
		Namespace: "version",
		Members: []MemberDefinition{{
			Scope: IdentifierMemberScope,
			Name:  "version:etag",
		}},
	}}})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}

	payload := []byte(`{"data":{"type":"articles","id":"1","relationships":{"comments":{"data":[{"type":"comments","id":"1","version:etag":"a"},{"type":"comments","id":"2","version:etag":"b"}]}}}}`)
	document, err := codec.Unmarshal(payload)
	if err != nil {
		t.Fatalf("decode to-many identifier members: %v", err)
	}
	identifiers := document.Data.one.Relationships["comments"].Data.many
	if len(identifiers) != 2 ||
		identifiers[0].AdditionalMembers["version:etag"] != "a" ||
		identifiers[1].AdditionalMembers["version:etag"] != "b" {
		t.Fatalf("identifier members were not preserved: %#v", identifiers)
	}
	encoded, err := codec.Marshal(document)
	if err != nil {
		t.Fatalf("encode to-many identifier members: %v", err)
	}
	if string(encoded) != string(payload) {
		t.Fatalf("unexpected round trip: got %s, want %s", encoded, payload)
	}
}
