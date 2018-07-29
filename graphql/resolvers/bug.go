package resolvers

import (
	"context"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
)

type bugResolver struct {
	cache cache.Cacher
}

func (bugResolver) Status(ctx context.Context, obj *bug.Snapshot) (Status, error) {
	return convertStatus(obj.Status)
}

func (bugResolver) Comments(ctx context.Context, obj *bug.Snapshot, input ConnectionInput) (CommentConnection, error) {
	var connection CommentConnection

	edger := func(comment bug.Comment, offset int) Edge {
		return CommentEdge{
			Node:   comment,
			Cursor: offsetToCursor(offset),
		}
	}

	edges, pageInfo, err := BugCommentPaginate(obj.Comments, edger, input)

	if err != nil {
		return connection, err
	}

	connection.Edges = edges
	connection.PageInfo = pageInfo
	connection.TotalCount = len(obj.Comments)

	return connection, nil
}

func (bugResolver) Operations(ctx context.Context, obj *bug.Snapshot, input ConnectionInput) (OperationConnection, error) {
	var connection OperationConnection

	edger := func(op bug.Operation, offset int) Edge {
		return OperationEdge{
			Node:   op.(OperationUnion),
			Cursor: offsetToCursor(offset),
		}
	}

	edges, pageInfo, err := BugOperationPaginate(obj.Operations, edger, input)

	if err != nil {
		return connection, err
	}

	connection.Edges = edges
	connection.PageInfo = pageInfo
	connection.TotalCount = len(obj.Operations)

	return connection, nil
}
