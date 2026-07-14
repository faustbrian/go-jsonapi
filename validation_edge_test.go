package jsonapi

import (
	"errors"
	"testing"
)

func TestValidateRejectsEdgeDocumentShapes(t *testing.T) {
	t.Parallel()

	validResource := ResourceObject{Type: "articles", ID: "1"}
	tests := []struct {
		document Document
		path     string
		code     string
	}{
		{
			document: Document{Data: &PrimaryData{kind: primaryDataOne}},
			path:     "/data", code: "required",
		},
		{
			document: Document{Data: &PrimaryData{kind: primaryDataKind(99)}},
			path:     "/data", code: "shape",
		},
		{
			document: Document{Data: ResourceData(ResourceObject{
				Type: "articles", ID: "1",
				Attributes: Attributes{"bad/name": true},
			})},
			path: "/data/attributes/bad~1name", code: "member-name",
		},
		{
			document: Document{Data: ResourceData(ResourceObject{
				Type: "articles", ID: "1",
				Relationships: Relationships{"id": {Meta: Meta{}}},
			})},
			path: "/data/relationships/id", code: "reserved-field",
		},
		{
			document: Document{Data: ResourceData(ResourceObject{
				Type: "articles", ID: "1",
				Relationships: Relationships{"bad/name": {Meta: Meta{}}},
			})},
			path: "/data/relationships/bad~1name", code: "member-name",
		},
		{
			document: Document{Data: ResourceData(ResourceObject{
				Type: "articles", ID: "1",
				Relationships: Relationships{"author": {
					Data: &RelationshipData{kind: relationshipDataOne},
				}},
			})},
			path: "/data/relationships/author/data", code: "required",
		},
		{
			document: Document{Data: ResourceData(ResourceObject{
				Type: "articles", ID: "1",
				Relationships: Relationships{"author": {
					Data: &RelationshipData{kind: relationshipDataKind(99)},
				}},
			})},
			path: "/data/relationships/author/data", code: "shape",
		},
		{
			document: Document{Data: ResourceData(ResourceObject{
				Type: "articles", ID: "1",
				Relationships: Relationships{"author": {
					Data: ToOne(Identifier{ID: "9"}),
				}},
			})},
			path: "/data/relationships/author/data/type", code: "required",
		},
		{
			document: Document{Data: ResourceData(ResourceObject{
				Type: "articles", ID: "1",
				Relationships: Relationships{"author": {
					Data: ToOne(Identifier{Type: "bad/type", ID: "9"}),
				}},
			})},
			path: "/data/relationships/author/data/type", code: "member-name",
		},
		{
			document: Document{
				Data:  ResourceData(validResource),
				Links: Links{"bad/name": URI("/articles/1")},
			},
			path: "/links/bad~1name", code: "member-name",
		},
		{
			document: Document{Errors: []ErrorObject{{
				Source: &ErrorSource{Pointer: "/data/~"},
			}}},
			path: "/errors/0/source/pointer", code: "json-pointer",
		},
	}
	for _, test := range tests {
		err := test.document.Validate()
		var validationError *ValidationError
		if !errors.As(err, &validationError) ||
			!hasViolation(validationError, test.path, test.code) {
			t.Fatalf("missing %s at %s: %T %#v", test.code, test.path, err, validationError)
		}
	}
}

func TestMemberNameAndJSONPointerGrammarBoundaries(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"a", "@context", "two words", "ümlaut"} {
		if !validMemberName(name) {
			t.Fatalf("valid member name rejected: %q", name)
		}
	}
	for _, name := range []string{"", "@", "-start", "end-", "bad/name"} {
		if validMemberName(name) {
			t.Fatalf("invalid member name accepted: %q", name)
		}
	}

	for _, pointer := range []string{"", "/data", "/a~0b/~1"} {
		if !validJSONPointer(pointer) {
			t.Fatalf("valid JSON Pointer rejected: %q", pointer)
		}
	}
	for _, pointer := range []string{"data", "/data/~", "/data/~2"} {
		if validJSONPointer(pointer) {
			t.Fatalf("invalid JSON Pointer accepted: %q", pointer)
		}
	}
}

func TestValidateRelationshipEndpointIdentifiersRejectResourceMembers(t *testing.T) {
	t.Parallel()

	document := Document{Data: ResourceData(ResourceObject{
		Type: "people", ID: "9",
		Relationships: Relationships{"team": {Data: NullRelationship()}},
		Links:         Links{"self": URI("/people/9")},
	})}
	err := document.ValidateWith(ValidationOptions{Context: ToOneRelationshipRequest})
	var validationError *ValidationError
	if !errors.As(err, &validationError) ||
		!hasViolation(validationError, "/data/relationships", "forbidden") ||
		!hasViolation(validationError, "/data/links", "forbidden") {
		t.Fatalf("missing identifier-only violations: %#v", validationError)
	}

	missing := Document{Meta: Meta{"request": true}}
	if err := missing.ValidateWith(ValidationOptions{
		Context: ToOneRelationshipRequest,
	}); err == nil {
		t.Fatal("relationship request without data was accepted")
	}

	localOnly := Document{Data: ResourceData(ResourceObject{
		Type: "people", LID: "local-person",
	})}
	err = localOnly.ValidateWith(ValidationOptions{Context: ToOneRelationshipRequest})
	if !errors.As(err, &validationError) ||
		!hasViolation(validationError, "/data/id", "required") {
		t.Fatalf("relationship identifier without server ID was accepted: %#v", validationError)
	}
}

func TestValidateDetectsLocalIDReuseAcrossResourceIDs(t *testing.T) {
	t.Parallel()

	document := Document{Data: ResourceCollection(
		ResourceObject{Type: "articles", ID: "1", LID: "same"},
		ResourceObject{Type: "articles", ID: "2", LID: "same"},
	)}
	err := document.Validate()
	var validationError *ValidationError
	if !errors.As(err, &validationError) ||
		!hasViolation(validationError, "/data/1/lid", "local-identity") {
		t.Fatalf("missing local-ID reuse violation: %#v", validationError)
	}
}

func TestValidateAcceptsToManyCompoundLinkage(t *testing.T) {
	t.Parallel()

	document := Document{
		Data: ResourceData(ResourceObject{
			Type: "articles", ID: "1",
			Relationships: Relationships{"tags": {Data: ToMany(
				Identifier{Type: "tags", ID: "2"},
				Identifier{Type: "tags", ID: "3"},
			)}},
		}),
		Included: []ResourceObject{
			{Type: "tags", ID: "2"},
			{Type: "tags", ID: "3"},
		},
	}
	if err := document.Validate(); err != nil {
		t.Fatalf("valid to-many compound linkage rejected: %v", err)
	}
}

func TestValidateIdentityTraversalIgnoresRelationshipsWithoutLinkage(t *testing.T) {
	t.Parallel()

	document := Document{Data: ResourceData(ResourceObject{
		Type: "articles", ID: "1",
		Relationships: Relationships{
			"author": {Data: ToOne(Identifier{Type: "people", ID: "9"})},
			"editor": {Data: NullRelationship()},
		},
	}), Included: []ResourceObject{{Type: "people", ID: "9"}}}
	if err := document.Validate(); err != nil {
		t.Fatalf("metadata-only relationship rejected: %v", err)
	}
}

func TestLanguageTagFastFailureForms(t *testing.T) {
	t.Parallel()

	for _, tag := range []string{"", "-en", "en-", "en--US"} {
		if validLanguageTag(tag) {
			t.Fatalf("invalid language tag accepted: %q", tag)
		}
	}
}
