package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssertError(t *testing.T) {
	test := &testing.T{}
	Assert(test, GenericError.Make(), GenericError.Msg("new test error message").Make())
	assert.False(t, test.Failed())
	Assert(test, GenericError.Make(), ArgumentError.Make())
	assert.True(t, test.Failed())
}

func TestAssertGoError(t *testing.T) {
	err := fmt.Errorf("just a default error")

	test := &testing.T{}
	Assert(test, err, err)
	assert.False(t, test.Failed())
	Assert(test, Wrap(err), err)
	assert.False(t, test.Failed())
	Assert(test, err, Wrap(err))
	assert.False(t, test.Failed())
	Assert(test, Wrap(err), Wrap(err))
	assert.False(t, test.Failed())
	Assert(test, GenericError, err)
	assert.True(t, test.Failed())
}

func TestAssertTemplate(t *testing.T) {
	test := &testing.T{}
	Assert(test, GenericError, GenericError.Msg("new test error message").Make())
	assert.False(t, test.Failed())
	Assert(test, GenericError, ArgumentError.Make())
	assert.True(t, test.Failed())
}

func TestAssertPanic(t *testing.T) {
	test := &testing.T{}
	assert.Panics(t, func() {
		Assert(test, "this is not a valid error type", GenericError.Msg("new test error message").Make())
	})
}

func TestAssertNil(t *testing.T) {
	test := &testing.T{}
	AssertNil(test, nil)
	assert.False(t, test.Failed())
	AssertNil(test, GenericError.Make())
	assert.True(t, test.Failed())
}
