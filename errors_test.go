package errors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetType(t *testing.T) {
	err := New("test").Make()
	assert.Equal(t, ErrorType("test"), getErrorType(err))
}

func TestMessage(t *testing.T) {
	err := New("test").Msg("Test message").Make()
	assert.Equal(t, "Test message", err.Error())
}

func TestMessageArgs(t *testing.T) {
	err := New("test").Msg("Test %v message", "foobar").Make()
	assert.Equal(t, "Test foobar message", err.Error())
}

func TestNewMessageArgs(t *testing.T) {
	err := New("Test %v message", "foobar").Make()
	assert.Equal(t, "Test foobar message", err.Error())
}

func TestTemplateArgs(t *testing.T) {
	err := New("foo %v bar").Args("42").Make()
	assert.Equal(t, "foo 42 bar", err.Error())
}

func TestErrorMessageArgs(t *testing.T) {
	err := New("test").Make().Msg("Test %v message", "foobar")
	assert.Equal(t, "Test foobar message", err.Error())
}

func TestEmptyMessage(t *testing.T) {
	err := New("test").Make()
	assert.Equal(t, "test", err.Error())
}

func TestFormatMessage(t *testing.T) {
	err := New("test").Msg("Test %v message").Make().Args("foobar")
	assert.Equal(t, "Test foobar message", err.Error())
}

func TestCause(t *testing.T) {
	err := New("test").Msg("Error").Make().Cause(New("suberr").Msg("Inner message").Make())
	assert.Equal(t, "Error: Inner message", err.Error())
}

func TestStrCause(t *testing.T) {
	err := New("test").Msg("Error").Make().StrCause("inner message").Safe()
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
	err := New("test").Msg("totally unsafe secret ane46ndsn4e").Track().Make()
	api := err.API()
	assert.False(t, strings.Contains(api.Message, "ane46ndsn4e"))
	assert.True(t, strings.Contains(api.Message, err.GetID()))
}

func TestUnsafeDisabled(t *testing.T) {
	PrintUnsafeErrors = true
	defer func() { PrintUnsafeErrors = false }()
	err := New("test").Msg("totally unsafe secret ane46ndsn4e").Make()
	assert.True(t, strings.Contains(err.API().Message, "ane46ndsn4e"))
}

func TestSafe(t *testing.T) {
	err := New("test").Msg("safe gibberish: ane46ndsn4e").Safe().Make()
	assert.True(t, strings.Contains(err.API().Message, "ane46ndsn4e"))
}

func TestEquals(t *testing.T) {
	err := GenericError.Make().Msg("test %v").Args("foobar").Cause(nil).Cause(fmt.Errorf("inner")).HTTPCode(400).ErrCode(42)
	assert.True(t, err.Is(GenericError))
	assert.False(t, err.Equals(nil))
	assert.True(t, InstanceOf(err, GenericError))
	assert.False(t, InstanceOf(nil, GenericError))
	assert.True(t, GenericError.Make().Equals(err))
	assert.True(t, AreEqual(err, GenericError.Make()))
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
	err := New("test api").API(400, 42).Make()
	expectedErr := APIError{400, 42, "test api"}
	assert.Equal(t, expectedErr, err.API())
}

func TestID(t *testing.T) {
	err := GenericError.Make()
	err = err.Msg("new %v message")
	id := err.GetID()
	assert.NotEqual(t, "", id)
	err = err.Args("test")
	assert.Equal(t, id, err.GetID())
}

func TestIDPersistence(t *testing.T) {
	err := GenericError.Make()
	err = err.Msg("new %v message")
	err2 := err.Args("foo").Safe().Expand("outer exception").ErrCode(4).HTTPCode(403).Untrack().NoTrace()
	assert.NotEqual(t, "", err2.GetID())
	err = err.Args("test")
	assert.Equal(t, err.GetID(), err2.GetID())
}

func TestStackTrace(t *testing.T) {
	err := GenericError.NoTrace().Trace().Make()
	err = err.Msg("new %v message")
	trace := err.GetStackTrace()
	assert.True(t, strings.Contains(trace, "TestStackTrace"), "Stack trace should contain 'TestStackTrace'")
	assert.False(t, strings.Contains(trace, "Msg"), "Stack trace should not contain 'Msg'")
	err = err.Args("test")
	assert.Equal(t, trace, err.GetStackTrace(), "Stack trace should not change once it is prepared")
}

func TestMakeTraced(t *testing.T) {
	err := innerMakeTraced()
	trace := err.GetStackTrace()
	assert.Contains(t, trace, "TestMakeTraced", "Stack trace should contain 'TestMakeTraced'")
	assert.NotContains(t, trace, "innerMakeTraced", "Stack trace should contain 'innerMakeTraced'")
}

func innerMakeTraced() Error {
	return GenericError.NoTrace().Trace().MakeTraced(1)
}

func TestErrorToRequest(t *testing.T) {
	err := New("TestError").Msg("This is a safe error message").HTTPCode(400).ErrCode(123).Safe().Untrack().Make()
	r := &requestAborter{}
	err.ToRequest(r)
	expected := API(400, 123, "This is a safe error message")
	assert.Equal(t, 400, r.lastHTTPCode)
	assert.Equal(t, expected, *r.lastError)
}

func TestToRequest(t *testing.T) {
	err := New("TestError").Msg("This is a safe error message").API(400, 123).Make()
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

func TestDefaultLogger(t *testing.T) {
	err := New("TestError").Msg("a safe error message").Make().StrCause("an unsafe cause").HTTPCode(500).ErrCode(42).Safe()
	err.ToLog()
}

func TestToRequestAndLog(t *testing.T) {
	err := New("TestError").Msg("a safe error message").Trace().Make().StrCause("an unsafe cause").HTTPCode(500).ErrCode(42).Safe()
	r := &requestAborter{}
	lb := setLogBuffer()
	err.ToRequestAndLog(r)
	str := lb.String()

	assert.True(t, strings.Contains(str, err.GetID()), "Log should contain error id")
	assert.True(t, strings.Contains(str, err.Error()), "Log should contain unsafe error message")
	assert.True(t, strings.Contains(str, "TestToRequestAndLog"), "Log should contain stack trace")
}

func TestToRequestAndLogUntracked(t *testing.T) {
	err := New("TestError").Msg("a safe error message").Make().HTTPCode(421).ErrCode(80).Safe().Untrack()
	r := &requestAborter{}
	lb := setLogBuffer()
	err.ToRequestAndLog(r)
	assert.Equal(t, "", lb.String())

	assert.Equal(t, 421, r.lastHTTPCode)
	expected := API(421, 80, "a safe error message")
	assert.Equal(t, expected, *r.lastError)
}

func TestToLogExcept(t *testing.T) {
	err := New("TestError").Msg("a safe error message").Trace().Make().StrCause("an unsafe cause").HTTPCode(500).ErrCode(42).Safe()
	r := &requestAborter{}
	lb := setLogBuffer()
	err.ToRequestAndLog(r, New("TestError").Make())
	str := lb.String()

	assert.False(t, strings.Contains(str, err.GetID()), "Log should not contain error id")
	assert.False(t, strings.Contains(str, err.Error()), "Log should not contain unsafe error message")
	assert.False(t, strings.Contains(str, err.SafeString()), "Log should not contain safe error message")
	assert.False(t, strings.Contains(str, "TestToLogExcept"), "Log should not contain stack trace")
}

func TestLogUntracked(t *testing.T) {
	err := New("TestError").Msg("a safe error message").Untrack().Make()
	lb := setLogBuffer()
	err.ToLog()
	assert.Equal(t, "", lb.String())
}

func TestForceLog(t *testing.T) {
	err := New("TestError").Msg("a safe error message").Make()
	lb := setLogBuffer()
	err.ForceLog()
	assert.True(t, strings.Contains(lb.String(), "a safe error message"))
}

func TestToRequestAndForceLog(t *testing.T) {
	err := New("TestError").Msg("a safe error message").API(421, 80).Make()
	r := &requestAborter{}
	lb := setLogBuffer()
	err.ToRequestAndForceLog(r)
	assert.True(t, strings.Contains(lb.String(), "a safe error message"))

	assert.Equal(t, 421, r.lastHTTPCode)
	expected := API(421, 80, "a safe error message")
	assert.Equal(t, expected, *r.lastError)
}

func TestForceLogUntrackedStackTrace(t *testing.T) {
	err := New("TestError").Msg("a safe error message").Trace().Make().Untrack()
	lb := setLogBuffer()
	err.ForceLog()
	str := lb.String()

	assert.True(t, strings.Contains(lb.String(), "a safe error message"))
	assert.True(t, strings.Contains(str, "TestForceLogUntrackedStackTrace"), "Log should contain stack trace")
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

type logBuffer struct {
	sb strings.Builder
}

func (lb *logBuffer) String() string {
	return lb.sb.String()
}

func (lb *logBuffer) Write(msg string, args ...interface{}) {
	lb.sb.WriteString(fmt.Sprintf(msg, args...))
}

func setLogBuffer() *logBuffer {
	lb := &logBuffer{strings.Builder{}}
	Logger = lb.Write
	return lb
}
