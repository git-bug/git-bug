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

func NewBridge(repo *cache.RepoCache, target string, name string) (*core.Bridge, error) {
	return core.NewBridge(repo, target, name)
}

func ConfiguredBridges(repo repository.RepoCommon) ([]string, error) {
	return core.ConfiguredBridges(repo)
}

func RemoveBridges(repo repository.RepoCommon, fullName string) error {
	return core.RemoveBridge(repo, fullName)
}
