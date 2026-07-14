package jsonapi

import (
	"fmt"
	"mime"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// MediaTypeJSONAPI is the registered JSON:API media type.
const MediaTypeJSONAPI = "application/vnd.api+json"

// MediaType describes extensions and profiles applied to a JSON:API payload.
type MediaType struct {
	Extensions []string
	Profiles   []string
}

// String returns the canonical JSON:API Content-Type value.
func (mediaType MediaType) String() string {
	parameters := make(map[string]string, 2)
	if len(mediaType.Extensions) > 0 {
		parameters["ext"] = strings.Join(uniqueSorted(mediaType.Extensions), " ")
	}
	if len(mediaType.Profiles) > 0 {
		parameters["profile"] = strings.Join(uniqueSorted(mediaType.Profiles), " ")
	}

	return mime.FormatMediaType(MediaTypeJSONAPI, parameters)
}

// NegotiatedMedia is the representation selected for an Accept header.
type NegotiatedMedia struct {
	MediaType   MediaType
	ContentType string
	VaryAccept  bool
}

// NegotiationError describes an HTTP content-negotiation failure without
// coupling the package to a particular HTTP framework.
type NegotiationError struct {
	Status  int
	Code    string
	Message string
}

// Error implements error.
func (err *NegotiationError) Error() string {
	return fmt.Sprintf("JSON:API negotiation failed (%d %s): %s", err.Status, err.Code, err.Message)
}

// Negotiator validates JSON:API request content types and selects response
// media types from Accept headers.
type Negotiator struct {
	extensions map[string]struct{}
	profiles   map[string]struct{}
}

// NewNegotiator constructs a negotiator from supported extension and profile
// URIs. Invalid or duplicate configuration is rejected before serving traffic.
func NewNegotiator(extensions, profiles []string) (*Negotiator, error) {
	negotiator := &Negotiator{
		extensions: make(map[string]struct{}, len(extensions)),
		profiles:   make(map[string]struct{}, len(profiles)),
	}
	if err := addSupportedURIs(negotiator.extensions, extensions, "extension"); err != nil {
		return nil, err
	}
	if err := addSupportedURIs(negotiator.profiles, profiles, "profile"); err != nil {
		return nil, err
	}

	return negotiator, nil
}

// CheckContentType validates the Content-Type of a JSON:API request payload.
// Unknown profiles are retained because profiles cannot change specification
// semantics; unsupported extensions fail with status 415.
func (negotiator *Negotiator) CheckContentType(header string) (MediaType, error) {
	if strings.TrimSpace(header) == "" {
		return MediaType{}, negotiationFailure(415, "unsupported-media-type", "Content-Type is required")
	}

	mediaName, parameters, err := mime.ParseMediaType(header)
	if err != nil || !strings.EqualFold(mediaName, MediaTypeJSONAPI) {
		return MediaType{}, negotiationFailure(415, "unsupported-media-type", "Content-Type must be application/vnd.api+json")
	}
	if unknown := unknownParameter(parameters, false); unknown != "" {
		return MediaType{}, negotiationFailure(415, "unsupported-parameter", "unsupported media type parameter: "+unknown)
	}

	mediaType, err := parseMediaTypeParameters(parameters)
	if err != nil {
		return MediaType{}, negotiationFailure(415, "invalid-parameter", err.Error())
	}
	for _, extension := range mediaType.Extensions {
		if _, supported := negotiator.extensions[extension]; !supported {
			return MediaType{}, negotiationFailure(415, "unsupported-extension", "unsupported extension: "+extension)
		}
	}

	return mediaType, nil
}

// NegotiateAccept selects a JSON:API response representation from an Accept
// header. Invalid candidates and candidates with unsupported extensions are
// ignored, while unknown profiles are ignored within otherwise valid choices.
func (negotiator *Negotiator) NegotiateAccept(header string) (NegotiatedMedia, error) {
	vary := len(negotiator.extensions) > 0 || len(negotiator.profiles) > 0
	if strings.TrimSpace(header) == "" {
		return negotiated(MediaType{}, vary), nil
	}

	candidates := splitHeaderValues(header)
	var selected *acceptCandidate
	for _, raw := range candidates {
		candidate, ok := negotiator.acceptCandidate(raw)
		if !ok {
			continue
		}
		if selected == nil || candidate.quality > selected.quality ||
			candidate.quality == selected.quality && candidate.contentType < selected.contentType {
			copy := candidate
			selected = &copy
		}
	}
	if selected == nil {
		return NegotiatedMedia{}, negotiationFailure(406, "not-acceptable", "no acceptable JSON:API representation")
	}

	return negotiated(selected.mediaType, vary), nil
}

type acceptCandidate struct {
	mediaType   MediaType
	contentType string
	quality     float64
}

func (negotiator *Negotiator) acceptCandidate(raw string) (acceptCandidate, bool) {
	mediaName, parameters, err := mime.ParseMediaType(strings.TrimSpace(raw))
	if err != nil {
		return acceptCandidate{}, false
	}

	quality := 1.0
	if rawQuality, exists := parameters["q"]; exists {
		quality, err = strconv.ParseFloat(rawQuality, 64)
		if err != nil || quality < 0 || quality > 1 {
			return acceptCandidate{}, false
		}
		delete(parameters, "q")
	}
	if quality == 0 {
		return acceptCandidate{}, false
	}

	if mediaName == "*/*" || strings.EqualFold(mediaName, "application/*") {
		if len(parameters) != 0 {
			return acceptCandidate{}, false
		}
		mediaType := MediaType{}
		return acceptCandidate{mediaType: mediaType, contentType: mediaType.String(), quality: quality}, true
	}
	if !strings.EqualFold(mediaName, MediaTypeJSONAPI) {
		return acceptCandidate{}, false
	}
	if unknownParameter(parameters, false) != "" {
		return acceptCandidate{}, false
	}

	requested, err := parseMediaTypeParameters(parameters)
	if err != nil {
		return acceptCandidate{}, false
	}
	for _, extension := range requested.Extensions {
		if _, supported := negotiator.extensions[extension]; !supported {
			return acceptCandidate{}, false
		}
	}

	mediaType := MediaType{Extensions: requested.Extensions}
	for _, profile := range requested.Profiles {
		if _, supported := negotiator.profiles[profile]; supported {
			mediaType.Profiles = append(mediaType.Profiles, profile)
		}
	}
	mediaType.Extensions = uniqueSorted(mediaType.Extensions)
	mediaType.Profiles = uniqueSorted(mediaType.Profiles)

	return acceptCandidate{mediaType: mediaType, contentType: mediaType.String(), quality: quality}, true
}

func negotiated(mediaType MediaType, vary bool) NegotiatedMedia {
	return NegotiatedMedia{
		MediaType:   mediaType,
		ContentType: mediaType.String(),
		VaryAccept:  vary,
	}
}

func parseMediaTypeParameters(parameters map[string]string) (MediaType, error) {
	var mediaType MediaType
	var err error
	if value, exists := parameters["ext"]; exists {
		mediaType.Extensions, err = parseURIList(value, "ext")
		if err != nil {
			return MediaType{}, err
		}
	}
	if value, exists := parameters["profile"]; exists {
		mediaType.Profiles, err = parseURIList(value, "profile")
		if err != nil {
			return MediaType{}, err
		}
	}

	return mediaType, nil
}

func parseURIList(value, parameter string) ([]string, error) {
	items := strings.Fields(value)
	if len(items) == 0 {
		return nil, fmt.Errorf("%s parameter must contain at least one URI", parameter)
	}
	for _, item := range items {
		parsed, err := url.Parse(item)
		if err != nil || !parsed.IsAbs() {
			return nil, fmt.Errorf("%s parameter contains an invalid URI: %s", parameter, item)
		}
	}

	return uniqueSorted(items), nil
}

func unknownParameter(parameters map[string]string, allowQuality bool) string {
	names := make([]string, 0, len(parameters))
	for name := range parameters {
		if name == "ext" || name == "profile" || allowQuality && name == "q" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}

	return names[0]
}

func addSupportedURIs(target map[string]struct{}, values []string, kind string) error {
	for _, value := range values {
		parsed, err := url.Parse(value)
		if err != nil || !parsed.IsAbs() {
			return fmt.Errorf("invalid supported %s URI: %q", kind, value)
		}
		if _, exists := target[value]; exists {
			return fmt.Errorf("duplicate supported %s URI: %q", kind, value)
		}
		target[value] = struct{}{}
	}

	return nil
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)

	return result
}

func splitHeaderValues(header string) []string {
	var values []string
	start := 0
	quoted := false
	escaped := false
	for index, character := range header {
		if escaped {
			escaped = false
			continue
		}
		if character == '\\' && quoted {
			escaped = true
			continue
		}
		if character == '"' {
			quoted = !quoted
			continue
		}
		if character == ',' && !quoted {
			values = append(values, header[start:index])
			start = index + 1
		}
	}
	values = append(values, header[start:])

	return values
}

func negotiationFailure(status int, code, message string) *NegotiationError {
	return &NegotiationError{Status: status, Code: code, Message: message}
}
