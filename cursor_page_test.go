package jsonapi

import (
	"errors"
	"testing"
)

func TestValidateCursorPageAcceptsConformantRangePage(t *testing.T) {
	t.Parallel()

	truncated := true
	meta, err := (CursorPageMeta{RangeTruncated: &truncated}).Meta()
	if err != nil {
		t.Fatalf("build page metadata: %v", err)
	}
	page := CursorPage{
		Request: CursorPageRequest{
			Size:          2,
			After:         "start",
			AfterPresent:  true,
			Before:        "end",
			BeforePresent: true,
			Range:         true,
		},
		Links: Links{
			"prev": URI("/articles?page[before]=one"),
			"next": URI("/articles?page[after]=two"),
		},
		Meta:    meta,
		Items:   []Meta{CursorItemMeta("one"), CursorItemMeta("two")},
		HasMore: true,
	}
	if err := page.Validate(); err != nil {
		t.Fatalf("expected conformant page: %v", err)
	}
}

func TestValidateCursorPageRejectsResultContractViolations(t *testing.T) {
	t.Parallel()

	base := func() CursorPage {
		return CursorPage{
			Request: CursorPageRequest{Size: 2},
			Links: Links{
				"prev": NullLink(),
				"next": NullLink(),
			},
			Items: []Meta{nil, nil},
		}
	}
	tests := map[string]struct {
		mutate func(*CursorPage)
		path   string
		code   string
	}{
		"used page size must be positive": {
			mutate: func(page *CursorPage) {
				page.Request.Size = 0
				page.Items = nil
			},
			path: "/data",
			code: "page-size",
		},
		"page exceeds used size": {
			mutate: func(page *CursorPage) { page.Items = append(page.Items, nil) },
			path:   "/data",
			code:   "page-size",
		},
		"short page claims more items": {
			mutate: func(page *CursorPage) {
				page.Items = page.Items[:1]
				page.HasMore = true
			},
			path: "/data",
			code: "page-size",
		},
		"truncated range requires metadata": {
			mutate: func(page *CursorPage) {
				page.Request.Range = true
				page.HasMore = true
			},
			path: "/meta/page/rangeTruncated",
			code: "required",
		},
		"ordinary page forbids range truncation": {
			mutate: func(page *CursorPage) {
				value := true
				page.Meta, _ = (CursorPageMeta{RangeTruncated: &value}).Meta()
			},
			path: "/meta/page/rangeTruncated",
			code: "forbidden",
		},
		"item cursor must be a string": {
			mutate: func(page *CursorPage) {
				page.Items[1] = Meta{"page": map[string]any{"cursor": 42}}
			},
			path: "/data/1/meta/page/cursor",
			code: "type",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			page := base()
			test.mutate(&page)
			err := page.Validate()
			var validationError *ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
			if !hasViolation(validationError, test.path, test.code) {
				t.Fatalf("missing %s at %s: %#v", test.code, test.path, validationError.Violations)
			}
		})
	}
}

func TestValidateCursorPageAtUsesNestedRelationshipPaths(t *testing.T) {
	t.Parallel()

	page := CursorPage{
		Request: CursorPageRequest{Size: 1},
		Links:   Links{"next": NullLink()},
		Items:   []Meta{nil},
	}
	err := page.ValidateAt("/data/relationships/comments")
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !hasViolation(validationError, "/data/relationships/comments/links/prev", "required") {
		t.Fatalf("unexpected nested violations: %#v", validationError.Violations)
	}
}
