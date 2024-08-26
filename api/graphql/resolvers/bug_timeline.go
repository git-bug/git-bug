package resolvers

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/bug"
)

var _ graph.BugCommentHistoryStepResolver = bugCommentHistoryStepResolver{}

type bugCommentHistoryStepResolver struct{}

func (bugCommentHistoryStepResolver) Date(_ context.Context, obj *bug.CommentHistoryStep) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

var _ graph.BugAddCommentTimelineItemResolver = bugAddCommentTimelineItemResolver{}

type bugAddCommentTimelineItemResolver struct{}

func (bugAddCommentTimelineItemResolver) Author(_ context.Context, obj *bug.AddCommentTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (bugAddCommentTimelineItemResolver) CreatedAt(_ context.Context, obj *bug.AddCommentTimelineItem) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

func (bugAddCommentTimelineItemResolver) LastEdit(_ context.Context, obj *bug.AddCommentTimelineItem) (*time.Time, error) {
	t := obj.LastEdit.Time()
	return &t, nil
}

var _ graph.BugCreateTimelineItemResolver = bugCreateTimelineItemResolver{}

type bugCreateTimelineItemResolver struct{}

func (r bugCreateTimelineItemResolver) Author(_ context.Context, obj *bug.CreateTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (bugCreateTimelineItemResolver) CreatedAt(_ context.Context, obj *bug.CreateTimelineItem) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

func (bugCreateTimelineItemResolver) LastEdit(_ context.Context, obj *bug.CreateTimelineItem) (*time.Time, error) {
	t := obj.LastEdit.Time()
	return &t, nil
}

var _ graph.BugLabelChangeTimelineItemResolver = bugLabelChangeTimelineItem{}

type bugLabelChangeTimelineItem struct{}

func (i bugLabelChangeTimelineItem) Author(_ context.Context, obj *bug.LabelChangeTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (bugLabelChangeTimelineItem) Date(_ context.Context, obj *bug.LabelChangeTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

var _ graph.BugSetStatusTimelineItemResolver = bugSetStatusTimelineItem{}

type bugSetStatusTimelineItem struct{}

func (i bugSetStatusTimelineItem) Author(_ context.Context, obj *bug.SetStatusTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (bugSetStatusTimelineItem) Date(_ context.Context, obj *bug.SetStatusTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}

var _ graph.BugSetTitleTimelineItemResolver = bugSetTitleTimelineItem{}

type bugSetTitleTimelineItem struct{}

func (i bugSetTitleTimelineItem) Author(_ context.Context, obj *bug.SetTitleTimelineItem) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (bugSetTitleTimelineItem) Date(_ context.Context, obj *bug.SetTitleTimelineItem) (*time.Time, error) {
	t := obj.UnixTime.Time()
	return &t, nil
}
