//go:build webui

package webui

import "embed"

// Available reports whether the web UI was compiled into this binary.
//
// The assets are produced by `pnpm run build` into webui/dist, which is not
// tracked in git. Guarding the embed behind a build tag means the package
// still compiles without them, so `go build`, `go test` and `go install` work
// on a fresh clone with no Node.js toolchain.
const Available = true

//go:embed all:dist
var assets embed.FS
