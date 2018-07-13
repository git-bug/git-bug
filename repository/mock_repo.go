package repository

import (
	"github.com/MichaelMure/git-bug/util"
)

// mockRepoForTest defines an instance of Repo that can be used for testing.
type mockRepoForTest struct{}

func NewMockRepoForTest() Repo {
	return &mockRepoForTest{}
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

func (r *mockRepoForTest) PullRefs(remote string, refPattern string) error {
	return nil
}

func (r *mockRepoForTest) StoreData([]byte) (util.Hash, error) {
	return "", nil
}

func (r *mockRepoForTest) StoreTree(mapping map[string]util.Hash) (util.Hash, error) {
	return "", nil
}

func (r *mockRepoForTest) StoreCommit(treeHash util.Hash) (util.Hash, error) {
	return "", nil
}

func (r *mockRepoForTest) StoreCommitWithParent(treeHash util.Hash, parent util.Hash) (util.Hash, error) {
	return "", nil
}

func (r *mockRepoForTest) UpdateRef(ref string, hash util.Hash) error {
	return nil
}
