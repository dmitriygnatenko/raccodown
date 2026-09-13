// Package web holds only the embedded frontend — go:embed patterns can't reach outside the
// directory of the file that declares them, so this lives next to index.html/css/js, separate from
// the actual binary entrypoint in cmd/raccodown.
package web

import "embed"

// WebFiles is the frontend, embedded into the binary.
//
//go:embed index.html css js
var WebFiles embed.FS
