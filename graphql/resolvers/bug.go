package resolvers

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/graphql/connections"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/models"
	"github.com/MichaelMure/git-bug/identity"
)

var _ graph.BugResolver = &bugResolver{}

type bugResolver struct{}

func (bugResolver) Status(ctx context.Context, obj *bug.Snapshot) (models.Status, error) {
	return convertStatus(obj.Status)
}

func (bugResolver) Comments(ctx context.Context, obj *bug.Snapshot, after *string, before *string, first *int, last *int) (models.CommentConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(comment bug.Comment, offset int) connections.Edge {
		return models.CommentEdge{
			Node:   comment,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []models.CommentEdge, nodes []bug.Comment, info models.PageInfo, totalCount int) (models.CommentConnection, error) {
		return models.CommentConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.CommentCon(obj.Comments, edger, conMaker, input)
}

func (bugResolver) Operations(ctx context.Context, obj *bug.Snapshot, after *string, before *string, first *int, last *int) (models.OperationConnection, error) {
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

	conMaker := func(edges []models.OperationEdge, nodes []bug.Operation, info models.PageInfo, totalCount int) (models.OperationConnection, error) {
		return models.OperationConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.OperationCon(obj.Operations, edger, conMaker, input)
}

func (bugResolver) Timeline(ctx context.Context, obj *bug.Snapshot, after *string, before *string, first *int, last *int) (models.TimelineItemConnection, error) {
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

	conMaker := func(edges []models.TimelineItemEdge, nodes []bug.TimelineItem, info models.PageInfo, totalCount int) (models.TimelineItemConnection, error) {
		return models.TimelineItemConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.TimelineItemCon(obj.Timeline, edger, conMaker, input)
}

func (bugResolver) LastEdit(ctx context.Context, obj *bug.Snapshot) (time.Time, error) {
	return obj.LastEditTime(), nil
}

func (bugResolver) Actors(ctx context.Context, obj *bug.Snapshot) ([]*identity.Interface, error) {
	actorsp := make([]*identity.Interface, len(obj.Actors))

	for i, actor := range obj.Actors {
		actorsp[i] = &actor
	}

	return actorsp, nil
}

func (bugResolver) Participants(ctx context.Context, obj *bug.Snapshot) ([]*identity.Interface, error) {
	participantsp := make([]*identity.Interface, len(obj.Participants))

	for i, participant := range obj.Participants {
		participantsp[i] = &participant
	}

	return participantsp, nil
}
