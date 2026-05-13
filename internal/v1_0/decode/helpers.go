// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"fmt"

	"github.com/supercsv/supercsv/internal/v1_0/byteclass"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
)

// Character classification flags for combined lookup table
const (
	charForbidden     = byteclass.CharForbidden
	charContinuation  = byteclass.CharContinuation
	charUnicodeWSLead = byteclass.CharUnicodeWSLead
)

// isUnicodeEdgeWS delegates to the shared byteclass edge-invalid lookup.
func isUnicodeEdgeWS(b []byte) bool {
	return byteclass.IsUnicodeEdgeWS(b)
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// trimASCIISpace trims ASCII whitespace (space, tab, \n, \r) from both ends.
func trimASCIISpace(data []byte) []byte {
	return byteclass.TrimASCIISpace(data)
}

// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// isNumericType checks if a scalar kind is numeric (cannot be quoted in containers).
func isNumericType(kind headerdef.ScalarKind) bool {
	return kind == headerdef.ScalarInt ||
		kind == headerdef.ScalarFloat ||
		kind == headerdef.ScalarDecimal ||
		kind == headerdef.ScalarBool ||
		kind == headerdef.ScalarBytesHex ||
		kind == headerdef.ScalarBytesB64 ||
		kind == headerdef.ScalarUUID
}

// isNullElement checks if a container element is the null literal "_".
// HOT_PATH -> zero allocations, ASCII only, no []byte->string
func isNullElement(data []byte) bool {
	return len(data) == 1 && data[0] == '_'
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateQuotedStringStructure validates quoted string structure without materialization.
func validateQuotedStringStructure(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("invalid structure: quoted string too short")
	}

	if data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid structure: missing quotes")
	}

	// Validate internal quote escaping without materializing
	content := data[1 : len(data)-1]
	n := len(content)
	for i := 0; i < n; i++ {
		if content[i] == '"' {
			// Must be part of a doubled quote pair
			if i+1 < n && content[i+1] == '"' {
				i++ // Skip the second quote
			} else {
				return fmt.Errorf("invalid structure: stray quote")
			}
		}
	}

	return nil
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// validateUnquotedStringStructure validates unquoted string without materialization.
func validateUnquotedStringStructure(data []byte) error {
	trimmed := trimASCIISpace(data)

	// Empty unquoted strings not allowed (must use null _ or quoted "")
	if len(trimmed) == 0 {
		return fmt.Errorf("empty unquoted string (use _ or \"\")")
	}

	lastLead := 0
	for i := 0; i < len(trimmed); i++ {
		c := byteclass.CharClass[trimmed[i]]
		if c&charForbidden != 0 {
			return fmt.Errorf("unquoted string contains forbidden character: %q", trimmed[i])
		}
		if c&charContinuation == 0 {
			lastLead = i
		}
	}

	if byteclass.CharClass[trimmed[0]]&charUnicodeWSLead != 0 && isUnicodeEdgeWS(trimmed) {
		return fmt.Errorf("unexpected whitespace character at the start or end of an unquoted string; use a quoted string")
	}
	if byteclass.CharClass[trimmed[lastLead]]&charUnicodeWSLead != 0 && isUnicodeEdgeWS(trimmed[lastLead:]) {
		return fmt.Errorf("unexpected whitespace character at the start or end of an unquoted string; use a quoted string")
	}

	return nil
}

// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// parseBracketedContent strips outer brackets from container value and returns inner content.
// Zero allocations.
func parseBracketedContent(value []byte) ([]byte, error) {
	value = trimASCIISpace(value)

	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return nil, fmt.Errorf("invalid structure: missing brackets")
	}

	content := value[1 : len(value)-1]

	// Allow at most one \n after the opening '[' (line continuation).
	// A second \n would create a blank line, which is invalid.
	// trimASCIISpace treats \n as whitespace and would silently swallow it,
	// so we must check before calling trimASCIISpace.
	sawNewline := false
	for i := 0; i < len(content); i++ {
		if content[i] == ' ' || content[i] == '\t' || content[i] == '\r' {
			continue
		}
		if content[i] == '\n' {
			if sawNewline {
				return nil, fmt.Errorf("blank line not allowed after opening bracket")
			}
			sawNewline = true
			continue
		}
		break
	}

	return trimASCIISpace(content), nil
}

// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// parsePrefix1D parses optional "N][" prefix for 1D containers.
// Input is bracket-stripped content, e.g. "3][1,2,3" from original "[3][1,2,3]".
// Returns (prefixLength, remainingContent, error).
// prefixLength = -1 if no prefix found.
// Zero allocations.
func parsePrefix1D(content []byte, fixedLength int, elementType headerdef.ScalarType) (int, []byte, error) {
	n := len(content)
	pos := 0

	// Skip leading whitespace
	for pos < n && byteclass.IsWhitespace[content[pos]] {
		pos++
	}

	// Check for strict prefix grammar: [1-9][0-9]*][
	if pos < n && content[pos] >= '1' && content[pos] <= '9' {
		prefixNum := 0
		for pos < n && content[pos] >= '0' && content[pos] <= '9' {
			prefixNum = prefixNum*10 + int(content[pos]-'0')
			pos++
		}

		// Check for ][ to confirm this is a prefix
		// No time disambiguation needed: [N][...] is unambiguous (unlike old [N:...] vs HH:MM:SS)
		if pos+1 < n && content[pos] == ']' && content[pos+1] == '[' {
			// Fixed-size containers cannot use prefix notation
			if fixedLength > 0 {
				return -1, nil, fmt.Errorf("prefix notation cannot be used with fixed-size arrays")
			}

			content = content[pos+2:]

			// Allow at most one \n after the prefix delimiter (line continuation).
			// A second \n would create a blank line, which is invalid.
			// trimASCIISpace treats \n as whitespace and would silently swallow it,
			// so we must check before calling trimASCIISpace.
			sawNewline := false
			for i := 0; i < len(content); i++ {
				if content[i] == ' ' || content[i] == '\t' || content[i] == '\r' {
					continue
				}
				if content[i] == '\n' {
					if sawNewline {
						return -1, nil, fmt.Errorf("blank line not allowed after prefix delimiter")
					}
					sawNewline = true
					continue
				}
				break
			}

			return prefixNum, trimASCIISpace(content), nil
		}
	}

	return -1, content, nil
}

// parseNextElement parses the next element from content starting at pos.
// WARM_PATH -> performance-critical value parsing.
// Runs after HOT_PATH field isolation.
// ASCII fast-path required; Unicode checks must be byte-level only.
// Avoid allocations; avoid []byte->string unless required.
// No Unicode tables; no rune decoding unless unavoidable.
// Returns (element, isQuoted, newPos, allowNewline, error).
// allowNewline tracks whether a \n continuation is currently permitted:
//   - true right after '[' and right after each ','
//   - false after any \n is consumed (second \n = blank line -> error)
//   - false inside element body and between element and comma
//
// Zero allocations.
func parseNextElement(content []byte, pos int, allowNewline bool) ([]byte, bool, int, bool, error) {
	n := len(content)

	// Loop 1: Skip leading whitespace.
	// A single \n is allowed when allowNewline is true (continuation after '[' or ',').
	// A second \n with allowNewline already false is a blank line -> error.
	for pos < n && byteclass.IsWhitespace[content[pos]] {
		if content[pos] == '\n' {
			if !allowNewline {
				return nil, false, pos, false, fmt.Errorf("blank line not allowed in container")
			}
			allowNewline = false
		}
		pos++
	}

	if pos >= n {
		return nil, false, pos, allowNewline, fmt.Errorf("unexpected end of content")
	}

	elemStart := pos
	isQuoted := false

	if content[pos] == '"' {
		// Quoted element
		isQuoted = true
		pos++

		for pos < n {
			if content[pos] == '"' {
				if pos+1 < n && content[pos+1] == '"' {
					// Doubled quote escape
					pos += 2
				} else {
					// End quote
					pos++
					break
				}
			} else {
				pos++
			}
		}
	} else {
		// Loop 2: Unquoted element - read until comma or end.
		// \n inside an unquoted element body is never valid.
		for pos < n && content[pos] != ',' {
			if content[pos] == '\n' {
				return nil, false, pos, false, fmt.Errorf("newline not allowed inside unquoted element")
			}
			pos++
		}
	}

	element := content[elemStart:pos]

	// Loop 3: Skip trailing whitespace after element (before comma or end).
	// allowNewline is false here; any \n is invalid.
	for pos < n && byteclass.IsWhitespace[content[pos]] {
		if content[pos] == '\n' {
			return nil, false, pos, false, fmt.Errorf("newline not allowed between element and comma")
		}
		pos++
	}

	// Handle comma if present
	if pos < n && content[pos] == ',' {
		pos++ // Skip comma

		// Comma resets continuation: the next \n is permitted.
		allowNewline = true

		// Loop 4: Skip whitespace after comma.
		// A single \n is allowed (continuation); a second \n is a blank line -> error.
		for pos < n && byteclass.IsWhitespace[content[pos]] {
			if content[pos] == '\n' {
				if !allowNewline {
					return nil, false, pos, false, fmt.Errorf("blank line not allowed in container")
				}
				allowNewline = false
			}
			pos++
		}

		// Check for double comma (consecutive commas)
		if pos < n && content[pos] == ',' {
			return nil, false, pos, false, fmt.Errorf("consecutive commas not allowed (missing element)")
		}
	}

	return element, isQuoted, pos, allowNewline, nil
}
