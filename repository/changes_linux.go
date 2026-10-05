package repository

import "golang.org/x/sys/unix"

// canWatch tells whether watching path for changes is meaningful: not on a
// network filesystem, where the changes made by another host are never
// reported. FUSE and 9p mounts (sshfs, WSL's drives) count as such.
func canWatch(path string) bool {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return false
	}
	switch uint32(stat.Type) {
	case unix.NFS_SUPER_MAGIC, unix.SMB_SUPER_MAGIC, unix.SMB2_SUPER_MAGIC,
		unix.CIFS_SUPER_MAGIC, unix.FUSE_SUPER_MAGIC, unix.V9FS_MAGIC,
		unix.AFS_SUPER_MAGIC, unix.CODA_SUPER_MAGIC, unix.NCP_SUPER_MAGIC:
		return false
	}
	return true
}
