package jsonapi_test

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	jsonapi "github.com/faustbrian/go-jsonapi/v2"
)

func TestCursorBudgetsPreserveAdmissionBoundarySemanticsHosted(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("cursor budget boundaries run only in hosted CI")
	}
	limits := jsonapi.DefaultQueryLimits()
	// Repeated values are intentionally invalid profile input: admission must
	// finish first, then preserve the profile's multiple-values classification.
	pageBytes := func(total int) jsonapi.ParameterFamily {
		remaining := total - len("page[after]")
		var values []string
		for remaining > 0 {
			size := min(remaining, limits.MaxValueBytes)
			values = append(values, strings.Repeat("x", size))
			remaining -= size
		}
		return jsonapi.ParameterFamily{"page[after]": values}
	}
	nameBytes := func(total int) jsonapi.ParameterFamily {
		family := make(jsonapi.ParameterFamily, limits.MaxParameters)
		for index := 0; index < limits.MaxParameters; index++ {
			size := total / (limits.MaxParameters - index)
			prefix := fmt.Sprintf("page[%03d]", index)
			family[prefix+strings.Repeat("x", size-len(prefix))] = nil
			total -= size
		}
		return family
	}
	parameterLimit := make(jsonapi.ParameterFamily, limits.MaxParameters)
	for index := 0; index < limits.MaxParameters; index++ {
		parameterLimit[fmt.Sprintf("page[%03d]", index)] = nil
	}
	tests := []struct {
		name  string
		query jsonapi.Query
		code  string
	}{
		{"parameter count", jsonapi.Query{Page: parameterLimit}, "unknown-parameter"},
		{"name length", jsonapi.Query{Page: jsonapi.ParameterFamily{strings.Repeat("x", limits.MaxNameBytes): nil}}, "unknown-parameter"},
		{"names exhaust aggregate", jsonapi.Query{Page: nameBytes(limits.MaxTotalBytes)}, "unknown-parameter"},
		{"value count", jsonapi.Query{Page: jsonapi.ParameterFamily{"page[after]": make([]string, limits.MaxValues)}}, "multiple-values"},
		{"values exhaust aggregate", jsonapi.Query{Page: pageBytes(limits.MaxTotalBytes)}, "multiple-values"},
		{"sort reservation exhausts aggregate", jsonapi.Query{Page: pageBytes(limits.MaxTotalBytes - len("sort")), Sort: []jsonapi.SortField{{}}}, "multiple-values"},
		{"sort name exhausts aggregate", jsonapi.Query{Page: pageBytes(limits.MaxTotalBytes - len("sort") - limits.MaxValueBytes), Sort: []jsonapi.SortField{{Name: strings.Repeat("x", limits.MaxValueBytes)}}}, "multiple-values"},
		{"comma exhausts aggregate", jsonapi.Query{Page: pageBytes(limits.MaxTotalBytes - len("sort") - limits.MaxValueBytes), Sort: []jsonapi.SortField{{Name: strings.Repeat("x", limits.MaxValueBytes-1)}, {}}}, "multiple-values"},
		{"sort reservation exceeds aggregate", jsonapi.Query{Page: pageBytes(limits.MaxTotalBytes - len("sort") - limits.MaxValueBytes + 1), Sort: []jsonapi.SortField{{Name: strings.Repeat("x", limits.MaxValueBytes)}}}, "limit"},
		{"accumulated names and comma exceed aggregate", jsonapi.Query{Page: pageBytes(limits.MaxTotalBytes - len("sort") - limits.MaxValueBytes + 1), Sort: []jsonapi.SortField{{Name: strings.Repeat("x", limits.MaxValueBytes/2-1)}, {Name: strings.Repeat("y", limits.MaxValueBytes/2)}}}, "limit"},
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
				t.Fatal("bounded endpoint configuration failed")
			}
			request, err := pagination.ParseQuery(test.query)
			var failure *jsonapi.CursorPaginationError
			if !errors.As(err, &failure) || failure.Status != 400 || failure.Code != test.code {
				t.Fatal("admission changed the expected typed refusal classification")
			}
			if calls != 0 || !reflect.DeepEqual(request, jsonapi.CursorPageRequest{}) {
				t.Error("refusal invoked a callback or returned a partial request")
			}
			if test.code == "limit" && (failure.Parameter != "" || failure.Cause != nil || failure.Message != "cursor pagination input exceeds resource limits") {
				t.Error("resource refusal was not fixed and redacted")
			}
		})
	}
}

func TestCursorEncodedSortExactBudgetsHosted(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("cursor budget boundaries run only in hosted CI")
	}
	maximum := jsonapi.DefaultQueryLimits().MaxValueBytes
	tests := []struct {
		name   string
		fields []jsonapi.SortField
		allow  bool
	}{
		{"ascending exact", []jsonapi.SortField{{Name: strings.Repeat("x", maximum)}}, true},
		{"descending exact", []jsonapi.SortField{{Name: strings.Repeat("x", maximum-1), Descending: true}}, true},
		{"comma exact", []jsonapi.SortField{{Name: strings.Repeat("x", maximum-1)}, {}}, true},
		{"descending over", []jsonapi.SortField{{Name: strings.Repeat("x", maximum), Descending: true}}, false},
		{"comma over", []jsonapi.SortField{{Name: strings.Repeat("x", maximum-1)}, {Name: "y"}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			pagination, err := jsonapi.NewCursorPagination(jsonapi.CursorPaginationConfig{
				DefaultSize: 10, MaxSize: 10,
				ValidateSort: func(fields []jsonapi.SortField) error {
					calls++
					if !reflect.DeepEqual(fields, test.fields) {
						t.Error("admitted sort changed before the callback")
					}
					return nil
				},
			})
			if err != nil {
				t.Fatal("equal finite default and maximum were rejected")
			}
			request, err := pagination.ParseQuery(jsonapi.Query{Sort: test.fields})
			if test.allow {
				if err != nil || calls != 1 || request.Size != 10 || request.SizePresent || request.Range {
					t.Error("exact encoded sort budget was not preserved")
				}
				return
			}
			var failure *jsonapi.CursorPaginationError
			if !errors.As(err, &failure) || failure.Status != 400 || failure.Code != "limit" || failure.Parameter != "" || failure.Cause != nil || failure.Message != "cursor pagination input exceeds resource limits" {
				t.Fatal("oversized encoded sort did not produce a fixed resource refusal")
			}
			if calls != 0 || !reflect.DeepEqual(request, jsonapi.CursorPageRequest{}) {
				t.Error("oversized encoded sort reached a callback or returned a partial request")
			}
		})
	}
}
