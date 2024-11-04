package response

import (
	"fmt"

	"github.com/gookit/validate"
)

type Response struct {
	Successful bool        `json:"successful"`
	Messages   []string    `json:"messages"`
	Data       interface{} `json:"data,omitempty"`
}

func Succeed() *Response {
	return SucceedWithData(nil)
}

func SucceedWithMessages(messages ...string) *Response {
	return SucceedWithData(nil, messages...)
}

func SucceedWithData(data interface{}, messages ...string) *Response {
	msgs := messages
	if len(messages) == 0 {
		// so that it's marshalled as `[]` instead of `null`
		msgs = make([]string, 0)
	}

	return &Response{
		Successful: true,
		Data:       data,
		Messages:   msgs,
	}
}

func Fail(messages ...string) *Response {
	return &Response{
		Successful: false,
		Messages:   messages,
	}
}

func FailWithValidationErrors(errs validate.Errors) *Response {
	var messages []string
	for field, fieldErrors := range errs {
		for validator, err := range fieldErrors {
			messages = append(messages, fmt.Sprintf("%s: failed %s with error: %s", field, validator, err))
		}
	}
	return Fail(messages...)
}
