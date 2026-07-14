package jsonapi

import (
	"errors"
	"testing"
)

func TestCursorPaginationParsesProfileParameters(t *testing.T) {
	t.Parallel()

	pagination, err := NewCursorPagination(CursorPaginationConfig{
		DefaultSize: 25,
		MaxSize:     100,
		AllowRange:  true,
		ValidateCursor: func(cursor string) error {
			if cursor == "bad" {
				return errors.New("invalid cursor")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("construct pagination parser: %v", err)
	}

	page, err := pagination.Parse(ParameterFamily{
		"page[size]":   {"10"},
		"page[after]":  {"abc"},
		"page[before]": {"xyz"},
	})
	if err != nil {
		t.Fatalf("parse cursor pagination: %v", err)
	}
	if page.Size != 10 || !page.SizePresent || page.After != "abc" || page.Before != "xyz" || !page.Range {
		t.Fatalf("unexpected page request: %#v", page)
	}
}

func TestCursorPaginationUsesContextualDefaultSize(t *testing.T) {
	t.Parallel()

	pagination, err := NewCursorPagination(CursorPaginationConfig{
		DefaultSize: 25,
		MaxSize:     100,
		AllowRange:  true,
	})
	if err != nil {
		t.Fatalf("construct pagination parser: %v", err)
	}

	ordinary, err := pagination.Parse(nil)
	if err != nil {
		t.Fatalf("parse ordinary page: %v", err)
	}
	if ordinary.Size != 25 {
		t.Fatalf("unexpected default size: %d", ordinary.Size)
	}

	rangePage, err := pagination.Parse(ParameterFamily{
		"page[after]":  {"abc"},
		"page[before]": {"xyz"},
	})
	if err != nil {
		t.Fatalf("parse range page: %v", err)
	}
	if rangePage.Size != 100 {
		t.Fatalf("range default must use max size, got %d", rangePage.Size)
	}
}

func TestCursorPaginationRejectsInvalidParameters(t *testing.T) {
	t.Parallel()

	pagination, err := NewCursorPagination(CursorPaginationConfig{
		DefaultSize: 10,
		MaxSize:     50,
		ValidateCursor: func(cursor string) error {
			if cursor == "invalid" {
				return errors.New("signature mismatch")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("construct pagination parser: %v", err)
	}

	tests := map[string]struct {
		family    ParameterFamily
		parameter string
		code      string
	}{
		"size is positive decimal": {
			family:    ParameterFamily{"page[size]": {"0"}},
			parameter: "page[size]",
			code:      "invalid-parameter",
		},
		"size occurs once": {
			family:    ParameterFamily{"page[size]": {"1", "2"}},
			parameter: "page[size]",
			code:      "multiple-values",
		},
		"size respects maximum": {
			family:    ParameterFamily{"page[size]": {"51"}},
			parameter: "page[size]",
			code:      "max-size-exceeded",
		},
		"cursor is validated": {
			family:    ParameterFamily{"page[after]": {"invalid"}},
			parameter: "page[after]",
			code:      "invalid-parameter",
		},
		"range must be supported": {
			family: ParameterFamily{
				"page[after]":  {"abc"},
				"page[before]": {"xyz"},
			},
			parameter: "page[before]",
			code:      "range-not-supported",
		},
		"unknown page member": {
			family:    ParameterFamily{"page[number]": {"2"}},
			parameter: "page[number]",
			code:      "unknown-parameter",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := pagination.Parse(test.family)
			if err == nil {
				t.Fatal("expected cursor pagination error")
			}
			var pageError *CursorPaginationError
			if !errors.As(err, &pageError) {
				t.Fatalf("expected CursorPaginationError, got %T: %v", err, err)
			}
			if pageError.Parameter != test.parameter || pageError.Code != test.code || pageError.Status != 400 {
				t.Fatalf("unexpected error: %#v", pageError)
			}
		})
	}
}

func TestCursorPaginationRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	invalid := []CursorPaginationConfig{
		{DefaultSize: 0, MaxSize: 10},
		{DefaultSize: 11, MaxSize: 10},
		{DefaultSize: 10, AllowRange: true},
	}
	for _, config := range invalid {
		if _, err := NewCursorPagination(config); err == nil {
			t.Fatalf("expected invalid config error: %#v", config)
		}
	}
}

func TestCursorPaginationPreservesEmptyCursorPresence(t *testing.T) {
	t.Parallel()

	pagination, err := NewCursorPagination(CursorPaginationConfig{
		DefaultSize: 10,
		MaxSize:     20,
		AllowRange:  true,
		ValidateCursor: func(string) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("construct pagination parser: %v", err)
	}

	page, err := pagination.Parse(ParameterFamily{
		"page[after]":  {""},
		"page[before]": {""},
	})
	if err != nil {
		t.Fatalf("parse empty opaque cursors: %v", err)
	}
	if !page.AfterPresent || !page.BeforePresent || !page.Range {
		t.Fatalf("cursor presence was lost: %#v", page)
	}
}
