package jsonapi

import (
	"errors"
	"testing"
)

func TestAtomicDocumentValidation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		document AtomicDocument
		path     string
		code     string
	}{
		"requires a top-level semantic member": {
			document: AtomicDocument{},
			path:     "",
			code:     "required",
		},
		"operations must not be empty": {
			document: AtomicDocument{Operations: []AtomicOperation{}},
			path:     "/atomic:operations",
			code:     "min-items",
		},
		"results must not be empty": {
			document: AtomicDocument{Results: []AtomicResult{}},
			path:     "/atomic:results",
			code:     "min-items",
		},
		"operations and results conflict": {
			document: AtomicDocument{
				Operations: []AtomicOperation{{
					Op:   AtomicAdd,
					Data: ResourceData(ResourceObject{Type: "articles"}),
				}},
				Results: []AtomicResult{{}},
			},
			path: "/atomic:results",
			code: "conflict",
		},
		"operations and errors conflict": {
			document: AtomicDocument{
				Operations: []AtomicOperation{{
					Op:   AtomicAdd,
					Data: ResourceData(ResourceObject{Type: "articles"}),
				}},
				Errors: []ErrorObject{{Title: "failed"}},
			},
			path: "/errors",
			code: "conflict",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := test.document.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			var validationError *ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
			if got := validationError.Violations[0]; got.Path != test.path || got.Code != test.code {
				t.Fatalf("unexpected violation: %#v", got)
			}
		})
	}
}

func TestAtomicDocumentAcceptsEachTopLevelForm(t *testing.T) {
	t.Parallel()

	documents := []AtomicDocument{
		{Operations: []AtomicOperation{{
			Op:   AtomicAdd,
			Data: ResourceData(ResourceObject{Type: "articles"}),
		}}},
		{Results: []AtomicResult{{}}},
		{Errors: []ErrorObject{{Title: "failed"}}},
		{Meta: Meta{}},
	}

	for _, document := range documents {
		if err := document.Validate(); err != nil {
			t.Fatalf("expected valid atomic document: %v", err)
		}
	}
}

func TestAtomicOperationValidation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		operation AtomicOperation
		path      string
		code      string
	}{
		"requires op": {
			operation: AtomicOperation{},
			path:      "/atomic:operations/0/op",
			code:      "required",
		},
		"rejects unknown op": {
			operation: AtomicOperation{Op: "replace", Data: NullData()},
			path:      "/atomic:operations/0/op",
			code:      "value",
		},
		"rejects ref and href together": {
			operation: AtomicOperation{
				Op:   AtomicRemove,
				Ref:  &AtomicReference{Type: "articles", ID: "1"},
				Href: "/articles/1",
			},
			path: "/atomic:operations/0/href",
			code: "conflict",
		},
		"reference requires type": {
			operation: AtomicOperation{
				Op:  AtomicRemove,
				Ref: &AtomicReference{ID: "1"},
			},
			path: "/atomic:operations/0/ref/type",
			code: "required",
		},
		"reference requires identity": {
			operation: AtomicOperation{
				Op:  AtomicRemove,
				Ref: &AtomicReference{Type: "articles"},
			},
			path: "/atomic:operations/0/ref/id",
			code: "required",
		},
		"reference identity is exclusive": {
			operation: AtomicOperation{
				Op: AtomicRemove,
				Ref: &AtomicReference{
					Type: "articles",
					ID:   "1",
					LID:  "local-1",
				},
			},
			path: "/atomic:operations/0/ref/lid",
			code: "conflict",
		},
		"relationship name must be valid": {
			operation: AtomicOperation{
				Op: AtomicUpdate,
				Ref: &AtomicReference{
					Type:         "articles",
					ID:           "1",
					Relationship: "-bad",
				},
				Data: NullData(),
			},
			path: "/atomic:operations/0/ref/relationship",
			code: "member-name",
		},
		"href must be a URI reference": {
			operation: AtomicOperation{
				Op:   AtomicRemove,
				Href: ":not-a-reference",
			},
			path: "/atomic:operations/0/href",
			code: "url",
		},
		"add requires data": {
			operation: AtomicOperation{Op: AtomicAdd},
			path:      "/atomic:operations/0/data",
			code:      "required",
		},
		"add resource must not use ref": {
			operation: AtomicOperation{
				Op:   AtomicAdd,
				Ref:  &AtomicReference{Type: "articles", ID: "1"},
				Data: ResourceData(ResourceObject{Type: "articles"}),
			},
			path: "/atomic:operations/0/ref",
			code: "forbidden",
		},
		"update requires data": {
			operation: AtomicOperation{Op: AtomicUpdate},
			path:      "/atomic:operations/0/data",
			code:      "required",
		},
		"remove resource requires target": {
			operation: AtomicOperation{Op: AtomicRemove},
			path:      "/atomic:operations/0/ref",
			code:      "required",
		},
		"relationship mutation requires data": {
			operation: AtomicOperation{
				Op: AtomicRemove,
				Ref: &AtomicReference{
					Type:         "articles",
					ID:           "1",
					Relationship: "comments",
				},
			},
			path: "/atomic:operations/0/data",
			code: "required",
		},
		"remove resource forbids data": {
			operation: AtomicOperation{
				Op:   AtomicRemove,
				Ref:  &AtomicReference{Type: "articles", ID: "1"},
				Data: ResourceData(ResourceObject{Type: "articles", ID: "1"}),
			},
			path: "/atomic:operations/0/data",
			code: "forbidden",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			document := AtomicDocument{Operations: []AtomicOperation{test.operation}}
			err := document.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			var validationError *ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
			if got := validationError.Violations[0]; got.Path != test.path || got.Code != test.code {
				t.Fatalf("unexpected first violation: %#v", got)
			}
		})
	}
}

func TestAtomicReferenceLIDMustBeAssignedByPriorOperation(t *testing.T) {
	t.Parallel()

	invalid := AtomicDocument{Operations: []AtomicOperation{
		{
			Op:  AtomicRemove,
			Ref: &AtomicReference{Type: "articles", LID: "new-article"},
		},
		{
			Op: AtomicAdd,
			Data: ResourceData(ResourceObject{
				Type: "articles",
				LID:  "new-article",
			}),
		},
	}}
	assertAtomicViolation(t, invalid, "/atomic:operations/0/ref/lid", "unresolved-lid")

	valid := AtomicDocument{Operations: []AtomicOperation{
		{
			Op: AtomicAdd,
			Data: ResourceData(ResourceObject{
				Type: "articles",
				LID:  "new-article",
			}),
		},
		{
			Op:  AtomicUpdate,
			Ref: &AtomicReference{Type: "articles", LID: "new-article"},
			Data: ResourceData(ResourceObject{
				Type: "articles",
				LID:  "new-article",
				Attributes: Attributes{
					"title": "Updated",
				},
			}),
		},
	}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected prior local identity to resolve: %v", err)
	}
}

func assertAtomicViolation(t *testing.T, document AtomicDocument, path, code string) {
	t.Helper()

	err := document.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	for _, violation := range validationError.Violations {
		if violation.Path == path && violation.Code == code {
			return
		}
	}
	t.Fatalf("missing violation at %q with code %q: %#v", path, code, validationError.Violations)
}

func TestAtomicOperationValidationAcceptsNormativeShapes(t *testing.T) {
	t.Parallel()

	operations := []AtomicOperation{
		{
			Op:   AtomicAdd,
			Href: "/articles",
			Data: ResourceData(ResourceObject{Type: "articles"}),
		},
		{
			Op: AtomicUpdate,
			Data: ResourceData(ResourceObject{
				Type:       "articles",
				ID:         "1",
				Attributes: Attributes{"title": "Updated"},
			}),
		},
		{
			Op:  AtomicRemove,
			Ref: &AtomicReference{Type: "articles", ID: "1"},
		},
		{
			Op: AtomicAdd,
			Ref: &AtomicReference{
				Type:         "articles",
				ID:           "1",
				Relationship: "comments",
			},
			Data: ResourceCollection(ResourceObject{Type: "comments", ID: "2"}),
		},
		{
			Op: AtomicUpdate,
			Ref: &AtomicReference{
				Type:         "articles",
				ID:           "1",
				Relationship: "author",
			},
			Data: NullData(),
		},
	}

	if err := (AtomicDocument{Operations: operations}).Validate(); err != nil {
		t.Fatalf("expected valid operations: %v", err)
	}
}
