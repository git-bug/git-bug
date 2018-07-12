// Package repository contains helper methods for working with a Git repo.
package repository

// Repo represents a source code repository.
type Repo interface {
	// GetPath returns the path to the repo.
	GetPath() string

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() (string, error)

	// GetCoreEditor returns the name of the editor that the user has used to configure git.
	GetCoreEditor() (string, error)

	// PullRefs pull git refs from a remote
	PullRefs(remote string, refPattern string) error

	// PushRefs push git refs to a remote
	PushRefs(remote string, refPattern string) error
}
