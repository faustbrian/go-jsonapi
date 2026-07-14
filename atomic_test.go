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
