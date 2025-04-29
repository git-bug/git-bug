package todosrht

import (
	"time"

	"github.com/git-bug/git-bug/bridge/core"
)

const (
	target = "todosrht"

	metaKeyTodoSourceHutId         = "todosrht-id"
	metaKeyTodoSourceHutTracker    = "todosrht-tracker"
	metaKeyTodoSourceHutRef        = "todosrht-ref"
	metaKeyTodoSourceHutUser       = "todosrht-user"
	metaKeyTodoSourceHutBaseUrl    = "todosrht-base-url"
	metaKeyTodoSourceHutExportTime = "todosrht-export-time"
	metaKeyTodoSourceHutLogin      = "todosrht-login"

	confKeyBaseUrl      = "base-url"
	confKeyTrackerName  = "tracker-name"
	confKeyDefaultLogin = "default-login"

	defaultTimeout = 60 * time.Second
)

var _ core.BridgeImpl = &TodoSourceHut{}

// TodoSourceHut Main object for the bridge
type TodoSourceHut struct{}

// Target returns "todosrht"
func (*TodoSourceHut) Target() string {
	return target
}

func (*TodoSourceHut) LoginMetaKey() string {
	return metaKeyTodoSourceHutLogin
}

// NewImporter returns the todosrht importer
func (*TodoSourceHut) NewImporter() core.Importer {
	return &todosrhtImporter{}
}

// NewExporter returns the todosrht exporter
func (*TodoSourceHut) NewExporter() core.Exporter {
	return &todosrhtExporter{}
}
