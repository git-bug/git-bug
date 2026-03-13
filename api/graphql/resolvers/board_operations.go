package resolvers

import (
	"context"
	"fmt"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/board"
)

var _ graph.BoardAddItemDraftOperationResolver = boardAddItemDraftOperationResolver{}

type boardAddItemDraftOperationResolver struct{}

func (boardAddItemDraftOperationResolver) Author(ctx context.Context, obj *board.AddItemDraftOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BoardAddItemEntityOperationResolver = boardAddItemEntityOperationResolver{}

type boardAddItemEntityOperationResolver struct{}

func (boardAddItemEntityOperationResolver) Author(ctx context.Context, obj *board.AddItemEntityOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

func (boardAddItemEntityOperationResolver) EntityType(ctx context.Context, obj *board.AddItemEntityOperation) (models.BoardItemEntityType, error) {
	switch obj.EntityType {
	case board.EntityTypeBug:
		return models.BoardItemEntityTypeBug, nil
	default:
		return "", fmt.Errorf("unknown entity type: %s", obj.EntityType)
	}
}

var _ graph.BoardCreateOperationResolver = boardCreateOperationResolver{}

type boardCreateOperationResolver struct{}

func (boardCreateOperationResolver) Author(ctx context.Context, obj *board.CreateOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BoardSetDescriptionOperationResolver = boardSetDescriptionOperationResolver{}

type boardSetDescriptionOperationResolver struct{}

func (boardSetDescriptionOperationResolver) Author(ctx context.Context, obj *board.SetDescriptionOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}

var _ graph.BoardSetTitleOperationResolver = boardSetTitleOperationResolver{}

type boardSetTitleOperationResolver struct{}

func (boardSetTitleOperationResolver) Author(ctx context.Context, obj *board.SetTitleOperation) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author()), nil
}
