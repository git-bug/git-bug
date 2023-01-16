package _select

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

func TestSelect(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	const typename = "foo"
	const namespace = "foos"

	resolve := func(args []string) (*cache.BugCache, []string, error) {
		return Resolve[*cache.BugCache](backend, typename, namespace, backend.Bugs(), args)
	}

	_, _, err = resolve([]string{})
	require.True(t, IsErrNoValidId(err))

	err = Select(backend, namespace, "invalid")
	require.NoError(t, err)

	// Resolve without a pattern should fail when no bug is selected
	_, _, err = resolve([]string{})
	require.Error(t, err)

	// generate a bunch of bugs

	rene, err := backend.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		_, _, err := backend.Bugs().NewRaw(rene, time.Now().Unix(), "title", "message", nil, nil)
		require.NoError(t, err)
	}

	// and two more for testing
	b1, _, err := backend.Bugs().NewRaw(rene, time.Now().Unix(), "title", "message", nil, nil)
	require.NoError(t, err)
	b2, _, err := backend.Bugs().NewRaw(rene, time.Now().Unix(), "title", "message", nil, nil)
	require.NoError(t, err)

	err = Select(backend, namespace, b1.Id())
	require.NoError(t, err)

	// normal select without args
	b3, _, err := resolve([]string{})
	require.NoError(t, err)
	require.Equal(t, b1.Id(), b3.Id())

	// override selection with same id
	b4, _, err := resolve([]string{b1.Id().String()})
	require.NoError(t, err)
	require.Equal(t, b1.Id(), b4.Id())

	// override selection with a prefix
	b5, _, err := resolve([]string{b1.Id().Human()})
	require.NoError(t, err)
	require.Equal(t, b1.Id(), b5.Id())

	// args that shouldn't override
	b6, _, err := resolve([]string{"arg"})
	require.NoError(t, err)
	require.Equal(t, b1.Id(), b6.Id())

	// override with a different id
	b7, _, err := resolve([]string{b2.Id().String()})
	require.NoError(t, err)
	require.Equal(t, b2.Id(), b7.Id())

	err = Clear(backend, namespace)
	require.NoError(t, err)

	// Resolve without a pattern should error again after clearing the selected bug
	_, _, err = resolve([]string{})
	require.Error(t, err)
}
