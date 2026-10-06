//go:build !linux && !windows

package repository

// canWatch tells whether watching path for changes is meaningful. On macOS and
// the BSDs, fsnotify watches a directory with a file descriptor per entry, so a
// repository of 10k bugs would take 10k of them: polling is used instead.
func canWatch(string) bool {
	return false
}
