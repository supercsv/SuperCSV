# SuperCSV Error Message Examples

**Version**: v1.0.0  
**Last Updated**: 2026-04-18

## Purpose

This document provides teaching examples for major error categories in the SuperCSV validator. Each example shows valid input, invalid input, the exact error output, and how to fix the problem.

For the complete error code catalog, see [error-codes.md](error-codes.md).

---

## Table of Contents

- [Structural Errors](#structural-errors)
- [Type Validation Errors](#type-validation-errors)
- [Container Errors](#container-errors)
- [Prefix Errors](#prefix-errors)
- [Enum Errors](#enum-errors)
- [Temporal Errors](#temporal-errors)
- [System Errors](#system-errors)

---

## Structural Errors

Structural errors occur when CSV or container syntax is malformed.

### Example: Quote in Unquoted Field

**Valid CSV**:
```csv
Name:string,Age:int
John Doe,30
Jane Smith,25
```

**Invalid CSV** (quote in unquoted field):
```csv
Name:string,Age:int
Jo"hn,30
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Name,"invalid structure: quote in unquoted field"
```

**What Caused It**: The unquoted field value `Jo"hn` contains a quote character. In SuperCSV, quotes are only allowed in properly quoted fields or must be escaped.

**How to Fix**:
- Remove the quote: `John`
- Properly quote the field: `"Jo\"hn"` (with escaped quote)
- Use a different character if quote is not needed

---

### Example: Missing Closing Bracket

**Valid CSV**:
```csv
Numbers:[]int
[1,2,3]
[4,5,6]
```

**Invalid CSV** (missing closing bracket):
```csv
Numbers:[]int
[1,2,3
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Numbers,"invalid structure: unterminated array"
```

**What Caused It**: The array literal `[1,2,3` is missing the closing bracket `]`.

**How to Fix**:
- Add the closing bracket: `[1,2,3]`

---

### Example: Column Count Mismatch

**Valid CSV**:
```csv
Name:string,Age:int,City:string
John,30,NYC
Jane,25,LA
```

**Invalid CSV** (too many columns):
```csv
Name:string,Age:int,City:string
John,30,NYC,ExtraField
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,,"expected 3 columns, got 4"
```

**What Caused It**: The header defines 3 columns, but the data row has 4 fields.

**How to Fix**:
- Remove the extra field: `John,30,NYC`
- Or add a column to the header if the extra field is intentional

---

## Type Validation Errors

Type errors occur when a value cannot be parsed as the declared type.

### Example: Invalid Integer

**Valid CSV**:
```csv
Age:int
30
25
42
```

**Invalid CSV** (non-numeric value):
```csv
Age:int
abc
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Age,"invalid int value: abc"
```

**What Caused It**: The field is declared as `int` but contains the non-numeric value `abc`.

**How to Fix**:
- Provide a valid integer: `30`
- If the value should be null, use the null literal: `_`
- If the field should be a string, change the type declaration: `Age:string`

---

### Example: Invalid Boolean

**Valid CSV**:
```csv
Active:bool
true
false
1
0
```

**Invalid CSV** (invalid boolean value):
```csv
Active:bool
maybe
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Active,"invalid bool value: maybe"
```

**What Caused It**: Boolean fields only accept `true`, `false`, `1`, or `0`. The value `maybe` is not recognized.

**How to Fix**:
- Use a valid boolean: `true` or `false`
- Or use numeric form: `1` or `0`

---

## Container Errors

Container errors occur when list or array syntax is malformed or violates schema rules.

### Example: Wrong Array Length

**Valid CSV**:
```csv
Scores:[3]int
[10,20,30]
[15,25,35]
```

**Invalid CSV** (wrong element count):
```csv
Scores:[3]int
[10,20]
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Scores,"expected 3 elements, got 2"
```

**What Caused It**: The schema declares a fixed-size array `[3]int` (exactly 3 integers), but the data contains only 2 elements.

**How to Fix**:
- Add the missing element: `[10,20,30]`
- Or change the schema to variable-length: `[]int`

---

### Example: Nested Containers in List

**Valid CSV** (arrays can nest):
```csv
Matrix:[2,3]int
[[1,2,3],[4,5,6]]
```

**Invalid CSV** (lists cannot nest):
```csv
Names:list<string>
[["John","Jane"],["Bob","Alice"]]
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Names,"lists cannot contain nested containers (element 0)"
```

**What Caused It**: Lists in SuperCSV cannot contain other containers. The value contains nested arrays.

**How to Fix**:
- Flatten the list: `["John","Jane","Bob","Alice"]`
- Or use a 2D array instead: Change type to `[2,2]string` or `[][]string`

---

### Example: Missing Comma Between Elements

**Valid CSV**:
```csv
Numbers:[]int
[1,2,3,4]
```

**Invalid CSV** (missing comma):
```csv
Numbers:[]int
[1 2 3]
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Numbers,"expected comma between elements at position 3"
```

**What Caused It**: Array elements must be separated by commas. Spaces alone are not valid separators.

**How to Fix**:
- Add commas between elements: `[1,2,3]`

---

## Prefix Errors

Prefix errors occur when array/list prefix notation is invalid or mismatched.

### Example: Non-Positive Prefix Length

**Valid CSV**:
```csv
Numbers:[]int
[3][1,2,3]
[5][10,20,30,40,50]
```

**Invalid CSV** (zero prefix):
```csv
Numbers:[]int
[0][1,2,3]
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Numbers,"invalid prefix: non-positive length"
```

**What Caused It**: Prefix notation requires a positive integer. Zero and negative values are forbidden.

**How to Fix**:
- Use the correct positive prefix: `[3][1,2,3]`
- Or omit the prefix entirely: `[1,2,3]`

---

### Example: Prefix Mismatch

**Valid CSV**:
```csv
Values:[]int
[3][1,2,3]
```

**Invalid CSV** (prefix doesn't match actual length):
```csv
Values:[]int
[5][1,2,3]
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Values,"prefix length 5 does not match actual length 3"
```

**What Caused It**: The prefix declares 5 elements, but the array contains only 3.

**How to Fix**:
- Fix the prefix to match: `[3][1,2,3]`
- Or add missing elements: `[5][1,2,3,4,5]`

---

## Enum Errors

Enum errors occur when values don't match the enum definition.

### Example: Invalid Enum Label

**Schema** (in header):
```csv
Status:enum<0=pending,1=active,2=completed>
```

**Valid CSV**:
```csv
Status:enum<0=pending,1=active,2=completed>
pending
active
completed
```

**Invalid CSV** (label not in enum):
```csv
Status:enum<0=pending,1=active,2=completed>
cancelled
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Status,"invalid enum label: cancelled"
```

**What Caused It**: The value `cancelled` is not defined in the enum. Only `pending`, `active`, and `completed` are valid.

**How to Fix**:
- Use a valid enum label: `pending`, `active`, or `completed`
- Or add `cancelled` to the enum definition if it's a valid state

---

### Example: Invalid Enum Value

**Schema**:
```csv
Priority:enum<1=low,2=medium,3=high>
```

**Valid CSV**:
```csv
Priority:enum<1=low,2=medium,3=high>
1
2
low
high
```

**Invalid CSV** (numeric value not in enum):
```csv
Priority:enum<1=low,2=medium,3=high>
99
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Priority,"invalid enum value: 99"
```

**What Caused It**: The numeric value `99` is not mapped in the enum. Only `1`, `2`, and `3` are valid.

**How to Fix**:
- Use a valid enum value: `1`, `2`, or `3`
- Or use the label form: `low`, `medium`, or `high`

---

## Temporal Errors

Temporal errors occur when date, time, timestamp, or duration values are malformed.

### Example: Invalid Date Format

**Valid CSV**:
```csv
BirthDate:date
2000-01-15
1995-12-25
```

**Invalid CSV** (invalid month):
```csv
BirthDate:date
2026-13-01
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,BirthDate,"invalid date format: 2026-13-01"
```

**What Caused It**: Month `13` is invalid. Months must be between 1 and 12.

**How to Fix**:
- Use a valid month: `2026-12-01` (December 1st)
- Ensure date follows YYYY-MM-DD format

---

### Example: Invalid Time Format

**Valid CSV**:
```csv
StartTime:time
09:30:00
14:15:30
23:59:59
```

**Invalid CSV** (hours out of range):
```csv
StartTime:time
25:00:00
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,StartTime,"invalid time format: 25:00:00"
```

**What Caused It**: Hours must be between 00 and 23. The value `25` is out of range.

**How to Fix**:
- Use valid hours (00-23): `23:00:00` for 11 PM
- Ensure time follows HH:MM:SS format

---

### Example: Malformed Timestamp

**Valid CSV**:
```csv
Created:timestamp
2026-01-18T10:30:00Z
2026-01-18T10:30:00-05:00
```

**Invalid CSV** (missing time component):
```csv
Created:timestamp
2026-01-18
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Created,"invalid timestamp: 2026-01-18"
```

**What Caused It**: Timestamps require both date and time components. The value is missing the time.

**How to Fix**:
- Add time and timezone: `2026-01-18T00:00:00Z`
- Or use `date` type if only the date is needed

---

## System Errors

System errors occur at the file or stream level.

### Example: Row Still In Use

**Code Context**:
```go
for {
    row, err := reader.Next()
    if err != nil {
        break
    }
    // Process row...
    // FORGOT TO CALL row.Release()
}
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
3,,"row still in use"
```

**What Caused It**: The previous row buffer was not released before attempting to read the next row.

**How to Fix**:
```go
for {
    row, err := reader.Next()
    if err != nil {
        break
    }
    // Process row...
    row.Release() // ← Add this
}
```

---

### Example: NUL Byte Not Allowed

**Invalid CSV** (contains binary null byte):
```
Name:string,Age:int
John\x00Smith,30
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
2,Name,"NUL byte not allowed"
```

**What Caused It**: SuperCSV files must be valid UTF-8 text. Binary null bytes (0x00) are forbidden.

**How to Fix**:
- Remove the null byte from the data
- Ensure the file is saved as UTF-8 text without binary characters

---

### Example: UTF-8 BOM Not Allowed

**Invalid File** (starts with BOM):
```
EF BB BF Name:string,Age:int
John,30
```

**Error Output**:
```
Line:int,ErrorSection:string,ErrorMsg:string
1,,"utf-8 BOM not allowed"
```

**What Caused It**: The file starts with a UTF-8 BOM (Byte Order Mark). SuperCSV forbids BOMs.

**How to Fix**:
- Save the file as UTF-8 without BOM (most editors have this option)
- Remove the first 3 bytes (EF BB BF) from the file

---

## Summary

These examples cover the 7 major error categories in SuperCSV:

1. **Structural** - CSV/container syntax (quotes, brackets, delimiters)
2. **Type Validation** - Value doesn't match declared type
3. **Container** - List/array syntax and schema violations
4. **Prefix** - Array prefix notation errors
5. **Enum** - Invalid enum labels or values
6. **Temporal** - Date/time/timestamp format errors
7. **System** - File-level and stream errors

For the complete list of all 64 error codes, see [error-codes.md](error-codes.md).

---

## See Also

- [error-codes.md](error-codes.md) — Complete error code catalog

---

*Last updated: 2026-04-18*  
*SuperCSV v1.0.0*
