// Package errs provides support for errors related to this app.
package errs

import (
	"fmt"
	"runtime"
)

// Error represents an error inside the application
type Error struct {
	Code     int               `json:"code"`
	Message  string            `json:"message"`
	FuncName string            `json:"-"`
	FileName string            `json:"-"`
	Fields   map[string]string `json:"fields,omitempty"`
}

func New(code int, format string, args ...any) error {
	//skip 1 frame and get info about the whatever calls "New".
	// when only need info about single stack frame, otherwise for full stack frames use "runtime.Callers()"
	pc, filename, line, _ := runtime.Caller(1)
	return &Error{
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
		FuncName: runtime.FuncForPC(pc).Name(),
		FileName: fmt.Sprintf("%s:%d", filename, line),
	}
}

func (er *Error) Error() string {
	return er.Message
}
