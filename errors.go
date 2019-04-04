package errors

import (
	"crypto/sha1"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	defaultHTTPCode = 500
	defaultErrCode  = 0
)

var (
	// PrintUnsafeErrors controls wether unsafe (technical) error messages should be visible to the user in response messages.
	PrintUnsafeErrors = false

	// GenericError represents a generic error without further information.
	GenericError = New("GenericError").Msg("A generic error occured").Template()
	// ConfigurationError an error that is caused by an invalid configuration.
	ConfigurationError = New("ConfigurationError").Msg("The specified configuration is not valid").Template()
)

// ErrorType represents the base type of an error regardless of the specific error message.
type ErrorType string

// RequestAborter defines the required functionality to abort an HTTP request and is compatible with *gin.Context.
type RequestAborter interface {
	AbortWithStatusJSON(int, interface{})
}

// Error is used as base error type in the whole application. Use Wrap(error) to encapsulate errors from third-party code.
type Error interface {
	error
	fmt.Stringer
	SafeString() string

	GetID() string
	GetStackTrace() string

	// Template marks the error as template and the next interaction will create the id and stack trace.
	Template() Error
	// Untracked disables id and stack trace printing for this error.
	Untracked() Error
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

	// GetType returns the type of the error that is used for comparison.
	GetType() ErrorType
	// Equals returns true when the error types are equal (ignoring the explicit error message).
	Equals(other error) bool

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
	ToRequestAndLog(r RequestAborter, except ...Error)

	// ToLog writes the error message with debug data to the log.
	ToLog(except ...Error)
}

type baseError struct {
	untracked  bool
	id         string
	stackTrace string
	errType    ErrorType
	message    string
	cause      Error
	httpCode   int
	errCode    int
	noLog      bool
	isSafe     bool
}

func (err baseError) getID() string {
	if len(err.id) > 0 {
		return err.id
	}
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%v|%v|%v", err.errType, err.message, time.Now())))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash[:8])
}

func (err baseError) GetID() string {
	return err.id
}
func (err baseError) GetStackTrace() string {
	return err.stackTrace
}

func (err baseError) getStackTrace(depth int) string {
	if len(err.stackTrace) > 0 {
		return err.stackTrace
	}
	//TOD rework using -> pc, file, line, ok := runtime.Caller(i)
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

func (err baseError) ToLog(except ...Error) {
	if !err.untracked {
		for _, exceptErr := range except {
			if AreEqual(err, exceptErr) {
				// do not print error as it is explicitly excluded
				return
			}
		}
		if len(err.id) > 0 {
			log.Errorf("[ERR %v] %v", err.id, err.Error())
		}
		if len(err.stackTrace) > 0 {
			log.Errorf("[STACK %v] %v", err.id, err.stackTrace)
		}
	}
}

func (err baseError) ToRequestAndLog(r RequestAborter, except ...Error) {
	err.ToLog(except...)
	err.ToRequest(r)
}

func (err baseError) Error() string {
	return err.String()
}
func (err baseError) String() string {
	return err.string(false)
}
func (err baseError) SafeString() string {
	return err.string(true)
}
func (err baseError) string(safe bool) string {
	if !safe || err.isSafe {
		var prefix string
		if err.message == "" {
			prefix = string(err.errType)
		} else {
			prefix = err.message
		}

		suffix := ""
		if err.cause != nil {
			if safe {
				suffix = err.cause.SafeString()
			} else {
				suffix = err.cause.String()
			}
			if len(suffix) > 0 {
				suffix = ": " + suffix
			}
		}

		return prefix + suffix
	}
	return ""
}

func (err baseError) Template() Error {
	return baseError{err.untracked, "", "", err.errType, err.message, err.cause, err.httpCode, err.errCode, err.noLog, false}
}
func (err baseError) Untracked() Error {
	return baseError{true, "", "", err.errType, err.message, err.cause, err.httpCode, err.errCode, err.noLog, false}
}
func (err baseError) Msg(msg string, args ...interface{}) Error {
	if len(args) == 0 {
		return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, msg, err.cause, err.httpCode, err.errCode, err.noLog, false}
	}
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, fmt.Sprintf(msg, args...), err.cause, err.httpCode, err.errCode, err.noLog, false}
}
func (err baseError) Args(args ...interface{}) Error {
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, fmt.Sprintf(err.message, args...), err.cause, err.httpCode, err.errCode, err.noLog, err.isSafe}
}
func (err baseError) Cause(cause error) Error {
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, err.message, Wrap(cause), err.httpCode, err.errCode, err.noLog, err.isSafe}
}
func (err baseError) StrCause(str string, args ...interface{}) Error {
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, err.message, GenericError.Msg(str, args...), err.httpCode, err.errCode, err.noLog, err.isSafe}
}
func (err baseError) Expand(msg string, args ...interface{}) Error {
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, fmt.Sprintf(msg, args...), err, err.httpCode, err.errCode, err.noLog, false}
}
func (err baseError) ExpandSafe(msg string, args ...interface{}) Error {
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, fmt.Sprintf(msg, args...), err, err.httpCode, err.errCode, err.noLog, true}
}

func (err baseError) GetType() ErrorType {
	return err.errType
}
func (err baseError) Equals(other error) bool {
	if other == nil {
		// err is obviously NOT nil, but other is...
		return false
	}

	return err.errType == getErrorType(other)
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

	return getErrorType(err1) == getErrorType(err2)
}

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

		err := baseError{false, "", "", errType, msg, nil, defaultHTTPCode, defaultErrCode, false, false}
		return baseError{err.untracked, err.getID(), err.getStackTrace(depth + 1), errType, msg, nil, defaultHTTPCode, defaultErrCode, false, false}
	}
}

// New creates a new Error with custom error type.
func New(errType ErrorType) Error {
	return baseError{false, "", "", errType, "", nil, defaultHTTPCode, defaultErrCode, false, false}
}

func getErrorType(err error) ErrorType {
	switch e := err.(type) {
	case Error:
		return e.GetType()
	default:
		return ErrorType(fmt.Sprintf("%T", err))
	}
}

// APIError represents a generic error repsonse object with code and message.
type APIError struct {
	ResponseCode int    `json:"-"`
	ErrorCode    int    `json:"code"`
	Message      string `json:"message"`
}

// ToRequest writes this APIError object to a HTTP request and aborts pipeline execution.
func (err APIError) ToRequest(r RequestAborter) {
	r.AbortWithStatusJSON(err.ResponseCode, err)
}

// API returns a new APIError object.
func API(httpCode, errCode int, message string) APIError {
	return APIError{httpCode, errCode, message}
}

// DefaultAPI returns a new APIError object using the default http and error codes.
func DefaultAPI(message string) APIError {
	return APIError{defaultHTTPCode, defaultErrCode, message}
}

func (err baseError) HTTPCode(code int) Error {
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, err.message, err.cause, code, err.errCode, err.noLog, err.isSafe}
}

func (err baseError) ErrCode(code int) Error {
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, err.message, err.cause, err.httpCode, code, err.noLog, err.isSafe}
}

func (err baseError) Safe() Error {
	return baseError{err.untracked, err.getID(), err.getStackTrace(1), err.errType, err.message, err.cause, err.httpCode, err.errCode, err.noLog, true}
}

func (err baseError) API() APIError {
	suffix := ""
	if !err.untracked && len(err.id) > 0 {
		suffix = " [ID " + err.id + "]"
	}

	if PrintUnsafeErrors {
		return APIError{err.httpCode, err.errCode, err.Error() + suffix}
	}
	if err.isSafe {
		return APIError{err.httpCode, err.errCode, err.SafeString() + suffix}
	}
	return APIError{err.httpCode, err.errCode, "An error occured" + suffix}
}

func (err baseError) ToRequest(r RequestAborter) {
	err.API().ToRequest(r)
}

// ToRequest writes the given error to a HTTP request and returns true if err was not nil.
func ToRequest(r RequestAborter, err error) bool {
	if err == nil {
		return false
	}
	Wrap(err).ToRequest(r)
	return true
}
