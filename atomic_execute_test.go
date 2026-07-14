package jsonapi

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestExecuteAtomicAppliesInOrderAndCommits(t *testing.T) {
	t.Parallel()

	transaction := &recordingAtomicTransaction{}
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		return transaction, nil
	})
	document := AtomicDocument{Operations: []AtomicOperation{
		{Op: AtomicRemove, Href: "/articles/1"},
		{Op: AtomicRemove, Href: "/articles/2"},
	}}

	result, err := ExecuteAtomic(context.Background(), beginner, document)
	if err != nil {
		t.Fatalf("execute operations: %v", err)
	}
	if !reflect.DeepEqual(transaction.hrefs, []string{"/articles/1", "/articles/2"}) {
		t.Fatalf("unexpected operation order: %#v", transaction.hrefs)
	}
	if !transaction.committed || transaction.rollbackCalls != 0 {
		t.Fatalf("unexpected transaction state: %#v", transaction)
	}
	if len(result.Results) != 2 {
		t.Fatalf("unexpected result count: %d", len(result.Results))
	}
}

func TestExecuteAtomicRollsBackAtFirstOperationFailure(t *testing.T) {
	t.Parallel()

	applyFailure := errors.New("apply failed")
	transaction := &recordingAtomicTransaction{failAt: 1, applyError: applyFailure}
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		return transaction, nil
	})
	document := AtomicDocument{Operations: []AtomicOperation{
		{Op: AtomicRemove, Href: "/articles/1"},
		{Op: AtomicRemove, Href: "/articles/2"},
		{Op: AtomicRemove, Href: "/articles/3"},
	}}

	_, err := ExecuteAtomic(context.Background(), beginner, document)
	var executionError *AtomicExecutionError
	if !errors.As(err, &executionError) {
		t.Fatalf("expected AtomicExecutionError, got %T: %v", err, err)
	}
	if executionError.Phase != "apply" || executionError.OperationIndex != 1 ||
		!errors.Is(err, applyFailure) {
		t.Fatalf("unexpected execution error: %#v", executionError)
	}
	if !reflect.DeepEqual(transaction.hrefs, []string{"/articles/1", "/articles/2"}) {
		t.Fatalf("execution continued after failure: %#v", transaction.hrefs)
	}
	if transaction.committed || transaction.rollbackCalls != 1 {
		t.Fatalf("unexpected transaction state: %#v", transaction)
	}
}

func TestExecuteAtomicDoesNotBeginInvalidRequest(t *testing.T) {
	t.Parallel()

	beginCalls := 0
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		beginCalls++
		return &recordingAtomicTransaction{}, nil
	})
	_, err := ExecuteAtomic(context.Background(), beginner, AtomicDocument{})
	if err == nil {
		t.Fatal("expected request validation error")
	}
	if beginCalls != 0 {
		t.Fatalf("began transaction for invalid request %d times", beginCalls)
	}
}

func TestExecuteAtomicRejectsMissingBeginner(t *testing.T) {
	t.Parallel()

	document := AtomicDocument{Operations: []AtomicOperation{{
		Op: AtomicRemove, Href: "/articles/1",
	}}}
	_, err := ExecuteAtomic(context.Background(), nil, document)
	var executionError *AtomicExecutionError
	if !errors.As(err, &executionError) || executionError.Phase != "begin" {
		t.Fatalf("unexpected missing beginner error: %T %#v", err, executionError)
	}
}

func TestExecuteAtomicReportsBeginFailures(t *testing.T) {
	t.Parallel()

	document := AtomicDocument{Operations: []AtomicOperation{{
		Op: AtomicRemove, Href: "/articles/1",
	}}}
	beginFailure := errors.New("database unavailable")
	tests := []struct {
		beginner AtomicTransactionBeginner
		cause    error
	}{
		{
			beginner: atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
				return nil, beginFailure
			}),
			cause: beginFailure,
		},
		{
			beginner: atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
				return nil, nil
			}),
		},
	}
	for _, test := range tests {
		_, err := ExecuteAtomic(context.Background(), test.beginner, document)
		var executionError *AtomicExecutionError
		if !errors.As(err, &executionError) || executionError.Phase != "begin" ||
			executionError.OperationIndex != -1 {
			t.Fatalf("unexpected begin error: %T %#v", err, executionError)
		}
		if test.cause != nil && !errors.Is(err, test.cause) {
			t.Fatalf("begin cause was not preserved: %v", err)
		}
	}
}

func TestExecuteAtomicRollsBackCommitFailure(t *testing.T) {
	t.Parallel()

	commitFailure := errors.New("commit failed")
	transaction := &recordingAtomicTransaction{commitError: commitFailure}
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		return transaction, nil
	})
	document := AtomicDocument{Operations: []AtomicOperation{{
		Op: AtomicRemove, Href: "/articles/1",
	}}}

	_, err := ExecuteAtomic(context.Background(), beginner, document)
	var executionError *AtomicExecutionError
	if !errors.As(err, &executionError) || executionError.Phase != "commit" ||
		executionError.OperationIndex != -1 || !errors.Is(err, commitFailure) {
		t.Fatalf("unexpected commit error: %T %#v", err, executionError)
	}
	if transaction.rollbackCalls != 1 {
		t.Fatalf("expected one rollback, got %d", transaction.rollbackCalls)
	}
}

func TestExecuteAtomicPreservesRollbackFailure(t *testing.T) {
	t.Parallel()

	applyFailure := errors.New("apply failed")
	rollbackFailure := errors.New("rollback failed")
	transaction := &recordingAtomicTransaction{
		failAt:        0,
		applyError:    applyFailure,
		rollbackError: rollbackFailure,
	}
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		return transaction, nil
	})
	document := AtomicDocument{Operations: []AtomicOperation{{
		Op: AtomicRemove, Href: "/articles/1",
	}}}

	_, err := ExecuteAtomic(context.Background(), beginner, document)
	if !errors.Is(err, applyFailure) || !errors.Is(err, rollbackFailure) {
		t.Fatalf("execution error did not preserve both causes: %v", err)
	}
}

func TestExecuteAtomicRollsBackAndRepanics(t *testing.T) {
	t.Parallel()

	transaction := &recordingAtomicTransaction{panicAt: 0, panicValue: "boom"}
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		return transaction, nil
	})
	document := AtomicDocument{Operations: []AtomicOperation{{
		Op: AtomicRemove, Href: "/articles/1",
	}}}

	defer func() {
		if recovered := recover(); recovered != "boom" {
			t.Fatalf("unexpected panic: %#v", recovered)
		}
		if transaction.rollbackCalls != 1 {
			t.Fatalf("expected panic rollback, got %d calls", transaction.rollbackCalls)
		}
	}()
	_, _ = ExecuteAtomic(context.Background(), beginner, document)
	t.Fatal("expected panic")
}

func TestExecuteAtomicStopsAndRollsBackOnCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	transaction := &recordingAtomicTransaction{cancelAt: 0, cancel: cancel}
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		return transaction, nil
	})
	document := AtomicDocument{Operations: []AtomicOperation{
		{Op: AtomicRemove, Href: "/articles/1"},
		{Op: AtomicRemove, Href: "/articles/2"},
	}}

	_, err := ExecuteAtomic(ctx, beginner, document)
	var executionError *AtomicExecutionError
	if !errors.As(err, &executionError) || executionError.Phase != "apply" ||
		executionError.OperationIndex != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected cancellation error: %T %#v", err, executionError)
	}
	if !reflect.DeepEqual(transaction.hrefs, []string{"/articles/1"}) {
		t.Fatalf("operation ran after cancellation: %#v", transaction.hrefs)
	}
	if transaction.rollbackContextError != nil {
		t.Fatalf("rollback received canceled context: %v", transaction.rollbackContextError)
	}
}

func TestExecuteAtomicChecksCancellationBeforeCommit(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	transaction := &recordingAtomicTransaction{cancelAt: 0, cancel: cancel}
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		return transaction, nil
	})
	document := AtomicDocument{Operations: []AtomicOperation{{
		Op: AtomicRemove, Href: "/articles/1",
	}}}
	_, err := ExecuteAtomic(ctx, beginner, document)
	var executionError *AtomicExecutionError
	if !errors.As(err, &executionError) || executionError.Phase != "commit" ||
		executionError.OperationIndex != -1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected pre-commit cancellation: %T %#v", err, executionError)
	}
	if transaction.rollbackCalls != 1 || transaction.committed {
		t.Fatalf("unexpected transaction state: %#v", transaction)
	}
}

func TestExecuteAtomicRollsBackInvalidResultsBeforeCommit(t *testing.T) {
	t.Parallel()

	transaction := &recordingAtomicTransaction{results: []AtomicResult{{
		Data: ResourceData(ResourceObject{Type: "articles"}),
	}}}
	beginner := atomicBeginnerFunc(func(context.Context) (AtomicTransaction, error) {
		return transaction, nil
	})
	document := AtomicDocument{Operations: []AtomicOperation{{
		Op: AtomicRemove, Href: "/articles/1",
	}}}

	_, err := ExecuteAtomic(context.Background(), beginner, document)
	var executionError *AtomicExecutionError
	if !errors.As(err, &executionError) || executionError.Phase != "result-validation" {
		t.Fatalf("unexpected result validation error: %T %#v", err, executionError)
	}
	if transaction.committed || transaction.rollbackCalls != 1 {
		t.Fatalf("invalid result escaped transaction: %#v", transaction)
	}
}

type atomicBeginnerFunc func(context.Context) (AtomicTransaction, error)

func (begin atomicBeginnerFunc) BeginAtomic(ctx context.Context) (AtomicTransaction, error) {
	return begin(ctx)
}

type recordingAtomicTransaction struct {
	hrefs                []string
	failAt               int
	applyError           error
	commitError          error
	rollbackError        error
	committed            bool
	rollbackCalls        int
	panicAt              int
	panicValue           any
	cancelAt             int
	cancel               context.CancelFunc
	rollbackContextError error
	results              []AtomicResult
}

func (transaction *recordingAtomicTransaction) ApplyAtomic(
	_ context.Context,
	operation AtomicOperation,
) (AtomicResult, error) {
	transaction.hrefs = append(transaction.hrefs, operation.Href)
	index := len(transaction.hrefs) - 1
	if transaction.panicValue != nil && index == transaction.panicAt {
		panic(transaction.panicValue)
	}
	if transaction.cancel != nil && index == transaction.cancelAt {
		transaction.cancel()
	}
	if transaction.applyError != nil && index == transaction.failAt {
		return AtomicResult{}, transaction.applyError
	}
	if index < len(transaction.results) {
		return transaction.results[index], nil
	}
	return AtomicResult{Meta: Meta{"href": operation.Href}}, nil
}

func (transaction *recordingAtomicTransaction) CommitAtomic(context.Context) error {
	if transaction.commitError != nil {
		return transaction.commitError
	}
	transaction.committed = true
	return nil
}

func (transaction *recordingAtomicTransaction) RollbackAtomic(ctx context.Context) error {
	transaction.rollbackCalls++
	transaction.rollbackContextError = ctx.Err()
	return transaction.rollbackError
}
