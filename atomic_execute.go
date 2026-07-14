package jsonapi

import (
	"context"
	"fmt"
)

// AtomicTransaction applies operations within one application-owned
// transaction. Implementations must not commit from ApplyAtomic.
type AtomicTransaction interface {
	ApplyAtomic(context.Context, AtomicOperation) (AtomicResult, error)
	CommitAtomic(context.Context) error
	RollbackAtomic(context.Context) error
}

// AtomicTransactionBeginner starts an application-owned transaction.
type AtomicTransactionBeginner interface {
	BeginAtomic(context.Context) (AtomicTransaction, error)
}

// AtomicExecutionError identifies the failed transaction phase and operation.
// OperationIndex is -1 for begin and commit failures.
type AtomicExecutionError struct {
	Phase          string
	OperationIndex int
	Cause          error
	RollbackCause  error
}

// Error implements error.
func (err *AtomicExecutionError) Error() string {
	if err.OperationIndex >= 0 {
		return fmt.Sprintf(
			"execute Atomic operation %d during %s: %v",
			err.OperationIndex,
			err.Phase,
			err.Cause,
		)
	}
	return fmt.Sprintf("execute Atomic transaction during %s: %v", err.Phase, err.Cause)
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
// order, and commits only after every operation succeeds. Any failure after a
// transaction begins triggers exactly one rollback attempt.
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

	transaction, err := beginner.BeginAtomic(ctx)
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

	finished := false
	defer func() {
		if recovered := recover(); recovered != nil {
			if !finished {
				_ = transaction.RollbackAtomic(context.WithoutCancel(ctx))
			}
			panic(recovered)
		}
	}()

	results := make([]AtomicResult, len(document.Operations))
	for index, operation := range document.Operations {
		if contextErr := ctx.Err(); contextErr != nil {
			finished = true
			return AtomicDocument{}, atomicExecutionFailure(
				ctx, transaction, "apply", index, contextErr,
			)
		}
		result, applyErr := transaction.ApplyAtomic(ctx, operation)
		if applyErr != nil {
			finished = true
			return AtomicDocument{}, atomicExecutionFailure(
				ctx, transaction, "apply", index, applyErr,
			)
		}
		results[index] = result
	}
	if contextErr := ctx.Err(); contextErr != nil {
		finished = true
		return AtomicDocument{}, atomicExecutionFailure(
			ctx, transaction, "commit", -1, contextErr,
		)
	}
	response := AtomicDocument{Results: results}
	if validationErr := response.ValidateWith(AtomicValidationOptions{
		Context:             AtomicResponseContext,
		ExpectedResultCount: len(document.Operations),
	}); validationErr != nil {
		finished = true
		return AtomicDocument{}, atomicExecutionFailure(
			ctx, transaction, "result-validation", -1, validationErr,
		)
	}
	if commitErr := transaction.CommitAtomic(ctx); commitErr != nil {
		finished = true
		return AtomicDocument{}, atomicExecutionFailure(
			ctx, transaction, "commit", -1, commitErr,
		)
	}
	finished = true

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
		RollbackCause:  transaction.RollbackAtomic(context.WithoutCancel(ctx)),
	}
}
