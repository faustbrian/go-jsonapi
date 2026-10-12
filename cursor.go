package jsonapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CursorPaginationProfileURI identifies the official Cursor Pagination
// profile. The profile normatively uses this HTTP URI.
const CursorPaginationProfileURI = "http://jsonapi.org/profiles/ethanresnick/cursor-pagination/"

const (
	// CursorUnsupportedSortTypeURI identifies an unsupported sort error.
	CursorUnsupportedSortTypeURI = "https://jsonapi.org/profiles/ethanresnick/cursor-pagination/unsupported-sort"
	// CursorMaxSizeExceededTypeURI identifies a maximum page size error.
	CursorMaxSizeExceededTypeURI = "https://jsonapi.org/profiles/ethanresnick/cursor-pagination/max-size-exceeded"
	// CursorRangeNotSupportedTypeURI identifies a rejected range request.
	CursorRangeNotSupportedTypeURI = "https://jsonapi.org/profiles/ethanresnick/cursor-pagination/range-pagination-not-supported"
)

// CursorPaginationConfig defines endpoint-specific page sizing, range
// support, and opaque cursor validation.
type CursorPaginationConfig struct {
	// DefaultSize is required and positive, and must not exceed MaxSize.
	DefaultSize int
	// MaxSize is a required positive endpoint ceiling; there is no unbounded zero
	// policy.
	MaxSize int
	// AllowRange permits requests with both page[after] and page[before].
	AllowRange bool
	// PageMember aliases the profile metadata element; empty selects page, not a
	// different query family.
	PageMember string
	// ValidateCursor optionally checks an opaque cursor synchronously;
	// authenticity and backend meaning remain caller-owned.
	ValidateCursor func(string) error
	// ValidateSort optionally checks the borrowed sort slice; it must not mutate
	// it or perform unbounded work.
	ValidateSort func([]SortField) error
}

// CursorPageRequest is the validated cursor pagination request for an
// endpoint. It is evidence for backend policy, not a fetched page or an
// authenticated cursor. Presence bits distinguish absence from explicit values.
type CursorPageRequest struct {
	// Size is the admitted effective page size, including the configured default.
	Size int
	// SizePresent records whether page[size] was explicitly supplied.
	SizePresent bool
	// After is an opaque requested cursor, not a decoded backend offset.
	After string
	// AfterPresent preserves explicit page[after] presence independently of its
	// value.
	AfterPresent bool
	// Before is an opaque requested cursor, not an independently validated
	// position.
	Before string
	// BeforePresent preserves explicit page[before] presence independently of its
	// value.
	BeforePresent bool
	// Range reports a request with both boundary cursors.
	Range bool
	// PageMember is the admitted metadata alias used for this request.
	PageMember string
}

// CursorPaginationError describes a profile query failure and its required
// HTTP status.
type CursorPaginationError struct {
	// Status is the profile query failure HTTP status.
	Status int
	// Parameter names the refused decoded query parameter.
	Parameter string
	// Code selects the profile error category and optional type link.
	Code string
	// Message is the diagnostic included by Error, excluding wrapped validator
	// text.
	Message string
	// MaxSize is the endpoint ceiling used for max-size-exceeded metadata.
	MaxSize int
	// PageMember selects the metadata alias; invalid aliases fall back to page in
	// ErrorObject.
	PageMember string
	// Cause retains application callback diagnostics without including their text
	// in Error.
	Cause error
}

// Error implements error.
func (err *CursorPaginationError) Error() string {
	return fmt.Sprintf("invalid cursor pagination parameter %q: %s", err.Parameter, err.Message)
}

// Unwrap returns an application cursor or sort validator failure without
// including its potentially sensitive text in Error.
func (err *CursorPaginationError) Unwrap() error {
	return err.Cause
}

// ErrorObject converts a profile failure to a JSON:API error object with the
// required source, type link, and profile metadata.
func (err *CursorPaginationError) ErrorObject(title, detail string) ErrorObject {
	object := ErrorObject{
		Status: "400",
		Code:   err.Code,
		Title:  title,
		Detail: detail,
		Source: &ErrorSource{Parameter: err.Parameter},
	}
	var typeURI string
	switch err.Code {
	case "unsupported-sort":
		typeURI = CursorUnsupportedSortTypeURI
	case "max-size-exceeded":
		typeURI = CursorMaxSizeExceededTypeURI
		member, memberErr := cursorPageMember(err.PageMember)
		if memberErr != nil {
			member = "page"
		}
		object.Meta = Meta{member: map[string]any{"maxSize": err.MaxSize}}
	case "range-not-supported":
		typeURI = CursorRangeNotSupportedTypeURI
	}
	if typeURI != "" {
		object.Links = Links{"type": URI(typeURI)}
	}

	return object
}

// ValidateCursorPaginationLinks enforces the profile requirement that every
// paginated data instance contains both prev and next links.
func ValidateCursorPaginationLinks(links Links) error {
	validator := documentValidator{}
	if _, exists := links["prev"]; !exists {
		validator.add("/links/prev", "required", "cursor pagination requires a prev link")
	}
	if _, exists := links["next"]; !exists {
		validator.add("/links/next", "required", "cursor pagination requires a next link")
	}
	validator.validateLinks(links, "/links")
	if len(validator.violations) == 0 {
		return nil
	}

	return &ValidationError{Violations: validator.violations}
}

// CursorPagination parses the page family for one configured endpoint. Use
// NewCursorPagination; immutable policy can be shared when callbacks are safe
// for concurrent calls. Callbacks execute synchronously without deadlines and
// ordinary panics are wrapped, not blocking or goroutine termination.
type CursorPagination struct {
	defaultSize    int
	maxSize        int
	allowRange     bool
	validateCursor func(string) error
	validateSort   func([]SortField) error
	pageMember     string
}

// NewCursorPagination validates endpoint policy before it serves requests.
// MaxSize must be positive and finite for both ordinary and range pagination.
func NewCursorPagination(config CursorPaginationConfig) (*CursorPagination, error) {
	if config.DefaultSize < 1 {
		return nil, fmt.Errorf("cursor pagination default size must be positive")
	}
	if config.MaxSize <= 0 {
		return nil, fmt.Errorf("cursor pagination max size must be positive")
	}
	if config.DefaultSize > config.MaxSize {
		return nil, fmt.Errorf("cursor pagination default size must not exceed max size")
	}
	pageMember, err := cursorPageMember(config.PageMember)
	if err != nil {
		return nil, err
	}

	return &CursorPagination{
		defaultSize:    config.DefaultSize,
		maxSize:        config.MaxSize,
		allowRange:     config.AllowRange,
		validateCursor: config.ValidateCursor,
		validateSort:   config.ValidateSort,
		pageMember:     pageMember,
	}, nil
}

// ParseQuery validates both the page family and the endpoint's stable sorting
// requirement. When ValidateSort is nil, the caller remains responsible for
// applying a unique order before fetching the page. Requested sort fields are
// borrowed by the validator, not copied or applied to storage. All failures
// return a zero request; this call performs no cursor decoding or data fetch.
func (pagination *CursorPagination) ParseQuery(query Query) (CursorPageRequest, error) {
	if err := pagination.admit(query.Page, query.Sort); err != nil {
		return CursorPageRequest{}, err
	}
	request, err := pagination.Parse(query.Page)
	if err != nil {
		return CursorPageRequest{}, err
	}
	if pagination.validateSort != nil {
		if err := callApplicationCallback("sort", func() error {
			return pagination.validateSort(query.Sort)
		}); err != nil {
			return CursorPageRequest{}, pagination.failureWithCause(
				"sort", "unsupported-sort", "requested sort is unsupported", 0, err,
			)
		}
	}

	return request, nil
}

// Parse validates the page parameter family according to the profile and
// endpoint configuration. Direct inputs use DefaultQueryLimits before any
// sorting or application callback. ParseQuery applies those limits to the
// combined page and sort input before validating either part.
func (pagination *CursorPagination) Parse(family ParameterFamily) (CursorPageRequest, error) {
	if err := pagination.admit(family, nil); err != nil {
		return CursorPageRequest{}, err
	}
	names := make([]string, 0, len(family))
	for name := range family {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if name != "page[size]" && name != "page[after]" && name != "page[before]" {
			return CursorPageRequest{}, pagination.failure(
				name,
				"unknown-parameter",
				"parameter is not defined by the Cursor Pagination profile",
				0,
			)
		}
	}

	request := CursorPageRequest{Size: pagination.defaultSize, PageMember: pagination.pageMember}
	if values, exists := family["page[size]"]; exists {
		if len(values) != 1 {
			return CursorPageRequest{}, pagination.failure(
				"page[size]", "multiple-values", "page size must occur once", 0,
			)
		}
		size, err := positiveDecimal(values[0])
		if err != nil {
			return CursorPageRequest{}, pagination.failure(
				"page[size]", "invalid-parameter", "page size must be a positive integer", 0,
			)
		}
		if size > pagination.maxSize {
			return CursorPageRequest{}, pagination.failure(
				"page[size]",
				"max-size-exceeded",
				"page size exceeds the endpoint maximum",
				pagination.maxSize,
			)
		}
		request.Size = size
		request.SizePresent = true
	}

	for _, field := range []struct {
		name    string
		target  *string
		present *bool
	}{
		{"page[after]", &request.After, &request.AfterPresent},
		{"page[before]", &request.Before, &request.BeforePresent},
	} {
		values, exists := family[field.name]
		if !exists {
			continue
		}
		if len(values) != 1 {
			return CursorPageRequest{}, pagination.failure(
				field.name, "multiple-values", "cursor parameter must occur once", 0,
			)
		}
		if pagination.validateCursor != nil {
			if err := callApplicationCallback("cursor", func() error {
				return pagination.validateCursor(values[0])
			}); err != nil {
				return CursorPageRequest{}, pagination.failureWithCause(
					field.name,
					"invalid-parameter",
					"cursor value is invalid",
					0,
					err,
				)
			}
		}
		*field.target = values[0]
		*field.present = true
	}

	request.Range = request.AfterPresent && request.BeforePresent
	if request.Range && !pagination.allowRange {
		return CursorPageRequest{}, pagination.failure(
			"page[before]",
			"range-not-supported",
			"endpoint does not support range pagination",
			0,
		)
	}
	if request.Range && !request.SizePresent {
		request.Size = pagination.maxSize
	}

	return request, nil
}

// admit bounds direct inputs and QueryParser output before sorting or callbacks.
// Remaining budgets avoid overflow; sort bytes are charged in their equivalent
// encoded form without constructing the joined value.
func (pagination *CursorPagination) admit(family ParameterFamily, fields []SortField) error {
	limits := DefaultQueryLimits()
	refuse := func() error {
		return pagination.failure("", "limit", "cursor pagination input exceeds resource limits", 0)
	}
	if len(family) > limits.MaxParameters || len(fields) > limits.MaxListItems {
		return refuse()
	}
	bytesRemaining, valuesRemaining := limits.MaxTotalBytes, limits.MaxValues
	for name, values := range family {
		if len(name) > limits.MaxNameBytes || len(name) > bytesRemaining || len(values) > valuesRemaining {
			return refuse()
		}
		bytesRemaining -= len(name)
		valuesRemaining -= len(values)
		for _, value := range values {
			if len(value) > limits.MaxValueBytes || len(value) > bytesRemaining {
				return refuse()
			}
			bytesRemaining -= len(value)
		}
	}
	if len(fields) == 0 {
		return nil
	}
	if len(family) == limits.MaxParameters || valuesRemaining == 0 || bytesRemaining < len("sort") {
		return refuse()
	}
	bytesRemaining -= len("sort")
	sortRemaining := limits.MaxValueBytes
	for index, field := range fields {
		prefix := 0
		if index > 0 {
			prefix++
		}
		if field.Descending {
			prefix++
		}
		if prefix > sortRemaining || prefix > bytesRemaining {
			return refuse()
		}
		sortRemaining -= prefix
		bytesRemaining -= prefix
		if len(field.Name) > sortRemaining || len(field.Name) > bytesRemaining {
			return refuse()
		}
		sortRemaining -= len(field.Name)
		bytesRemaining -= len(field.Name)
	}
	return nil
}

func positiveDecimal(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("empty decimal")
	}
	if strings.Trim(value, "0123456789") != "" {
		return 0, fmt.Errorf("non-decimal character")
	}
	size, err := strconv.Atoi(value)
	if err != nil || size < 1 {
		return 0, fmt.Errorf("decimal must be positive")
	}

	return size, nil
}

func (pagination *CursorPagination) failure(
	parameter, code, message string,
	maxSize int,
) *CursorPaginationError {
	err := cursorFailure(parameter, code, message, maxSize)
	err.PageMember = pagination.pageMember
	return err
}

func (pagination *CursorPagination) failureWithCause(
	parameter, code, message string,
	maxSize int,
	cause error,
) error {
	err := pagination.failure(parameter, code, message, maxSize)
	err.Cause = cause
	return err
}

func cursorFailure(parameter, code, message string, maxSize int) *CursorPaginationError {
	return &CursorPaginationError{
		Status:    400,
		Parameter: parameter,
		Code:      code,
		Message:   message,
		MaxSize:   maxSize,
	}
}
