package repository

// TODO: remove this file and its tests once go-git writes refs under a lock
// file, skips lock files and broken loose refs when listing, and git-bug uses
// that version (go-git branch ref-lockfile). UpdateRef, RemoveRef and
// RemoveTrackingRef then go through CheckAndSetReference and RemoveReference,
// ListRefs and ListTrackingRefs through IterReferences.
//
// GoGitRepo writes and lists the refs of git-bug's namespaces itself, rather
// than through go-git, which gets both wrong in ways git-bug depends on:
//
//   - a rejected CheckAndSetReference leaves an empty loose ref file behind,
//     which then breaks listing refs for the whole repository
//     (https://github.com/go-git/go-git/issues/2399),
//   - it can't create a ref only if it doesn't exist yet, atomically,
//   - it locks a ref with flock on the ref file, which the git binary ignores,
//     and rewrites packed-refs under an flock too, ignoring packed-refs.lock:
//     a removal can then undo a git pack-refs running meanwhile, losing the
//     refs it packed,
//   - listing refs reads every file under refs/ as a ref, including the .lock
//     files of an update in progress.
//
// This follows git's own protocol for loose refs instead: an update holds
// <ref>.lock, created exclusively and already holding the new value, while it
// compares, then renames it over the ref. A removal holds <ref>.lock and
// packed-refs.lock while it removes the ref from packed-refs, then the loose
// ref.
// Listing reads the loose refs and packed-refs of one namespace, skipping lock
// files and broken refs like git does.
//
// Everything else stays with go-git: resolving a single ref (which already falls
// back to packed-refs on a broken loose file), and the refs written by fetch
// and push.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/sync/errgroup"
)

const refLockSuffix = ".lock"

// refLockTimeout is how long an update waits for the lock of a ref held by
// another writer. Updates hold it for a few syscalls, so running out means the
// lock was most likely left behind by a crash.
var refLockTimeout = time.Second

// UpdateRef points a local ref to commit, only if it currently points to old.
// An empty old means that the ref must not exist yet.
// Returns ErrRefChanged otherwise, and the ref is left unchanged.
func (repo *GoGitRepo) UpdateRef(ctx context.Context, namespace string, key string, old Hash, commit Hash) error {
	name, err := refName(refPrefix(namespace), key)
	if err != nil {
		return err
	}
	path := filepath.Join(repo.path, filepath.FromSlash(name))

	// the lock already holds the new value: it only has to be renamed if the
	// comparison succeeds
	lockPath, err := repo.lockRef(ctx, path, []byte(commit.String()+"\n"))
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
func (repo *GoGitRepo) RemoveRef(ctx context.Context, namespace string, key string) error {
	name, err := refName(refPrefix(namespace), key)
	if err != nil {
		return err
	}
	return repo.removeRef(ctx, name)
}

// RemoveTrackingRef deletes a tracking ref.
// RemoveTrackingRef is idempotent.
func (repo *GoGitRepo) RemoveTrackingRef(ctx context.Context, remote string, namespace string, key string) error {
	name, err := refName(trackingRefPrefix(remote, namespace), key)
	if err != nil {
		return err
	}
	return repo.removeRef(ctx, name)
}

// removeRef deletes the ref name, as git does: from packed-refs first, then
// the loose ref, which would otherwise expose the packed value meanwhile.
func (repo *GoGitRepo) removeRef(ctx context.Context, name string) error {
	path := filepath.Join(repo.path, filepath.FromSlash(name))

	// holding the lock orders the removal with the updates: one comparing
	// before it can't then recreate the ref. The lock is never renamed, so its
	// content doesn't matter, but it must not be empty (see lockRef).
	lockPath, err := repo.lockRef(ctx, path, []byte(plumbing.ZeroHash.String()+"\n"))
	if err != nil {
		return fmt.Errorf("can't remove %s: %w", name, err)
	}
	defer os.Remove(lockPath)

	// held until the loose ref is gone, even when packed-refs doesn't list the
	// ref: a git pack-refs could otherwise pack the loose ref right before it
	// is removed, which would bring the ref back from packed-refs
	packedPath := filepath.Join(repo.path, "packed-refs")
	packedLockPath, err := repo.lockRef(ctx, packedPath, nil)
	if err != nil {
		return fmt.Errorf("can't remove %s: %w", name, err)
	}
	defer os.Remove(packedLockPath)

	packed, err := os.ReadFile(packedPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	// keep every line but the ref's, and the "^<hash>" lines peeling it
	var kept []string
	removed, peeling := false, false
	for _, line := range strings.SplitAfter(string(packed), "\n") {
		if peeling && strings.HasPrefix(line, "^") {
			continue
		}
		_, lineName, _ := strings.Cut(strings.TrimSuffix(line, "\n"), " ")
		peeling = line != "" && line[0] != '#' && line[0] != '^' && lineName == name
		if peeling {
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	if removed {
		// renaming a temporary file rather than the lock keeps the lock held,
		// like git's packed-refs.new
		tmpPath, err := writeTempFile(repo.path, []byte(strings.Join(kept, "")))
		if err != nil {
			return err
		}
		defer os.Remove(tmpPath)
		if err := os.Rename(tmpPath, packedPath); err != nil {
			return err
		}
	}

	err = os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// lockRef creates the lock file at path+".lock", of a ref or of packed-refs,
// holding content, and returns its path. It waits for a concurrent writer to
// release the lock, or for ctx to be done.
//
// The lock file of a ref never exists empty: go-git lists refs by reading every
// file under refs/, lock files included, and an empty one makes the whole
// listing fail. packed-refs.lock is outside of refs/, and can be empty. So
// content is written to a temporary file outside of refs/ first, then
// hard-linked as the lock, which fails if the lock exists, like an exclusive
// create. On a filesystem without hard links, the lock is created exclusively
// then written, and the empty window comes back.
func (repo *GoGitRepo) lockRef(ctx context.Context, path string, content []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	lockPath := path + refLockSuffix

	tmpPath, err := writeTempFile(repo.path, content)
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
			return "", fmt.Errorf("%s exists: another process is updating refs, or crashed doing so; if no git or git-bug process is running, remove it", lockPath)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(wait):
		}
		wait = min(2*wait, 50*time.Millisecond)
	}
}

// writeTempFile writes content to a new temporary file in dir, and returns its
// path.
//
// The file becomes the ref, so it is created with the permissions of a ref
// created directly (0666 less the umask), not the 0600 of os.CreateTemp.
func writeTempFile(dir string, content []byte) (string, error) {
	// a random 64-bit name practically never collides: running out of
	// attempts means the filesystem rejects every name, not bad luck
	const attempts = 100
	var err error
	for range attempts {
		path := filepath.Join(dir, fmt.Sprintf("ref-%016x.tmp", rand.Uint64()))
		err = createFileExclusive(path, content)
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

// refName returns the name of the ref of key under prefix, rejecting the
// names git would reject: those also can't escape the repository once turned
// into a path, nor be taken for a lock file. The key must not leave its
// namespace either.
func refName(prefix string, key string) (string, error) {
	name := prefix + key
	if strings.Contains(key, "/") {
		return "", fmt.Errorf("invalid ref key %q", key)
	}
	if err := plumbing.ReferenceName(name).Validate(); err != nil {
		return "", fmt.Errorf("invalid ref %q: %w", name, err)
	}
	return name, nil
}

// ListRefs returns every local ref of a namespace, by key.
func (repo *GoGitRepo) ListRefs(ctx context.Context, namespace string) (map[string]Hash, error) {
	return repo.listRefs(ctx, refPrefix(namespace))
}

// ListTrackingRefs returns every tracking ref of a namespace for a remote,
// by key.
func (repo *GoGitRepo) ListTrackingRefs(ctx context.Context, remote string, namespace string) (map[string]Hash, error) {
	return repo.listRefs(ctx, trackingRefPrefix(remote, namespace))
}

// TODO: replace listRefs and listPackedRefs with IterReferences, see the top of
// this file.
//
// listRefs returns the refs under prefix, by the rest of their name. Loose refs
// take precedence over packed ones.
//
// Loose refs are read before packed-refs, as git does: git pack-refs writes the
// new packed-refs, then deletes the loose refs it packed. In this order, a ref
// is either still loose when read, or already in the packed-refs read after.
// The other order can miss it, or return its outdated packed value.
func (repo *GoGitRepo) listRefs(ctx context.Context, prefix string) (map[string]Hash, error) {
	// the prefix comes from a namespace, and a remote name: check it like a
	// ref name before walking it as a path
	if err := plumbing.ReferenceName(strings.TrimSuffix(prefix, "/")).Validate(); err != nil {
		return nil, fmt.Errorf("invalid ref prefix %q: %w", prefix, err)
	}

	root := filepath.Join(repo.path, filepath.FromSlash(prefix))
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
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
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// reading a ref file is mostly the CPU time of its syscalls, so the files
	// are read by a worker per CPU, each taking every workers-th one
	values := make([]string, len(paths))
	workers := min(runtime.GOMAXPROCS(0), len(paths), 8)
	var g errgroup.Group
	for w := range workers {
		g.Go(func() error {
			for i := w; i < len(paths); i += workers {
				if err := ctx.Err(); err != nil {
					return err
				}
				content, err := os.ReadFile(paths[i])
				if errors.Is(err, fs.ErrNotExist) {
					// removed while walking
					continue
				}
				if err != nil {
					return err
				}
				values[i] = strings.TrimSpace(string(content))
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	loose := make(map[string]Hash, len(paths))
	for i, value := range values {
		if !plumbing.IsHash(value) {
			// removed while walking, an empty file being written by a writer
			// not using a lock file or left behind by go-git (#2399), a
			// symbolic ref, or garbage: none is an entity, and like git, one
			// broken ref doesn't fail the listing of the others
			continue
		}
		rel, err := filepath.Rel(root, paths[i])
		if err != nil {
			return nil, err
		}
		loose[filepath.ToSlash(rel)] = Hash(value)
	}

	refs, err := repo.listPackedRefs(ctx, prefix)
	if err != nil {
		return nil, err
	}
	maps.Copy(refs, loose)

	return refs, nil
}

// listPackedRefs returns the refs of packed-refs under prefix, by the rest of
// their name.
func (repo *GoGitRepo) listPackedRefs(ctx context.Context, prefix string) (map[string]Hash, error) {
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
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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
