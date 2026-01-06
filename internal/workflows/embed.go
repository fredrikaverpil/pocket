// Package workflows embeds workflow templates.
package workflows

import "embed"

//go:embed github/golang/*.tmpl github/generic/*.tmpl github/markdown/*.tmpl
var FS embed.FS
