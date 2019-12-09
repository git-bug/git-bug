// Package github contains the Github bridge implementation
package github

import (
	"context"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
)

type Github struct{}

func (*Github) Target() string {
	return target
}

func (*Github) NewImporter() core.Importer {
	return &githubImporter{}
}

func (*Github) NewExporter() core.Exporter {
	return &githubExporter{}
}

func buildClient(token *auth.Token) *githubv4.Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token.Value},
	)
	httpClient := oauth2.NewClient(context.TODO(), src)

	return githubv4.NewClient(httpClient)
}
