package resolvers

import (
	"context"
	"fmt"
	"image/color"

	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/models"
	"github.com/MichaelMure/git-bug/bug"
)

var _ graph.LabelResolver = &labelResolver{}

type labelResolver struct{}

func (labelResolver) Name(ctx context.Context, obj *bug.Label) (string, error) {
	return obj.String(), nil
}

func (labelResolver) Color(ctx context.Context, obj *bug.Label) (*color.RGBA, error) {
	rgba := obj.Color().RGBA()
	return &rgba, nil
}

var _ graph.LabelChangeResultResolver = &labelChangeResultResolver{}

type labelChangeResultResolver struct{}

func (labelChangeResultResolver) Status(ctx context.Context, obj *bug.LabelChangeResult) (models.LabelChangeStatus, error) {
	switch obj.Status {
	case bug.LabelChangeAdded:
		return models.LabelChangeStatusAdded, nil
	case bug.LabelChangeRemoved:
		return models.LabelChangeStatusRemoved, nil
	case bug.LabelChangeDuplicateInOp:
		return models.LabelChangeStatusDuplicateInOp, nil
	case bug.LabelChangeAlreadySet:
		return models.LabelChangeStatusAlreadyExist, nil
	case bug.LabelChangeDoesntExist:
		return models.LabelChangeStatusDoesntExist, nil
	}

	return "", fmt.Errorf("unknown status")
}
