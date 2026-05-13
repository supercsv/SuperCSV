// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

/*
package supr provides a complete implementation of the SuperCSV v1.0 format.

SuperCSV is a typed, schema-enforced CSV format with support for:
  - 13 scalar types (int, float, string, bool, date, time, uuid, etc.)
  - Enums (name-only and value=name styles)
  - Containers (lists and arrays with 1D/2D dimensions)
  - Null values
  - Header schemas with type enforcement

# Quick Start

Validate a file:

	v := supr.NewValidator(r)
	for v.Next() {
	    if err := v.RowError(); err != nil {
	        log.Println(err)
	    }
	}
	if err := v.Err(); err != nil {
	    log.Fatal(err)
	}

Decode a file:

	d := supr.NewDecoder(r)
	for d.Next() {
	    row := d.Row()
	    // Process row
	}
	if err := d.Err(); err != nil {
	    log.Fatal(err)
	}

Encode a file:

	e := supr.NewFileEncoder(header, w)
	e.WriteHeader()
	for _, row := range rows {
	    e.WriteRow(row)
	}
	e.Close()

# Type System

SuperCSV has a rich type system with scalar types, enums, and containers.

Scalar types: String, Int, Float, Decimal, Bool, BytesHex, BytesB64,
Date, Time, Timestamp, Duration, Timezone, UUID

Enums use one of two uniform styles. All items must use the same form.
Decoded enum fields are returned as int32 - the 0-based declaration index into EnumSpec.Values.

	// Name-only style
	enumType := supr.Enum([]supr.EnumValue{
	    {Name: "low"},
	    {Name: "medium"},
	    {Name: "high"},
	})

	// Value=name style (value may be any Identifier, not just digits)
	enumType := supr.Enum([]supr.EnumValue{
	    {Name: "low",    Value: "0"},
	    {Name: "medium", Value: "5"},
	    {Name: "high",   Value: "10"},
	})

	// After decoding, use the int32 index to retrieve the name:
	// idx := row[col].(int32)
	// name := enumType.EnumSpec().Values[idx].Name

Containers support lists and arrays:

	listType := supr.List(supr.Int)                    // list<int>
	arrayType := supr.ArrFixed2D(supr.String, 3, 4)   // array<string>[3][4]

# Streaming vs Batch

The API is streaming-first for memory efficiency:

	// Streaming (constant memory)
	d := supr.NewDecoder(r)
	for d.Next() {
	    processRow(d.Row())  // Don't store, buffer reused
	}

	// Batch (loads all rows)
	file, err := supr.DecodeFile("data.csv")
	for _, row := range file.Rows {
	    processRow(row)  // Safe to store
	}

# Concurrency

Validator and Decoder instances are not safe for concurrent use. Each goroutine
must create its own instance with its own io.Reader.

Header, Type, EnumSpec, and EnumValue are immutable and safe to share across
goroutines.

# Resource Management

Validator and Decoder do NOT close the io.Reader.
Encoder.Close() flushes buffers but does NOT close the io.Writer.
The caller is responsible for closing all I/O resources.

# Specification

For the complete SuperCSV v1.0 specification, see:
  - internal/v1_0/spec/supercsv-spec-v1.0.md
  - internal/v1_0/spec/supercsv-type-table-v1.0.md
  - internal/v1_0/spec/supercsv-model-v1.0.md
*/
package supr
