package resolvers

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entity"
)

var _ graph.CommentHistoryStepResolver = commentHistoryStepResolver{}

type commentHistoryStepResolver struct{}

func (commentHistoryStepResolver) Date(_ context.Context, obj *bug.CommentHistoryStep) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

var _ graph.AddCommentTimelineItemResolver = addCommentTimelineItemResolver{}

type addCommentTimelineItemResolver struct{}

func (addCommentTimelineItemResolver) ID(_ context.Context, obj *bug.AddCommentTimelineItem) (entity.CombinedId, error) {
	return obj.CombinedId(), nil
}

func (addCommentTimelineItemResolver) Author(_ context.Context, obj *bug.AddCommentTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (addCommentTimelineItemResolver) CreatedAt(_ context.Context, obj *bug.AddCommentTimelineItem) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

func (addCommentTimelineItemResolver) LastEdit(_ context.Context, obj *bug.AddCommentTimelineItem) (*time.Time, error) {
	t := obj.LastEdit.Time()
	return &t, nil
}

var _ graph.CreateTimelineItemResolver = createTimelineItemResolver{}

type createTimelineItemResolver struct{}

func (createTimelineItemResolver) ID(_ context.Context, obj *bug.CreateTimelineItem) (entity.CombinedId, error) {
	return obj.CombinedId(), nil
}

func (r createTimelineItemResolver) Author(_ context.Context, obj *bug.CreateTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (createTimelineItemResolver) CreatedAt(_ context.Context, obj *bug.CreateTimelineItem) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

func (createTimelineItemResolver) LastEdit(_ context.Context, obj *bug.CreateTimelineItem) (*time.Time, error) {
	t := obj.LastEdit.Time()
	return &t, nil
}

var _ graph.LabelChangeTimelineItemResolver = labelChangeTimelineItem{}

type labelChangeTimelineItem struct{}

func (labelChangeTimelineItem) ID(_ context.Context, obj *bug.LabelChangeTimelineItem) (entity.CombinedId, error) {
	return obj.CombinedId(), nil
}

func (i labelChangeTimelineItem) Author(_ context.Context, obj *bug.LabelChangeTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (labelChangeTimelineItem) Date(_ context.Context, obj *bug.LabelChangeTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

var _ graph.SetStatusTimelineItemResolver = setStatusTimelineItem{}

type setStatusTimelineItem struct{}

func (setStatusTimelineItem) ID(_ context.Context, obj *bug.SetStatusTimelineItem) (entity.CombinedId, error) {
	return obj.CombinedId(), nil
}

func (i setStatusTimelineItem) Author(_ context.Context, obj *bug.SetStatusTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (setStatusTimelineItem) Date(_ context.Context, obj *bug.SetStatusTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

var _ graph.SetTitleTimelineItemResolver = setTitleTimelineItem{}

type setTitleTimelineItem struct{}

func (setTitleTimelineItem) ID(_ context.Context, obj *bug.SetTitleTimelineItem) (entity.CombinedId, error) {
	return obj.CombinedId(), nil
}

func (i setTitleTimelineItem) Author(_ context.Context, obj *bug.SetTitleTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (setTitleTimelineItem) Date(_ context.Context, obj *bug.SetTitleTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}
