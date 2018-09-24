package repository

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/lamport"
)

// mockRepoForTest defines an instance of Repo that can be used for testing.
type mockRepoForTest struct {
	config      map[string]string
	blobs       map[git.Hash][]byte
	trees       map[git.Hash]string
	commits     map[git.Hash]commit
	refs        map[string]git.Hash
	createClock lamport.Clock
	editClock   lamport.Clock
}

type commit struct {
	treeHash git.Hash
	parent   git.Hash
}

func NewMockRepoForTest() *mockRepoForTest {
	return &mockRepoForTest{
		config:      make(map[string]string),
		blobs:       make(map[git.Hash][]byte),
		trees:       make(map[git.Hash]string),
		commits:     make(map[git.Hash]commit),
		refs:        make(map[string]git.Hash),
		createClock: lamport.NewClock(),
		editClock:   lamport.NewClock(),
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

func (r *mockRepoForTest) StoreConfig(key string, value string) error {
	r.config[key] = value
	return nil
}

func (r *mockRepoForTest) ReadConfigs(keyPrefix string) (map[string]string, error) {
	result := make(map[string]string)

	for key, val := range r.config {
		if strings.HasPrefix(key, keyPrefix) {
			result[key] = val
		}
	}

	return result, nil
}

// PushRefs push git refs to a remote
func (r *mockRepoForTest) PushRefs(remote string, refSpec string) (string, error) {
	return "", nil
}

func (r *mockRepoForTest) FetchRefs(remote string, refSpec string) (string, error) {
	return "", nil
}

func (r *mockRepoForTest) StoreData(data []byte) (git.Hash, error) {
	rawHash := sha1.Sum(data)
	hash := git.Hash(fmt.Sprintf("%x", rawHash))
	r.blobs[hash] = data
	return hash, nil
}

func (r *mockRepoForTest) ReadData(hash git.Hash) ([]byte, error) {
	data, ok := r.blobs[hash]

	if !ok {
		return nil, fmt.Errorf("unknown hash")
	}

	return data, nil
}

func (r *mockRepoForTest) StoreTree(entries []TreeEntry) (git.Hash, error) {
	buffer := prepareTreeEntries(entries)
	rawHash := sha1.Sum(buffer.Bytes())
	hash := git.Hash(fmt.Sprintf("%x", rawHash))
	r.trees[hash] = buffer.String()

	return hash, nil
}

func (r *mockRepoForTest) StoreCommit(treeHash git.Hash) (git.Hash, error) {
	rawHash := sha1.Sum([]byte(treeHash))
	hash := git.Hash(fmt.Sprintf("%x", rawHash))
	r.commits[hash] = commit{
		treeHash: treeHash,
	}
	return hash, nil
}

func (r *mockRepoForTest) StoreCommitWithParent(treeHash git.Hash, parent git.Hash) (git.Hash, error) {
	rawHash := sha1.Sum([]byte(treeHash + parent))
	hash := git.Hash(fmt.Sprintf("%x", rawHash))
	r.commits[hash] = commit{
		treeHash: treeHash,
		parent:   parent,
	}
	return hash, nil
}

func (r *mockRepoForTest) UpdateRef(ref string, hash git.Hash) error {
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

func (r *mockRepoForTest) ListCommits(ref string) ([]git.Hash, error) {
	var hashes []git.Hash

	hash := r.refs[ref]

	for {
		commit, ok := r.commits[hash]

		if !ok {
			break
		}

		hashes = append([]git.Hash{hash}, hashes...)
		hash = commit.parent
	}

	return hashes, nil
}

func (r *mockRepoForTest) ListEntries(hash git.Hash) ([]TreeEntry, error) {
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

func (r *mockRepoForTest) FindCommonAncestor(hash1 git.Hash, hash2 git.Hash) (git.Hash, error) {
	panic("implement me")
}

func (r *mockRepoForTest) GetTreeHash(commit git.Hash) (git.Hash, error) {
	panic("implement me")
}

func (r *mockRepoForTest) LoadClocks() error {
	return nil
}

func (r *mockRepoForTest) WriteClocks() error {
	return nil
}

func (r *mockRepoForTest) CreateTimeIncrement() (lamport.Time, error) {
	return r.createClock.Increment(), nil
}

func (r *mockRepoForTest) EditTimeIncrement() (lamport.Time, error) {
	return r.editClock.Increment(), nil
}

func (r *mockRepoForTest) CreateWitness(time lamport.Time) error {
	r.createClock.Witness(time)
	return nil
}

func (r *mockRepoForTest) EditWitness(time lamport.Time) error {
	r.editClock.Witness(time)
	return nil
}
