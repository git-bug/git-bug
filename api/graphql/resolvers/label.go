package resolvers

import (
	"context"
	"image/color"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/entities/bug"
)

var _ graph.LabelResolver = &labelResolver{}

type labelResolver struct{}

func (labelResolver) Name(ctx context.Context, obj *bug.Label) (string, error) {
	return obj.String(), nil
}

func (labelResolver) Color(ctx context.Context, obj *bug.Label) (*color.RGBA, error) {
	rgba := obj.Color().RGBA()
	return &rgba, nil
}
