package repository

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/99designs/keyring"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var _ ClockedRepo = &mockRepoForTest{}
var _ TestedRepo = &mockRepoForTest{}

// mockRepoForTest defines an instance of Repo that can be used for testing.
type mockRepoForTest struct {
	config       *MemConfig
	globalConfig *MemConfig
	keyring      *keyring.ArrayKeyring
	blobs        map[Hash][]byte
	trees        map[Hash]string
	commits      map[Hash]commit
	refs         map[string]Hash
	clocks       map[string]lamport.Clock
}

type commit struct {
	treeHash Hash
	parent   Hash
}

func NewMockRepoForTest() *mockRepoForTest {
	return &mockRepoForTest{
		config:       NewMemConfig(),
		globalConfig: NewMemConfig(),
		keyring:      keyring.NewArrayKeyring(nil),
		blobs:        make(map[Hash][]byte),
		trees:        make(map[Hash]string),
		commits:      make(map[Hash]commit),
		refs:         make(map[string]Hash),
		clocks:       make(map[string]lamport.Clock),
	}
}

// LocalConfig give access to the repository scoped configuration
func (r *mockRepoForTest) LocalConfig() Config {
	return r.config
}

// GlobalConfig give access to the git global configuration
func (r *mockRepoForTest) GlobalConfig() Config {
	return r.globalConfig
}

// Keyring give access to a user-wide storage for secrets
func (r *mockRepoForTest) Keyring() Keyring {
	return r.keyring
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

// GetRemotes returns the configured remotes repositories.
func (r *mockRepoForTest) GetRemotes() (map[string]string, error) {
	return map[string]string{
		"origin": "git://github.com/MichaelMure/git-bug",
	}, nil
}

// PushRefs push git refs to a remote
func (r *mockRepoForTest) PushRefs(remote string, refSpec string) (string, error) {
	return "", nil
}

func (r *mockRepoForTest) FetchRefs(remote string, refSpec string) (string, error) {
	return "", nil
}

func (r *mockRepoForTest) StoreData(data []byte) (Hash, error) {
	rawHash := sha1.Sum(data)
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.blobs[hash] = data
	return hash, nil
}

func (r *mockRepoForTest) ReadData(hash Hash) ([]byte, error) {
	data, ok := r.blobs[hash]

	if !ok {
		return nil, fmt.Errorf("unknown hash")
	}

	return data, nil
}

func (r *mockRepoForTest) StoreTree(entries []TreeEntry) (Hash, error) {
	buffer := prepareTreeEntries(entries)
	rawHash := sha1.Sum(buffer.Bytes())
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.trees[hash] = buffer.String()

	return hash, nil
}

func (r *mockRepoForTest) StoreCommit(treeHash Hash) (Hash, error) {
	rawHash := sha1.Sum([]byte(treeHash))
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.commits[hash] = commit{
		treeHash: treeHash,
	}
	return hash, nil
}

func (r *mockRepoForTest) StoreCommitWithParent(treeHash Hash, parent Hash) (Hash, error) {
	rawHash := sha1.Sum([]byte(treeHash + parent))
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.commits[hash] = commit{
		treeHash: treeHash,
		parent:   parent,
	}
	return hash, nil
}

func (r *mockRepoForTest) UpdateRef(ref string, hash Hash) error {
	r.refs[ref] = hash
	return nil
}

func (r *mockRepoForTest) RemoveRef(ref string) error {
	delete(r.refs, ref)
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

func (r *mockRepoForTest) ListRefs(refPrefix string) ([]string, error) {
	var keys []string

	for k := range r.refs {
		if strings.HasPrefix(k, refPrefix) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (r *mockRepoForTest) ListCommits(ref string) ([]Hash, error) {
	var hashes []Hash

	hash := r.refs[ref]

	for {
		commit, ok := r.commits[hash]

		if !ok {
			break
		}

		hashes = append([]Hash{hash}, hashes...)
		hash = commit.parent
	}

	return hashes, nil
}

func (r *mockRepoForTest) ReadTree(hash Hash) ([]TreeEntry, error) {
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

func (r *mockRepoForTest) FindCommonAncestor(hash1 Hash, hash2 Hash) (Hash, error) {
	ancestor1 := []Hash{hash1}

	for hash1 != "" {
		c, ok := r.commits[hash1]
		if !ok {
			return "", fmt.Errorf("unknown commit %v", hash1)
		}
		ancestor1 = append(ancestor1, c.parent)
		hash1 = c.parent
	}

	for {
		for _, ancestor := range ancestor1 {
			if ancestor == hash2 {
				return ancestor, nil
			}
		}

		c, ok := r.commits[hash2]
		if !ok {
			return "", fmt.Errorf("unknown commit %v", hash1)
		}

		if c.parent == "" {
			return "", fmt.Errorf("no ancestor found")
		}

		hash2 = c.parent
	}
}

func (r *mockRepoForTest) GetTreeHash(commit Hash) (Hash, error) {
	c, ok := r.commits[commit]
	if !ok {
		return "", fmt.Errorf("unknown commit")
	}

	return c.treeHash, nil
}

func (r *mockRepoForTest) GetOrCreateClock(name string) (lamport.Clock, error) {
	if c, ok := r.clocks[name]; ok {
		return c, nil
	}

	c := lamport.NewMemClock()
	r.clocks[name] = c
	return c, nil
}

func (r *mockRepoForTest) AddRemote(name string, url string) error {
	panic("implement me")
}
