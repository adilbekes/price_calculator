package pricecalculator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomError_ErrorReturnsFormattedMessage(t *testing.T) {
	err := NewRequestError("bad field: %s", "start_time")
	assert.Equal(t, "bad field: start_time", err.Error())
}

func TestNewDurationError_WrapsErrInvalidDuration(t *testing.T) {
	err := NewDurationError("too short")
	assert.ErrorIs(t, err, ErrInvalidDuration)
	assert.NotErrorIs(t, err, ErrInvalidRequest)
	assert.NotErrorIs(t, err, ErrInvalidPeriods)
}

func TestNewPeriodsError_WrapsErrInvalidPeriods(t *testing.T) {
	err := NewPeriodsError("no periods")
	assert.ErrorIs(t, err, ErrInvalidPeriods)
	assert.NotErrorIs(t, err, ErrInvalidDuration)
	assert.NotErrorIs(t, err, ErrInvalidRequest)
}

func TestNewRequestError_WrapsErrInvalidRequest(t *testing.T) {
	err := NewRequestError("bad param")
	assert.ErrorIs(t, err, ErrInvalidRequest)
	assert.NotErrorIs(t, err, ErrInvalidDuration)
	assert.NotErrorIs(t, err, ErrInvalidPeriods)
}

func TestCustomError_Unwrap_AllowsStandardErrorsAs(t *testing.T) {
	err := NewDurationError("wrapped")
	var ce *CustomError
	require.True(t, errors.As(err, &ce))
	assert.Equal(t, ErrInvalidDuration, ce.Unwrap())
}

