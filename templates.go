package errors

import (
	"crypto/sha1"
	"fmt"
	"runtime/debug"
	"strings"
	"time"
)

var (
	// GenericError represents a generic error with stack trace.
	GenericError = New("An error occured").Trace()
	// ConfigurationError is an error that is caused by an invalid configuration.
	ConfigurationError = New("The specified configuration is not valid")
	// ArgumentError denotes a missing or invalid argument.
	ArgumentError = New("An invalid argument has been supplied")
)

// Template represents an error template that can be instatiated to an error using Make().
type Template struct {
	errType ErrorType
	content content
	flags   flags
	api     apiData
}

// New returns an error template and uses the message format string as error type.
func New(msg string, args ...interface{}) Template {
	content := content{message: msg, cause: nil}
	if len(args) > 0 {
		// hack: go-vet erroneously detects missing args when calling Sprintf directly
		// -> using the encapsulation prevents go-vet from processing the format string
		content.message = fmt.Sprintf(fmt.Sprintf("%s", msg), args...)
	}
	flags := flags{track: true, trace: false, isSafe: false}
	api := apiData{defaultHTTPCode, defaultErrCode}
	return Template{ErrorType(msg), content, flags, api}
}

// GetType returns the underlying error type of this template.
func (t Template) GetType() ErrorType {
	return t.errType
}

// Track enables id printing for this error.
func (t Template) Track() Template {
	flags := t.flags
	flags.track = true
	return Template{t.errType, t.content, flags, t.api}
}

// Untrack disabled id and stack trace printing for this error.
func (t Template) Untrack() Template {
	flags := t.flags
	flags.track = false
	flags.trace = false
	return Template{t.errType, t.content, flags, t.api}
}

// Trace enables stack trace printing.
func (t Template) Trace() Template {
	flags := t.flags
	flags.track = true
	flags.trace = true
	return Template{t.errType, t.content, flags, t.api}
}

// NoTrace disables stack trace printing.
func (t Template) NoTrace() Template {
	flags := t.flags
	flags.trace = false
	return Template{t.errType, t.content, flags, t.api}
}

// Safe marks the error as safe for printing to end-user.
func (t Template) Safe() Template {
	flags := t.flags
	flags.isSafe = true
	return Template{t.errType, t.content, flags, t.api}
}

// Msg replaces the error message. You can supply all formatting args later using Args() to skip formatting in this call.
func (t Template) Msg(msg string, args ...interface{}) Template {
	content := t.content
	if len(args) == 0 {
		content.message = msg
	} else {
		// hack: go-vet erroneously detects missing args when calling Sprintf directly
		// -> using the encapsulation prevents go-vet from processing the format string
		content.message = fmt.Sprintf(fmt.Sprintf("%s", msg), args...)
	}
	return Template{t.errType, content, t.flags, t.api}
}

// Args fills the message placeholders with the given arguments.
func (t Template) Args(args ...interface{}) Template {
	content := t.content
	content.message = fmt.Sprintf(content.message, args...)
	return Template{t.errType, content, t.flags, t.api}
}

// API untracks the error, marks it as safe and update the error and response codes.
func (t Template) API(httpCode, errCode int) Template {
	flags := t.flags
	flags.track = false
	flags.trace = false
	flags.isSafe = true
	api := t.api
	api.httpCode = httpCode
	api.errCode = errCode
	return Template{t.errType, t.content, flags, api}
}

// HTTPCode sets the http response code.
func (t Template) HTTPCode(code int) Template {
	api := t.api
	api.httpCode = code
	return Template{t.errType, t.content, t.flags, api}
}

// ErrCode sets the api error code.
func (t Template) ErrCode(code int) Template {
	api := t.api
	api.errCode = code
	return Template{t.errType, t.content, t.flags, api}
}

// Make instatiates an error using this template. A call to this method generates a new ID and StackTrace from the calling location if tracked and traced.
func (t Template) Make() Error {
	return t.make(1)
}

// MakeTraced instatiates an error using this template. A call to this method tracks and traces the error and generates a new ID and StackTrace from the calling location. Use the depth parameter to skip a certain number of stack frames in the trace.
func (t Template) MakeTraced(depth int) Error {
	return t.make(depth + 1)
}

func (t Template) make(depth int) Error {
	trace := trace{generateID(t.errType, t.content.message), getStackTrace(depth + 1)}
	return baseError{t.errType, t.content, t.flags, trace, t.api}
}

func generateID(errType ErrorType, message string) string {
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%v|%v|%v", errType, message, time.Now())))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash[:8])
}

func getStackTrace(depth int) string {
	//TODO rework using -> pc, file, line, ok := runtime.Caller(i)
	fullTrace := string(debug.Stack())
	lines := strings.Split(fullTrace, "\n")
	var sb strings.Builder
	if len(lines) > 0 {
		// first line contains information on the executing goroutine
		sb.WriteString(lines[0])
		// skip this frame (getStackTrace) and the internal ones denoted by depth
		// every frame consists of two lines in the stack trace
		for i := 1 + 2*(depth+2); i < len(lines); i++ {
			sb.WriteString("\n")
			sb.WriteString(lines[i])
		}
	}
	return sb.String()
}
