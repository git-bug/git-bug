//go:build !webui

package webui

import "embed"

// Available reports whether the web UI was compiled into this binary. See the
// documentation in //webui:assets.go.
const Available = false

// assets is empty here: nothing was embedded. Callers are expected to check
// Available and fail with a useful message rather than serve an empty site.
var assets embed.FS
