// Package github contains the Github bridge implementation
package github

import (
	"context"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func init() {
	core.Register(&Github{})
}

type Github struct{}

func (*Github) Target() string {
	return "github"
}

func (*Github) Importer() core.Importer {
	return &githubImporter{}
}

func (*Github) Exporter() core.Exporter {
	return nil
}

func buildClient(conf core.Configuration) *githubv4.Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: conf[keyToken]},
	)
	httpClient := oauth2.NewClient(context.TODO(), src)

	return githubv4.NewClient(httpClient)
}
