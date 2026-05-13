# **SuperCSV v1.0 Specification**

"a transformational reinterpretation of CSV" — *self‑proclaimed father of modern CSV*

A minimal, typed, human‑readable, machine‑parseable tabular data format.  
Designed for clarity, lossless read/write, and wide compatibility.

---

## Overview

- A minimal, typed, human-readable, machine-parseable tabular data format
- Files consist of a single header row followed by zero or more data rows
- The header row declares column names and types; column types are defined in the companion Type Table
- Columns are comma-separated; whitespace around delimiters is ignored
- Comments and blank lines may appear where allowed by the structural rules
- Null `_` represents missing or undefined value — valid anywhere a value is expected
- Declared version directive

---

## Core File Rules

### Extension

SuperCSV files use the `.supr` extension.

---

### Version Declaration

A SuperCSV file MUST begin with a version declaration:

```
((SuperCSV v1.0))
```

Rules:

- This line **MUST** appear before any header, comment, or blank line.
- The directive name `SuperCSV` is matched using ASCII case‑insensitive comparison.
- The version identifier **MUST** match exactly `v1.0`.
- The version declaration is treated as a meta line for parsing purposes.
- Spacing follows the examples shown exactly.

---

### File Structure

A SuperCSV file consists of:

1. One **header row** (required)  
2. Zero or more **data rows**  
3. Zero or more **comment lines**  
4. Zero or more **blank lines**

Comment lines and blank lines may appear anywhere after the version directive line.

---

### Encoding

UTF‑8 without BOM.  

---

### Whitespace

- Leading and trailing whitespace around a field is ignored
- Leading and trailing whitespace inside an unquoted string is ALWAYS trimmed
- Internal whitespace in unquoted strings is preserved
- Whitespace inside quoted strings is preserved exactly
- Whitespace around commas is ignored
- Whitespace inside list and array brackets is ignored when it appears
  between tokens ([, ], ,, or a value); whitespace that is part of an
  unquoted string value is preserved
- Whitespace around : in the header row is ignored
- After trimming, the null literal must be exactly _
- These rules work together with the unquoted‑string rules in [String Values](#string-values)
- Newlines are not general whitespace. They are permitted only immediately after a comma between fields, inside quoted strings, or at permitted positions within a container field (see [Multi‑Line Container Values](#multi-line-container-values)).

#### Structural whitespace

Only **ASCII space (U+0020)** and **ASCII tab (U+0009)** are treated as structural whitespace. These characters may appear around commas, around `:`, and at the edges of unquoted fields, and are trimmed or ignored according to the normal parsing rules. All other Unicode whitespace characters (for example U+00A0 NO‑BREAK SPACE or U+2003 EM SPACE) are **not** structural and are never skipped by the parser.

#### Invisible and visually space-like characters inside values

Some characters are invisible or visually space-like at value edges and are hard to review reliably in unquoted text. These characters are allowed only when they are unambiguously part of the value:

- **Inside quoted strings** — preserved exactly as written.
- **Inside unquoted strings, but not at the edges** — for example `Jean\u00A0Paul` is valid and the NBSP is part of the value.

Any character in the canonical edge-invalid set is invalid at the **leading or trailing edge** of an unquoted string. To represent such content at either edge, the value **must** be written as a quoted string.

Canonical edge-invalid set:

- **Unicode whitespace set**: `U+000B`, `U+000C`, `U+0085`, `U+00A0`, `U+1680`, `U+2000–U+200A`, `U+2028`, `U+2029`, `U+202F`, `U+205F`, `U+3000`
- **Additional invisible format characters**: `U+200B`, `U+200C`, `U+200D`, `U+2060`, `U+FEFF`

These characters remain valid inside quoted strings and remain valid inside unquoted strings when they are internal to the value rather than at either edge.

#### Empty fields

Unquoted empty fields are invalid in v1.0 (and also reserved).

(Implementations may, through options, allow unquoted empty fields to be treated as null to aid with reading legacy CSV files. This behaviour is outside the v1.0 specification.)

---

## Lexical Constructs

### Identifiers

Identifiers are used for:

- Column names  
- EnumItem names  
- Type parameters  

Grammar:

```
identifier ::= [A-Za-z0-9][A-Za-z0-9_-]*
```

Rules:

- ASCII only
- MUST NOT exceed 255 characters
- Must not be quoted  
- Must not contain leading, trailing, or internal whitespace  
- Must not contain spaces  
- Must not contain punctuation other than `_` and `-`  
- Case‑sensitive  
- Any whitespace adjacent to an identifier is not part of the identifier and makes it invalid
- `_` is reserved as the null literal and cannot be used as an identifier or enum name

Identifiers are **not** a value type.

Examples (valid):
```
Name
UserId
HTTPStatus
x
x1
snake_case
foo_
kebab-case
1Name
2026Data
6set
3dModel
```

Examples (invalid):
```
_            # reserved null literal
 Name        # leading whitespace
Name         # trailing whitespace
First Name   # internal whitespace
"Name"       # quoted
Name!        # invalid punctuation
```

---

### Delimiters

Columns are separated by commas.  
Whitespace around commas is ignored.  
Trailing commas are not permitted.
Newlines are row delimiters and terminate the current row, except when a newline appears after a comma and before the next value (i.e., between fields).
Newlines inside a quoted string are a valid part of that value.

---

### Comments and Metadata

SuperCSV supports **comments** (human‑facing) and **metadata syntax** (machine‑facing), in both **line‑level** and **inline (field‑level)** forms.  
Comments and metadata syntax never change row or column structure.

Comments are for human readability only. They do not affect validation, decoding, encoded output, or data meaning.

**Metadata is reserved syntax in v1.0.** The `(( … ))` form is recognised as metadata syntax so positional and structural rules can be applied correctly. In v1.0, only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning. Any other metadata block is recognised syntax in v1.0, but has no defined meaning.

In v1.0, comments and non-version metadata are part of the file syntax and are recognised during parsing, but they are not part of the decoded data model. They do not affect validation, data meaning, or encoded output, and implementations must not preserve or emit them.

The metadata form is reserved for future versions, which will define metadata rules.

#### Design Intent (for explanation)

v1.0 defines the syntax and positioning rules for comments and metadata up front so every implementation parses the format the same way. Comments are allowed in v1.0. Metadata syntax is recognised as part of that same parsing model, but only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning in v1.0. This keeps the parser stable now and leaves a clear path for future versions to add metadata rules without redesigning the core format.

Non-version metadata has no defined meaning in v1.0, but its positional and structural rules are defined in advance so future versions can introduce metadata in a consistent and backward-compatible way. If metadata is added in later versions, it will follow these established rules.

Examples and tests may include non-version metadata blocks to verify parser correctness under the defined positional and structural rules. Such cases do not represent meaningful metadata or any v1.0 semantic behaviour.

#### Line‑Level Forms

Three line‑level forms are recognised:

- `#` — line comment
- `( … )` — line comment
- `(( … ))` — line metadata syntax

A line is treated as a **line‑level comment or metadata** when it meets one of the following criteria after trimming leading ASCII structural whitespace (space and tab):

- `# …` — if the first non‑whitespace character on the physical line is `#`, the line is a line‑level comment regardless of other content after `#`.
- `( … )` — if, after trimming leading and trailing ASCII structural whitespace, the physical line consists of **only** a single `( … )` block, it is a line‑level comment. If additional non‑whitespace content appears outside the block, the line is a data or header line with an inline prefix comment — not a line‑level comment.
- `(( … ))` — if, after trimming leading and trailing ASCII structural whitespace, the physical line consists of **only** a single `(( … ))` block, it is treated as line‑level metadata syntax. If additional non‑whitespace content appears outside the block, the line is a data or header line with an inline prefix metadata block — not line‑level metadata syntax.

Leading and trailing ASCII structural whitespace around a standalone line comment or line metadata block is ignored when determining whether the physical line is a line‑level comment or metadata-syntax line. Internal whitespace inside the block is preserved exactly, subject to the normal block‑content rules.

- **Line comment:**

A line comment may take two forms

  ```text
  # some text
  ```

  ```text
  (some text)
  ```
Style recommendation:

- Use # for section breaks or prominent comments.
- Use ( … ) for softer, less intrusive comments, especially on continuation lines.

- **Line metadata syntax:**

  ```text
  ((SuperCSV v1.0))
  ```

Rules:

- Line comments and line metadata syntax blocks are **entire lines** and never part of a row.
- Line comments may appear anywhere in the file, except where strict‑mode version rules apply.
- In v1.0, `(( … ))` metadata blocks follow defined positional rules aligned with comment placement rules where applicable, but only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning.
- Metadata syntax is recognised so structural validation can distinguish metadata blocks from data. Any non-version metadata block is recognised in v1.0, but has no defined meaning.

---

#### Inline Forms (Field‑Level)

Inline comment blocks relate to a **single field value** in data rows, or to a **single HeaderField** in header rows.

- A field or header field may have at most one comment block
- A comment may appear before or after the field or header field
- Inline comment blocks may contain empty or whitespace‑only text. Empty blocks are valid.
- Leading/trailing whitespace is removed, internal whitespace is preserved exactly

Form:

  ```text
  ( some text )
  ```

Examples (all valid):

```text
42 (approx)
(approx) 42
() 42
```

Inline metadata `(( … ))` blocks follow the same structural and positional rules as the corresponding inline comment form where applicable, but in v1.0 only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning. Any other inline metadata block is recognised in v1.0, but has no defined meaning.

---

#### Forbidden Placements

Inline comment blocks **cannot** appear:

- between list/array elements  
- inside quoted strings  
- inside containers  
- inside type declarations  
- inside identifiers  

Inline comments never change field count, container shape, or type structure. The same restrictions apply to metadata blocks.

---

#### Bracketing Rules

- `(` begins an inline comment block.  
- `((` begins an inline metadata block.  
- Inline comment and inline metadata blocks **do not nest**.  
- A block ends at the first corresponding closing delimiter:  
  - `)` closes `( … )`  
  - `))` closes `(( … ))`  
- Any additional `(` appearing before the block's closing delimiter makes the block invalid.  
- Block contents are interpreted literally; no escaping or special sequences are recognised.

---

#### Allowed / Forbidden Characters Inside Blocks

**Allowed:**  
All characters except those listed below.

**Forbidden inside `( … )` and `(( … ))`:**

- `(`  
- `)`  
- newline  
- control characters  

These characters terminate or invalidate the block.

---

#### Comment Scope

Comments relate to either **rows** or **fields**:

- **Row‑level:** A line‑level comment appearing between completed rows (no trailing comma pending) relates to the next row as a whole.
- **Field‑level:** An inline comment on the same line as a field value, or a line‑level comment appearing mid‑continuation (trailing comma pending), relates to a specific field.

Constraints:

- Each field may have at most one inline comment `( … )`.
- If a standalone comment line mid‑continuation and an inline comment on the field line would give the same field two comments, the file is invalid.

Metadata syntax `(( … ))` is recognised using the same positional rules where applicable, but in v1.0 only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning. Non-version metadata blocks are recognised in v1.0, but they do not define row-level or field-level metadata semantics.

---

### Blank Lines

A blank line is empty or contains only whitespace.  
Blank lines are ignored.

---

## Header Definition

The first non‑blank, non‑comment line is the **header row**.

### Syntax

```
HeaderField ::= Identifier ":" Type
HeaderRow   ::= HeaderField { "," HeaderField }
```

Type names are always written in lowercase

Whitespace around `:` and `,` is ignored.

---

### Multi‑Line Headers

Headers may span multiple lines for readability. If a header line ends with a comma (optionally followed by whitespace), the next line continues the same logical header row.

**Rules:**

- A header line ending with `,` (optionally followed by whitespace) continues to the next physical line  
- Continuation applies only during header parsing (before any data rows)  
- Continuation ends when a line does not end with a comma  
- Blank lines may appear in the middle of a multi‑line header and do not terminate the header continuation
- The combined lines must parse as a single `HeaderRow`  
- Continuation splits **between columns only** — type declarations cannot be broken across lines  

#### Comments in headers

- Line‑level comments (`# …` and `( … )`) may appear between header continuation lines and are skipped using the normal line‑level classification rule; they relate to the **next header field** (see [Comment Scope](#comment-scope)). Unlike data rows, both line‑level comment forms (`# …` and `( … )`) are permitted in headers.
- Standalone comment lines immediately attached to the header relate to the **next header field**, including the first header field.
- To keep comment lines file-level rather than attaching them to the first header field, separate them from the header with a blank line.
- Inline comments may appear before or after any header field; each field may have at most one comment (see [Inline Forms (Field‑Level)](#inline-forms-field-level))
- A standalone comment line and an inline comment on the same header field both count toward the one‑per‑field limit
- Inline comments are not permitted after the trailing comma

Metadata syntax `(( … ))` is recognised in the same structural positions as comments where applicable, but in v1.0 only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning. Any other metadata block in a header is recognised syntax, but has no defined meaning.

**Example:**

```supr
# Wide header definition split across lines
Id:int,
Name:string,
Tags:list<string>,
Scores:arr<float>[3],
Status:enum<pending,active,done>,
Notes:string
1,Alice,[work,urgent],[9.5,8.0,7.5],active,Needs review
```

Equivalent to:

```supr
Id:int,Name:string,Tags:list<string>,Scores:arr<float>[3],Status:enum<pending,active,done>,Notes:string
1,Alice,[work,urgent],[9.5,8.0,7.5],active,Needs review
```

---

### Column Names

Column names identify fields in the header definition.

Rules:

- MUST be valid identifiers (see [Identifiers](#identifiers))
- MUST NOT be quoted
- MUST NOT contain leading or trailing whitespace
- MUST NOT contain spaces or punctuation beyond `_` and `-`
- MUST be unique under case‑insensitive comparison
- Original casing is preserved for round‑tripping
- Convention: Column names use UpperCamelCase. Other styles are permitted but discouraged.
- No stylistic requirements are enforced by the parser

---

### Type References

Column types are defined in **SuperCsvTypeTable v1.0**.  
The header row must reference only types defined in that table.  
Type aliases are defined in the Type Table (see Type Aliases in SuperCsvTypeTable v1.0).

ScalarType, EnumType, and container types are defined as follows:

- ScalarType — any non‑container built‑in type from SuperCsvTypeTable v1.0
  (e.g. int, float, decimal, bool, string, bytes<hex>, bytes<b64>, date, time, datetime, datetimetz, timestamp, duration, timezone, uuid)

- EnumType — an inline enum declaration using `enum<...>`

- Container types — list<T> and arr<T>; container types MUST NOT nest
  (i.e., T cannot itself be a list or array)

---

## Value Definition

### String Values

Unquoted strings are allowed only if, after trimming leading and trailing
whitespace, the resulting value contains none of:

- `,`
- `#`
- `[` or `]`
- `(` or `)`
- `<` or `>`
- `{` or `}`
- `"`
- `'`
- `` ` `` (backtick)
- `;`
- `:`
- `=`
- `?`
- `/` or `\`
- `|`
- `@`
- control characters (tabs, newlines)

Unquoted strings may contain internal spaces. Any leading or trailing
whitespace in an unquoted string is always trimmed. If leading or trailing
whitespace must be preserved, a quoted string must be used.

In addition, an unquoted string must not begin or end with any character from
the canonical edge-invalid set defined in [Invisible and visually space-like characters inside values](#invisible-and-visually-space-like-characters-inside-values). Those characters are allowed inside quoted strings and when internal to an unquoted string, but they must be quoted if they appear at either edge.

Quoted strings use the standard double‑quote form:

- "value"
- doubled quotes represent a literal quote: ""
- may contain any UTF‑8
- empty string is ""

After trimming leading and trailing whitespace:
- An unquoted empty field is invalid unless it is exactly `_` (the null literal)
- A quoted empty string `""` is a valid empty string (distinct from null)
- Quoted strings preserve all internal whitespace exactly as written

---

### Numeric Values

Must be valid literals for their declared type.  
Numeric values must not be quoted.
Null literal `_` allowed for all numeric items.

---

### Boolean Values

Allowed forms are defined in the type table.

---

### List Values

Null literal `_` allowed for Lists

Basic syntax:

```
[item1,item2,item3]
```

Rules:

- Whitespace around commas is ignored
- Whitespace inside brackets is ignored
- These whitespace rules apply only to list syntax; they do not relax the unquoted‑string rules for string elements
- Items must match the list's element type
- Empty list is `[]` (only for dynamic-size lists)
- For fixed-size lists, [] is invalid because the value must match the declared size.
- Null literal `_` allowed as an item
- Container nesting is not permitted in SuperCSV v1.0.


Element type:  
T must be a ScalarType or EnumType.

Dynamic-size list:

```
list<T>
```

Fixed-size list:

```
list<T>[N]       # N > 0
```

Value‑level prefix (non‑empty only):

```
[N][item1,item2,item3]
```

Prefix form is not permitted for empty or fixed‑size containers in SuperCSV v1.0.

---

### Array Values

Null literal `_` is allowed for Arrays.

#### 1D array (`arr<T>[N]`)

```
[1,2,3,4]
```

#### 2D array (`arr<T>[R,C]`)

```
[[1,2,3],[4,5,6],[7,8,9]]
```

#### Dynamic-size array (`arr<T>`)

```
[1,2,3]                     # valid 1D
[[1,2],[3,4],[5,6]]         # valid 2D (rectangular)
```

Rules:

- Whitespace around commas is ignored
- Whitespace inside brackets is ignored
- These whitespace rules apply only to array syntax; they do not relax the unquoted‑string rules for string elements
- 2D arrays must have uniform row size
- Arrays may be 1D or 2D only
- No deeper nesting
- Container nesting is not permitted in SuperCSV v1.0.
- Empty array is `[]` (only for dynamic-size arrays)
- For fixed-size arrays, [] is invalid because the value must match the declared size.
- Null literal `_` allowed as element
- Dynamic-size arrays (`arr<T>`) may contain either 1D or 2D values; 2D arrays must be rectangular

Element type:  
T must be a ScalarType or EnumType.

Fixed-size:

```
arr<T>[N]    # N > 0
```

Fixed-size:

```
arr<T>[R,C]  # R > 0 AND C > 0
```

Dynamic-size:
```
arr<T>
```

Value‑level prefix (non‑empty only):

```
[N][1,2,3]
[R,C][[1,2],[3,4]]
```

Prefix form is not permitted for empty or fixed‑size containers in SuperCSV v1.0.

---

### Enum Values

#### Enum terminology

**An EnumType is defined by `enum<...>` in the header.**

Each **EnumItem** inside the brackets is either:

- **a name**  
  **name-only EnumItem** — e.g. `red`
- **a value–name pair**  
  **value–name EnumItem** — e.g. `E=red` or `1=red`

An **EnumValue** in a data row may be **either** the name **or** the value of an EnumItem.

---

EnumValues in data rows must match the name or declared value of one of the column's EnumItems.  
The null literal `_` represents missing data and is **not** a defined EnumValue.

- **Missing-data marker**: `_` is allowed as a field value  
- **Not a name**: `_` cannot appear as an EnumItem name  
- **Not part of the domain**: `_` is not included in the EnumType's allowed EnumValues  

#### Grammar

```
enum<item1,item2,...>
item ::= Identifier | Identifier=Identifier
```

#### Rules

- **EnumItem names** MUST be valid Identifiers (see [Identifiers](#identifiers))  
- **EnumItem values** MUST be valid Identifiers (see [Identifiers](#identifiers)); this includes pure digit strings such as `0` and `42`  
- **Name uniqueness**: names MUST be unique under case‑insensitive comparison within the EnumType  
- **No cross-item value/name collision**: a value may match its own item's name (case-insensitive) but must not match any other item's name (case-insensitive)  
- **Values may duplicate**: two or more items may share the same value  
- **Matching is case‑insensitive** — values from data rows are compared case‑insensitively against declared names and values  
- **Uniform style**: all EnumItems MUST use the same form — either all name-only or all value=name; mixing is forbidden  
- **EnumValues are Identifiers and must not be quoted**  
- **Null literal**: `_` is not a valid EnumValue  
- **Optional null**: `_` may appear instead of an EnumValue  

#### Lookup and Decoding Rule

When a data row field is resolved against an EnumType:

1. **Name match is checked first.** If the field matches any EnumItem name, that item is selected. Name matches are always unambiguous because names are unique.
2. **Value match is checked second.** If no name matches, the field is compared against EnumItem values in declaration order. The **first matching item** is selected.

This guarantees deterministic decoding even when values are duplicated.

#### Encoding Rule

- Encoders MUST use name-form when the EnumType contains duplicate values, to avoid ambiguity.
- Encoders MAY use value-form only when all values are unique within the EnumType.

#### Example

Header:

```
Color:enum<0=low,1=medium,2=high>
```

Valid EnumValues in rows:

- **Name form**: `low`, `medium`, `high`  
- **Value form**: `0`, `1`, `2`

Header with identifier values:

```
Color:enum<L=low,M=medium,H=high>
```

Valid EnumValues in rows:

- **Name form**: `low`, `medium`, `high`  
- **Value form**: `L`, `M`, `H`

---

### Null

Literal:

```
_
```

Represents missing or undefined value.  
Valid anywhere a value is expected.  
Never quoted.  
Not allowed as an EnumItem name, EnumItem value, or type name.

---

### Container Prefix Clarifications

Prefix form is not permitted for empty or fixed‑size containers in SuperCSV v1.0.

Forbidden patterns:
- [0][] → INVALID (zero-count prefix is never allowed)
- [0][1,2,3] → INVALID (prefix count does not match element count)
- [R,0][…] or [0,C][…] → INVALID (declares zero total elements via a zero dimension)

Valid empty container:
- [] → VALID (only the bracket form may represent an empty list/array)

For fixed-size containers, [] is invalid because the value must match the declared size.

---

## Row Definition

### Row Validation

Each row must:

- match the number of columns  
- match declared types  
- follow quoting rules  
- follow list/array syntax
- A row ends at the physical line break unless the break occurs immediately after a comma or inside a quoted string.
- If a row ends before all columns are present, the row is invalid.

Whitespace‑only rows are considered blank lines (see [Blank Lines](#blank-lines)) and are ignored. 
Invalid rows cause a parse error.

---

### Multi‑Line Rows

Data rows may span multiple physical lines when a newline appears **after a comma** and before the next value (i.e., between fields), and in this case the newline is treated as whitespace and does **not** terminate the row.

Newline **inside a quoted string**, is always part of that string and allowed and preserved.

A newline appearing **after a value** (outside a quoted string) **terminates the row**. If the row terminates before all columns are present, the row is invalid.

#### Rules

- A newline **after a comma** and before the next value continues the same logical row  
- A newline **inside a quoted string** is part of that value and continues the same logical row  
- A newline **after a value** (outside quotes) terminates the row  
- Continuation applies only during data‑row parsing (after the header row)  
- Blank lines within a multi‑line row do not terminate the row

#### Comments in data rows

**Within a data row** (trailing comma pending):

- Line‑level comments `( … )` may appear between continuation lines; they are skipped using the normal line‑level classification rule and relate to the **next field** (see [Comment Scope](#comment-scope))
- `# …` comments are **not** permitted within data rows, even if preceded by whitespace — `#` is a section‑break marker and would be misleading mid‑row
- Inline comments may appear before or after any field value; each field may have at most one comment (see [Inline Forms (Field‑Level)](#inline-forms-field-level))
- Inline comments are not permitted after the trailing comma
- A standalone comment line and an inline comment on the same field both count toward the one‑per‑field limit
- Blank lines between a comment and the next field do not change which field it relates to

**Between data rows** (no trailing comma pending):

- A standalone `( … )` line relates to the **next row** as a whole, not to the first field
- To place a comment on the **first field**, use inline form on the same line as the field value: `(comment) value, …`

Metadata syntax `(( … ))` is recognised in the same structural positions as comments where applicable, but in v1.0 only the version declaration (`((SuperCSV v1.0))`) has defined metadata meaning. Any other metadata block in a data row is recognised syntax, but has no defined meaning.

```supr
# Multi-line row examples (valid)
Name:string, Age:int

Bob, 35

# trailing comma continuation example
Dan,
43 (completes row)

# inline metadata syntax example (metadata has no v1.0 meaning)
Mark,56 (comment) (( meta ))
```

Equivalent to:

```supr
Name:string, Age:int
Bob,35
Dan,43
Mark,56
```

```supr
# Multi-line row examples (structurally valid with interleaved lines)
Name:string, Age:int

Bob, 35

# standalone comment ok, previous blank line ok and ignored
Dan,
(comment line between continuation lines relates to the next field)
(( metadata syntax between continuation lines; no v1.0 meaning ))
43

# standalone metadata syntax after terminating line - allowed structurally
(( standalone metadata syntax after terminating line ))
```

```supr
# Multi-line row examples (invalid)
Name:string, Age:int

John (invalid - newline after value and before continuation comma)
, 35 (not reached)

# recognised metadata syntax on Bob line; still no v1.0 meaning
Bob  (( meta tbc )),
(( recognised metadata syntax here, but invalid because no data follows the continuation ))
```

```supr
# Comment scope examples plus metadata-syntax placement examples
Name:string, Age:int, Score:float

# row-level comment (no trailing comma pending — relates to next row)
(( row-level metadata syntax uses the same placement rule, but has no v1.0 meaning ))
Alice,
(field comment relates to Age) 30,
(( field metadata syntax )) 95.5

# first-field comment must be inline
(field comment) Bob, 25, 88.0

# blank line mid-continuation does not change field association
Carol,
(relates to Age despite blank line below)

40,
99.9
```

```supr
# Comment and metadata scope examples (invalid structural cases)
Name:string, Age:int

# invalid — # comment mid-data-row
Alice,
# not permitted mid-row — only ( … ) comments are allowed within data rows
30

# invalid — two comments on same field
Bob,
(first comment)
(second comment) 25

# invalid — two metadata blocks on the same field position
Carol,
(( meta1 ))
(( meta2 )) 35
```

---

### Multi‑Line Container Values

Container values (`list<T>` and `arr<T>`) may span multiple physical lines for readability. A newline is permitted at specific positions within the container literal and is treated as whitespace — it does not terminate the row.

A newline appearing **outside a permitted position** within a container terminates the row. If the row terminates before the container is closed, the value is invalid.

#### Rules

- A newline **after a comma** inside a container continues the same container value
- A newline **after the last field** inside a container (before the closing `]`) continues the same container value
- A newline **after the opening `[`** continues the same container value  
- A newline **after the closing `]` of an inner array** (i.e. when still inside the outer container) continues the same container value  
- A newline **after the final closing `]`** terminates the row normally  
- A newline **after any other position** (e.g. mid‑element, after a value) terminates the row and the container value is invalid  
- Blank lines are **not** permitted within a container value  
- Comments and metadata are **not** permitted within a container value  

```supr
# Multi-line container examples (valid)
Name:string, Tags:list<string>, Matrix:arr<int>

Alice,
[
  work,
  urgent
],
[[1,2],[3,4]]

Bob, [a,b,c], [
  [1,2],
  [3,4]
]
```

```supr
# Multi-line container examples (invalid)
Name:string, Tags:list<string>

Alice, [work
,urgent]  (invalid - newline after value, not after comma)

Bob, [
# comment not allowed inside container
  work, urgent
]
```

---

## Error Model

Validators MUST emit each error as a three‑field SuperCSV row:

```
Line:int, ErrorSection:string, ErrorMsg:string
```

### Line
Physical CSV line number (1‑based), including blank and comment lines.

### ErrorSection
Identifies the header, row or column and, if applicable, the position inside its value.

Valid forms (v1.0):
- Price — scalar column
- Tags(4) — 4th element of a list or 1D array
- Matrix(2,3) — element at row 2, column 3 of a 2D array
- headerErr — header-level error (literal keyword)
- rowErr — row-level error (literal keyword)

Rules:
- Indices are 1‑based

### ErrorMsg
Human‑readable message derived from the error code's template.

- MUST be a valid SuperCSV string literal.
- MUST follow string quoting rules.
- Message templates SHOULD NOT contain double quotes.
- Implementations SHOULD use single quotes when quoting values.

### Error Message Examples

```
8, Price, "invalid int value: 'abc'"
14, Tags(4), "invalid enum label: 'blueish'"
19, Scores(1), "expected 3 elements, got 2"
12, Matrix(2,3), "invalid int value: '/'"
3, Price, "int values must not be quoted"
5, Tags, "container values must not be quoted"
8, Matrix, "expected shape [3,3], got [3,2]"
9, rowErr, "expected 5 columns, got 6"
2, headerErr, "invalid identifier: ' Name'"
```

---

## Example

A complete SuperCSV file with typed headers and two data rows:

```
((SuperCSV v1.0))
Name:string, Score:int, Flags:list<bool>, Matrix:arr<int>, Level:enum<0=low,1=medium,2=high>

Ras, 42, [true,false,true], [[1,2],[3,4]], medium
Alex, 29, [1,0,1], [[5,6],[7,8]], 2
```

Additional examples covering all types and features are in the `examples/` directory.

---

## Conformance

### Required Behaviors

*To be completed in a future editorial pass.*

### Forbidden Behaviors

*To be completed in a future editorial pass.*

### Implementation Notes

*To be completed in a future editorial pass.*

---

## External References

### Grammar (Normative)

The formal grammar for SuperCSV v1.0 is defined in:

```
grammar.ebnf
```

This file provides a machine-readable EBNF definition of the SuperCSV syntax and serves as a cross-check for parser implementations.

---

### Type Table (Normative)

All built‑in types are defined in:

```
SuperCsvTypeTable v1.0
```

This file is the canonical, versioned registry of types.

---
