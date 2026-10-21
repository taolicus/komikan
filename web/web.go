// Package web holds the static assets of the Komikan application.
//
// The assets live in their own package so that the embed directive has a
// single home and the HTTP layer can serve them without knowing where they
// come from.
package web

import "embed"

// Files contains the static assets served by the application.
//
//go:embed index.html favicon.svg
var Files embed.FS
