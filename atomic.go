package jsonapi

import "strconv"

// AtomicExtensionURI identifies the official Atomic Operations extension.
const AtomicExtensionURI = "https://jsonapi.org/ext/atomic"

// AtomicOperationCode identifies an Atomic Operations mutation.
type AtomicOperationCode string

const (
	AtomicAdd    AtomicOperationCode = "add"
	AtomicUpdate AtomicOperationCode = "update"
	AtomicRemove AtomicOperationCode = "remove"
)

// AtomicDocument is a document using the official Atomic Operations
// extension. Operations, results, errors, and meta retain nil-versus-empty
// presence semantics.
type AtomicDocument struct {
	JSONAPI    *JSONAPI
	Links      Links
	Operations []AtomicOperation
	Results    []AtomicResult
	Errors     []ErrorObject
	Meta       Meta
}

// AtomicOperation describes one ordered mutation in an atomic request.
type AtomicOperation struct {
	Op   AtomicOperationCode
	Ref  *AtomicReference
	Href string
	Data *PrimaryData
	Meta Meta
}

// AtomicReference identifies a resource or one of its relationships.
type AtomicReference struct {
	Type         string `json:"type"`
	ID           string `json:"id,omitempty"`
	LID          string `json:"lid,omitempty"`
	Relationship string `json:"relationship,omitempty"`
}

// AtomicResult describes the result at the same position as its operation.
type AtomicResult struct {
	Data *PrimaryData
	Meta Meta
}

// Validate checks Atomic Operations document and operation invariants.
func (document AtomicDocument) Validate() error {
	validator := documentValidator{}

	if document.Operations == nil && document.Results == nil &&
		document.Errors == nil && document.Meta == nil {
		validator.add("", "required", "atomic document must contain operations, results, errors, or meta")
	}
	if document.Operations != nil && len(document.Operations) == 0 {
		validator.add("/atomic:operations", "min-items", "operations must contain at least one item")
	}
	if document.Results != nil && len(document.Results) == 0 {
		validator.add("/atomic:results", "min-items", "results must contain at least one item")
	}
	if document.Operations != nil && document.Results != nil {
		validator.add("/atomic:results", "conflict", "operations and results must not coexist")
	}
	if (document.Operations != nil || document.Results != nil) && document.Errors != nil {
		validator.add("/errors", "conflict", "atomic members and errors must not coexist")
	}
	if document.JSONAPI != nil {
		validator.validateJSONAPI(*document.JSONAPI)
	}
	validator.validateLinks(document.Links, "/links")
	validator.validateAtomicOperations(document.Operations)
	for index, result := range document.Results {
		validator.validatePrimaryData(
			result.Data,
			"/atomic:results/"+strconv.Itoa(index)+"/data",
			identityID,
			false,
			identityID,
		)
	}
	for index, apiError := range document.Errors {
		validator.validateError(apiError, "/errors/"+strconv.Itoa(index))
	}

	if len(validator.violations) == 0 {
		return nil
	}

	return &ValidationError{Violations: validator.violations}
}

func (validator *documentValidator) validateAtomicOperations(operations []AtomicOperation) {
	assigned := make(map[string]struct{})
	for index, operation := range operations {
		path := "/atomic:operations/" + strconv.Itoa(index)
		validator.validateAtomicOperation(operation, path)
		if operation.Ref != nil && operation.Ref.LID != "" {
			key := resourceKey(operation.Ref.Type, "", operation.Ref.LID)
			if _, exists := assigned[key]; !exists {
				validator.add(
					path+"/ref/lid",
					"unresolved-lid",
					"reference lid must be assigned by a prior operation",
				)
			}
		}
		if operation.Op == AtomicAdd && operation.Data != nil &&
			operation.Data.kind == primaryDataOne && operation.Data.one != nil {
			resource := operation.Data.one
			if resource.Type != "" && resource.LID != "" {
				assigned[resourceKey(resource.Type, "", resource.LID)] = struct{}{}
			}
		}
	}
}

func (validator *documentValidator) validateAtomicOperation(
	operation AtomicOperation,
	path string,
) {
	if operation.Op == "" {
		validator.add(path+"/op", "required", "operation code is required")
	} else if operation.Op != AtomicAdd && operation.Op != AtomicUpdate &&
		operation.Op != AtomicRemove {
		validator.add(path+"/op", "value", "operation code must be add, update, or remove")
	}
	if operation.Ref != nil && operation.Href != "" {
		validator.add(path+"/href", "conflict", "ref and href must not coexist")
	}
	if operation.Ref != nil {
		validator.validateAtomicReference(*operation.Ref, path+"/ref")
	}
	if operation.Href != "" {
		validator.validateURL(operation.Href, path+"/href")
	}

	relationshipTarget := operation.Ref != nil && operation.Ref.Relationship != ""
	switch operation.Op {
	case AtomicAdd:
		if operation.Data == nil {
			validator.add(path+"/data", "required", "add operation requires data")
		} else if relationshipTarget {
			validator.validateAtomicRelationshipData(operation.Data, path+"/data", true)
		} else {
			if operation.Ref != nil {
				validator.add(path+"/ref", "forbidden", "resource creation must not use ref")
			}
			validator.validateAtomicResourceData(operation.Data, path+"/data", identityOptional)
		}
	case AtomicUpdate:
		if operation.Data == nil {
			validator.add(path+"/data", "required", "update operation requires data")
		} else if relationshipTarget {
			validator.validateAtomicRelationshipData(operation.Data, path+"/data", false)
		} else {
			validator.validateAtomicResourceData(operation.Data, path+"/data", identityEither)
		}
	case AtomicRemove:
		if operation.Ref == nil && operation.Href == "" {
			validator.add(path+"/ref", "required", "remove operation requires ref or href")
		}
		if relationshipTarget && operation.Data == nil {
			validator.add(path+"/data", "required", "relationship removal requires data")
		} else if relationshipTarget {
			validator.validateAtomicRelationshipData(operation.Data, path+"/data", true)
		} else if operation.Data != nil && operation.Ref != nil {
			validator.add(path+"/data", "forbidden", "resource removal must not contain data")
		} else if operation.Data != nil {
			validator.validatePrimaryData(operation.Data, path+"/data", identityEither, false, identityEither)
		}
	}
}

func (validator *documentValidator) validateAtomicReference(
	reference AtomicReference,
	path string,
) {
	if reference.Type == "" {
		validator.add(path+"/type", "required", "reference type is required")
	} else if !validMemberName(reference.Type) {
		validator.add(path+"/type", "member-name", "reference type must be a valid member name")
	}
	if reference.ID == "" && reference.LID == "" {
		validator.add(path+"/id", "required", "reference requires id or lid")
	}
	if reference.ID != "" && reference.LID != "" {
		validator.add(path+"/lid", "conflict", "reference id and lid must not coexist")
	}
	if reference.Relationship != "" && !validMemberName(reference.Relationship) {
		validator.add(
			path+"/relationship",
			"member-name",
			"relationship must be a valid member name",
		)
	}
}

func (validator *documentValidator) validateAtomicResourceData(
	data *PrimaryData,
	path string,
	identity identityRequirement,
) {
	if data.kind != primaryDataOne || data.one == nil {
		validator.add(path, "shape", "resource operation data must be one resource object")
		return
	}
	validator.validateResource(*data.one, path, identity, true, identityEither)
}

func (validator *documentValidator) validateAtomicRelationshipData(
	data *PrimaryData,
	path string,
	requireMany bool,
) {
	if requireMany && data.kind != primaryDataMany {
		validator.add(path, "shape", "add and remove relationship data must be an array")
		return
	}
	if data.kind == primaryDataNull {
		return
	}
	if data.kind == primaryDataOne && data.one != nil {
		validator.validateAtomicIdentifier(*data.one, path)
		return
	}
	if data.kind == primaryDataMany {
		for index, resource := range data.many {
			validator.validateAtomicIdentifier(resource, path+"/"+strconv.Itoa(index))
		}
		return
	}
	validator.add(path, "shape", "relationship data must be null, an identifier, or an array")
}

func (validator *documentValidator) validateAtomicIdentifier(
	resource ResourceObject,
	path string,
) {
	validator.validateIdentifier(Identifier{
		Type: resource.Type,
		ID:   resource.ID,
		LID:  resource.LID,
		Meta: resource.Meta,
	}, path, identityEither)
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
