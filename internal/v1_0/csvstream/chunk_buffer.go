// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"bufio"
	"errors"
	"io"
	"sync"
)

type byteChunk struct {
	data []byte
	end  int
}

type chunkCursor struct {
	chunk  int
	offset int
	Col    int // Column position within the current line (0-based)
}

type chunkBuffer struct {
	reader    *bufio.Reader
	chunkSize int
	chunks    []*byteChunk
	cursor    chunkCursor
	line      int
	col       int // Current column position (0-based, reset on newline)
	hitEOF    bool
	pool      *sync.Pool // Pool for recycling byteChunk instances
}

func newChunkBuffer(r *bufio.Reader, size int) *chunkBuffer {
	return &chunkBuffer{
		reader:    r,
		chunkSize: size,
		line:      1,
		pool: &sync.Pool{
			New: func() interface{} {
				return &byteChunk{
					data: make([]byte, size),
				}
			},
		},
	}
}

// BYTE_LEVEL_HOT_PATH_DO_NOT_TOUCH -> 100M+ calls, I/O critical, CRLF normalization, maximally optimized
func (b *chunkBuffer) readByte() (byte, chunkCursor, int, error) {
	for {
		if b.cursor.chunk >= len(b.chunks) {
			if err := b.fillChunk(); err != nil {
				return 0, chunkCursor{}, 0, err
			}
		}
		chunk := b.chunks[b.cursor.chunk]
		if b.cursor.offset >= chunk.end {
			b.cursor.chunk++
			b.cursor.offset = 0
			continue
		}
		pos := chunkCursor{chunk: b.cursor.chunk, offset: b.cursor.offset, Col: b.col}
		value := chunk.data[b.cursor.offset]
		b.cursor.offset++
		line := b.line

		// Normalize CRLF: detect \r\n sequences and lone \r, return \n for both
		if value == '\r' {
			// Peek next byte to check for \n
			nextCursor := b.cursor
			for {
				if nextCursor.chunk >= len(b.chunks) {
					if b.hitEOF {
						// Lone \r at EOF
						break
					}
					if err := b.fillChunk(); err != nil {
						if errors.Is(err, io.EOF) {
							// Lone \r at EOF
							break
						}
						return 0, chunkCursor{}, 0, err
					}
				}
				nextChunk := b.chunks[nextCursor.chunk]
				if nextCursor.offset >= nextChunk.end {
					nextCursor.chunk++
					nextCursor.offset = 0
					continue
				}
				// Check if next byte is \n
				if nextChunk.data[nextCursor.offset] == '\n' {
					// Consume the \n
					b.cursor = nextCursor
					b.cursor.offset++
					if b.cursor.offset >= nextChunk.end {
						b.cursor.chunk++
						b.cursor.offset = 0
					}
				}
				break
			}
			value = '\n'
		}

		if value == '\n' {
			b.line++
			b.col = 0
		} else {
			b.col++
		}
		return value, pos, line, nil
	}
}

// BYTE_LEVEL_HOT_PATH_DO_NOT_TOUCH -> Called every 64KB, I/O critical, chunk pool management
func (b *chunkBuffer) fillChunk() error {
	if b.hitEOF {
		return io.EOF
	}

	// Retrieve chunk from pool or allocate new one
	chunk := b.pool.Get().(*byteChunk)
	// Reset chunk state for reuse
	chunk.data = chunk.data[:cap(chunk.data)]
	chunk.end = 0

	// Read directly into the chunk's backing array
	n, err := b.reader.Read(chunk.data)
	if n > 0 {
		chunk.data = chunk.data[:n]
		chunk.end = n
		b.chunks = append(b.chunks, chunk)
		if errors.Is(err, io.EOF) {
			b.hitEOF = true
		}
		return nil
	}

	if err != nil {
		// Return unused chunk to pool
		b.pool.Put(chunk)
		if errors.Is(err, io.EOF) {
			b.hitEOF = true
		}
		return err
	}

	// If we read zero bytes and no error, return to pool and try again
	b.pool.Put(chunk)
	return b.fillChunk()
}

// BYTE_LEVEL_HOT_PATH_DO_NOT_TOUCH -> Called per row, memory management critical
func (b *chunkBuffer) releaseUpTo(cursor chunkCursor) {
	if cursor.chunk > len(b.chunks) {
		cursor.chunk = len(b.chunks)
		cursor.offset = 0
	}
	// Return released chunks to pool
	for i := 0; i < cursor.chunk; i++ {
		if b.chunks[i] != nil {
			b.pool.Put(b.chunks[i])
			b.chunks[i] = nil
		}
	}
	if cursor.chunk > 0 {
		b.chunks = b.chunks[cursor.chunk:]
		b.cursor.chunk -= cursor.chunk
		if b.cursor.chunk < 0 {
			b.cursor.chunk = 0
		}
	}
	if len(b.chunks) == 0 {
		b.cursor = chunkCursor{}
		return
	}
	if cursor.offset == 0 {
		return
	}
	chunk := b.chunks[0]
	if cursor.offset >= chunk.end {
		b.chunks = b.chunks[1:]
		b.cursor.chunk--
		if b.cursor.chunk < 0 {
			b.cursor.chunk = 0
		}
		if len(b.chunks) == 0 {
			b.cursor = chunkCursor{}
			return
		}
		b.cursor.offset = 0
		return
	}
	copy(chunk.data[0:], chunk.data[cursor.offset:chunk.end])
	chunk.end -= cursor.offset
	if b.cursor.chunk == 0 {
		if b.cursor.offset >= cursor.offset {
			b.cursor.offset -= cursor.offset
		} else {
			b.cursor.offset = 0
		}
	}
}

// BYTE_LEVEL_HOT_PATH_DO_NOT_TOUCH -> Called frequently during field parsing, bounds checking
func (b *chunkBuffer) clampCursor(c chunkCursor) chunkCursor {
	if len(b.chunks) == 0 {
		return chunkCursor{}
	}
	if c.chunk >= len(b.chunks) {
		c.chunk = len(b.chunks) - 1
		c.offset = b.chunks[c.chunk].end
		return c
	}
	chunk := b.chunks[c.chunk]
	if chunk == nil {
		return chunkCursor{}
	}
	if c.offset > chunk.end {
		c.offset = chunk.end
	}
	return c
}
