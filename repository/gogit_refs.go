package repository

// GoGitRepo writes and lists the refs of git-bug's namespaces itself, rather
// than through go-git, which gets both wrong in ways git-bug depends on:
//
//   - a rejected CheckAndSetReference leaves an empty loose ref file behind,
//     which then breaks listing refs for the whole repository
//     (https://github.com/go-git/go-git/issues/2399),
//   - it can't create a ref only if it doesn't exist yet, atomically,
//   - it locks a ref with flock on the ref file, which the git binary ignores,
//   - listing refs reads every file under refs/ as a ref, including the .lock
//     files of an update in progress.
//
// This follows git's own protocol for loose refs instead: an update holds
// <ref>.lock, created exclusively and already holding the new value, while it
// compares, then renames it over the ref. Listing reads the loose refs and packed-refs of one namespace,
// skipping lock files and broken refs like git does.
//
// Removing a local ref holds the same lock, so that it can't interleave with an
// update, but the removal itself, including rewriting packed-refs, is still
// done by go-git.
//
// Everything else stays with go-git: resolving a single ref (which already falls
// back to packed-refs on a broken loose file), removing tracking refs, and the
// refs written by fetch and push.

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

const refLockSuffix = ".lock"

// refLockTimeout is how long an update waits for the lock of a ref held by
// another writer. Updates hold it for a few syscalls, so running out means the
// lock was most likely left behind by a crash.
var refLockTimeout = time.Second

// UpdateRef points a local ref to commit, only if it currently points to old.
// An empty old means that the ref must not exist yet.
// Returns ErrRefChanged otherwise, and the ref is left unchanged.
func (repo *GoGitRepo) UpdateRef(namespace string, key string, old Hash, commit Hash) error {
	name, err := refName(namespace, key)
	if err != nil {
		return err
	}
	path := filepath.Join(repo.path, filepath.FromSlash(name))

	// the lock already holds the new value: it only has to be renamed if the
	// comparison succeeds
	lockPath, err := repo.lockRef(path, []byte(commit.String()+"\n"))
	if err != nil {
		return fmt.Errorf("can't update %s: %w", name, err)
	}
	// until renamed over the ref, the lock is ours to remove
	lockNeedsCleanup := true
	defer func() {
		if lockNeedsCleanup {
			_ = os.Remove(lockPath)
		}
	}()

	// under the lock, nobody else following the protocol can move the ref
	current, err := repo.lookupRef(name)
	if errors.Is(err, ErrNotFound) {
		current = ""
	} else if err != nil {
		return err
	}
	if current != old {
		return fmt.Errorf("%w: %s", ErrRefChanged, name)
	}

	if err := os.Rename(lockPath, path); err != nil {
		return err
	}
	lockNeedsCleanup = false
	return nil
}

// RemoveRef deletes a local ref.
// RemoveRef is idempotent.
func (repo *GoGitRepo) RemoveRef(namespace string, key string) error {
	name, err := refName(namespace, key)
	if err != nil {
		return err
	}
	path := filepath.Join(repo.path, filepath.FromSlash(name))

	// holding the lock orders the removal with the updates: one comparing
	// before it can't then recreate the ref. The lock is never renamed, so its
	// content doesn't matter, but it must not be empty (see lockRef).
	lockPath, err := repo.lockRef(path, []byte(plumbing.ZeroHash.String()+"\n"))
	if err != nil {
		return fmt.Errorf("can't remove %s: %w", name, err)
	}
	defer os.Remove(lockPath)

	return repo.r.Storer.RemoveReference(plumbing.ReferenceName(name))
}

// lockRef creates the lock file of the ref at path, holding content, and
// returns its path. It waits for a concurrent writer to release the lock.
//
// The lock file never exists empty: go-git lists refs by reading every file
// under refs/, lock files included, and an empty one makes the whole listing
// fail. So content is written to a temporary file outside of refs/ first, then
// hard-linked as the lock, which fails if the lock exists, like an exclusive
// create. On a filesystem without hard links, the lock is created exclusively
// then written, and the empty window comes back.
func (repo *GoGitRepo) lockRef(path string, content []byte) (string, error) {
	lockPath := path + refLockSuffix

	tmpPath, err := writeTempFile(repo.localStorage.Root(), content)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpPath)

	link := true
	// the directory of the ref almost always exists, so it's only created when
	// missing: on the first ref of a namespace, or when git pruned it after
	// removing its last ref, possibly again while we recreate it, like git's
	// raceproof_create_file retries
	dirRetries := 3
	deadline := time.Now().Add(refLockTimeout)
	wait := time.Millisecond
	for {
		if link {
			err = os.Link(tmpPath, lockPath)
		} else {
			err = createFileExclusive(lockPath, content)
		}
		if errors.Is(err, fs.ErrNotExist) && dirRetries > 0 {
			dirRetries--
			if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
				return "", err
			}
			continue
		}
		if link && err != nil && !errors.Is(err, fs.ErrExist) {
			link = false
			continue
		}
		if err == nil {
			return lockPath, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("%s exists: another process is updating the ref, or crashed doing so; if no git or git-bug process is running, remove it", lockPath)
		}
		time.Sleep(wait)
		wait = min(2*wait, 50*time.Millisecond)
	}
}

// writeTempFile writes content to a new temporary file in dir, and returns its
// path.
//
// The file becomes the ref, so it is created with the permissions of a ref
// created directly (0666 less the umask), not the 0600 of os.CreateTemp.
func writeTempFile(dir string, content []byte) (string, error) {
	// the directory is only missing before the first write to the local storage
	madeDir := false
	// a random 64-bit name practically never collides: running out of
	// attempts means the filesystem rejects every name, not bad luck
	const attempts = 100
	var err error
	for range attempts {
		path := filepath.Join(dir, fmt.Sprintf("ref-%016x.tmp", rand.Uint64()))
		err = createFileExclusive(path, content)
		if errors.Is(err, fs.ErrNotExist) && !madeDir {
			madeDir = true
			if err := os.MkdirAll(dir, 0777); err != nil {
				return "", err
			}
			continue
		}
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("can't create a temporary file in %s after %d attempts: %w", dir, attempts, err)
}

// createFileExclusive creates the file at path holding content, failing with
// fs.ErrExist if it already exists.
func createFileExclusive(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	_, err = f.Write(content)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
	}
	return err
}

// refName returns the name of the local ref of namespace and key, rejecting the
// names git would reject: those also can't escape the repository once turned
// into a path, nor be taken for a lock file. The key must not leave its
// namespace either.
func refName(namespace string, key string) (string, error) {
	name := refPrefix(namespace) + key
	if strings.Contains(key, "/") {
		return "", fmt.Errorf("invalid ref key %q", key)
	}
	if err := plumbing.ReferenceName(name).Validate(); err != nil {
		return "", fmt.Errorf("invalid ref %q: %w", name, err)
	}
	return name, nil
}

// ListRefs returns every local ref of a namespace, by key.
func (repo *GoGitRepo) ListRefs(namespace string) (map[string]Hash, error) {
	return repo.listRefs(refPrefix(namespace))
}

// ListTrackingRefs returns every tracking ref of a namespace for a remote,
// by key.
func (repo *GoGitRepo) ListTrackingRefs(remote string, namespace string) (map[string]Hash, error) {
	return repo.listRefs(trackingRefPrefix(remote, namespace))
}

// listRefs returns the refs under prefix, by the rest of their name. Loose refs
// take precedence over packed ones.
//
// Loose refs are read before packed-refs, as git does: git pack-refs writes the
// new packed-refs, then deletes the loose refs it packed. In this order, a ref
// is either still loose when read, or already in the packed-refs read after.
// The other order can miss it, or return its outdated packed value.
func (repo *GoGitRepo) listRefs(prefix string) (map[string]Hash, error) {
	// the prefix comes from a namespace, and a remote name: check it like a
	// ref name before walking it as a path
	if err := plumbing.ReferenceName(strings.TrimSuffix(prefix, "/")).Validate(); err != nil {
		return nil, fmt.Errorf("invalid ref prefix %q: %w", prefix, err)
	}

	loose := make(map[string]Hash)

	root := filepath.Join(repo.path, filepath.FromSlash(prefix))
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// no loose ref at all, or removed while walking
				return nil
			}
			return err
		}
		if d.IsDir() || strings.HasSuffix(d.Name(), refLockSuffix) {
			return nil
		}

		content, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			// removed while walking
			return nil
		}
		if err != nil {
			return err
		}
		value := strings.TrimSpace(string(content))
		if !plumbing.IsHash(value) {
			// an empty file being written by a writer not using a lock file or
			// left behind by go-git (#2399), a symbolic ref, or garbage: none
			// is an entity, and like git, one broken ref doesn't fail the
			// listing of the others
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		loose[filepath.ToSlash(rel)] = Hash(value)
		return nil
	})
	if err != nil {
		return nil, err
	}

	refs, err := repo.listPackedRefs(prefix)
	if err != nil {
		return nil, err
	}
	maps.Copy(refs, loose)

	return refs, nil
}

// listPackedRefs returns the refs of packed-refs under prefix, by the rest of
// their name.
func (repo *GoGitRepo) listPackedRefs(prefix string) (map[string]Hash, error) {
	refs := make(map[string]Hash)

	f, err := os.Open(filepath.Join(repo.path, "packed-refs"))
	if errors.Is(err, fs.ErrNotExist) {
		return refs, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// "<hash> <name>" lines, plus a "# pack-refs with:" header and "^<hash>"
	// lines peeling the annotated tag above them
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line == "" || line[0] == '#' || line[0] == '^' {
			continue
		}
		hash, name, ok := strings.Cut(line, " ")
		if !ok || !plumbing.IsHash(hash) {
			return nil, fmt.Errorf("malformed packed-refs line: %q", line)
		}
		if key, ok := strings.CutPrefix(name, prefix); ok {
			refs[key] = Hash(hash)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return refs, nil
}
