package resolvers

import (
	"context"
	"image/color"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/graphql/graph"
)

var _ graph.LabelResolver = &labelResolver{}

type labelResolver struct{}

func (labelResolver) Name(ctx context.Context, obj *bug.Label) (string, error) {
	return obj.String(), nil
}

func (labelResolver) Color(ctx context.Context, obj *bug.Label) (*color.RGBA, error) {
	rgba := obj.RGBA()
	return &rgba, nil
}
