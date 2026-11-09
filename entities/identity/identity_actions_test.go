package identity

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestIdentityPushPull(t *testing.T) {
	repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)

	identity1, err := NewIdentity(repoA, "name1", "email1")
	require.NoError(t, err)
	err = identity1.Commit(t.Context(), repoA)
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
	err = identity2.Commit(t.Context(), repoB)
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
	err = identity1.Commit(t.Context(), repoA)
	require.NoError(t, err)

	err = identity2.Mutate(repoB, func(orig *Mutator) {
		orig.Name = "name2b"
		orig.Email = "email2b"
	})
	require.NoError(t, err)
	err = identity2.Commit(t.Context(), repoB)
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
	err = identity1.Commit(t.Context(), repoA)
	require.NoError(t, err)

	identity1B, err := Read(t.Context(), repoB, identity1.Id())
	require.NoError(t, err)

	err = identity1B.Mutate(repoB, func(orig *Mutator) {
		orig.Name = "name1concurrent"
		orig.Email = "name1concurrent"
	})
	require.NoError(t, err)
	err = identity1B.Commit(t.Context(), repoB)
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
		require.NoError(t, i.Commit(t.Context(), repoA))
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

	ids, err := ListLocalIds(t.Context(), repoB)
	require.NoError(t, err)
	require.Empty(t, ids)
}

// An identity that can't be merged doesn't hold back the others, and those
// pulled are reported as updated.
func TestIdentityMergeContinuesOnFailure(t *testing.T) {
	repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)

	var identities []*Identity
	for _, name := range []string{"name0", "name1", "name2", "name3"} {
		i, err := NewIdentity(repoA, name, "email")
		require.NoError(t, err)
		require.NoError(t, i.Commit(t.Context(), repoA))
		identities = append(identities, i)
	}
	_, err := Push(t.Context(), repoA, "origin")
	require.NoError(t, err)
	require.NoError(t, Pull(t.Context(), repoB, "origin"))

	// the first one diverges in B, the others are only updated in A
	diverged, err := Read(t.Context(), repoB, identities[0].Id())
	require.NoError(t, err)
	require.NoError(t, diverged.Mutate(repoB, func(m *Mutator) { m.Name = "concurrent" }))
	require.NoError(t, diverged.Commit(t.Context(), repoB))

	update := func(suffix string) {
		for _, i := range identities {
			require.NoError(t, i.Mutate(repoA, func(m *Mutator) { m.Name += suffix }))
			require.NoError(t, i.Commit(t.Context(), repoA))
		}
		_, err := Push(t.Context(), repoA, "origin")
		require.NoError(t, err)
	}

	update("b")
	_, err = Fetch(t.Context(), repoB, "origin")
	require.NoError(t, err)

	statuses := make(map[entity.Id]entity.MergeStatus)
	for result := range MergeAll(t.Context(), repoB, "origin") {
		require.NoError(t, result.Err)
		statuses[result.Id] = result.Status
	}
	require.Equal(t, map[entity.Id]entity.MergeStatus{
		identities[0].Id(): entity.MergeStatusInvalid,
		identities[1].Id(): entity.MergeStatusUpdated,
		identities[2].Id(): entity.MergeStatusUpdated,
		identities[3].Id(): entity.MergeStatusUpdated,
	}, statuses)

	// Pull fails on the diverged one, and merges the others all the same
	update("c")
	require.Error(t, Pull(t.Context(), repoB, "origin"))
	for _, i := range identities[1:] {
		read, err := Read(t.Context(), repoB, i.Id())
		require.NoError(t, err)
		require.Equal(t, i.Name(), read.Name())
	}
}

// failingUpdateRepo fails every UpdateRef with updateRef's error.
type failingUpdateRepo struct {
	*repotest.GoGitRepo
	updateRef func() error
}

func (r failingUpdateRepo) UpdateRef(ctx context.Context, namespace string, key string, old repository.Hash, commit repository.Hash) error {
	return r.updateRef()
}

// A merge that fails writing the ref is an error, not invalid remote data.
func TestIdentityMergeUpdateRefFailure(t *testing.T) {
	repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)

	var identities []*Identity
	for _, name := range []string{"name1", "name2"} {
		i, err := NewIdentity(repoA, name, "email")
		require.NoError(t, err)
		require.NoError(t, i.Commit(t.Context(), repoA))
		identities = append(identities, i)
	}
	_, err := Push(t.Context(), repoA, "origin")
	require.NoError(t, err)
	require.NoError(t, Pull(t.Context(), repoB, "origin"))

	// B now has to fast-forward both
	for _, i := range identities {
		require.NoError(t, i.Mutate(repoA, func(m *Mutator) { m.Name += "b" }))
		require.NoError(t, i.Commit(t.Context(), repoA))
	}
	_, err = Push(t.Context(), repoA, "origin")
	require.NoError(t, err)
	_, err = Fetch(t.Context(), repoB, "origin")
	require.NoError(t, err)

	errUpdate := errors.New("update failed")

	tests := []struct {
		name string
		// updateRef is the failure, given the cancel of the merge context
		updateRef func(cancel context.CancelFunc) error
		wantErr   error
		// wantCount is the number of results: all of them, or the first only
		// when cut short
		wantCount int
	}{
		{
			name:      "failure",
			updateRef: func(context.CancelFunc) error { return errUpdate },
			wantErr:   errUpdate,
			wantCount: len(identities),
		},
		{
			name: "canceled",
			updateRef: func(cancel context.CancelFunc) error {
				cancel()
				return context.Canceled
			},
			wantErr:   context.Canceled,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			repo := failingUpdateRepo{
				GoGitRepo: repoB,
				updateRef: func() error { return tt.updateRef(cancel) },
			}

			var results []entity.MergeResult
			for result := range MergeAll(ctx, repo, "origin") {
				results = append(results, result)
			}
			require.Len(t, results, tt.wantCount)
			for _, result := range results {
				require.Equal(t, entity.MergeStatusError, result.Status)
				require.ErrorIs(t, result.Err, tt.wantErr)
				require.NotEmpty(t, result.Id)
			}
		})
	}
}

// A reference of the namespace whose key can't hold an identity is ignored
// everywhere, and the key of an identity in the outdated repository format is an
// error everywhere.
func TestIdentityKeysNotEntities(t *testing.T) {
	// setup returns two repositories sharing a remote, the first one with an
	// identity and a reference with this key to a commit that isn't an identity,
	// both pushed and fetched by the second one
	setup := func(t *testing.T, key string) (repository.Repo, repository.Repo, *Identity) {
		repoA, repoB, _ := repotest.SetupGoGitReposAndRemote(t)

		i, err := NewIdentity(repoA, "name", "email")
		require.NoError(t, err)
		require.NoError(t, i.Commit(t.Context(), repoA))

		tree, err := repoA.StoreTree(t.Context(), nil)
		require.NoError(t, err)
		commit, err := repoA.StoreCommit(t.Context(), tree)
		require.NoError(t, err)
		require.NoError(t, repoA.UpdateRef(t.Context(), Namespace, key, "", commit))

		_, err = Push(t.Context(), repoA, "origin")
		require.NoError(t, err)
		_, err = Fetch(t.Context(), repoB, "origin")
		require.NoError(t, err)

		return repoA, repoB, i
	}

	mergeAll := func(t *testing.T, repo repository.Repo) map[entity.Id]entity.MergeStatus {
		statuses := make(map[entity.Id]entity.MergeStatus)
		for result := range MergeAll(t.Context(), repo, "origin") {
			require.NoError(t, result.Err)
			statuses[result.Id] = result.Status
		}
		return statuses
	}

	t.Run("ignored", func(t *testing.T) {
		const key = "not-an-id"
		repoA, repoB, i := setup(t, key)

		ids, err := ListLocalIds(t.Context(), repoA)
		require.NoError(t, err)
		require.Equal(t, []entity.Id{i.Id()}, ids)

		var read []*Identity
		for streamed := range ReadAll(t.Context(), repoA) {
			require.NoError(t, streamed.Err)
			require.EqualValues(t, 1, streamed.TotalEntities)
			read = append(read, streamed.Entity)
		}
		require.Len(t, read, 1)

		require.Equal(t, map[entity.Id]entity.MergeStatus{i.Id(): entity.MergeStatusNew}, mergeAll(t, repoB))
		_, err = repoB.ResolveRef(t.Context(), Namespace, key)
		require.ErrorIs(t, err, repository.ErrNotFound)
		require.NoError(t, Pull(t.Context(), repoB, "origin"))

		require.NoError(t, RemoveAll(t.Context(), repoA))
		ids, err = ListLocalIds(t.Context(), repoA)
		require.NoError(t, err)
		require.Empty(t, ids)
		_, err = repoA.ResolveRef(t.Context(), Namespace, key)
		require.NoError(t, err)
	})

	t.Run("outdated format", func(t *testing.T) {
		key := strings.Repeat("0", 40)
		repoA, repoB, i := setup(t, key)

		_, err := ListLocalIds(t.Context(), repoA)
		require.ErrorIs(t, err, entity.ErrOutdatedId)

		var readErr error
		for streamed := range ReadAll(t.Context(), repoA) {
			if streamed.Err != nil {
				readErr = streamed.Err
			}
		}
		require.ErrorIs(t, readErr, entity.ErrOutdatedId)

		require.Equal(t, map[entity.Id]entity.MergeStatus{
			i.Id():         entity.MergeStatusNew,
			entity.Id(key): entity.MergeStatusInvalid,
		}, mergeAll(t, repoB))
		_, err = repoB.ResolveRef(t.Context(), Namespace, key)
		require.ErrorIs(t, err, repository.ErrNotFound)
		require.Error(t, Pull(t.Context(), repoB, "origin"))

		// nothing is removed
		require.ErrorIs(t, RemoveAll(t.Context(), repoA), entity.ErrOutdatedId)
		_, err = repoA.ResolveRef(t.Context(), Namespace, i.Id().String())
		require.NoError(t, err)
	})
}
