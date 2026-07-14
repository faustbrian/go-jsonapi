package jsonapi

import (
	"errors"
	"net/url"
	"reflect"
	"testing"
)

func TestParseQueryParameters(t *testing.T) {
	t.Parallel()

	values, err := url.ParseQuery(
		"include=comments.author,ratings&" +
			"fields%5Barticles%5D=title,body&fields%5Bpeople%5D=&" +
			"sort=-created,author.name&" +
			"page%5Boffset%5D=10&page%5Blimit%5D=5&" +
			"filter%5Bauthor.status%5D=active&filter%5Btags%5D%5B%5D=go",
	)
	if err != nil {
		t.Fatalf("parse URL query: %v", err)
	}

	query, err := ParseQuery(values)
	if err != nil {
		t.Fatalf("parse JSON:API query: %v", err)
	}
	if !query.IncludePresent || !reflect.DeepEqual(query.Include, []string{"comments.author", "ratings"}) {
		t.Fatalf("unexpected include: %#v", query.Include)
	}
	if !reflect.DeepEqual(query.Fields, map[string][]string{
		"articles": {"title", "body"},
		"people":   {},
	}) {
		t.Fatalf("unexpected fields: %#v", query.Fields)
	}
	if !query.SortPresent || !reflect.DeepEqual(query.Sort, []SortField{
		{Name: "created", Descending: true},
		{Name: "author.name"},
	}) {
		t.Fatalf("unexpected sort: %#v", query.Sort)
	}
	if !reflect.DeepEqual(query.Page, ParameterFamily{
		"page[offset]": {"10"},
		"page[limit]":  {"5"},
	}) {
		t.Fatalf("unexpected page family: %#v", query.Page)
	}
	if !reflect.DeepEqual(query.Filter, ParameterFamily{
		"filter[author.status]": {"active"},
		"filter[tags][]":        {"go"},
	}) {
		t.Fatalf("unexpected filter family: %#v", query.Filter)
	}
}

func TestParseQueryPreservesExplicitEmptyIncludeAndSort(t *testing.T) {
	t.Parallel()

	query, err := ParseQuery(url.Values{"include": {""}})
	if err != nil {
		t.Fatalf("parse empty include: %v", err)
	}
	if !query.IncludePresent || query.Include == nil || len(query.Include) != 0 {
		t.Fatalf("explicit empty include was not preserved: %#v", query)
	}

	_, err = ParseQuery(url.Values{"sort": {""}})
	assertQueryError(t, err, "sort", "invalid-value")
}

func TestQueryParserAcceptsRegisteredCustomAndExtensionFamilies(t *testing.T) {
	t.Parallel()

	parser, err := NewQueryParser(
		[]string{"customFlag"},
		[]string{"atomic"},
	)
	if err != nil {
		t.Fatalf("create query parser: %v", err)
	}
	query, err := parser.Parse(url.Values{
		"customFlag[mode]": {"strict"},
		"atomic:mode":      {"ordered"},
	})
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if !reflect.DeepEqual(query.Custom["customFlag"], ParameterFamily{
		"customFlag[mode]": {"strict"},
	}) {
		t.Fatalf("unexpected custom family: %#v", query.Custom)
	}
	if !reflect.DeepEqual(query.Extensions["atomic"], ParameterFamily{
		"atomic:mode": {"ordered"},
	}) {
		t.Fatalf("unexpected extension family: %#v", query.Extensions)
	}
}

func TestParseQueryRejectsMalformedOrUnknownParameters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		values    url.Values
		parameter string
		code      string
	}{
		"unknown reserved name": {
			values:    url.Values{"future": {"value"}},
			parameter: "future",
			code:      "unknown-parameter",
		},
		"unregistered custom family": {
			values:    url.Values{"customFlag": {"true"}},
			parameter: "customFlag",
			code:      "unknown-parameter",
		},
		"invalid family selector": {
			values:    url.Values{"filter[_]": {"value"}},
			parameter: "filter[_]",
			code:      "invalid-name",
		},
		"malformed brackets": {
			values:    url.Values{"page[size": {"10"}},
			parameter: "page[size",
			code:      "invalid-name",
		},
		"empty include path component": {
			values:    url.Values{"include": {"comments..author"}},
			parameter: "include",
			code:      "invalid-value",
		},
		"invalid fieldset type": {
			values:    url.Values{"fields[bad/type]": {"title"}},
			parameter: "fields[bad/type]",
			code:      "invalid-name",
		},
		"repeated singular parameter": {
			values:    url.Values{"sort": {"title", "created"}},
			parameter: "sort",
			code:      "multiple-values",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseQuery(test.values)
			assertQueryError(t, err, test.parameter, test.code)
		})
	}
}

func TestNewQueryParserRejectsInvalidRegistration(t *testing.T) {
	t.Parallel()

	_, err := NewQueryParser([]string{"lowercase"}, nil)
	if err == nil {
		t.Fatal("expected invalid custom family error")
	}
	_, err = NewQueryParser(nil, []string{"bad-namespace"})
	if err == nil {
		t.Fatal("expected invalid extension namespace error")
	}
}

func assertQueryError(t *testing.T, err error, parameter, code string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected query error")
	}
	var queryError *QueryError
	if !errors.As(err, &queryError) {
		t.Fatalf("expected QueryError, got %T: %v", err, err)
	}
	if queryError.Status != 400 || queryError.Parameter != parameter || queryError.Code != code {
		t.Fatalf(
			"unexpected error: got status %d parameter %q code %q",
			queryError.Status,
			queryError.Parameter,
			queryError.Code,
		)
	}
}
