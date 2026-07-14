package jsonapi

import (
	"fmt"
	"mime"
	"net/url"
	"regexp"
	"sort"
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

// ValidationContext identifies the protocol boundary at which a document is
// being validated.
type ValidationContext uint8

const (
	// GenericDocument applies context-independent JSON:API document rules.
	GenericDocument ValidationContext = iota
	// Response applies server response identity rules.
	Response
	// CreateRequest applies resource creation request rules.
	CreateRequest
	// UpdateRequest applies resource update request rules.
	UpdateRequest
	// ToOneRelationshipRequest applies to-one relationship mutation rules.
	ToOneRelationshipRequest
	// ToManyRelationshipRequest applies to-many relationship mutation rules.
	ToManyRelationshipRequest
)

// ValidationOptions configures context and optional endpoint identity checks.
type ValidationOptions struct {
	Context      ValidationContext
	ExpectedType string
	ExpectedID   string
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
	return document.ValidateWith(ValidationOptions{})
}

// ValidateWith checks a document using rules for a specific request or
// response boundary.
func (document Document) ValidateWith(options ValidationOptions) error {
	validator := documentValidator{options: options}
	validator.validateDocument(document)
	if len(validator.violations) == 0 {
		return nil
	}

	return &ValidationError{Violations: validator.violations}
}

type documentValidator struct {
	violations []Violation
	options    ValidationOptions
}

type identityRequirement uint8

const (
	identityEither identityRequirement = iota
	identityOptional
	identityID
)

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
	if validator.requestContext() {
		if document.Data == nil {
			validator.add("/data", "required", "request document must contain data")
		}
		if len(document.Errors) > 0 {
			validator.add("/errors", "forbidden", "request document must not contain errors")
		}
	}

	if document.JSONAPI != nil {
		validator.validateJSONAPI(*document.JSONAPI)
	}
	validator.validateLinks(document.Links, "/links")
	switch validator.options.Context {
	case CreateRequest:
		validator.validateResourceMutation(document.Data, "/data", identityOptional)
	case UpdateRequest:
		validator.validateResourceMutation(document.Data, "/data", identityID)
	case ToOneRelationshipRequest:
		validator.validateRelationshipPrimaryData(document.Data, "/data", false)
	case ToManyRelationshipRequest:
		validator.validateRelationshipPrimaryData(document.Data, "/data", true)
	case Response:
		validator.validatePrimaryData(document.Data, "/data", identityID, false, identityID)
	default:
		validator.validatePrimaryData(document.Data, "/data", identityEither, false, identityEither)
	}
	validator.validateIncluded(document, validator.includedIdentity())
	validator.validateDocumentIdentity(document)
	for index, apiError := range document.Errors {
		validator.validateError(apiError, "/errors/"+strconv.Itoa(index))
	}
}

func (validator *documentValidator) requestContext() bool {
	return validator.options.Context == CreateRequest ||
		validator.options.Context == UpdateRequest ||
		validator.options.Context == ToOneRelationshipRequest ||
		validator.options.Context == ToManyRelationshipRequest
}

func (validator *documentValidator) includedIdentity() identityRequirement {
	if validator.options.Context == Response {
		return identityID
	}

	return identityEither
}

func (validator *documentValidator) validateJSONAPI(object JSONAPI) {
	validator.validateURIList(object.Ext, "/jsonapi/ext")
	validator.validateURIList(object.Profile, "/jsonapi/profile")
}

func (validator *documentValidator) validateURIList(values []string, path string) {
	seen := make(map[string]int, len(values))
	for index, value := range values {
		itemPath := path + "/" + strconv.Itoa(index)
		parsed, err := url.Parse(value)
		if value == "" || err != nil || !parsed.IsAbs() {
			validator.add(itemPath, "uri", "value must be an absolute URI")
		}
		if previous, exists := seen[value]; exists {
			validator.add(
				itemPath,
				"duplicate-uri",
				fmt.Sprintf("URI duplicates item at index %d", previous),
			)
		} else {
			seen[value] = index
		}
	}
}

func (validator *documentValidator) validatePrimaryData(
	data *PrimaryData,
	path string,
	identity identityRequirement,
	requireRelationshipData bool,
	identifierIdentity identityRequirement,
) {
	if data == nil || data.kind == primaryDataNull {
		return
	}
	if data.kind == primaryDataOne {
		if data.one == nil {
			validator.add(path, "required", "single primary data must contain a resource")
			return
		}
		validator.validateResource(*data.one, path, identity, requireRelationshipData, identifierIdentity)
		return
	}
	if data.kind != primaryDataMany {
		validator.add(path, "shape", "primary data must be null, a resource, or an array")
		return
	}
	for index, resource := range data.many {
		validator.validateResource(
			resource,
			path+"/"+strconv.Itoa(index),
			identity,
			requireRelationshipData,
			identifierIdentity,
		)
	}
}

func (validator *documentValidator) validateResource(
	resource ResourceObject,
	path string,
	identity identityRequirement,
	requireRelationshipData bool,
	identifierIdentity identityRequirement,
) {
	if resource.Type == "" {
		validator.add(path+"/type", "required", "resource type is required")
	} else if !validMemberName(resource.Type) {
		validator.add(path+"/type", "member-name", "resource type must be a valid member name")
	}
	if identity == identityID && resource.ID == "" {
		validator.add(path+"/id", "required", "resource id is required")
	} else if identity == identityEither && resource.ID == "" && resource.LID == "" {
		validator.add(path+"/id", "required", "resource id or lid is required")
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
		validator.validateRelationship(
			relationship,
			fieldPath,
			requireRelationshipData,
			identifierIdentity,
		)
	}
	validator.validateLinks(resource.Links, path+"/links")
}

func (validator *documentValidator) validateRelationship(
	relationship Relationship,
	path string,
	requireData bool,
	identifierIdentity identityRequirement,
) {
	if relationship.Links == nil && relationship.Data == nil && relationship.Meta == nil {
		validator.add(path, "required", "relationship must contain links, data, or meta")
	}
	if requireData && relationship.Data == nil {
		validator.add(path+"/data", "required", "resource mutation relationships require data")
	}
	validator.validateLinks(relationship.Links, path+"/links")
	validator.validateRelationshipData(relationship.Data, path+"/data", identifierIdentity)
}

func (validator *documentValidator) validateRelationshipData(
	data *RelationshipData,
	path string,
	identity identityRequirement,
) {
	if data == nil || data.kind == relationshipDataNull {
		return
	}
	if data.kind == relationshipDataOne {
		if data.one == nil {
			validator.add(path, "required", "to-one linkage must contain an identifier")
			return
		}
		validator.validateIdentifier(*data.one, path, identity)
		return
	}
	if data.kind != relationshipDataMany {
		validator.add(path, "shape", "linkage must be null, an identifier, or an array")
		return
	}
	for index, identifier := range data.many {
		validator.validateIdentifier(identifier, path+"/"+strconv.Itoa(index), identity)
	}
}

func (validator *documentValidator) validateIdentifier(
	identifier Identifier,
	path string,
	identity identityRequirement,
) {
	if identifier.Type == "" {
		validator.add(path+"/type", "required", "resource identifier type is required")
	} else if !validMemberName(identifier.Type) {
		validator.add(path+"/type", "member-name", "resource identifier type must be a valid member name")
	}
	if identity == identityID && identifier.ID == "" {
		validator.add(path+"/id", "required", "resource identifier id is required")
	} else if identity == identityEither && identifier.ID == "" && identifier.LID == "" {
		validator.add(path+"/id", "required", "resource identifier requires id or lid")
	}
}

func (validator *documentValidator) validateResourceMutation(
	data *PrimaryData,
	path string,
	identity identityRequirement,
) {
	if data == nil {
		return
	}
	if data.kind != primaryDataOne || data.one == nil {
		validator.add(path, "shape", "resource mutation data must be one resource object")
		return
	}
	resource := *data.one
	validator.validateResource(resource, path, identity, true, identityEither)
	validator.validateExpectedIdentity(resource, path)
}

func (validator *documentValidator) validateExpectedIdentity(resource ResourceObject, path string) {
	if validator.options.ExpectedType != "" && resource.Type != "" &&
		resource.Type != validator.options.ExpectedType {
		validator.add(path+"/type", "endpoint-mismatch", "resource type does not match endpoint")
	}
	if validator.options.ExpectedID != "" && resource.ID != "" &&
		resource.ID != validator.options.ExpectedID {
		validator.add(path+"/id", "endpoint-mismatch", "resource id does not match endpoint")
	}
}

func (validator *documentValidator) validateRelationshipPrimaryData(
	data *PrimaryData,
	path string,
	many bool,
) {
	if data == nil {
		return
	}
	if !many && data.kind == primaryDataNull {
		return
	}
	if !many && data.kind == primaryDataOne && data.one != nil {
		validator.validatePrimaryIdentifier(*data.one, path)
		return
	}
	if many && data.kind == primaryDataMany {
		for index, resource := range data.many {
			validator.validatePrimaryIdentifier(resource, path+"/"+strconv.Itoa(index))
		}
		return
	}
	validator.add(path, "shape", "relationship data has the wrong to-one or to-many shape")
}

func (validator *documentValidator) validatePrimaryIdentifier(resource ResourceObject, path string) {
	validator.validateIdentifier(Identifier{
		Type: resource.Type,
		ID:   resource.ID,
		LID:  resource.LID,
		Meta: resource.Meta,
	}, path, identityID)
	if resource.Attributes != nil {
		validator.add(path+"/attributes", "forbidden", "resource identifier must not contain attributes")
	}
	if resource.Relationships != nil {
		validator.add(path+"/relationships", "forbidden", "resource identifier must not contain relationships")
	}
	if resource.Links != nil {
		validator.add(path+"/links", "forbidden", "resource identifier must not contain links")
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

func (validator *documentValidator) validateIncluded(
	document Document,
	identity identityRequirement,
) {
	if document.Included == nil {
		return
	}

	included := make(map[string]int, len(document.Included))
	for index, resource := range document.Included {
		path := "/included/" + strconv.Itoa(index)
		validator.validateResource(resource, path, identity, false, identity)
		key := resourceKey(resource.Type, resource.ID, resource.LID)
		if _, exists := included[key]; !exists {
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

type resourceObservation struct {
	resource ResourceObject
	path     string
}

type identityObservation struct {
	resourceType string
	id           string
	lid          string
	path         string
}

func (validator *documentValidator) validateDocumentIdentity(document Document) {
	resources := documentResources(document)
	canonical := make(map[string]string, len(resources))
	var identities []identityObservation
	for _, observation := range resources {
		resource := observation.resource
		if resource.Type != "" && (resource.ID != "" || resource.LID != "") {
			key := resourceKey(resource.Type, resource.ID, resource.LID)
			if previous, exists := canonical[key]; exists {
				validator.add(
					observation.path,
					"duplicate-resource",
					"resource duplicates canonical object at "+previous,
				)
			} else {
				canonical[key] = observation.path
			}
		}
		identities = append(identities, identityObservation{
			resourceType: resource.Type,
			id:           resource.ID,
			lid:          resource.LID,
			path:         observation.path,
		})
		identities = append(identities, relationshipIdentityObservations(
			resource.Relationships,
			observation.path+"/relationships",
		)...)
	}
	validator.validateLocalIdentities(identities)
}

func documentResources(document Document) []resourceObservation {
	var resources []resourceObservation
	if document.Data != nil {
		switch document.Data.kind {
		case primaryDataOne:
			if document.Data.one != nil {
				resources = append(resources, resourceObservation{*document.Data.one, "/data"})
			}
		case primaryDataMany:
			for index, resource := range document.Data.many {
				resources = append(resources, resourceObservation{
					resource: resource,
					path:     "/data/" + strconv.Itoa(index),
				})
			}
		}
	}
	for index, resource := range document.Included {
		resources = append(resources, resourceObservation{
			resource: resource,
			path:     "/included/" + strconv.Itoa(index),
		})
	}

	return resources
}

func relationshipIdentityObservations(
	relationships Relationships,
	path string,
) []identityObservation {
	names := make([]string, 0, len(relationships))
	for name := range relationships {
		names = append(names, name)
	}
	sort.Strings(names)
	var observations []identityObservation
	for _, name := range names {
		data := relationships[name].Data
		if data == nil || data.kind == relationshipDataNull {
			continue
		}
		dataPath := path + "/" + escapePointerToken(name) + "/data"
		if data.kind == relationshipDataOne && data.one != nil {
			observations = append(observations, identityFromIdentifier(*data.one, dataPath))
		}
		if data.kind == relationshipDataMany {
			for index, identifier := range data.many {
				observations = append(
					observations,
					identityFromIdentifier(identifier, dataPath+"/"+strconv.Itoa(index)),
				)
			}
		}
	}

	return observations
}

func identityFromIdentifier(identifier Identifier, path string) identityObservation {
	return identityObservation{
		resourceType: identifier.Type,
		id:           identifier.ID,
		lid:          identifier.LID,
		path:         path,
	}
}

func (validator *documentValidator) validateLocalIdentities(observations []identityObservation) {
	byID := make(map[string]identityObservation)
	byLID := make(map[string]identityObservation)
	for _, observation := range observations {
		if observation.resourceType == "" || observation.lid == "" {
			continue
		}
		if observation.id != "" {
			idKey := observation.resourceType + "\x00" + observation.id
			if previous, exists := byID[idKey]; exists && previous.lid != observation.lid {
				validator.add(
					observation.path+"/lid",
					"local-identity",
					"lid differs from another representation of this resource",
				)
			} else {
				byID[idKey] = observation
			}
			lidKey := observation.resourceType + "\x00" + observation.lid
			if previous, exists := byLID[lidKey]; exists && previous.id != observation.id {
				validator.add(
					observation.path+"/lid",
					"local-identity",
					"lid identifies a different resource id elsewhere in the document",
				)
			} else {
				byLID[lidKey] = observation
			}
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
