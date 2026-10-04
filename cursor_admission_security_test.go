package jsonapi_test

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	jsonapi "github.com/faustbrian/go-jsonapi"
)

func TestCursorAdmissionSecurityRejectsDirectOversizeInputs(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("cursor admission diagnostic runs only in hosted CI")
	}

	limits := jsonapi.DefaultQueryLimits()
	marker := "private-cursor-marker"
	tests := []struct {
		name       string
		query      func() jsonapi.Query
		parseQuery bool
	}{
		{"family count", func() jsonapi.Query {
			family := make(jsonapi.ParameterFamily, limits.MaxParameters+1)
			for i := 0; i <= limits.MaxParameters; i++ {
				family[fmt.Sprintf("page[unknown%d]", i)] = []string{"value"}
			}
			return jsonapi.Query{Page: family}
		}, false},
		{"name bytes", func() jsonapi.Query {
			name := marker + strings.Repeat("x", limits.MaxNameBytes+1-len(marker))
			return jsonapi.Query{Page: jsonapi.ParameterFamily{name: {"value"}}}
		}, false},
		{"cursor bytes", func() jsonapi.Query {
			value := marker + strings.Repeat("x", limits.MaxValueBytes+1-len(marker))
			return jsonapi.Query{Page: jsonapi.ParameterFamily{"page[after]": {value}}}
		}, false},
		{"value count", func() jsonapi.Query {
			return jsonapi.Query{Page: jsonapi.ParameterFamily{"page[after]": make([]string, limits.MaxValues+1)}}
		}, false},
		{"sort count", func() jsonapi.Query {
			return jsonapi.Query{Sort: make([]jsonapi.SortField, limits.MaxListItems+1)}
		}, true},
		{"sort name bytes", func() jsonapi.Query {
			return jsonapi.Query{Sort: []jsonapi.SortField{{Name: strings.Repeat("x", limits.MaxValueBytes+1)}}}
		}, true},
		{"ParseQuery cursor bytes", func() jsonapi.Query {
			return jsonapi.Query{Page: jsonapi.ParameterFamily{"page[before]": {strings.Repeat("x", limits.MaxValueBytes+1)}}}
		}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			pagination, err := jsonapi.NewCursorPagination(jsonapi.CursorPaginationConfig{
				DefaultSize: 1, MaxSize: 10,
				ValidateCursor: func(string) error { calls++; return nil },
				ValidateSort:   func([]jsonapi.SortField) error { calls++; return nil },
			})
			if err != nil {
				t.Fatal(err)
			}
			query := test.query()
			var request jsonapi.CursorPageRequest
			if test.parseQuery {
				request, err = pagination.ParseQuery(query)
			} else {
				request, err = pagination.Parse(query.Page)
			}
			var failure *jsonapi.CursorPaginationError
			if !errors.As(err, &failure) || failure.Status != 400 || failure.Code != "limit" {
				t.Errorf("wanted typed HTTP400 limit refusal; got %T", err)
			}
			if !reflect.DeepEqual(request, jsonapi.CursorPageRequest{}) {
				t.Error("refusal returned a partial request")
			}
			if calls != 0 {
				t.Error("refusal invoked application callback")
			}
			if err != nil && strings.Contains(err.Error(), marker) {
				t.Error("refusal exposed input marker")
			}
		})
	}
}

func TestCursorAdmissionSecurityAcceptsExactBoundaries(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("cursor admission diagnostic runs only in hosted CI")
	}

	limits := jsonapi.DefaultQueryLimits()
	cursor := strings.Repeat("x", limits.MaxValueBytes)
	sort := make([]jsonapi.SortField, limits.MaxListItems)
	for i := range sort {
		sort[i].Name = "id"
	}
	cursorCalls, sortCalls := 0, 0
	pagination, err := jsonapi.NewCursorPagination(jsonapi.CursorPaginationConfig{
		DefaultSize: 1, MaxSize: 10, AllowRange: true,
		ValidateCursor: func(value string) error {
			cursorCalls++
			if value != cursor {
				t.Error("cursor changed")
			}
			return nil
		},
		ValidateSort: func(value []jsonapi.SortField) error {
			sortCalls++
			if !reflect.DeepEqual(value, sort) {
				t.Error("sort changed")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	family := jsonapi.ParameterFamily{"page[size]": {"10"}, "page[after]": {cursor}, "page[before]": {cursor}}
	direct, err := pagination.Parse(family)
	if err != nil {
		t.Fatal(err)
	}
	query, err := pagination.ParseQuery(jsonapi.Query{Page: family, Sort: sort})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(direct, query) || direct.Size != 10 || !direct.Range || direct.After != cursor || direct.Before != cursor {
		t.Error("exact-boundary request was not preserved")
	}
	if cursorCalls != 4 || sortCalls != 1 {
		t.Error("accepted request did not invoke expected validators")
	}
}

func TestCursorAdmissionSecurityRejectsUnboundedConfiguration(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("cursor admission diagnostic runs only in hosted CI")
	}

	for _, maximum := range []int{0, -1} {
		pagination, err := jsonapi.NewCursorPagination(jsonapi.CursorPaginationConfig{DefaultSize: 1, MaxSize: maximum})
		if err == nil || pagination != nil {
			t.Error("unbounded configuration became request-serving")
		}
	}
}
