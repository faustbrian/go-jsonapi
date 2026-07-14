package jsonapi

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// ParameterFamily preserves the decoded names and values of a JSON:API query
// parameter family for application-defined processing.
type ParameterFamily map[string][]string

// SortField is one ordered sorting criterion.
type SortField struct {
	Name       string
	Descending bool
}

// Query contains parsed core query parameters and raw extension points for
// pagination, filtering, registered custom families, and extensions.
type Query struct {
	Include        []string
	IncludePresent bool
	Fields         map[string][]string
	Sort           []SortField
	SortPresent    bool
	Page           ParameterFamily
	Filter         ParameterFamily
	Custom         map[string]ParameterFamily
	Extensions     map[string]ParameterFamily
}

// QueryError identifies an invalid query parameter and the HTTP status required
// by JSON:API.
type QueryError struct {
	Status    int
	Parameter string
	Code      string
	Message   string
}

// Error implements error.
func (err *QueryError) Error() string {
	return fmt.Sprintf("invalid JSON:API query parameter %q: %s", err.Parameter, err.Message)
}

// QueryParser recognizes core JSON:API parameters plus explicitly registered
// implementation and extension parameter families.
type QueryParser struct {
	customFamilies      map[string]struct{}
	extensionNamespaces map[string]struct{}
}

// NewQueryParser constructs a parser with the custom family base names and
// extension namespaces understood by an application.
func NewQueryParser(customFamilies, extensionNamespaces []string) (*QueryParser, error) {
	parser := &QueryParser{
		customFamilies:      make(map[string]struct{}, len(customFamilies)),
		extensionNamespaces: make(map[string]struct{}, len(extensionNamespaces)),
	}
	for _, family := range customFamilies {
		if !validMemberName(family) || onlyLowercaseASCII(family) {
			return nil, fmt.Errorf("invalid implementation-specific query family: %q", family)
		}
		if _, exists := parser.customFamilies[family]; exists {
			return nil, fmt.Errorf("duplicate implementation-specific query family: %q", family)
		}
		parser.customFamilies[family] = struct{}{}
	}
	for _, namespace := range extensionNamespaces {
		if !validExtensionNamespace(namespace) {
			return nil, fmt.Errorf("invalid extension query namespace: %q", namespace)
		}
		if _, exists := parser.extensionNamespaces[namespace]; exists {
			return nil, fmt.Errorf("duplicate extension query namespace: %q", namespace)
		}
		parser.extensionNamespaces[namespace] = struct{}{}
	}

	return parser, nil
}

// ParseQuery parses core JSON:API query parameters without any custom or
// extension families.
func ParseQuery(values url.Values) (Query, error) {
	return (&QueryParser{}).Parse(values)
}

// Parse validates and classifies decoded URL query values.
func (parser *QueryParser) Parse(values url.Values) (Query, error) {
	var query Query
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		parameterValues := values[name]
		base, selectors, valid := parseFamilyName(name)
		if !valid {
			return Query{}, queryFailure(name, "invalid-name", "parameter family name is malformed")
		}

		switch base {
		case "include":
			if len(selectors) != 0 {
				return Query{}, queryFailure(name, "invalid-name", "include is not a parameter family")
			}
			if len(parameterValues) != 1 {
				return Query{}, queryFailure(name, "multiple-values", "include must occur once")
			}
			include, err := parseInclude(parameterValues[0])
			if err != nil {
				return Query{}, queryFailure(name, "invalid-value", err.Error())
			}
			query.IncludePresent = true
			query.Include = include
		case "fields":
			if len(selectors) != 1 || selectors[0] == "" || strings.Contains(selectors[0], ".") {
				return Query{}, queryFailure(name, "invalid-name", "fields parameter must identify one resource type")
			}
			if len(parameterValues) != 1 {
				return Query{}, queryFailure(name, "multiple-values", "fieldset must occur once per resource type")
			}
			fields, err := parseFields(parameterValues[0])
			if err != nil {
				return Query{}, queryFailure(name, "invalid-value", err.Error())
			}
			if query.Fields == nil {
				query.Fields = make(map[string][]string)
			}
			query.Fields[selectors[0]] = fields
		case "sort":
			if len(selectors) != 0 {
				return Query{}, queryFailure(name, "invalid-name", "sort is not a parameter family")
			}
			if len(parameterValues) != 1 {
				return Query{}, queryFailure(name, "multiple-values", "sort must occur once")
			}
			fields, err := parseSort(parameterValues[0])
			if err != nil {
				return Query{}, queryFailure(name, "invalid-value", err.Error())
			}
			query.SortPresent = true
			query.Sort = fields
		case "page":
			query.Page = addFamilyValue(query.Page, name, parameterValues)
		case "filter":
			query.Filter = addFamilyValue(query.Filter, name, parameterValues)
		default:
			if namespace, extension := extensionQueryBase(base); extension {
				if _, supported := parser.extensionNamespaces[namespace]; !supported {
					return Query{}, queryFailure(name, "unknown-parameter", "extension query namespace is not registered")
				}
				if query.Extensions == nil {
					query.Extensions = make(map[string]ParameterFamily)
				}
				query.Extensions[namespace] = addFamilyValue(query.Extensions[namespace], name, parameterValues)
				continue
			}
			if _, supported := parser.customFamilies[base]; !supported {
				return Query{}, queryFailure(name, "unknown-parameter", "query parameter family is not registered")
			}
			if query.Custom == nil {
				query.Custom = make(map[string]ParameterFamily)
			}
			query.Custom[base] = addFamilyValue(query.Custom[base], name, parameterValues)
		}
	}

	return query, nil
}

func parseFamilyName(name string) (string, []string, bool) {
	bracket := strings.IndexByte(name, '[')
	if bracket < 0 {
		if name == "" {
			return "", nil, false
		}
		return name, nil, true
	}
	base := name[:bracket]
	if base == "" {
		return "", nil, false
	}
	remainder := name[bracket:]
	var selectors []string
	for remainder != "" {
		if remainder[0] != '[' {
			return "", nil, false
		}
		close := strings.IndexByte(remainder, ']')
		if close < 0 {
			return "", nil, false
		}
		selector := remainder[1:close]
		if selector != "" && !validMemberPath(selector) {
			return "", nil, false
		}
		selectors = append(selectors, selector)
		remainder = remainder[close+1:]
	}

	return base, selectors, true
}

func parseInclude(value string) ([]string, error) {
	if value == "" {
		return []string{}, nil
	}
	paths := strings.Split(value, ",")
	for _, path := range paths {
		if !validMemberPath(path) {
			return nil, fmt.Errorf("include must contain comma-separated relationship paths")
		}
	}

	return paths, nil
}

func parseFields(value string) ([]string, error) {
	if value == "" {
		return []string{}, nil
	}
	fields := strings.Split(value, ",")
	for _, field := range fields {
		if !validMemberName(field) {
			return nil, fmt.Errorf("fieldset must contain comma-separated field names")
		}
	}

	return fields, nil
}

func parseSort(value string) ([]SortField, error) {
	if value == "" {
		return nil, fmt.Errorf("sort must contain at least one sort field")
	}
	items := strings.Split(value, ",")
	result := make([]SortField, len(items))
	for index, item := range items {
		descending := strings.HasPrefix(item, "-")
		name := strings.TrimPrefix(item, "-")
		if !validMemberPath(name) {
			return nil, fmt.Errorf("sort must contain valid sort fields")
		}
		result[index] = SortField{Name: name, Descending: descending}
	}

	return result, nil
}

func validMemberPath(path string) bool {
	if path == "" {
		return false
	}
	for _, member := range strings.Split(path, ".") {
		if !validMemberName(member) {
			return false
		}
	}

	return true
}

func addFamilyValue(family ParameterFamily, name string, values []string) ParameterFamily {
	if family == nil {
		family = make(ParameterFamily)
	}
	copyOfValues := make([]string, len(values))
	copy(copyOfValues, values)
	family[name] = copyOfValues

	return family
}

func extensionQueryBase(base string) (string, bool) {
	colon := strings.IndexByte(base, ':')
	if colon <= 0 || colon == len(base)-1 {
		return "", false
	}
	namespace := base[:colon]
	name := base[colon+1:]
	if !validExtensionNamespace(namespace) || !onlyLowercaseASCII(name) {
		return "", false
	}

	return namespace, true
}

func validExtensionNamespace(namespace string) bool {
	if namespace == "" {
		return false
	}
	for _, character := range namespace {
		if !(character >= 'a' && character <= 'z') &&
			!(character >= 'A' && character <= 'Z') &&
			!(character >= '0' && character <= '9') {
			return false
		}
	}

	return true
}

func onlyLowercaseASCII(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < 'a' || character > 'z' {
			return false
		}
	}

	return true
}

func queryFailure(parameter, code, message string) *QueryError {
	return &QueryError{Status: 400, Parameter: parameter, Code: code, Message: message}
}
