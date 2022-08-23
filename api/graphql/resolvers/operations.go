package resolvers

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/models"
	"github.com/MichaelMure/git-bug/entities/bug"
)

var _ graph.CreateOperationResolver = createOperationResolver{}

type createOperationResolver struct{}

func (createOperationResolver) Author(_ context.Context, obj *bug.CreateOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

func (createOperationResolver) Date(_ context.Context, obj *bug.CreateOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

var _ graph.AddCommentOperationResolver = addCommentOperationResolver{}

type addCommentOperationResolver struct{}

func (addCommentOperationResolver) Author(_ context.Context, obj *bug.AddCommentOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

func (addCommentOperationResolver) Date(_ context.Context, obj *bug.AddCommentOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

var _ graph.EditCommentOperationResolver = editCommentOperationResolver{}

type editCommentOperationResolver struct{}

func (editCommentOperationResolver) Target(_ context.Context, obj *bug.EditCommentOperation) (string, error) {
	return obj.Target.String(), nil
}

func (editCommentOperationResolver) Author(_ context.Context, obj *bug.EditCommentOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

func (editCommentOperationResolver) Date(_ context.Context, obj *bug.EditCommentOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

var _ graph.LabelChangeOperationResolver = labelChangeOperationResolver{}

type labelChangeOperationResolver struct{}

func (labelChangeOperationResolver) Author(_ context.Context, obj *bug.LabelChangeOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

func (labelChangeOperationResolver) Date(_ context.Context, obj *bug.LabelChangeOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

var _ graph.SetStatusOperationResolver = setStatusOperationResolver{}

type setStatusOperationResolver struct{}

func (setStatusOperationResolver) Author(_ context.Context, obj *bug.SetStatusOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

func (setStatusOperationResolver) Date(_ context.Context, obj *bug.SetStatusOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

var _ graph.SetTitleOperationResolver = setTitleOperationResolver{}

type setTitleOperationResolver struct{}

func (setTitleOperationResolver) Author(_ context.Context, obj *bug.SetTitleOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

func (setTitleOperationResolver) Date(_ context.Context, obj *bug.SetTitleOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}
