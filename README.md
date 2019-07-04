# Errors for GO

This library introduces an advanced error type for golang. It's advantages are better error comparison for error handling and automated test, as well as a built-in distinction between safe error messages to be passed to an end-user (e.g. API client) and technical details that can be written to a log file for further investigation.

## Getting started

As this repository is compatible with go mod, it is sufficient to include the following package in your go code:

```
import "github.com/sbreitf1/errors"
```


## Usage

The main error type of this package is `Error` which is also compatible with the commonly used `error` interface. Every instance of `Error` is typically generated from a globally defined template using `Make()`:

```golang
import "github.com/sbreitf1/errors"

var (
    // template definition:
    ArgumentError = errors.New("Argument %s is not valid")
)

func FooBar(positiveValue int) errors.Error {
    if positiveValue < 0 {
        // instantiate error from template:
        return ArgumentError.Make().Args("positiveValue")
    }
    return nil
}
```

As can be seen in this example, templates can define format strings as consumed by `fmt.Sprintf()` that are evaluated by a later call to `Args()` supplying the content. You can also overwrite the whole message using `Msg()` on the generated error, but this will force the error message to be marked as unsafe.

A key element of this error type is the ability to define the *safeness* of error messages that specify which information can be displayed to API users without revealing critical secrets and implementation details. Call the `Safe()` mutator function after changing the error message via `Msg()` to allow printing the message in public contexts:

```golang
SafeArgumentError := errors.New("Argument %s is not valid").Safe()
```

Errors derived from this template will be safe and can be printed to public contexts. A call to `Args()` will maintain the safeness state as it only fills expected fields. Changing the message using `Msg()`, however, will remove the safeness-flag as stated above. A call to `SafeString()` will return the safe error message. If the error is not safe, a generic error message will be returned, including a unique id referring to this error instance. Printing the full error message `Error()` including stack trace and id to a log file allows for an indepth view without revealing details to the API client.


### Comparison

Another advantage of this error package is the advanced and unified typing system. When creating a new error template using the global function `New(ErrorType)` you must specify a string denoting the error type of the new error. This error type string is used for comparison regardless of the actual message content. This allows for detailed error messages that can be compared using the same mechanisms as generic errors:

```golang
FileNotFoundError := errors.New("File %s not found")

err1 := FileNotFoundError.Make().Args("foo.txt")
err1.Is(FileNotFoundError) // => true, is instance of template FileNotFoundError

err2 := FileNotFoundError.Make().Args("bar.jpg");
err2.Equals(err1) // => true, different error messages but same type "FileNotFoundError"
```

Alternatively you can use the global functions `AreEqual(error,error)` and `InstanceOf(error,Template)` for checking in cases where the values might be `nil`:

```golang
InstanceOf(err1, FileNotFoundError) // => true
AreEqual(err1, err2) // => true
```


### Logging and HTTP Responses

This package offers detailed logging and is compatible with the [Gin](https://github.com/gin-gonic/gin) framework for HTTP request handling. To use both functions in conjunction, you only need one call on a returned error object:

```golang
func handleRequest(c *gin.Context) {
    if err := someHandler(); err != nil {
        err.ToRequestAndLog(c)
        return
    }
}
```

The method `ToRequestAndLog(RequestAborter, ...TypedError)` executes both `ToLog(...TypedError)` and `ToRequest(RequestAborter)`. The interface `RequestAborter` specifies the method `AbortWithStatusJSON(int, interface{})` as used by the Gin framework for returning the given HTTP response code and JSON representation of a data object.

You can pass an arbitrary collection of errors and templates to `ToLog(...TypedError)` to specify which errors should be ignored. You may list functional errors here that should be reported to the API client but are not required in a log file. Furthermore, you can redirect logging by setting `errors.Logger` to an arbitrary function `(string, ...interface{})` to write to a custom logger.

If you carefully maintain the error flags and error propagation in your application code, you won't need any conditions here as `ToRequestAndLog` will consider all parameters when printing the error message to log and request.


### Mutator Functions

Mutator functions like `Msg()`, `Args()` and `Safe()` are used to change a specific property of the error. Every mutator function returns a new copy of `Error` allowing for a compact syntax. The following mutator functions are available on **templates**:

| Function | Effect |
| --- | --- |
| `Track()` | Generate id for this error and print message to log (default) |
| `Untrack()` | Disable automatic print to log. No id and stack trace will be generated for untracked errors |
| `Trace()` | Allow stack traces for this error. This also marks the error template as tracked |
| `NoTrace()` | Disallow stack traces for this error (default) |
| `Safe()` | Set the safeness flag for this error |
| `Msg(string, args...)` | Set the message for this error. If no args are supplied, the format string will be evaluated after a call to `Args(args...)` |
| `HTTPCode(int)` | Sets the HTTP response code for this error |
| `ErrCode(int)` | Sets the API error code for this error |
| `API(int, int)` | A shortcut for `.HTTPCode(int).ErrCode(int).Safe().Untrack()` often used for functional API errors |

Most of these methods are also available on **errors**. See the following list for a complete overview:

| Function | Effect |
| --- | --- |
| `Untrack()` | Remove id and stack trace from this error and disable automatic log printing |
| `NoTrace()` | Remove stack trace from this error |
| `Safe()` | Set the safeness flag for this error |
| `Msg(string, args...)` | Set the message for this error. If no args are supplied, the format string will be evaluated after a call to `Args(args...)` |
| `Args(args...)` | Pass the format arguments for a previous call to `Msg(string)` |
| `Cause(error)` | Saves a causing error as nested object in this error. Cause error strings will be appended to the error message |
| `StrCause(string, args...)` | Generates a new generic error with message and appends it as cause |
| `Expand(string, args...)` | Returns a copy of this error with the given error message and sets itself as cause |
| `ExpandSafe(string, args...)` | Returns a copy of this error with the given error message with safeness-flag and sets itself as cause |
| `HTTPCode(int)` | Sets the HTTP response code for this error |
| `ErrCode(int)` | Sets the API error code for this error |


### Interopability

Log output of the errors package can be processed by any method that accepts parameters like `fmt.Sprintf`. If you are using [Logrus](https://github.com/sirupsen/logrus) you can simply use the `Errorf` function as logger:

```golang
errors.Logger = logrus.Errorf
```
By default all errors are printed directly to StdOut.


## Best Practices

### Error instantiation

Always use globally defined error templates for error instantiation. You may also define format messages without passing arguments to delay filling in the actual values when they are available in the application context. Use `Make()` during execution to instantiate a new error and prepare id and stack trace at this location:

```golang
var (
    DatabaseError = errors.New("Database unreachable")
    ElementNotFoundError = errors.New("Did not find resource %s").API(404, 0)
)

func example() {
    dbErr := DatabaseError.Make()
    queryErr := ElementNotFoundError.Make().Args("foobar")
}
```

### Error propagation

Use `Cause(error)` to encapsulate a typical error object in the error model of this package:

```golang
var (
    ReadFileError = errors.New("Unable to read file %q")
)

func readData(file string) (string, errors.Error) {
    data, err := ioutil.ReadFile(file)
    if err != nil {
        return "", ReadFileError.Make().Args(file).Cause(err);
    }
    return string(data), nil
}
```

Then use `Expand(string, args...)` to propagate errors while maintaining the original error type:

```golang
func readResourceFile(relativePath string) (string, errors.Error) {
    data, err := readData(filepath.Join("data/resources", relativePath))
    if err != nil {
        return "", err.Expand("Could not read resource file")
    }
    return data, nil
}
```

Alternatively, you can use `Cause(error)` to propagate errors to semantically distinct steps:

```golang
var (
    ReadKeyError = errors.New("Unable to parse key")
)

func parseKey(file string) (*Key, errors.Error) {
    data, err := ioutil.ReadFile(file)
    if err != nil {
        return nil, ReadKeyError.Make().Cause(err)
    }
    [...]
}
```

For a fast way of propagating classical errors, you can use `errors.Wrap` that generates a new traced error using the input error type name:

```golang
func deleteFile(file string) (errors.Error) {
    return errors.Wrap(os.Remove(file))
}
```
As you can see, no special handling for `<nil>` is required here, as `errors.Wrap` will also return `<nil>` in this case.


## TL;DR

TODO: short overview