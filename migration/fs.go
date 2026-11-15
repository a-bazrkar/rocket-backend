package migration

import "embed"

// FS holds the versioned up and down SQL files.
//
//go:embed *.sql
var FS embed.FS
