package dag

import (
	"crypto/ed25519"
	"crypto/rand"
	mathrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func TestOperationPackReadWrite(t *testing.T) {
	repo, author, _, resolver, def := makeTestContext()

	opp := &operationPack{
		Author: author,
		Operations: []Operation{
			newOp1(author, "foo"),
			newOp2(author, "bar"),
		},
		CreateTime: 123,
		EditTime:   456,
	}

	commitHash, err := opp.Write(def, repo)
	require.NoError(t, err)

	commit, err := repo.ReadCommit(commitHash)
	require.NoError(t, err)

	opp2, err := readOperationPack(def, repo, resolver, commit)
	require.NoError(t, err)

	for _, op := range opp.Operations {
		// force the creation of the id
		op.Id()
	}
	require.Equal(t, opp, opp2)
}

func TestOperationPackSignedReadWrite(t *testing.T) {
	type makerFn func() (repository.ClockedRepo, identity.Interface, identity.Interface, entity.Resolvers, Definition)

	for _, maker := range []makerFn{
		makeTestContext,
		func() (repository.ClockedRepo, identity.Interface, identity.Interface, entity.Resolvers, Definition) {
			return makeTestContextGoGit(t)
		},
	} {
		repo, author, _, resolver, def := maker()

		testKey, err := newTestKey()
		require.NoError(t, err)

		err = author.(*identity.Identity).Mutate(repo, func(orig *identity.Mutator) {
			orig.Keys = append(orig.Keys, testKey)
		})
		require.NoError(t, err)

		opp := &operationPack{
			Author: author,
			Operations: []Operation{
				newOp1(author, "foo"),
				newOp2(author, "bar"),
			},
			CreateTime: 123,
			EditTime:   456,
		}

		commitHash, err := opp.Write(def, repo)
		require.NoError(t, err)

		commit, err := repo.ReadCommit(commitHash)
		require.NoError(t, err)

		opp2, err := readOperationPack(def, repo, resolver, commit)
		require.NoError(t, err)

		for _, op := range opp.Operations {
			// force the creation of the id
			op.Id()
		}
		require.Equal(t, opp, opp2)
	}
}

func TestOperationPackFiles(t *testing.T) {
	repo, author, _, resolver, def := makeTestContext()

	blobHash1, err := repo.StoreData(randomData())
	require.NoError(t, err)

	blobHash2, err := repo.StoreData(randomData())
	require.NoError(t, err)

	opp := &operationPack{
		Author: author,
		Operations: []Operation{
			newOp1(author, "foo", blobHash1, blobHash2),
			newOp1(author, "foo", blobHash2),
		},
		CreateTime: 123,
		EditTime:   456,
	}

	commitHash, err := opp.Write(def, repo)
	require.NoError(t, err)

	commit, err := repo.ReadCommit(commitHash)
	require.NoError(t, err)

	opp2, err := readOperationPack(def, repo, resolver, commit)
	require.NoError(t, err)

	for _, op := range opp.Operations {
		// force the creation of the id
		op.Id()
	}
	require.Equal(t, opp, opp2)

	require.ElementsMatch(t, opp2.Operations[0].(OperationWithFiles).GetFiles(), []repository.Hash{
		blobHash1,
		blobHash2,
	})
	require.ElementsMatch(t, opp2.Operations[1].(OperationWithFiles).GetFiles(), []repository.Hash{
		blobHash2,
	})

	tree, err := repo.ReadTree(commit.TreeHash)
	require.NoError(t, err)

	extraTreeHash, ok := repository.SearchTreeEntry(tree, extraEntryName)
	require.True(t, ok)

	extraTree, err := repo.ReadTree(extraTreeHash.Hash)
	require.NoError(t, err)
	require.ElementsMatch(t, extraTree, []repository.TreeEntry{
		{
			ObjectType: repository.Blob,
			Hash:       blobHash1,
			Name:       "file0",
		},
		{
			ObjectType: repository.Blob,
			Hash:       blobHash2,
			Name:       "file1",
		},
	})
}

func randomData() []byte {
	var letterRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterRunes[mathrand.Intn(len(letterRunes))]
	}
	return b
}

// newTestKey creates an in-memory Ed25519 signing key backed by an in-process SSH
// agent so no real SSH_AUTH_SOCK is required.
func newTestKey() (*identity.Key, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}
	ag := agent.NewKeyring().(agent.ExtendedAgent)
	if err := ag.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		return nil, err
	}
	signer := repository.NewSSHAgentSignerWithAgent(sshPub, ag)
	k, err := identity.NewSSHKey(sshPub)
	if err != nil {
		return nil, err
	}
	// NewKeyWithSigner injects a signer so Signer() works without a real SSH agent.
	return identity.NewKeyWithSigner(k.PublicKeyMultibase(), identity.KeyOriginSSH, signer), nil
}
