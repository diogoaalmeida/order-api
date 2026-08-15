// Package migrations embeds the SQL migration files into the binary so the
// app can apply them on startup without needing the files mounted separately.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
