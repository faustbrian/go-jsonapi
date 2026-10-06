package jsonapi_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	jsonapi "github.com/faustbrian/go-jsonapi/v2"
)

func TestLinkScopeValidationReportsEveryForbiddenMemberHosted(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("link scope characterization runs only in hosted CI")
	}
	links := func(allowed string) jsonapi.Links {
		result := jsonapi.Links{allowed: jsonapi.URI("/allowed")}
		for index := 0; index < 8; index++ {
			result[fmt.Sprintf("wrong%d", index)] = jsonapi.URI("/unrelated")
		}
		return result
	}
	tests := []struct {
		name     string
		document jsonapi.Document
		path     string
		allowed  string
	}{
		{"resource", jsonapi.Document{Data: jsonapi.ResourceData(jsonapi.ResourceObject{
			Type: "articles", ID: "1", Links: links("self"),
		})}, "/data/links", "self"},
		{"relationship", jsonapi.Document{Data: jsonapi.ResourceData(jsonapi.ResourceObject{
			Type: "articles", ID: "1", Relationships: jsonapi.Relationships{
				"author": {Links: links("self")},
			},
		})}, "/data/relationships/author/links", "self"},
		{"error", jsonapi.Document{Errors: []jsonapi.ErrorObject{{Links: links("about")}}}, "/errors/0/links", "about"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var failure *jsonapi.ValidationError
			if !errors.As(test.document.Validate(), &failure) {
				t.Fatal("mixed link scopes did not produce typed validation failure")
			}
			found := make(map[string]int)
			for _, violation := range failure.Violations {
				if violation.Code == "link-scope" {
					found[violation.Path]++
				}
			}
			for index := 0; index < 8; index++ {
				if found[test.path+fmt.Sprintf("/wrong%d", index)] != 1 {
					t.Error("scope validation omitted or duplicated a forbidden member")
				}
			}
			if len(found) != 8 || found[test.path+"/"+test.allowed] != 0 {
				t.Error("scope validation did not conserve allowed and forbidden members")
			}
		})
	}
}
