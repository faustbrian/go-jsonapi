package jsonapi

import (
	"errors"
	"testing"
)

func TestNewCodecRejectsRegistrationConflictsAndUnsupportedScopes(t *testing.T) {
	t.Parallel()

	tests := []CodecOptions{
		{Extensions: []ExtensionDefinition{
			{URI: "https://example.com/one", Namespace: "one"},
			{URI: "https://example.com/one", Namespace: "two"},
		}},
		{Extensions: []ExtensionDefinition{
			{URI: "https://example.com/one", Namespace: "same"},
			{URI: "https://example.com/two", Namespace: "same"},
		}},
		{Extensions: []ExtensionDefinition{{
			URI:       "https://example.com/ext",
			Namespace: "ext",
			Members: []MemberDefinition{{
				Scope: 0,
				Name:  "ext:value",
			}},
		}}},
		{Extensions: []ExtensionDefinition{{
			URI:       "https://example.com/ext",
			Namespace: "ext",
			Members: []MemberDefinition{
				{Scope: ResourceMemberScope, Name: "ext:value"},
				{Scope: ResourceMemberScope, Name: "ext:value"},
			},
		}}},
	}

	for _, options := range tests {
		if _, err := NewCodec(options); err == nil {
			t.Fatalf("expected codec registration failure: %#v", options)
		}
	}
}

func TestCodecRejectsMalformedDocumentsAtEverySanitizationBoundary(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}
	tests := []struct {
		payload string
		path    string
		code    string
	}{
		{payload: `{`, path: "", code: "syntax"},
		{payload: `{"data":null,"data":null}`, path: "/data", code: "duplicate-member"},
		{payload: `[]`, path: "", code: "type"},
		{payload: `{"jsonapi":[]}`, path: "/jsonapi", code: "type"},
		{payload: `{"links":[]}`, path: "/links", code: "type"},
		{payload: `{"data":true}`, path: "/data", code: "type"},
		{payload: `{"included":null}`, path: "/included", code: "type"},
		{payload: `{"included":[null]}`, path: "/included/0", code: "type"},
		{payload: `{"errors":null}`, path: "/errors", code: "type"},
		{payload: `{"errors":[null]}`, path: "/errors/0", code: "type"},
		{payload: `{"errors":[{"links":[]}]}`, path: "/errors/0/links", code: "type"},
		{payload: `{"errors":[{"source":[]}]}`, path: "/errors/0/source", code: "type"},
		{payload: `{"data":{"type":"articles","id":"1","links":[]}}`, path: "/data/links", code: "type"},
		{payload: `{"data":{"type":"articles","id":"1","relationships":{"author":null}}}`, path: "/data/relationships/author", code: "type"},
		{payload: `{"data":{"type":"articles","id":"1","relationships":{"author":{"links":[]}}}}`, path: "/data/relationships/author/links", code: "type"},
		{payload: `{"data":{"type":"articles","id":"1","relationships":{"author":{"data":[null]}}}}`, path: "/data/relationships/author/data/0", code: "type"},
	}

	for _, test := range tests {
		_, err := codec.Unmarshal([]byte(test.payload))
		var decodeError *DecodeError
		if !errors.As(err, &decodeError) || decodeError.Path != test.path ||
			decodeError.Code != test.code {
			t.Errorf(
				"unexpected error for %s: got %T %#v, want path %q code %q",
				test.payload,
				err,
				decodeError,
				test.path,
				test.code,
			)
		}
	}
}

func TestCodecMarshalRejectsInvalidCoreDocument(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(CodecOptions{})
	if err != nil {
		t.Fatalf("construct codec: %v", err)
	}
	if _, err := codec.Marshal(Document{}); err == nil {
		t.Fatal("expected core document validation failure")
	}
}
