package resolvers

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/bug"
)

type commentHistoryStepResolver struct{}

func (commentHistoryStepResolver) Date(ctx context.Context, obj *bug.CommentHistoryStep) (time.Time, error) {
	return obj.UnixTime.Time(), nil
}

type commentTimelineItemResolver struct{}

func (commentTimelineItemResolver) CreatedAt(ctx context.Context, obj *bug.CommentTimelineItem) (time.Time, error) {
	return obj.CreatedAt.Time(), nil
}

func (commentTimelineItemResolver) LastEdit(ctx context.Context, obj *bug.CommentTimelineItem) (time.Time, error) {
	return obj.LastEdit.Time(), nil
}

type createTimelineItemResolver struct{}

func (createTimelineItemResolver) CreatedAt(ctx context.Context, obj *bug.CreateTimelineItem) (time.Time, error) {
	return obj.CreatedAt.Time(), nil

}

func (createTimelineItemResolver) LastEdit(ctx context.Context, obj *bug.CreateTimelineItem) (time.Time, error) {
	return obj.LastEdit.Time(), nil

}
