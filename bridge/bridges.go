package bridge

import (
	"github.com/MichaelMure/git-bug/bridge/core"
	_ "github.com/MichaelMure/git-bug/bridge/github"
	"github.com/MichaelMure/git-bug/repository"
)

// Targets return all known bridge implementation target
func Targets() []string {
	return core.Targets()
}

func ConfiguredBridges(repo repository.RepoCommon) ([]string, error) {
	return core.ConfiguredBridges(repo)
}
