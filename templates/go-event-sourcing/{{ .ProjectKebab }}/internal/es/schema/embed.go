// Package schema carries the DDL the store applies on open, embedded so a
// binary needs nothing alongside it.
//
// Files are applied in filename order, so name new ones with an increasing
// prefix. Each is applied once and recorded with a hash of its contents: edit
// an already-applied file and the store refuses to open rather than leaving
// two databases with the same name and different shapes.
package schema

import "embed"

//go:embed *.sql
var FS embed.FS
