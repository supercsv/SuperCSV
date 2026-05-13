// SuperCSV
// Copyright (c) 2025-2026 Ras
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package v1_0

import (
	"github.com/supercsv/supercsv/internal/v1_0/headerdef"
)

// parseHeader clones the existing headerdef.ParseHeader behavior for v1.0.
// Keeping this indirection lets the streaming validator swap in
// spec-driven tweaks later without touching the shared schema package.
func parseHeader(row *headerdef.Row) (*headerdef.HeaderDef, error) {
	return headerdef.ParseHeader(row)
}
