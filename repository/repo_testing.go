package repository

import (
	"io"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/util/lamport"
)

type RepoCreator func(t testing.TB, bare bool) TestedRepo

// Test suite for a Repo implementation
func RepoTest(t *testing.T, creator RepoCreator) {
	for bare, name := range map[bool]string{
		false: "Plain",
		true:  "Bare",
	} {
		t.Run(name, func(t *testing.T) {
			repo := creator(t, bare)

			t.Run("Data", func(t *testing.T) {
				RepoDataTest(t, repo)
				RepoDataSignatureTest(t, repo)
			})

			t.Run("Browse", func(t *testing.T) {
				RepoBrowseTest(t, repo)
			})

			t.Run("Config", func(t *testing.T) {
				RepoConfigTest(t, repo)
			})

			t.Run("Storage", func(t *testing.T) {
				RepoStorageTest(t, repo)
			})

			t.Run("Index", func(t *testing.T) {
				RepoIndexTest(t, repo)
			})

			t.Run("Clocks", func(t *testing.T) {
				RepoClockTest(t, repo)
			})
		})
	}
}

// helper to test a RepoConfig
func RepoConfigTest(t *testing.T, repo RepoConfig) {
	testConfig(t, repo.LocalConfig())
}

func RepoStorageTest(t *testing.T, repo RepoStorage) {
	storage := repo.LocalStorage()

	err := storage.MkdirAll("foo/bar", 0755)
	require.NoError(t, err)

	f, err := storage.Create("foo/bar/foofoo")
	require.NoError(t, err)

	_, err = f.Write([]byte("hello"))
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)

	// remove all
	err = storage.RemoveAll(".")
	require.NoError(t, err)

	fi, err := storage.ReadDir(".")
	// a real FS would remove the root directory with RemoveAll and subsequent call would fail
	// a memory FS would still have a virtual root and subsequent call would succeed
	// not ideal, but will do for now
	if err == nil {
		require.Empty(t, fi)
	} else {
		require.True(t, os.IsNotExist(err))
	}
}

func randomHash() Hash {
	var letterRunes = "abcdef0123456789"
	b := make([]byte, idLengthSHA256)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return Hash(b)
}

// helper to test a RepoData
func RepoDataTest(t *testing.T, repo RepoData) {
	// Blob

	data := randomData()

	blobHash1, err := repo.StoreData(data)
	require.NoError(t, err)
	require.True(t, blobHash1.IsValid())

	blob1Read, err := repo.ReadData(blobHash1)
	require.NoError(t, err)
	require.Equal(t, data, blob1Read)

	_, err = repo.ReadData(randomHash())
	require.ErrorIs(t, err, ErrNotFound)

	// Tree

	blobHash2, err := repo.StoreData(randomData())
	require.NoError(t, err)
	blobHash3, err := repo.StoreData(randomData())
	require.NoError(t, err)

	tree1 := []TreeEntry{
		{
			ObjectType: Blob,
			Hash:       blobHash1,
			Name:       "blob1",
		},
		{
			ObjectType: Blob,
			Hash:       blobHash2,
			Name:       "blob2",
		},
	}

	treeHash1, err := repo.StoreTree(tree1)
	require.NoError(t, err)
	require.True(t, treeHash1.IsValid())

	tree1Read, err := repo.ReadTree(treeHash1)
	require.NoError(t, err)
	require.ElementsMatch(t, tree1, tree1Read)

	tree2 := []TreeEntry{
		{
			ObjectType: Tree,
			Hash:       treeHash1,
			Name:       "tree1",
		},
		{
			ObjectType: Blob,
			Hash:       blobHash3,
			Name:       "blob3",
		},
	}

	treeHash2, err := repo.StoreTree(tree2)
	require.NoError(t, err)
	require.True(t, treeHash2.IsValid())

	tree2Read, err := repo.ReadTree(treeHash2)
	require.NoError(t, err)
	require.ElementsMatch(t, tree2, tree2Read)

	_, err = repo.ReadTree(randomHash())
	require.ErrorIs(t, err, ErrNotFound)

	// Commit

	commit1, err := repo.StoreCommit(treeHash1)
	require.NoError(t, err)
	require.True(t, commit1.IsValid())

	// commit with a parent
	commit2, err := repo.StoreCommit(treeHash2, commit1)
	require.NoError(t, err)
	require.True(t, commit2.IsValid())

	// ReadTree should accept tree and commit hashes
	tree1read, err := repo.ReadTree(commit1)
	require.NoError(t, err)
	require.Equal(t, tree1read, tree1)

	c2, err := repo.ReadCommit(commit2)
	require.NoError(t, err)
	c2expected := Commit{Hash: commit2, Parents: []Hash{commit1}, TreeHash: treeHash2}
	require.Equal(t, c2expected, c2)

	_, err = repo.ReadCommit(randomHash())
	require.ErrorIs(t, err, ErrNotFound)

	// Ref

	exist1, err := repo.RefExist("refs/bugs/ref1")
	require.NoError(t, err)
	require.False(t, exist1)

	err = repo.UpdateRef("refs/bugs/ref1", commit2)
	require.NoError(t, err)

	exist1, err = repo.RefExist("refs/bugs/ref1")
	require.NoError(t, err)
	require.True(t, exist1)

	h, err := repo.ResolveRef("refs/bugs/ref1")
	require.NoError(t, err)
	require.Equal(t, commit2, h)

	ls, err := repo.ListRefs("refs/bugs")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"refs/bugs/ref1"}, ls)

	err = repo.CopyRef("refs/bugs/ref1", "refs/bugs/ref2")
	require.NoError(t, err)

	ls, err = repo.ListRefs("refs/bugs")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"refs/bugs/ref1", "refs/bugs/ref2"}, ls)

	commits, err := repo.ListCommits("refs/bugs/ref2")
	require.NoError(t, err)
	require.Equal(t, []Hash{commit1, commit2}, commits)

	_, err = repo.ResolveRef("/refs/bugs/refnotexist")
	require.ErrorIs(t, err, ErrNotFound)

	err = repo.CopyRef("/refs/bugs/refnotexist", "refs/foo")
	require.ErrorIs(t, err, ErrNotFound)

	// Cleanup

	err = repo.RemoveRef("refs/bugs/ref1")
	require.NoError(t, err)

	// RemoveRef is idempotent
	err = repo.RemoveRef("refs/bugs/ref1")
	require.NoError(t, err)
}

func RepoDataSignatureTest(t *testing.T, repo RepoData) {
	data := randomData()

	blobHash, err := repo.StoreData(data)
	require.NoError(t, err)

	treeHash, err := repo.StoreTree([]TreeEntry{
		{
			ObjectType: Blob,
			Hash:       blobHash,
			Name:       "blob",
		},
	})
	require.NoError(t, err)

	pgpEntity1, err := openpgp.NewEntity("", "", "", nil)
	require.NoError(t, err)
	keyring1 := openpgp.EntityList{pgpEntity1}

	pgpEntity2, err := openpgp.NewEntity("", "", "", nil)
	require.NoError(t, err)
	keyring2 := openpgp.EntityList{pgpEntity2}

	commitHash1, err := repo.StoreSignedCommit(treeHash, pgpEntity1)
	require.NoError(t, err)

	commit1, err := repo.ReadCommit(commitHash1)
	require.NoError(t, err)

	_, err = openpgp.CheckDetachedSignature(keyring1, commit1.SignedData, commit1.Signature, nil)
	require.NoError(t, err)

	_, err = openpgp.CheckDetachedSignature(keyring2, commit1.SignedData, commit1.Signature, nil)
	require.Error(t, err)

	commitHash2, err := repo.StoreSignedCommit(treeHash, pgpEntity1, commitHash1)
	require.NoError(t, err)

	commit2, err := repo.ReadCommit(commitHash2)
	require.NoError(t, err)

	_, err = openpgp.CheckDetachedSignature(keyring1, commit2.SignedData, commit2.Signature, nil)
	require.NoError(t, err)

	_, err = openpgp.CheckDetachedSignature(keyring2, commit2.SignedData, commit2.Signature, nil)
	require.Error(t, err)
}

func RepoIndexTest(t *testing.T, repo RepoIndex) {
	idx, err := repo.GetIndex("a")
	require.NoError(t, err)

	// simple indexing
	err = idx.IndexOne("id1", []string{"foo", "bar", "foobar barfoo"})
	require.NoError(t, err)

	// batched indexing
	indexer, closer := idx.IndexBatch()
	err = indexer("id2", []string{"hello", "foo bar"})
	require.NoError(t, err)
	err = indexer("id3", []string{"Hola", "Esta bien"})
	require.NoError(t, err)
	err = closer()
	require.NoError(t, err)

	// search
	res, err := idx.Search([]string{"foobar"})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"id1"}, res)

	res, err = idx.Search([]string{"foo"})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"id1", "id2"}, res)

	// re-indexing an item replace previous versions
	err = idx.IndexOne("id2", []string{"hello"})
	require.NoError(t, err)

	res, err = idx.Search([]string{"foo"})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"id1"}, res)

	err = idx.Clear()
	require.NoError(t, err)

	res, err = idx.Search([]string{"foo"})
	require.NoError(t, err)
	require.Empty(t, res)
}

// helper to test a RepoClock
func RepoClockTest(t *testing.T, repo RepoClock) {
	allClocks, err := repo.AllClocks()
	require.NoError(t, err)
	require.Len(t, allClocks, 0)

	clock, err := repo.GetOrCreateClock("foo")
	require.NoError(t, err)
	require.Equal(t, lamport.Time(1), clock.Time())

	time, err := clock.Increment()
	require.NoError(t, err)
	require.Equal(t, lamport.Time(2), time)
	require.Equal(t, lamport.Time(2), clock.Time())

	clock2, err := repo.GetOrCreateClock("foo")
	require.NoError(t, err)
	require.Equal(t, lamport.Time(2), clock2.Time())

	clock3, err := repo.GetOrCreateClock("bar")
	require.NoError(t, err)
	require.Equal(t, lamport.Time(1), clock3.Time())

	allClocks, err = repo.AllClocks()
	require.NoError(t, err)
	require.Equal(t, map[string]lamport.Clock{
		"foo": clock,
		"bar": clock3,
	}, allClocks)
}

func randomData() []byte {
	var letterRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return b
}

// browsable is the interface required by RepoBrowseTest.
type browsable interface {
	RepoConfig
	RepoData
	RepoBrowse
}

// RepoBrowseTest exercises the RepoBrowse interface against any implementation.
//
// Commit graph (oldest → newest):
//
//	c1 ── c2 ── c3   refs/heads/main (default)
//	       └────────  refs/heads/feature
//	c1 ←── refs/tags/v1.0
func RepoBrowseTest(t *testing.T, repo browsable) {
	t.Helper()

	require.NoError(t, repo.LocalConfig().StoreString("init.defaultBranch", "main"))

	// ── build fixture ─────────────────────────────────────────────────────────

	readmeV1 := []byte("# Hello\n")
	readmeV3 := []byte("# Hello\n\n## Updated\n")
	mainV1 := []byte("package main\n")
	mainV2 := []byte("package main\n\n// updated\n")
	libV1 := []byte("package lib\n")
	utilV1 := []byte("package util\n")

	hReadmeV1, err := repo.StoreData(readmeV1)
	require.NoError(t, err)
	hReadmeV3, err := repo.StoreData(readmeV3)
	require.NoError(t, err)
	hMainV1, err := repo.StoreData(mainV1)
	require.NoError(t, err)
	hMainV2, err := repo.StoreData(mainV2)
	require.NoError(t, err)
	hLibV1, err := repo.StoreData(libV1)
	require.NoError(t, err)
	hUtilV1, err := repo.StoreData(utilV1)
	require.NoError(t, err)

	srcTreeV1, err := repo.StoreTree([]TreeEntry{
		{ObjectType: Blob, Hash: hLibV1, Name: "lib.go"},
	})
	require.NoError(t, err)
	rootTreeV1, err := repo.StoreTree([]TreeEntry{
		{ObjectType: Blob, Hash: hReadmeV1, Name: "README.md"},
		{ObjectType: Blob, Hash: hMainV1, Name: "main.go"},
		{ObjectType: Tree, Hash: srcTreeV1, Name: "src"},
	})
	require.NoError(t, err)

	srcTreeV2, err := repo.StoreTree([]TreeEntry{
		{ObjectType: Blob, Hash: hLibV1, Name: "lib.go"},
		{ObjectType: Blob, Hash: hUtilV1, Name: "util.go"},
	})
	require.NoError(t, err)
	rootTreeV2, err := repo.StoreTree([]TreeEntry{
		{ObjectType: Blob, Hash: hReadmeV1, Name: "README.md"},
		{ObjectType: Blob, Hash: hMainV2, Name: "main.go"},
		{ObjectType: Tree, Hash: srcTreeV2, Name: "src"},
	})
	require.NoError(t, err)

	rootTreeV3, err := repo.StoreTree([]TreeEntry{
		{ObjectType: Blob, Hash: hReadmeV3, Name: "README.md"},
		{ObjectType: Blob, Hash: hMainV2, Name: "main.go"},
		{ObjectType: Tree, Hash: srcTreeV2, Name: "src"},
	})
	require.NoError(t, err)

	c1, err := repo.StoreCommit(rootTreeV1)
	require.NoError(t, err)
	c2, err := repo.StoreCommit(rootTreeV2, c1)
	require.NoError(t, err)
	c3, err := repo.StoreCommit(rootTreeV3, c2)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateRef("refs/heads/main", c3))
	require.NoError(t, repo.UpdateRef("refs/heads/feature", c2))
	require.NoError(t, repo.UpdateRef("refs/tags/v1.0", c1))

	// ── Branches ──────────────────────────────────────────────────────────────

	t.Run("Branches", func(t *testing.T) {
		branches, err := repo.Branches()
		require.NoError(t, err)
		require.Len(t, branches, 2)

		byName := make(map[string]BranchInfo)
		for _, b := range branches {
			byName[b.Name] = b
		}

		require.Equal(t, c3, byName["main"].Hash)
		require.True(t, byName["main"].IsDefault)

		require.Equal(t, c2, byName["feature"].Hash)
		require.False(t, byName["feature"].IsDefault)
	})

	// ── Tags ──────────────────────────────────────────────────────────────────

	t.Run("Tags", func(t *testing.T) {
		tags, err := repo.Tags()
		require.NoError(t, err)
		require.Len(t, tags, 1)
		require.Equal(t, "v1.0", tags[0].Name)
		require.Equal(t, c1, tags[0].Hash)
	})

	// ── TreeAtPath ────────────────────────────────────────────────────────────

	t.Run("TreeAtPath", func(t *testing.T) {
		entries, err := repo.TreeAtPath("main", "")
		require.NoError(t, err)
		byName := make(map[string]TreeEntry)
		for _, e := range entries {
			byName[e.Name] = e
		}
		require.Equal(t, Blob, byName["README.md"].ObjectType)
		require.Equal(t, Blob, byName["main.go"].ObjectType)
		require.Equal(t, Tree, byName["src"].ObjectType)

		// subdirectory
		srcEntries, err := repo.TreeAtPath("main", "src")
		require.NoError(t, err)
		srcByName := make(map[string]TreeEntry)
		for _, e := range srcEntries {
			srcByName[e.Name] = e
		}
		require.Equal(t, Blob, srcByName["lib.go"].ObjectType)
		require.Equal(t, Blob, srcByName["util.go"].ObjectType)

		// v1.0 tag (at c1) predates util.go — src only has lib.go
		v1Src, err := repo.TreeAtPath("v1.0", "src")
		require.NoError(t, err)
		require.Len(t, v1Src, 1)
		require.Equal(t, "lib.go", v1Src[0].Name)

		// unknown ref
		_, err = repo.TreeAtPath("nonexistent-ref", "")
		require.ErrorIs(t, err, ErrNotFound)

		// path resolves to a blob, not a tree
		_, err = repo.TreeAtPath("main", "README.md")
		require.Error(t, err)
	})

	// ── BlobAtPath ────────────────────────────────────────────────────────────

	t.Run("BlobAtPath", func(t *testing.T) {
		rc, size, hash, err := repo.BlobAtPath("main", "README.md")
		require.NoError(t, err)
		defer rc.Close()
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, readmeV3, data)
		require.Equal(t, int64(len(readmeV3)), size)
		require.NotEmpty(t, hash)

		// feature branch still has readmeV1
		rc2, _, _, err := repo.BlobAtPath("feature", "README.md")
		require.NoError(t, err)
		data2, err := io.ReadAll(rc2)
		rc2.Close()
		require.NoError(t, err)
		require.Equal(t, readmeV1, data2)

		// file in subdirectory
		rc3, _, _, err := repo.BlobAtPath("main", "src/lib.go")
		require.NoError(t, err)
		data3, err := io.ReadAll(rc3)
		rc3.Close()
		require.NoError(t, err)
		require.Equal(t, libV1, data3)

		// path not found
		_, _, _, err = repo.BlobAtPath("main", "nonexistent.go")
		require.ErrorIs(t, err, ErrNotFound)

		// hash is stable across calls for the same content
		rc4, _, hash2, err := repo.BlobAtPath("main", "README.md")
		require.NoError(t, err)
		rc4.Close()
		require.Equal(t, hash, hash2, "blob hash should be stable across calls")

		// different content → different hash
		rc5, _, hashLib, err := repo.BlobAtPath("main", "src/lib.go")
		require.NoError(t, err)
		rc5.Close()
		require.NotEqual(t, hash, hashLib, "different files should have different hashes")
	})

	// ── CommitLog ─────────────────────────────────────────────────────────────

	t.Run("CommitLog", func(t *testing.T) {
		// all commits, newest first
		commits, err := repo.CommitLog("main", "", 10, "", nil, nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
		require.Equal(t, c3, commits[0].Hash)
		require.Equal(t, c2, commits[1].Hash)
		require.Equal(t, c1, commits[2].Hash)

		// limit
		limited, err := repo.CommitLog("main", "", 2, "", nil, nil)
		require.NoError(t, err)
		require.Len(t, limited, 2)
		require.Equal(t, c3, limited[0].Hash)
		require.Equal(t, c2, limited[1].Hash)

		// after cursor (exclusive): start after c3 → get c2, c1
		after, err := repo.CommitLog("main", "", 10, c3, nil, nil)
		require.NoError(t, err)
		require.Len(t, after, 2)
		require.Equal(t, c2, after[0].Hash)
		require.Equal(t, c1, after[1].Hash)

		// feature branch only has c1, c2
		featureLog, err := repo.CommitLog("feature", "", 10, "", nil, nil)
		require.NoError(t, err)
		require.Len(t, featureLog, 2)
		require.Equal(t, c2, featureLog[0].Hash)

		// path filtering: only commits that touched the given path
		// README.md was created in c1 and updated in c3
		readmeLog, err := repo.CommitLog("main", "README.md", 10, "", nil, nil)
		require.NoError(t, err)
		require.Len(t, readmeLog, 2)
		require.Equal(t, c3, readmeLog[0].Hash)
		require.Equal(t, c1, readmeLog[1].Hash)
	})

	t.Run("CommitLog_since-until", func(t *testing.T) {
		// since = far future → no commits
		future := time.Now().Add(24 * time.Hour)
		none, err := repo.CommitLog("main", "", 10, "", &future, nil)
		require.NoError(t, err)
		require.Empty(t, none, "since=future should return no commits")

		// until = zero time (long before any real commit) → no commits
		zero := time.Time{}
		none2, err := repo.CommitLog("main", "", 10, "", nil, &zero)
		require.NoError(t, err)
		require.Empty(t, none2, "until=zero should return no commits")

		// Both bounds open → all commits returned (filtering is a no-op)
		all, err := repo.CommitLog("main", "", 10, "", nil, nil)
		require.NoError(t, err)
		require.Len(t, all, 3, "nil since/until should return all commits")

		// since = far past and until = far future → all commits still returned
		past := time.Unix(0, 0)
		all2, err := repo.CommitLog("main", "", 10, "", &past, &future)
		require.NoError(t, err)
		require.Len(t, all2, 3, "wide since/until bounds should return all commits")
	})

	// ── LastCommitForEntries ──────────────────────────────────────────────────

	t.Run("LastCommitForEntries", func(t *testing.T) {
		result, err := repo.LastCommitForEntries("main", "", []string{"README.md", "main.go", "src"})
		require.NoError(t, err)

		// README.md was last changed in c3
		require.Equal(t, c3, result["README.md"].Hash)
		// main.go was last changed in c2
		require.Equal(t, c2, result["main.go"].Hash)
		// src tree changed in c2 (util.go added)
		require.Equal(t, c2, result["src"].Hash)

		// subdirectory: last commits for entries in src/
		srcResult, err := repo.LastCommitForEntries("main", "src", []string{"lib.go", "util.go"})
		require.NoError(t, err)
		// lib.go was added in c1 and never changed
		require.Equal(t, c1, srcResult["lib.go"].Hash)
		// util.go was added in c2
		require.Equal(t, c2, srcResult["util.go"].Hash)

		// requesting a name that doesn't exist returns no entry for it
		partial, err := repo.LastCommitForEntries("main", "", []string{"README.md", "ghost.txt"})
		require.NoError(t, err)
		require.Contains(t, partial, "README.md")
		require.NotContains(t, partial, "ghost.txt")
	})

	t.Run("LastCommitForEntries_cache-subset", func(t *testing.T) {
		// First call with one name — seeds (or hits) the cache for this directory.
		r1, err := repo.LastCommitForEntries("main", "", []string{"README.md"})
		require.NoError(t, err)
		require.Contains(t, r1, "README.md")
		require.Equal(t, c3, r1["README.md"].Hash)

		// Second call for the same directory but a different name.
		// A buggy implementation that caches only the requested subset would
		// return an empty map here (cache hit, but "main.go" was never stored).
		r2, err := repo.LastCommitForEntries("main", "", []string{"main.go"})
		require.NoError(t, err)
		require.Contains(t, r2, "main.go", "second call with different name should hit correct result, not empty cache")
		require.Equal(t, c2, r2["main.go"].Hash)

		// Third call requesting both names should also work.
		r3, err := repo.LastCommitForEntries("main", "", []string{"README.md", "main.go"})
		require.NoError(t, err)
		require.Equal(t, c3, r3["README.md"].Hash)
		require.Equal(t, c2, r3["main.go"].Hash)
	})

	t.Run("LastCommitForEntries_concurrent", func(t *testing.T) {
		// Use the "feature" ref so the cache is cold for this key.
		// This exercises the singleflight path.
		// The race detector will catch any data races in the cache or walk logic.
		const workers = 20
		type result struct {
			m   map[string]CommitMeta
			err error
		}
		results := make([]result, workers)
		var wg sync.WaitGroup
		wg.Add(workers)
		for i := range workers {
			go func() {
				defer wg.Done()
				m, err := repo.LastCommitForEntries("feature", "", []string{"README.md", "main.go", "src"})
				results[i] = result{m, err}
			}()
		}
		wg.Wait()
		for _, r := range results {
			require.NoError(t, r.err)
			require.Equal(t, c1, r.m["README.md"].Hash) // feature is at c2, README unchanged since c1
			require.Equal(t, c2, r.m["main.go"].Hash)
			require.Equal(t, c2, r.m["src"].Hash)
		}
	})

	// ── CommitDetail ──────────────────────────────────────────────────────────

	t.Run("CommitDetail", func(t *testing.T) {
		detail, err := repo.CommitDetail(c2)
		require.NoError(t, err)
		require.Equal(t, c2, detail.Hash)
		require.Equal(t, []Hash{c1}, detail.Parents)

		filesByPath := make(map[string]ChangedFile)
		for _, f := range detail.Files {
			filesByPath[f.Path] = f
		}
		require.Equal(t, ChangeStatusModified, filesByPath["main.go"].Status)
		require.Equal(t, ChangeStatusAdded, filesByPath["src/util.go"].Status)

		// initial commit: diffs against empty tree, everything is "added"
		initDetail, err := repo.CommitDetail(c1)
		require.NoError(t, err)
		for _, f := range initDetail.Files {
			require.Equal(t, ChangeStatusAdded, f.Status, "file %s", f.Path)
		}

		// unknown hash
		_, err = repo.CommitDetail(randomHash())
		require.ErrorIs(t, err, ErrNotFound)
	})

	// ── CommitFileDiff ────────────────────────────────────────────────────────

	t.Run("CommitFileDiff", func(t *testing.T) {
		fd, err := repo.CommitFileDiff(c2, "main.go")
		require.NoError(t, err)
		require.Equal(t, "main.go", fd.Path)
		require.False(t, fd.IsBinary)
		require.False(t, fd.IsNew)
		require.False(t, fd.IsDelete)
		require.NotEmpty(t, fd.Hunks)

		// find the added lines
		var addedContent []string
		for _, h := range fd.Hunks {
			for _, l := range h.Lines {
				if l.Type == DiffLineAdded {
					addedContent = append(addedContent, l.Content)
				}
			}
		}
		require.Contains(t, addedContent, "// updated")

		// new file in initial commit
		initFD, err := repo.CommitFileDiff(c1, "main.go")
		require.NoError(t, err)
		require.True(t, initFD.IsNew)
		require.Equal(t, "main.go", initFD.Path)

		// file not in this commit's diff
		_, err = repo.CommitFileDiff(c3, "main.go")
		require.ErrorIs(t, err, ErrNotFound)

		// unknown hash
		_, err = repo.CommitFileDiff(randomHash(), "main.go")
		require.ErrorIs(t, err, ErrNotFound)
	})
}
