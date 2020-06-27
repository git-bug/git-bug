package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/api/graphql/connections"
	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/models"
	"github.com/MichaelMure/git-bug/bug"
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
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(comment bug.Comment, offset int) connections.Edge {
		return models.CommentEdge{
			Node:   &comment,
			Cursor: connections.OffsetToCursor(offset),
		}
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

	comments, err := obj.Comments()
	if err != nil {
		return nil, err
	}

	return connections.CommentCon(comments, edger, conMaker, input)
}

func (bugResolver) Operations(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*models.OperationConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(op bug.Operation, offset int) connections.Edge {
		return models.OperationEdge{
			Node:   op,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.OperationEdge, nodes []bug.Operation, info *models.PageInfo, totalCount int) (*models.OperationConnection, error) {
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

	return connections.OperationCon(ops, edger, conMaker, input)
}

func (bugResolver) Timeline(_ context.Context, obj models.BugWrapper, after *string, before *string, first *int, last *int) (*models.TimelineItemConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(op bug.TimelineItem, offset int) connections.Edge {
		return models.TimelineItemEdge{
			Node:   op,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.TimelineItemEdge, nodes []bug.TimelineItem, info *models.PageInfo, totalCount int) (*models.TimelineItemConnection, error) {
		return &models.TimelineItemConnection{
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

	return connections.TimelineItemCon(timeline, edger, conMaker, input)
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

	return connections.IdentityCon(actors, edger, conMaker, input)
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

	return connections.IdentityCon(participants, edger, conMaker, input)
}
