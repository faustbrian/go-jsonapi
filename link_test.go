package jsonapi

import (
	"errors"
	"testing"
)

func TestLinkObjectSupportsEveryJSONAPI11Member(t *testing.T) {
	t.Parallel()

	describedBy := URI("https://example.com/schemas/comments")
	document := Document{
		Data: NullData(),
		Links: Links{
			"related": LinkFromObject(LinkObject{
				Href:        "https://example.com/articles/1/comments",
				Rel:         "related",
				DescribedBy: &describedBy,
				Title:       "Comments",
				Type:        "application/vnd.api+json",
				Hreflang:    LanguageTags("en", "fi"),
				Meta:        Meta{"count": 10},
			}),
		},
	}

	got, err := Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}
	want := `{"links":{"related":{"href":"https://example.com/articles/1/comments","rel":"related","describedby":"https://example.com/schemas/comments","title":"Comments","type":"application/vnd.api+json","hreflang":["en","fi"],"meta":{"count":10}}},"data":null}`
	if string(got) != want {
		t.Fatalf("unexpected JSON:\n got: %s\nwant: %s", got, want)
	}
}

func TestLinkObjectRoundTripPreservesHreflangShape(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"scalar": `{"links":{"self":{"href":"/articles","hreflang":"en"}},"data":null}`,
		"array":  `{"links":{"self":{"href":"/articles","hreflang":["en","fi"]}},"data":null}`,
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

func TestLinkObjectRoundTripSupportsNestedDescribedByObject(t *testing.T) {
	t.Parallel()

	payload := `{"links":{"self":{"href":"/articles","describedby":{"href":"/schema","type":"application/schema+json"}}},"data":null}`
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
}

func TestValidateRejectsInvalidLinkObjectMembers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		object LinkObject
		path   string
		code   string
	}{
		"missing href": {
			object: LinkObject{},
			path:   "/links/self/href",
			code:   "required",
		},
		"invalid relation": {
			object: LinkObject{Href: "/articles", Rel: "bad relation"},
			path:   "/links/self/rel",
			code:   "link-relation",
		},
		"invalid describedby": {
			object: LinkObject{Href: "/articles", DescribedBy: linkPointer(URI(":bad"))},
			path:   "/links/self/describedby",
			code:   "url",
		},
		"invalid media type": {
			object: LinkObject{Href: "/articles", Type: "not a media type"},
			path:   "/links/self/type",
			code:   "media-type",
		},
		"invalid language tag": {
			object: LinkObject{Href: "/articles", Hreflang: LanguageTag("not_a_tag")},
			path:   "/links/self/hreflang",
			code:   "language-tag",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := (Document{
				Data:  NullData(),
				Links: Links{"self": LinkFromObject(test.object)},
			}).Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			var validationError *ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
			if !hasViolation(validationError, test.path, test.code) {
				t.Fatalf(
					"missing violation path %q code %q in %#v",
					test.path,
					test.code,
					validationError.Violations,
				)
			}
		})
	}
}

func linkPointer(link Link) *Link {
	return &link
}
