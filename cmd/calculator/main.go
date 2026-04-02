// calculator is a language-agnostic CLI binary for the price calculator.
//
// Usage:
//
//	echo '<json>' | calculator                           # stdin
//	calculator -d '<json>'                               # JSON string flag
//	calculator -f request.json                           # JSON file flag
//	calculator -f request.json -o result.json            # file input and output
//
// Input  – JSON object on stdin/flag (see CalculateRequest)
// Output – JSON object on stdout (see CalculateResult), or {"error":"..."} on failure
// Exit   – 0 on success, 1 on error
//
// Example input:
//
//	{
//	  "duration": 150,
//	  "mode": "RoundUp",
//	  "periods": [
//	    {"duration": 60,  "price": 1000},
//	    {"duration": 120, "price": 1800},
//	    {"duration": 180, "price": 2500}
//	  ]
//	}
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"price_calculator/pkg/pricecalculator"
	"strings"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeErrorToStderr(msg string) error {
	return json.NewEncoder(os.Stderr).Encode(errorResponse{Error: msg})
}

func writeErrorToFile(filename string, msg string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(file).Encode(errorResponse{Error: msg}); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("encode error: %v; close error: %w", err, closeErr)
		}
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func reportError(msg string, outputFile string) {
	if err := writeErrorToStderr(msg); err != nil {
		// Nothing safer to do if writing to stderr itself fails.
	}
	if outputFile != "" {
		if err := writeErrorToFile(outputFile, msg); err != nil {
			if stderrErr := writeErrorToStderr(fmt.Sprintf("failed to write error file: %s", err)); stderrErr != nil {
				// Nothing safer to do if writing to stderr itself fails.
			}
		}
	}
}

func main() {
	dataFlag := flag.String("d", "", "JSON request string")
	fileFlag := flag.String("f", "", "JSON request file")
	outputFlag := flag.String("o", "", "JSON output file (if not set, output to stdout)")
	flag.Parse()

	var inputData io.Reader

	// Determine input source: -d flag > -f flag > stdin
	if *dataFlag != "" && *fileFlag != "" {
		reportError("cannot use both -d and -f flags", *outputFlag)
		os.Exit(1)
	}

	if *dataFlag != "" {
		inputData = strings.NewReader(*dataFlag)
	} else if *fileFlag != "" {
		// File argument provided
		content, err := os.ReadFile(*fileFlag)
		if err != nil {
			reportError(fmt.Sprintf("failed to open file: %s", err), *outputFlag)
			os.Exit(1)
		}
		inputData = bytes.NewReader(content)
	} else {
		// Default to stdin
		inputData = os.Stdin
	}

	var req pricecalculator.CalculateRequest
	if err := json.NewDecoder(inputData).Decode(&req); err != nil {
		reportError(fmt.Sprintf("invalid input JSON: %s", err), *outputFlag)
		os.Exit(1)
	}

	calc := pricecalculator.NewCalculator()
	result, err := calc.Calculate(req)
	if err != nil {
		reportError(err.Error(), *outputFlag)
		os.Exit(1)
	}

	// Write output to file or stdout
	if *outputFlag != "" {
		file, err := os.Create(*outputFlag)
		if err != nil {
			reportError(fmt.Sprintf("failed to create output file: %s", err), *outputFlag)
			os.Exit(1)
		}

		if err := json.NewEncoder(file).Encode(result); err != nil {
			if closeErr := file.Close(); closeErr != nil {
				reportError(fmt.Sprintf("failed to close output file: %s", closeErr), *outputFlag)
				os.Exit(1)
			}
			reportError(fmt.Sprintf("failed to encode result: %s", err), *outputFlag)
			os.Exit(1)
		}

		if err := file.Close(); err != nil {
			reportError(fmt.Sprintf("failed to close output file: %s", err), *outputFlag)
			os.Exit(1)
		}
	} else {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			if stderrErr := writeErrorToStderr(fmt.Sprintf("failed to encode result: %s", err)); stderrErr != nil {
				// Nothing safer to do if writing to stderr itself fails.
			}
			os.Exit(1)
		}
	}
}
