package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Assert performs test assertions to ensure error equality. The expected error can be of type error or errors.Template.
func Assert(t *testing.T, expected interface{}, actual error, msgAndArgs ...interface{}) bool {
	errStr := "<nil>"
	if actual != nil {
		errStr = actual.Error()
	}

	switch e := expected.(type) {
	case Template:
		if !InstanceOf(actual, e) {
			return assert.Fail(t, fmt.Sprintf("Expected error of type %q, but got %q instead", e.GetType(), errStr), msgAndArgs...)
		}
		return true

	case error:
		if !AreEqual(actual, e) {
			return assert.Fail(t, fmt.Sprintf("Expected error of type %q, but got %q instead", Wrap(e).GetType(), errStr), msgAndArgs...)
		}
		return true

	default:
		panic(fmt.Sprintf("assertError requires expected error of type 'error' or 'errors.Template', but got '%T' instead", expected))
	}
}

// AssertNil performs test assertions to ensure the given error is nil.
func AssertNil(t *testing.T, actual error, msgAndArgs ...interface{}) bool {
	if (actual) == nil {
		return true
	}

	return assert.Fail(t, fmt.Sprintf("Expected no error, but got %q instead", actual.Error()), msgAndArgs...)
}
