package identity

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/repository"
)

func TestPushPull(t *testing.T) {
	repoA, repoB, remote := repository.SetupReposAndRemote(t)
	defer repository.CleanupTestRepos(t, repoA, repoB, remote)

	identity1 := NewIdentity("name1", "email1")
	err := identity1.Commit(repoA)
	require.NoError(t, err)

	// A --> remote --> B
	_, err = Push(repoA, "origin")
	require.NoError(t, err)

	err = Pull(repoB, "origin")
	require.NoError(t, err)

	identities := allIdentities(t, ReadAllLocalIdentities(repoB))

	if len(identities) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A
	identity2 := NewIdentity("name2", "email2")
	err = identity2.Commit(repoB)
	require.NoError(t, err)

	_, err = Push(repoB, "origin")
	require.NoError(t, err)

	err = Pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// Update both

	identity1.addVersionForTest(&Version{
		name:  "name1b",
		email: "email1b",
	})
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	identity2.addVersionForTest(&Version{
		name:  "name2b",
		email: "email2b",
	})
	err = identity2.Commit(repoB)
	require.NoError(t, err)

	//  A --> remote --> B

	_, err = Push(repoA, "origin")
	require.NoError(t, err)

	err = Pull(repoB, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoB))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A

	_, err = Push(repoB, "origin")
	require.NoError(t, err)

	err = Pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// Concurrent update

	identity1.addVersionForTest(&Version{
		name:  "name1c",
		email: "email1c",
	})
	err = identity1.Commit(repoA)
	require.NoError(t, err)

	identity1B, err := ReadLocal(repoB, identity1.Id())
	require.NoError(t, err)

	identity1B.addVersionForTest(&Version{
		name:  "name1concurrent",
		email: "email1concurrent",
	})
	err = identity1B.Commit(repoB)
	require.NoError(t, err)

	//  A --> remote --> B

	_, err = Push(repoA, "origin")
	require.NoError(t, err)

	// Pulling a non-fast-forward update should fail
	err = Pull(repoB, "origin")
	require.Error(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoB))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A

	// Pushing a non-fast-forward update should fail
	_, err = Push(repoB, "origin")
	require.Error(t, err)

	err = Pull(repoA, "origin")
	require.NoError(t, err)

	identities = allIdentities(t, ReadAllLocalIdentities(repoA))

	if len(identities) != 2 {
		t.Fatal("Unexpected number of bugs")
	}
}

func allIdentities(t testing.TB, identities <-chan StreamedIdentity) []*Identity {
	var result []*Identity
	for streamed := range identities {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Identity)
	}
	return result
}
