# DFA Type Validators

This package implements deterministic finite automaton (DFA) validators for all SuperCSV scalar types. DFAs enable zero-allocation validation by processing values byte-by-byte without materializing strings.

## Design Principles

1. **Zero Allocations**: All DFAs validate in-place without allocating memory
2. **Streaming**: Process one byte at a time, suitable for streaming CSV parsing
3. **Reusable**: DFAs can be Reset() and reused across multiple field validations
4. **Spec Compliant**: All implementations match v0.2.0 TwoPassValidator behavior exactly

## Performance

All DFA implementations achieve zero allocations:

```
BenchmarkAllScalarDFAs/bool-8           72833332        16.28 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/int-8           135659348         9.08 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/float-8          70581591        15.61 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/decimal-8        67504471        17.49 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/string-8         64928739        17.76 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/date-8           44352453        27.38 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/time-8           52692148        22.55 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/timestamp-8      22773985        61.09 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/bytes-8          39832701        32.39 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/uuid-8            8982586       135.20 ns/op      0 B/op      0 allocs/op
BenchmarkAllScalarDFAs/duration-8       51312092        21.77 ns/op      0 B/op      0 allocs/op
```

## Interface

All DFAs implement the `DFA` interface:

```go
type DFA interface {
    Reset()              // Reset to initial state for reuse
    Step(b byte) bool    // Process one byte, returns false if invalid
    Accept() bool        // Check if current state is accepting
    Error() string       // Get error message if rejected
}
```

## Usage

```go
// Create DFA for a scalar type
scalar := schema.ScalarType{Kind: schema.ScalarInt}
dfa := NewDFA(scalar)

// Validate a value byte-by-byte
value := "42"
dfa.Reset()
for i := 0; i < len(value); i++ {
    if !dfa.Step(value[i]) {
        // Invalid character encountered
        fmt.Println(dfa.Error())
        break
    }
}
if dfa.Accept() {
    // Value is valid
}

// Reuse for next field
dfa.Reset()
// ...
```

## Type Implementations

### BoolDFA
- Pattern: `true|false|1|0`
- Case-insensitive for text forms
- Examples: `true`, `True`, `FALSE`, `1`, `0`

### IntDFA
- Pattern: `[+-]?[0-9]+`
- Range: -9223372036854775808 to 9223372036854775807 (int64)
- Overflow detection during parsing
- Examples: `42`, `-42`, `+42`, `9223372036854775807`

### FloatDFA
- Pattern: `[+-]?[0-9]+(.[0-9]+)?([eE][+-]?[0-9]+)?`
- Special values (case-insensitive): `nan`, `inf`, `-inf`, `infinity`, `-infinity`
- NOTE: `+inf` and `+infinity` are NOT allowed (only unsigned or negative)
- Examples: `3.14`, `-3.14`, `1e5`, `1.23e-10`, `nan`, `-infinity`

### DecimalDFA
- Pattern: `[+-]?[0-9]+(\.[0-9]+)?`
- No scientific notation
- Examples: `12.345`, `-0.0001`, `+42.0`, `1000000.0000001`

### StringDFA
- Passthrough validator - always accepts
- No validation performed

### DateDFA
- Pattern: `YYYY-MM-DD` or `YYYY/MM/DD`
- Both dash and slash separators accepted
- Basic range validation: month 01-12, day 01-31
- Examples: `2025-01-09`, `2025/01/09`

### TimeDFA
- Pattern: `HH:MM:SS` or `HH:MM:SS.fffffffff`
- No timezone suffixes allowed
- Fractional seconds: 0-9 digits (nanosecond precision)
- Examples: `14:30:00`, `23:59:59.123`, `08:15:42.987654321`

### TimestampDFA
- Pattern: `(YYYY-MM-DD|YYYY/MM/DD)[T| ]HH:MM:SS(.fffffffff)?(Z|±HH:MM)?`
- Date separator: `-` or `/`
- Date/time separator: `T` or space
- Optional fractional seconds and timezone
- 8 layout combinations supported
- Examples: `2025-01-05 14:30:00`, `2025/01/05T14:30:00.987654321Z`

### BytesDFA
- Hex: `[0-9A-Fa-f]+` with even length
- Base64: standard encoding (padded or unpadded)
- **Rejects `0x` or `0X` prefix** (breaking change from earlier versions)
- Examples valid: `DEADBEEF`, `deadbeef`, `aGVsbG8=`
- Examples invalid: `0xDEADBEEF`, `ABC` (odd length)

### UUIDDFA
- Pattern: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
- 36 characters total (32 hex + 4 dashes)
- Example: `550e8400-e29b-41d4-a716-446655440000`

### DurationDFA
- Pattern: `P[n]D[T[n]H[n]M[n]S]` or `PT[n]H[n]M[n]S`
- Supported units: Days (D), Hours (H), Minutes (M), Seconds (S)
- NOT supported: Years (Y), Months (M in date part), Weeks (W)
- Fractional seconds allowed: `PT1.5S`
- Examples: `P2D`, `PT1H30M`, `PT90.5S`, `P1DT2H3M4.5S`

### TimezoneDFA
- Passthrough validator - always accepts
- Any string is valid per spec
- Examples: `UTC`, `America/New_York`, `+05:00`, empty string

### EnumDFA
- Trie-based matching for allowed labels and codes
- Per-column DFA constructed from enum specification
- Fast O(n) lookup where n = value length
- Example: Given enum `{red:1, green:2, blue:3}`, accepts `red`, `green`, `blue`, `1`, `2`, `3`

## v0.2.0 Spec Compliance

All DFA implementations have been updated to match the v0.2.0 spec alignment:

- ✅ **Date**: Both `-` and `/` separators
- ✅ **Timestamp**: Slash + space separator (8 layouts)
- ✅ **Float**: Accepts NaN/Inf special values (rejects +inf)
- ✅ **Decimal**: Rejects space after dot
- ✅ **Time**: Rejects timezone suffixes
- ✅ **Bytes**: Rejects 0x prefix
- ✅ **Duration**: Fractional seconds, excludes Y/M/W

## Testing

Comprehensive test coverage with 100+ test cases:
- Valid input acceptance tests
- Invalid input rejection tests
- Edge case handling (overflow, range limits, format variations)
- Factory method tests (NewDFA)

Run tests:
```bash
go test ./internal/validate/v3/dfa/...
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./internal/validate/v3/dfa/...
```

## Next Steps

These DFAs will be integrated into the streaming validator (Step 10) to enable:
1. Single-pass CSV parsing + validation
2. Zero-allocation validation for valid data
3. 50-70% performance improvement over v0.2.0
4. Multi-core parallel validation via chunk-based processing
