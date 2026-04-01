package pricecalculator

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRequest  = errors.New("invalid request")
	ErrInvalidPeriods  = errors.New("invalid pricing periods")
	ErrInvalidDuration = errors.New("invalid duration")
)

// CustomError provides a descriptive error message while wrapping an original error
type CustomError struct {
	Message string
	Wrapped error
}

func (e *CustomError) Error() string {
	return e.Message
}

func (e *CustomError) Unwrap() error {
	return e.Wrapped
}

// NewDurationError creates a descriptive duration error
func NewDurationError(format string, args ...interface{}) error {
	return &CustomError{
		Message: fmt.Sprintf(format, args...),
		Wrapped: ErrInvalidDuration,
	}
}

// NewPeriodsError creates a descriptive periods error
func NewPeriodsError(format string, args ...interface{}) error {
	return &CustomError{
		Message: fmt.Sprintf(format, args...),
		Wrapped: ErrInvalidPeriods,
	}
}

// NewRequestError creates a descriptive request error
func NewRequestError(format string, args ...interface{}) error {
	return &CustomError{
		Message: fmt.Sprintf(format, args...),
		Wrapped: ErrInvalidRequest,
	}
}


