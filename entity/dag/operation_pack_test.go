package dag

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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

	commitHash, err := opp.Write(t.Context(), def, repo)
	require.NoError(t, err)

	commit, err := repo.ReadCommit(t.Context(), commitHash)
	require.NoError(t, err)

	opp2, err := readOperationPack(t.Context(), def, repo, resolver, commit)
	require.NoError(t, err)

	for _, op := range opp.Operations {
		// force the creation of the id
		op.Id()
		// read with the edit time of their pack
		op.setEditTime(opp.EditTime)
	}
	require.Equal(t, opp, opp2)

	t.Run("no operation", func(t *testing.T) {
		// a merge commit holds a pack without operation, written as an empty
		// array: a writer never writes null
		data, err := json.Marshal(&operationPack{Author: author, EditTime: 456})
		require.NoError(t, err)
		var raw map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &raw))
		require.Equal(t, `[]`, string(raw["ops"]))
	})
}

func TestOperationPackSignedReadWrite(t *testing.T) {
	type makerFn func() (repository.Repo, identity.Interface, identity.Interface, entity.Resolvers, Definition)

	for _, maker := range []makerFn{
		makeTestContext,
		func() (repository.Repo, identity.Interface, identity.Interface, entity.Resolvers, Definition) {
			return makeTestContextGoGit(t)
		},
	} {
		repo, author, _, resolver, def := maker()

		err := author.(*identity.Identity).Mutate(repo, func(orig *identity.Mutator) {
			orig.Keys = append(orig.Keys, identity.GenerateKey())
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

		commitHash, err := opp.Write(t.Context(), def, repo)
		require.NoError(t, err)

		commit, err := repo.ReadCommit(t.Context(), commitHash)
		require.NoError(t, err)

		opp2, err := readOperationPack(t.Context(), def, repo, resolver, commit)
		require.NoError(t, err)

		for _, op := range opp.Operations {
			// force the creation of the id
			op.Id()
			// read with the edit time of their pack
			op.setEditTime(opp.EditTime)
		}
		require.Equal(t, opp, opp2)
	}
}

func TestOperationPackFiles(t *testing.T) {
	repo, author, _, resolver, def := makeTestContext()

	blobHash1, err := repo.StoreData(t.Context(), randomData())
	require.NoError(t, err)

	blobHash2, err := repo.StoreData(t.Context(), randomData())
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

	commitHash, err := opp.Write(t.Context(), def, repo)
	require.NoError(t, err)

	commit, err := repo.ReadCommit(t.Context(), commitHash)
	require.NoError(t, err)

	opp2, err := readOperationPack(t.Context(), def, repo, resolver, commit)
	require.NoError(t, err)

	for _, op := range opp.Operations {
		// force the creation of the id
		op.Id()
		// read with the edit time of their pack
		op.setEditTime(opp.EditTime)
	}
	require.Equal(t, opp, opp2)

	require.ElementsMatch(t, opp2.Operations[0].(OperationWithFiles).GetFiles(), []repository.Hash{
		blobHash1,
		blobHash2,
	})
	require.ElementsMatch(t, opp2.Operations[1].(OperationWithFiles).GetFiles(), []repository.Hash{
		blobHash2,
	})

	tree, err := repo.ReadTree(t.Context(), commit.TreeHash)
	require.NoError(t, err)

	extraTreeHash, ok := repository.SearchTreeEntry(tree, extraEntryName)
	require.True(t, ok)

	extraTree, err := repo.ReadTree(t.Context(), extraTreeHash.Hash)
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
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return b
}

func TestUnmarshallPack(t *testing.T) {
	_, author, _, resolvers, def := makeTestContext()
	id := string(author.Id())

	// the pack rules of dag-entity.md §5 in the spec
	op := `{"type":1,"timestamp":1609459200,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAA","field_1":"foo"}`
	cases := []struct {
		name  string
		raw   string
		valid bool
	}{
		{"the reference: every rule followed", `{"author":{"id":"` + id + `"},"ops":[` + op + `]}`, true},
		{"a member of author that the reader does not know is ignored", `{"author":{"id":"` + id + `","name":"Alice"},"ops":[` + op + `]}`, true},
		{"a pack without author", `{"ops":[` + op + `]}`, false},
		{"an author that is null is absent", `{"author":null,"ops":[` + op + `]}`, false},
		{"an author without id", `{"author":{},"ops":[` + op + `]}`, false},
		{"an author id in uppercase", `{"author":{"id":"` + strings.ToUpper(id) + `"},"ops":[` + op + `]}`, false},
		{"an author id of 40 characters", `{"author":{"id":"` + id[:40] + `"},"ops":[` + op + `]}`, false},
		{"an element of ops without type", `{"author":{"id":"` + id + `"},"ops":[{"timestamp":1609459200,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAA"}]}`, false},
		{"a type that is null is absent", `{"author":{"id":"` + id + `"},"ops":[{"type":null,"timestamp":1609459200,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAA"}]}`, false},
		{"a type that is a string", `{"author":{"id":"` + id + `"},"ops":[{"type":"1","timestamp":1609459200,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAA"}]}`, false},
		{"a type with a fraction", `{"author":{"id":"` + id + `"},"ops":[{"type":1.0,"timestamp":1609459200,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAA"}]}`, false},
		{"an element of ops that is null", `{"author":{"id":"` + id + `"},"ops":[null]}`, false},
		{"an element of ops that is not an object", `{"author":{"id":"` + id + `"},"ops":[1]}`, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := unmarshallPack(t.Context(), def, resolvers, []byte(c.raw))
			if c.valid {
				require.NoError(t, err)
			} else {
				// rejected by the pack rules, not by the operation unmarshaler
				require.ErrorContains(t, err, "invalid operation pack")
			}
		})
	}
}
