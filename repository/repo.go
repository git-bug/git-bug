// Package repository contains helper methods for working with a Git repo.
package repository

import "github.com/MichaelMure/git-bug/util"

// Repo represents a source code repository.
type Repo interface {
	// GetPath returns the path to the repo.
	GetPath() string

	// GetUserName returns the name the the user has used to configure git
	GetUserName() (string, error)

	// GetUserEmail returns the email address that the user has used to configure git.
	GetUserEmail() (string, error)

	// GetCoreEditor returns the name of the editor that the user has used to configure git.
	GetCoreEditor() (string, error)

	// PullRefs pull git refs from a remote
	PullRefs(remote string, refPattern string, remoteRefPattern string) error

	// PushRefs push git refs to a remote
	PushRefs(remote string, refPattern string) error

	// StoreData will store arbitrary data and return the corresponding hash
	StoreData(data []byte) (util.Hash, error)

	// StoreTree will store a mapping key-->Hash as a Git tree
	StoreTree(mapping map[string]util.Hash) (util.Hash, error)

	// StoreCommit will store a Git commit with the given Git tree
	StoreCommit(treeHash util.Hash) (util.Hash, error)

	// StoreCommit will store a Git commit with the given Git tree
	StoreCommitWithParent(treeHash util.Hash, parent util.Hash) (util.Hash, error)

	// UpdateRef will create or update a Git reference
	UpdateRef(ref string, hash util.Hash) error
}
