package repository

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/go-git/go-billy/v5/osfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests change the repository behind GoGitRepo's back with the git
// binary, the way a `git fetch` or `git gc` in a terminal would while the webui
// is running.

// setupPrimedRepo returns a GoGitRepo whose packfile index has been built from
// one existing pack, and the directory of its worktree.
func setupPrimedRepo(t *testing.T) (*GoGitRepo, string) {
	t.Helper()

	dir := t.TempDir()
	repo, err := InitGoGitRepo(dir, namespace)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	h, err := repo.StoreData([]byte("already packed"))
	require.NoError(t, err)
	// repack only packs reachable objects.
	runGit(t, dir, "update-ref", "refs/test/primed", string(h))
	runGit(t, dir, "repack", "-a", "-d")
	require.NoDirExists(t, filepath.Join(dir, ".git", "objects", string(h)[:2]))

	// A lookup served from a pack makes go-git build its index of the
	// packs present right now.
	_, err = repo.ReadData(h)
	require.NoError(t, err)

	return repo, dir
}

// setupSource creates a repository with three commits on main, each adding one
// file under a/, and a branch "base" on the second commit.
func setupSource(t *testing.T) string {
	t.Helper()

	src := t.TempDir()
	runGit(t, src, "init", "-b", "main")
	require.NoError(t, os.Mkdir(filepath.Join(src, "a"), 0o755))
	for _, name := range []string{"x", "y", "z"} {
		require.NoError(t, os.WriteFile(filepath.Join(src, "a", name), []byte(name), 0o644))
		runGit(t, src, "add", ".")
		runGit(t, src, "commit", "-m", "add "+name)
	}
	runGit(t, src, "branch", "base", "HEAD~1")
	return src
}

// fetchAsPack fetches refspec from src into dir and makes git keep the result as
// a pack, as it does by default for any fetch of 100 objects or more.
func fetchAsPack(t *testing.T, dir, src, refspec string) {
	t.Helper()
	runGit(t, dir, "-c", "fetch.unpackLimit=1", "fetch", src, refspec)
}

// fetchAsLoose fetches refspec from src into dir as loose objects.
func fetchAsLoose(t *testing.T, dir, src, refspec string) {
	t.Helper()
	runGit(t, dir, "-c", "fetch.unpackLimit=100000", "fetch", src, refspec)
}

// Each subtest sets up its own repository: once a read has rebuilt the index,
// any later read would pass whether the fix works or not.
func TestGoGitStorage(t *testing.T) {
	requireGitBinary(t)

	// setupFetched returns a primed repository into which main was then
	// fetched as a pack behind its back.
	setupFetched := func(t *testing.T) (*GoGitRepo, string) {
		repo, dir := setupPrimedRepo(t)
		fetchAsPack(t, dir, setupSource(t), "main:refs/heads/main")
		return repo, dir
	}

	t.Run("object in external pack", func(t *testing.T) {
		t.Run("ReadCommit", func(t *testing.T) {
			repo, dir := setupFetched(t)

			commit, err := repo.ReadCommit(Hash(runGit(t, dir, "rev-parse", "main")))
			require.NoError(t, err)
			require.Equal(t, Hash(runGit(t, dir, "rev-parse", "main^{tree}")), commit.TreeHash)
		})

		t.Run("ReadTree", func(t *testing.T) {
			repo, dir := setupFetched(t)

			entries, err := repo.ReadTree(Hash(runGit(t, dir, "rev-parse", "main^{tree}")))
			require.NoError(t, err)
			require.Len(t, entries, 1)
		})

		t.Run("ReadData", func(t *testing.T) {
			repo, dir := setupFetched(t)

			r, err := repo.ReadData(Hash(runGit(t, dir, "rev-parse", "main:a/x")))
			require.NoError(t, err)
			require.NoError(t, r.Close())
		})

		t.Run("HasEncodedObject", func(t *testing.T) {
			repo, dir := setupFetched(t)
			s := repo.r.Storer.(*reindexingStorage)

			h := plumbing.NewHash(runGit(t, dir, "rev-parse", "main:a/x"))
			require.NoError(t, s.HasEncodedObject(h))
		})

		t.Run("DeltaObject", func(t *testing.T) {
			repo, dir := setupFetched(t)
			s := repo.r.Storer.(*reindexingStorage)

			h := plumbing.NewHash(runGit(t, dir, "rev-parse", "main:a/x"))
			obj, err := s.DeltaObject(plumbing.AnyObject, h)
			require.NoError(t, err)
			require.Equal(t, h, obj.Hash())
		})

		t.Run("HashesWithPrefix", func(t *testing.T) {
			repo, dir := setupFetched(t)
			s := repo.r.Storer.(*reindexingStorage)

			h := plumbing.NewHash(runGit(t, dir, "rev-parse", "main:a/x"))
			hashes, err := s.HashesWithPrefix(h[:4])
			require.NoError(t, err)
			require.Contains(t, hashes, h)
		})

		// An indexed object matching the prefix too must not hide the new
		// pack's: the empty prefix matches the primed blob, already indexed.
		t.Run("HashesWithPrefix with an indexed match", func(t *testing.T) {
			repo, dir := setupFetched(t)
			s := repo.r.Storer.(*reindexingStorage)

			hashes, err := s.HashesWithPrefix(nil)
			require.NoError(t, err)
			require.Contains(t, hashes, plumbing.NewHash(runGit(t, dir, "rev-parse", "refs/test/primed")))
			require.Contains(t, hashes, plumbing.NewHash(runGit(t, dir, "rev-parse", "main:a/x")))
		})
	})

	// The retry must neither hide a genuine miss nor rebuild the index when
	// the packs didn't change.
	t.Run("missing object", func(t *testing.T) {
		repo, _ := setupPrimedRepo(t)
		s := repo.r.Storer.(*reindexingStorage)
		packs := s.packs
		require.NotEmpty(t, packs)

		missing := plumbing.NewHash("0123456789abcdef0123456789abcdef01234567")
		_, err := s.EncodedObject(plumbing.AnyObject, missing)
		require.ErrorIs(t, err, plumbing.ErrObjectNotFound)
		require.False(t, s.stale)
		// a rebuild would record a new list of the same packs: compare the lists
		// themselves, not their content
		require.Same(t, &packs[0], &s.packs[0], "the index was rebuilt")

		_, err = repo.ReadCommit(Hash(missing.String()))
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("IterEncodedObjects", func(t *testing.T) {
		repo, dir := setupFetched(t)
		s := repo.r.Storer.(*reindexingStorage)

		iter, err := s.IterEncodedObjects(plumbing.BlobObject)
		require.NoError(t, err)
		var blobs []plumbing.Hash
		err = iter.ForEach(func(obj plumbing.EncodedObject) error {
			blobs = append(blobs, obj.Hash())
			return nil
		})
		require.NoError(t, err)

		for _, path := range []string{"a/x", "a/y", "a/z"} {
			require.Contains(t, blobs, plumbing.NewHash(runGit(t, dir, "rev-parse", "main:"+path)))
		}
	})

	// A pack whose .idx git hasn't written yet can't be indexed, and go-git
	// would crash on reaching it: iterating must fail instead.
	t.Run("IterEncodedObjects with a pack being written", func(t *testing.T) {
		repo, dir := setupPrimedRepo(t)
		s := repo.r.Storer.(*reindexingStorage)
		pack := filepath.Join(dir, ".git", "objects", "pack", "pack-0123456789abcdef0123456789abcdef01234567.pack")
		require.NoError(t, os.WriteFile(pack, []byte("not yet indexed"), 0o644))

		_, err := s.IterEncodedObjects(plumbing.BlobObject)
		require.Error(t, err)

		// once the pack is complete (here: gone), iterating works again
		require.NoError(t, os.Remove(pack))
		iter, err := s.IterEncodedObjects(plumbing.BlobObject)
		require.NoError(t, err)
		require.NoError(t, iter.ForEach(func(plumbing.EncodedObject) error { return nil }))
	})

	// go-git's own fetch writes its pack through PackfileWriter, which adds it
	// to the index when the writer closes.
	t.Run("go-git fetch", func(t *testing.T) {
		setup := func(t *testing.T) (*GoGitRepo, Hash) {
			repo, _ := setupPrimedRepo(t)
			src := setupSource(t)
			runGit(t, src, "update-ref", "refs/bugs/b1", "main")
			AddRemote(t, repo, "src", src)
			return repo, Hash(runGit(t, src, "rev-parse", "main"))
		}

		t.Run("objects readable", func(t *testing.T) {
			repo, head := setup(t)

			_, err := repo.FetchRefs("src", "bugs")
			require.NoError(t, err)

			refs, err := repo.ListTrackingRefs("src", "bugs")
			require.NoError(t, err)
			require.Equal(t, map[string]Hash{"b1": head}, refs)

			log, err := repo.CommitLog("refs/remotes/src/bugs/b1", "", 0, "", nil, nil)
			require.NoError(t, err)
			require.Len(t, log, 3)
		})

		// The index is rebuilt eagerly after a reindex because the pack
		// writer adds to the index map on close, and would panic on a nil
		// one. Run with -race.
		t.Run("during reindex", func(t *testing.T) {
			repo, head := setup(t)
			s := repo.r.Storer.(*reindexingStorage)

			done := make(chan struct{})
			var wg sync.WaitGroup
			wg.Go(func() {
				for {
					select {
					case <-done:
						return
					default:
						s.Reindex()
					}
				}
			})
			_, err := repo.FetchRefs("src", "bugs")
			close(done)
			wg.Wait()
			require.NoError(t, err)

			_, err = repo.ReadCommit(head)
			require.NoError(t, err)
		})
	})

	// A fix applied at GoGitRepo's call sites passes the subtests above, but
	// not these: the tip commit is loose so the direct lookup of it succeeds,
	// and the first packed object is only reached by go-git itself, deep in a
	// history walk.
	t.Run("traversal into external pack", func(t *testing.T) {
		setup := func(t *testing.T) *GoGitRepo {
			repo, dir := setupPrimedRepo(t)
			src := setupSource(t)
			fetchAsPack(t, dir, src, "base:refs/heads/base")
			fetchAsLoose(t, dir, src, "main:refs/heads/main")
			return repo
		}

		t.Run("CommitLog", func(t *testing.T) {
			repo := setup(t)

			log, err := repo.CommitLog("main", "", 0, "", nil, nil)
			require.NoError(t, err)
			require.Len(t, log, 3)
		})

		t.Run("LastCommitForEntries", func(t *testing.T) {
			repo := setup(t)

			last, err := repo.LastCommitForEntries("main", "a", []string{"x", "y", "z"})
			require.NoError(t, err)
			require.Equal(t, "add x", last["x"].Message)
			require.Equal(t, "add y", last["y"].Message)
			require.Equal(t, "add z", last["z"].Message)
		})
	})

	// `git gc` replaces the packs the index points to: the lookup then fails on
	// opening a file that is gone, not with ErrObjectNotFound.
	t.Run("external repack", func(t *testing.T) {
		repo, dir := setupFetched(t)

		// Load the new pack into the index, without reading the object below.
		_, err := repo.ReadCommit(Hash(runGit(t, dir, "rev-parse", "main")))
		require.NoError(t, err)

		runGit(t, dir, "repack", "-a", "-d")

		r, err := repo.ReadData(Hash(runGit(t, dir, "rev-parse", "main:a/y")))
		require.NoError(t, err)
		require.NoError(t, r.Close())
	})

	// Reindexing must not race with reads, including reads that do not hold
	// GoGitRepo's rMutex. Run with -race.
	t.Run("concurrent reindex", func(t *testing.T) {
		repo, dir := setupFetched(t)
		blob := Hash(runGit(t, dir, "rev-parse", "main:a/z"))
		s := repo.r.Storer.(*reindexingStorage)

		var wg sync.WaitGroup
		for range 4 {
			wg.Go(func() {
				for range 50 {
					r, err := repo.ReadData(blob)
					if assert.NoError(t, err) {
						_ = r.Close()
					}
				}
			})
		}
		wg.Go(func() {
			for range 50 {
				s.Reindex()
			}
		})
		wg.Go(func() {
			for range 50 {
				_, _ = s.EncodedObjectSize(plumbing.NewHash(string(blob)))
			}
		})
		wg.Wait()
	})

	// The canaries test go-git itself, not our code. The preconditions are
	// behaviors of go-git that the workarounds rely on: if one fails after a
	// go-git upgrade, the workaround named in the message may be broken. The
	// bugs are go-git defects that the workarounds exist for: if one fails,
	// go-git fixed it, and the workaround named in the message can go.
	t.Run("canary", func(t *testing.T) {
		// plainStorage opens dir's git directory with go-git's own storage,
		// as gogit.PlainOpen would, without reindexingStorage.
		plainStorage := func(dir string) *filesystem.Storage {
			return filesystem.NewStorage(osfs.New(filepath.Join(dir, ".git")), cache.NewObjectLRUDefault())
		}

		// packBytes returns the content of a pack holding all of src's objects.
		packBytes := func(t *testing.T, src string) []byte {
			runGit(t, src, "repack", "-a", "-d")
			packs, err := filepath.Glob(filepath.Join(src, ".git", "objects", "pack", "pack-*.pack"))
			require.NoError(t, err)
			require.Len(t, packs, 1)
			data, err := os.ReadFile(packs[0])
			require.NoError(t, err)
			return data
		}

		t.Run("preconditions", func(t *testing.T) {
			t.Run("looking up the zero hash builds the index", func(t *testing.T) {
				_, dir := setupPrimedRepo(t)
				s := plainStorage(dir)

				require.ErrorIs(t, s.HasEncodedObject(plumbing.ZeroHash), plumbing.ErrObjectNotFound,
					"reindexingStorage.loadIndex treats any other error as a failed index build")

				fetchAsPack(t, dir, setupSource(t), "main:refs/heads/main")
				_, err := s.EncodedObject(plumbing.AnyObject, plumbing.NewHash(runGit(t, dir, "rev-parse", "main")))
				require.ErrorIs(t, err, plumbing.ErrObjectNotFound,
					"HasEncodedObject no longer builds the index: reindexingStorage.loadIndex must build it another way")
			})

			t.Run("ObjectPacks lists the packs on disk", func(t *testing.T) {
				_, dir := setupPrimedRepo(t)
				s := plainStorage(dir)

				before, err := s.ObjectPacks()
				require.NoError(t, err)
				fetchAsPack(t, dir, setupSource(t), "main:refs/heads/main")
				after, err := s.ObjectPacks()
				require.NoError(t, err)
				require.Len(t, after, len(before)+1,
					"ObjectPacks no longer reads the directory: reindexingStorage.refresh can't detect new packs with it")
			})

			t.Run("the pack writer indexes its pack on close", func(t *testing.T) {
				_, dir := setupPrimedRepo(t)
				s := plainStorage(dir)
				src := setupSource(t)
				head := plumbing.NewHash(runGit(t, src, "rev-parse", "main"))
				require.ErrorIs(t, s.HasEncodedObject(plumbing.ZeroHash), plumbing.ErrObjectNotFound) // builds the index

				w, err := s.PackfileWriter()
				require.NoError(t, err)
				_, err = w.Write(packBytes(t, src))
				require.NoError(t, err)
				require.NoError(t, w.Close())

				_, err = s.EncodedObject(plumbing.AnyObject, head)
				require.NoError(t, err,
					"the pack writer no longer adds its pack to the index on close: review what lockedPackWriter and the eager index rebuild in reindexingStorage guard")
			})

			t.Run("traversals read objects through Repository.Storer", func(t *testing.T) {
				src := setupSource(t)
				s := &countingStorage{Storage: plainStorage(src), reads: map[plumbing.Hash]int{}}
				r, err := gogit.Open(s, nil)
				require.NoError(t, err)

				head := plumbing.NewHash(runGit(t, src, "rev-parse", "main"))
				first := plumbing.NewHash(runGit(t, src, "rev-parse", "main~2"))
				tree := plumbing.NewHash(runGit(t, src, "rev-parse", "main:a"))

				iter, err := r.Log(&gogit.LogOptions{From: head})
				require.NoError(t, err)
				require.NoError(t, iter.ForEach(func(c *object.Commit) error {
					_, err := c.Tree()
					if err != nil {
						return err
					}
					_, err = c.File("a/x")
					return err
				}))

				msg := "go-git reads some objects around the storer it was opened with: reindexingStorage no longer sees every read"
				require.Positive(t, s.reads[first], msg)
				require.Positive(t, s.reads[tree], msg)
			})
		})

		t.Run("objects bugs", func(t *testing.T) {
			t.Run("a pack added externally is invisible", func(t *testing.T) {
				_, dir := setupPrimedRepo(t)
				s := plainStorage(dir)
				_, err := s.EncodedObject(plumbing.AnyObject, plumbing.NewHash(runGit(t, dir, "rev-parse", "refs/test/primed")))
				require.NoError(t, err)

				fetchAsPack(t, dir, setupSource(t), "main:refs/heads/main")
				_, err = s.EncodedObject(plumbing.AnyObject, plumbing.NewHash(runGit(t, dir, "rev-parse", "main")))
				require.ErrorIs(t, err, plumbing.ErrObjectNotFound,
					"go-git now sees packs added by other processes: reindexingStorage's retry can go")
			})

			t.Run("a pack deleted externally fails the read", func(t *testing.T) {
				_, dir := setupPrimedRepo(t)
				fetchAsPack(t, dir, setupSource(t), "main:refs/heads/main")
				s := plainStorage(dir)
				_, err := s.EncodedObject(plumbing.AnyObject, plumbing.NewHash(runGit(t, dir, "rev-parse", "main")))
				require.NoError(t, err)

				runGit(t, dir, "repack", "-a", "-d")
				_, err = s.EncodedObject(plumbing.AnyObject, plumbing.NewHash(runGit(t, dir, "rev-parse", "main:a/y")))
				require.Error(t, err,
					"go-git now recovers from packs removed by other processes: reindexingStorage's retry can go")
			})

			t.Run("iterating objects crashes on a pack added externally", func(t *testing.T) {
				// The crash skips closing the pack go-git just opened, and
				// Windows can't delete an open file, so the temp dir cleanup
				// would fail. The bug isn't Windows specific: other platforms
				// still catch go-git fixing it.
				if runtime.GOOS == "windows" {
					t.Skip("go-git leaks the pack's file handle when crashing")
				}

				_, dir := setupPrimedRepo(t)
				s := plainStorage(dir)
				require.ErrorIs(t, s.HasEncodedObject(plumbing.ZeroHash), plumbing.ErrObjectNotFound) // builds the index

				fetchAsPack(t, dir, setupSource(t), "main:refs/heads/main")
				require.Panics(t, func() {
					iter, err := s.IterEncodedObjects(plumbing.BlobObject)
					if err == nil {
						_ = iter.ForEach(func(plumbing.EncodedObject) error { return nil })
					}
				}, "IterEncodedObjects no longer crashes on a pack it hasn't indexed (go-git#2448): reindexingStorage.IterEncodedObjects needn't refresh first")
			})

			t.Run("closing a pack writer after Reindex crashes", func(t *testing.T) {
				_, dir := setupPrimedRepo(t)
				s := plainStorage(dir)
				pack := packBytes(t, setupSource(t))

				w, err := s.PackfileWriter()
				require.NoError(t, err)
				s.Reindex()
				_, err = w.Write(pack)
				require.NoError(t, err)
				require.Panics(t, func() { _ = w.Close() },
					"the pack writer no longer crashes on a dropped index: reindexingStorage needn't rebuild the index right after Reindex")
			})
		})
	})
}

// countingStorage counts the object reads made through it.
type countingStorage struct {
	*filesystem.Storage
	reads map[plumbing.Hash]int
}

func (s *countingStorage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	s.reads[h]++
	return s.Storage.EncodedObject(t, h)
}
