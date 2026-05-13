// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

// Package validate provides a version router for SuperCSV validators.
// This file routes validation requests to the appropriate versioned validator.
// Currently routes all requests to v1_0 (spec v1.0 implementation).
package validate

import (
	v1_0 "github.com/supercsv/supercsv/internal/v1_0/validate"
	verrors "github.com/supercsv/supercsv/internal/v1_0/validate/errors"
)

// Sink receives every validation error as it is discovered. Returning a non-nil
// error aborts validation and surfaces the sink failure to the caller.
type Sink func(*verrors.ValidationError) error

// Options configures validator behavior for ValidateFile.
type Options struct {
	MaxFieldSize  int // Maximum field size in bytes (0 = unlimited)
	MaxErrors     int
	StrictVersion string
	Sink          Sink
}

// Result captures the set of validation errors discovered while scanning a file.
type Result struct {
	Errors    []*verrors.ValidationError
	Truncated bool
}

// ValidateFile validates a SuperCSV file at the given path and returns every
// validation error discovered (subject to MaxErrors). Fatal errors (CSV parse,
// header, version) are returned in result.Errors with Fatal=true.
// Only true system errors (file I/O) are returned as Go errors.
//
// This is a convenience wrapper around the V1.0 streaming validator.
func ValidateFile(path string, opts Options) (*Result, error) {
	validator := v1_0.NewFileValidator(opts.MaxFieldSize, opts.MaxErrors, opts.StrictVersion)
	result, err := validator.ValidateFile(path)
	if err != nil {
		// Only true system errors (file not found, etc.)
		return nil, err
	}

	// Convert V1.0 result to package result
	// Note: V1.0 doesn't support Sink callback pattern, collects all errors
	var errors []*verrors.ValidationError
	if opts.Sink != nil {
		for _, verr := range result.Errors {
			if err := opts.Sink(verr); err != nil {
				return nil, err
			}
		}
	}
	errors = result.Errors

	return &Result{
		Errors:    errors,
		Truncated: false, // V1.0 doesn't truncate, maxErrors handled internally
	}, nil
}
