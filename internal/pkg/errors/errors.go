package errors

import "fmt"

type level int

const (
	warn  level = iota + 1 // Warning severity, this error level can be see by server or client
	err                    // Error severity, this error level can be see only by server
	panic                  // Panic severity, application will crashed
)

type Error struct {
	msg      error
	severity level
}

func Nil() Error {
	return Error{}
}

func (e Error) IsError() bool {
	if e.msg != nil {
		return true
	}
	return false
}

func (e Error) Error() error {
	return e.msg
}

func (e Error) String() string {
	if e.msg == nil {
		return ""
	}
	return e.msg.Error()
}

// IsW Is Warning
func (e Error) IsW() bool {
	if e.severity == warn {
		return true
	}
	return false
}

// IsE Is Error
func (e Error) IsE() bool {
	if e.severity == err {
		return true
	}
	return false
}

// IsP Is Panic
func (e Error) IsP() bool {
	if e.severity == panic {
		return true
	}
	return false
}

func (e *Error) SetPrefix(prefix string) {
	if e.msg != nil {
		e.msg = fmt.Errorf(prefix + ": " + e.msg.Error())
	}
}

// W Warning
func W(e error) Error {
	if e == nil {
		return Nil()
	}
	return Error{
		msg:      e,
		severity: warn,
	}
}

// E Error
func E(e error) Error {
	if e == nil {
		return Nil()
	}
	return Error{
		msg:      e,
		severity: err,
	}
}

// P Panic
func P(e error) Error {
	if e == nil {
		return Nil()
	}
	return Error{
		msg:      e,
		severity: panic,
	}
}

// WF Warning from formatted string
func WF(format string, a ...any) Error {
	return Error{
		msg:      fmt.Errorf(format, a...),
		severity: warn,
	}
}

// EF Error from formatted string
func EF(format string, a ...any) Error {
	return Error{
		msg:      fmt.Errorf(format, a...),
		severity: err,
	}
}

// PF Panic from formatted string
func PF(format string, a ...any) Error {
	return Error{
		msg:      fmt.Errorf(format, a...),
		severity: panic,
	}
}
