package response

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator"
)

const (
	StatusOK    = "OK"
	StatusError = "Error"
)

type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func OK() Response {
	return Response{
		Status: StatusOK,
	}
}

func OKWithData(data interface{}) Response {
	return Response{
		Status: StatusOK,
		Data:   data,
	}
}

func Error(msg string) Response {
	return Response{
		Status: StatusError,
		Error:  msg,
	}
}

func ValidationError(errs validator.ValidationErrors) Response {
	var errMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMsgs = append(errMsgs, fmt.Sprintf("Field %s is a required field", err.Field()))
		default:
			errMsgs = append(errMsgs, fmt.Sprintf("Field %s is not valid", err.Field()))
		}
	}

	return Response{
		Status: StatusError,
		Error:  strings.Join(errMsgs, ", "),
	}
}
