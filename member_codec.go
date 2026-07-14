package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Members contains registered extension or profile members attached to a
// JSON:API-defined object.
type Members map[string]any

// MemberScope identifies the JSON:API object where a custom member is valid.
type MemberScope uint8

const (
	// TopLevelMemberScope registers a top-level document member.
	TopLevelMemberScope MemberScope = iota + 1
	// ResourceMemberScope registers a resource object member.
	ResourceMemberScope
	// RelationshipMemberScope registers a relationship object member.
	RelationshipMemberScope
	// IdentifierMemberScope registers a resource identifier member.
	IdentifierMemberScope
	// JSONAPIMemberScope registers a JSON:API object member.
	JSONAPIMemberScope
	// ErrorMemberScope registers an error object member.
	ErrorMemberScope
	// ErrorSourceMemberScope registers an error source object member.
	ErrorSourceMemberScope
	// LinkObjectMemberScope registers a link object member.
	LinkObjectMemberScope
)

// MemberDefinition declares one extension member and its optional value
// validator.
type MemberDefinition struct {
	Scope    MemberScope
	Name     string
	Validate func(any) error
}

// ExtensionDefinition declares an applied JSON:API extension.
type ExtensionDefinition struct {
	URI       string
	Namespace string
	Members   []MemberDefinition
}

// ProfileDefinition declares an applied JSON:API profile and its optional
// document-level implementation-semantics validator.
type ProfileDefinition struct {
	URI              string
	ValidateDocument func(Document) error
}

// CodecOptions configures applied extensions and core validation context.
type CodecOptions struct {
	Extensions []ExtensionDefinition
	Profiles   []ProfileDefinition
	Validation ValidationOptions
}

// Codec is a strict document codec with explicitly registered extension
// members.
type Codec struct {
	validation ValidationOptions
	members    map[MemberScope]map[string]MemberDefinition
	profiles   []ProfileDefinition
}

// NewCodec validates all extension definitions before constructing a codec.
func NewCodec(options CodecOptions) (*Codec, error) {
	codec := &Codec{
		validation: options.Validation,
		members:    make(map[MemberScope]map[string]MemberDefinition),
	}
	seenURIs := make(map[string]struct{}, len(options.Extensions))
	seenNamespaces := make(map[string]struct{}, len(options.Extensions))
	for _, extension := range options.Extensions {
		parsed, err := url.Parse(extension.URI)
		if extension.URI == "" || err != nil || !parsed.IsAbs() {
			return nil, fmt.Errorf("extension URI must be absolute: %q", extension.URI)
		}
		if _, exists := seenURIs[extension.URI]; exists {
			return nil, fmt.Errorf("duplicate extension URI: %q", extension.URI)
		}
		seenURIs[extension.URI] = struct{}{}
		if !validExtensionNamespace(extension.Namespace) {
			return nil, fmt.Errorf("invalid extension namespace: %q", extension.Namespace)
		}
		if _, exists := seenNamespaces[extension.Namespace]; exists {
			return nil, fmt.Errorf("duplicate extension namespace: %q", extension.Namespace)
		}
		seenNamespaces[extension.Namespace] = struct{}{}
		for _, definition := range extension.Members {
			if definition.Scope != TopLevelMemberScope &&
				definition.Scope != ResourceMemberScope &&
				definition.Scope != RelationshipMemberScope &&
				definition.Scope != IdentifierMemberScope &&
				definition.Scope != JSONAPIMemberScope &&
				definition.Scope != ErrorMemberScope &&
				definition.Scope != ErrorSourceMemberScope &&
				definition.Scope != LinkObjectMemberScope {
				return nil, fmt.Errorf("unsupported member scope: %d", definition.Scope)
			}
			prefix := extension.Namespace + ":"
			if !strings.HasPrefix(definition.Name, prefix) ||
				!validMemberName(strings.TrimPrefix(definition.Name, prefix)) {
				return nil, fmt.Errorf(
					"extension member %q must use namespace %q",
					definition.Name,
					extension.Namespace,
				)
			}
			if codec.members[definition.Scope] == nil {
				codec.members[definition.Scope] = make(map[string]MemberDefinition)
			}
			if _, exists := codec.members[definition.Scope][definition.Name]; exists {
				return nil, fmt.Errorf("duplicate registered member: %q", definition.Name)
			}
			codec.members[definition.Scope][definition.Name] = definition
		}
	}
	seenProfiles := make(map[string]struct{}, len(options.Profiles))
	for _, profile := range options.Profiles {
		parsed, err := url.Parse(profile.URI)
		if profile.URI == "" || err != nil || !parsed.IsAbs() {
			return nil, fmt.Errorf("profile URI must be absolute: %q", profile.URI)
		}
		if _, exists := seenProfiles[profile.URI]; exists {
			return nil, fmt.Errorf("duplicate profile URI: %q", profile.URI)
		}
		seenProfiles[profile.URI] = struct{}{}
		codec.profiles = append(codec.profiles, profile)
	}

	return codec, nil
}

// Marshal validates and deterministically encodes a registered document.
func (codec *Codec) Marshal(document Document) ([]byte, error) {
	if err := validateDocumentMembers(document, codec.members); err != nil {
		return nil, err
	}
	if err := document.ValidateWith(codec.validation); err != nil {
		return nil, err
	}
	if err := codec.validateProfiles(document); err != nil {
		return nil, err
	}

	return json.Marshal(document)
}

// Unmarshal strictly decodes registered extension members and the core
// document.
func (codec *Codec) Unmarshal(payload []byte) (Document, error) {
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
	topMembers, err := codec.extractMembers(root, TopLevelMemberScope, "")
	if err != nil {
		return Document{}, err
	}
	var jsonapiMembers Members
	if raw, exists := root["jsonapi"]; exists {
		sanitized, extracted, sanitizeErr := codec.sanitizeObject(
			raw,
			JSONAPIMemberScope,
			"/jsonapi",
		)
		if sanitizeErr != nil {
			return Document{}, sanitizeErr
		}
		root["jsonapi"] = sanitized
		jsonapiMembers = extracted
	}
	var documentLinks map[string]linkMemberState
	if raw, exists := root["links"]; exists {
		sanitized, extracted, sanitizeErr := codec.sanitizeLinks(raw, "/links")
		if sanitizeErr != nil {
			return Document{}, sanitizeErr
		}
		root["links"] = sanitized
		documentLinks = extracted
	}

	var primaryMembers []resourceMemberState
	if raw, exists := root["data"]; exists {
		sanitized, extracted, sanitizeErr := codec.sanitizePrimaryData(raw, "/data")
		if sanitizeErr != nil {
			return Document{}, sanitizeErr
		}
		root["data"] = sanitized
		primaryMembers = extracted
	}
	var includedMembers []resourceMemberState
	if raw, exists := root["included"]; exists {
		sanitized, extracted, sanitizeErr := codec.sanitizeResourceArray(raw, "/included")
		if sanitizeErr != nil {
			return Document{}, sanitizeErr
		}
		root["included"] = sanitized
		includedMembers = extracted
	}
	var errorMembers []errorMemberState
	if raw, exists := root["errors"]; exists {
		sanitized, extracted, sanitizeErr := codec.sanitizeErrors(raw, "/errors")
		if sanitizeErr != nil {
			return Document{}, sanitizeErr
		}
		root["errors"] = sanitized
		errorMembers = extracted
	}
	// root contains only RawMessages from the already validated payload.
	sanitized, _ := json.Marshal(root)
	document, err := decodeDocument(sanitized)
	if err != nil {
		return Document{}, err
	}
	document.AdditionalMembers = topMembers
	if document.JSONAPI != nil {
		document.JSONAPI.AdditionalMembers = jsonapiMembers
	}
	attachLinkMembers(document.Links, documentLinks)
	attachPrimaryMembers(document.Data, primaryMembers)
	for index := range document.Included {
		if index < len(includedMembers) {
			attachResourceMembers(&document.Included[index], includedMembers[index])
		}
	}
	for index := range document.Errors {
		if index < len(errorMembers) {
			document.Errors[index].AdditionalMembers = errorMembers[index].members
			attachLinkMembers(document.Errors[index].Links, errorMembers[index].links)
			if document.Errors[index].Source != nil {
				document.Errors[index].Source.AdditionalMembers = errorMembers[index].source
			}
		}
	}
	if err := document.ValidateWith(codec.validation); err != nil {
		return Document{}, err
	}
	if err := codec.validateProfiles(document); err != nil {
		return Document{}, err
	}

	return document, nil
}

func (codec *Codec) validateProfiles(document Document) error {
	for _, profile := range codec.profiles {
		if profile.ValidateDocument == nil {
			continue
		}
		if err := profile.ValidateDocument(document); err != nil {
			return err
		}
	}
	return nil
}

func (codec *Codec) sanitizeErrors(
	raw json.RawMessage,
	path string,
) (json.RawMessage, []errorMemberState, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || items == nil {
		return nil, nil, decodeFailure(path, "type", "errors must be an array", err)
	}
	members := make([]errorMemberState, len(items))
	for index, item := range items {
		itemPath := path + "/" + fmt.Sprintf("%d", index)
		object, err := decodeObject(item, itemPath)
		if err != nil {
			return nil, nil, err
		}
		extracted, err := codec.extractMembers(object, ErrorMemberScope, itemPath)
		if err != nil {
			return nil, nil, err
		}
		state := errorMemberState{members: extracted}
		if rawLinks, exists := object["links"]; exists {
			sanitizedLinks, links, sanitizeErr := codec.sanitizeLinks(
				rawLinks,
				itemPath+"/links",
			)
			if sanitizeErr != nil {
				return nil, nil, sanitizeErr
			}
			object["links"] = sanitizedLinks
			state.links = links
		}
		if rawSource, exists := object["source"]; exists {
			sanitizedSource, sourceMembers, sanitizeErr := codec.sanitizeObject(
				rawSource,
				ErrorSourceMemberScope,
				itemPath+"/source",
			)
			if sanitizeErr != nil {
				return nil, nil, sanitizeErr
			}
			object["source"] = sanitizedSource
			state.source = sourceMembers
		}
		sanitized, _ := json.Marshal(object)
		items[index] = sanitized
		members[index] = state
	}
	sanitized, err := json.Marshal(items)
	return sanitized, members, err
}

type errorMemberState struct {
	members Members
	source  Members
	links   map[string]linkMemberState
}

type linkMemberState struct {
	members     Members
	describedBy *linkMemberState
}

func (codec *Codec) sanitizeLinks(
	raw json.RawMessage,
	path string,
) (json.RawMessage, map[string]linkMemberState, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, nil, err
	}
	states := make(map[string]linkMemberState)
	for name, rawLink := range object {
		trimmed := bytes.TrimSpace(rawLink)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			continue
		}
		linkPath := path + "/" + escapePointerToken(name)
		sanitized, state, sanitizeErr := codec.sanitizeLinkObject(rawLink, linkPath)
		if sanitizeErr != nil {
			return nil, nil, sanitizeErr
		}
		object[name] = sanitized
		if len(state.members) > 0 || state.describedBy != nil {
			states[name] = state
		}
	}
	sanitized, err := json.Marshal(object)
	return sanitized, states, err
}

func (codec *Codec) sanitizeLinkObject(
	raw json.RawMessage,
	path string,
) (json.RawMessage, linkMemberState, error) {
	// Callers only pass object-shaped RawMessages from a valid document.
	object, _ := decodeObject(raw, path)
	members, err := codec.extractMembers(object, LinkObjectMemberScope, path)
	if err != nil {
		return nil, linkMemberState{}, err
	}
	state := linkMemberState{members: members}
	if rawDescribedBy, exists := object["describedby"]; exists {
		trimmed := bytes.TrimSpace(rawDescribedBy)
		if len(trimmed) > 0 && trimmed[0] == '{' {
			sanitized, nested, sanitizeErr := codec.sanitizeLinkObject(
				rawDescribedBy,
				path+"/describedby",
			)
			if sanitizeErr != nil {
				return nil, linkMemberState{}, sanitizeErr
			}
			object["describedby"] = sanitized
			state.describedBy = &nested
		}
	}
	sanitized, err := json.Marshal(object)
	return sanitized, state, err
}

func (codec *Codec) sanitizeObject(
	raw json.RawMessage,
	scope MemberScope,
	path string,
) (json.RawMessage, Members, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, nil, err
	}
	members, err := codec.extractMembers(object, scope, path)
	if err != nil {
		return nil, nil, err
	}
	sanitized, err := json.Marshal(object)
	return sanitized, members, err
}

func (codec *Codec) sanitizePrimaryData(
	raw json.RawMessage,
	path string,
) (json.RawMessage, []resourceMemberState, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return raw, nil, nil
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return codec.sanitizeResourceArray(raw, path)
	}
	sanitized, members, err := codec.sanitizeResource(raw, path)
	return sanitized, []resourceMemberState{members}, err
}

func (codec *Codec) sanitizeResourceArray(
	raw json.RawMessage,
	path string,
) (json.RawMessage, []resourceMemberState, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || items == nil {
		return nil, nil, decodeFailure(path, "type", "value must be an array", err)
	}
	members := make([]resourceMemberState, len(items))
	for index, item := range items {
		sanitized, extracted, err := codec.sanitizeResource(
			item,
			path+"/"+fmt.Sprintf("%d", index),
		)
		if err != nil {
			return nil, nil, err
		}
		items[index] = sanitized
		members[index] = extracted
	}
	sanitized, err := json.Marshal(items)
	return sanitized, members, err
}

func (codec *Codec) sanitizeResource(
	raw json.RawMessage,
	path string,
) (json.RawMessage, resourceMemberState, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, resourceMemberState{}, err
	}
	members, err := codec.extractMembers(object, ResourceMemberScope, path)
	if err != nil {
		return nil, resourceMemberState{}, err
	}
	state := resourceMemberState{members: members}
	if rawLinks, exists := object["links"]; exists {
		sanitized, links, sanitizeErr := codec.sanitizeLinks(rawLinks, path+"/links")
		if sanitizeErr != nil {
			return nil, resourceMemberState{}, sanitizeErr
		}
		object["links"] = sanitized
		state.links = links
	}
	if rawRelationships, exists := object["relationships"]; exists {
		sanitized, relationships, sanitizeErr := codec.sanitizeRelationships(
			rawRelationships,
			path+"/relationships",
		)
		if sanitizeErr != nil {
			return nil, resourceMemberState{}, sanitizeErr
		}
		object["relationships"] = sanitized
		state.relationships = relationships
	}
	sanitized, err := json.Marshal(object)
	return sanitized, state, err
}

type resourceMemberState struct {
	members       Members
	links         map[string]linkMemberState
	relationships map[string]relationshipMemberState
}

type relationshipMemberState struct {
	members     Members
	links       map[string]linkMemberState
	identifiers []Members
}

func (codec *Codec) sanitizeRelationships(
	raw json.RawMessage,
	path string,
) (json.RawMessage, map[string]relationshipMemberState, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, nil, err
	}
	states := make(map[string]relationshipMemberState)
	for name, rawRelationship := range object {
		relationshipPath := path + "/" + escapePointerToken(name)
		relationship, decodeErr := decodeObject(rawRelationship, relationshipPath)
		if decodeErr != nil {
			return nil, nil, decodeErr
		}
		members, extractErr := codec.extractMembers(
			relationship,
			RelationshipMemberScope,
			relationshipPath,
		)
		if extractErr != nil {
			return nil, nil, extractErr
		}
		state := relationshipMemberState{members: members}
		if rawLinks, exists := relationship["links"]; exists {
			sanitizedLinks, links, sanitizeErr := codec.sanitizeLinks(
				rawLinks,
				relationshipPath+"/links",
			)
			if sanitizeErr != nil {
				return nil, nil, sanitizeErr
			}
			relationship["links"] = sanitizedLinks
			state.links = links
		}
		if rawData, exists := relationship["data"]; exists {
			sanitizedData, identifiers, sanitizeErr := codec.sanitizeIdentifierData(
				rawData,
				relationshipPath+"/data",
			)
			if sanitizeErr != nil {
				return nil, nil, sanitizeErr
			}
			relationship["data"] = sanitizedData
			state.identifiers = identifiers
		}
		sanitized, _ := json.Marshal(relationship)
		object[name] = sanitized
		if len(state.members) > 0 || len(state.links) > 0 || len(state.identifiers) > 0 {
			states[name] = state
		}
	}
	sanitized, err := json.Marshal(object)
	return sanitized, states, err
}

func (codec *Codec) sanitizeIdentifierData(
	raw json.RawMessage,
	path string,
) (json.RawMessage, []Members, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return raw, nil, nil
	}
	if trimmed[0] == '{' {
		sanitized, members, err := codec.sanitizeIdentifier(raw, path)
		return sanitized, []Members{members}, err
	}
	if trimmed[0] != '[' {
		return raw, nil, nil
	}
	var items []json.RawMessage
	_ = json.Unmarshal(raw, &items)
	states := make([]Members, len(items))
	for index, item := range items {
		sanitized, members, err := codec.sanitizeIdentifier(
			item,
			path+"/"+fmt.Sprintf("%d", index),
		)
		if err != nil {
			return nil, nil, err
		}
		items[index] = sanitized
		states[index] = members
	}
	sanitized, err := json.Marshal(items)
	return sanitized, states, err
}

func (codec *Codec) sanitizeIdentifier(
	raw json.RawMessage,
	path string,
) (json.RawMessage, Members, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, nil, err
	}
	members, err := codec.extractMembers(object, IdentifierMemberScope, path)
	if err != nil {
		return nil, nil, err
	}
	sanitized, err := json.Marshal(object)
	return sanitized, members, err
}

func (codec *Codec) extractMembers(
	object map[string]json.RawMessage,
	scope MemberScope,
	path string,
) (Members, error) {
	rules := codec.members[scope]
	names := make([]string, 0, len(rules))
	for name := range rules {
		names = append(names, name)
	}
	sort.Strings(names)
	var members Members
	for _, name := range names {
		rawValue, exists := object[name]
		if !exists {
			continue
		}
		value := stripAtMembers(decodeValidValue(rawValue))
		if validate := rules[name].Validate; validate != nil {
			if validationErr := validate(value); validationErr != nil {
				return nil, memberValueError(path, name, validationErr)
			}
		}
		if members == nil {
			members = make(Members)
		}
		members[name] = value
		delete(object, name)
	}
	return members, nil
}

func attachPrimaryMembers(data *PrimaryData, members []resourceMemberState) {
	if data == nil {
		return
	}
	if data.kind == primaryDataOne && data.one != nil && len(members) > 0 {
		attachResourceMembers(data.one, members[0])
	}
	if data.kind == primaryDataMany {
		for index := range data.many {
			if index < len(members) {
				attachResourceMembers(&data.many[index], members[index])
			}
		}
	}
}

func attachResourceMembers(resource *ResourceObject, state resourceMemberState) {
	resource.AdditionalMembers = state.members
	attachLinkMembers(resource.Links, state.links)
	for name, relationshipState := range state.relationships {
		relationship, exists := resource.Relationships[name]
		if !exists {
			continue
		}
		relationship.AdditionalMembers = relationshipState.members
		attachLinkMembers(relationship.Links, relationshipState.links)
		attachIdentifierMembers(relationship.Data, relationshipState.identifiers)
		resource.Relationships[name] = relationship
	}
}

func attachLinkMembers(links Links, states map[string]linkMemberState) {
	for name, state := range states {
		link, exists := links[name]
		if !exists {
			continue
		}
		link.additionalMembers = state.members
		if link.describedBy != nil && state.describedBy != nil {
			attachLinkState(link.describedBy, *state.describedBy)
		}
		links[name] = link
	}
}

func attachLinkState(link *Link, state linkMemberState) {
	link.additionalMembers = state.members
	if link.describedBy != nil && state.describedBy != nil {
		attachLinkState(link.describedBy, *state.describedBy)
	}
}

func attachIdentifierMembers(data *RelationshipData, members []Members) {
	if data == nil {
		return
	}
	if data.kind == relationshipDataOne && data.one != nil && len(members) > 0 {
		data.one.AdditionalMembers = members[0]
	}
	if data.kind == relationshipDataMany {
		for index := range data.many {
			if index < len(members) {
				data.many[index].AdditionalMembers = members[index]
			}
		}
	}
}

func validateDocumentMembers(
	document Document,
	registry map[MemberScope]map[string]MemberDefinition,
) error {
	validator := documentValidator{}
	validateScopedMembers(
		&validator,
		document.AdditionalMembers,
		registry[TopLevelMemberScope],
		"",
	)
	if document.JSONAPI != nil {
		validateScopedMembers(
			&validator,
			document.JSONAPI.AdditionalMembers,
			registry[JSONAPIMemberScope],
			"/jsonapi",
		)
	}
	validateLinkDocumentMembers(
		&validator,
		document.Links,
		"/links",
		registry[LinkObjectMemberScope],
	)
	for index, apiError := range document.Errors {
		path := "/errors/" + fmt.Sprintf("%d", index)
		validateScopedMembers(
			&validator,
			apiError.AdditionalMembers,
			registry[ErrorMemberScope],
			path,
		)
		validateLinkDocumentMembers(
			&validator,
			apiError.Links,
			path+"/links",
			registry[LinkObjectMemberScope],
		)
		if apiError.Source != nil {
			validateScopedMembers(
				&validator,
				apiError.Source.AdditionalMembers,
				registry[ErrorSourceMemberScope],
				path+"/source",
			)
		}
	}
	for _, observation := range documentResources(document) {
		validateLinkDocumentMembers(
			&validator,
			observation.resource.Links,
			observation.path+"/links",
			registry[LinkObjectMemberScope],
		)
		for name, value := range observation.resource.AdditionalMembers {
			definition, exists := registry[ResourceMemberScope][name]
			if !exists {
				validator.add(
					observation.path+"/"+escapePointerToken(name),
					"unregistered-member",
					"member is not registered for this object scope",
				)
				continue
			}
			if definition.Validate != nil {
				if err := definition.Validate(value); err != nil {
					validator.violations = append(
						validator.violations,
						memberValueViolation(observation.path, name, err),
					)
				}
			}
		}
		for name, relationship := range observation.resource.Relationships {
			path := observation.path + "/relationships/" + escapePointerToken(name)
			validateLinkDocumentMembers(
				&validator,
				relationship.Links,
				path+"/links",
				registry[LinkObjectMemberScope],
			)
			for memberName, value := range relationship.AdditionalMembers {
				definition, exists := registry[RelationshipMemberScope][memberName]
				if !exists {
					validator.add(
						path+"/"+escapePointerToken(memberName),
						"unregistered-member",
						"member is not registered for this object scope",
					)
					continue
				}
				if definition.Validate != nil {
					if err := definition.Validate(value); err != nil {
						validator.violations = append(
							validator.violations,
							memberValueViolation(path, memberName, err),
						)
					}
				}
			}
			validateIdentifierDocumentMembers(
				&validator,
				relationship.Data,
				path+"/data",
				registry[IdentifierMemberScope],
			)
		}
	}
	if len(validator.violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: validator.violations}
}

func validateLinkDocumentMembers(
	validator *documentValidator,
	links Links,
	path string,
	registry map[string]MemberDefinition,
) {
	for name, link := range links {
		validateLinkStateMembers(
			validator,
			link,
			path+"/"+escapePointerToken(name),
			registry,
		)
	}
}

func validateLinkStateMembers(
	validator *documentValidator,
	link Link,
	path string,
	registry map[string]MemberDefinition,
) {
	validateScopedMembers(validator, link.additionalMembers, registry, path)
	if link.describedBy != nil {
		validateLinkStateMembers(validator, *link.describedBy, path+"/describedby", registry)
	}
}

func validateIdentifierDocumentMembers(
	validator *documentValidator,
	data *RelationshipData,
	path string,
	registry map[string]MemberDefinition,
) {
	if data == nil {
		return
	}
	if data.kind == relationshipDataOne && data.one != nil {
		validateScopedMembers(validator, data.one.AdditionalMembers, registry, path)
	}
	if data.kind == relationshipDataMany {
		for index, identifier := range data.many {
			validateScopedMembers(
				validator,
				identifier.AdditionalMembers,
				registry,
				path+"/"+fmt.Sprintf("%d", index),
			)
		}
	}
}

func validateScopedMembers(
	validator *documentValidator,
	members Members,
	registry map[string]MemberDefinition,
	path string,
) {
	for name, value := range members {
		definition, exists := registry[name]
		if !exists {
			validator.add(
				path+"/"+escapePointerToken(name),
				"unregistered-member",
				"member is not registered for this object scope",
			)
			continue
		}
		if definition.Validate != nil {
			if err := definition.Validate(value); err != nil {
				validator.violations = append(
					validator.violations,
					memberValueViolation(path, name, err),
				)
			}
		}
	}
}

func memberValueError(path, name string, err error) error {
	return &ValidationError{Violations: []Violation{
		memberValueViolation(path, name, err),
	}}
}

func memberValueViolation(path, name string, err error) Violation {
	return Violation{
		Path:    path + "/" + escapePointerToken(name),
		Code:    "member-value",
		Message: err.Error(),
	}
}

func marshalObjectWithMembers(core any, members Members) ([]byte, error) {
	payload, err := json.Marshal(core)
	if err != nil || len(members) == 0 {
		return payload, err
	}
	var existing map[string]json.RawMessage
	if err := json.Unmarshal(payload, &existing); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(members))
	for name := range members {
		if _, collision := existing[name]; collision {
			return nil, fmt.Errorf("additional member %q conflicts with a core member", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	buffer := bytes.NewBuffer(make([]byte, 0, len(payload)+len(names)*16))
	buffer.Write(payload[:len(payload)-1])
	hasMembers := len(existing) > 0
	for _, name := range names {
		nameJSON, _ := json.Marshal(name)
		valueJSON, err := json.Marshal(members[name])
		if err != nil {
			return nil, err
		}
		if hasMembers {
			buffer.WriteByte(',')
		}
		buffer.Write(nameJSON)
		buffer.WriteByte(':')
		buffer.Write(valueJSON)
		hasMembers = true
	}
	buffer.WriteByte('}')
	return buffer.Bytes(), nil
}
