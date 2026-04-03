package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureDemoStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = old
	})

	fn()
	require.NoError(t, w.Close())

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(b)
}

func TestMain_PrintsAllPricingModesAndSummary(t *testing.T) {
	out := captureDemoStdout(t, func() {
		main()
	})

	assert.Contains(t, out, "Requested duration: 130 minutes")
	assert.Contains(t, out, "Periods: 60")
	assert.True(t, strings.Contains(out, "Mode: RoundUp"))
	assert.True(t, strings.Contains(out, "Mode: ProrateMinimum"))
	assert.True(t, strings.Contains(out, "Mode: ProrateAny"))
	assert.True(t, strings.Contains(out, "Mode: RoundUpMinimumAndProrateAny"))
	assert.Contains(t, out, "Total price")
	assert.Contains(t, out, "Covered minutes")
	assert.Contains(t, out, "Breakdown")
}

