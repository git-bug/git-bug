package repository

import (
	"crypto/sha1"
	"fmt"
	"github.com/MichaelMure/git-bug/util"
	"github.com/pkg/errors"
)

// mockRepoForTest defines an instance of Repo that can be used for testing.
type mockRepoForTest struct {
	blobs   map[util.Hash][]byte
	trees   map[util.Hash]string
	commits map[util.Hash]commit
	refs    map[string]util.Hash
}

type commit struct {
	treeHash util.Hash
	parent   util.Hash
}

func NewMockRepoForTest() Repo {
	return &mockRepoForTest{
		blobs:   make(map[util.Hash][]byte),
		trees:   make(map[util.Hash]string),
		commits: make(map[util.Hash]commit),
		refs:    make(map[string]util.Hash),
	}
}

// GetPath returns the path to the repo.
func (r *mockRepoForTest) GetPath() string {
	return "~/mockRepo/"
}

func (r *mockRepoForTest) GetUserName() (string, error) {
	return "René Descartes", nil
}

// GetUserEmail returns the email address that the user has used to configure git.
func (r *mockRepoForTest) GetUserEmail() (string, error) {
	return "user@example.com", nil
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (r *mockRepoForTest) GetCoreEditor() (string, error) {
	return "vi", nil
}

// PushRefs push git refs to a remote
func (r *mockRepoForTest) PushRefs(remote string, refPattern string) error {
	return nil
}

func (r *mockRepoForTest) FetchRefs(remote string, refPattern string, remoteRefPattern string) error {
	return nil
}

func (r *mockRepoForTest) StoreData(data []byte) (util.Hash, error) {
	rawHash := sha1.Sum(data)
	hash := util.Hash(fmt.Sprintf("%x", rawHash))
	r.blobs[hash] = data
	return hash, nil
}

func (r *mockRepoForTest) ReadData(hash util.Hash) ([]byte, error) {
	data, ok := r.blobs[hash]

	if !ok {
		return nil, errors.New("unknown hash")
	}

	return data, nil
}

func (r *mockRepoForTest) StoreTree(entries []TreeEntry) (util.Hash, error) {
	buffer := prepareTreeEntries(entries)
	rawHash := sha1.Sum(buffer.Bytes())
	hash := util.Hash(fmt.Sprintf("%x", rawHash))
	r.trees[hash] = buffer.String()

	return hash, nil
}

func (r *mockRepoForTest) StoreCommit(treeHash util.Hash) (util.Hash, error) {
	rawHash := sha1.Sum([]byte(treeHash))
	hash := util.Hash(fmt.Sprintf("%x", rawHash))
	r.commits[hash] = commit{
		treeHash: treeHash,
	}
	return hash, nil
}

func (r *mockRepoForTest) StoreCommitWithParent(treeHash util.Hash, parent util.Hash) (util.Hash, error) {
	rawHash := sha1.Sum([]byte(treeHash + parent))
	hash := util.Hash(fmt.Sprintf("%x", rawHash))
	r.commits[hash] = commit{
		treeHash: treeHash,
		parent:   parent,
	}
	return hash, nil
}

func (r *mockRepoForTest) UpdateRef(ref string, hash util.Hash) error {
	r.refs[ref] = hash
	return nil
}

func (r *mockRepoForTest) RefExist(ref string) (bool, error) {
	_, exist := r.refs[ref]
	return exist, nil
}

func (r *mockRepoForTest) CopyRef(source string, dest string) error {
	hash, exist := r.refs[source]

	if !exist {
		return errors.New("Unknown ref")
	}

	r.refs[dest] = hash
	return nil
}

func (r *mockRepoForTest) ListRefs(refspec string) ([]string, error) {
	keys := make([]string, len(r.refs))

	i := 0
	for k := range r.refs {
		keys[i] = k
		i++
	}

	return keys, nil
}

func (r *mockRepoForTest) ListCommits(ref string) ([]util.Hash, error) {
	var hashes []util.Hash

	hash := r.refs[ref]

	for {
		commit, ok := r.commits[hash]

		if !ok {
			break
		}

		hashes = append([]util.Hash{hash}, hashes...)
		hash = commit.parent
	}

	return hashes, nil
}

func (r *mockRepoForTest) ListEntries(hash util.Hash) ([]TreeEntry, error) {
	var data string

	data, ok := r.trees[hash]

	if !ok {
		// Git will understand a commit hash to reach a tree
		commit, ok := r.commits[hash]

		if !ok {
			return nil, errors.New("unknown hash")
		}

		data, ok = r.trees[commit.treeHash]

		if !ok {
			return nil, errors.New("unknown hash")
		}
	}

	return readTreeEntries(data)
}

func (r *mockRepoForTest) FindCommonAncestor(hash1 util.Hash, hash2 util.Hash) (util.Hash, error) {
	panic("implement me")
}

func (r *mockRepoForTest) GetTreeHash(commit util.Hash) (util.Hash, error) {
	panic("implement me")
}
