package exit

import "errors"

type Error struct {
	Code    Code
	Label   string
	Message string
}

var _ error = (*Error)(nil)

func (e *Error) Error() string {
	return e.Message
}

func New(code Code, label, message string) *Error {
	return &Error{Code: code, Label: label, Message: message}
}

func NewUsage(message string) *Error {
	return New(Usage, "usage", message)
}

func NewRefused(message string) *Error {
	return New(Refused, "refused", message)
}

func NewConflict(message string) *Error {
	return New(State, "conflict", message)
}

const internalLabel = "internal"

func ExitCodeFor(err error) int {
	return classify(err).Code.Status()
}

func classify(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return New(Usage, internalLabel, err.Error())
}
