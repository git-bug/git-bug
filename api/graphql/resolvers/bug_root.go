package resolvers

import (
	"github.com/git-bug/git-bug/api/graphql/graph"
)

type bugRootSubResolver struct{}

func (bugRootSubResolver) BugAddCommentOperation() graph.BugAddCommentOperationResolver {
	return &bugAddCommentOperationResolver{}
}

func (bugRootSubResolver) BugAddCommentTimelineItem() graph.BugAddCommentTimelineItemResolver {
	return &bugAddCommentTimelineItemResolver{}
}

func (bugRootSubResolver) BugComment() graph.BugCommentResolver {
	return &commentResolver{}
}

func (bugRootSubResolver) BugCommentHistoryStep() graph.BugCommentHistoryStepResolver {
	return &bugCommentHistoryStepResolver{}
}

func (bugRootSubResolver) BugCreateOperation() graph.BugCreateOperationResolver {
	return &bugCreateOperationResolver{}
}

func (bugRootSubResolver) BugCreateTimelineItem() graph.BugCreateTimelineItemResolver {
	return &bugCreateTimelineItemResolver{}
}

func (bugRootSubResolver) BugEditCommentOperation() graph.BugEditCommentOperationResolver {
	return &bugEditCommentOperationResolver{}
}

func (bugRootSubResolver) BugLabelChangeOperation() graph.BugLabelChangeOperationResolver {
	return &bugLabelChangeOperationResolver{}
}

func (bugRootSubResolver) BugLabelChangeTimelineItem() graph.BugLabelChangeTimelineItemResolver {
	return &bugLabelChangeTimelineItem{}
}

func (bugRootSubResolver) BugSetStatusOperation() graph.BugSetStatusOperationResolver {
	return &bugSetStatusOperationResolver{}
}

func (bugRootSubResolver) BugSetStatusTimelineItem() graph.BugSetStatusTimelineItemResolver {
	return &bugSetStatusTimelineItem{}
}

func (bugRootSubResolver) BugSetTitleOperation() graph.BugSetTitleOperationResolver {
	return &bugSetTitleOperationResolver{}
}

func (bugRootSubResolver) BugSetTitleTimelineItem() graph.BugSetTitleTimelineItemResolver {
	return &bugSetTitleTimelineItem{}
}
