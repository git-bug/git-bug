package resolvers

import (
	"context"
	"image/color"

	"github.com/MichaelMure/git-bug/graphql/graph"
)

var _ graph.ColorResolver = &colorResolver{}

type colorResolver struct{}

func (colorResolver) R(ctx context.Context, obj *color.RGBA) (int, error) {
	return int(obj.R), nil
}

func (colorResolver) G(ctx context.Context, obj *color.RGBA) (int, error) {
	return int(obj.G), nil
}

func (colorResolver) B(ctx context.Context, obj *color.RGBA) (int, error) {
	return int(obj.B), nil
}
