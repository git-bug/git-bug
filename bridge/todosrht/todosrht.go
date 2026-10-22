package todosrht

import (
	"fmt"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
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
type TodoSourceHut struct {
	importer *todosrhtImporter
	exporter *todosrhtExporter
}

// Target returns "todosrht"
func (b *TodoSourceHut) Target() string {
	return target
}

func (b *TodoSourceHut) LoginMetaKey() string {
	return metaKeyTodoSourceHutLogin
}

func (*TodoSourceHut) CredentialMetadata(login, baseURL string) (map[string]string, map[string]string, error) {
	if baseURL == "" {
		return nil, nil, fmt.Errorf("--base-url is required for todosrht credentials")
	}
	baseURL, err := validateBaseURL(baseURL)
	if err != nil {
		return nil, nil, err
	}
	return map[string]string{identityLoginKey(baseURL): login},
		map[string]string{auth.MetaKeyLogin: login, auth.MetaKeyBaseURL: baseURL}, nil
}

// NewImporter returns the todosrht importer
func (b *TodoSourceHut) NewImporter() core.Importer {
	if b.importer == nil {
		b.importer = &todosrhtImporter{}
	}
	return b.importer
}

// NewExporter returns the todosrht exporter
func (b *TodoSourceHut) NewExporter() core.Exporter {
	if b.exporter == nil {
		b.exporter = &todosrhtExporter{}
	}
	return b.exporter
}
