package bridge

import (
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/github"
)

// Bridges return all known bridges
func Bridges() []*core.Bridge {
	return []*core.Bridge{
		core.NewBridge(&github.Github{}),
	}
}
