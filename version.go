package main

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/git-bug/git-bug/commands"
	"golang.org/x/mod/semver"
)

var (
	version = "undefined"
)

// getVersion returns a string representing the version information defined when
// the binary was built, or a sane default indicating a local build. a string is
// always returned. an error may be returned along with the string in the event
// that we detect a local build but are unable to get build metadata.
//
// TODO: support validation of the version (that it's a real version)
// TODO: support notifying the user if their version is out of date
func getVersion() (string, error) {
	var arch string
	var commit string
	var modified bool
	var platform string

	var v strings.Builder

	// this supports overriding the default version if the deprecated var used
	// for setting the exact version for releases is supplied. we are doing this
	// in order to give downstream package maintainers a longer window to
	// migrate.
	//
	// TODO: 0.12.0: remove support for old build tags
	if version == "undefined" && commands.GitExactTag != "" {
		version = commands.GitExactTag
	}

	// automatically add the v prefix if it's missing
	if version != "undefined" && !strings.HasPrefix(version, "v") {
		version = fmt.Sprintf("v%s", version)
	}

	// reset the version string to undefined if it is invalid
	if ok := semver.IsValid(version); !ok {
		version = "undefined"
	}

	v.WriteString(version)

	info, ok := debug.ReadBuildInfo()
	if !ok {
		v.WriteString(fmt.Sprintf(" (no build info)\n"))
		return v.String(), errors.New("unable to read build metadata")
	}

	for _, kv := range info.Settings {
		switch kv.Key {
		case "GOOS":
			platform = kv.Value
		case "GOARCH":
			arch = kv.Value
		case "vcs.modified":
			if kv.Value == "true" {
				modified = true
			}
		case "vcs.revision":
			commit = kv.Value
		}
	}

	if commit != "" {
		v.WriteString(fmt.Sprintf(" %.12s", commit))
	}

	if modified {
		v.WriteString("/dirty")
	}

	v.WriteString(fmt.Sprintf(" %s", info.GoVersion))

	if platform != "" {
		v.WriteString(fmt.Sprintf(" %s", platform))
	}

	if arch != "" {
		v.WriteString(fmt.Sprintf(" %s", arch))
	}

	return fmt.Sprint(v.String(), "\n"), nil
}
