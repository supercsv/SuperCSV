// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

const (
	defaultChunkSize = 64 * 1024
)

var (
	errStrayQuote               = errors.New("invalid structure: quote in unquoted field")
	errUnclosedQuotedField      = errors.New("invalid structure: unclosed quoted field")
	errUnexpectedCharAfterQuote = errors.New("invalid structure: unexpected character after quote")
	errNULByte                  = errors.New("NUL byte not allowed")
	errInlineComment            = errors.New("inline comments not allowed")
	errRowInUse                 = errors.New("row still in use")
	errInvalidBOM               = errors.New("non-UTF-8 BOM detected: file must be UTF-8 encoded")

	// ErrFieldTooLarge is returned by emitField() when a field exceeds MaxFieldSize.
	ErrFieldTooLarge = errors.New("supercsv: field exceeds maximum size")
)

type StreamOptions struct {
	ChunkSize     int
	MaxFieldSize  int
	HeaderMode    bool   // Enable angle bracket tracking for type declarations
	StrictVersion string // Required SuperCSV version (empty = no check)
}

type CSVStream struct {
	buffer                *chunkBuffer
	options               StreamOptions
	row                   RawRow
	rowInUse              bool
	materialized          []byte
	metaLineScratch       []byte
	fieldStart            chunkCursor // Where the current field began
	fieldEnd              chunkCursor // Where the current field ends (exclusive)
	fieldQuoted           bool        // Whether the current field is quoted
	containerDepth        int         // Bracket nesting depth for container-aware parsing
	genericDepth          int         // Angle bracket depth for type parameters (header mode only)
	inContainerQuote      bool        // Whether we're inside a quote within a container
	containerNewlineOK    bool        // Whether \n at the current position is a valid continuation (after '[', ',', or inner ']')
	containerSawNewline   bool        // Whether one \n has already been consumed at this continuation point (blank-line guard)
	containerInnerRowOpen bool        // Whether the most recent '[' at depth 1->2 was a structural inner-row open
	headerMode            bool        // Whether angle brackets should affect container depth

	// Physical line tracking for header continuation validation
	skippedCommentLines          int8 // Number of # comment lines skipped before the next row
	headerCommentAttachWindow    bool // True after a blank line opens a pre-header # comment attachment window
	headerAttachableCommentLines int8 // Number of # comment lines eligible to attach to the first header field

	// Per-field annotation block counters (reset per field in emitField)
	fieldCommentBlocks int8 // Comment ( ) blocks seen for the current field
	fieldMetaBlocks    int8 // Metadata (( )) blocks seen for the current field

	// Standalone annotation line counters (consumed by skipLeadingBlanks, read by validator)
	skippedStandaloneComments int8 // Standalone ( ) comment lines skipped before current row
	skippedStandaloneMeta     int8 // Standalone (( )) metadata lines skipped before current row

	// Row line tracking for error reporting
	rowStartLine int // Physical line where current row started (for parse error context)
}

// parserState represents the current parsing state in the CSV state machine.
// The parser transitions between these states based on encountered characters.
type parserState int

const (
	stateOutside    parserState = iota // Between fields, expecting delimiter, quote, or data
	stateInUnquoted                    // Inside an unquoted field
	stateInQuoted                      // Inside a quoted field
	stateQuote                         // Just saw a quote inside a quoted field (could be escape or end)
)

// State transitions:
// Outside     -> Unquoted | Quoted | Outside
// InUnquoted  -> Outside  | InUnquoted
// InQuoted    -> InQuoted | Quote
// Quote       -> Outside  | InQuoted | Quote

// action represents what the main loop should do after a handler processes a byte.
// Handlers return an action to indicate whether processing should continue,
// emit a field, or complete the current row.
type action int

const (
	actionNone                 action = iota // Continue processing
	actionEmitField                          // Emit current field to row
	actionEndRow                             // End current row and return
	actionIncrementBracket                   // Increment container depth
	actionDecrementBracket                   // Decrement container depth
	actionIncrementGeneric                   // Increment generic depth (header mode)
	actionDecrementGeneric                   // Decrement generic depth (header mode)
	actionToggleContainerQuote               // Toggle inContainerQuote flag
	actionSkipAnnotation                     // Consume trailing ( ) or (( )) annotation block after a quoted field
)

// stepResult is the return type for state handler functions.
// It encapsulates the next state, required action, and any error encountered.
// Handlers are pure functions that only return stepResult; the main loop
// handles all state mutations based on the returned result.
type stepResult struct {
	next   parserState // Next state to transition to
	action action      // Action for main loop to perform
	err    error       // Error encountered, if any
}

// Helper constructors for stepResult to eliminate boilerplate.
// These functions make handler code more readable and less error-prone.

// next returns a stepResult that transitions to the given state with no action.
func next(s parserState) stepResult {
	return stepResult{next: s}
}

// emit returns a stepResult that emits the current field and transitions to the given state.
func emit(s parserState) stepResult {
	return stepResult{next: s, action: actionEmitField}
}

// endRow returns a stepResult that completes the current row and transitions to the given state.
func endRow(s parserState) stepResult {
	return stepResult{next: s, action: actionEndRow}
}

// fail returns a stepResult containing an error, halting processing.
func fail(err error) stepResult {
	return stepResult{err: err}
}

// incrementBracket returns a stepResult that increments bracket depth and transitions to the given state.
func incrementBracket(s parserState) stepResult {
	return stepResult{next: s, action: actionIncrementBracket}
}

// decrementBracket returns a stepResult that decrements bracket depth and transitions to the given state.
func decrementBracket(s parserState) stepResult {
	return stepResult{next: s, action: actionDecrementBracket}
}

// incrementGeneric returns a stepResult that increments generic depth and transitions to the given state.
func incrementGeneric(s parserState) stepResult {
	return stepResult{next: s, action: actionIncrementGeneric}
}

// decrementGeneric returns a stepResult that decrements generic depth and transitions to the given state.
func decrementGeneric(s parserState) stepResult {
	return stepResult{next: s, action: actionDecrementGeneric}
}

// toggleQuote returns a stepResult that toggles the inContainerQuote flag and transitions to the given state.
func toggleQuote(s parserState) stepResult {
	return stepResult{next: s, action: actionToggleContainerQuote}
}

// skipAnnotation returns a stepResult that triggers consumption of a trailing ( ) or (( )) annotation block.
func skipAnnotation(s parserState) stepResult {
	return stepResult{next: s, action: actionSkipAnnotation}
}

func NewCSVStream(r io.Reader, opts StreamOptions) (*CSVStream, error) {
	normalized, err := normalizeStreamOptions(opts)
	if err != nil {
		return nil, err
	}
	size := normalized.ChunkSize
	br := bufio.NewReader(r)
	if err := rejectBOM(br); err != nil {
		return nil, err
	}
	if err := checkStrictVersionFirstLine(br, normalized.StrictVersion); err != nil {
		return nil, err
	}
	buffer := newChunkBuffer(br, size)
	stream := &CSVStream{
		buffer:     buffer,
		options:    normalized,
		headerMode: normalized.HeaderMode,
	}
	stream.row.parser = stream
	return stream, nil
}

func normalizeStreamOptions(opts StreamOptions) (StreamOptions, error) {
	normalized := StreamOptions{
		ChunkSize:     opts.ChunkSize,
		MaxFieldSize:  opts.MaxFieldSize,
		HeaderMode:    opts.HeaderMode,
		StrictVersion: opts.StrictVersion,
	}
	if normalized.ChunkSize <= 0 {
		normalized.ChunkSize = defaultChunkSize
	}
	if normalized.MaxFieldSize < 0 {
		return StreamOptions{}, fmt.Errorf("invalid field size: %d (expected >= 0)", normalized.MaxFieldSize)
	}
	return normalized, nil
}

func rejectBOM(br *bufio.Reader) error {
	peek, err := br.Peek(4)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return err
	}
	// Check UTF-32 before UTF-16: UTF-32 LE (FF FE 00 00) shares a 2-byte prefix with UTF-16 LE (FF FE).
	if len(peek) >= 4 && peek[0] == 0xFF && peek[1] == 0xFE && peek[2] == 0x00 && peek[3] == 0x00 {
		return errInvalidBOM // UTF-32 LE
	}
	if len(peek) >= 4 && peek[0] == 0x00 && peek[1] == 0x00 && peek[2] == 0xFE && peek[3] == 0xFF {
		return errInvalidBOM // UTF-32 BE
	}
	if len(peek) >= 2 && peek[0] == 0xFF && peek[1] == 0xFE {
		return errInvalidBOM // UTF-16 LE
	}
	if len(peek) >= 2 && peek[0] == 0xFE && peek[1] == 0xFF {
		return errInvalidBOM // UTF-16 BE
	}
	if len(peek) >= 3 && peek[0] == 0xEF && peek[1] == 0xBB && peek[2] == 0xBF {
		// UTF-8 BOM: silently consume and continue
		br.Discard(3)
	}
	return nil
}

// checkStrictVersionFirstLine validates that the first line matches the canonical version directive.
// Strict-version checking is non-negotiable: either line 1 is exactly "((SuperCSV v1.0))" or it's an error.
func checkStrictVersionFirstLine(br *bufio.Reader, strictVersion string) error {
	if strictVersion == "" {
		return nil
	}

	line, err := peekFirstLine(br)
	if err != nil {
		return err
	}
	if !isCanonicalVersionDirectiveLine(line) {
		return fmt.Errorf("strict-version %s: file has no version", strictVersion)
	}

	prefix := []byte("((supercsv ")

	// Extract declared version (everything after "((SuperCSV " and before "))")
	declaredVersion := string(line[len(prefix) : len(line)-2])

	// Check if version matches
	if declaredVersion != strictVersion {
		return fmt.Errorf("strict-version %s: line 1 declares %s", strictVersion, declaredVersion)
	}

	return nil
}

func peekFirstLine(br *bufio.Reader) ([]byte, error) {
	peek, err := br.Peek(256)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return nil, err
	}

	// Extract first line
	end := 0
	for end < len(peek) && peek[end] != '\n' && peek[end] != '\r' {
		end++
	}
	return peek[:end], nil
}

func isCanonicalVersionDirectiveLine(line []byte) bool {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	prefix := []byte("((supercsv ")
	if len(line) < len(prefix)+2 {
		return false
	}
	if !matchPrefixIgnoreCaseASCII(line, prefix) {
		return false
	}
	if line[len(line)-2] != ')' || line[len(line)-1] != ')' {
		return false
	}
	return true
}

// skipLeadingBlanks skips blank lines, comment lines, and leading whitespace
// to find the start of the next data row.
// Returns the first data byte, its cursor position, and line number,
// or an error if NUL is encountered or EOF is reached.
// Comment lines (lines starting with # as first non-whitespace) are skipped.
//
// During header mode, tracks skipped blank/comment lines for continuation validation.
func (p *CSVStream) skipLeadingBlanks() (byte, chunkCursor, int, error) {
	for {
		preCursor := p.buffer.cursor
		preLine := p.buffer.line
		preCol := p.buffer.col

		b, pos, line, err := p.buffer.readByte()
		if err != nil {
			return 0, chunkCursor{}, 0, err
		}

		if b == 0 {
			return 0, chunkCursor{}, 0, p.wrapError(errNULByte, pos)
		}

		// Skip blank lines (newlines)
		if b == '\n' {
			if p.headerMode {
				p.skippedCommentLines = 0
				p.headerCommentAttachWindow = true
				p.headerAttachableCommentLines = 0
				p.skippedStandaloneComments = 0
				p.skippedStandaloneMeta = 0
			}
			continue
		}

		// Skip comment lines (# as first non-whitespace character)
		if b == '#' {
			if p.headerMode {
				if p.headerCommentAttachWindow {
					p.headerAttachableCommentLines++
				}
			}
			p.skippedCommentLines++
			// Skip to end of line
			for {
				cb, _, _, err := p.buffer.readByte()
				if err != nil {
					return 0, chunkCursor{}, 0, err
				}
				if cb == '\n' {
					break
				}
			}
			continue
		}

		// Skip line-level annotation blocks if first non-whitespace character is '('
		if b == '(' {
			lineBuf, rerr := p.readLineStartingWithParen(b)
			if rerr != nil {
				return 0, chunkCursor{}, 0, rerr
			}

			trimmed := trimWhitespace(lineBuf)
			if isLineLevelCommentOrMeta(trimmed) {
				if p.headerMode && isCanonicalVersionDirectiveLine(trimmed) {
					p.headerCommentAttachWindow = false
					p.headerAttachableCommentLines = 0
					continue
				}

				sc, sm := countBlocksOnLine(trimmed)
				if p.headerMode {
					p.headerCommentAttachWindow = false
					p.headerAttachableCommentLines = 0
					p.skippedStandaloneComments += sc
					p.skippedStandaloneMeta += sm
				} else {
					// Count standalone annotation blocks for continuation tracking.
					// The validator uses these to detect duplicates when standalone
					// annotation lines appear between continuation rows.
					p.skippedStandaloneComments += sc
					p.skippedStandaloneMeta += sm
				}
				continue
			}

			// Not a line-level annotation: the line starts with inline prefix
			// annotation block(s) followed by data. Reset cursor to before '('
			// and consume the annotation blocks so the caller receives the
			// first actual data byte.
			p.buffer.cursor = preCursor
			p.buffer.line = preLine
			p.buffer.col = preCol

			skipOK := true
			for {
				ab, apos, aline, aerr := p.buffer.readByte()
				if aerr != nil {
					return 0, chunkCursor{}, 0, aerr
				}
				if ab == '(' {
					hitNL, skipErr := p.skipAnnotationBlockStream()
					if skipErr != nil {
						return 0, chunkCursor{}, 0, skipErr
					}
					if hitNL {
						// Annotation block wasn't closed before newline -
						// treat as end of line and restart the outer loop.
						skipOK = false
						break
					}
					continue
				}
				if isSpace(ab) {
					continue
				}
				// Found actual data byte
				return ab, apos, aline, nil
			}
			if !skipOK {
				continue
			}
		}

		// Skip leading whitespace (spaces and tabs)
		if isSpace(b) {
			continue
		}

		// Found first data byte
		return b, pos, line, nil
	}
}

// readLineStartingWithParen reads a physical line that has already consumed the
// first byte '(' from the stream. Returns the full line bytes including trailing
// '\n' when present. EOF is treated as valid line termination.
func (p *CSVStream) readLineStartingWithParen(first byte) ([]byte, error) {
	var stack [256]byte
	n := 0
	stack[n] = first
	n++

	for {
		b, _, _, err := p.buffer.readByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return stack[:n], nil
			}
			return nil, err
		}

		if n < len(stack) {
			stack[n] = b
			n++
			if b == '\n' {
				return stack[:n], nil
			}
			continue
		}

		// Overflow path: switch to reusable scratch buffer.
		buf := p.metaLineScratch[:0]
		buf = append(buf, stack[:n]...)

		for {
			buf = append(buf, b)
			if b == '\n' {
				p.metaLineScratch = buf
				return buf, nil
			}

			b, _, _, err = p.buffer.readByte()
			if err != nil {
				if errors.Is(err, io.EOF) {
					p.metaLineScratch = buf
					return buf, nil
				}
				return nil, err
			}
		}
	}
}

// skipAnnotationBlockStream consumes a ( ) or (( )) annotation block from the stream.
// The opening '(' has already been consumed by the caller (handleQuote via actionSkipAnnotation).
// Returns hitNewline=true if a bare '\n' terminates the block before it is closed,
// meaning the row has ended and the caller should finishRow.
func (p *CSVStream) skipAnnotationBlockStream() (hitNewline bool, err error) {
	b, _, _, err := p.buffer.readByte()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}

	double := b == '('

	// Count this annotation block
	if double {
		p.fieldMetaBlocks++
	} else {
		p.fieldCommentBlocks++
	}

	if !double {
		// Single-paren block: b is the first content byte; scan until ')'.
		for {
			if b == ')' {
				return false, nil
			}
			if b == '\n' {
				return true, nil
			}
			b, _, _, err = p.buffer.readByte()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return false, nil
				}
				return false, err
			}
		}
	}

	// Double-paren block: scan until '))'.
	for {
		b, _, _, err = p.buffer.readByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}
		if b == '\n' {
			return true, nil
		}
		if b == ')' {
			b2, _, _, err2 := p.buffer.readByte()
			if err2 != nil {
				if errors.Is(err2, io.EOF) {
					return false, nil
				}
				return false, err2
			}
			if b2 == ')' {
				return false, nil
			}
			if b2 == '\n' {
				return true, nil
			}
		}
	}
}

// wrapError wraps a parser error with column context.
// Preserves the original error for errors.Is() compatibility using %w.
// Column numbers are 1-based for human readability.
func (p *CSVStream) wrapError(base error, pos chunkCursor) error {
	// Column is 1-based for user readability
	col := pos.Col + 1
	return fmt.Errorf("column %d: %w", col, base)
}

// NextRow reads and returns the next CSV row from the stream.
//
// NextRow uses a state machine to parse CSV data efficiently:
//   - Handles quoted and unquoted fields
//   - Supports multiline quoted fields
//   - Rejects inline comments (#)
//   - Skips blank lines and leading whitespace
//   - Validates quote escaping ("" -> ")
//
// The returned rawRow contains field slices that point directly into the
// internal chunk buffer for zero-copy parsing. Fields that span chunk boundaries
// are materialized into a separate buffer. The row must be released via
// rawRow.Release() before calling NextRow again.
//
// Returns io.EOF when no more rows are available.
// Returns an error if:
//   - The previous row hasn't been released (errRowInUse)
//   - Invalid CSV syntax is encountered (stray quotes, unclosed fields, etc.)
//   - A NUL byte or inline comment is found
//   - A field exceeds MaxFieldSize (if configured)
//
// Performance characteristics:
//   - Zero allocations for fields within a single chunk
//   - Minimal allocations for cross-chunk fields (materialized buffer reuse)
//   - Array-based handler dispatch for branch prediction
//   - Inline-friendly helper functions (next, emit, endRow, fail)
func (p *CSVStream) NextRow() (*RawRow, error) {
	if p.rowInUse {
		return nil, errRowInUse
	}
	p.row.Fields = p.row.Fields[:0]
	p.materialized = p.materialized[:0]
	p.row.releaseCursor = chunkCursor{}
	p.row.parser = p
	p.containerDepth = 0            // Reset bracket depth for new row
	p.inContainerQuote = false      // Reset quote tracking for new row
	p.containerNewlineOK = false    // Reset newline continuation for new row
	p.containerSawNewline = false   // Reset blank-line guard for new row
	p.containerInnerRowOpen = false // Reset inner-row tracking for new row
	p.fieldCommentBlocks = 0        // Reset annotation counters for new row
	p.fieldMetaBlocks = 0

	// Skip leading blanks and get first data byte
	b, pos, line, err := p.skipLeadingBlanks()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, err
	}

	// Initialize row tracking
	state := stateOutside
	rowEndLine := line
	p.fieldStart = pos
	p.rowStartLine = line // Store for error reporting

	// Handle first byte of row
	if b == '"' {
		state = stateInQuoted
		p.fieldQuoted = true
	} else if b == ',' {
		// Empty field at start
		p.fieldEnd = pos
		p.fieldQuoted = false
		if err := p.emitField(); err != nil {
			return nil, err
		}
		state = stateOutside
	} else if b == '[' {
		state = stateInUnquoted
		p.fieldQuoted = false
		p.containerDepth++
		p.containerNewlineOK = true // field opened as container; \n continuation enabled
	} else if b == ']' {
		state = stateInUnquoted
		p.fieldQuoted = false
		if p.containerDepth > 0 {
			p.containerDepth--
		}
	} else {
		state = stateInUnquoted
		p.fieldQuoted = false
	}

	// Handler dispatch array for performance
	type handlerFunc func(*CSVStream, byte, chunkCursor, int) stepResult
	handlers := [4]handlerFunc{
		stateOutside:    (*CSVStream).handleOutside,
		stateInUnquoted: (*CSVStream).handleInUnquoted,
		stateInQuoted:   (*CSVStream).handleInQuoted,
		stateQuote:      (*CSVStream).handleQuote,
	}

	// Main parsing loop
	for {
		b, pos, line, err := p.buffer.readByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if state == stateInQuoted || state == stateQuote {
					eofPos := p.buffer.clampCursor(p.buffer.cursor)
					return nil, p.wrapError(errUnclosedQuotedField, eofPos)
				}
				p.fieldEnd = p.buffer.clampCursor(p.buffer.cursor)
				if emitErr := p.emitField(); emitErr != nil {
					return nil, emitErr
				}
				if rowEndLine == 0 {
					rowEndLine = p.rowStartLine
				}
				return p.finishRow(p.rowStartLine, rowEndLine), nil
			}
			return nil, err
		}

		if b == 0 {
			return nil, p.wrapError(errNULByte, pos)
		}

		// Dispatch to handler via array
		res := handlers[state](p, b, pos, line)

		// Handle errors
		if res.err != nil {
			return nil, p.wrapError(res.err, pos)
		}

		// Handle mutations allowed by handlers
		if res.next == stateInQuoted && state == stateOutside {
			p.fieldStart = pos
			p.fieldQuoted = true
		} else if res.next == stateInUnquoted && state == stateOutside {
			p.fieldStart = pos
			p.fieldQuoted = false
		}

		// Handle actions
		switch res.action {
		case actionEmitField:
			p.fieldEnd = pos
			if err := p.emitField(); err != nil {
				return nil, err
			}
		case actionEndRow:
			// Emit final field before ending row
			p.fieldEnd = pos
			if err := p.emitField(); err != nil {
				return nil, err
			}
			rowEndLine = line
			return p.finishRow(p.rowStartLine, rowEndLine), nil
		case actionIncrementBracket:
			p.containerDepth++
			if state == stateOutside {
				// Field starts as container (depth 0->1): enable newline continuation.
				p.containerNewlineOK = true
			} else if p.containerDepth == 1 {
				// Prefix re-open: [N][ pattern - depth went 1->0->1 within the same field.
				// Re-enable newline continuation so \n after [N][ is a valid line break.
				p.containerNewlineOK = true
				p.containerSawNewline = false
			} else if p.containerDepth == 2 {
				// Inner row opens (depth 1->2): structural if containerNewlineOK was true (not mid-content).
				// containerNewlineOK stays true through '[' (guard excludes '[') so we can check it here.
				if p.containerNewlineOK {
					p.containerInnerRowOpen = true
				}
				// Clear so \n inside the inner row is not swallowed (depth 2 has no continuation).
				p.containerNewlineOK = false
				p.containerSawNewline = false
			}
		case actionDecrementBracket:
			if p.containerDepth > 0 {
				p.containerDepth--
			}
			if p.containerDepth == 1 && p.containerInnerRowOpen {
				// Structural inner row closed (depth 2->1): \n before the outer ']' is valid continuation.
				p.containerNewlineOK = true
				p.containerSawNewline = false
				p.containerInnerRowOpen = false
			}
		case actionIncrementGeneric:
			p.genericDepth++
		case actionDecrementGeneric:
			p.genericDepth--
		case actionToggleContainerQuote:
			p.inContainerQuote = !p.inContainerQuote
		case actionSkipAnnotation:
			hitNL, skipErr := p.skipAnnotationBlockStream()
			if skipErr != nil {
				return nil, p.wrapError(skipErr, pos)
			}
			if hitNL {
				p.fieldEnd = pos
				if emitErr := p.emitField(); emitErr != nil {
					return nil, emitErr
				}
				rowEndLine = line
				return p.finishRow(p.rowStartLine, rowEndLine), nil
			}
		}

		// Update state
		state = res.next
	}
}

// BYTE_LEVEL_HOT_PATH_DO_NOT_TOUCH -> 100M+ calls, state machine, maximally optimized
// handleOutside processes a byte when between fields or after a delimiter.
// This is the starting state for each field.
//
// State transitions:
//   - ',' -> emit field if containerDepth == 0, else start unquoted field
//   - '\n' -> emit field (field was empty)
//   - '"' -> stateInQuoted (start quoted field)
//   - space/tab -> stateOutside (skip whitespace between fields)
//   - '#' -> error (inline comments not allowed)
//   - '(' -> skip inline annotation block, stay in stateOutside
//   - '[' -> increment containerDepth, start unquoted field
//   - ']' -> decrement containerDepth, start unquoted field
//   - '<' -> increment genericDepth (header mode only), start unquoted field
//   - '>' -> decrement genericDepth (header mode only), start unquoted field
//   - other -> stateInUnquoted (start unquoted field)
//
// The main loop handles mutations:
//   - Sets fieldStart when transitioning to InQuoted or InUnquoted
//   - Sets fieldQuoted based on transition target
func (p *CSVStream) handleOutside(b byte, pos chunkCursor, line int) stepResult {
	switch b {
	case ',':
		// Only split field if outside containers and generic types
		if p.containerDepth == 0 && p.genericDepth == 0 {
			return emit(stateOutside)
		}
		// Inside container or generic type - comma is part of field content
		return next(stateInUnquoted)
	case '\n':
		return endRow(stateOutside)
	case ' ', '\t':
		return next(stateOutside)
	case '#':
		return fail(errInlineComment)
	case '(':
		// Inline prefix annotation: consume ( ) or (( )) block, stay between fields
		return skipAnnotation(stateOutside)
	case '"':
		// Quotes inside containers are literal characters, not CSV quoting
		if p.containerDepth > 0 {
			return toggleQuote(stateInUnquoted)
		}
		return next(stateInQuoted)
	case '[':
		// Only track brackets when not inside a quoted string within container
		if p.containerDepth > 0 && p.inContainerQuote {
			return next(stateInUnquoted)
		}
		return incrementBracket(stateInUnquoted)
	case ']':
		// Only track brackets when not inside a quoted string within container
		if p.containerDepth > 0 && p.inContainerQuote {
			return next(stateInUnquoted)
		}
		return decrementBracket(stateInUnquoted)
	case '<':
		// Track angle brackets for type parameters in header mode only
		if p.containerDepth > 0 && p.inContainerQuote {
			return next(stateInUnquoted)
		}
		if p.headerMode {
			return incrementGeneric(stateInUnquoted)
		}
		return next(stateInUnquoted)
	case '>':
		// Track angle brackets for type parameters in header mode only
		if p.containerDepth > 0 && p.inContainerQuote {
			return next(stateInUnquoted)
		}
		if p.headerMode {
			return decrementGeneric(stateInUnquoted)
		}
		return next(stateInUnquoted)
	default:
		return next(stateInUnquoted)
	}
}

// BYTE_LEVEL_HOT_PATH_DO_NOT_TOUCH -> 100M+ calls, state machine, maximally optimized
// handleInUnquoted processes a byte when inside an unquoted field.
//
// State transitions:
//   - ','	-> emit field if containerDepth == 0, else continue in field
//   - '\n' -> emit field and return to Outside
//   - '"' -> error (stray quote inside unquoted field)
//   - '[' -> increment containerDepth, continue in field
//   - ']' -> decrement containerDepth, continue in field
//   - '<' -> increment genericDepth (header mode only), continue in field
//   - '>' -> decrement genericDepth (header mode only), continue in field
//   - other -> stay in InUnquoted (continue accumulating field)
//
// Note: containerNewlineOK and containerSawNewline are mutated directly here (exception to pure-handler rule)
// because they must be cleared on element-start bytes at 100M+ calls/s. Clearing via action on every default
// byte would be equally costly.
func (p *CSVStream) handleInUnquoted(b byte, pos chunkCursor, line int) stepResult {
	// Any non-whitespace, non-comma, non-newline, non-'[' byte at depth==1 means element
	// content has started: newline continuation is no longer valid at this position.
	// '[' excluded: must remain visible in actionIncrementBracket to detect structural inner rows.
	if p.containerNewlineOK && p.containerDepth == 1 && b != ',' && b != '[' && b != '\n' && b != ' ' && b != '\t' {
		p.containerNewlineOK = false
		p.containerSawNewline = false
	}
	switch b {
	case ',':
		// Only split field if outside containers and generic types
		if p.containerDepth == 0 && p.genericDepth == 0 {
			return emit(stateOutside)
		}
		// Comma at the outer container level re-enables newline continuation.
		if p.containerDepth == 1 {
			p.containerNewlineOK = true
			p.containerSawNewline = false
		}
		// Inside container or generic type - comma is part of field content
		return next(stateInUnquoted)
	case '\n':
		// Swallow the newline in two cases:
		// 1. containerNewlineOK: valid continuation position (after '[', ',', prefix '][', or inner ']').
		//    containerNewlineOK is NOT cleared on swallow - it stays true so that the '[' of the
		//    next inner row is visible as structural in actionIncrementBracket.
		//    containerSawNewline is set to detect a second consecutive \n (blank line -> end row).
		// 2. inContainerQuote: \n is a literal character inside a quoted element (e.g. ["a\nb"]).
		if p.containerNewlineOK {
			if p.containerSawNewline {
				// Second \n at this continuation point = blank line: end the row.
				return endRow(stateOutside)
			}
			p.containerSawNewline = true
			return next(stateInUnquoted)
		}
		if p.inContainerQuote {
			return next(stateInUnquoted)
		}
		return endRow(stateOutside)
	case '"':
		// Quotes inside containers are literal characters, not CSV syntax
		if p.containerDepth > 0 {
			return toggleQuote(stateInUnquoted)
		}
		return fail(errStrayQuote)
	case '[':
		// Only track brackets when not inside a quoted string within container
		if p.containerDepth > 0 && p.inContainerQuote {
			return next(stateInUnquoted)
		}
		return incrementBracket(stateInUnquoted)
	case ']':
		// Only track brackets when not inside a quoted string within container
		if p.containerDepth > 0 && p.inContainerQuote {
			return next(stateInUnquoted)
		}
		return decrementBracket(stateInUnquoted)
	case '<':
		// Track angle brackets for type parameters in header mode only
		if p.containerDepth > 0 && p.inContainerQuote {
			return next(stateInUnquoted)
		}
		if p.headerMode {
			return incrementGeneric(stateInUnquoted)
		}
		return next(stateInUnquoted)
	case '>':
		// Track angle brackets for type parameters in header mode only
		if p.containerDepth > 0 && p.inContainerQuote {
			return next(stateInUnquoted)
		}
		if p.headerMode {
			return decrementGeneric(stateInUnquoted)
		}
		return next(stateInUnquoted)
	default:
		return next(stateInUnquoted)
	}
}

// BYTE_LEVEL_HOT_PATH_DO_NOT_TOUCH -> 100M+ calls, state machine, maximally optimized
// handleInQuoted processes a byte when inside a quoted field.
//
// State transitions:
//   - '"' -> stateQuote (potential escape or field end)
//   - other -> stay in InQuoted (continue accumulating field, including newlines)
//
// Quoted fields can contain any character including commas and newlines.
// Handlers are pure - no state mutations.
func (p *CSVStream) handleInQuoted(b byte, pos chunkCursor, line int) stepResult {
	if b == '"' {
		return next(stateQuote)
	}
	return next(stateInQuoted)
}

// BYTE_LEVEL_HOT_PATH_DO_NOT_TOUCH -> 100M+ calls, state machine, maximally optimized
// handleQuote processes a byte immediately after seeing a quote inside a quoted field.
// Disambiguates between escaped quotes ("") and field-closing quotes.
//
// State transitions:
//   - '"' -> stateInQuoted (escaped quote: "" -> ")
//   - ',' or '\n' -> emit field (quote was field closer)
//   - space/tab -> stateQuote (skip trailing whitespace after closing quote)
//   - '(' -> actionSkipAnnotation (consume trailing ( ) or (( )) annotation block)
//   - other -> error (unexpected character after closing quote)
//
// Handlers are pure - no state mutations.
func (p *CSVStream) handleQuote(b byte, pos chunkCursor, line int) stepResult {
	switch b {
	case '"':
		return next(stateInQuoted)
	case ',':
		return emit(stateOutside)
	case '\n':
		return endRow(stateOutside)
	case ' ', '\t':
		return next(stateQuote)
	case '(':
		return skipAnnotation(stateQuote)
	default:
		return fail(errUnexpectedCharAfterQuote)
	}
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// emitField appends the current field to the row using parser-internal state.
// Uses p.fieldStart as the field start, p.fieldEnd as the field end,
// and p.fieldQuoted to determine if the field was quoted.
// Fields use deferred materialization with inlined first segment for zero allocation.
// Resets field state after emission.
func (p *CSVStream) emitField() error {
	fieldLen, err := p.fieldLength(p.fieldStart, p.fieldEnd)
	if err != nil {
		return err
	}
	if p.options.MaxFieldSize > 0 && fieldLen > p.options.MaxFieldSize {
		return ErrFieldTooLarge
	}

	field := rawField{
		firstChunk:    p.fieldStart.chunk,
		firstStart:    p.fieldStart.offset,
		firstEnd:      p.fieldEnd.offset,
		Quoted:        p.fieldQuoted,
		CommentBlocks: p.fieldCommentBlocks,
		MetaBlocks:    p.fieldMetaBlocks,
		parser:        p,
	}

	// Multi-chunk field: build extra segments
	if p.fieldStart.chunk != p.fieldEnd.chunk {
		field.extraSegments = p.buildExtraSegments(p.fieldStart, p.fieldEnd)
		// Update firstEnd to end of first chunk
		field.firstEnd = p.buffer.chunks[p.fieldStart.chunk].end
	}

	p.row.Fields = append(p.row.Fields, field)

	// Reset field state
	p.fieldStart = p.buffer.cursor
	p.fieldEnd = chunkCursor{}
	p.fieldQuoted = false
	p.fieldCommentBlocks = 0
	p.fieldMetaBlocks = 0

	return nil
}

// buildExtraSegments constructs the segment list for chunks after the first chunk.
// The first segment is stored inline in rawField, this builds the rest.
func (p *CSVStream) buildExtraSegments(start, end chunkCursor) []fieldSegment {
	var segments []fieldSegment
	// Start from second chunk (first is inlined in rawField)
	for i := start.chunk + 1; i <= end.chunk; i++ {
		if i >= len(p.buffer.chunks) || p.buffer.chunks[i] == nil {
			break // Defensive: chunk was released or index is stale
		}
		chunk := p.buffer.chunks[i]
		seg := fieldSegment{chunk: i}
		if i == end.chunk {
			seg.start = 0
			seg.end = end.offset
		} else {
			seg.start = 0
			seg.end = chunk.end
		}
		segments = append(segments, seg)
	}
	return segments
}

// materializeSegments copies all segments of a multi-chunk field into contiguous memory.
// Includes the inlined first segment and all extra segments.
// Single-pass implementation: allocates once, then copies all segments in one traversal.
func (p *CSVStream) materializeSegments(f *rawField) []byte {
	startIdx := len(p.materialized)

	// Copy first segment (inlined) - grow buffer as we go
	firstChunk := p.buffer.chunks[f.firstChunk]
	p.materialized = append(p.materialized, firstChunk.data[f.firstStart:f.firstEnd]...)

	// Copy extra segments - append directly without pre-calculating length
	for _, seg := range f.extraSegments {
		chunk := p.buffer.chunks[seg.chunk]
		p.materialized = append(p.materialized, chunk.data[seg.start:seg.end]...)
	}

	return p.materialized[startIdx:]
}

func (p *CSVStream) fieldLength(start, end chunkCursor) (int, error) {
	if start.chunk > end.chunk {
		return 0, errors.New("supercsv: invalid field range")
	}
	if start.chunk == end.chunk {
		return end.offset - start.offset, nil
	}
	length := 0
	for i := start.chunk; i <= end.chunk; i++ {
		chunk := p.buffer.chunks[i]
		if chunk == nil {
			return 0, errors.New("supercsv: missing chunk")
		}
		if i == start.chunk {
			length += chunk.end - start.offset
			continue
		}
		if i == end.chunk {
			length += end.offset
			continue
		}
		length += chunk.end
	}
	return length, nil
}

func (p *CSVStream) finishRow(startLine, endLine int) *RawRow {
	p.row.Line = startLine
	if endLine > startLine {
		p.row.RowEnd = endLine
	} else {
		p.row.RowEnd = 0
	}
	p.row.releaseCursor = p.buffer.cursor
	p.rowInUse = true
	return &p.row
}

func (p *CSVStream) releaseRow(cursor chunkCursor) {
	p.rowInUse = false
	p.materialized = p.materialized[:0]
	p.buffer.releaseUpTo(cursor)
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

// trimWhitespace removes leading and trailing spaces and tabs from byte slice.
// Matches behavior of strings.TrimSpace but byte-level and inline.
func trimWhitespace(data []byte) []byte {
	// Trim leading whitespace
	start := 0
	for start < len(data) && (data[start] == ' ' || data[start] == '\t') {
		start++
	}
	// Trim trailing whitespace
	end := len(data)
	for end > start && (data[end-1] == ' ' || data[end-1] == '\t') {
		end--
	}
	return data[start:end]
}

// isLineLevelCommentOrMeta checks whether a physical line consists entirely of
// annotation blocks - one or more ( ... ) or (( ... )) blocks with only
// whitespace between them and no other content.
//
// The old first/last-byte check was wrong: a data row like
//
//	( c1 ) alpha (( m1 )), ...
//
// starts with '(' and ends with ')' but is NOT a line-level annotation.
// This replacement does a proper forward scan to confirm every non-whitespace
// byte belongs to an annotation block.
func isLineLevelCommentOrMeta(trimmed []byte) bool {
	n := len(trimmed)
	if n < 2 {
		return false
	}

	// Remove trailing newline before structural checks.
	if trimmed[n-1] == '\n' {
		trimmed = trimmed[:n-1]
		n = len(trimmed)
		if n < 2 {
			return false
		}
	}

	i := 0
	foundBlock := false
	for i < n {
		// Skip whitespace between blocks.
		for i < n && (trimmed[i] == ' ' || trimmed[i] == '\t') {
			i++
		}
		if i == n {
			break
		}
		if trimmed[i] != '(' {
			return false // non-annotation content
		}
		foundBlock = true
		i++ // consume '('

		double := i < n && trimmed[i] == '('
		if double {
			i++ // consume second '('
		}

		// Scan for the matching close paren(s).
		// Reject nested '(' - it means this is data, not a pure annotation block.
		closed := false
		for i < n {
			b := trimmed[i]
			if b == '(' {
				return false // nested open paren: line has data content
			}
			if double {
				if b == ')' && i+1 < n && trimmed[i+1] == ')' {
					i += 2 // consume '))'
					closed = true
					break
				}
			} else {
				if b == ')' {
					i++ // consume ')'
					closed = true
					break
				}
			}
			i++
		}
		if !closed {
			return false // unclosed block
		}
	}
	return foundBlock
}

// countBlocksOnLine counts the number of comment ( ) and metadata (( )) blocks
// on a standalone annotation line. trimmed must have already passed
// isLineLevelCommentOrMeta. Trailing newline is stripped if present.
func countBlocksOnLine(trimmed []byte) (comments, meta int8) {
	n := len(trimmed)
	if n == 0 {
		return
	}
	if trimmed[n-1] == '\n' {
		n--
		trimmed = trimmed[:n]
	}

	i := 0
	for i < n {
		for i < n && (trimmed[i] == ' ' || trimmed[i] == '\t') {
			i++
		}
		if i >= n || trimmed[i] != '(' {
			break
		}
		i++ // consume '('

		double := i < n && trimmed[i] == '('
		if double {
			i++ // consume second '('
		}

		closed := false
		for i < n {
			b := trimmed[i]
			if double {
				if b == ')' && i+1 < n && trimmed[i+1] == ')' {
					i += 2
					closed = true
					break
				}
			} else {
				if b == ')' {
					i++
					closed = true
					break
				}
			}
			i++
		}
		if !closed {
			break
		}
		if double {
			meta++
		} else {
			comments++
		}
	}
	return
}

// countSuffixAnnotations counts annotation blocks at the end (suffix) of
// raw field bytes. Used for unquoted fields where the stream does not
// consume suffix annotations. Returns the number of comment and metadata
// suffix blocks found.
func countSuffixAnnotations(data []byte) (comments, meta int8) {
	trimmed := trimWhitespace(data)
	for {
		n := len(trimmed)
		if n < 2 || trimmed[n-1] != ')' {
			return
		}
		// Check double-paren suffix (metadata) first
		if n >= 4 && trimmed[n-2] == ')' {
			if start := lastIndexDoubleOpen(trimmed, n-2); start != -1 {
				inner := trimmed[start+2 : n-2]
				if !hasForbiddenBlockContent(inner) {
					meta++
					trimmed = trimWhitespace(trimmed[:start])
					continue
				}
			}
		}
		// Check single-paren suffix (comment)
		if start := lastIndexSingleOpen(trimmed, n-1); start != -1 {
			inner := trimmed[start+1 : n-1]
			if !hasForbiddenBlockContent(inner) {
				comments++
				trimmed = trimWhitespace(trimmed[:start])
				continue
			}
		}
		return
	}
}

// stripMetadataBlocksBytes removes inline ( comment ) and (( metadata )) blocks
// from unquoted field bytes. Blocks are stripped only at field boundaries.
func stripMetadataBlocksBytes(data []byte) []byte {
	trimmed := trimWhitespace(data)
	if len(trimmed) == 0 {
		return trimmed
	}

	hasOpenParen := false
	for _, b := range trimmed {
		if b == '(' {
			hasOpenParen = true
			break
		}
	}
	if !hasOpenParen {
		return trimmed
	}

	data = trimmed
	for {
		n := len(trimmed)
		if n == 0 {
			return trimmed
		}
		canHavePrefix := trimmed[0] == '('
		canHaveSuffix := trimmed[n-1] == ')'

		if canHavePrefix && len(trimmed) >= 4 && trimmed[0] == '(' && trimmed[1] == '(' {
			if end := indexDoubleClose(trimmed, 2); end != -1 {
				inner := trimmed[2:end]
				if !hasForbiddenBlockContent(inner) {
					data = trimmed[end+2:]
					trimmed = trimWhitespace(data)
					continue
				}
			}
		}

		if canHavePrefix && len(trimmed) >= 2 && trimmed[0] == '(' {
			if end := indexSingleClose(trimmed, 1); end != -1 {
				inner := trimmed[1:end]
				if !hasForbiddenBlockContent(inner) {
					data = trimmed[end+1:]
					trimmed = trimWhitespace(data)
					continue
				}
			}
		}

		if canHaveSuffix && len(trimmed) >= 4 && trimmed[len(trimmed)-2] == ')' && trimmed[len(trimmed)-1] == ')' {
			if start := lastIndexDoubleOpen(trimmed, len(trimmed)-2); start != -1 {
				inner := trimmed[start+2 : len(trimmed)-2]
				if !hasForbiddenBlockContent(inner) {
					data = trimmed[:start]
					trimmed = trimWhitespace(data)
					continue
				}
			}
		}

		if canHaveSuffix && len(trimmed) >= 2 && trimmed[len(trimmed)-1] == ')' {
			if start := lastIndexSingleOpen(trimmed, len(trimmed)-1); start != -1 {
				inner := trimmed[start+1 : len(trimmed)-1]
				if !hasForbiddenBlockContent(inner) {
					data = trimmed[:start]
					trimmed = trimWhitespace(data)
					continue
				}
			}
		}

		return trimmed
	}
}

func indexDoubleClose(data []byte, start int) int {
	for i := start; i+1 < len(data); i++ {
		if data[i] == ')' && data[i+1] == ')' {
			return i
		}
	}
	return -1
}

func indexSingleClose(data []byte, start int) int {
	for i := start; i < len(data); i++ {
		if data[i] == ')' {
			return i
		}
	}
	return -1
}

func lastIndexDoubleOpen(data []byte, limit int) int {
	for i := limit - 1; i-1 >= 0; i-- {
		if data[i-1] == '(' && data[i] == '(' {
			return i - 1
		}
	}
	return -1
}

func lastIndexSingleOpen(data []byte, limit int) int {
	for i := limit - 1; i >= 0; i-- {
		if data[i] == '(' {
			return i
		}
	}
	return -1
}

func hasForbiddenBlockContent(inner []byte) bool {
	for _, b := range inner {
		if b == '(' || b == ')' || b == '\n' || b == '\r' || b < 0x20 || b == 0x7f {
			return true
		}
	}
	return false
}

// matchPrefixIgnoreCaseASCII reports whether data begins with prefix,
// using ASCII-only case folding via bitmask (d|0x20).
// Prefix MUST be lowercase for this to work correctly.
func matchPrefixIgnoreCaseASCII(data, prefix []byte) bool {
	if len(data) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		d := data[i]
		p := prefix[i]

		// Convert only ASCII letters to lowercase via bitmask.
		// For non-letters, d|0x20 == d, so comparison still works.
		if d|0x20 != p {
			return false
		}
	}
	return true
}

// CurrentLine returns the physical line number where the current row started (1-indexed).
// This is used for accurate error reporting when parse errors occur mid-row.
// Returns the line where the current row being parsed started, not the current byte position.
func (p *CSVStream) CurrentLine() int {
	return p.rowStartLine
}

// SetHeaderMode sets or clears the header mode for the stream.
func (p *CSVStream) SetHeaderMode(enabled bool) {
	p.headerMode = enabled
}

// HeaderMode returns whether the stream is in header mode.
func (p *CSVStream) HeaderMode() bool {
	return p.headerMode
}

// CommentLineSkipped returns whether any # comment line was skipped before the
// current row and resets the counter.
func (p *CSVStream) CommentLineSkipped() bool {
	return p.CommentLinesSkipped() > 0
}

// CommentLinesSkipped returns the number of skipped # comment lines before the
// current row and resets the counter.
func (p *CSVStream) CommentLinesSkipped() int8 {
	count := p.skippedCommentLines
	p.skippedCommentLines = 0
	return count
}

// HeaderAttachableCommentLines returns the number of skipped # comment lines that are
// eligible to attach to the first header field, and resets that counter.
func (p *CSVStream) HeaderAttachableCommentLines() int8 {
	count := p.headerAttachableCommentLines
	p.headerAttachableCommentLines = 0
	p.headerCommentAttachWindow = false
	return count
}

// ResetSkipTracking resets the header skip tracking flags.
func (p *CSVStream) ResetSkipTracking() {
	p.headerCommentAttachWindow = false
	p.headerAttachableCommentLines = 0
}

// StandaloneAnnotationsSkipped returns the counts of standalone comment and
// metadata annotation lines skipped by skipLeadingBlanks during data-row reading,
// and resets both counters. Used by the continuation path to detect duplicate
// annotations across standalone lines and inline annotations on the same field.
func (p *CSVStream) StandaloneAnnotationsSkipped() (comments, meta int8) {
	comments = p.skippedStandaloneComments
	meta = p.skippedStandaloneMeta
	p.skippedStandaloneComments = 0
	p.skippedStandaloneMeta = 0
	return
}
