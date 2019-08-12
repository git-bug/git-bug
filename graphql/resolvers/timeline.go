package resolvers

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/models"
)

var _ graph.CommentHistoryStepResolver = commentHistoryStepResolver{}

type commentHistoryStepResolver struct{}

func (commentHistoryStepResolver) Date(ctx context.Context, obj *bug.CommentHistoryStep) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

var _ graph.AddCommentTimelineItemResolver = addCommentTimelineItemResolver{}

type addCommentTimelineItemResolver struct{}

func (addCommentTimelineItemResolver) ID(ctx context.Context, obj *bug.AddCommentTimelineItem) (string, error) {
	return obj.Id().String(), nil
}

func (addCommentTimelineItemResolver) CreatedAt(ctx context.Context, obj *bug.AddCommentTimelineItem) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

func (addCommentTimelineItemResolver) LastEdit(ctx context.Context, obj *bug.AddCommentTimelineItem) (*time.Time, error) {
	t := obj.LastEdit.Time()
	return &t, nil
}

var _ graph.CreateTimelineItemResolver = createTimelineItemResolver{}

type createTimelineItemResolver struct{}

func (createTimelineItemResolver) ID(ctx context.Context, obj *bug.CreateTimelineItem) (string, error) {
	return obj.Id().String(), nil
}

func (createTimelineItemResolver) CreatedAt(ctx context.Context, obj *bug.CreateTimelineItem) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

func (createTimelineItemResolver) LastEdit(ctx context.Context, obj *bug.CreateTimelineItem) (*time.Time, error) {
	t := obj.LastEdit.Time()
	return &t, nil
}

var _ graph.LabelChangeTimelineItemResolver = labelChangeTimelineItem{}

type labelChangeTimelineItem struct{}

func (labelChangeTimelineItem) ID(ctx context.Context, obj *bug.LabelChangeTimelineItem) (string, error) {
	return obj.Id().String(), nil
}

func (labelChangeTimelineItem) Date(ctx context.Context, obj *bug.LabelChangeTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

var _ graph.SetStatusTimelineItemResolver = setStatusTimelineItem{}

type setStatusTimelineItem struct{}

func (setStatusTimelineItem) ID(ctx context.Context, obj *bug.SetStatusTimelineItem) (string, error) {
	return obj.Id().String(), nil
}

func (setStatusTimelineItem) Date(ctx context.Context, obj *bug.SetStatusTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

func (setStatusTimelineItem) Status(ctx context.Context, obj *bug.SetStatusTimelineItem) (models.Status, error) {
	return convertStatus(obj.Status)
}

var _ graph.SetTitleTimelineItemResolver = setTitleTimelineItem{}

type setTitleTimelineItem struct{}

func (setTitleTimelineItem) ID(ctx context.Context, obj *bug.SetTitleTimelineItem) (string, error) {
	return obj.Id().String(), nil
}

func (setTitleTimelineItem) Date(ctx context.Context, obj *bug.SetTitleTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}
