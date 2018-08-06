package repository

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/util"
)

// mockRepoForTest defines an instance of Repo that can be used for testing.
type mockRepoForTest struct {
	blobs       map[util.Hash][]byte
	trees       map[util.Hash]string
	commits     map[util.Hash]commit
	refs        map[string]util.Hash
	createClock util.LamportClock
	editClock   util.LamportClock
}

type commit struct {
	treeHash util.Hash
	parent   util.Hash
}

func NewMockRepoForTest() Repo {
	return &mockRepoForTest{
		blobs:       make(map[util.Hash][]byte),
		trees:       make(map[util.Hash]string),
		commits:     make(map[util.Hash]commit),
		refs:        make(map[string]util.Hash),
		createClock: util.NewLamportClock(),
		editClock:   util.NewLamportClock(),
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
func (r *mockRepoForTest) PushRefs(remote string, refSpec string) error {
	return nil
}

func (r *mockRepoForTest) FetchRefs(remote string, refSpec string) error {
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
		return nil, fmt.Errorf("unknown hash")
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
		return fmt.Errorf("Unknown ref")
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

// ListIds will return a list of Git ref matching the given refspec,
// stripped to only the last part of the ref
func (r *mockRepoForTest) ListIds(refspec string) ([]string, error) {
	keys := make([]string, len(r.refs))

	i := 0
	for k := range r.refs {
		splitted := strings.Split(k, "/")
		keys[i] = splitted[len(splitted)-1]
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
			return nil, fmt.Errorf("unknown hash")
		}

		data, ok = r.trees[commit.treeHash]

		if !ok {
			return nil, fmt.Errorf("unknown hash")
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

func (r *mockRepoForTest) LoadClocks() error {
	return nil
}

func (r *mockRepoForTest) WriteClocks() error {
	return nil
}

func (r *mockRepoForTest) CreateTimeIncrement() (util.LamportTime, error) {
	return r.createClock.Increment(), nil
}

func (r *mockRepoForTest) EditTimeIncrement() (util.LamportTime, error) {
	return r.editClock.Increment(), nil
}

func (r *mockRepoForTest) CreateWitness(time util.LamportTime) error {
	r.createClock.Witness(time)
	return nil
}

func (r *mockRepoForTest) EditWitness(time util.LamportTime) error {
	r.editClock.Witness(time)
	return nil
}
