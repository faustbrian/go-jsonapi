// Package jsonapi implements explicit JSON:API 1.1 document models, checked
// encoding/decoding, query parsing and HTTP media negotiation. It also provides
// separate Atomic Operations and Cursor Pagination helpers. The base authority
// is JSON:API 1.1 at json-api/json-api revision
// 52fc018adcc15358cb0f9d3de119f7e0fbe34ea6; the interpretation register is
// docs/specification-decisions.md, not a claim of complete server behavior.
//
// Start with Marshal/Unmarshal for core documents, NewCodec for explicitly
// registered extensions/profiles, NewQueryParser for query families and
// NewNegotiator for headers. MarshalJSON methods are wire encoders, not the
// checked API: direct encoding/json use bypasses protocol validation. Use
// With methods and data constructors to preserve absence, null and empty values.
// Model copies and With methods are shallow; maps, pointers and nested values
// stay caller-owned and must not be mutated during use.
//
// Decoder budgets bound byte/structural work, not custom JSON methods or
// application callbacks. Those collaborators must be prompt, safe for concurrent
// use where needed, and free of uncoordinated mutation. No package goroutine,
// background I/O or global protocol registry is acquired. Applications retain
// HTTP response writing, URI resolution, authorization, field projection,
// cursor authenticity, storage transactions and external-effect/retry policy.
// The independently resolved peer harness is outside the public module.
package jsonapi

import "encoding/json"

// Attributes contains caller-owned non-relationship resource fields. Values
// may contain mutable containers or custom JSON methods; constructors do not
// deep-copy them and application code owns their resource/disclosure policy.
type Attributes map[string]any

// Meta contains caller-owned non-standard information. Nil means absence;
// non-nil empty metadata preserves object presence. Nested values are borrowed.
type Meta map[string]any

// Document is a top-level JSON:API document.
//
// Data is a pointer so callers can distinguish an absent data member from a
// data member whose value is null.
type Document struct {
	// JSONAPI declares protocol metadata; nil omits it and does not configure a
	// Codec.
	JSONAPI *JSONAPI `json:"jsonapi,omitempty"`
	// Links holds top-level links; nil omits them, a non-nil empty map emits {}.
	Links Links `json:"links,omitempty"`
	// Data is nil when absent; NullData distinguishes an explicit null member.
	Data *PrimaryData `json:"data,omitempty"`
	// Included holds compound resources; nil omits them, an empty non-nil slice
	// emits [].
	Included []ResourceObject `json:"included,omitempty"`
	// Errors holds error objects; nil omits them and even an empty present array
	// conflicts with Data.
	Errors []ErrorObject `json:"errors,omitempty"`
	// Meta holds caller-owned metadata; nil omits it, an empty non-nil map emits
	// {}.
	Meta Meta `json:"meta,omitempty"`
	// AdditionalMembers holds caller-owned custom members; checked Codec encoding
	// requires scope registration (except @-Members).
	AdditionalMembers Members `json:"-"`
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
	// Version is optional; WithVersion preserves presence even for an empty
	// string.
	Version string `json:"version,omitempty"`
	// Ext lists applied extension URIs; nil omits the declaration, not the
	// registry.
	Ext []string `json:"ext,omitempty"`
	// Profile lists applied profile URIs; declarations alone do not activate
	// callbacks.
	Profile []string `json:"profile,omitempty"`
	// Meta holds caller-owned protocol metadata, preserving non-nil empty
	// presence.
	Meta Meta `json:"meta,omitempty"`
	// AdditionalMembers holds caller-owned custom members; checked Codec encoding
	// requires scope registration (except @-Members).
	AdditionalMembers Members `json:"-"`
	versionPresent    bool
}

// WithVersion returns a copy whose version member is present, including when
// value is the empty string.
func (object JSONAPI) WithVersion(value string) JSONAPI {
	object.Version = value
	object.versionPresent = true
	return object
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

	core := struct {
		Version *string   `json:"version,omitempty"`
		Ext     *[]string `json:"ext,omitempty"`
		Profile *[]string `json:"profile,omitempty"`
		Meta    *Meta     `json:"meta,omitempty"`
	}{
		Version: optionalString(object.Version, object.versionPresent),
		Ext:     extensions,
		Profile: profiles,
		Meta:    meta,
	}

	return marshalObjectWithMembers(core, object.AdditionalMembers)
}

// ResourceObject is a JSON:API resource object.
type ResourceObject struct {
	// Type names the required resource type; validation rejects an empty type.
	Type string `json:"type"`
	// ID is server identity; WithID preserves explicit empty-string presence.
	ID string `json:"id,omitempty"`
	// LID is local identity; WithLID preserves presence separately from ID.
	LID string `json:"lid,omitempty"`
	// Attributes holds caller-owned values; these names share the namespace with
	// Relationships.
	Attributes Attributes `json:"attributes,omitempty"`
	// Relationships holds caller-owned linkage and relationship metadata, not
	// fetched resources.
	Relationships Relationships `json:"relationships,omitempty"`
	// Links holds caller-owned resource links, preserving non-nil empty presence.
	Links Links `json:"links,omitempty"`
	// Meta holds caller-owned non-standard resource information.
	Meta Meta `json:"meta,omitempty"`
	// AdditionalMembers holds caller-owned custom members; checked Codec encoding
	// requires scope registration (except @-Members).
	AdditionalMembers Members `json:"-"`
	idPresent         bool
	lidPresent        bool
}

// WithID returns a copy whose id member is present, including when value is
// the empty string.
func (resource ResourceObject) WithID(value string) ResourceObject {
	resource.ID = value
	resource.idPresent = true
	return resource
}

// WithLID returns a copy whose lid member is present, including when value is
// the empty string.
func (resource ResourceObject) WithLID(value string) ResourceObject {
	resource.LID = value
	resource.lidPresent = true
	return resource
}

func (resource ResourceObject) hasID() bool {
	return resource.ID != "" || resource.idPresent
}

func (resource ResourceObject) hasLID() bool {
	return resource.LID != "" || resource.lidPresent
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
		ID            *string        `json:"id,omitempty"`
		LID           *string        `json:"lid,omitempty"`
		Attributes    *Attributes    `json:"attributes,omitempty"`
		Relationships *Relationships `json:"relationships,omitempty"`
		Links         *Links         `json:"links,omitempty"`
		Meta          *Meta          `json:"meta,omitempty"`
	}{
		Type:          resource.Type,
		ID:            optionalString(resource.ID, resource.idPresent),
		LID:           optionalString(resource.LID, resource.lidPresent),
		Attributes:    attributes,
		Relationships: relationships,
		Links:         links,
		Meta:          meta,
	}

	return marshalObjectWithMembers(core, resource.AdditionalMembers)
}

// Identifier identifies a resource by type and either server or local ID.
type Identifier struct {
	// Type names the resource type to which this linkage refers.
	Type string `json:"type"`
	// ID is server identity; WithID retains presence even when empty.
	ID string `json:"id,omitempty"`
	// LID is local identity; context validation determines whether it can satisfy
	// identity.
	LID string `json:"lid,omitempty"`
	// Meta holds caller-owned linkage metadata, preserving non-nil empty
	// presence.
	Meta Meta `json:"meta,omitempty"`
	// AdditionalMembers holds caller-owned custom members; checked Codec encoding
	// requires scope registration (except @-Members).
	AdditionalMembers Members `json:"-"`
	idPresent         bool
	lidPresent        bool
}

// WithID returns a copy whose id member is present, including when value is
// the empty string.
func (identifier Identifier) WithID(value string) Identifier {
	identifier.ID = value
	identifier.idPresent = true
	return identifier
}

// WithLID returns a copy whose lid member is present, including when value is
// the empty string.
func (identifier Identifier) WithLID(value string) Identifier {
	identifier.LID = value
	identifier.lidPresent = true
	return identifier
}

func (identifier Identifier) hasID() bool {
	return identifier.ID != "" || identifier.idPresent
}

func (identifier Identifier) hasLID() bool {
	return identifier.LID != "" || identifier.lidPresent
}

// MarshalJSON implements json.Marshaler while preserving an explicitly empty
// identifier meta object.
func (identifier Identifier) MarshalJSON() ([]byte, error) {
	var meta *Meta
	if identifier.Meta != nil {
		meta = &identifier.Meta
	}

	core := struct {
		Type string  `json:"type"`
		ID   *string `json:"id,omitempty"`
		LID  *string `json:"lid,omitempty"`
		Meta *Meta   `json:"meta,omitempty"`
	}{
		Type: identifier.Type,
		ID:   optionalString(identifier.ID, identifier.idPresent),
		LID:  optionalString(identifier.LID, identifier.lidPresent),
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
// Construct values with NullData, ResourceData, or ResourceCollection. The zero
// value encodes null through MarshalJSON but is not valid checked primary data.
type PrimaryData struct {
	kind primaryDataKind
	one  *ResourceObject
	many []ResourceObject
}

// NullData returns a primary data member whose JSON value is null.
func NullData() *PrimaryData {
	return &PrimaryData{kind: primaryDataNull}
}

// ResourceData copies the outer resource value into primary data. Nested maps,
// slices, pointers and custom values remain shared with the caller.
func ResourceData(resource ResourceObject) *PrimaryData {
	return &PrimaryData{kind: primaryDataOne, one: &resource}
}

// ResourceCollection copies the outer slice and resource values into primary
// data. With no arguments it encodes an empty array. Nested values remain shared
// with the caller, so this is not a deep snapshot.
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

// Relationships maps relationship names to caller-owned relationship objects.
// It shares the field namespace with Attributes; validation does not fetch links.
type Relationships map[string]Relationship

// Relationship is a JSON:API relationship object.
type Relationship struct {
	// Links holds relationship links; qualifying links can satisfy a relationship
	// with absent Data.
	Links Links `json:"links,omitempty"`
	// Data is nil when linkage is absent; use NullRelationship for explicit null
	// linkage.
	Data *RelationshipData `json:"data,omitempty"`
	// Meta can supply the recognized metadata member required by a relationship.
	Meta Meta `json:"meta,omitempty"`
	// AdditionalMembers holds caller-owned custom members; checked Codec encoding
	// requires scope registration (except @-Members).
	AdditionalMembers Members `json:"-"`
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

// ToOne copies an outer resource identifier into relationship data; metadata
// and additional member values remain shared.
func ToOne(identifier Identifier) *RelationshipData {
	return &RelationshipData{kind: relationshipDataOne, one: &identifier}
}

// ToMany copies the outer identifier slice into relationship data. With no
// arguments it encodes an empty array. Identifier metadata and additional values
// remain shared with the caller.
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

// Links maps relation names to caller-owned links. Validation checks shape and
// URI syntax, not authorization, reachability or SSRF policy; no link is fetched.
type Links map[string]Link

// Link is a string, object, null, or registered extension-defined links-object
// member value. Construct values with URI, ObjectLink, NullLink, or
// ExtensionLinkValue.
type Link struct {
	href              string
	rel               string
	describedBy       *Link
	title             string
	targetType        string
	hreflang          *LinkHreflang
	meta              Meta
	additionalMembers Members
	object            bool
	null              bool
	extensionValue    any
	extension         bool
	hrefPresent       bool
	relPresent        bool
	titlePresent      bool
	targetTypePresent bool
}

// LinkObject contains every member supported by a JSON:API 1.1 link object.
type LinkObject struct {
	// Href is a URI-reference; LinkFromObject marks it present even when empty.
	Href string
	// Rel is a registered relation token or absolute extension URI, not a
	// dereferenced endpoint.
	Rel string
	// DescribedBy borrows the link to a resource description; recursive links
	// require bounded structure.
	DescribedBy *Link
	// Title is human-readable link text; WithTitle on Link preserves explicit
	// empty presence.
	Title string
	// Type is the target media type hint, not the resource type.
	Type string
	// Hreflang borrows a scalar or array language hint; LanguageTag and
	// LanguageTags choose the wire form.
	Hreflang *LinkHreflang
	// Meta borrows non-standard link metadata.
	Meta Meta
	// AdditionalMembers holds caller-owned custom members; checked Codec encoding
	// requires scope registration (except @-Members).
	AdditionalMembers Members
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

// LinkFromObject copies the outer link object into a wire representation; nested
// link, language, metadata and additional-member collaborators remain borrowed.
func LinkFromObject(object LinkObject) Link {
	return Link{
		href:              object.Href,
		rel:               object.Rel,
		describedBy:       object.DescribedBy,
		title:             object.Title,
		targetType:        object.Type,
		hreflang:          object.Hreflang,
		meta:              object.Meta,
		additionalMembers: object.AdditionalMembers,
		object:            true,
		hrefPresent:       true,
		relPresent:        object.Rel != "",
		titlePresent:      object.Title != "",
		targetTypePresent: object.Type != "",
	}
}

// WithRel returns a copy whose rel member is present, including when value is
// the empty string.
func (link Link) WithRel(value string) Link {
	link.rel = value
	link.relPresent = true
	return link
}

// WithTitle returns a copy whose title member is present, including when value
// is the empty string.
func (link Link) WithTitle(value string) Link {
	link.title = value
	link.titlePresent = true
	return link
}

// WithType returns a copy whose type member is present, including when value
// is the empty string.
func (link Link) WithType(value string) Link {
	link.targetType = value
	link.targetTypePresent = true
	return link
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

// ExtensionLinkValue returns an opaque value for a member defined by an
// extension in a JSON:API links object. A Codec must register that member at
// LinksObjectMemberScope before the value can be marshaled.
func ExtensionLinkValue(value any) Link {
	return Link{extensionValue: value, extension: true}
}

// ExtensionValue returns the opaque extension-defined links-object member
// value and whether this Link represents one. The value is borrowed, not cloned.
func (link Link) ExtensionValue() (any, bool) {
	return link.extensionValue, link.extension
}

// MarshalJSON implements json.Marshaler.
func (link Link) MarshalJSON() ([]byte, error) {
	if link.extension {
		return json.Marshal(link.extensionValue)
	}
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

	core := struct {
		Href        string  `json:"href"`
		Rel         *string `json:"rel,omitempty"`
		DescribedBy *Link   `json:"describedby,omitempty"`
		Title       *string `json:"title,omitempty"`
		Type        *string `json:"type,omitempty"`
		Hreflang    any     `json:"hreflang,omitempty"`
		Meta        *Meta   `json:"meta,omitempty"`
	}{
		Href:        link.href,
		Rel:         optionalString(link.rel, link.relPresent),
		DescribedBy: link.describedBy,
		Title:       optionalString(link.title, link.titlePresent),
		Type:        optionalString(link.targetType, link.targetTypePresent),
		Hreflang:    link.hreflangValue(),
		Meta:        meta,
	}

	return marshalObjectWithMembers(core, link.additionalMembers)
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
	// ID identifies this occurrence; WithID preserves an explicitly empty member.
	ID string `json:"id,omitempty"`
	// Links holds error links such as about and type, not automatic HTTP
	// redirects.
	Links Links `json:"links,omitempty"`
	// Status is a three-digit HTTP status string when present; WithStatus
	// preserves presence.
	Status string `json:"status,omitempty"`
	// Code is application-specific, not an HTTP status or automatic retry
	// classification.
	Code string `json:"code,omitempty"`
	// Title is a human-readable summary; caller policy owns disclosure and
	// localization.
	Title string `json:"title,omitempty"`
	// Detail is occurrence-specific text; the caller is responsible for safe
	// disclosure.
	Detail string `json:"detail,omitempty"`
	// Source borrows request-source metadata; nil omits it.
	Source *ErrorSource `json:"source,omitempty"`
	// Meta borrows non-standard error metadata.
	Meta Meta `json:"meta,omitempty"`
	// AdditionalMembers holds caller-owned custom members; checked Codec encoding
	// requires scope registration (except @-Members).
	AdditionalMembers Members `json:"-"`
	present           uint8
}

const (
	errorIDPresent uint8 = 1 << iota
	errorStatusPresent
	errorCodePresent
	errorTitlePresent
	errorDetailPresent
)

// WithID returns a copy whose id member is present.
func (apiError ErrorObject) WithID(value string) ErrorObject {
	apiError.ID = value
	apiError.present = apiError.present | errorIDPresent
	return apiError
}

// WithStatus returns a copy whose status member is present.
func (apiError ErrorObject) WithStatus(value string) ErrorObject {
	apiError.Status = value
	apiError.present |= errorStatusPresent
	return apiError
}

// WithCode returns a copy whose code member is present.
func (apiError ErrorObject) WithCode(value string) ErrorObject {
	apiError.Code = value
	apiError.present |= errorCodePresent
	return apiError
}

// WithTitle returns a copy whose title member is present.
func (apiError ErrorObject) WithTitle(value string) ErrorObject {
	apiError.Title = value
	apiError.present |= errorTitlePresent
	return apiError
}

// WithDetail returns a copy whose detail member is present.
func (apiError ErrorObject) WithDetail(value string) ErrorObject {
	apiError.Detail = value
	apiError.present |= errorDetailPresent
	return apiError
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

	core := struct {
		ID     *string      `json:"id,omitempty"`
		Links  *Links       `json:"links,omitempty"`
		Status *string      `json:"status,omitempty"`
		Code   *string      `json:"code,omitempty"`
		Title  *string      `json:"title,omitempty"`
		Detail *string      `json:"detail,omitempty"`
		Source *ErrorSource `json:"source,omitempty"`
		Meta   *Meta        `json:"meta,omitempty"`
	}{
		ID:     optionalString(apiError.ID, apiError.present&errorIDPresent != 0),
		Links:  links,
		Status: optionalString(apiError.Status, apiError.present&errorStatusPresent != 0),
		Code:   optionalString(apiError.Code, apiError.present&errorCodePresent != 0),
		Title:  optionalString(apiError.Title, apiError.present&errorTitlePresent != 0),
		Detail: optionalString(apiError.Detail, apiError.present&errorDetailPresent != 0),
		Source: apiError.Source,
		Meta:   meta,
	}

	return marshalObjectWithMembers(core, apiError.AdditionalMembers)
}

// ErrorSource identifies the source of an error in a request.
type ErrorSource struct {
	// Pointer identifies a request value by JSON Pointer; WithPointer preserves
	// the empty root pointer.
	Pointer string `json:"pointer,omitempty"`
	// Parameter identifies a request query parameter; WithParameter preserves
	// explicit empty presence.
	Parameter string `json:"parameter,omitempty"`
	// Header identifies a request header; WithHeader preserves explicit empty
	// presence.
	Header string `json:"header,omitempty"`
	// AdditionalMembers holds caller-owned custom members; checked Codec encoding
	// requires scope registration (except @-Members).
	AdditionalMembers Members `json:"-"`
	present           uint8
}

const (
	sourcePointerPresent uint8 = 1 << iota
	sourceParameterPresent
	sourceHeaderPresent
)

// WithPointer returns a copy whose pointer member is present.
func (source ErrorSource) WithPointer(value string) ErrorSource {
	source.Pointer = value
	source.present = source.present | sourcePointerPresent
	return source
}

// WithParameter returns a copy whose parameter member is present.
func (source ErrorSource) WithParameter(value string) ErrorSource {
	source.Parameter = value
	source.present |= sourceParameterPresent
	return source
}

// WithHeader returns a copy whose header member is present.
func (source ErrorSource) WithHeader(value string) ErrorSource {
	source.Header = value
	source.present |= sourceHeaderPresent
	return source
}

// MarshalJSON implements json.Marshaler for registered extension members.
func (source ErrorSource) MarshalJSON() ([]byte, error) {
	core := struct {
		Pointer   *string `json:"pointer,omitempty"`
		Parameter *string `json:"parameter,omitempty"`
		Header    *string `json:"header,omitempty"`
	}{
		Pointer:   optionalString(source.Pointer, source.present&sourcePointerPresent != 0),
		Parameter: optionalString(source.Parameter, source.present&sourceParameterPresent != 0),
		Header:    optionalString(source.Header, source.present&sourceHeaderPresent != 0),
	}

	return marshalObjectWithMembers(core, source.AdditionalMembers)
}

// Presence is independent of content: the empty string may be a present ID
// or pointer, while an untouched optional field is omitted.
func optionalString(value string, explicitlyPresent bool) *string {
	if value == "" && !explicitlyPresent {
		return nil
	}
	return &value
}
