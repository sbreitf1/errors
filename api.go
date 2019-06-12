package errors

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

func (err baseError) API() APIError {
	suffix := ""
	if err.flags.track && len(err.trace.id) > 0 {
		suffix = " [ID " + err.trace.id + "]"
	}

	if PrintUnsafeErrors {
		return APIError{err.api.httpCode, err.api.errCode, err.Error() + suffix}
	}
	if err.flags.isSafe {
		return APIError{err.api.httpCode, err.api.errCode, err.SafeString() + suffix}
	}
	return APIError{err.api.httpCode, err.api.errCode, "An error occured" + suffix}
}

// ToRequest writes the given error to a HTTP request and returns true if err was not nil.
func ToRequest(r RequestAborter, err error) bool {
	if err == nil {
		return false
	}
	Wrap(err).ToRequest(r)
	return true
}
