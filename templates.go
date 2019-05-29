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
	GenericError = New("GenericError").Msg("A generic error occured").WithStackTrace()
	// ConfigurationError an error that is caused by an invalid configuration.
	ConfigurationError = New("ConfigurationError").Msg("The specified configuration is not valid")
)

// Template represents an error template that can be instatiated to an error using Make().
type Template struct {
	errType ErrorType
	content content
	flags   flags
	api     apiData
}

// New returns an empty error template with the given error type.
func New(errType ErrorType) Template {
	content := content{message: "", cause: nil}
	flags := flags{untracked: false, withStackTrace: defaultWithStackTrace, noLog: false, isSafe: false}
	api := apiData{defaultHTTPCode, defaultErrCode}
	return Template{errType, content, flags, api}
}

// GetType returns the underlying error type of this template.
func (t Template) GetType() ErrorType {
	return t.errType
}

// Untracked disables id and stack trace printing for this error.
func (t Template) Untracked() Template {
	flags := t.flags
	flags.untracked = true
	return Template{t.errType, t.content, flags, t.api}
}

// WithStackTrace enables stack trace printing.
func (t Template) WithStackTrace() Template {
	flags := t.flags
	flags.withStackTrace = true
	return Template{t.errType, t.content, flags, t.api}
}

// WithoutStackTrace disables stack trace printing.
func (t Template) WithoutStackTrace() Template {
	flags := t.flags
	flags.withStackTrace = false
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

// Make instatiates an error using this template. A call to this method generates a new ID and StackTrace from the calling location.
func (t Template) Make() Error {
	trace := trace{generateID(t.errType, t.content.message), getStackTrace(1)}
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
