package jsonapi

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
	Type         string
	ID           string
	LID          string
	Relationship string
}

// AtomicResult describes the result at the same position as its operation.
type AtomicResult struct {
	Data *PrimaryData
	Meta Meta
}

// Validate checks top-level Atomic Operations document invariants.
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

	if len(validator.violations) == 0 {
		return nil
	}

	return &ValidationError{Violations: validator.violations}
}
