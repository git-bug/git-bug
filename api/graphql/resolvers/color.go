package resolvers

import (
	"context"
	"image/color"

	"github.com/git-bug/git-bug/api/graphql/graph"
)

var _ graph.ColorResolver = &colorResolver{}

type colorResolver struct{}

func (colorResolver) R(_ context.Context, obj *color.RGBA) (int, error) {
	return int(obj.R), nil
}

func (colorResolver) G(_ context.Context, obj *color.RGBA) (int, error) {
	return int(obj.G), nil
}

func (colorResolver) B(_ context.Context, obj *color.RGBA) (int, error) {
	return int(obj.B), nil
}
