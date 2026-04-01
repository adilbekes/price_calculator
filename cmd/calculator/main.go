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

func writeError(msg string) {
	_ = json.NewEncoder(os.Stdout).Encode(errorResponse{Error: msg})
}

func writeErrorToStderr(msg string) {
	_ = json.NewEncoder(os.Stderr).Encode(errorResponse{Error: msg})
}

func writeErrorToFile(filename string, msg string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(errorResponse{Error: msg})
}

func reportError(msg string, outputFile string) {
	writeErrorToStderr(msg)
	if outputFile != "" {
		_ = writeErrorToFile(outputFile, msg)
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
		file, err := os.Open(*fileFlag)
		if err != nil {
			reportError(fmt.Sprintf("failed to open file: %s", err), *outputFlag)
			os.Exit(1)
		}
		defer file.Close()
		inputData = file
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
		defer file.Close()
		if err := json.NewEncoder(file).Encode(result); err != nil {
			reportError(fmt.Sprintf("failed to encode result: %s", err), *outputFlag)
			os.Exit(1)
		}
	} else {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			writeErrorToStderr(fmt.Sprintf("failed to encode result: %s", err))
			os.Exit(1)
		}
	}
}
