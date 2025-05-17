package resolvers

import (
	"context"

	"github.com/git-bug/git-bug/api/graphql/connections"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/board"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity/dag"
)

var _ graph.BoardResolver = &boardResolver{}

type boardResolver struct{}

func (boardResolver) HumanID(ctx context.Context, obj models.BoardWrapper) (string, error) {
	return obj.Id().Human(), nil
}

func (boardResolver) Columns(ctx context.Context, obj models.BoardWrapper, after *string, before *string, first *int, last *int) (*models.BoardColumnConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(column *board.Column, offset int) connections.Edge {
		return models.BoardColumnEdge{
			Node:   column,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.BoardColumnEdge, nodes []*board.Column, info *models.PageInfo, totalCount int) (*models.BoardColumnConnection, error) {
		return &models.BoardColumnConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	columns, err := obj.Columns()
	if err != nil {
		return nil, err
	}

	return connections.Connection(columns, edger, conMaker, input)
}

func (boardResolver) Actors(ctx context.Context, obj models.BoardWrapper, after *string, before *string, first *int, last *int) (*models.IdentityConnection, error) {
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

func (boardResolver) Operations(ctx context.Context, obj models.BoardWrapper, after *string, before *string, first *int, last *int) (*models.OperationConnection, error) {
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

var _ graph.BoardColumnResolver = &boardColumnResolver{}

type boardColumnResolver struct{}

func (b boardColumnResolver) Items(ctx context.Context, obj *board.Column, after *string, before *string, first *int, last *int) (*models.BoardItemConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(item board.Item, offset int) connections.Edge {
		return models.BoardItemEdge{
			Node:   item,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.BoardItemEdge, nodes []board.Item, info *models.PageInfo, totalCount int) (*models.BoardItemConnection, error) {
		return &models.BoardItemConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(obj.Items, edger, conMaker, input)
}

var _ graph.BoardItemBugResolver = &boardItemBugResolver{}

type boardItemBugResolver struct{}

func (boardItemBugResolver) Author(ctx context.Context, obj *board.BugItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BoardItemDraftResolver = &boardItemDraftResolver{}

type boardItemDraftResolver struct{}

func (boardItemDraftResolver) Author(ctx context.Context, obj *board.Draft) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

func (boardItemDraftResolver) Labels(ctx context.Context, obj *board.Draft) ([]common.Label, error) {
	return nil, nil
}
