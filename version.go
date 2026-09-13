package main

import (
	"runtime/debug"
	"strings"

	"github.com/git-bug/git-bug/commands"
	"golang.org/x/mod/semver"
)

// version is injected at build time with `-X main.version=<version>`. See
// //:.goreleaser.yaml for releases, and //:Makefile for local builds.
var version = "undefined"

// getVersion returns the version information embedded in the binary, in the
// format documented by `git-bug version`:
//
//	<version> [commit[/dirty]] <compiler version> <platform> <arch>
//
// Only <version> has to be injected at build time. Everything else comes from
// the build metadata that go stamps into the binary on its own, which also
// covers binaries built with `go install`.
//
// TODO: support notifying the user if their version is out of date
func getVersion() string {
	// this supports overriding the default version if the deprecated var used
	// for setting the exact version for releases is supplied. we are doing this
	// in order to give downstream package maintainers a longer window to
	// migrate.
	//
	// TODO: 0.12.0: remove support for old build tags
	if version == "undefined" && commands.GitExactTag != "" {
		version = commands.GitExactTag
	}

	// add the v prefix if it's missing, then fall back to "undefined" if what
	// we are left with isn't a real version. this guards against whatever a
	// downstream packager might inject.
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		version = "undefined"
	}

	var v strings.Builder
	v.WriteString(version)

	info, ok := debug.ReadBuildInfo()
	if !ok {
		v.WriteString(" (no build info)\n")
		return v.String()
	}

	var arch, commit, platform string
	var modified bool

	for _, kv := range info.Settings {
		switch kv.Key {
		case "GOOS":
			platform = kv.Value
		case "GOARCH":
			arch = kv.Value
		case "vcs.modified":
			modified = kv.Value == "true"
		case "vcs.revision":
			commit = kv.Value
		}
	}

	if commit != "" {
		if len(commit) > 12 {
			commit = commit[:12]
		}
		v.WriteString(" " + commit)
		if modified {
			v.WriteString("/dirty")
		}
	}

	v.WriteString(" " + info.GoVersion)

	if platform != "" {
		v.WriteString(" " + platform)
	}

	if arch != "" {
		v.WriteString(" " + arch)
	}

	v.WriteString("\n")

	return v.String()
}
