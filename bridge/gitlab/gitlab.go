package gitlab

import (
	"github.com/xanzy/go-gitlab"

	"github.com/MichaelMure/git-bug/bridge/core"
)

func init() {
	core.Register(&Gitlab{})
}

type Gitlab struct{}

func (*Gitlab) Target() string {
	return target
}

func (*Gitlab) NewImporter() core.Importer {
	return &gitlabImporter{}
}

func (*Gitlab) NewExporter() core.Exporter {
	return &gitlabExporter{}
}

func buildClient(token string) *gitlab.Client {
	return gitlab.NewClient(nil, token)
}

func buildClientFromUsernameAndPassword(username, password string) (*gitlab.Client, error) {
	return gitlab.NewBasicAuthClient(nil, "https://gitlab.com", username, password)

}
