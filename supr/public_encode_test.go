// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package supr_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/supercsv/supercsv/supr"
)

// TestEncodeStreamingWorkflow verifies NewFileEncoder -> WriteHeader -> WriteRow -> Close.
func TestEncodeStreamingWorkflow(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
		{Name: "name", Type: supr.String},
		{Name: "score", Type: supr.Float},
		{Name: "active", Type: supr.Bool},
	})

	var buf bytes.Buffer
	e := supr.NewFileEncoder(header, &buf)

	if err := e.WriteHeader(); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}

	rows := [][]any{
		{int64(1), "Alice", 95.5, true},
		{int64(2), "Bob", 87.3, false},
		{int64(3), "Charlie", 92.1, true},
	}
	for i, row := range rows {
		if err := e.WriteRow(row); err != nil {
			t.Fatalf("WriteRow %d: %v", i, err)
		}
	}

	if err := e.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify RowsWritten
	if e.RowsWritten() != 3 {
		t.Errorf("RowsWritten() = %d, want 3", e.RowsWritten())
	}

	// Compare against golden file
	golden := readTestdataFile(t, "roundtrip", "golden", "encode", "streaming_workflow.supr")
	if buf.String() != golden {
		t.Errorf("encoded output does not match golden file\ngot:\n%s\nwant:\n%s", buf.String(), golden)
	}

	// Verify output is valid SuperCSV by decoding it
	d := supr.NewDecoder(strings.NewReader(buf.String()))
	file, err := d.DecodeFile()
	if err != nil {
		t.Fatalf("re-decode failed: %v", err)
	}
	if len(file.Rows) != 3 {
		t.Fatalf("re-decoded %d rows, want 3", len(file.Rows))
	}
}

// TestEncodeContainersAndEnums verifies encoding of lists and enums.
func TestEncodeContainersAndEnums(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
		{Name: "tags", Type: supr.List(supr.String)},
		{Name: "grade", Type: supr.Enum([]supr.EnumValue{
			{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}, {Name: "F"},
		})},
	})

	var buf bytes.Buffer
	e := supr.NewFileEncoder(header, &buf)

	if err := e.WriteHeader(); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}

	// Enum by label string, list as []any
	row := []any{int64(1), []any{"math", "sci"}, "A"}
	if err := e.WriteRow(row); err != nil {
		t.Fatalf("WriteRow: %v", err)
	}

	// Null values
	rowNull := []any{int64(2), nil, nil}
	if err := e.WriteRow(rowNull); err != nil {
		t.Fatalf("WriteRow null: %v", err)
	}

	if err := e.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify round-trip
	golden := readTestdataFile(t, "roundtrip", "golden", "encode", "containers_and_enums.supr")
	if buf.String() != golden {
		t.Errorf("encoded output does not match golden file\ngot:\n%s\nwant:\n%s", buf.String(), golden)
	}

	d := supr.NewDecoder(strings.NewReader(buf.String()))
	file, err := d.DecodeFile()
	if err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if len(file.Rows) != 2 {
		t.Fatalf("re-decoded %d rows, want 2", len(file.Rows))
	}

	// Row 0: tags should decode back to []any{"math", "sci"}
	tags, ok := file.Rows[0][1].([]any)
	if !ok {
		t.Fatalf("row 0 tags type = %T, want []any", file.Rows[0][1])
	}
	if len(tags) != 2 {
		t.Fatalf("row 0 tags len = %d, want 2", len(tags))
	}

	// Row 1: null values should decode as nil
	if file.Rows[1][2] != nil {
		t.Errorf("row 1 grade = %v, want nil", file.Rows[1][2])
	}
}

// TestEncodeFileBatch verifies the EncodeFile convenience function.
func TestEncodeFileBatch(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "x", Type: supr.Int},
		{Name: "y", Type: supr.String},
	})

	file := &supr.File{
		Header: header,
		Rows: [][]any{
			{int64(10), "hello"},
			{int64(20), "world"},
		},
	}

	var buf bytes.Buffer
	if err := supr.EncodeFile(&buf, file); err != nil {
		t.Fatalf("EncodeFile: %v", err)
	}

	// Compare against golden file
	pubAssertEncodesTo(t, file, "roundtrip", "golden", "encode", "file_batch.supr")

	// Verify by decoding
	d := supr.NewDecoder(strings.NewReader(buf.String()))
	result, err := d.DecodeFile()
	if err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(result.Rows))
	}
	if got, ok := result.Rows[0][0].(int64); !ok || got != 10 {
		t.Errorf("row 0 col 0 = %v, want 10", result.Rows[0][0])
	}
}

// TestEncodeSmallHeaderTypes verifies WithSmallHeaderTypes option.
func TestEncodeSmallHeaderTypes(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
		{Name: "name", Type: supr.String},
		{Name: "ts", Type: supr.Timestamp},
	})

	var buf bytes.Buffer
	e := supr.NewFileEncoder(header, &buf, supr.WithSmallHeaderTypes())
	if err := e.WriteHeader(); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if err := e.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	output := buf.String()
	expect := "((SuperCSV v1.0))\nid:int,name:str,ts:ts\n"
	if !strings.HasPrefix(output, expect) {
		t.Errorf("small header output = %q, want prefix %q", output, expect)
	}
}

// TestEncodeTinyHeaderTypes verifies WithTinyHeaderTypes option.
func TestEncodeTinyHeaderTypes(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
		{Name: "name", Type: supr.String},
		{Name: "ts", Type: supr.Timestamp},
	})

	var buf bytes.Buffer
	e := supr.NewFileEncoder(header, &buf, supr.WithTinyHeaderTypes())
	if err := e.WriteHeader(); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if err := e.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	output := buf.String()
	expect := "((SuperCSV v1.0))\nid:i,name:s,ts:ts\n"
	if !strings.HasPrefix(output, expect) {
		t.Errorf("tiny header output = %q, want prefix %q", output, expect)
	}
}

// TestEncodeRowsWritten verifies RowsWritten tracks correctly.
func TestEncodeRowsWritten(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "v", Type: supr.Int},
	})

	var buf bytes.Buffer
	e := supr.NewFileEncoder(header, &buf)

	if e.RowsWritten() != 0 {
		t.Errorf("RowsWritten() before write = %d, want 0", e.RowsWritten())
	}

	if err := e.WriteHeader(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		if err := e.WriteRow([]any{int64(i)}); err != nil {
			t.Fatal(err)
		}
		if e.RowsWritten() != i+1 {
			t.Errorf("RowsWritten() after %d writes = %d, want %d", i+1, e.RowsWritten(), i+1)
		}
	}

	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestEncodeGoldenScalars encodes all scalar types and compares against encode_scalars.supr.
func TestEncodeGoldenScalars(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "a_int", Type: supr.Int},
		{Name: "b_float", Type: supr.Float},
		{Name: "c_string", Type: supr.String},
		{Name: "d_bool", Type: supr.Bool},
		{Name: "e_decimal", Type: supr.Decimal},
		{Name: "f_date", Type: supr.Date},
		{Name: "g_time", Type: supr.Time},
		{Name: "h_timestamp", Type: supr.Timestamp},
		{Name: "i_datetime", Type: supr.Datetime},
		{Name: "j_datetimetz", Type: supr.DatetimeTZ},
		{Name: "k_duration", Type: supr.Duration},
		{Name: "l_timezone", Type: supr.Timezone},
		{Name: "m_uuid", Type: supr.UUID},
		{Name: "n_bytes_hex", Type: supr.BytesHex},
		{Name: "o_bytes_b64", Type: supr.BytesB64},
	})

	rows := [][]any{
		{int64(42), 3.14, "hello", true, "123.456", "2024-01-15", "13:45:00", "2024-01-15T13:45:00", "2024-01-15T13:45:00", "2024-01-15T13:45:00+05:30", "PT1H30M", "+05:30", "550e8400-e29b-41d4-a716-446655440000", "48656C6C6F", "SGVsbG8="},
		{int64(-7), 0.0, "has,comma", false, "0.00", "2023-12-31", "23:59:59", "2023-12-31T23:59:59", "2023-12-31T23:59:59", "2023-12-31T23:59:59Z", "P2DT5H", "Z", "00000000-0000-0000-0000-000000000000", "FF", "/w=="},
	}

	var buf bytes.Buffer
	if err := supr.EncodeFile(&buf, &supr.File{Header: header, Rows: rows}); err != nil {
		t.Fatalf("EncodeFile: %v", err)
	}

	golden := readTestdataFile(t, "encode", "encode_scalars.supr")

	if buf.String() != golden {
		t.Errorf("encoded output does not match golden file\ngot:\n%s\nwant:\n%s", buf.String(), golden)
	}
}

// TestEncodeGoldenContainers encodes container types and compares against encode_containers.supr.
func TestEncodeGoldenContainers(t *testing.T) {
	header := supr.NewHeader([]supr.Column{
		{Name: "a_list", Type: supr.List(supr.String)},
		{Name: "b_list_fixed", Type: supr.ListFixed(supr.Int, 3)},
		{Name: "c_arr", Type: supr.Arr(supr.Float)},
		{Name: "d_arr_1d", Type: supr.ArrFixed1D(supr.Int, 2)},
		{Name: "e_arr_2d", Type: supr.ArrFixed2D(supr.Int, 2, 3)},
		{Name: "f_enum", Type: supr.Enum([]supr.EnumValue{
			{Name: "low"}, {Name: "med"}, {Name: "high"},
		})},
		{Name: "g_enum_coded", Type: supr.Enum([]supr.EnumValue{
			{Name: "S", Value: "1"}, {Name: "M", Value: "2"}, {Name: "L", Value: "3"},
		})},
	})

	rows := [][]any{
		{
			[]any{"hello", "world"},             // list<string>
			[]any{int64(1), int64(2), int64(3)}, // list<int>[3]
			[]any{1.1, 2.2, 3.3},                // arr<float> dynamic
			[]any{int64(10), int64(20)},         // arr<int>[2]
			[]any{[]any{int64(1), int64(2), int64(3)}, []any{int64(4), int64(5), int64(6)}}, // arr<int>[2,3]
			"low", // enum label
			"S",   // enum coded
		},
		{
			[]any{"foo"},                        // list<string>
			[]any{int64(4), int64(5), int64(6)}, // list<int>[3]
			[]any{9.9, 8.8},                     // arr<float> dynamic
			[]any{int64(30), int64(40)},         // arr<int>[2]
			[]any{[]any{int64(7), int64(8), int64(9)}, []any{int64(10), int64(11), int64(12)}}, // arr<int>[2,3]
			"high", // enum label
			"L",    // enum coded
		},
	}

	var buf bytes.Buffer
	if err := supr.EncodeFile(&buf, &supr.File{Header: header, Rows: rows}); err != nil {
		t.Fatalf("EncodeFile: %v", err)
	}

	golden := readTestdataFile(t, "encode", "encode_containers.supr")

	if buf.String() != golden {
		t.Errorf("encoded output does not match golden file\ngot:\n%s\nwant:\n%s", buf.String(), golden)
	}
}
