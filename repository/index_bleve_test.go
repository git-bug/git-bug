package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBleveIndexes(t *testing.T) {
	const commit1 = Hash("1111111111111111111111111111111111111111")

	// newIndexes returns the bleve indexes in root, closed with the test.
	newIndexes := func(t *testing.T, root string) RepoIndex {
		t.Helper()
		indexes := NewBleveIndexes(root)
		t.Cleanup(func() { require.NoError(t, indexes.Close()) })
		return indexes
	}

	t.Run("created on disk", func(t *testing.T) {
		root := t.TempDir()
		indexes := newIndexes(t, root)

		// Can create indices
		indexA, err := indexes.GetIndex("a")
		require.NoError(t, err)
		require.NotZero(t, indexA)
		require.FileExists(t, filepath.Join(root, "a", "index_meta.json"))
		require.FileExists(t, filepath.Join(root, "a", "store", "root.bolt"))

		indexB, err := indexes.GetIndex("b")
		require.NoError(t, err)
		require.NotZero(t, indexB)
		require.DirExists(t, filepath.Join(root, "b"))

		// Can get an existing index
		indexA, err = indexes.GetIndex("a")
		require.NoError(t, err)
		require.NotZero(t, indexA)
	})

	t.Run("documents and their record survive a reopen", func(t *testing.T) {
		indexes := newIndexes(t, t.TempDir())

		idx, err := indexes.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))
		require.NoError(t, b.Apply())

		require.NoError(t, indexes.Close())

		idx, err = indexes.GetIndex("a")
		require.NoError(t, err)
		builtFrom, err := idx.BuiltFrom()
		require.NoError(t, err)
		require.Equal(t, map[string]Hash{"id1": commit1}, builtFrom)
		res, err := idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Equal(t, []string{"id1"}, res)
	})

	t.Run("search", func(t *testing.T) {
		indexes := newIndexes(t, t.TempDir())

		idx, err := indexes.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"the server crashes on login"}, commit1))
		require.NoError(t, b.Set("id2", []string{"login works, the server crashes later"}, commit1))
		require.NoError(t, b.Set("id3", []string{"-x +y title:z (abc"}, commit1))
		require.NoError(t, b.Set("id4", []string{"see file.go, and foo-bar"}, commit1))
		require.NoError(t, b.Set("id5", []string{"数据库连接失败"}, commit1))
		require.NoError(t, b.Set("id6", []string{"库存数据"}, commit1))
		require.NoError(t, b.Apply())

		requireSearch := func(t *testing.T, terms []string, expected ...string) {
			t.Helper()
			res, err := idx.Search(terms)
			require.NoError(t, err)
			require.ElementsMatch(t, expected, res)
		}

		// words are stemmed
		requireSearch(t, []string{"crash"}, "id1", "id2")
		// a term of several words is a phrase
		requireSearch(t, []string{"crashes on login"}, "id1")
		requireSearch(t, []string{"login crashes"})
		// so is a single word that splits into several
		requireSearch(t, []string{"foo-bar"}, "id4")
		requireSearch(t, []string{"bar-foo"})
		requireSearch(t, []string{"file.go"}, "id4")
		requireSearch(t, []string{"数据库"}, "id5")
		requireSearch(t, []string{"数据"}, "id5", "id6")
		// the caller's terms are left untouched
		terms := []string{"crashes on login"}
		requireSearch(t, terms, "id1")
		require.Equal(t, []string{"crashes on login"}, terms)
		// a stop word alone matches nothing, and doesn't prevent other matches
		requireSearch(t, []string{"the"})
		requireSearch(t, []string{"the", "later"}, "id2")
		// the query syntax of bleve is not interpreted
		requireSearch(t, []string{"-x"}, "id3")
		requireSearch(t, []string{"+y"}, "id3")
		requireSearch(t, []string{"title:z"}, "id3")
		requireSearch(t, []string{"(abc"}, "id3")
		requireSearch(t, []string{`"server`}, "id1", "id2")
	})

	t.Run("a document without text is recorded, not indexed", func(t *testing.T) {
		indexes := newIndexes(t, t.TempDir())

		idx, err := indexes.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))
		require.NoError(t, b.Set("id2", nil, commit1))
		require.NoError(t, b.Set("id3", []string{"", "  "}, commit1))
		require.NoError(t, b.Apply())

		builtFrom, err := idx.BuiltFrom()
		require.NoError(t, err)
		require.Equal(t, map[string]Hash{"id1": commit1, "id2": commit1, "id3": commit1}, builtFrom)
		count, err := idx.(*bleveIndex).index.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(1), count)

		// a document losing its text is dropped
		b = idx.NewBatch()
		require.NoError(t, b.Set("id1", nil, commit1))
		require.NoError(t, b.Apply())
		count, err = idx.(*bleveIndex).index.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(0), count)
		res, err := idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Empty(t, res)
	})

	t.Run("an index with another layout is replaced by an empty one", func(t *testing.T) {
		indexes := newIndexes(t, t.TempDir())

		idx, err := indexes.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))
		require.NoError(t, b.Apply())
		require.NoError(t, idx.(*bleveIndex).index.SetInternal(layoutVersionKey, []byte("0")))

		require.NoError(t, indexes.Close())

		idx, err = indexes.GetIndex("a")
		require.NoError(t, err)
		builtFrom, err := idx.BuiltFrom()
		require.NoError(t, err)
		require.Empty(t, builtFrom)
		res, err := idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Empty(t, res)

		// and the new one keeps its layout across a reopen
		b = idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))
		require.NoError(t, b.Apply())
		require.NoError(t, indexes.Close())
		idx, err = indexes.GetIndex("a")
		require.NoError(t, err)
		res, err = idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Equal(t, []string{"id1"}, res)
	})

	t.Run("a malformed record is rejected", func(t *testing.T) {
		indexes := newIndexes(t, t.TempDir())

		idx, err := indexes.GetIndex("a")
		require.NoError(t, err)
		require.NoError(t, idx.(*bleveIndex).index.SetInternal(builtFromKey, []byte("not a record")))

		_, err = idx.BuiltFrom()
		require.Error(t, err)
		require.Error(t, idx.NewBatch().Apply())
	})

	t.Run("a batch that fails leaves neither documents nor record", func(t *testing.T) {
		indexes := newIndexes(t, t.TempDir())

		idx, err := indexes.GetIndex("a")
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", []string{"marker"}, commit1))

		// the index is closed underneath the batch
		require.NoError(t, indexes.Close())
		require.Error(t, b.Apply())

		idx, err = indexes.GetIndex("a")
		require.NoError(t, err)
		builtFrom, err := idx.BuiltFrom()
		require.NoError(t, err)
		require.Empty(t, builtFrom)
		res, err := idx.Search([]string{"marker"})
		require.NoError(t, err)
		require.Empty(t, res)
	})

	t.Run("an index a Clear failed on still closes", func(t *testing.T) {
		root := t.TempDir()
		indexes := newIndexes(t, root)

		idx, err := indexes.GetIndex("a")
		require.NoError(t, err)

		// no new index can be made under a file
		file := filepath.Join(root, "file")
		require.NoError(t, os.WriteFile(file, nil, 0o644))
		idx.(*bleveIndex).path = filepath.Join(file, "a")
		require.Error(t, idx.Clear())

		_, err = idx.Search([]string{"marker"})
		require.Error(t, err)
		require.NoError(t, idx.Close())
	})
}
