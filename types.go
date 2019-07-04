package errors

// ErrorType represents the base type of an error regardless of the specific error message.
type ErrorType string

func (e ErrorType) String() string {
	return string(e)
}

// RequestAborter defines the required functionality to abort an HTTP request and is compatible with *gin.Context.
type RequestAborter interface {
	AbortWithStatusJSON(int, interface{})
}

// TypedError represents errors and templates that define an error type.
type TypedError interface {
	// GetType returns the type of the error that is used for comparison.
	GetType() ErrorType
}

type content struct {
	message string
	cause   Error
}

type flags struct {
	track   bool
	trace   bool
	isSafe  bool
	tags    map[string]interface{}
	strTags map[string]string
	intTags map[string]int
}

type trace struct {
	id         string
	stackTrace string
}

type apiData struct {
	httpCode int
	errCode  int
}
