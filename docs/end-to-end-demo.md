# End-to-End Demo

This walkthrough shows one complete consumer flow:

1. Define a typed SuperCSV header in Go.
2. Generate rows in memory.
3. Encode them to a `.supr` file.
4. Validate the file with the CLI.
5. Decode it again with batch and streaming APIs.
6. Compare `DecodeFull`, `DecodeShallow`, and `DecodeRaw`.

If you want the full API reference after this walkthrough, continue with the [Encode, Decode & Validate Guide](supercsv-encode-decode-validate-guide.md).

## What We Are Building

The demo file uses one headerDef with scalar, enum, list, and array fields:

```text
id:int,name:string,startDt:date,status:enum<new|active|paused>,tags:list<string>,scores:arr<int>[3]
```

## Setup

Create a new Go module and add the SDK:

```bash
mkdir supercsv-demo
cd supercsv-demo
go mod init example.com/supercsv-demo
go get github.com/supercsv/supercsv/supr
```

Install the CLI validator as well:

```bash
go install github.com/supercsv/supercsv/cmd/supercsv-validate@latest
```

## Encode a `.supr` File

Create `main.go`:

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/supercsv/supercsv/supr"
)

func main() {
	statusType := supr.Enum([]supr.EnumValue{
		{Name: "new"},
		{Name: "active"},
		{Name: "paused"},
	})

	header := supr.NewHeader([]supr.Column{
		{Name: "id", Type: supr.Int},
		{Name: "name", Type: supr.String},
		{Name: "startDt", Type: supr.Date},
		{Name: "status", Type: statusType},
		{Name: "tags", Type: supr.List(supr.String)},
		{Name: "scores", Type: supr.ArrFixed1D(supr.Int, 3)},
	})

	rows := [][]any{
		{int64(1), "Ada", "2025-01-10", "active", []any{"ops", "blue"}, []any{int64(9), int64(8), int64(9)}},
		{int64(2), "Linus", "2025-02-01", "new", []any{"infra"}, []any{int64(7), int64(8), int64(8)}},
		{int64(3), "Mina", "2025-02-14", "paused", nil, []any{int64(6), int64(6), int64(7)}},
	}

	out, err := os.Create("people.supr")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	file := &supr.File{
		Header: header,
		Rows:   rows,
	}

	if err := supr.EncodeFile(out, file); err != nil {
		log.Fatal(err)
	}

	fmt.Println("wrote people.supr")
}
```

Run it:

```bash
go run .
```

You should now have a valid `people.supr` file in the current directory.

## Validate the File with the CLI

Validate the file from the command line:

```bash
supercsv-validate people.supr
```

Use quiet mode when you want to suppress per-error output:

```bash
supercsv-validate --quiet people.supr
```

`--quiet` still prints the final summary, but it suppresses the per-error rows and is useful in scripts or CI.

## Decode the Whole File at Once

For small or moderate files, batch decode is the simplest path:

```go
package main

import (
	"fmt"
	"log"

	"github.com/supercsv/supercsv/supr"
)

func main() {
	file, err := supr.DecodeFile("people.supr")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("columns:", len(file.Header.Columns))
	fmt.Println("rows:", len(file.Rows))
	fmt.Printf("first row: %#v\n", file.Rows[0])
}
```

This returns a `*supr.File` with a parsed `Header` and all decoded rows.

If you want the demo to show the real decoded Go types clearly, print every field explicitly:

```go
for _, row := range file.Rows {
	id := row[0].(int64)
	name := row[1].(string)
	startDt := row[2].(supr.DateValue)
	status := row[3].(supr.EnumField)
	tags := row[4]
	scores := row[5].([]any)

	fmt.Printf(
		"id=%d name=%s startDt=%s status=%s tags=%v scores=%v\n",
		id,
		name,
		startDt.String(),
		status.Name(),
		tags,
		scores,
	)
}
```

## Stream Rows Instead of Loading Everything

For larger files, use the streaming decoder:

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/supercsv/supercsv/supr"
)

func main() {
	f, err := os.Open("people.supr")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	d := supr.NewDecoder(f)

	header := d.Header()
	// Use String() for the compact canonical header, or iterate Columns for structured access.
	fmt.Println("header:", header.String())
	fmt.Println("schema:")
	for _, col := range header.Columns {
		fmt.Printf("  %s: %s\n", col.Name, col.Type.String())
	}

	for d.Next() {
		row := d.Row()
		id := row[0].(int64)
		name := row[1].(string)
		startDt := row[2].(supr.DateValue)
		status := row[3].(supr.EnumField)
		tags := row[4]
		scores := row[5].([]any)

		fmt.Printf(
			"row=%d id=%d name=%s startDt=%s status=%s tags=%v scores=%v\n",
			d.RowNum(),
			id,
			name,
			startDt.String(),
			status.Name(),
			tags,
			scores,
		)
	}

	if err := d.Err(); err != nil {
		log.Fatal(err)
	}
}
```

Use streaming decode when you want constant-memory iteration over a file.

## Decode Modes

SuperCSV exposes three decode modes. They all validate the file structure, but they return field values differently.

### `DecodeFull`

This is the default. Scalars decode to typed Go values, and containers decode to nested `[]any` values.

```go
d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeFull))
```

Typical results for the demo file:

- `id` becomes `int64`
- `name` becomes `string`
- `startDt` becomes `supr.DateValue`
- `status` becomes `supr.EnumField`
- `tags` becomes `[]any`
- `scores` becomes `[]any`

Use this when you want the richest decoded structure.

### `DecodeShallow`

Scalars still decode, but containers stay as their original text form.

```go
d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeShallow))
```

Typical results for the demo file:

- `id` becomes `int64`
- `name` becomes `string`
- `tags` stays as a `string` like `[ops,blue]`
- `scores` stays as a `string` like `[9,8,9]`

Use this when you want typed scalars but want to defer container parsing.

### `DecodeRaw`

Every field is returned as raw bytes.

```go
d := supr.NewDecoder(f, supr.WithDecodeMode(supr.DecodeRaw))
```

Typical results for the demo file:

- `id` becomes `[]byte("1")`
- `name` becomes `[]byte("Ada")`
- `tags` becomes `[]byte("[ops,blue]")`
- `scores` becomes `[]byte("[9,8,9]")`

Use this when you want maximum control over downstream parsing.

## When to Use Which API

- Use `supr.EncodeFile` when you already have all rows in memory.
- Use `supr.NewFileEncoder` when you want to stream rows out incrementally.
- Use `supr.DecodeFile` when you want the whole file loaded into a `*supr.File`.
- Use `supr.NewDecoder` when you want streaming reads.
- Use `supercsv-validate` when you want a CLI check in scripts, tooling, or CI.

## Extended Demo: Broader Type Coverage

Use this extended version when you want to cover more of the SuperCSV.

### Recommended Coverage Set

For a broader end-to-end demo, use one wider header that covers:

- scalar types
- enums
- list
- fixed-size 1D array
- a dynamic array
- a fixed 2D array

An example extended header:

```text
id:int,name:string,active:bool,ratio:float,amount:decimal,startDt:date,startTime:time,localDt:datetime,createdAt:timestamp,expiresAt:datetimetz,span:duration,zone:timezone,userId:uuid,payloadHex:bytes<hex>,payloadB64:bytes<b64>,status:enum<new|active|paused>,priority:enum<100=low,200=medium,300=high>,tags:list<string>,scores:arr<int>[3],steps:arr<int>,matrix:arr<float>[2,2]
```


### Example Extended Rows

```go
rows := [][]any{
	{
		int64(1),
		"Ada",
		true,
		0.875,
		"1234.567",
		"2025-01-10",
		"09:30:00",
		"2025-01-10T09:30:00",
		"2025-01-10T09:30:00Z",
		"2025-06-01T17:45:00+12:00",
		"P2DT3H15M",
		"Pacific/Auckland",
		"550e8400-e29b-41d4-a716-446655440000",
		"4a6f686e",
		"Sm9obg==",
		"active",
		"200",
		[]any{"ops", "blue"},
		[]any{int64(9), int64(8), int64(9)},
		[]any{int64(100), int64(250), int64(400)},
		[][]any{{1.1, 1.2}, {2.1, 2.2}},
	},
	{
		int64(2),
		"Linus",
		false,
		0.625,
		"98.125",
		"2025-02-01",
		"14:05:30",
		"2025-02-01T14:05:30",
		"2025-02-01T14:05:30+01:00",
		"2025-09-30T08:00:00-05:00",
		"PT6H45M",
		"America/New_York",
		"123e4567-e89b-12d3-a456-426614174000",
		"deadbeef",
		"Z29waGVy",
		"new",
		"100",
		[]any{"infra", "green"},
		[]any{int64(7), int64(8), int64(8)},
		[]any{int64(50), int64(125)},
		[][]any{{3.1, 3.2}, {4.1, 4.2}},
	},
}
```

### Decode the Extended Example

Example decode in `DecodeFull` mode:

```go
for _, row := range file.Rows {
	id := row[0].(int64)
	name := row[1].(string)
	amount := row[4].(supr.DecimalValue)
	startDt := row[5].(supr.DateValue)
	createdAt := row[8].(supr.TimestampValue)
	expiresAt := row[9].(supr.DatetimeTZValue)
	zone := row[11].(string)
	status := row[15].(supr.EnumField)
	priority := row[16].(supr.EnumField)
	tags := row[17].([]any)
	scores := row[18].([]any)
	steps := row[19].([]any)
	matrix := row[20].([][]any)

	fmt.Printf(
		"id=%d name=%s amount=%s startDt=%s createdAt=%s expiresAt=%s zone=%s status=%s priority=%s(%s) tags=%v scores=%v steps=%v matrix=%v\n",
		id,
		name,
		amount.String(),
		startDt.String(),
		createdAt.String(),
		expiresAt.String(),
		zone,
		status.Name(),
		priority.Name(),
		priority.Value(),
		tags,
		scores,
		steps,
		matrix,
	)
}
```

### Notes

- This extended example keeps every row fully populated so the main decode example stays readable.
- Null handling and empty-container behavior are intentionally left out here and are covered separately in the SDK guide and spec.
- For fixed containers such as `arr<int>[3]` and `arr<float>[2,2]`, the element count and shape must match the declared dimensions.

## Next Steps

- Continue with the [Encode, Decode & Validate Guide](supercsv-encode-decode-validate-guide.md) for options, row-level validation, and more decode details.
- Read the [Quick Reference](../internal/v1_0/spec/supercsv-quick-reference-v1.0.md) when you want the format rules in compact form.
- Read the full [Specification](../internal/v1_0/spec/supercsv-spec-v1.0.md) when you need the canonical rules.