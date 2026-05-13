// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

// rawField represents a parsed CSV field using deferred materialization.
// The field is stored as chunk references, avoiding memory allocation for the common case.
//
// Performance optimization: The first segment is inlined to avoid allocating
// a []fieldSegment slice for single-chunk fields (~99% of all fields).
// Multi-chunk fields allocate the extraSegments slice only when needed.
type rawField struct {
	// First segment (inlined to avoid allocation)
	firstChunk int // Chunk index
	firstStart int // Start offset in chunk (inclusive)
	firstEnd   int // End offset in chunk (exclusive)

	extraSegments []fieldSegment // Additional segments for multi-chunk fields (nil for single-chunk)
	Quoted        bool           // Whether field was quoted
	CommentBlocks int8           // Number of comment ( ) annotation blocks on this field
	MetaBlocks    int8           // Number of metadata (( )) annotation blocks on this field
	parser        *CSVStream     // Reference to parser for materialization
}

// fieldSegment represents additional segments for multi-chunk fields.
// Only allocated when a field spans multiple chunks (rare).
type fieldSegment struct {
	chunk int // Index into chunkBuffer.chunks
	start int // Start offset within chunk.data
	end   int // End offset (exclusive) within chunk.data
}

// HOT_PATH -> zero allocations, ASCII only, no []byte->string
// Bytes materializes the field into a contiguous []byte.
// For single-chunk fields, returns a slice directly into chunk memory (zero-copy).
// For multi-chunk fields, allocates and copies segments into materialized buffer.
func (f *rawField) Bytes() []byte {
	if f.parser == nil {
		return nil
	}
	// Fast path: single-chunk field (common case ~99%)
	if len(f.extraSegments) == 0 {
		chunk := f.parser.buffer.chunks[f.firstChunk]
		data := chunk.data[f.firstStart:f.firstEnd]
		if f.Quoted {
			// Common case: no trailing content after closing quote (comma immediately follows).
			// Slow path: trailing whitespace or annotation present - strip it.
			if len(data) > 0 && data[len(data)-1] == '"' {
				return data
			}
			return stripMetadataBlocksBytes(data)
		}
		return stripMetadataBlocksBytes(data)
	}
	// Multi-chunk field: materialize by copying all segments
	data := f.parser.materializeSegments(f)
	if f.Quoted {
		if len(data) > 0 && data[len(data)-1] == '"' {
			return data
		}
		return stripMetadataBlocksBytes(data)
	}
	return stripMetadataBlocksBytes(data)
}

// RawBytes returns the raw field bytes without any annotation stripping or
// whitespace trimming. Used by continuation logic which must see syntactic
// content (e.g. inline comments) that Bytes() would strip.
// Zero allocations for single-chunk fields.
func (f *rawField) RawBytes() []byte {
	if f.parser == nil {
		return nil
	}
	if len(f.extraSegments) == 0 {
		chunk := f.parser.buffer.chunks[f.firstChunk]
		return chunk.data[f.firstStart:f.firstEnd]
	}
	return f.parser.materializeSegments(f)
}

// AnnotationCounts returns the total comment and metadata annotation block
// counts for this field, combining stream-level counts (prefix, standalone,
// quoted-suffix) with suffix blocks counted from raw bytes (unquoted only).
func (f *rawField) AnnotationCounts() (comments, meta int8) {
	comments = f.CommentBlocks
	meta = f.MetaBlocks
	if !f.Quoted {
		sc, sm := countSuffixAnnotations(f.RawBytes())
		comments += sc
		meta += sm
	}
	return
}

// HasOnlyAnnotations returns true if the field is unquoted, has at least one
// annotation block counted by the stream (CommentBlocks or MetaBlocks > 0),
// and the raw bytes contain no actual value - only annotation blocks and whitespace.
// Used to detect annotation-after-trailing-comma errors.
func (f *rawField) HasOnlyAnnotations() bool {
	if f.Quoted || (f.CommentBlocks == 0 && f.MetaBlocks == 0) {
		return false
	}
	stripped := stripMetadataBlocksBytes(f.RawBytes())
	return len(stripped) == 0
}

type RawRow struct {
	Fields        []rawField
	Line          int
	RowEnd        int
	releaseCursor chunkCursor
	parser        *CSVStream
}

func (r *RawRow) Release() {
	if r == nil || r.parser == nil {
		return
	}
	r.parser.releaseRow(r.releaseCursor)
	r.parser = nil
	// Clear field segments without retaining references
	for i := range r.Fields {
		r.Fields[i].extraSegments = nil
		r.Fields[i].parser = nil
	}
	r.Fields = r.Fields[:0]
	r.Line = 0
	r.RowEnd = 0
}
