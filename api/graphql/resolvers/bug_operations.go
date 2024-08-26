package resolvers

import (
	"context"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/bug"
)

var _ graph.BugCreateOperationResolver = bugCreateOperationResolver{}

type bugCreateOperationResolver struct{}

func (bugCreateOperationResolver) Author(_ context.Context, obj *bug.CreateOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BugAddCommentOperationResolver = bugAddCommentOperationResolver{}

type bugAddCommentOperationResolver struct{}

func (bugAddCommentOperationResolver) Author(_ context.Context, obj *bug.AddCommentOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BugEditCommentOperationResolver = bugEditCommentOperationResolver{}

type bugEditCommentOperationResolver struct{}

func (bugEditCommentOperationResolver) Target(_ context.Context, obj *bug.EditCommentOperation) (string, error) {
	return obj.Target.String(), nil
}

func (bugEditCommentOperationResolver) Author(_ context.Context, obj *bug.EditCommentOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BugLabelChangeOperationResolver = bugLabelChangeOperationResolver{}

type bugLabelChangeOperationResolver struct{}

func (bugLabelChangeOperationResolver) Author(_ context.Context, obj *bug.LabelChangeOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BugSetStatusOperationResolver = bugSetStatusOperationResolver{}

type bugSetStatusOperationResolver struct{}

func (bugSetStatusOperationResolver) Author(_ context.Context, obj *bug.SetStatusOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BugSetTitleOperationResolver = bugSetTitleOperationResolver{}

type bugSetTitleOperationResolver struct{}

func (bugSetTitleOperationResolver) Author(_ context.Context, obj *bug.SetTitleOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}
