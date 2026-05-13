# SuperCSV v1.0 — Example Files

This collection demonstrates the expressive range of SuperCSV using clean, real-world examples.

## Quick Glance Example
```
# Minimal mixed-type table
Name:string, Age:int, Active:bool, Locations:list<string>

Alice, 30, true, [team,remote]
Bob, 25, false, [onsite]
```

Quoted strings follow standard CSV escaping. Example with an embedded quote:

```
Name:string, Quote:string

Ada, "She said ""hi"""
```

## SuperCSV Example Checklist

- Header names follow `[A-Za-z0-9][A-Za-z0-9_-]*` and reference only types from `supercsv-type-table-v1.0.md`.
- Column counts match across header and every data row (ignoring blank/comment lines).
- Comments and blank lines do not affect table structure or row alignment.
- String quoting rules: commas/whitespace/hash/brackets/quotes/semicolons/control chars -> quoted, otherwise unquoted.
- Numeric and boolean literals appear unquoted and use only allowed forms for their declared type.
- Lists and arrays obey container rules:
  - Element types align with the header.
  - Length/shape constraints (`list<T>[N]`, `arr<T>[N]`, `arr<T>[R,C]`) are honored.
  - Value-level prefixes (`[N][...]`, `[R,C][[...]]`) are used only when the header has no fixed size/shape and never for empty containers.
  - No trailing commas; no comments inside containers; no mixed quoting styles within a container.
  - Nulls (`_`) count toward fixed lengths.
- 2D arrays appear only as:
  - raw (`arr<T>` with a 2D literal),
  - shape-constrained (`arr<T>[R,C]`), or
  - prefix-annotated (`[R,C][[...]]`).
- Null literal `_` appears only where the type permits nulls.
- Enums use either declared labels or allowed numeric codes.
- Multi-line or pretty-printed containers may appear only as documentation-only formatting and must be clearly marked invalid.

---

## Examples
SuperCSV includes a set of real-world `.supr` files that exercise every v1.0 feature. All live in `spec/examples/`:

- `people.supr` — simple typed people table
- `inventory.supr` — arrays per row
- `enums-labels.supr` — simple label enums
- `enums-numbered.supr` — numeric-code enums
- `enums-containers.supr` — enums inside list/array containers
- `lists.supr` — list containers
- `mixed-containers.supr` — arrays + lists in one schema
- `inline-comments.supr` — heavy comment usage / logical sections
- `fixed-length-arrays.supr` — fixed `arr<T>[N]`
- `arrays-with-prefix.supr` — value-level array prefixes
- `strings.supr` — exhaustive string quoting/escaping cases
- `matrices-raw.supr` — raw 2D arrays
- `matrices-shape.supr` — shape-constrained 2D arrays
- `matrices-prefix.supr` — prefix-annotated 2D arrays
- `edge-cases-empty-containers.supr` — empty list/array samples without prefixes
- `edge-cases-zero-row-matrix.supr` — contrast zero-row vs one-row-zero-col matrices
- `edge-cases-fixed-array-nulls.supr` — nulls counted inside fixed-length arrays
- `edge-cases-2d-array-nulls.supr` — null placements inside 2D arrays
- `everything-everywhere.supr` — kitchen-sink table mixing comments, blanks, enums, nulls, and constrained containers
- `everything-everywhere-all-at-once.supr` — 50-row type zoo covering nearly every built-in type

---

## Edge Case Snippets

### Empty Containers (No Prefix)
```
Tags:list<string>
[]          # valid empty list

Values:arr<int>
[]          # valid empty array

[0][]        # invalid — value-level prefix forbidden for empty containers
```

### Zero-Row vs. One-Row-Zero-Col Matrix
```
# Zero rows, zero columns
Matrix:arr<float>
[]

# One row with zero columns (not the same as above)
Matrix:arr<float>
[[]]
```

### Nulls Inside Fixed-Length Arrays
```
RGB:arr<float>[3]
[1.0, _, 0.5]   # null counts toward the fixed length
```

### Nulls Inside 2D Arrays
```
Matrix:arr<float>
[[1.0, _], [_, 4.0]]
```

---

### Pretty-Printed 2D Arrays (Documentation Only)
```
# 3x3 matrices (pretty-printed, not valid SuperCSV syntax)
Name:string, Matrix:arr<float>

Identity, [
  [1.0, 0.0, 0.0],
  [0.0, 1.0, 0.0],
  [0.0, 0.0, 1.0]
]

Rotate90, [
  [0.0, -1.0, 0.0],
  [1.0,  0.0, 0.0],
  [0.0,  0.0, 1.0]
]

Scale2x, [
  [2.0, 0.0, 0.0],
  [0.0, 2.0, 0.0],
  [0.0, 0.0, 1.0]
]
```

---
