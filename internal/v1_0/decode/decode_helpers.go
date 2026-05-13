// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// decodeString handles both quoted and unquoted strings
// Quoted strings: removes outer quotes and unescapes doubled quotes
// Unquoted strings: returns as-is
func decodeString(b []byte) string {
	// Check if quoted (starts and ends with ")
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		// Remove outer quotes
		content := b[1 : len(b)-1]

		// Check if any escaped quotes exist
		hasEscape := false
		for i := 0; i < len(content)-1; i++ {
			if content[i] == '"' && content[i+1] == '"' {
				hasEscape = true
				break
			}
		}

		// Fast path: no escapes
		if !hasEscape {
			return string(content)
		}

		// Slow path: unescape doubled quotes
		result := make([]byte, 0, len(content))
		i := 0
		for i < len(content) {
			if i < len(content)-1 && content[i] == '"' && content[i+1] == '"' {
				result = append(result, '"')
				i += 2 // Skip both quotes
			} else {
				result = append(result, content[i])
				i++
			}
		}

		return string(result)
	}

	// Unquoted string - return as-is
	return string(b)
}
