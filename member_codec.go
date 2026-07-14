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
	ResourceMemberScope MemberScope = iota + 1
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

// CodecOptions configures applied extensions and core validation context.
type CodecOptions struct {
	Extensions []ExtensionDefinition
	Validation ValidationOptions
}

// Codec is a strict document codec with explicitly registered extension
// members.
type Codec struct {
	validation ValidationOptions
	members    map[MemberScope]map[string]MemberDefinition
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
			if definition.Scope != ResourceMemberScope {
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

	return codec, nil
}

// Marshal validates and deterministically encodes a registered document.
func (codec *Codec) Marshal(document Document) ([]byte, error) {
	if err := document.ValidateWith(codec.validation); err != nil {
		return nil, err
	}
	if err := validateDocumentMembers(document, codec.members); err != nil {
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

	var primaryMembers []Members
	if raw, exists := root["data"]; exists {
		sanitized, extracted, sanitizeErr := codec.sanitizePrimaryData(raw, "/data")
		if sanitizeErr != nil {
			return Document{}, sanitizeErr
		}
		root["data"] = sanitized
		primaryMembers = extracted
	}
	var includedMembers []Members
	if raw, exists := root["included"]; exists {
		sanitized, extracted, sanitizeErr := codec.sanitizeResourceArray(raw, "/included")
		if sanitizeErr != nil {
			return Document{}, sanitizeErr
		}
		root["included"] = sanitized
		includedMembers = extracted
	}
	sanitized, err := json.Marshal(root)
	if err != nil {
		return Document{}, decodeFailure("", "syntax", "could not normalize document", err)
	}
	document, err := UnmarshalWith(sanitized, codec.validation)
	if err != nil {
		return Document{}, err
	}
	attachPrimaryMembers(document.Data, primaryMembers)
	for index := range document.Included {
		if index < len(includedMembers) {
			document.Included[index].AdditionalMembers = includedMembers[index]
		}
	}

	return document, nil
}

func (codec *Codec) sanitizePrimaryData(
	raw json.RawMessage,
	path string,
) (json.RawMessage, []Members, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return raw, nil, nil
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return codec.sanitizeResourceArray(raw, path)
	}
	sanitized, members, err := codec.sanitizeResource(raw, path)
	return sanitized, []Members{members}, err
}

func (codec *Codec) sanitizeResourceArray(
	raw json.RawMessage,
	path string,
) (json.RawMessage, []Members, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || items == nil {
		return nil, nil, decodeFailure(path, "type", "value must be an array", err)
	}
	members := make([]Members, len(items))
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
) (json.RawMessage, Members, error) {
	object, err := decodeObject(raw, path)
	if err != nil {
		return nil, nil, err
	}
	rules := codec.members[ResourceMemberScope]
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
		value, decodeErr := decodeAdditionalMember(rawValue, path+"/"+escapePointerToken(name))
		if decodeErr != nil {
			return nil, nil, decodeErr
		}
		if validate := rules[name].Validate; validate != nil {
			if validationErr := validate(value); validationErr != nil {
				return nil, nil, memberValueError(path, name, validationErr)
			}
		}
		if members == nil {
			members = make(Members)
		}
		members[name] = value
		delete(object, name)
	}
	sanitized, err := json.Marshal(object)
	return sanitized, members, err
}

func decodeAdditionalMember(raw json.RawMessage, path string) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, decodeFailure(path, "syntax", "invalid extension member", err)
	}
	return stripAtMembers(value), nil
}

func attachPrimaryMembers(data *PrimaryData, members []Members) {
	if data == nil {
		return
	}
	if data.kind == primaryDataOne && data.one != nil && len(members) > 0 {
		data.one.AdditionalMembers = members[0]
	}
	if data.kind == primaryDataMany {
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
	for _, observation := range documentResources(document) {
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
	}
	if len(validator.violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: validator.violations}
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
	for _, name := range names {
		nameJSON, _ := json.Marshal(name)
		valueJSON, err := json.Marshal(members[name])
		if err != nil {
			return nil, err
		}
		buffer.WriteByte(',')
		buffer.Write(nameJSON)
		buffer.WriteByte(':')
		buffer.Write(valueJSON)
	}
	buffer.WriteByte('}')
	return buffer.Bytes(), nil
}
