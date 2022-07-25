package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/api/graphql/connections"
	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/models"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity/dag"
)

var _ graph.BugResolver = &bugResolver{}

type bugResolver struct{}

func (bugResolver) ID(_ context.Context, obj models.BugWrapper) (string, error) {
	return obj.Id().String(), nil
}

func (bugResolver) HumanID(_ context.Context, obj models.BugWrapper) (string, error) {
	return obj.Id().Human(), nil
}

func (bugResolver) Status(_ context.Context, obj models.BugWrapper) (models.Status, error) {
	return convertStatus(obj.Status())
}

func (bugResolver) Comments(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*models.CommentConnection, error) {
	input := connections.Input{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	comments, err := obj.Comments()
	if err != nil {
		return nil, err
	}

	conMaker := func(edges []*models.CommentEdge, nodes []bug.Comment, info *models.PageInfo, totalCount int) (*models.CommentConnection, error) {
		var commentNodes []*bug.Comment
		for _, c := range nodes {
			commentNodes = append(commentNodes, &c)
		}
		return &models.CommentConnection{
			Edges:      edges,
			Nodes:      commentNodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Paginate(comments, input)
}

func (bugResolver) Operations(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*connections.Result[dag.Operation], error) {
	input := connections.Input{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	ops, err := obj.Operations()
	if err != nil {
		return nil, err
	}

	return connections.Paginate(ops, input)
}

func (bugResolver) Timeline(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*connections.Result[bug.TimelineItem], error) {
	input := connections.Input{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	timeline, err := obj.Timeline()
	if err != nil {
		return nil, err
	}

	return connections.Paginate(timeline, input)
}

func (bugResolver) Actors(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*connections.Result[models.IdentityWrapper], error) {
	input := connections.Input{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	actors, err := obj.Actors()
	if err != nil {
		return nil, err
	}

	return connections.Paginate(actors, input)
}

func (bugResolver) Participants(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*connections.Result[models.IdentityWrapper], error) {
	input := connections.Input{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	participants, err := obj.Participants()
	if err != nil {
		return nil, err
	}

	return connections.Paginate(participants, input)
}
