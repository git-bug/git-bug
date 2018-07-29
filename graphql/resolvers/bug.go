package resolvers

import (
	"context"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/graphql/connections"
	"github.com/MichaelMure/git-bug/graphql/models"
)

type bugResolver struct{}

func (bugResolver) Status(ctx context.Context, obj *bug.Snapshot) (models.Status, error) {
	return convertStatus(obj.Status)
}

func (bugResolver) Comments(ctx context.Context, obj *bug.Snapshot, input models.ConnectionInput) (models.CommentConnection, error) {
	edger := func(comment bug.Comment, offset int) connections.Edge {
		return models.CommentEdge{
			Node:   comment,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []models.CommentEdge, info models.PageInfo, totalCount int) models.CommentConnection {
		return models.CommentConnection{
			Edges:      edges,
			PageInfo:   info,
			TotalCount: totalCount,
		}
	}

	return connections.BugCommentCon(obj.Comments, edger, conMaker, input)
}

func (bugResolver) Operations(ctx context.Context, obj *bug.Snapshot, input models.ConnectionInput) (models.OperationConnection, error) {
	edger := func(op bug.Operation, offset int) connections.Edge {
		return models.OperationEdge{
			Node:   op.(models.OperationUnion),
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []models.OperationEdge, info models.PageInfo, totalCount int) models.OperationConnection {
		return models.OperationConnection{
			Edges:      edges,
			PageInfo:   info,
			TotalCount: totalCount,
		}
	}

	return connections.BugOperationCon(obj.Operations, edger, conMaker, input)
}
