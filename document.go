// Package jsonapi provides explicit types for building and validating
// JSON:API 1.1 documents.
package jsonapi

import "encoding/json"

// Attributes contains the non-relationship fields of a resource object.
type Attributes map[string]any

// Meta contains non-standard information associated with a JSON:API object.
type Meta map[string]any

// Document is a top-level JSON:API document.
//
// Data is a pointer so callers can distinguish an absent data member from a
// data member whose value is null.
type Document struct {
	JSONAPI           *JSONAPI         `json:"jsonapi,omitempty"`
	Links             Links            `json:"links,omitempty"`
	Data              *PrimaryData     `json:"data,omitempty"`
	Included          []ResourceObject `json:"included,omitempty"`
	Errors            []ErrorObject    `json:"errors,omitempty"`
	Meta              Meta             `json:"meta,omitempty"`
	AdditionalMembers Members          `json:"-"`
}

// MarshalJSON implements json.Marshaler while preserving explicitly empty
// top-level arrays and objects.
func (document Document) MarshalJSON() ([]byte, error) {
	var links *Links
	if document.Links != nil {
		links = &document.Links
	}
	var included *[]ResourceObject
	if document.Included != nil {
		included = &document.Included
	}
	var errorsMember *[]ErrorObject
	if document.Errors != nil {
		errorsMember = &document.Errors
	}
	var meta *Meta
	if document.Meta != nil {
		meta = &document.Meta
	}

	core := struct {
		JSONAPI  *JSONAPI          `json:"jsonapi,omitempty"`
		Links    *Links            `json:"links,omitempty"`
		Data     *PrimaryData      `json:"data,omitempty"`
		Included *[]ResourceObject `json:"included,omitempty"`
		Errors   *[]ErrorObject    `json:"errors,omitempty"`
		Meta     *Meta             `json:"meta,omitempty"`
	}{
		JSONAPI:  document.JSONAPI,
		Links:    links,
		Data:     document.Data,
		Included: included,
		Errors:   errorsMember,
		Meta:     meta,
	}

	return marshalObjectWithMembers(core, document.AdditionalMembers)
}

// JSONAPI describes the JSON:API implementation and applied extensions and
// profiles.
type JSONAPI struct {
	Version string   `json:"version,omitempty"`
	Ext     []string `json:"ext,omitempty"`
	Profile []string `json:"profile,omitempty"`
	Meta    Meta     `json:"meta,omitempty"`
}

// MarshalJSON implements json.Marshaler while preserving explicitly empty
// extension, profile, and meta members.
func (object JSONAPI) MarshalJSON() ([]byte, error) {
	var extensions *[]string
	if object.Ext != nil {
		extensions = &object.Ext
	}
	var profiles *[]string
	if object.Profile != nil {
		profiles = &object.Profile
	}
	var meta *Meta
	if object.Meta != nil {
		meta = &object.Meta
	}

	return json.Marshal(struct {
		Version string    `json:"version,omitempty"`
		Ext     *[]string `json:"ext,omitempty"`
		Profile *[]string `json:"profile,omitempty"`
		Meta    *Meta     `json:"meta,omitempty"`
	}{
		Version: object.Version,
		Ext:     extensions,
		Profile: profiles,
		Meta:    meta,
	})
}

// ResourceObject is a JSON:API resource object.
type ResourceObject struct {
	Type              string        `json:"type"`
	ID                string        `json:"id,omitempty"`
	LID               string        `json:"lid,omitempty"`
	Attributes        Attributes    `json:"attributes,omitempty"`
	Relationships     Relationships `json:"relationships,omitempty"`
	Links             Links         `json:"links,omitempty"`
	Meta              Meta          `json:"meta,omitempty"`
	AdditionalMembers Members       `json:"-"`
}

// MarshalJSON implements json.Marshaler while preserving explicitly empty
// resource containers.
func (resource ResourceObject) MarshalJSON() ([]byte, error) {
	var attributes *Attributes
	if resource.Attributes != nil {
		attributes = &resource.Attributes
	}
	var relationships *Relationships
	if resource.Relationships != nil {
		relationships = &resource.Relationships
	}
	var links *Links
	if resource.Links != nil {
		links = &resource.Links
	}
	var meta *Meta
	if resource.Meta != nil {
		meta = &resource.Meta
	}

	core := struct {
		Type          string         `json:"type"`
		ID            string         `json:"id,omitempty"`
		LID           string         `json:"lid,omitempty"`
		Attributes    *Attributes    `json:"attributes,omitempty"`
		Relationships *Relationships `json:"relationships,omitempty"`
		Links         *Links         `json:"links,omitempty"`
		Meta          *Meta          `json:"meta,omitempty"`
	}{
		Type:          resource.Type,
		ID:            resource.ID,
		LID:           resource.LID,
		Attributes:    attributes,
		Relationships: relationships,
		Links:         links,
		Meta:          meta,
	}

	return marshalObjectWithMembers(core, resource.AdditionalMembers)
}

// Identifier identifies a resource by type and either server or local ID.
type Identifier struct {
	Type              string  `json:"type"`
	ID                string  `json:"id,omitempty"`
	LID               string  `json:"lid,omitempty"`
	Meta              Meta    `json:"meta,omitempty"`
	AdditionalMembers Members `json:"-"`
}

// MarshalJSON implements json.Marshaler while preserving an explicitly empty
// identifier meta object.
func (identifier Identifier) MarshalJSON() ([]byte, error) {
	var meta *Meta
	if identifier.Meta != nil {
		meta = &identifier.Meta
	}

	core := struct {
		Type string `json:"type"`
		ID   string `json:"id,omitempty"`
		LID  string `json:"lid,omitempty"`
		Meta *Meta  `json:"meta,omitempty"`
	}{
		Type: identifier.Type,
		ID:   identifier.ID,
		LID:  identifier.LID,
		Meta: meta,
	}

	return marshalObjectWithMembers(core, identifier.AdditionalMembers)
}

type primaryDataKind uint8

const (
	primaryDataNull primaryDataKind = iota + 1
	primaryDataOne
	primaryDataMany
)

// PrimaryData represents null, one resource, or a collection of resources.
// Construct values with NullData, ResourceData, or ResourceCollection.
type PrimaryData struct {
	kind primaryDataKind
	one  *ResourceObject
	many []ResourceObject
}

// NullData returns a primary data member whose JSON value is null.
func NullData() *PrimaryData {
	return &PrimaryData{kind: primaryDataNull}
}

// ResourceData returns a primary data member containing one resource.
func ResourceData(resource ResourceObject) *PrimaryData {
	return &PrimaryData{kind: primaryDataOne, one: &resource}
}

// ResourceCollection returns a primary data member containing a resource
// collection. With no arguments it serializes as an empty array.
func ResourceCollection(resources ...ResourceObject) *PrimaryData {
	items := make([]ResourceObject, len(resources))
	copy(items, resources)

	return &PrimaryData{kind: primaryDataMany, many: items}
}

// MarshalJSON implements json.Marshaler.
func (data PrimaryData) MarshalJSON() ([]byte, error) {
	switch data.kind {
	case primaryDataOne:
		return json.Marshal(data.one)
	case primaryDataMany:
		return json.Marshal(data.many)
	default:
		return []byte("null"), nil
	}
}

// Relationships maps relationship names to relationship objects.
type Relationships map[string]Relationship

// Relationship is a JSON:API relationship object.
type Relationship struct {
	Links             Links             `json:"links,omitempty"`
	Data              *RelationshipData `json:"data,omitempty"`
	Meta              Meta              `json:"meta,omitempty"`
	AdditionalMembers Members           `json:"-"`
}

// MarshalJSON implements json.Marshaler while preserving explicitly empty
// relationship links and meta objects.
func (relationship Relationship) MarshalJSON() ([]byte, error) {
	var links *Links
	if relationship.Links != nil {
		links = &relationship.Links
	}
	var meta *Meta
	if relationship.Meta != nil {
		meta = &relationship.Meta
	}

	core := struct {
		Links *Links            `json:"links,omitempty"`
		Data  *RelationshipData `json:"data,omitempty"`
		Meta  *Meta             `json:"meta,omitempty"`
	}{
		Links: links,
		Data:  relationship.Data,
		Meta:  meta,
	}

	return marshalObjectWithMembers(core, relationship.AdditionalMembers)
}

type relationshipDataKind uint8

const (
	relationshipDataNull relationshipDataKind = iota + 1
	relationshipDataOne
	relationshipDataMany
)

// RelationshipData represents null, one resource identifier, or a collection
// of resource identifiers.
type RelationshipData struct {
	kind relationshipDataKind
	one  *Identifier
	many []Identifier
}

// NullRelationship returns relationship data whose JSON value is null.
func NullRelationship() *RelationshipData {
	return &RelationshipData{kind: relationshipDataNull}
}

// ToOne returns relationship data containing one resource identifier.
func ToOne(identifier Identifier) *RelationshipData {
	return &RelationshipData{kind: relationshipDataOne, one: &identifier}
}

// ToMany returns relationship data containing a resource identifier
// collection. With no arguments it serializes as an empty array.
func ToMany(identifiers ...Identifier) *RelationshipData {
	items := make([]Identifier, len(identifiers))
	copy(items, identifiers)

	return &RelationshipData{kind: relationshipDataMany, many: items}
}

// MarshalJSON implements json.Marshaler.
func (data RelationshipData) MarshalJSON() ([]byte, error) {
	switch data.kind {
	case relationshipDataOne:
		return json.Marshal(data.one)
	case relationshipDataMany:
		return json.Marshal(data.many)
	default:
		return []byte("null"), nil
	}
}

// Links maps link relation names to links.
type Links map[string]Link

// Link is a string, object, or null JSON:API link value. Construct values with
// URI, ObjectLink, or NullLink.
type Link struct {
	href        string
	rel         string
	describedBy *Link
	title       string
	targetType  string
	hreflang    *LinkHreflang
	meta        Meta
	object      bool
	null        bool
}

// LinkObject contains every member supported by a JSON:API 1.1 link object.
type LinkObject struct {
	Href        string
	Rel         string
	DescribedBy *Link
	Title       string
	Type        string
	Hreflang    *LinkHreflang
	Meta        Meta
}

// LinkHreflang represents the scalar or array form of a link object's
// hreflang member. Construct values with LanguageTag or LanguageTags.
type LinkHreflang struct {
	values []string
	many   bool
}

// URI returns a link represented by a URI string.
func URI(href string) Link {
	return Link{href: href}
}

// ObjectLink returns a link object with an href and optional meta object.
func ObjectLink(href string, meta Meta) Link {
	return LinkFromObject(LinkObject{Href: href, Meta: meta})
}

// LinkFromObject returns a link represented by a JSON:API 1.1 link object.
func LinkFromObject(object LinkObject) Link {
	return Link{
		href:        object.Href,
		rel:         object.Rel,
		describedBy: object.DescribedBy,
		title:       object.Title,
		targetType:  object.Type,
		hreflang:    object.Hreflang,
		meta:        object.Meta,
		object:      true,
	}
}

// LanguageTag returns the scalar form of a link object's hreflang member.
func LanguageTag(tag string) *LinkHreflang {
	return &LinkHreflang{values: []string{tag}}
}

// LanguageTags returns the array form of a link object's hreflang member.
func LanguageTags(tags ...string) *LinkHreflang {
	values := make([]string, len(tags))
	copy(values, tags)

	return &LinkHreflang{values: values, many: true}
}

// NullLink returns a null link.
func NullLink() Link {
	return Link{null: true}
}

// MarshalJSON implements json.Marshaler.
func (link Link) MarshalJSON() ([]byte, error) {
	if link.null {
		return []byte("null"), nil
	}
	if !link.object {
		return json.Marshal(link.href)
	}

	var meta *Meta
	if link.meta != nil {
		meta = &link.meta
	}

	return json.Marshal(struct {
		Href        string `json:"href"`
		Rel         string `json:"rel,omitempty"`
		DescribedBy *Link  `json:"describedby,omitempty"`
		Title       string `json:"title,omitempty"`
		Type        string `json:"type,omitempty"`
		Hreflang    any    `json:"hreflang,omitempty"`
		Meta        *Meta  `json:"meta,omitempty"`
	}{
		Href:        link.href,
		Rel:         link.rel,
		DescribedBy: link.describedBy,
		Title:       link.title,
		Type:        link.targetType,
		Hreflang:    link.hreflangValue(),
		Meta:        meta,
	})
}

func (link Link) hreflangValue() any {
	if link.hreflang == nil {
		return nil
	}
	if link.hreflang.many {
		return link.hreflang.values
	}
	if len(link.hreflang.values) == 0 {
		return ""
	}

	return link.hreflang.values[0]
}

// ErrorObject describes one JSON:API error.
type ErrorObject struct {
	ID     string       `json:"id,omitempty"`
	Links  Links        `json:"links,omitempty"`
	Status string       `json:"status,omitempty"`
	Code   string       `json:"code,omitempty"`
	Title  string       `json:"title,omitempty"`
	Detail string       `json:"detail,omitempty"`
	Source *ErrorSource `json:"source,omitempty"`
	Meta   Meta         `json:"meta,omitempty"`
}

// MarshalJSON implements json.Marshaler while preserving explicitly empty
// error links and meta objects.
func (apiError ErrorObject) MarshalJSON() ([]byte, error) {
	var links *Links
	if apiError.Links != nil {
		links = &apiError.Links
	}
	var meta *Meta
	if apiError.Meta != nil {
		meta = &apiError.Meta
	}

	return json.Marshal(struct {
		ID     string       `json:"id,omitempty"`
		Links  *Links       `json:"links,omitempty"`
		Status string       `json:"status,omitempty"`
		Code   string       `json:"code,omitempty"`
		Title  string       `json:"title,omitempty"`
		Detail string       `json:"detail,omitempty"`
		Source *ErrorSource `json:"source,omitempty"`
		Meta   *Meta        `json:"meta,omitempty"`
	}{
		ID:     apiError.ID,
		Links:  links,
		Status: apiError.Status,
		Code:   apiError.Code,
		Title:  apiError.Title,
		Detail: apiError.Detail,
		Source: apiError.Source,
		Meta:   meta,
	})
}

// ErrorSource identifies the source of an error in a request.
type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
	Header    string `json:"header,omitempty"`
}
