package errors

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetType(t *testing.T) {
	err := New("test")
	assert.Equal(t, ErrorType("test"), getErrorType(err))
}

func TestMessage(t *testing.T) {
	err := New("test").Msg("Test message")
	assert.Equal(t, "Test message", err.Error())
}

func TestMessageArgs(t *testing.T) {
	err := New("test").Msg("Test %v message", "foobar")
	assert.Equal(t, "Test foobar message", err.Error())
}

func TestEmptyMessage(t *testing.T) {
	err := New("test")
	assert.Equal(t, "test", err.Error())
}

func TestFormatMessage(t *testing.T) {
	err := New("test").Msg("Test %v message").Args("foobar")
	assert.Equal(t, "Test foobar message", err.Error())
}

func TestCause(t *testing.T) {
	err := New("test").Msg("Error").Cause(New("suberr").Msg("Inner message"))
	assert.Equal(t, "Error: Inner message", err.Error())
}

func TestStrCause(t *testing.T) {
	err := New("test").Msg("Error").StrCause("inner message").Safe()
	assert.Equal(t, "Error: inner message", err.Error())
	assert.Equal(t, "Error", err.SafeString())
}

func TestExpand(t *testing.T) {
	err1 := Wrap(fmt.Errorf("error one")).Safe()
	err2 := err1.Expand("other message")
	assert.Equal(t, "other message: error one", err2.Error())
	assert.Equal(t, "", err2.SafeString())
	assert.Equal(t, err1.GetType(), err2.GetType())
}

func TestExpandSafe(t *testing.T) {
	err1 := Wrap(fmt.Errorf("error one")).Safe()
	err2 := err1.ExpandSafe("other message")
	assert.Equal(t, "other message: error one", err2.SafeString())
	assert.Equal(t, err1.GetType(), err2.GetType())
}

func TestExpandSafeWithUnsafeCause(t *testing.T) {
	err1 := Wrap(fmt.Errorf("error one"))
	err2 := err1.ExpandSafe("other message")
	assert.Equal(t, "other message", err2.SafeString())
	assert.Equal(t, err1.GetType(), err2.GetType())
}

func TestUnsafe(t *testing.T) {
	err := New("test").Msg("totally unsafe secret ane46ndsn4e")
	api := err.API()
	assert.False(t, strings.Contains(api.Message, "ane46ndsn4e"))
	assert.True(t, strings.Contains(api.Message, err.GetID()))
}

func TestUnsafeDisabled(t *testing.T) {
	PrintUnsafeErrors = true
	defer func() { PrintUnsafeErrors = false }()
	err := New("test").Msg("totally unsafe secret ane46ndsn4e")
	assert.True(t, strings.Contains(err.API().Message, "ane46ndsn4e"))
}

func TestSafe(t *testing.T) {
	err := New("test").Msg("safe gibberish: ane46ndsn4e").Safe()
	assert.True(t, strings.Contains(err.API().Message, "ane46ndsn4e"))
}

func TestTemplate(t *testing.T) {
	err := New("test").Msg("safe gibberish: ane46ndsn4e")
	oldID := err.GetID()
	err2 := err.Template().Msg("another try")
	assert.NotEqual(t, oldID, err2.GetID())
}

func TestEquals(t *testing.T) {
	err := GenericError.Msg("test %v").Args("foobar").Cause(nil).Cause(fmt.Errorf("inner")).HTTPCode(400).ErrCode(42)
	assert.True(t, err.Equals(GenericError))
	assert.False(t, err.Equals(nil))
	assert.True(t, AreEqual(err, GenericError))
	assert.True(t, AreEqual(GenericError, err))
	assert.False(t, AreEqual(err, nil))
	assert.False(t, AreEqual(nil, err))
	assert.True(t, AreEqual(nil, nil))
}

func TestWrap(t *testing.T) {
	err := Wrap(fmt.Errorf("inner error"))
	assert.True(t, strings.Contains(err.Error(), "inner error"))
}

func TestWrapNil(t *testing.T) {
	assert.Nil(t, Wrap(nil))
}

func TestWrapNoMessage(t *testing.T) {
	err := Wrap(fmt.Errorf(""))
	assert.True(t, strings.Contains(err.Error(), string(err.GetType())))
}

func TestNoMoreWrapping(t *testing.T) {
	err := Wrap(fmt.Errorf(""))
	assert.Equal(t, err, Wrap(err))
}

func TestWithTypeWrap(t *testing.T) {
	err := WrapT(fmt.Errorf("inner error")).Safe()
	assert.True(t, strings.Contains(err.Error(), "inner error"))
	assert.True(t, strings.Contains(err.Error(), string(err.GetType())))
}

func TestWrapOnlyType(t *testing.T) {
	err := WrapT(fmt.Errorf("")).Safe()
	assert.True(t, strings.Contains(err.Error(), string(err.GetType())))
}

func TestDefaultAPI(t *testing.T) {
	err := DefaultAPI("test api")
	expectedErr := APIError{defaultHTTPCode, defaultErrCode, "test api"}
	assert.Equal(t, expectedErr, err)
}

func TestToAPI(t *testing.T) {
	err := GenericError.Msg("test api").HTTPCode(400).ErrCode(42).Untracked().Safe()
	expectedErr := APIError{400, 42, "test api"}
	assert.Equal(t, expectedErr, err.API())
}

func TestID(t *testing.T) {
	err := GenericError
	assert.Equal(t, "", err.GetID())
	err = err.Msg("new %v message")
	id := err.GetID()
	assert.NotEqual(t, "", id)
	err = err.Args("test")
	assert.Equal(t, id, err.GetID())
}

func TestStackTrace(t *testing.T) {
	err := GenericError
	assert.Equal(t, "", err.GetStackTrace())
	err = err.Msg("new %v message")
	trace := err.GetStackTrace()
	assert.True(t, strings.Contains(err.GetStackTrace(), "TestStackTrace"), "Stack trace should contain 'TestStackTrace'")
	assert.False(t, strings.Contains(err.GetStackTrace(), "Msg"), "Stack trace should not contain 'Msg'")
	err = err.Args("test")
	assert.Equal(t, trace, err.GetStackTrace(), "Stack trace should not change once it is prepared")
}

func TestErrorToRequest(t *testing.T) {
	err := New("TestError").Msg("This is a safe error message").HTTPCode(400).ErrCode(123).Untracked().Safe()
	r := &requestAborter{}
	err.ToRequest(r)
	expected := API(400, 123, "This is a safe error message")
	assert.Equal(t, 400, r.lastHTTPCode)
	assert.Equal(t, expected, *r.lastError)
}

func TestToRequest(t *testing.T) {
	err := New("TestError").Msg("This is a safe error message").HTTPCode(400).ErrCode(123).Untracked().Safe()
	r := &requestAborter{}
	ToRequest(r, err)
	expected := API(400, 123, "This is a safe error message")
	assert.Equal(t, 400, r.lastHTTPCode)
	assert.Equal(t, expected, *r.lastError)
}

func TestNilToRequest(t *testing.T) {
	r := &requestAborter{}
	ToRequest(r, nil)
	assert.Nil(t, r.lastError)
}

func TestToLog(t *testing.T) {
	err := New("TestError").Msg("a safe error message").StrCause("an unsafe cause").HTTPCode(500).ErrCode(42).Safe()
	r := &requestAborter{}
	buffer := new(bytes.Buffer)
	log.SetOutput(buffer)
	err.ToRequestAndLog(r)
	str := buffer.String()

	assert.True(t, strings.Contains(str, err.GetID()), "Log should contain error id")
	assert.True(t, strings.Contains(str, err.Error()), "Log should contain unsafe error message")
	assert.True(t, strings.Contains(str, "TestToLog"), "Log should contain stack trace")
}

func TestToLogExcept(t *testing.T) {
	err := New("TestError").Msg("a safe error message").StrCause("an unsafe cause").HTTPCode(500).ErrCode(42).Safe()
	r := &requestAborter{}
	buffer := new(bytes.Buffer)
	log.SetOutput(buffer)
	err.ToRequestAndLog(r, New("TestError"))
	str := buffer.String()

	assert.False(t, strings.Contains(str, err.GetID()), "Log should not contain error id")
	assert.False(t, strings.Contains(str, err.Error()), "Log should not contain unsafe error message")
	assert.False(t, strings.Contains(str, "TestToLog"), "Log should not contain stack trace")
}

/* ############################################# */
/* ###                Helper                 ### */
/* ############################################# */

type requestAborter struct {
	lastHTTPCode int
	lastError    *APIError
}

func (r *requestAborter) AbortWithStatusJSON(code int, obj interface{}) {
	r.lastHTTPCode = code
	err, ok := obj.(APIError)
	if !ok {
		panic(fmt.Sprintf("AbortWithStatusJSON expects APIError but got %T", obj))
	}
	r.lastError = &err
}
