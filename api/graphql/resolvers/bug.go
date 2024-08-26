package resolvers

import (
	"context"

	"github.com/git-bug/git-bug/api/graphql/connections"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entity/dag"
)

var _ graph.BugResolver = &bugResolver{}

type bugResolver struct{}

func (bugResolver) HumanID(_ context.Context, obj models.BugWrapper) (string, error) {
	return obj.Id().Human(), nil
}

func (bugResolver) Comments(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*models.BugCommentConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(comment bug.Comment, offset int) connections.Edge {
		return models.BugCommentEdge{
			Node:   &comment,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.BugCommentEdge, nodes []bug.Comment, info *models.PageInfo, totalCount int) (*models.BugCommentConnection, error) {
		var commentNodes []*bug.Comment
		for _, c := range nodes {
			commentNodes = append(commentNodes, &c)
		}
		return &models.BugCommentConnection{
			Edges:      edges,
			Nodes:      commentNodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	comments, err := obj.Comments()
	if err != nil {
		return nil, err
	}

	return connections.Connection(comments, edger, conMaker, input)
}

func (bugResolver) Operations(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*models.OperationConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(op dag.Operation, offset int) connections.Edge {
		return models.OperationEdge{
			Node:   op,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.OperationEdge, nodes []dag.Operation, info *models.PageInfo, totalCount int) (*models.OperationConnection, error) {
		return &models.OperationConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	ops, err := obj.Operations()
	if err != nil {
		return nil, err
	}

	return connections.Connection(ops, edger, conMaker, input)
}

func (bugResolver) Timeline(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*models.BugTimelineItemConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(op bug.TimelineItem, offset int) connections.Edge {
		return models.BugTimelineItemEdge{
			Node:   op,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.BugTimelineItemEdge, nodes []bug.TimelineItem, info *models.PageInfo, totalCount int) (*models.BugTimelineItemConnection, error) {
		return &models.BugTimelineItemConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	timeline, err := obj.Timeline()
	if err != nil {
		return nil, err
	}

	return connections.Connection(timeline, edger, conMaker, input)
}

func (bugResolver) Actors(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*models.IdentityConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(actor models.IdentityWrapper, offset int) connections.Edge {
		return models.IdentityEdge{
			Node:   actor,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.IdentityEdge, nodes []models.IdentityWrapper, info *models.PageInfo, totalCount int) (*models.IdentityConnection, error) {
		return &models.IdentityConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	actors, err := obj.Actors()
	if err != nil {
		return nil, err
	}

	return connections.Connection(actors, edger, conMaker, input)
}

func (bugResolver) Participants(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*models.IdentityConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(participant models.IdentityWrapper, offset int) connections.Edge {
		return models.IdentityEdge{
			Node:   participant,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.IdentityEdge, nodes []models.IdentityWrapper, info *models.PageInfo, totalCount int) (*models.IdentityConnection, error) {
		return &models.IdentityConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	participants, err := obj.Participants()
	if err != nil {
		return nil, err
	}

	return connections.Connection(participants, edger, conMaker, input)
}

var _ graph.BugCommentResolver = &commentResolver{}

type commentResolver struct{}

func (c commentResolver) Author(_ context.Context, obj *bug.Comment) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}
