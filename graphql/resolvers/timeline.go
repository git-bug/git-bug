package resolvers

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/graphql/models"
)

type commentHistoryStepResolver struct{}

func (commentHistoryStepResolver) Date(ctx context.Context, obj *bug.CommentHistoryStep) (time.Time, error) {
	return obj.UnixTime.Time(), nil
}

type addCommentTimelineItemResolver struct{}

func (addCommentTimelineItemResolver) CreatedAt(ctx context.Context, obj *bug.AddCommentTimelineItem) (time.Time, error) {
	return obj.CreatedAt.Time(), nil
}

func (addCommentTimelineItemResolver) LastEdit(ctx context.Context, obj *bug.AddCommentTimelineItem) (time.Time, error) {
	return obj.LastEdit.Time(), nil
}

type createTimelineItemResolver struct{}

func (createTimelineItemResolver) CreatedAt(ctx context.Context, obj *bug.CreateTimelineItem) (time.Time, error) {
	return obj.CreatedAt.Time(), nil
}

func (createTimelineItemResolver) LastEdit(ctx context.Context, obj *bug.CreateTimelineItem) (time.Time, error) {
	return obj.LastEdit.Time(), nil
}

type labelChangeTimelineItem struct{}

func (labelChangeTimelineItem) Date(ctx context.Context, obj *bug.LabelChangeTimelineItem) (time.Time, error) {
	return obj.UnixTime.Time(), nil
}

type setStatusTimelineItem struct{}

func (setStatusTimelineItem) Date(ctx context.Context, obj *bug.SetStatusTimelineItem) (time.Time, error) {
	return obj.UnixTime.Time(), nil
}

func (setStatusTimelineItem) Status(ctx context.Context, obj *bug.SetStatusTimelineItem) (models.Status, error) {
	return convertStatus(obj.Status)
}

type setTitleTimelineItem struct{}

func (setTitleTimelineItem) Date(ctx context.Context, obj *bug.SetTitleTimelineItem) (time.Time, error) {
	return obj.UnixTime.Time(), nil
}
