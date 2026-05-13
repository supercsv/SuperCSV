// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	validate "github.com/supercsv/supercsv/internal"
	v1_0 "github.com/supercsv/supercsv/internal/v1_0/validate"
	verrors "github.com/supercsv/supercsv/internal/v1_0/validate/errors"
	"github.com/supercsv/supercsv/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("supercsv-validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		showVersion   bool
		quiet         bool
		maxFieldSize  int
		maxErrors     int
		jsonOutput    bool
		strictVersion string
	)
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	fs.BoolVar(&quiet, "quiet", false, "suppress per-error output")
	fs.BoolVar(&quiet, "q", false, "alias for --quiet")
	fs.IntVar(&maxFieldSize, "max-field-size", 0, "maximum field size in bytes (0 = unlimited)")
	fs.IntVar(&maxErrors, "max-errors", 0, "stop after N errors (0 = unlimited)")
	fs.BoolVar(&jsonOutput, "json", false, "emit newline-delimited JSON errors (default: supr format)")
	fs.StringVar(&strictVersion, "strict-version", "", "require a specific SuperCSV version header")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if showVersion {
		fmt.Fprintf(stdout, "supercsv-validate %s\n", version.Version)
		return 0
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "usage: supercsv-validate [flags] file.supr")
		fs.PrintDefaults()
		return 2
	}
	if maxErrors < 0 {
		maxErrors = 0
	}

	// Default to SUPR format, switch to JSON if --json flag used
	format := "supr"
	if jsonOutput {
		format = "json"
	}

	path := fs.Arg(0)
	var out formatter
	if !quiet {
		var ferr error
		out, ferr = newFormatter(format, stdout)
		if ferr != nil {
			fmt.Fprintf(stderr, "supercsv-validate: %v\n", ferr)
			return 2
		}
	}
	// Use v1.0 validator with strictVersion support
	validator := v1_0.NewFileValidator(maxFieldSize, maxErrors, strictVersion)
	result, err := validator.ValidateFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "supercsv-validate: %v\n", err)
		return 2
	} // Output errors if requested
	if out != nil {
		for _, verr := range result.Errors {
			if ferr := out.Write(verr); ferr != nil {
				fmt.Fprintf(stderr, "supercsv-validate: %v\n", ferr)
				return 2
			}
		}
		if ferr := out.Flush(); ferr != nil {
			fmt.Fprintf(stderr, "supercsv-validate: %v\n", ferr)
			return 2
		}
	}

	// Convert v1.0 result to standard format for printSummary
	summaryResult := &validate.Result{
		Errors:    result.Errors,
		Truncated: maxErrors > 0 && len(result.Errors) >= maxErrors,
	}

	printSummary(stderr, path, summaryResult)
	// Check if any errors are fatal
	hasFatal := false
	for _, verr := range result.Errors {
		if verr.Fatal {
			hasFatal = true
			break
		}
	}
	if hasFatal {
		return 2
	}
	if len(result.Errors) > 0 {
		return 1
	}
	return 0
}

type formatter interface {
	Write(*verrors.ValidationError) error
	Flush() error
}

func newFormatter(format string, w io.Writer) (formatter, error) {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		return &jsonFormatter{enc: enc}, nil
	case "supr":
		// NOTE: This uses encoding/csv rather than the SuperCSV encoder deliberately.
		// The error report is a simple 3-column diagnostic stream, not a full .supr file.
		// Keeping it independent of the encoder avoids circularity (encoder bug -> broken
		// error output) and keeps the CLI's output consistent even when debugging the
		// encoder itself. Dogfooding the encoder would be nice, but this isn't the right
		// place for it.
		return &suprFormatter{w: csv.NewWriter(w)}, nil
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

type jsonFormatter struct {
	enc *json.Encoder
}

func (j *jsonFormatter) Write(err *verrors.ValidationError) error {
	// External error format has 3 fields only (Line, ErrorSection, ErrorMsg).
	// Error codes are internal; not exposed in public output.
	payload := struct {
		Line         int    `json:"line"`
		ErrorSection string `json:"errorSection"`
		ErrorMsg     string `json:"errorMsg"`
	}{
		Line:         err.Row,
		ErrorSection: err.Column,
		ErrorMsg:     err.Msg,
	}
	return j.enc.Encode(payload)
}

func (j *jsonFormatter) Flush() error {
	return nil
}

type suprFormatter struct {
	w           *csv.Writer
	wroteHeader bool
}

func (s *suprFormatter) Write(err *verrors.ValidationError) error {
	if err := s.ensureHeader(); err != nil {
		return err
	}
	line := strconv.Itoa(err.Row)
	errorSection := err.Column
	errorMsg := err.Msg

	record := []string{
		line,
		errorSection,
		errorMsg,
	}
	return s.w.Write(record)
}

func (s *suprFormatter) Flush() error {
	if err := s.ensureHeader(); err != nil {
		return err
	}
	s.w.Flush()
	return s.w.Error()
}

func (s *suprFormatter) ensureHeader() error {
	if s.wroteHeader {
		return nil
	}
	if err := s.w.Write([]string{"Line:int", "ErrorSection:string", "ErrorMsg:string"}); err != nil {
		return err
	}
	s.wroteHeader = true
	return nil
}

func printSummary(w io.Writer, path string, result *validate.Result) {
	if len(result.Errors) == 0 {
		fmt.Fprintf(w, "%s: OK\n", path)
		return
	}

	// Count actual validation errors (excluding fatal "max errors reached" error)
	errorCount := len(result.Errors)
	for _, err := range result.Errors {
		if err.Fatal && err.Code == "max_errors_reached" {
			errorCount--
			break
		}
	}

	fmt.Fprintf(w, "%s: %d validation error(s)\n", path, errorCount)
	if result.Truncated {
		fmt.Fprintf(w, "%s: stopped after %d errors (--max-errors)\n", path, errorCount)
	}
}
