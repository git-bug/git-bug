// Package bridge contains the high-level public functions to use and manage bridges
package bridge

import (
	"github.com/MichaelMure/git-bug/bridge/core"
	_ "github.com/MichaelMure/git-bug/bridge/github"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

// Targets return all known bridge implementation target
func Targets() []string {
	return core.Targets()
}

// Instantiate a new Bridge for a repo, from the given target and name
func NewBridge(repo *cache.RepoCache, target string, name string) (*core.Bridge, error) {
	return core.NewBridge(repo, target, name)
}

// Instantiate a new bridge for a repo, from the combined target and name contained
// in the full name
func NewBridgeFullName(repo *cache.RepoCache, fullName string) (*core.Bridge, error) {
	return core.NewBridgeFullName(repo, fullName)
}

// Attempt to retrieve a default bridge for the given repo. If zero or multiple
// bridge exist, it fails.
func DefaultBridge(repo *cache.RepoCache) (*core.Bridge, error) {
	return core.DefaultBridge(repo)
}

// ConfiguredBridges return the list of bridge that are configured for the given
// repo
func ConfiguredBridges(repo repository.RepoCommon) ([]string, error) {
	return core.ConfiguredBridges(repo)
}

// Remove a configured bridge
func RemoveBridges(repo repository.RepoCommon, fullName string) error {
	return core.RemoveBridge(repo, fullName)
}
