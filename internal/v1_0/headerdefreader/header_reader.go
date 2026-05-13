// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package headerdefreader

import (
	"fmt"
	"io"

	csvstream "github.com/supercsv/supercsv/internal/v1_0/csvstream"
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
)

// ReadHeaderWithContinuation reads a header definition from the current stream
// position, including any continuation rows, and returns a Row for ParseHeader.
// The caller owns header-mode transitions before and after this read.
func ReadHeaderWithContinuation(stream *csvstream.CSVStream) (*headerdef.Row, error) {
	var accumulatedFields []string
	var accumulatedCommentCounts []int8
	var accumulatedMetaCounts []int8
	var startLine, endLine int
	inContinuation := false
	annotationAfterTrailingComma := false

	for {
		rawRow, err := stream.NextRow()
		commentLinesSkipped := stream.CommentLinesSkipped()
		attachableCommentLines := stream.HeaderAttachableCommentLines()
		standaloneComments, standaloneMeta := stream.StandaloneAnnotationsSkipped()
		if err != nil {
			if err == io.EOF {
				if inContinuation {
					return nil, fmt.Errorf("header continuation: unexpected EOF after comma")
				}
				return nil, fmt.Errorf("empty file: no header row")
			}
			return nil, fmt.Errorf("failed to read header: %w", err)
		}

		stream.ResetSkipTracking()

		if startLine == 0 {
			startLine = rawRow.Line
		}
		endLine = rawRow.RowEnd
		if endLine == 0 {
			endLine = rawRow.Line
		}

		hasTrailingComma := false
		if len(rawRow.Fields) > 0 && !rawRow.Fields[len(rawRow.Fields)-1].Quoted {
			lastField := rawRow.Fields[len(rawRow.Fields)-1].Bytes()
			isWhitespace := true
			for i := 0; i < len(lastField); i++ {
				if !asciiWhitespace[lastField[i]] {
					isWhitespace = false
					break
				}
			}
			hasTrailingComma = isWhitespace
		}

		if inContinuation && len(accumulatedFields) > 0 {
			lastField := accumulatedFields[len(accumulatedFields)-1]
			openBrackets := 0
			for _, ch := range lastField {
				if ch == '<' {
					openBrackets++
				} else if ch == '>' {
					openBrackets--
				}
			}
			if openBrackets > 0 {
				return nil, fmt.Errorf("header mid-type split: unclosed angle bracket at line %d", endLine)
			}
		}

		fieldCount := len(rawRow.Fields)
		if hasTrailingComma {
			if rawRow.Fields[len(rawRow.Fields)-1].HasOnlyAnnotations() {
				annotationAfterTrailingComma = true
			}
			fieldCount--
		}

		fields := make([]string, 0, fieldCount)
		commentCounts := make([]int8, 0, fieldCount)
		metaCounts := make([]int8, 0, fieldCount)
		for i := 0; i < fieldCount; i++ {
			field := rawRow.Fields[i]
			cc, mc := field.AnnotationCounts()
			if i == 0 {
				if inContinuation {
					cc += commentLinesSkipped
				} else {
					cc += attachableCommentLines
				}
				cc += standaloneComments
				mc += standaloneMeta
			}
			fields = append(fields, string(field.Bytes()))
			commentCounts = append(commentCounts, cc)
			metaCounts = append(metaCounts, mc)
		}

		rawRow.Release()

		accumulatedFields = append(accumulatedFields, fields...)
		accumulatedCommentCounts = append(accumulatedCommentCounts, commentCounts...)
		accumulatedMetaCounts = append(accumulatedMetaCounts, metaCounts...)

		if !hasTrailingComma {
			break
		}

		inContinuation = true
	}

	schemaRow := &headerdef.Row{
		Fields:                       accumulatedFields,
		CommentCounts:                accumulatedCommentCounts,
		MetaCounts:                   accumulatedMetaCounts,
		AnnotationAfterTrailingComma: annotationAfterTrailingComma,
		Line:                         startLine,
		EndLine:                      endLine,
	}

	if endLine == startLine {
		schemaRow.EndLine = 0
	}

	return schemaRow, nil
}

var asciiWhitespace = [256]bool{
	' ':  true,
	'\t': true,
	'\r': true,
	'\n': true,
}
