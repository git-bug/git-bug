package identity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestIdentityPushPull(t *testing.T) {
	repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)

	identity1, err := NewIdentity(repoA, "name1", "email1")
	require.NoError(t, err)
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	// A --> remote --> B
	_, err = Push(t.Context(), repoA, "origin")
	require.NoError(t, err)

	err = Pull(t.Context(), repoB, "origin")
	require.NoError(t, err)

	identities := allIdentities(t, ReadAll(t.Context(), repoB))

	if len(identities) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A
	identity2, err := NewIdentity(repoB, "name2", "email2")
	require.NoError(t, err)
	err = identity2.Commit(repoB)
	require.NoError(t, err)

	_, err = Push(t.Context(), repoB, "origin")
	require.NoError(t, err)

	err = Pull(t.Context(), repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAll(t.Context(), repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// Update both

	err = identity1.Mutate(repoA, func(orig *Mutator) {
		orig.Name = "name1b"
		orig.Email = "email1b"
	})
	require.NoError(t, err)
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	err = identity2.Mutate(repoB, func(orig *Mutator) {
		orig.Name = "name2b"
		orig.Email = "email2b"
	})
	require.NoError(t, err)
	err = identity2.Commit(repoB)
	require.NoError(t, err)

	//  A --> remote --> B

	_, err = Push(t.Context(), repoA, "origin")
	require.NoError(t, err)

	err = Pull(t.Context(), repoB, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAll(t.Context(), repoB))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A

	_, err = Push(t.Context(), repoB, "origin")
	require.NoError(t, err)

	err = Pull(t.Context(), repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAll(t.Context(), repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// Concurrent update

	err = identity1.Mutate(repoA, func(orig *Mutator) {
		orig.Name = "name1c"
		orig.Email = "email1c"
	})
	require.NoError(t, err)
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	identity1B, err := Read(t.Context(), repoB, identity1.Id())
	require.NoError(t, err)

	err = identity1B.Mutate(repoB, func(orig *Mutator) {
		orig.Name = "name1concurrent"
		orig.Email = "name1concurrent"
	})
	require.NoError(t, err)
	err = identity1B.Commit(repoB)
	require.NoError(t, err)

	//  A --> remote --> B

	_, err = Push(t.Context(), repoA, "origin")
	require.NoError(t, err)

	// Pulling a non-fast-forward update should fail
	err = Pull(t.Context(), repoB, "origin")
	require.Error(t, err)

	identities = allIdentities(t, ReadAll(t.Context(), repoB))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A

	// Pushing a non-fast-forward update should fail
	_, err = Push(t.Context(), repoB, "origin")
	require.Error(t, err)

	err = Pull(t.Context(), repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAll(t.Context(), repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}
}

func allIdentities(t testing.TB, identities <-chan entity.StreamedEntity[*Identity]) []*Identity {
	var result []*Identity
	for streamed := range identities {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Entity)
	}
	return result
}

// A merge cut short is an error, not invalid remote data, and nothing more is
// merged.
func TestIdentityMergeCanceled(t *testing.T) {
	repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)

	for _, name := range []string{"name1", "name2"} {
		i, err := NewIdentity(repoA, name, "email")
		require.NoError(t, err)
		require.NoError(t, i.Commit(repoA))
	}

	_, err := Push(t.Context(), repoA, "origin")
	require.NoError(t, err)
	_, err = Fetch(t.Context(), repoB, "origin")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var results []entity.MergeResult
	for result := range MergeAll(ctx, repoB, "origin") {
		results = append(results, result)
	}
	require.Len(t, results, 1)
	require.Equal(t, entity.MergeStatusError, results[0].Status)
	require.ErrorIs(t, results[0].Err, context.Canceled)

	ids, err := ListLocalIds(repoB)
	require.NoError(t, err)
	require.Empty(t, ids)
}
