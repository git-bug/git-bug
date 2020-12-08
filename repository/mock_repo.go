package repository

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"sync"

	"github.com/99designs/keyring"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"

	"github.com/MichaelMure/git-bug/util/lamport"
)

var _ ClockedRepo = &mockRepoForTest{}
var _ TestedRepo = &mockRepoForTest{}

// mockRepoForTest defines an instance of Repo that can be used for testing.
type mockRepoForTest struct {
	*mockRepoConfig
	*mockRepoKeyring
	*mockRepoCommon
	*mockRepoStorage
	*mockRepoData
	*mockRepoClock
}

func NewMockRepoForTest() *mockRepoForTest {
	return &mockRepoForTest{
		mockRepoConfig:  NewMockRepoConfig(),
		mockRepoKeyring: NewMockRepoKeyring(),
		mockRepoCommon:  NewMockRepoCommon(),
		mockRepoStorage: NewMockRepoStorage(),
		mockRepoData:    NewMockRepoData(),
		mockRepoClock:   NewMockRepoClock(),
	}
}

var _ RepoConfig = &mockRepoConfig{}

type mockRepoConfig struct {
	localConfig  *MemConfig
	globalConfig *MemConfig
}

func NewMockRepoConfig() *mockRepoConfig {
	return &mockRepoConfig{
		localConfig:  NewMemConfig(),
		globalConfig: NewMemConfig(),
	}
}

// LocalConfig give access to the repository scoped configuration
func (r *mockRepoConfig) LocalConfig() Config {
	return r.localConfig
}

// GlobalConfig give access to the git global configuration
func (r *mockRepoConfig) GlobalConfig() Config {
	return r.globalConfig
}

// AnyConfig give access to a merged local/global configuration
func (r *mockRepoConfig) AnyConfig() ConfigRead {
	return mergeConfig(r.localConfig, r.globalConfig)
}

var _ RepoKeyring = &mockRepoKeyring{}

type mockRepoKeyring struct {
	keyring *keyring.ArrayKeyring
}

func NewMockRepoKeyring() *mockRepoKeyring {
	return &mockRepoKeyring{
		keyring: keyring.NewArrayKeyring(nil),
	}
}

// Keyring give access to a user-wide storage for secrets
func (r *mockRepoKeyring) Keyring() Keyring {
	return r.keyring
}

var _ RepoCommon = &mockRepoCommon{}

type mockRepoCommon struct{}

func NewMockRepoCommon() *mockRepoCommon {
	return &mockRepoCommon{}
}

func (r *mockRepoCommon) GetUserName() (string, error) {
	return "René Descartes", nil
}

// GetUserEmail returns the email address that the user has used to configure git.
func (r *mockRepoCommon) GetUserEmail() (string, error) {
	return "user@example.com", nil
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (r *mockRepoCommon) GetCoreEditor() (string, error) {
	return "vi", nil
}

// GetRemotes returns the configured remotes repositories.
func (r *mockRepoCommon) GetRemotes() (map[string]string, error) {
	return map[string]string{
		"origin": "git://github.com/MichaelMure/git-bug",
	}, nil
}

var _ RepoStorage = &mockRepoStorage{}

type mockRepoStorage struct {
	localFs billy.Filesystem
}

func NewMockRepoStorage() *mockRepoStorage {
	return &mockRepoStorage{localFs: memfs.New()}
}

func (m *mockRepoStorage) LocalStorage() billy.Filesystem {
	return m.localFs
}

var _ RepoData = &mockRepoData{}

type commit struct {
	treeHash Hash
	parent   Hash
}

type mockRepoData struct {
	blobs   map[Hash][]byte
	trees   map[Hash]string
	commits map[Hash]commit
	refs    map[string]Hash
}

func NewMockRepoData() *mockRepoData {
	return &mockRepoData{
		blobs:   make(map[Hash][]byte),
		trees:   make(map[Hash]string),
		commits: make(map[Hash]commit),
		refs:    make(map[string]Hash),
	}
}

// PushRefs push git refs to a remote
func (r *mockRepoData) PushRefs(remote string, refSpec string) (string, error) {
	return "", nil
}

func (r *mockRepoData) FetchRefs(remote string, refSpec string) (string, error) {
	return "", nil
}

func (r *mockRepoData) StoreData(data []byte) (Hash, error) {
	rawHash := sha1.Sum(data)
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.blobs[hash] = data
	return hash, nil
}

func (r *mockRepoData) ReadData(hash Hash) ([]byte, error) {
	data, ok := r.blobs[hash]

	if !ok {
		return nil, fmt.Errorf("unknown hash")
	}

	return data, nil
}

func (r *mockRepoData) StoreTree(entries []TreeEntry) (Hash, error) {
	buffer := prepareTreeEntries(entries)
	rawHash := sha1.Sum(buffer.Bytes())
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.trees[hash] = buffer.String()

	return hash, nil
}

func (r *mockRepoData) StoreCommit(treeHash Hash) (Hash, error) {
	rawHash := sha1.Sum([]byte(treeHash))
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.commits[hash] = commit{
		treeHash: treeHash,
	}
	return hash, nil
}

func (r *mockRepoData) StoreCommitWithParent(treeHash Hash, parent Hash) (Hash, error) {
	rawHash := sha1.Sum([]byte(treeHash + parent))
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.commits[hash] = commit{
		treeHash: treeHash,
		parent:   parent,
	}
	return hash, nil
}

func (r *mockRepoData) UpdateRef(ref string, hash Hash) error {
	r.refs[ref] = hash
	return nil
}

func (r *mockRepoData) RemoveRef(ref string) error {
	delete(r.refs, ref)
	return nil
}

func (r *mockRepoData) RefExist(ref string) (bool, error) {
	_, exist := r.refs[ref]
	return exist, nil
}

func (r *mockRepoData) CopyRef(source string, dest string) error {
	hash, exist := r.refs[source]

	if !exist {
		return fmt.Errorf("Unknown ref")
	}

	r.refs[dest] = hash
	return nil
}

func (r *mockRepoData) ListRefs(refPrefix string) ([]string, error) {
	var keys []string

	for k := range r.refs {
		if strings.HasPrefix(k, refPrefix) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (r *mockRepoData) ListCommits(ref string) ([]Hash, error) {
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

func (r *mockRepoData) ReadTree(hash Hash) ([]TreeEntry, error) {
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

func (r *mockRepoData) FindCommonAncestor(hash1 Hash, hash2 Hash) (Hash, error) {
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

func (r *mockRepoData) GetTreeHash(commit Hash) (Hash, error) {
	c, ok := r.commits[commit]
	if !ok {
		return "", fmt.Errorf("unknown commit")
	}

	return c.treeHash, nil
}

func (r *mockRepoData) AddRemote(name string, url string) error {
	panic("implement me")
}

func (m mockRepoForTest) GetLocalRemote() string {
	panic("implement me")
}

func (m mockRepoForTest) EraseFromDisk() error {
	// nothing to do
	return nil
}

type mockRepoClock struct {
	mu     sync.Mutex
	clocks map[string]lamport.Clock
}

func NewMockRepoClock() *mockRepoClock {
	return &mockRepoClock{
		clocks: make(map[string]lamport.Clock),
	}
}

func (r *mockRepoClock) GetOrCreateClock(name string) (lamport.Clock, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.clocks[name]; ok {
		return c, nil
	}

	c := lamport.NewMemClock()
	r.clocks[name] = c
	return c, nil
}
