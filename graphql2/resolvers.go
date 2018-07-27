package graphql2

import (
	"context"
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql2/gen"
	"time"
)

type Backend struct {
	cache cache.RootCache
}

func (*Backend) Bug_labels(ctx context.Context, obj *bug.Snapshot) ([]*bug.Label, error) {
	return obj.Labels
}

func (*Backend) LabelChangeOperation_added(ctx context.Context, obj *operations.LabelChangeOperation) ([]*bug.Label, error) {
	panic("implement me")
}

func (*Backend) LabelChangeOperation_removed(ctx context.Context, obj *operations.LabelChangeOperation) ([]*bug.Label, error) {
	panic("implement me")
}

func (*Backend) AddCommentOperation_date(ctx context.Context, obj *operations.AddCommentOperation) (time.Time, error) {
	return obj.Time(), nil
}

func (*Backend) Bug_status(ctx context.Context, obj *bug.Snapshot) (gen.Status, error) {
	return convertStatus(obj.Status)
}

func (*Backend) Bug_comments(ctx context.Context, obj *bug.Snapshot, after *string, before *string, first *int, last *int, query *string) (gen.CommentConnection, error) {
	panic("implement me")
}

func (*Backend) Bug_operations(ctx context.Context, obj *bug.Snapshot, after *string, before *string, first *int, last *int, query *string) (gen.OperationConnection, error) {
	panic("implement me")
}

func (*Backend) CreateOperation_date(ctx context.Context, obj *operations.CreateOperation) (time.Time, error) {
	return obj.Time(), nil
}

func (*Backend) LabelChangeOperation_date(ctx context.Context, obj *operations.LabelChangeOperation) (time.Time, error) {
	return obj.Time(), nil
}

func (*Backend) RootQuery_allBugs(ctx context.Context, after *string, before *string, first *int, last *int, query *string) (gen.BugConnection, error) {
	panic("implement me")
}

func (*Backend) RootQuery_bug(ctx context.Context, id string) (*bug.Snapshot, error) {
	panic("implement me")
}

func (*Backend) SetStatusOperation_date(ctx context.Context, obj *operations.SetStatusOperation) (time.Time, error) {
	return obj.Time(), nil
}

func (*Backend) SetStatusOperation_status(ctx context.Context, obj *operations.SetStatusOperation) (gen.Status, error) {
	return convertStatus(obj.Status)
}

func (*Backend) SetTitleOperation_date(ctx context.Context, obj *operations.SetTitleOperation) (time.Time, error) {
	return obj.Time(), nil
}

func convertStatus(status bug.Status) (gen.Status, error) {
	switch status {
	case bug.OpenStatus:
		return gen.StatusOpen, nil
	case bug.ClosedStatus:
		return gen.StatusClosed, nil
	}

	return "", fmt.Errorf("Unknown status")
}
