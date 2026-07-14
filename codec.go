package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// DecodeError identifies a malformed JSON representation by JSON Pointer.
type DecodeError struct {
	Path    string
	Code    string
	Message string
	Cause   error
}

// Error implements error.
func (err *DecodeError) Error() string {
	if err.Path == "" {
		return "decode JSON:API document: " + err.Message
	}

	return fmt.Sprintf("decode JSON:API document at %q: %s", err.Path, err.Message)
}

// Unwrap returns the underlying JSON decoding error, when present.
func (err *DecodeError) Unwrap() error {
	return err.Cause
}

// Marshal validates and deterministically encodes a JSON:API document.
func Marshal(document Document) ([]byte, error) {
	return MarshalWith(document, ValidationOptions{})
}

// MarshalWith validates in the supplied protocol context and deterministically
// encodes a JSON:API document.
func MarshalWith(document Document, options ValidationOptions) ([]byte, error) {
	if err := document.ValidateWith(options); err != nil {
		return nil, err
	}
	if err := validateDocumentMembers(document, nil); err != nil {
		return nil, err
	}

	return json.Marshal(document)
}

// Unmarshal strictly decodes and validates a JSON:API document.
func Unmarshal(payload []byte) (Document, error) {
	return UnmarshalWith(payload, ValidationOptions{})
}

// UnmarshalWith strictly decodes and validates a JSON:API document in the
// supplied protocol context.
func UnmarshalWith(payload []byte, options ValidationOptions) (Document, error) {
	if !json.Valid(payload) {
		return Document{}, decodeFailure("", "syntax", "invalid JSON", nil)
	}
	if err := rejectDuplicateMembers(payload); err != nil {
		return Document{}, err
	}

	root, err := decodeObject(payload, "")
	if err != nil {
		return Document{}, err
	}
	if err := rejectUnknown(root, "", "jsonapi", "links", "data", "included", "errors", "meta"); err != nil {
		return Document{}, err
	}

	var document Document
	if raw, exists := root["jsonapi"]; exists {
		object, decodeErr := decodeJSONAPI(raw, "/jsonapi")
		if decodeErr != nil {
			return Document{}, decodeErr
		}
		document.JSONAPI = &object
	}
	if raw, exists := root["links"]; exists {
		links, decodeErr := decodeLinks(raw, "/links")
		if decodeErr != nil {
			return Document{}, decodeErr
		}
		document.Links = links
	}
	if raw, exists := root["data"]; exists {
		data, decodeErr := decodePrimaryData(raw, "/data")
		if decodeErr != nil {
			return Document{}, decodeErr
		}
		document.Data = data
	}
	if raw, exists := root["included"]; exists {
		included, decodeErr := decodeResourceArray(raw, "/included")
		if decodeErr != nil {
			return Document{}, decodeErr
		}
		document.Included = included
	}
	if raw, exists := root["errors"]; exists {
		errors, decodeErr := decodeErrors(raw, "/errors")
		if decodeErr != nil {
			return Document{}, decodeErr
		}
		document.Errors = errors
	}
	if raw, exists := root["meta"]; exists {
		meta, decodeErr := decodeMeta(raw, "/meta")
		if decodeErr != nil {
			return Document{}, decodeErr
		}
		document.Meta = meta
	}

	if err := document.ValidateWith(options); err != nil {
		return Document{}, err
	}

	return document, nil
}

func rejectDuplicateMembers(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	return scanJSONValue(decoder, "")
}

func scanJSONValue(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return decodeFailure(path, "syntax", "invalid JSON token", err)
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			nameToken, tokenErr := decoder.Token()
			if tokenErr != nil {
				return decodeFailure(path, "syntax", "invalid object member", tokenErr)
			}
			name, ok := nameToken.(string)
			if !ok {
				return decodeFailure(path, "syntax", "object member name must be a string", nil)
			}
			memberPath := path + "/" + escapePointerToken(name)
			if _, exists := seen[name]; exists {
				return decodeFailure(memberPath, "duplicate-member", "object member occurs more than once", nil)
			}
			seen[name] = struct{}{}
			if err := scanJSONValue(decoder, memberPath); err != nil {
				return err
			}
		}
	case '[':
		for index := 0; decoder.More(); index++ {
			if err := scanJSONValue(decoder, path+"/"+strconv.Itoa(index)); err != nil {
				return err
			}
		}
	default:
		return decodeFailure(path, "syntax", "unexpected JSON delimiter", nil)
	}
	if _, err := decoder.Token(); err != nil {
		return decodeFailure(path, "syntax", "unterminated JSON value", err)
	}

	return nil
}

func decodeJSONAPI(raw json.RawMessage, path string) (JSONAPI, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return JSONAPI{}, err
	}
	if err := rejectUnknown(object, path, "version", "ext", "profile", "meta"); err != nil {
		return JSONAPI{}, err
	}

	var result JSONAPI
	if value, exists := object["version"]; exists {
		if err := decodeString(value, path+"/version", &result.Version); err != nil {
			return JSONAPI{}, err
		}
	}
	if value, exists := object["ext"]; exists {
		if err := decodeStringArray(value, path+"/ext", &result.Ext); err != nil {
			return JSONAPI{}, err
		}
	}
	if value, exists := object["profile"]; exists {
		if err := decodeStringArray(value, path+"/profile", &result.Profile); err != nil {
			return JSONAPI{}, err
		}
	}
	if value, exists := object["meta"]; exists {
		meta, decodeErr := decodeMeta(value, path+"/meta")
		if decodeErr != nil {
			return JSONAPI{}, decodeErr
		}
		result.Meta = meta
	}

	return result, nil
}

func decodePrimaryData(raw json.RawMessage, path string) (*PrimaryData, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return NullData(), nil
	}
	if len(trimmed) == 0 {
		return nil, decodeFailure(path, "type", "primary data has no value", nil)
	}

	switch trimmed[0] {
	case '{':
		resource, err := decodeResource(trimmed, path)
		if err != nil {
			return nil, err
		}
		return ResourceData(resource), nil
	case '[':
		resources, err := decodeResourceArray(trimmed, path)
		if err != nil {
			return nil, err
		}
		return ResourceCollection(resources...), nil
	default:
		return nil, decodeFailure(path, "type", "primary data must be null, an object, or an array", nil)
	}
}

func decodeResourceArray(raw json.RawMessage, path string) ([]ResourceObject, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || items == nil {
		return nil, decodeFailure(path, "type", "resource collection must be an array", err)
	}

	resources := make([]ResourceObject, len(items))
	for index, item := range items {
		resource, err := decodeResource(item, path+"/"+strconv.Itoa(index))
		if err != nil {
			return nil, err
		}
		resources[index] = resource
	}

	return resources, nil
}

func decodeResource(raw json.RawMessage, path string) (ResourceObject, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return ResourceObject{}, err
	}
	if err := rejectUnknown(object, path, "type", "id", "lid", "attributes", "relationships", "links", "meta"); err != nil {
		return ResourceObject{}, err
	}

	var resource ResourceObject
	for name, target := range map[string]*string{
		"type": &resource.Type,
		"id":   &resource.ID,
		"lid":  &resource.LID,
	} {
		if value, exists := object[name]; exists {
			if err := decodeString(value, path+"/"+name, target); err != nil {
				return ResourceObject{}, err
			}
		}
	}
	if value, exists := object["attributes"]; exists {
		attributes, decodeErr := decodeAttributes(value, path+"/attributes")
		if decodeErr != nil {
			return ResourceObject{}, decodeErr
		}
		resource.Attributes = attributes
	}
	if value, exists := object["relationships"]; exists {
		relationships, decodeErr := decodeRelationships(value, path+"/relationships")
		if decodeErr != nil {
			return ResourceObject{}, decodeErr
		}
		resource.Relationships = relationships
	}
	if value, exists := object["links"]; exists {
		links, decodeErr := decodeLinks(value, path+"/links")
		if decodeErr != nil {
			return ResourceObject{}, decodeErr
		}
		resource.Links = links
	}
	if value, exists := object["meta"]; exists {
		meta, decodeErr := decodeMeta(value, path+"/meta")
		if decodeErr != nil {
			return ResourceObject{}, decodeErr
		}
		resource.Meta = meta
	}

	return resource, nil
}

func decodeAttributes(raw json.RawMessage, path string) (Attributes, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, err
	}

	attributes := make(Attributes, len(object))
	for name, value := range object {
		if strings.HasPrefix(name, "@") {
			continue
		}
		var attribute any
		if err := json.Unmarshal(value, &attribute); err != nil {
			return nil, decodeFailure(path+"/"+escapePointerToken(name), "syntax", "invalid attribute value", err)
		}
		attributes[name] = stripAtMembers(attribute)
	}

	return attributes, nil
}

func decodeRelationships(raw json.RawMessage, path string) (Relationships, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, err
	}

	relationships := make(Relationships, len(object))
	for name, value := range object {
		if strings.HasPrefix(name, "@") {
			continue
		}
		relationship, decodeErr := decodeRelationship(value, path+"/"+escapePointerToken(name))
		if decodeErr != nil {
			return nil, decodeErr
		}
		relationships[name] = relationship
	}

	return relationships, nil
}

func decodeRelationship(raw json.RawMessage, path string) (Relationship, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return Relationship{}, err
	}
	if err := rejectUnknown(object, path, "links", "data", "meta"); err != nil {
		return Relationship{}, err
	}

	var relationship Relationship
	if value, exists := object["links"]; exists {
		links, decodeErr := decodeLinks(value, path+"/links")
		if decodeErr != nil {
			return Relationship{}, decodeErr
		}
		relationship.Links = links
	}
	if value, exists := object["data"]; exists {
		data, decodeErr := decodeRelationshipData(value, path+"/data")
		if decodeErr != nil {
			return Relationship{}, decodeErr
		}
		relationship.Data = data
	}
	if value, exists := object["meta"]; exists {
		meta, decodeErr := decodeMeta(value, path+"/meta")
		if decodeErr != nil {
			return Relationship{}, decodeErr
		}
		relationship.Meta = meta
	}

	return relationship, nil
}

func decodeRelationshipData(raw json.RawMessage, path string) (*RelationshipData, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return NullRelationship(), nil
	}
	if len(trimmed) == 0 {
		return nil, decodeFailure(path, "type", "relationship data has no value", nil)
	}

	switch trimmed[0] {
	case '{':
		identifier, err := decodeIdentifier(trimmed, path)
		if err != nil {
			return nil, err
		}
		return ToOne(identifier), nil
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err != nil || items == nil {
			return nil, decodeFailure(path, "type", "to-many linkage must be an array", err)
		}
		identifiers := make([]Identifier, len(items))
		for index, item := range items {
			identifier, err := decodeIdentifier(item, path+"/"+strconv.Itoa(index))
			if err != nil {
				return nil, err
			}
			identifiers[index] = identifier
		}
		return ToMany(identifiers...), nil
	default:
		return nil, decodeFailure(path, "type", "relationship data must be null, an object, or an array", nil)
	}
}

func decodeIdentifier(raw json.RawMessage, path string) (Identifier, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return Identifier{}, err
	}
	if err := rejectUnknown(object, path, "type", "id", "lid", "meta"); err != nil {
		return Identifier{}, err
	}

	var identifier Identifier
	for name, target := range map[string]*string{
		"type": &identifier.Type,
		"id":   &identifier.ID,
		"lid":  &identifier.LID,
	} {
		if value, exists := object[name]; exists {
			if err := decodeString(value, path+"/"+name, target); err != nil {
				return Identifier{}, err
			}
		}
	}
	if value, exists := object["meta"]; exists {
		meta, decodeErr := decodeMeta(value, path+"/meta")
		if decodeErr != nil {
			return Identifier{}, decodeErr
		}
		identifier.Meta = meta
	}

	return identifier, nil
}

func decodeLinks(raw json.RawMessage, path string) (Links, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, err
	}

	links := make(Links, len(object))
	for name, value := range object {
		if strings.HasPrefix(name, "@") {
			continue
		}
		link, decodeErr := decodeLink(value, path+"/"+escapePointerToken(name))
		if decodeErr != nil {
			return nil, decodeErr
		}
		links[name] = link
	}

	return links, nil
}

func decodeLink(raw json.RawMessage, path string) (Link, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return NullLink(), nil
	}
	if len(trimmed) == 0 {
		return Link{}, decodeFailure(path, "type", "link has no value", nil)
	}
	if trimmed[0] == '"' {
		var href string
		if err := decodeString(trimmed, path, &href); err != nil {
			return Link{}, err
		}
		return URI(href), nil
	}
	if trimmed[0] != '{' {
		return Link{}, decodeFailure(path, "type", "link must be a string, object, or null", nil)
	}

	object, err := decodeObject(trimmed, path)
	if err != nil {
		return Link{}, err
	}
	if err := rejectUnknown(
		object,
		path,
		"href",
		"rel",
		"describedby",
		"title",
		"type",
		"hreflang",
		"meta",
	); err != nil {
		return Link{}, err
	}
	var result LinkObject
	if value, exists := object["href"]; exists {
		if err := decodeString(value, path+"/href", &result.Href); err != nil {
			return Link{}, err
		}
	}
	for name, target := range map[string]*string{
		"rel":   &result.Rel,
		"title": &result.Title,
		"type":  &result.Type,
	} {
		if value, exists := object[name]; exists {
			if err := decodeString(value, path+"/"+name, target); err != nil {
				return Link{}, err
			}
		}
	}
	if value, exists := object["describedby"]; exists {
		describedBy, decodeErr := decodeLink(value, path+"/describedby")
		if decodeErr != nil {
			return Link{}, decodeErr
		}
		result.DescribedBy = &describedBy
	}
	if value, exists := object["hreflang"]; exists {
		hreflang, decodeErr := decodeHreflang(value, path+"/hreflang")
		if decodeErr != nil {
			return Link{}, decodeErr
		}
		result.Hreflang = hreflang
	}
	if value, exists := object["meta"]; exists {
		decoded, decodeErr := decodeMeta(value, path+"/meta")
		if decodeErr != nil {
			return Link{}, decodeErr
		}
		result.Meta = decoded
	}

	return LinkFromObject(result), nil
}

func decodeHreflang(raw json.RawMessage, path string) (*LinkHreflang, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, decodeFailure(path, "type", "hreflang has no value", nil)
	}
	if trimmed[0] == '"' {
		var tag string
		if err := decodeString(trimmed, path, &tag); err != nil {
			return nil, err
		}
		return LanguageTag(tag), nil
	}
	if trimmed[0] != '[' {
		return nil, decodeFailure(path, "type", "hreflang must be a string or array of strings", nil)
	}
	var tags []string
	if err := decodeStringArray(trimmed, path, &tags); err != nil {
		return nil, err
	}

	return LanguageTags(tags...), nil
}

func decodeErrors(raw json.RawMessage, path string) ([]ErrorObject, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || items == nil {
		return nil, decodeFailure(path, "type", "errors must be an array", err)
	}

	result := make([]ErrorObject, len(items))
	for index, item := range items {
		apiError, err := decodeError(item, path+"/"+strconv.Itoa(index))
		if err != nil {
			return nil, err
		}
		result[index] = apiError
	}

	return result, nil
}

func decodeError(raw json.RawMessage, path string) (ErrorObject, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return ErrorObject{}, err
	}
	if err := rejectUnknown(object, path, "id", "links", "status", "code", "title", "detail", "source", "meta"); err != nil {
		return ErrorObject{}, err
	}

	var result ErrorObject
	for name, target := range map[string]*string{
		"id":     &result.ID,
		"status": &result.Status,
		"code":   &result.Code,
		"title":  &result.Title,
		"detail": &result.Detail,
	} {
		if value, exists := object[name]; exists {
			if err := decodeString(value, path+"/"+name, target); err != nil {
				return ErrorObject{}, err
			}
		}
	}
	if value, exists := object["links"]; exists {
		links, decodeErr := decodeLinks(value, path+"/links")
		if decodeErr != nil {
			return ErrorObject{}, decodeErr
		}
		result.Links = links
	}
	if value, exists := object["source"]; exists {
		source, decodeErr := decodeErrorSource(value, path+"/source")
		if decodeErr != nil {
			return ErrorObject{}, decodeErr
		}
		result.Source = &source
	}
	if value, exists := object["meta"]; exists {
		meta, decodeErr := decodeMeta(value, path+"/meta")
		if decodeErr != nil {
			return ErrorObject{}, decodeErr
		}
		result.Meta = meta
	}

	return result, nil
}

func decodeErrorSource(raw json.RawMessage, path string) (ErrorSource, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return ErrorSource{}, err
	}
	if err := rejectUnknown(object, path, "pointer", "parameter", "header"); err != nil {
		return ErrorSource{}, err
	}

	var result ErrorSource
	fields := []struct {
		name   string
		target *string
	}{
		{"pointer", &result.Pointer},
		{"parameter", &result.Parameter},
		{"header", &result.Header},
	}
	for _, field := range fields {
		name := field.name
		if value, exists := object[name]; exists {
			if err := decodeString(value, path+"/"+name, field.target); err != nil {
				return ErrorSource{}, err
			}
		}
	}

	return result, nil
}

func decodeMeta(raw json.RawMessage, path string) (Meta, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, err
	}

	meta := make(Meta, len(object))
	for name, value := range object {
		if strings.HasPrefix(name, "@") {
			continue
		}
		var item any
		decoder := json.NewDecoder(bytes.NewReader(value))
		decoder.UseNumber()
		if err := decoder.Decode(&item); err != nil {
			return nil, decodeFailure(path+"/"+escapePointerToken(name), "syntax", "invalid meta value", err)
		}
		meta[name] = stripAtMembers(item)
	}

	return meta, nil
}

func decodeObject(raw []byte, path string) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, decodeFailure(path, "type", "value must be an object", nil)
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &object); err != nil {
		return nil, decodeFailure(path, "type", "value must be an object", err)
	}

	return object, nil
}

func rejectUnknown(object map[string]json.RawMessage, path string, allowed ...string) error {
	known := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		known[name] = struct{}{}
	}

	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if strings.HasPrefix(name, "@") {
			continue
		}
		if _, exists := known[name]; !exists {
			return decodeFailure(
				path+"/"+escapePointerToken(name),
				"unknown-member",
				"member is not defined by JSON:API",
				nil,
			)
		}
	}

	return nil
}

func decodeString(raw json.RawMessage, path string, target *string) error {
	if err := json.Unmarshal(raw, target); err != nil {
		return decodeFailure(path, "type", "value must be a string", err)
	}

	return nil
}

func decodeStringArray(raw json.RawMessage, path string, target *[]string) error {
	if err := json.Unmarshal(raw, target); err != nil || *target == nil {
		return decodeFailure(path, "type", "value must be an array of strings", err)
	}

	return nil
}

func stripAtMembers(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for name, child := range typed {
			if strings.HasPrefix(name, "@") {
				delete(typed, name)
				continue
			}
			typed[name] = stripAtMembers(child)
		}
	case []any:
		for index, child := range typed {
			typed[index] = stripAtMembers(child)
		}
	}

	return value
}

func decodeFailure(path, code, message string, cause error) *DecodeError {
	return &DecodeError{Path: path, Code: code, Message: message, Cause: cause}
}
