package github

import "github.com/shurcooL/githubv4"

type ImportEvent interface {
	isImportEvent()
}

type RateLimitingEvent struct {
	msg string
}

func (RateLimitingEvent) isImportEvent() {}

type IssueEvent struct {
	issue
}

func (IssueEvent) isImportEvent() {}

type IssueEditEvent struct {
	issueId githubv4.ID
	userContentEdit
}

func (IssueEditEvent) isImportEvent() {}

type TimelineEvent struct {
	issueId githubv4.ID
	timelineItem
}

func (TimelineEvent) isImportEvent() {}

type CommentEditEvent struct {
	commentId githubv4.ID
	userContentEdit
}

func (CommentEditEvent) isImportEvent() {}
