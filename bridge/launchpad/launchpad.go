// Package launchpad contains the Launchpad bridge implementation
package launchpad

import (
	"github.com/MichaelMure/git-bug/bridge/core"
)

func init() {
	core.Register(&Launchpad{})
}

type Launchpad struct{}

func (*Launchpad) Target() string {
	return "launchpad-preview"
}

func (*Launchpad) NewImporter() core.Importer {
	return &launchpadImporter{}
}

func (*Launchpad) NewExporter() core.Exporter {
	return nil
}
