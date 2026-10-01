package repository

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/stretchr/testify/require"
)

func goGitRepo(t *testing.T, repo TestedRepo) *GoGitRepo {
	t.Helper()
	return repo.(*replaceKeyring).TestedRepo.(*GoGitRepo)
}

func storeTestCommits(t *testing.T, repo RepoData, n int) []Hash {
	t.Helper()
	commits := make([]Hash, n)
	for i := range commits {
		blobHash, err := repo.StoreData(randomData())
		require.NoError(t, err)
		treeHash, err := repo.StoreTree([]TreeEntry{{ObjectType: Blob, Hash: blobHash, Name: "blob"}})
		require.NoError(t, err)
		commits[i], err = repo.StoreCommit(treeHash)
		require.NoError(t, err)
	}
	return commits
}

// packRef moves a loose ref into packed-refs, as git pack-refs does.
func packRef(t *testing.T, repo *GoGitRepo, namespace string, key string) {
	t.Helper()
	name := refPrefix(namespace) + key
	path := filepath.Join(repo.path, filepath.FromSlash(name))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	f, err := os.OpenFile(filepath.Join(repo.path, "packed-refs"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	require.NoError(t, err)
	_, err = f.WriteString(strings.TrimSpace(string(content)) + " " + name + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Remove(path))
}

func requireNoLooseRef(t *testing.T, repo *GoGitRepo, namespace string, key string) {
	t.Helper()
	path := filepath.Join(repo.path, filepath.FromSlash(refPrefix(namespace)+key))
	require.NoFileExists(t, path)
	require.NoFileExists(t, path+refLockSuffix)
	requireNoTempFiles(t, repo)
}

// requireNoTempFiles checks that no update left its temporary file behind.
func requireNoTempFiles(t *testing.T, repo *GoGitRepo) {
	t.Helper()
	tmp, err := filepath.Glob(filepath.Join(repo.localStorage.Root(), "ref-*.tmp"))
	require.NoError(t, err)
	require.Empty(t, tmp)
}

func requireRef(t *testing.T, repo TestedRepo, namespace string, key string, expected Hash) {
	t.Helper()
	h, err := repo.ResolveRef(namespace, key)
	require.NoError(t, err)
	require.Equal(t, expected, h)
}

// shortenLockTimeout makes waiting for a held lock fail fast, for the duration
// of the test.
func shortenLockTimeout(t *testing.T) {
	timeout := refLockTimeout
	refLockTimeout = 20 * time.Millisecond
	t.Cleanup(func() { refLockTimeout = timeout })
}

func TestGoGitRepo_Refs(t *testing.T) {
	t.Run("UpdateRef", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)
		gr := goGitRepo(t, repo)
		commits := storeTestCommits(t, repo, 3)

		// A rejected update must leave nothing behind, including on a packed ref:
		// go-git used to leave an empty loose ref file that broke listing refs
		// (https://github.com/go-git/go-git/issues/2399).
		for _, tc := range []struct {
			name      string
			namespace string
			exists    bool
			old       Hash
		}{
			{"rejected on packed ref", "rejected-packed", true, commits[1]},
			{"rejected create on packed ref", "rejected-create", true, ""},
			{"rejected on missing ref", "rejected-missing", false, commits[1]},
		} {
			t.Run(tc.name, func(t *testing.T) {
				key := randomKey()
				expected := map[string]Hash{}
				if tc.exists {
					require.NoError(t, repo.UpdateRef(tc.namespace, key, "", commits[0]))
					packRef(t, gr, tc.namespace, key)
					requireRef(t, repo, tc.namespace, key, commits[0])
					expected[key] = commits[0]
				}

				require.ErrorIs(t, repo.UpdateRef(tc.namespace, key, tc.old, commits[2]), ErrRefChanged)
				requireNoLooseRef(t, gr, tc.namespace, key)

				refs, err := repo.ListRefs(tc.namespace)
				require.NoError(t, err)
				require.Equal(t, expected, refs)
			})
		}

		t.Run("loose ref takes precedence over packed", func(t *testing.T) {
			const namespace = "precedence"
			key := randomKey()
			require.NoError(t, repo.UpdateRef(namespace, key, "", commits[0]))
			packRef(t, gr, namespace, key)

			require.NoError(t, repo.UpdateRef(namespace, key, commits[0], commits[1]))
			requireRef(t, repo, namespace, key, commits[1])
			refs, err := repo.ListRefs(namespace)
			require.NoError(t, err)
			require.Equal(t, map[string]Hash{key: commits[1]}, refs)
		})

		// A ref gets the permissions of a file created normally (0666 less the
		// umask), not the owner-only ones of a temporary file, which would lock
		// out the other users of a shared repository.
		t.Run("permissions", func(t *testing.T) {
			const namespace = "perms"
			key := randomKey()
			require.NoError(t, repo.UpdateRef(namespace, key, "", commits[0]))

			reference := filepath.Join(t.TempDir(), "reference")
			require.NoError(t, createFileExclusive(reference, nil))
			want, err := os.Stat(reference)
			require.NoError(t, err)

			got, err := os.Stat(filepath.Join(gr.path, "refs", namespace, key))
			require.NoError(t, err)
			require.Equal(t, want.Mode().Perm(), got.Mode().Perm())
		})
	})

	// Concurrent updates of the same ref, all from the same old value: exactly one
	// must succeed, the others being rejected.
	t.Run("concurrent UpdateRef", func(t *testing.T) {
		const namespace = "concurrent"
		repo := CreateGoGitTestRepo(t, false)
		gr := goGitRepo(t, repo)
		commits := storeTestCommits(t, repo, 21)

		for _, tc := range []struct {
			name   string
			exists bool
		}{
			{"update", true},
			{"create", false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				key := randomKey()
				var old Hash
				candidates := commits[1:]
				if tc.exists {
					old = commits[0]
					require.NoError(t, repo.UpdateRef(namespace, key, "", old))
				}

				errs := make([]error, len(candidates))
				var wg sync.WaitGroup
				for i, commit := range candidates {
					wg.Go(func() { errs[i] = repo.UpdateRef(namespace, key, old, commit) })
				}
				wg.Wait()

				var winners []Hash
				for i, err := range errs {
					if err == nil {
						winners = append(winners, candidates[i])
						continue
					}
					require.ErrorIs(t, err, ErrRefChanged)
				}
				require.Len(t, winners, 1)
				requireRef(t, repo, namespace, key, winners[0])
				requireNoTempFiles(t, gr)
			})
		}
	})

	t.Run("RemoveRef", func(t *testing.T) {
		const namespace = "removing"
		repo := CreateGoGitTestRepo(t, false)
		gr := goGitRepo(t, repo)
		commits := storeTestCommits(t, repo, 2)

		for _, tc := range []struct {
			name  string
			setup func(t *testing.T, key string)
		}{
			{"missing ref", func(t *testing.T, key string) {}},
			{"loose ref", func(t *testing.T, key string) {
				require.NoError(t, repo.UpdateRef(namespace, key, "", commits[0]))
			}},
			{"packed ref", func(t *testing.T, key string) {
				require.NoError(t, repo.UpdateRef(namespace, key, "", commits[0]))
				packRef(t, gr, namespace, key)
			}},
			{"loose ref over packed", func(t *testing.T, key string) {
				require.NoError(t, repo.UpdateRef(namespace, key, "", commits[0]))
				packRef(t, gr, namespace, key)
				require.NoError(t, repo.UpdateRef(namespace, key, commits[0], commits[1]))
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				key := randomKey()
				tc.setup(t, key)

				require.NoError(t, repo.RemoveRef(namespace, key))
				_, err := repo.ResolveRef(namespace, key)
				require.ErrorIs(t, err, ErrNotFound)
				requireNoLooseRef(t, gr, namespace, key)

				// idempotent
				require.NoError(t, repo.RemoveRef(namespace, key))
			})
		}
	})

	// A lock held by someone else (another process, or one that crashed) makes
	// the operation fail, and is left alone. For a removal, this is what keeps it
	// from happening between the comparison and the rename of an update, which
	// would then recreate the removed ref.
	t.Run("held lock", func(t *testing.T) {
		const namespace = "locked"
		repo := CreateGoGitTestRepo(t, false)
		gr := goGitRepo(t, repo)
		commits := storeTestCommits(t, repo, 2)
		shortenLockTimeout(t)

		for _, tc := range []struct {
			name string
			op   func(key string) error
		}{
			{"update", func(key string) error { return repo.UpdateRef(namespace, key, commits[0], commits[1]) }},
			{"remove", func(key string) error { return repo.RemoveRef(namespace, key) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				key := randomKey()
				require.NoError(t, repo.UpdateRef(namespace, key, "", commits[0]))
				lockPath := filepath.Join(gr.path, "refs", namespace, key+refLockSuffix)
				require.NoError(t, os.WriteFile(lockPath, []byte("held"), 0666))

				err := tc.op(key)
				require.Error(t, err)
				require.NotErrorIs(t, err, ErrRefChanged)
				requireNoTempFiles(t, gr)

				content, err := os.ReadFile(lockPath)
				require.NoError(t, err)
				require.Equal(t, "held", string(content))
				requireRef(t, repo, namespace, key, commits[0])

				// once released, the operation goes through
				require.NoError(t, os.Remove(lockPath))
				require.NoError(t, tc.op(key))
			})
		}
	})

	t.Run("ListRefs", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)
		gr := goGitRepo(t, repo)
		commits := storeTestCommits(t, repo, 1)

		// Listing refs must survive what another writer leaves in the ref directory
		// while it works: its lock file, or an empty ref file.
		for _, tc := range []struct {
			name    string
			file    string
			content []byte
		}{
			{"skips lock file", randomKey() + refLockSuffix, []byte(commits[0] + "\n")},
			{"skips empty lock file", randomKey() + refLockSuffix, nil},
			{"skips empty ref file", randomKey(), nil},
			{"skips symbolic ref", randomKey(), []byte("ref: refs/heads/master\n")},
			{"skips garbage ref file", randomKey(), []byte("not a hash\n")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				namespace := "listing-" + strings.ReplaceAll(tc.name, " ", "-")
				key := randomKey()
				require.NoError(t, repo.UpdateRef(namespace, key, "", commits[0]))
				dir := filepath.Join(gr.path, "refs", namespace)
				require.NoError(t, os.WriteFile(filepath.Join(dir, tc.file), tc.content, 0666))

				refs, err := repo.ListRefs(namespace)
				require.NoError(t, err)
				require.Equal(t, map[string]Hash{key: commits[0]}, refs)
			})
		}

		t.Run("missing namespace", func(t *testing.T) {
			refs, err := repo.ListRefs("nothing-here")
			require.NoError(t, err)
			require.Empty(t, refs)
		})
	})

	t.Run("invalid ref names", func(t *testing.T) {
		repo := CreateGoGitTestRepo(t, false)
		commits := storeTestCommits(t, repo, 1)

		for _, tc := range []struct {
			name      string
			namespace string
			key       string
		}{
			{"empty key", "invalid", ""},
			{"key escaping", "invalid", "../escape"},
			{"key with slash", "invalid", "a/b"},
			{"key with backslash", "invalid", `a\b`},
			{"hidden key", "invalid", ".hidden"},
			{"lock key", "invalid", "x" + refLockSuffix},
			{"key with dot-dot", "invalid", "a..b"},
			{"key with space", "invalid", "a b"},
			{"key with control char", "invalid", "a\x01b"},
			{"namespace escaping", "../../..", "key"},
			{"empty namespace", "", "key"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				require.Error(t, repo.UpdateRef(tc.namespace, tc.key, "", commits[0]))
				require.Error(t, repo.RemoveRef(tc.namespace, tc.key))
			})
		}

		for _, tc := range []struct {
			name      string
			remote    string
			namespace string
		}{
			{"namespace escaping", "origin", "../../.."},
			{"remote escaping", "../../..", "invalid"},
			{"remote with space", "a b", "invalid"},
		} {
			t.Run("list "+tc.name, func(t *testing.T) {
				_, err := repo.ListTrackingRefs(tc.remote, tc.namespace)
				require.Error(t, err)
				if tc.remote == "origin" {
					_, err = repo.ListRefs(tc.namespace)
					require.Error(t, err)
				}
			})
		}
	})

	// The refs written here must be readable and writable by the git binary, and
	// the other way around, including once git packed them.
	t.Run("with git binary", func(t *testing.T) {
		requireGitBinary(t)

		repo := CreateGoGitTestRepo(t, false)
		gr := goGitRepo(t, repo)
		commits := storeTestCommits(t, repo, 3)

		// createPackedRefs creates two refs pointing to commits[0], packed by git.
		createPackedRefs := func(t *testing.T, namespace string) (string, string) {
			t.Helper()
			key1, key2 := randomKey(), randomKey()
			require.NoError(t, repo.UpdateRef(namespace, key1, "", commits[0]))
			require.NoError(t, repo.UpdateRef(namespace, key2, "", commits[0]))
			runGit(t, gr.path, "pack-refs", "--all")
			requireNoLooseRef(t, gr, namespace, key1)
			requireNoLooseRef(t, gr, namespace, key2)
			return key1, key2
		}

		t.Run("list refs packed by git", func(t *testing.T) {
			const namespace = "interop-list"
			key1, key2 := createPackedRefs(t, namespace)

			refs, err := repo.ListRefs(namespace)
			require.NoError(t, err)
			require.Equal(t, map[string]Hash{key1: commits[0], key2: commits[0]}, refs)
		})

		t.Run("rejected update leaves nothing that would break git", func(t *testing.T) {
			const namespace = "interop-rejected"
			key1, _ := createPackedRefs(t, namespace)

			require.ErrorIs(t, repo.UpdateRef(namespace, key1, commits[1], commits[2]), ErrRefChanged)
			require.Len(t, strings.Split(runGit(t, gr.path, "for-each-ref", refPrefix(namespace)), "\n"), 2)
		})

		t.Run("git sees our updates", func(t *testing.T) {
			const namespace = "interop-ours"
			key1, _ := createPackedRefs(t, namespace)

			require.NoError(t, repo.UpdateRef(namespace, key1, commits[0], commits[1]))
			require.Equal(t, commits[1].String(), runGit(t, gr.path, "rev-parse", refPrefix(namespace)+key1))
		})

		t.Run("we see git's updates", func(t *testing.T) {
			const namespace = "interop-theirs"
			_, key2 := createPackedRefs(t, namespace)

			runGit(t, gr.path, "update-ref", refPrefix(namespace)+key2, commits[2].String(), commits[0].String())
			requireRef(t, repo, namespace, key2, commits[2])
			require.NoError(t, repo.UpdateRef(namespace, key2, commits[2], commits[0]))
		})
	})

	// The canaries test go-git itself, not our code: each one checks that a go-git
	// ref handling defect listed at the top of gogit_refs.go is still there. If one
	// fails, go-git fixed it, and that reason for writing and listing refs
	// ourselves is gone.
	t.Run("canary", func(t *testing.T) {
		hash1 := plumbing.NewHash("1111111111111111111111111111111111111111")
		hash2 := plumbing.NewHash("2222222222222222222222222222222222222222")

		// plainStorage creates an empty bare repository, opened with
		// go-git's own storage.
		plainStorage := func(t *testing.T) (*filesystem.Storage, string) {
			dir := t.TempDir()
			_, err := gogit.PlainInit(dir, true)
			require.NoError(t, err)
			return filesystem.NewStorage(osfs.New(dir), cache.NewObjectLRUDefault()), dir
		}

		listRefs := func(s *filesystem.Storage) ([]string, error) {
			iter, err := s.IterReferences()
			if err != nil {
				return nil, err
			}
			var names []string
			err = iter.ForEach(func(ref *plumbing.Reference) error {
				names = append(names, ref.Name().String())
				return nil
			})
			return names, err
		}

		t.Run("a rejected update leaves an empty ref file", func(t *testing.T) {
			s, dir := plainStorage(t)

			err := s.CheckAndSetReference(
				plumbing.NewHashReference("refs/bugs/x", hash1),
				plumbing.NewHashReference("refs/bugs/x", hash2),
			)
			require.Error(t, err)

			msg := "go-git no longer leaves an empty ref behind (go-git#2399): one of the reasons listed in gogit_refs.go is gone"
			fi, err := os.Stat(filepath.Join(dir, "refs", "bugs", "x"))
			require.NoError(t, err, msg)
			require.Zero(t, fi.Size(), msg)
			_, err = listRefs(s)
			require.Error(t, err, msg)
		})

		t.Run("lock files are listed as refs", func(t *testing.T) {
			s, dir := plainStorage(t)
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "refs", "bugs"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "refs", "bugs", "x.lock"), []byte(hash1.String()+"\n"), 0o644))

			names, err := listRefs(s)
			require.NoError(t, err)
			require.Contains(t, names, "refs/bugs/x.lock",
				"go-git no longer lists lock files as refs: one of the reasons listed in gogit_refs.go is gone")
		})
	})
}
