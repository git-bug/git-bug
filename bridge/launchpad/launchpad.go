// Package launchpad contains the Launchpad bridge implementation
package launchpad

import (
	"time"

	"github.com/git-bug/git-bug/bridge/core"
)

const (
	target = "launchpad-preview"

	metaKeyLaunchpadID    = "launchpad-id"
	metaKeyLaunchpadLogin = "launchpad-login"

	confKeyProject = "project"

	defaultTimeout = 60 * time.Second
)

var _ core.BridgeImpl = &Launchpad{}

type Launchpad struct{}

func (*Launchpad) Target() string {
	return "launchpad-preview"
}

func (Launchpad) LoginMetaKey() string {
	return metaKeyLaunchpadLogin
}

func (*Launchpad) NewImporter() core.Importer {
	return &launchpadImporter{}
}

func (*Launchpad) NewExporter() core.Exporter {
	return nil
}
