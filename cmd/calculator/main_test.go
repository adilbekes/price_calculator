package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = old
	})

	fn()
	require.NoError(t, w.Close())

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(b)
}

func captureStdout(t *testing.T, fn func()) string {
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

func resetFlagsForMain(t *testing.T, args []string) {
	t.Helper()
	oldArgs := os.Args
	oldFlags := flag.CommandLine

	os.Args = args
	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlags
	})
}

func TestWriteErrorToStderr_WritesJSON(t *testing.T) {
	stderr := captureStderr(t, func() {
		err := writeErrorToStderr("boom")
		require.NoError(t, err)
	})

	var resp errorResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.Equal(t, "boom", resp.Error)
}

func TestWriteErrorToFile_WritesJSON(t *testing.T) {
	out := filepath.Join(t.TempDir(), "error.json")
	require.NoError(t, writeErrorToFile(out, "broken"))

	b, err := os.ReadFile(out)
	require.NoError(t, err)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(b, &resp))
	assert.Equal(t, "broken", resp.Error)
}

func TestWriteErrorToFile_InvalidPath_ReturnsError(t *testing.T) {
	invalid := filepath.Join(t.TempDir(), "missing", "error.json")
	err := writeErrorToFile(invalid, "broken")
	require.Error(t, err)
}

func TestReportError_WritesToStderrAndOutputFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "error.json")
	stderr := captureStderr(t, func() {
		reportError("failed", out)
	})

	assert.Contains(t, stderr, "failed")

	b, err := os.ReadFile(out)
	require.NoError(t, err)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(b, &resp))
	assert.Equal(t, "failed", resp.Error)
}

func TestReportError_WhenFileWriteFails_ReportsSecondaryError(t *testing.T) {
	invalid := filepath.Join(t.TempDir(), "missing", "error.json")
	stderr := captureStderr(t, func() {
		reportError("primary", invalid)
	})

	assert.Contains(t, stderr, "primary")
	assert.Contains(t, stderr, "failed to write error file")
}

func TestMain_WithDataFlag_WritesResultToStdout(t *testing.T) {
	resetFlagsForMain(t, []string{
		"calculator",
		"-d",
		`{"duration":60,"mode":"RoundUp","periods":[{"duration":60,"price":1000}]}`,
	})

	stdout := captureStdout(t, func() {
		main()
	})

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	assert.Equal(t, float64(1000), got["total"])
	assert.Equal(t, float64(60), got["covered"])
}

func TestMain_WithDataAndOutputFile_WritesResultFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "result.json")
	resetFlagsForMain(t, []string{
		"calculator",
		"-d",
		`{"duration":60,"mode":"RoundUp","periods":[{"duration":60,"price":1000}]}`,
		"-o",
		out,
	})

	main()

	b, err := os.ReadFile(out)
	require.NoError(t, err)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, float64(1000), got["total"])
}

func TestMain_ExitsWhenBothInputFlagsProvided(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessCalculatorMain", "--", "-d", `{}`, "-f", "x.json")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS_CALCULATOR=1")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, string(out), "cannot use both -d and -f flags")
}

func TestMain_ExitsOnInvalidJSONInput(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessCalculatorMain", "--", "-d", `{invalid`)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS_CALCULATOR=1")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, string(out), "invalid input JSON")
}

func TestHelperProcessCalculatorMain(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_CALCULATOR") != "1" {
		return
	}

	sep := -1
	for i, arg := range os.Args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep == -1 {
		os.Exit(2)
	}

	args := append([]string{"calculator"}, os.Args[sep+1:]...)
	os.Args = args
	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	main()
	os.Exit(0)
}

func TestMain_ExitsOnMissingFileInput(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessCalculatorMain", "--", "-f", "no_such_input.json")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS_CALCULATOR=1")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.True(t, strings.Contains(string(out), "failed to open file"))
}

