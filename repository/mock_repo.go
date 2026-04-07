package repository

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/99designs/keyring"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-billy/v5/memfs"

	"github.com/git-bug/git-bug/util/lamport"
)

var _ ClockedRepo = &mockRepo{}
var _ TestedRepo = &mockRepo{}

// mockRepo defines an instance of Repo that can be used for testing.
type mockRepo struct {
	*mockRepoConfig
	*mockRepoKeyring
	*mockRepoCommon
	*mockRepoStorage
	*mockRepoIndex
	*mockRepoDataBrowse
	*mockRepoClock
	*mockRepoTest
}

func (m *mockRepo) Close() error { return nil }

func NewMockRepo() *mockRepo {
	return &mockRepo{
		mockRepoConfig:     NewMockRepoConfig(),
		mockRepoKeyring:    NewMockRepoKeyring(),
		mockRepoCommon:     NewMockRepoCommon(),
		mockRepoStorage:    NewMockRepoStorage(),
		mockRepoIndex:      newMockRepoIndex(),
		mockRepoDataBrowse: newMockRepoDataBrowse(),
		mockRepoClock:      NewMockRepoClock(),
		mockRepoTest:       NewMockRepoTest(),
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
		"origin": "git://github.com/git-bug/git-bug",
	}, nil
}

var _ RepoStorage = &mockRepoStorage{}

type mockRepoStorage struct {
	localFs LocalStorage
}

func NewMockRepoStorage() *mockRepoStorage {
	return &mockRepoStorage{localFs: billyLocalStorage{Filesystem: memfs.New()}}
}

func (m *mockRepoStorage) LocalStorage() LocalStorage {
	return m.localFs
}

var _ RepoIndex = &mockRepoIndex{}

type mockRepoIndex struct {
	indexesMutex sync.Mutex
	indexes      map[string]Index
}

func newMockRepoIndex() *mockRepoIndex {
	return &mockRepoIndex{
		indexes: make(map[string]Index),
	}
}

func (m *mockRepoIndex) GetIndex(name string) (Index, error) {
	m.indexesMutex.Lock()
	defer m.indexesMutex.Unlock()

	if index, ok := m.indexes[name]; ok {
		return index, nil
	}

	index := newIndex()
	m.indexes[name] = index
	return index, nil
}

var _ Index = &mockIndex{}

type mockIndex map[string][]string

func newIndex() *mockIndex {
	m := make(map[string][]string)
	return (*mockIndex)(&m)
}

func (m *mockIndex) IndexOne(id string, texts []string) error {
	(*m)[id] = texts
	return nil
}

func (m *mockIndex) IndexBatch() (indexer func(id string, texts []string) error, closer func() error) {
	indexer = func(id string, texts []string) error {
		(*m)[id] = texts
		return nil
	}
	closer = func() error { return nil }
	return indexer, closer
}

func (m *mockIndex) Search(terms []string) (ids []string, err error) {
loop:
	for id, texts := range *m {
		for _, text := range texts {
			for _, s := range strings.Fields(text) {
				for _, term := range terms {
					if s == term {
						ids = append(ids, id)
						continue loop
					}
				}
			}
		}
	}
	return ids, nil
}

func (m *mockIndex) DocCount() (uint64, error) {
	return uint64(len(*m)), nil
}

func (m *mockIndex) Remove(id string) error {
	delete(*m, id)
	return nil
}

func (m *mockIndex) Clear() error {
	for k, _ := range *m {
		delete(*m, k)
	}
	return nil
}

func (m *mockIndex) Close() error {
	return nil
}

var _ RepoData = &mockRepoDataBrowse{}

type commit struct {
	treeHash Hash
	parents  []Hash
	sig      string
	date     time.Time
	message  string
}

type mockRepoDataBrowse struct {
	blobs         map[Hash][]byte
	trees         map[Hash]string
	commits       map[Hash]commit
	refs          map[string]Hash
	defaultBranch string
}

func newMockRepoDataBrowse() *mockRepoDataBrowse {
	return &mockRepoDataBrowse{
		blobs:         make(map[Hash][]byte),
		trees:         make(map[Hash]string),
		commits:       make(map[Hash]commit),
		refs:          make(map[string]Hash),
		defaultBranch: "main",
	}
}

func (r *mockRepoDataBrowse) FetchRefs(remote string, prefixes ...string) (string, error) {
	panic("implement me")
}

// PushRefs push git refs to a remote
func (r *mockRepoDataBrowse) PushRefs(remote string, prefixes ...string) (string, error) {
	panic("implement me")
}

func (r *mockRepoDataBrowse) StoreData(data []byte) (Hash, error) {
	rawHash := sha1.Sum(data)
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.blobs[hash] = data
	return hash, nil
}

func (r *mockRepoDataBrowse) ReadData(hash Hash) (io.ReadCloser, error) {
	data, ok := r.blobs[hash]
	if !ok {
		return nil, ErrNotFound
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func (r *mockRepoDataBrowse) StoreTree(entries []TreeEntry) (Hash, error) {
	buffer := prepareTreeEntries(entries)
	rawHash := sha1.Sum(buffer.Bytes())
	hash := Hash(fmt.Sprintf("%x", rawHash))
	r.trees[hash] = buffer.String()

	return hash, nil
}

func (r *mockRepoDataBrowse) ReadTree(hash Hash) ([]TreeEntry, error) {
	var data string

	data, ok := r.trees[hash]

	if !ok {
		// Git will understand a commit hash to reach a tree
		commit, ok := r.commits[hash]

		if !ok {
			return nil, ErrNotFound
		}

		data, ok = r.trees[commit.treeHash]

		if !ok {
			return nil, ErrNotFound
		}
	}

	return readTreeEntries(data)
}

func (r *mockRepoDataBrowse) StoreCommit(treeHash Hash, parents ...Hash) (Hash, error) {
	return r.StoreSignedCommit(treeHash, nil, parents...)
}

func (r *mockRepoDataBrowse) StoreSignedCommit(treeHash Hash, signKey *openpgp.Entity, parents ...Hash) (Hash, error) {
	hasher := sha1.New()
	hasher.Write([]byte(treeHash))
	for _, parent := range parents {
		hasher.Write([]byte(parent))
	}
	rawHash := hasher.Sum(nil)
	hash := Hash(fmt.Sprintf("%x", rawHash))
	c := commit{
		treeHash: treeHash,
		parents:  parents,
		date:     time.Now(),
	}
	if signKey != nil {
		// unlike go-git, we only sign the tree hash for simplicity instead of all the fields (parents ...)
		var sig bytes.Buffer
		if err := openpgp.DetachSign(&sig, signKey, strings.NewReader(string(treeHash)), nil); err != nil {
			return "", err
		}
		c.sig = sig.String()
	}
	r.commits[hash] = c
	return hash, nil
}

func (r *mockRepoDataBrowse) ReadCommit(hash Hash) (Commit, error) {
	c, ok := r.commits[hash]
	if !ok {
		return Commit{}, ErrNotFound
	}

	result := Commit{
		Hash:     hash,
		Parents:  c.parents,
		TreeHash: c.treeHash,
	}

	if c.sig != "" {
		// Note: this is actually incorrect as the signed data should be the full commit (+comment, +date ...)
		// but only the tree hash work for our purpose here.
		result.SignedData = strings.NewReader(string(c.treeHash))
		result.Signature = strings.NewReader(c.sig)
	}

	return result, nil
}

func (r *mockRepoDataBrowse) ResolveRef(ref string) (Hash, error) {
	h, ok := r.refs[ref]
	if !ok {
		return "", ErrNotFound
	}
	return h, nil
}

func (r *mockRepoDataBrowse) UpdateRef(ref string, hash Hash) error {
	r.refs[ref] = hash
	return nil
}

func (r *mockRepoDataBrowse) RemoveRef(ref string) error {
	delete(r.refs, ref)
	return nil
}

func (r *mockRepoDataBrowse) ListRefs(refPrefix string) ([]string, error) {
	var keys []string

	for k := range r.refs {
		if strings.HasPrefix(k, refPrefix) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (r *mockRepoDataBrowse) RefExist(ref string) (bool, error) {
	_, exist := r.refs[ref]
	return exist, nil
}

func (r *mockRepoDataBrowse) CopyRef(source string, dest string) error {
	hash, exist := r.refs[source]

	if !exist {
		return ErrNotFound
	}

	r.refs[dest] = hash
	return nil
}

func (r *mockRepoDataBrowse) ListCommits(ref string) ([]Hash, error) {
	return nonNativeListCommits(r, ref)
}

// resolveRef resolves a ref matching the RepoBrowse contract:
// refs/heads/<ref>, refs/tags/<ref>, full ref name, raw commit hash.
func (r *mockRepoDataBrowse) resolveRef(ref string) (Hash, error) {
	for _, candidate := range []string{"refs/heads/" + ref, "refs/tags/" + ref, ref} {
		if h, ok := r.refs[candidate]; ok {
			return h, nil
		}
	}
	if _, ok := r.commits[Hash(ref)]; ok {
		return Hash(ref), nil
	}
	return "", ErrNotFound
}

// treeEntriesAtHash parses the entries of the tree stored under hash.
func (r *mockRepoDataBrowse) treeEntriesAtHash(hash Hash) ([]TreeEntry, error) {
	data, ok := r.trees[hash]
	if !ok {
		return nil, ErrNotFound
	}
	return readTreeEntries(data)
}

// treeEntriesAt returns the directory entries at path inside the tree rooted at
// treeHash. path="" returns root entries. Returns ErrNotFound if path doesn't
// exist or resolves to a blob rather than a tree.
func (r *mockRepoDataBrowse) treeEntriesAt(treeHash Hash, path string) ([]TreeEntry, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return r.treeEntriesAtHash(treeHash)
	}
	seg, rest, _ := strings.Cut(path, "/")
	entries, err := r.treeEntriesAtHash(treeHash)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.Name != seg || e.ObjectType != Tree {
			continue
		}
		if rest == "" {
			return r.treeEntriesAtHash(e.Hash)
		}
		return r.treeEntriesAt(e.Hash, rest)
	}
	return nil, ErrNotFound
}

// blobHashAt walks the tree to find the blob hash for the file at path.
func (r *mockRepoDataBrowse) blobHashAt(treeHash Hash, path string) (Hash, error) {
	path = strings.Trim(path, "/")
	seg, rest, hasRest := strings.Cut(path, "/")
	entries, err := r.treeEntriesAtHash(treeHash)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.Name != seg {
			continue
		}
		if !hasRest {
			return e.Hash, nil
		}
		if e.ObjectType != Tree {
			return "", ErrNotFound
		}
		return r.blobHashAt(e.Hash, rest)
	}
	return "", ErrNotFound
}

// diffTrees returns the changed files between two trees, recursing into
// sub-trees. fromHash=="" means an empty (non-existent) tree.
func (r *mockRepoDataBrowse) diffTrees(fromHash, toHash Hash, prefix string) []ChangedFile {
	var fromEntries, toEntries []TreeEntry
	if fromHash != "" {
		fromEntries, _ = r.treeEntriesAtHash(fromHash)
	}
	if toHash != "" {
		toEntries, _ = r.treeEntriesAtHash(toHash)
	}

	fromMap := make(map[string]TreeEntry, len(fromEntries))
	for _, e := range fromEntries {
		fromMap[e.Name] = e
	}
	toMap := make(map[string]TreeEntry, len(toEntries))
	for _, e := range toEntries {
		toMap[e.Name] = e
	}

	var result []ChangedFile
	for _, e := range toEntries {
		path := prefix + e.Name
		f, existed := fromMap[e.Name]
		if e.ObjectType == Tree {
			var sub Hash
			if existed {
				sub = f.Hash
			}
			result = append(result, r.diffTrees(sub, e.Hash, path+"/")...)
		} else if !existed {
			result = append(result, ChangedFile{Path: path, Status: ChangeStatusAdded})
		} else if f.Hash != e.Hash {
			result = append(result, ChangedFile{Path: path, Status: ChangeStatusModified})
		}
	}
	for _, f := range fromEntries {
		if _, exists := toMap[f.Name]; exists {
			continue
		}
		path := prefix + f.Name
		if f.ObjectType == Tree {
			result = append(result, r.diffTrees(f.Hash, "", path+"/")...)
		} else {
			result = append(result, ChangedFile{Path: path, Status: ChangeStatusDeleted})
		}
	}
	return result
}

func mockCommitMeta(hash Hash, c commit) CommitMeta {
	return CommitMeta{
		Hash:    hash,
		Parents: c.parents,
		Date:    c.date,
		Message: c.message,
	}
}

func (r *mockRepoDataBrowse) Branches() ([]BranchInfo, error) {
	var branches []BranchInfo
	for ref, hash := range r.refs {
		name, ok := strings.CutPrefix(ref, "refs/heads/")
		if !ok {
			continue
		}
		branches = append(branches, BranchInfo{
			Name:      name,
			Hash:      hash,
			IsDefault: name == r.defaultBranch,
		})
	}
	return branches, nil
}

func (r *mockRepoDataBrowse) Tags() ([]TagInfo, error) {
	var tags []TagInfo
	for ref, hash := range r.refs {
		name, ok := strings.CutPrefix(ref, "refs/tags/")
		if !ok {
			continue
		}
		tags = append(tags, TagInfo{Name: name, Hash: hash})
	}
	return tags, nil
}

func (r *mockRepoDataBrowse) TreeAtPath(ref, path string) ([]TreeEntry, error) {
	startHash, err := r.resolveRef(ref)
	if err != nil {
		return nil, ErrNotFound
	}
	c, ok := r.commits[startHash]
	if !ok {
		return nil, ErrNotFound
	}
	return r.treeEntriesAt(c.treeHash, path)
}

func (r *mockRepoDataBrowse) BlobAtPath(ref, path string) (io.ReadCloser, int64, Hash, error) {
	startHash, err := r.resolveRef(ref)
	if err != nil {
		return nil, 0, "", ErrNotFound
	}
	c, ok := r.commits[startHash]
	if !ok {
		return nil, 0, "", ErrNotFound
	}
	blobHash, err := r.blobHashAt(c.treeHash, path)
	if err != nil {
		return nil, 0, "", ErrNotFound
	}
	data, ok := r.blobs[blobHash]
	if !ok {
		return nil, 0, "", ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), blobHash, nil
}

func (r *mockRepoDataBrowse) CommitLog(ref, path string, limit int, after Hash, since, until *time.Time) ([]CommitMeta, error) {
	startHash, err := r.resolveRef(ref)
	if err != nil {
		return nil, ErrNotFound
	}
	path = strings.Trim(path, "/")
	var result []CommitMeta
	skipping := after != ""
	current := startHash
	seen := make(map[Hash]bool)
	for {
		if seen[current] {
			break
		}
		seen[current] = true
		c, ok := r.commits[current]
		if !ok {
			break
		}
		if skipping {
			if current == after {
				skipping = false
			}
			if len(c.parents) == 0 {
				break
			}
			current = c.parents[0]
			continue
		}
		meta := mockCommitMeta(current, c)
		if since != nil && meta.Date.Before(*since) {
			if len(c.parents) == 0 {
				break
			}
			current = c.parents[0]
			continue
		}
		if until != nil && meta.Date.After(*until) {
			if len(c.parents) == 0 {
				break
			}
			current = c.parents[0]
			continue
		}
		if path != "" {
			var fromTreeHash Hash
			if len(c.parents) > 0 {
				if parent, ok := r.commits[c.parents[0]]; ok {
					fromTreeHash = parent.treeHash
				}
			}
			touched := false
			for _, f := range r.diffTrees(fromTreeHash, c.treeHash, "") {
				if f.Path == path || strings.HasPrefix(f.Path, path+"/") {
					touched = true
					break
				}
			}
			if !touched {
				if len(c.parents) == 0 {
					break
				}
				current = c.parents[0]
				continue
			}
		}
		result = append(result, meta)
		if limit > 0 && len(result) >= limit {
			break
		}
		if len(c.parents) == 0 {
			break
		}
		current = c.parents[0]
	}
	return result, nil
}

func (r *mockRepoDataBrowse) LastCommitForEntries(ref, path string, names []string) (map[string]CommitMeta, error) {
	startHash, err := r.resolveRef(ref)
	if err != nil {
		return nil, ErrNotFound
	}
	path = strings.Trim(path, "/")
	remaining := make(map[string]bool, len(names))
	for _, n := range names {
		remaining[n] = true
	}
	result := make(map[string]CommitMeta)
	current := startHash
	seen := make(map[Hash]bool)
	for len(remaining) > 0 {
		if seen[current] {
			break
		}
		seen[current] = true
		c, ok := r.commits[current]
		if !ok {
			break
		}
		curEntries, err := r.treeEntriesAt(c.treeHash, path)
		if err != nil {
			if len(c.parents) == 0 {
				break
			}
			current = c.parents[0]
			continue
		}
		curMap := make(map[string]Hash, len(curEntries))
		for _, e := range curEntries {
			curMap[e.Name] = e.Hash
		}
		if len(c.parents) == 0 {
			for name := range remaining {
				if _, ok := curMap[name]; ok {
					result[name] = mockCommitMeta(current, c)
					delete(remaining, name)
				}
			}
			break
		}
		pc, ok := r.commits[c.parents[0]]
		if !ok {
			break
		}
		parentEntries, _ := r.treeEntriesAt(pc.treeHash, path)
		parentMap := make(map[string]Hash, len(parentEntries))
		for _, e := range parentEntries {
			parentMap[e.Name] = e.Hash
		}
		for name := range remaining {
			cur, curExists := curMap[name]
			par, parExists := parentMap[name]
			if curExists && (!parExists || cur != par) {
				result[name] = mockCommitMeta(current, c)
				delete(remaining, name)
			}
		}
		current = c.parents[0]
	}
	return result, nil
}

func (r *mockRepoDataBrowse) CommitDetail(hash Hash) (CommitDetail, error) {
	c, ok := r.commits[hash]
	if !ok {
		return CommitDetail{}, ErrNotFound
	}
	var fromTreeHash Hash
	if len(c.parents) > 0 {
		if parent, ok := r.commits[c.parents[0]]; ok {
			fromTreeHash = parent.treeHash
		}
	}
	return CommitDetail{
		CommitMeta: mockCommitMeta(hash, c),
		Files:      r.diffTrees(fromTreeHash, c.treeHash, ""),
	}, nil
}

func (r *mockRepoDataBrowse) CommitFileDiff(hash Hash, filePath string) (FileDiff, error) {
	c, ok := r.commits[hash]
	if !ok {
		return FileDiff{}, ErrNotFound
	}
	var fromTreeHash Hash
	if len(c.parents) > 0 {
		if parent, ok := r.commits[c.parents[0]]; ok {
			fromTreeHash = parent.treeHash
		}
	}
	files := r.diffTrees(fromTreeHash, c.treeHash, "")
	var matched *ChangedFile
	for i := range files {
		if files[i].Path == filePath {
			matched = &files[i]
			break
		}
	}
	if matched == nil {
		return FileDiff{}, ErrNotFound
	}
	fd := FileDiff{
		Path:     filePath,
		IsNew:    matched.Status == ChangeStatusAdded,
		IsDelete: matched.Status == ChangeStatusDeleted,
	}
	var oldContent, newContent []byte
	if fromTreeHash != "" {
		if bh, err := r.blobHashAt(fromTreeHash, filePath); err == nil {
			oldContent = r.blobs[bh]
		}
	}
	if bh, err := r.blobHashAt(c.treeHash, filePath); err == nil {
		newContent = r.blobs[bh]
	}
	fd.Hunks = mockDiffHunks(oldContent, newContent)
	return fd, nil
}

// mockDiffHunks produces a single DiffHunk using a prefix/suffix scan.
func mockDiffHunks(old, new []byte) []DiffHunk {
	oldLines := splitBlobLines(old)
	newLines := splitBlobLines(new)
	i := 0
	for i < len(oldLines) && i < len(newLines) && oldLines[i] == newLines[i] {
		i++
	}
	j, k := len(oldLines), len(newLines)
	for j > i && k > i && oldLines[j-1] == newLines[k-1] {
		j--
		k--
	}
	if j == i && k == i {
		return nil // no changed region
	}
	oldLine, newLine := 1, 1
	var lines []DiffLine
	for _, l := range oldLines[:i] {
		lines = append(lines, DiffLine{Type: DiffLineContext, Content: l, OldLine: oldLine, NewLine: newLine})
		oldLine++
		newLine++
	}
	for _, l := range oldLines[i:j] {
		lines = append(lines, DiffLine{Type: DiffLineDeleted, Content: l, OldLine: oldLine})
		oldLine++
	}
	for _, l := range newLines[i:k] {
		lines = append(lines, DiffLine{Type: DiffLineAdded, Content: l, NewLine: newLine})
		newLine++
	}
	for _, l := range oldLines[j:] {
		lines = append(lines, DiffLine{Type: DiffLineContext, Content: l, OldLine: oldLine, NewLine: newLine})
		oldLine++
		newLine++
	}
	return []DiffHunk{{OldStart: 1, OldLines: len(oldLines), NewStart: 1, NewLines: len(newLines), Lines: lines}}
}

func splitBlobLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}

var _ RepoClock = &mockRepoClock{}

type mockRepoClock struct {
	mu     sync.Mutex
	clocks map[string]lamport.Clock
}

func NewMockRepoClock() *mockRepoClock {
	return &mockRepoClock{
		clocks: make(map[string]lamport.Clock),
	}
}

func (r *mockRepoClock) AllClocks() (map[string]lamport.Clock, error) {
	return r.clocks, nil
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

func (r *mockRepoClock) Increment(name string) (lamport.Time, error) {
	c, err := r.GetOrCreateClock(name)
	if err != nil {
		return lamport.Time(0), err
	}
	return c.Increment()
}

func (r *mockRepoClock) Witness(name string, time lamport.Time) error {
	c, err := r.GetOrCreateClock(name)
	if err != nil {
		return err
	}
	return c.Witness(time)
}

var _ repoTest = &mockRepoTest{}

type mockRepoTest struct{}

func NewMockRepoTest() *mockRepoTest {
	return &mockRepoTest{}
}

func (r *mockRepoTest) AddRemote(name string, url string) error {
	panic("implement me")
}

func (r mockRepoTest) GetLocalRemote() string {
	panic("implement me")
}

func (r mockRepoTest) EraseFromDisk() error {
	// nothing to do
	return nil
}
