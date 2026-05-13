// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

// Package v1_0 implements SuperCSV v1.0 decoding.
//
// The decoder provides three modes:
//
//   - Raw: Returns raw []byte fields wrapped in []any (no type decoding)
//   - Shallow: Decodes scalar types, returns containers as strings
//   - Full: Full decoding including containers (lists/arrays)
//
// Example usage:
//
//	decoder := NewFileDecoder(1024*1024, DecodeFull)
//	result, err := decoder.DecodeFile("data.csv")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, row := range result.Rows {
//	    // Process row ([]any with decoded values)
//	}
//
// Architecture:
//
// The decoder is the sibling of the validator, sharing schema and
// container parsing logic but maintaining separate code paths.
// This enables optional single-pass fusion in future versions.
package v1_0
