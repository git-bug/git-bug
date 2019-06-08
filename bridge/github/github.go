// Package github contains the Github bridge implementation
package github

import (
	"context"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/MichaelMure/git-bug/bridge/core"
)

func init() {
	core.Register(&Github{})
}

type Github struct{}

func (*Github) Target() string {
	return "github"
}

func (*Github) NewImporter() core.Importer {
	return &githubImporter{}
}

func (*Github) NewExporter() core.Exporter {
	return &githubExporter{}
}

func buildClient(token string) *githubv4.Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.TODO(), src)

	return githubv4.NewClient(httpClient)
}
