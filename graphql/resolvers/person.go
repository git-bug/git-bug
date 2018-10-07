package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/bug"
)

type personResolver struct{}

func (personResolver) Name(ctx context.Context, obj *bug.Person) (*string, error) {
	if obj.Name == "" {
		return nil, nil
	}
	return &obj.Name, nil
}

func (personResolver) Email(ctx context.Context, obj *bug.Person) (*string, error) {
	if obj.Email == "" {
		return nil, nil
	}
	return &obj.Email, nil
}

func (personResolver) Login(ctx context.Context, obj *bug.Person) (*string, error) {
	if obj.Login == "" {
		return nil, nil
	}
	return &obj.Login, nil
}

func (personResolver) AvatarURL(ctx context.Context, obj *bug.Person) (*string, error) {
	if obj.AvatarUrl == "" {
		return nil, nil
	}
	return &obj.AvatarUrl, nil
}
