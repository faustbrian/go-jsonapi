package jsonapi

import (
	"context"
	"fmt"
)

// AtomicTransaction applies operations within one application-owned
// transaction. Implementations must not commit from ApplyAtomic.
type AtomicTransaction interface {
	// ApplyAtomic applies one operation without committing and returns its
	// corresponding result.
	ApplyAtomic(context.Context, AtomicOperation) (AtomicResult, error)
	// CommitAtomic finalizes the transaction after all results pass validation.
	CommitAtomic(context.Context) error
	// RollbackAtomic attempts cleanup once after a handled post-begin failure;
	// bound work independently of caller cancellation.
	RollbackAtomic(context.Context) error
}

// AtomicTransactionBeginner starts an application-owned transaction.
type AtomicTransactionBeginner interface {
	// BeginAtomic returns a usable transaction on success; failed partial
	// acquisition remains the implementation's cleanup responsibility.
	BeginAtomic(context.Context) (AtomicTransaction, error)
}

// AtomicExecutionError identifies the failed transaction phase and operation.
// OperationIndex is -1 for begin and commit failures.
type AtomicExecutionError struct {
	// Phase names begin, apply, result-validation or commit failure.
	Phase string
	// OperationIndex is the zero-based apply index, or -1 for transaction-wide
	// failures.
	OperationIndex int
	// Cause preserves the original failure, including sensitive application
	// diagnostic data.
	Cause error
	// RollbackCause preserves a separate rollback failure when cleanup was
	// attempted.
	RollbackCause error
}

// AtomicPanicError identifies a panic raised by an application transaction
// callback. Value is retained for explicit diagnostics but omitted from Error.
type AtomicPanicError struct {
	// Phase names the transaction callback that panicked.
	Phase string
	// Value retains its original panic value; Error deliberately omits it.
	Value any
}

// Error implements error without formatting the potentially sensitive panic
// value.
func (err *AtomicPanicError) Error() string {
	return "Atomic transaction callback panicked during " + err.Phase
}

// Unwrap exposes a panic value that is itself an error.
func (err *AtomicPanicError) Unwrap() error {
	cause, _ := err.Value.(error)
	return cause
}

// Error implements error.
func (err *AtomicExecutionError) Error() string {
	if err.OperationIndex >= 0 {
		return fmt.Sprintf(
			"execute Atomic operation %d during %s",
			err.OperationIndex,
			err.Phase,
		)
	}
	return fmt.Sprintf("execute Atomic transaction during %s", err.Phase)
}

// Unwrap exposes both the primary and rollback failures to errors.Is/As.
func (err *AtomicExecutionError) Unwrap() []error {
	errors := []error{err.Cause}
	if err.RollbackCause != nil {
		errors = append(errors, err.RollbackCause)
	}
	return errors
}

// ExecuteAtomic validates an Atomic request, applies operations in document
// order, validates corresponding results, and commits only after success. Nil
// beginner/ctx and canceled ctx are refused after request validation. Each
// handled post-begin failure returns zero response data and attempts rollback
// once, including a commit error; that attempt cannot establish commit durability.
// Ordinary transaction callback panics become AtomicPanicError. Blocking and
// goroutine termination are not isolated, and no retry is performed.
//
// Rollback receives context.WithoutCancel(ctx), retaining values but removing
// caller cancellation/deadlines with no replacement package deadline. The
// transaction implementation must bound cleanup independently. Failed partial
// Begin acquisition stays implementation-owned; external effects and transaction
// isolation/atomicity are application obligations, not provided by this helper.
func ExecuteAtomic(
	ctx context.Context,
	beginner AtomicTransactionBeginner,
	document AtomicDocument,
) (AtomicDocument, error) {
	if err := document.ValidateWith(AtomicValidationOptions{Context: AtomicRequestContext}); err != nil {
		return AtomicDocument{}, err
	}
	if beginner == nil {
		return AtomicDocument{}, &AtomicExecutionError{
			Phase: "begin", OperationIndex: -1,
			Cause: fmt.Errorf("transaction beginner is required"),
		}
	}
	if ctx == nil {
		return AtomicDocument{}, &AtomicExecutionError{
			Phase: "begin", OperationIndex: -1,
			Cause: fmt.Errorf("context is required"),
		}
	}
	if contextErr := ctx.Err(); contextErr != nil {
		return AtomicDocument{}, &AtomicExecutionError{
			Phase: "begin", OperationIndex: -1, Cause: contextErr,
		}
	}

	transaction, err := beginAtomicSafely(ctx, beginner)
	if err != nil {
		return AtomicDocument{}, &AtomicExecutionError{
			Phase: "begin", OperationIndex: -1, Cause: err,
		}
	}
	if transaction == nil {
		return AtomicDocument{}, &AtomicExecutionError{
			Phase: "begin", OperationIndex: -1,
			Cause: fmt.Errorf("transaction beginner returned nil transaction"),
		}
	}

	results := make([]AtomicResult, len(document.Operations))
	for index, operation := range document.Operations {
		if contextErr := ctx.Err(); contextErr != nil {
			return AtomicDocument{}, atomicExecutionFailure(
				ctx, transaction, "apply", index, contextErr,
			)
		}
		result, applyErr := applyAtomicSafely(ctx, transaction, operation)
		if applyErr != nil {
			return AtomicDocument{}, atomicExecutionFailure(
				ctx, transaction, "apply", index, applyErr,
			)
		}
		results[index] = result
	}
	if contextErr := ctx.Err(); contextErr != nil {
		return AtomicDocument{}, atomicExecutionFailure(
			ctx, transaction, "commit", -1, contextErr,
		)
	}
	response := AtomicDocument{Results: results}
	if validationErr := response.ValidateWith(AtomicValidationOptions{
		Context:             AtomicResponseContext,
		ExpectedResultCount: len(document.Operations),
		ExpectedOperations:  document.Operations,
	}); validationErr != nil {
		return AtomicDocument{}, atomicExecutionFailure(
			ctx, transaction, "result-validation", -1, validationErr,
		)
	}
	if commitErr := commitAtomicSafely(ctx, transaction); commitErr != nil {
		return AtomicDocument{}, atomicExecutionFailure(
			ctx, transaction, "commit", -1, commitErr,
		)
	}
	return response, nil
}

func atomicExecutionFailure(
	ctx context.Context,
	transaction AtomicTransaction,
	phase string,
	operationIndex int,
	cause error,
) error {
	return &AtomicExecutionError{
		Phase:          phase,
		OperationIndex: operationIndex,
		Cause:          cause,
		// Detach cancellation so cleanup is attempted; the application adapter
		// must supply its own bound because this helper adds no deadline.
		RollbackCause: rollbackAtomicSafely(context.WithoutCancel(ctx), transaction),
	}
}

func beginAtomicSafely(
	ctx context.Context,
	beginner AtomicTransactionBeginner,
) (transaction AtomicTransaction, err error) {
	defer func() {
		if value := recover(); value != nil {
			transaction = nil
			err = &AtomicPanicError{Phase: "begin", Value: value}
		}
	}()
	return beginner.BeginAtomic(ctx)
}

func applyAtomicSafely(
	ctx context.Context,
	transaction AtomicTransaction,
	operation AtomicOperation,
) (result AtomicResult, err error) {
	defer func() {
		if value := recover(); value != nil {
			result = AtomicResult{}
			err = &AtomicPanicError{Phase: "apply", Value: value}
		}
	}()
	return transaction.ApplyAtomic(ctx, operation)
}

func commitAtomicSafely(ctx context.Context, transaction AtomicTransaction) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = &AtomicPanicError{Phase: "commit", Value: value}
		}
	}()
	return transaction.CommitAtomic(ctx)
}

func rollbackAtomicSafely(ctx context.Context, transaction AtomicTransaction) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = &AtomicPanicError{Phase: "rollback", Value: value}
		}
	}()
	return transaction.RollbackAtomic(ctx)
}
