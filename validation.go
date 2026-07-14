package jsonapi

import (
	"fmt"
	"mime"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/language"
)

var registeredLinkRelation = regexp.MustCompile(`^[a-z][a-z0-9.-]*$`)

// Violation describes one JSON:API conformance failure.
type Violation struct {
	Path    string
	Code    string
	Message string
}

// ValidationError reports every conformance violation found in a document.
type ValidationError struct {
	Violations []Violation
}

// Error implements error.
func (err *ValidationError) Error() string {
	if len(err.Violations) == 0 {
		return "JSON:API validation failed"
	}

	first := err.Violations[0]
	if len(err.Violations) == 1 {
		return fmt.Sprintf("JSON:API validation failed at %q: %s", first.Path, first.Message)
	}

	return fmt.Sprintf(
		"JSON:API validation failed at %q: %s (and %d more violations)",
		first.Path,
		first.Message,
		len(err.Violations)-1,
	)
}

// Validate checks the context-independent structural requirements of a
// JSON:API document and returns all violations in document order.
func (document Document) Validate() error {
	validator := documentValidator{}
	validator.validateDocument(document)
	if len(validator.violations) == 0 {
		return nil
	}

	return &ValidationError{Violations: validator.violations}
}

type documentValidator struct {
	violations []Violation
}

func (validator *documentValidator) validateDocument(document Document) {
	if document.Data == nil && len(document.Errors) == 0 && document.Meta == nil {
		validator.add("", "required", "document must contain data, errors, or meta")
	}
	if document.Data != nil && len(document.Errors) > 0 {
		validator.add("/errors", "conflict", "data and errors must not coexist")
	}
	if document.Data == nil && document.Included != nil {
		validator.add("/included", "requires-data", "included requires top-level data")
	}

	if document.JSONAPI != nil {
		validator.validateJSONAPI(*document.JSONAPI)
	}
	validator.validateLinks(document.Links, "/links")
	validator.validatePrimaryData(document.Data, "/data")
	validator.validateIncluded(document)
	for index, apiError := range document.Errors {
		validator.validateError(apiError, "/errors/"+strconv.Itoa(index))
	}
}

func (validator *documentValidator) validateJSONAPI(object JSONAPI) {
	for index, extension := range object.Ext {
		validator.validateURL(extension, "/jsonapi/ext/"+strconv.Itoa(index))
	}
	for index, profile := range object.Profile {
		validator.validateURL(profile, "/jsonapi/profile/"+strconv.Itoa(index))
	}
}

func (validator *documentValidator) validatePrimaryData(data *PrimaryData, path string) {
	if data == nil || data.kind == primaryDataNull {
		return
	}
	if data.kind == primaryDataOne {
		if data.one == nil {
			validator.add(path, "required", "single primary data must contain a resource")
			return
		}
		validator.validateResource(*data.one, path)
		return
	}
	if data.kind != primaryDataMany {
		validator.add(path, "shape", "primary data must be null, a resource, or an array")
		return
	}
	for index, resource := range data.many {
		validator.validateResource(resource, path+"/"+strconv.Itoa(index))
	}
}

func (validator *documentValidator) validateResource(resource ResourceObject, path string) {
	if resource.Type == "" {
		validator.add(path+"/type", "required", "resource type is required")
	} else if !validMemberName(resource.Type) {
		validator.add(path+"/type", "member-name", "resource type must be a valid member name")
	}
	if resource.ID == "" && resource.LID == "" {
		validator.add(path+"/id", "required", "resource id is required")
	}

	for name := range resource.Attributes {
		fieldPath := path + "/attributes/" + escapePointerToken(name)
		if name == "id" || name == "type" || name == "lid" {
			validator.add(fieldPath, "reserved-field", "attribute name conflicts with resource identity")
		} else if !validMemberName(name) {
			validator.add(fieldPath, "member-name", "attribute name is invalid")
		}
	}
	for name, relationship := range resource.Relationships {
		fieldPath := path + "/relationships/" + escapePointerToken(name)
		if name == "id" || name == "type" || name == "lid" {
			validator.add(fieldPath, "reserved-field", "relationship name conflicts with resource identity")
		} else if !validMemberName(name) {
			validator.add(fieldPath, "member-name", "relationship name is invalid")
		}
		if _, exists := resource.Attributes[name]; exists && !strings.HasPrefix(name, "@") {
			validator.add(fieldPath, "duplicate-field", "attribute and relationship names must be unique")
		}
		validator.validateRelationship(relationship, fieldPath)
	}
	validator.validateLinks(resource.Links, path+"/links")
}

func (validator *documentValidator) validateRelationship(relationship Relationship, path string) {
	if relationship.Links == nil && relationship.Data == nil && relationship.Meta == nil {
		validator.add(path, "required", "relationship must contain links, data, or meta")
	}
	validator.validateLinks(relationship.Links, path+"/links")
	validator.validateRelationshipData(relationship.Data, path+"/data")
}

func (validator *documentValidator) validateRelationshipData(data *RelationshipData, path string) {
	if data == nil || data.kind == relationshipDataNull {
		return
	}
	if data.kind == relationshipDataOne {
		if data.one == nil {
			validator.add(path, "required", "to-one linkage must contain an identifier")
			return
		}
		validator.validateIdentifier(*data.one, path)
		return
	}
	if data.kind != relationshipDataMany {
		validator.add(path, "shape", "linkage must be null, an identifier, or an array")
		return
	}
	for index, identifier := range data.many {
		validator.validateIdentifier(identifier, path+"/"+strconv.Itoa(index))
	}
}

func (validator *documentValidator) validateIdentifier(identifier Identifier, path string) {
	if identifier.Type == "" {
		validator.add(path+"/type", "required", "resource identifier type is required")
	} else if !validMemberName(identifier.Type) {
		validator.add(path+"/type", "member-name", "resource identifier type must be a valid member name")
	}
	if identifier.ID == "" && identifier.LID == "" {
		validator.add(path+"/id", "required", "resource identifier requires id or lid")
	}
}

func (validator *documentValidator) validateLinks(links Links, path string) {
	for name, link := range links {
		linkPath := path + "/" + escapePointerToken(name)
		if !validMemberName(name) {
			validator.add(linkPath, "member-name", "link name is invalid")
		}
		validator.validateLink(link, linkPath)
	}
}

func (validator *documentValidator) validateLink(link Link, path string) {
	if link.null {
		return
	}
	if link.object && link.href == "" {
		validator.add(path+"/href", "required", "link object href is required")
	} else {
		validator.validateURL(link.href, path)
	}
	if link.rel != "" && !validLinkRelation(link.rel) {
		validator.add(path+"/rel", "link-relation", "rel must be a registered relation or URI")
	}
	if link.describedBy != nil {
		validator.validateLink(*link.describedBy, path+"/describedby")
	}
	if link.targetType != "" {
		if mediaType, _, err := mime.ParseMediaType(link.targetType); err != nil || mediaType == "" {
			validator.add(path+"/type", "media-type", "type must be a valid media type")
		}
	}
	if link.hreflang != nil {
		for _, tag := range link.hreflang.values {
			if !validLanguageTag(tag) {
				validator.add(path+"/hreflang", "language-tag", "hreflang must contain valid language tags")
				break
			}
		}
	}
}

func validLanguageTag(tag string) bool {
	if tag == "" || strings.HasPrefix(tag, "-") || strings.HasSuffix(tag, "-") || strings.Contains(tag, "--") {
		return false
	}
	for _, character := range tag {
		if character != '-' &&
			!(character >= 'a' && character <= 'z') &&
			!(character >= 'A' && character <= 'Z') &&
			!(character >= '0' && character <= '9') {
			return false
		}
	}
	_, err := language.Parse(tag)

	return err == nil
}

func validLinkRelation(relation string) bool {
	if registeredLinkRelation.MatchString(relation) {
		return true
	}
	parsed, err := url.Parse(relation)

	return err == nil && parsed.IsAbs()
}

func (validator *documentValidator) validateURL(value, path string) {
	parsed, err := url.Parse(value)
	if value == "" || err != nil || (parsed.Scheme == "" && parsed.Path == "") {
		validator.add(path, "url", "link must contain a valid URL")
	}
}

func (validator *documentValidator) validateError(apiError ErrorObject, path string) {
	if apiError.ID == "" && apiError.Links == nil && apiError.Status == "" &&
		apiError.Code == "" && apiError.Title == "" && apiError.Detail == "" &&
		apiError.Source == nil && apiError.Meta == nil {
		validator.add(path, "required", "error object must contain at least one member")
	}
	validator.validateLinks(apiError.Links, path+"/links")
	if apiError.Source != nil && !validJSONPointer(apiError.Source.Pointer) {
		validator.add(path+"/source/pointer", "json-pointer", "source pointer must be a JSON Pointer")
	}
}

func (validator *documentValidator) validateIncluded(document Document) {
	if document.Included == nil {
		return
	}

	included := make(map[string]int, len(document.Included))
	for index, resource := range document.Included {
		path := "/included/" + strconv.Itoa(index)
		validator.validateResource(resource, path)
		key := resourceKey(resource.Type, resource.ID, resource.LID)
		if previous, exists := included[key]; exists {
			validator.add(
				path,
				"duplicate-resource",
				fmt.Sprintf("resource duplicates included resource at index %d", previous),
			)
		} else {
			included[key] = index
		}
	}

	reachable := make(map[string]bool, len(document.Included))
	queue := validator.primaryLinkage(document.Data)
	for len(queue) > 0 {
		identifier := queue[0]
		queue = queue[1:]
		key := identifierKey(identifier)
		index, exists := included[key]
		if !exists || reachable[key] {
			continue
		}
		reachable[key] = true
		queue = append(queue, relationshipIdentifiers(document.Included[index].Relationships)...)
	}

	for key, index := range included {
		if !reachable[key] {
			validator.add(
				"/included/"+strconv.Itoa(index),
				"full-linkage",
				"included resource is not linked from primary data",
			)
		}
	}
}

func (validator *documentValidator) primaryLinkage(data *PrimaryData) []Identifier {
	if data == nil || data.kind == primaryDataNull {
		return nil
	}
	if data.kind == primaryDataOne && data.one != nil {
		return relationshipIdentifiers(data.one.Relationships)
	}

	var identifiers []Identifier
	for _, resource := range data.many {
		identifiers = append(identifiers, relationshipIdentifiers(resource.Relationships)...)
	}

	return identifiers
}

func relationshipIdentifiers(relationships Relationships) []Identifier {
	var identifiers []Identifier
	for _, relationship := range relationships {
		data := relationship.Data
		if data == nil || data.kind == relationshipDataNull {
			continue
		}
		if data.kind == relationshipDataOne && data.one != nil {
			identifiers = append(identifiers, *data.one)
		}
		if data.kind == relationshipDataMany {
			identifiers = append(identifiers, data.many...)
		}
	}

	return identifiers
}

func resourceKey(resourceType, id, lid string) string {
	if id != "" {
		return resourceType + "\x00id\x00" + id
	}

	return resourceType + "\x00lid\x00" + lid
}

func identifierKey(identifier Identifier) string {
	return resourceKey(identifier.Type, identifier.ID, identifier.LID)
}

func validMemberName(name string) bool {
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, "@") {
		name = strings.TrimPrefix(name, "@")
		if name == "" {
			return false
		}
	}

	runes := []rune(name)
	if !globallyAllowed(runes[0]) || !globallyAllowed(runes[len(runes)-1]) {
		return false
	}
	for _, character := range runes[1 : len(runes)-1] {
		if !globallyAllowed(character) && character != '-' && character != '_' && character != ' ' {
			return false
		}
	}

	return true
}

func globallyAllowed(character rune) bool {
	return character >= unicode.MaxASCII ||
		character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}

func validJSONPointer(pointer string) bool {
	if pointer == "" {
		return true
	}
	if !strings.HasPrefix(pointer, "/") {
		return false
	}
	for index := 0; index < len(pointer); index++ {
		if pointer[index] != '~' {
			continue
		}
		if index+1 >= len(pointer) || pointer[index+1] != '0' && pointer[index+1] != '1' {
			return false
		}
		index++
	}

	return true
}

func escapePointerToken(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")

	return strings.ReplaceAll(token, "/", "~1")
}

func (validator *documentValidator) add(path, code, message string) {
	validator.violations = append(validator.violations, Violation{
		Path:    path,
		Code:    code,
		Message: message,
	})
}
