package repository

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

// canWatch tells whether watching path for changes is meaningful: not on a
// network drive, where the changes made by another host are never reported.
func canWatch(path string) bool {
	volume := filepath.VolumeName(path)
	if len(volume) != 2 {
		// a UNC path, \\server\share
		return false
	}
	root, err := windows.UTF16PtrFromString(volume + `\`)
	if err != nil {
		return false
	}
	return windows.GetDriveType(root) != windows.DRIVE_REMOTE
}
