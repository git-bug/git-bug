package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/graphql/models"
)

type createOperationResolver struct{}

func (createOperationResolver) Date(ctx context.Context, obj *bug.CreateOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

type addCommentOperationResolver struct{}

func (addCommentOperationResolver) Date(ctx context.Context, obj *bug.AddCommentOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

type editCommentOperationResolver struct{}

func (editCommentOperationResolver) Date(ctx context.Context, obj *bug.EditCommentOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

type labelChangeOperation struct{}

func (labelChangeOperation) Date(ctx context.Context, obj *bug.LabelChangeOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

type setStatusOperationResolver struct{}

func (setStatusOperationResolver) Date(ctx context.Context, obj *bug.SetStatusOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

func (setStatusOperationResolver) Status(ctx context.Context, obj *bug.SetStatusOperation) (models.Status, error) {
	return convertStatus(obj.Status)
}

type setTitleOperationResolver struct{}

func (setTitleOperationResolver) Date(ctx context.Context, obj *bug.SetTitleOperation) (*time.Time, error) {
	t := obj.Time()
	return &t, nil
}

func convertStatus(status bug.Status) (models.Status, error) {
	switch status {
	case bug.OpenStatus:
		return models.StatusOpen, nil
	case bug.ClosedStatus:
		return models.StatusClosed, nil
	}

	return "", fmt.Errorf("Unknown status")
}
