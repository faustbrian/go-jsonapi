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
	JSONAPI  *JSONAPI         `json:"jsonapi,omitempty"`
	Links    Links            `json:"links,omitempty"`
	Data     *PrimaryData     `json:"data,omitempty"`
	Included []ResourceObject `json:"included,omitempty"`
	Errors   []ErrorObject    `json:"errors,omitempty"`
	Meta     Meta             `json:"meta,omitempty"`
}

// JSONAPI describes the JSON:API implementation and applied extensions and
// profiles.
type JSONAPI struct {
	Version string   `json:"version,omitempty"`
	Ext     []string `json:"ext,omitempty"`
	Profile []string `json:"profile,omitempty"`
	Meta    Meta     `json:"meta,omitempty"`
}

// ResourceObject is a JSON:API resource object.
type ResourceObject struct {
	Type          string        `json:"type"`
	ID            string        `json:"id,omitempty"`
	LID           string        `json:"lid,omitempty"`
	Attributes    Attributes    `json:"attributes,omitempty"`
	Relationships Relationships `json:"relationships,omitempty"`
	Links         Links         `json:"links,omitempty"`
	Meta          Meta          `json:"meta,omitempty"`
}

// Identifier identifies a resource by type and either server or local ID.
type Identifier struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	LID  string `json:"lid,omitempty"`
	Meta Meta   `json:"meta,omitempty"`
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
	Links Links             `json:"links,omitempty"`
	Data  *RelationshipData `json:"data,omitempty"`
	Meta  Meta              `json:"meta,omitempty"`
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
	href   string
	meta   Meta
	object bool
	null   bool
}

// URI returns a link represented by a URI string.
func URI(href string) Link {
	return Link{href: href}
}

// ObjectLink returns a link object with an href and optional meta object.
func ObjectLink(href string, meta Meta) Link {
	return Link{href: href, meta: meta, object: true}
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

	return json.Marshal(struct {
		Href string `json:"href"`
		Meta Meta   `json:"meta,omitempty"`
	}{
		Href: link.href,
		Meta: link.meta,
	})
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

// ErrorSource identifies the source of an error in a request.
type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
	Header    string `json:"header,omitempty"`
}
