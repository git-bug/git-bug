package repository

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
)

// mockRepoForTest defines an instance of Repo that can be used for testing.
type mockRepoForTest struct {}

// GetPath returns the path to the repo.
func (r *mockRepoForTest) GetPath() string { return "~/mockRepo/" }

// GetRepoStateHash returns a hash which embodies the entire current state of a repository.
func (r *mockRepoForTest) GetRepoStateHash() (string, error) {
	repoJSON, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(repoJSON))), nil
}

// GetUserEmail returns the email address that the user has used to configure git.
func (r *mockRepoForTest) GetUserEmail() (string, error) { return "user@example.com", nil }

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (r *mockRepoForTest) GetCoreEditor() (string, error) { return "vi", nil }

// PushRefs push git refs to a remote
func (r *mockRepoForTest) PushRefs(remote string, refPattern string) error {
	return nil
}