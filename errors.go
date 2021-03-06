package errors

import (
	"fmt"
)

const (
	defaultHTTPCode = 500
	defaultErrCode  = 0
)

var (
	// PrintUnsafeErrors controls wether unsafe (technical) error messages should be visible to the user in response messages.
	PrintUnsafeErrors = false

	// Logger is called to print errors and stack traces to log.
	Logger = DefaultStdOutLogger
)

// DefaultStdOutLogger prints all error messages to StdOut.
func DefaultStdOutLogger(msg string, args ...interface{}) {
	fmt.Printf(msg+"\n", args...)
}

/* ############################################# */
/* ###                 Error                 ### */
/* ############################################# */

// Error is used as base error type in the whole application. Use Wrap(error) to encapsulate errors from third-party code.
type Error interface {
	error
	TypedError
	fmt.Stringer

	SafeString() string

	GetID() string
	GetStackTrace() string

	// Untrack disables id and stack trace printing for this error.
	Untrack() Error
	// NoTrace disables stack trace printing.
	NoTrace() Error
	// Msg returns a new Error object and replaces the error message. You can supply all formatting args later using Args() to skip formatting in this call.
	Msg(msg string, args ...interface{}) Error
	// Args returns a new Error object with filled placeholders. A safe message remains safe.
	Args(args ...interface{}) Error
	// Cause adds the given error as cause. It's error message will be appended to the output.
	Cause(err error) Error
	// StrCause adds a detailed error message as cause.
	StrCause(str string, args ...interface{}) Error
	// Expand creates a copy of this error with given message and sets the current error as cause.
	Expand(msg string, args ...interface{}) Error
	// ExpandSafe creates a copy of this error with given message and sets the current error as cause. The expanded message is marked as safe.
	ExpandSafe(msg string, args ...interface{}) Error

	// Tag adds a named tag to the error.
	Tag(tag string) Error
	// IsTagged returns whether the error contains a named tag. Tags with attached value are not captured by this method.
	IsTagged(tag string) bool
	// TagStr adds a named tag with string value to the error.
	TagStr(tag, value string) Error
	// GetTagStr returns a string tag or false, if no tag is set.
	GetTagStr(tag string) (string, bool)
	// TagInt adds a named tag with integer value to the error.
	TagInt(tag string, value int) Error
	// GetTagInt returns an integer tag or false, if no tag is set.
	GetTagInt(tag string) (int, bool)

	// Equals returns true when the error types are equal (ignoring the explicit error message).
	Equals(other error) bool
	// Is returns trhe when the error is an instance of the given template.
	Is(template Template) bool

	// HTTPCode sets the http response code.
	HTTPCode(code int) Error
	// ErrCode sets the api error code.
	ErrCode(code int) Error
	// Safe marks the error as safe for printing to end-user.
	Safe() Error
	// API returns the corresponding APIError object.
	API() APIError
	// ToRequest writes the APIError message representation to a HTTP request and aborts pipeline execution.
	ToRequest(r RequestAborter)
	// ToRequestAndLog calls ToRequest(r) and ToLog(...except).
	ToRequestAndLog(r RequestAborter, except ...TypedError)
	// ToRequestAndLog calls ToRequest(r) and ForceLog(...except).
	ToRequestAndForceLog(r RequestAborter, except ...TypedError)

	// ToLog writes the error message with debug data to the log.
	ToLog(except ...TypedError)
	// ForceLog writes the error message (and also untracked ones) with debug data to the log.
	ForceLog(except ...TypedError)
}

type baseError struct {
	errType ErrorType
	content content
	flags   flags
	trace   trace
	api     apiData
}

func (err baseError) GetType() ErrorType {
	return err.errType
}
func (err baseError) GetID() string {
	return err.trace.id
}
func (err baseError) GetStackTrace() string {
	return err.trace.stackTrace
}

func (err baseError) IsTagged(tag string) bool {
	_, ok := err.flags.tags[tag]
	return ok
}
func (err baseError) GetTagStr(tag string) (string, bool) {
	if val, ok := err.flags.tags[tag]; ok {
		strVal, ok := val.(string)
		return strVal, ok
	}
	return "", false
}
func (err baseError) GetTagInt(tag string) (int, bool) {
	if val, ok := err.flags.tags[tag]; ok {
		intVal, ok := val.(int)
		return intVal, ok
	}
	return 0, false
}

/* ############################################# */
/* ###           Mutator Functions           ### */
/* ############################################# */

func (err baseError) Untrack() Error {
	flags := err.flags
	flags.track = false
	return baseError{err.errType, err.content, flags, err.trace, err.api}
}
func (err baseError) NoTrace() Error {
	flags := err.flags
	flags.trace = false
	return baseError{err.errType, err.content, flags, err.trace, err.api}
}
func (err baseError) Safe() Error {
	flags := err.flags
	flags.isSafe = true
	return baseError{err.errType, err.content, flags, err.trace, err.api}
}
func (err baseError) Msg(msg string, args ...interface{}) Error {
	content := err.content
	if len(args) == 0 {
		content.message = msg
	} else {
		// hack: go-vet erroneously detects missing args when calling Sprintf directly
		// -> using the encapsulation prevents go-vet from processing the format string
		content.message = fmt.Sprintf(fmt.Sprintf("%s", msg), args...)
	}
	flags := err.flags
	flags.isSafe = false
	return baseError{err.errType, content, flags, err.trace, err.api}
}
func (err baseError) Args(args ...interface{}) Error {
	content := err.content
	content.message = fmt.Sprintf(content.message, args...)
	return baseError{err.errType, content, err.flags, err.trace, err.api}
}
func (err baseError) Cause(cause error) Error {
	content := err.content
	content.cause = Wrap(cause)
	return baseError{err.errType, content, err.flags, err.trace, err.api}
}
func (err baseError) StrCause(str string, args ...interface{}) Error {
	content := err.content
	content.cause = GenericError.Msg(str, args...).Untrack().Make()
	return baseError{err.errType, content, err.flags, err.trace, err.api}
}
func (err baseError) Expand(msg string, args ...interface{}) Error {
	content := err.content
	content.message = fmt.Sprintf(msg, args...)
	content.cause = err
	flags := err.flags
	flags.isSafe = false
	return baseError{err.errType, content, flags, err.trace, err.api}
}
func (err baseError) ExpandSafe(msg string, args ...interface{}) Error {
	content := err.content
	content.message = fmt.Sprintf(msg, args...)
	content.cause = err
	flags := err.flags
	flags.isSafe = true
	return baseError{err.errType, content, flags, err.trace, err.api}
}

func (err baseError) Tag(tag string) Error {
	flags := err.flags
	flags.tags[tag] = nil
	return baseError{err.errType, err.content, flags, err.trace, err.api}
}

func (err baseError) TagStr(tag, value string) Error {
	flags := err.flags
	flags.tags[tag] = value
	return baseError{err.errType, err.content, flags, err.trace, err.api}
}

func (err baseError) TagInt(tag string, value int) Error {
	flags := err.flags
	flags.tags[tag] = value
	return baseError{err.errType, err.content, flags, err.trace, err.api}
}

func (err baseError) HTTPCode(code int) Error {
	api := err.api
	api.httpCode = code
	return baseError{err.errType, err.content, err.flags, err.trace, api}
}

func (err baseError) ErrCode(code int) Error {
	api := err.api
	api.errCode = code
	return baseError{err.errType, err.content, err.flags, err.trace, api}
}

/* ############################################# */
/* ###              Comparison               ### */
/* ############################################# */

func (err baseError) Equals(other error) bool {
	if other == nil {
		// err is obviously NOT nil, but other is...
		return false
	}

	return err.errType == getErrorType(other)
}
func (err baseError) Is(template Template) bool {
	return err.errType == template.GetType()
}

// AreEqual returns true if the type of both errors is the same regardless of the specific error message. Also returns true if both errors are nil.
func AreEqual(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		// both are nil
		return true
	} else if err1 == nil || err2 == nil {
		// one is nil, the other not
		return false
	}

	return areEqual(getErrorType(err1), getErrorType(err2))
}

func areEqual(type1, type2 ErrorType) bool {
	return type1 == type2
}

// InstanceOf returns true if the given error is an instance of the given template. A nil error always returns false.
func InstanceOf(err error, template Template) bool {
	if err == nil {
		return false
	}

	return getErrorType(err) == template.GetType()
}

/* ############################################# */
/* ###             Instantiation             ### */
/* ############################################# */

// Wrap encapsulates any go-error in the extended Error type. Returns nil if baseErr is nil.
func Wrap(baseErr error) Error {
	return wrap(baseErr, false, 1)
}

// WrapT encapsulates any go-error in the extended Error type and appends the base error type to the message. Returns nil if baseErr is nil.
func WrapT(baseErr error) Error {
	return wrap(baseErr, true, 1)
}

func wrap(baseErr error, withType bool, depth int) Error {
	if baseErr == nil {
		// do not generate Error out of nowhere...
		return nil
	}

	switch e := baseErr.(type) {
	case Error:
		// do not further wrap Error interface
		return e
	default:
		errType := getErrorType(baseErr)

		msg := baseErr.Error()
		if withType {
			if msg != "" {
				msg = "[" + string(errType) + "] " + msg
			} else {
				msg = string(errType)
			}
		}

		return New(string(errType)).Msg(msg).Trace().make(depth + 1)
	}
}

func getErrorType(err error) ErrorType {
	switch e := err.(type) {
	case Error:
		return e.GetType()
	default:
		return ErrorType(fmt.Sprintf("%T", err))
	}
}

/* ############################################# */
/* ###             Error Output              ### */
/* ############################################# */

func (err baseError) Error() string {
	return err.String()
}
func (err baseError) String() string {
	return err.string(false)
}
func (err baseError) SafeString() string {
	return err.string(true)
}
func (err baseError) string(onlySafe bool) string {
	if !onlySafe || err.flags.isSafe {
		var prefix string
		if err.content.message == "" {
			prefix = string(err.errType)
		} else {
			prefix = err.content.message
		}

		suffix := ""
		if err.content.cause != nil {
			if onlySafe {
				suffix = err.content.cause.SafeString()
			} else {
				suffix = err.content.cause.String()
			}
			if len(suffix) > 0 {
				suffix = ": " + suffix
			}
		}

		return prefix + suffix
	}
	return ""
}

func (err baseError) ToRequestAndLog(r RequestAborter, except ...TypedError) {
	err.ToLog(except...)
	err.ToRequest(r)
}

func (err baseError) ToRequestAndForceLog(r RequestAborter, except ...TypedError) {
	err.ForceLog(except...)
	err.ToRequest(r)
}

func (err baseError) ToRequest(r RequestAborter) {
	err.API().ToRequest(r)
}

func (err baseError) ToLog(except ...TypedError) {
	if err.flags.track {
		err.toLog(except...)
	}
}

func (err baseError) ForceLog(except ...TypedError) {
	err.toLog(except...)
}

func (err baseError) toLog(except ...TypedError) {
	for _, exceptErr := range except {
		if areEqual(err.errType, exceptErr.GetType()) {
			// do not print error as it is explicitly excluded
			return
		}
	}
	if len(err.trace.id) > 0 {
		if !err.flags.track {
			Logger("[ERR] %v", err.Error())
		} else {
			Logger("[ERR %v] %v", err.trace.id, err.Error())
		}
	}
	if err.flags.trace && len(err.trace.stackTrace) > 0 {
		if !err.flags.track {
			Logger("[STACK] %v", err.trace.stackTrace)
		} else {
			Logger("[STACK %v] %v", err.trace.id, err.trace.stackTrace)
		}
	}
}
