package repository

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSqliteIndex(t *testing.T) {
	const testCommit = Hash("1111111111111111111111111111111111111111")

	excerpt := []byte(`{}`)

	newIndex := func(t *testing.T, path string) RepoIndex {
		t.Helper()
		repoIndex := NewSqliteIndex(path)
		t.Cleanup(func() { require.NoError(t, repoIndex.Close()) })
		return repoIndex
	}

	t.Run("search", func(t *testing.T) {
		idx, err := newIndex(t, ":memory:").GetIndex("a", 1)
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", testCommit, excerpt, []string{"the server crashes on login"}))
		require.NoError(t, b.Set("id2", testCommit, excerpt, []string{"login works, the server crashes later"}))
		require.NoError(t, b.Set("id3", testCommit, excerpt, []string{"-x +y title:z (abc"}))
		require.NoError(t, b.Set("id4", testCommit, excerpt, []string{"see file.go, and foo-bar"}))
		require.NoError(t, b.Set("id5", testCommit, excerpt, []string{"数据库连接失败"}))
		require.NoError(t, b.Set("id6", testCommit, excerpt, []string{"库存数据"}))
		require.NoError(t, b.Set("id7", testCommit, excerpt, []string{"first text ends with words", "_ second starts"}))
		require.NoError(t, b.Apply(t.Context()))

		tests := []struct {
			name     string
			phrases  []string
			expected []string
		}{
			{"no phrase", nil, nil},
			{"words are stemmed", []string{"crash"}, []string{"id1", "id2"}},
			{"a phrase", []string{"crashes on login"}, []string{"id1"}},
			{"a phrase in another order", []string{"login crashes"}, nil},
			{"any of the phrases", []string{"later", "foo"}, []string{"id2", "id4"}},
			{"a word that splits is a phrase", []string{"foo-bar"}, []string{"id4"}},
			{"a word that splits, in another order", []string{"bar-foo"}, nil},
			{"a file name", []string{"file.go"}, []string{"id4"}},
			{"a phrase doesn't span two texts", []string{"words second"}, nil},
			{"nor match their separator", []string{"words _ second"}, nil},
			{"a CJK word", []string{"数据库"}, []string{"id5"}},
			{"a shorter CJK word", []string{"数据"}, []string{"id5", "id6"}},
			{"a stop word alone", []string{"the"}, nil},
			{"a stop word and another phrase", []string{"the", "later"}, []string{"id2"}},
			{"no syntax: -", []string{"-x"}, []string{"id3"}},
			{"no syntax: +", []string{"+y"}, []string{"id3"}},
			{"no syntax: column", []string{"title:z"}, []string{"id3"}},
			{"no syntax: (", []string{"(abc"}, []string{"id3"}},
			{`no syntax: "`, []string{`"server`}, []string{"id1", "id2"}},
			{"no syntax: *", []string{"serv*"}, nil},
			{"no syntax: NEAR", []string{"NEAR(server login)"}, nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				res, err := idx.Search(t.Context(), tt.phrases)
				require.NoError(t, err)
				require.ElementsMatch(t, tt.expected, res)
			})
		}
	})

	t.Run("documents survive a reopen", func(t *testing.T) {
		repoIndex := newIndex(t, filepath.Join(t.TempDir(), "index.db"))

		idx, err := repoIndex.GetIndex("a", 1)
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", testCommit, []byte(`{"n":1}`), []string{"marker"}))
		require.NoError(t, b.Apply(t.Context()))
		require.NoError(t, repoIndex.Close())

		idx, err = repoIndex.GetIndex("a", 1)
		require.NoError(t, err)
		builtFrom, err := idx.AllBuiltFrom(t.Context())
		require.NoError(t, err)
		require.Equal(t, map[string]Hash{"id1": testCommit}, builtFrom)
		commit, got, err := idx.Excerpt(t.Context(), "id1")
		require.NoError(t, err)
		require.Equal(t, testCommit, commit)
		require.JSONEq(t, `{"n":1}`, string(got))
		res, err := idx.Search(t.Context(), []string{"marker"})
		require.NoError(t, err)
		require.Equal(t, []string{"id1"}, res)
	})

	t.Run("a file that isn't a database fails with what to do", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "index.db")
		garbage := bytes.Repeat([]byte("x"), 4096)
		require.NoError(t, os.WriteFile(path, garbage, 0o644))

		_, err := newIndex(t, path).GetIndex("a", 1)
		require.ErrorContains(t, err, path+" is damaged, delete it")

		// left as it is
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, garbage, content)
	})

	t.Run("a document deleted by hand takes its tokens along", func(t *testing.T) {
		idx, err := newIndex(t, ":memory:").GetIndex("a", 1)
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", testCommit, excerpt, []string{"marker"}))
		require.NoError(t, b.Apply(t.Context()))

		_, err = idx.(*sqliteIndex).db.Exec("DELETE FROM docs WHERE id = 'id1'")
		require.NoError(t, err)

		// the next document gets the num of the deleted one, not its tokens
		b = idx.NewBatch()
		require.NoError(t, b.Set("id2", testCommit, excerpt, nil))
		require.NoError(t, b.Apply(t.Context()))
		res, err := idx.Search(t.Context(), []string{"marker"})
		require.NoError(t, err)
		require.Empty(t, res)
	})

	t.Run("a path with URI characters", func(t *testing.T) {
		dir := "a?b#c%41"
		if runtime.GOOS == "windows" {
			// Windows doesn't allow ? in a file name
			dir = "a#c%41"
		}
		path := filepath.Join(t.TempDir(), dir, "index.db")
		idx, err := newIndex(t, path).GetIndex("a", 1)
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", testCommit, excerpt, nil))
		require.NoError(t, b.Apply(t.Context()))
		require.FileExists(t, path)
	})

	t.Run("an excerpt that isn't an object is refused", func(t *testing.T) {
		idx, err := newIndex(t, ":memory:").GetIndex("a", 1)
		require.NoError(t, err)
		for _, invalid := range []string{``, `null`, `[]`, `"text"`, `{`} {
			b := idx.NewBatch()
			require.NoError(t, b.Set("valid", testCommit, excerpt, []string{"marker"}))
			require.NoError(t, b.Set("id1", testCommit, []byte(invalid), nil))
			require.Error(t, b.Apply(t.Context()), invalid)
		}

		// nothing of the batches landed
		builtFrom, err := idx.AllBuiltFrom(t.Context())
		require.NoError(t, err)
		require.Empty(t, builtFrom)
		res, err := idx.Search(t.Context(), []string{"marker"})
		require.NoError(t, err)
		require.Empty(t, res)
	})

	t.Run("a batch failing leaves nothing", func(t *testing.T) {
		repoIndex := newIndex(t, filepath.Join(t.TempDir(), "index.db"))

		idx, err := repoIndex.GetIndex("a", 1)
		require.NoError(t, err)
		b := idx.NewBatch()
		require.NoError(t, b.Set("id1", testCommit, excerpt, []string{"marker"}))
		// the database is closed underneath the batch
		require.NoError(t, repoIndex.Close())
		require.Error(t, b.Apply(t.Context()))

		idx, err = repoIndex.GetIndex("a", 1)
		require.NoError(t, err)
		builtFrom, err := idx.AllBuiltFrom(t.Context())
		require.NoError(t, err)
		require.Empty(t, builtFrom)
	})

	t.Run("two connections write at the same time", func(t *testing.T) {
		const docs = 50
		write := func(repoIndex RepoIndex, prefix string) error {
			idx, err := repoIndex.GetIndex("a", 1)
			if err != nil {
				return err
			}
			for i := 0; i < docs; i++ {
				b := idx.NewBatch()
				err = b.Set(fmt.Sprintf("%s%d", prefix, i), testCommit, excerpt, []string{"marker"})
				if err != nil {
					return err
				}
				err = b.Apply(t.Context())
				if err != nil {
					return err
				}
			}
			return nil
		}

		// each has a database of its own on the same file, locking each other
		// as two processes would
		path := filepath.Join(t.TempDir(), "index.db")
		first, second := newIndex(t, path), newIndex(t, path)
		idx, err := first.GetIndex("a", 1)
		require.NoError(t, err)

		// the first holds the write lock while the second starts, for the second
		// to wait on it rather than fail
		tx, err := idx.(*sqliteIndex).db.Begin()
		require.NoError(t, err)
		// on failure, releases the lock and lets the second finish on its own
		defer tx.Rollback()
		done := make(chan error, 1)
		go func() { done <- write(second, "second") }()
		require.NoError(t, tx.Commit())
		require.NoError(t, write(first, "first"))
		require.NoError(t, <-done)

		// both see everything, the index of the first, open before the second
		// wrote, included
		for _, repoIndex := range []RepoIndex{first, second} {
			idx, err := repoIndex.GetIndex("a", 1)
			require.NoError(t, err)
			res, err := idx.Search(t.Context(), []string{"marker"})
			require.NoError(t, err)
			require.Len(t, res, 2*docs)
			builtFrom, err := idx.AllBuiltFrom(t.Context())
			require.NoError(t, err)
			require.Len(t, builtFrom, 2*docs)
		}
	})
}
