// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import "github.com/supercsv/supercsv/internal/v1_0/headerdef"

// Value type aliases - canonical definitions live in headerdef.
// These aliases preserve backward compatibility for all existing code
// that references decode.Date, decode.Time, etc.
type (
	Date       = headerdef.Date
	Time       = headerdef.Time
	Timestamp  = headerdef.Timestamp
	Datetime   = headerdef.Datetime
	DatetimeTZ = headerdef.DatetimeTZ
	Duration   = headerdef.Duration
	Decimal    = headerdef.Decimal
)
