package resolvers

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/graphql/models"
)

type commentHistoryStepResolver struct{}

func (commentHistoryStepResolver) Date(ctx context.Context, obj *bug.CommentHistoryStep) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

type addCommentTimelineItemResolver struct{}

func (addCommentTimelineItemResolver) CreatedAt(ctx context.Context, obj *bug.AddCommentTimelineItem) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

func (addCommentTimelineItemResolver) LastEdit(ctx context.Context, obj *bug.AddCommentTimelineItem) (*time.Time, error) {
	t := obj.LastEdit.Time()
	return &t, nil
}

type createTimelineItemResolver struct{}

func (createTimelineItemResolver) CreatedAt(ctx context.Context, obj *bug.CreateTimelineItem) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

func (createTimelineItemResolver) LastEdit(ctx context.Context, obj *bug.CreateTimelineItem) (*time.Time, error) {
	t := obj.LastEdit.Time()
	return &t, nil
}

type labelChangeTimelineItem struct{}

func (labelChangeTimelineItem) Date(ctx context.Context, obj *bug.LabelChangeTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

type setStatusTimelineItem struct{}

func (setStatusTimelineItem) Date(ctx context.Context, obj *bug.SetStatusTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

func (setStatusTimelineItem) Status(ctx context.Context, obj *bug.SetStatusTimelineItem) (models.Status, error) {
	return convertStatus(obj.Status)
}

type setTitleTimelineItem struct{}

func (setTitleTimelineItem) Date(ctx context.Context, obj *bug.SetTitleTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}
