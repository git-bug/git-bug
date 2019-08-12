package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/identity"
)

var _ graph.IdentityResolver = &identityResolver{}

type identityResolver struct{}

func (identityResolver) ID(ctx context.Context, obj *identity.Interface) (string, error) {
	return (*obj).Id().String(), nil
}

func (identityResolver) HumanID(ctx context.Context, obj *identity.Interface) (string, error) {
	return (*obj).Id().Human(), nil
}

func (identityResolver) Name(ctx context.Context, obj *identity.Interface) (*string, error) {
	return nilIfEmpty((*obj).Name())
}

func (identityResolver) Email(ctx context.Context, obj *identity.Interface) (*string, error) {
	return nilIfEmpty((*obj).Email())
}

func (identityResolver) Login(ctx context.Context, obj *identity.Interface) (*string, error) {
	return nilIfEmpty((*obj).Login())
}

func (identityResolver) DisplayName(ctx context.Context, obj *identity.Interface) (string, error) {
	return (*obj).DisplayName(), nil
}

func (identityResolver) AvatarURL(ctx context.Context, obj *identity.Interface) (*string, error) {
	return nilIfEmpty((*obj).AvatarUrl())
}

func (identityResolver) IsProtected(ctx context.Context, obj *identity.Interface) (bool, error) {
	return (*obj).IsProtected(), nil
}

func nilIfEmpty(s string) (*string, error) {
	if s == "" {
		return nil, nil
	}
	return &s, nil
}
