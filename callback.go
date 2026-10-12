package jsonapi

// CallbackError contains a failure from an application-supplied validator.
// Error deliberately omits the callback's error or panic value. Cause and
// PanicValue remain available for explicit diagnostics. Only wrapped validator
// seams convert ordinary panics; custom JSON methods, blocking and goroutine
// termination retain their own behavior. Error does not globally redact input
// paths, URI configuration or other package diagnostics.
type CallbackError struct {
	// Phase names the invoked extension, profile, cursor or sort seam.
	Phase string
	// Cause retains returned errors or error-valued panics without redacting
	// their contents.
	Cause error
	// PanicValue retains the original panic value; use CallbackPanicValue to
	// distinguish a panic.
	PanicValue any
	panicked   bool
}

// Error implements error without disclosing application-owned values.
func (err *CallbackError) Error() string {
	return "JSON:API application callback failed during " + err.Phase
}

// Unwrap exposes an error returned by or panicked from the callback.
func (err *CallbackError) Unwrap() error {
	return err.Cause
}

// CallbackPhase identifies the callback seam that failed.
func (err *CallbackError) CallbackPhase() string {
	return err.Phase
}

// CallbackPanicValue returns the panic value and whether the callback panicked.
func (err *CallbackError) CallbackPanicValue() (any, bool) {
	return err.PanicValue, err.panicked
}

// callApplicationCallback converts ordinary panics at the declared seam only.
// It is not a goroutine, timeout, mutation or runtime.Goexit isolation boundary.
func callApplicationCallback(phase string, callback func() error) (err error) {
	defer func() {
		if value := recover(); value != nil {
			cause, _ := value.(error)
			err = &CallbackError{
				Phase: phase, Cause: cause, PanicValue: value, panicked: true,
			}
		}
	}()
	if cause := callback(); cause != nil {
		return &CallbackError{Phase: phase, Cause: cause}
	}
	return nil
}
