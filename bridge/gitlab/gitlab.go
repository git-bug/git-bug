package gitlab

import (
	"net/http"
	"time"

	"github.com/xanzy/go-gitlab"

	"github.com/MichaelMure/git-bug/bridge/core"
)

const (
	target = "gitlab"

	keyGitlabId      = "gitlab-id"
	keyGitlabUrl     = "gitlab-url"
	keyGitlabLogin   = "gitlab-login"
	keyGitlabProject = "gitlab-project-id"

	keyProjectID = "project-id"
	keyToken     = "token"

	defaultTimeout = 60 * time.Second
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
	client := &http.Client{
		Timeout: defaultTimeout,
	}

	return gitlab.NewClient(client, token)
}
