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

func (r bugRootSubResolver) BugComment() graph.BugCommentResolver {
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

func (r bugRootSubResolver) BugEditCommentOperation() graph.BugEditCommentOperationResolver {
	return &bugEditCommentOperationResolver{}
}

func (bugRootSubResolver) BugLabelChangeOperation() graph.BugLabelChangeOperationResolver {
	return &bugLabelChangeOperationResolver{}
}

func (r bugRootSubResolver) BugLabelChangeTimelineItem() graph.BugLabelChangeTimelineItemResolver {
	return &bugLabelChangeTimelineItem{}
}

func (bugRootSubResolver) BugSetStatusOperation() graph.BugSetStatusOperationResolver {
	return &bugSetStatusOperationResolver{}
}

func (r bugRootSubResolver) BugSetStatusTimelineItem() graph.BugSetStatusTimelineItemResolver {
	return &bugSetStatusTimelineItem{}
}

func (r bugRootSubResolver) BugSetTitleOperation() graph.BugSetTitleOperationResolver {
	return &bugSetTitleOperationResolver{}
}

func (r bugRootSubResolver) BugSetTitleTimelineItem() graph.BugSetTitleTimelineItemResolver {
	return &bugSetTitleTimelineItem{}
}

func (bugRootSubResolver) BugUpdateHeadOperation() graph.BugUpdateHeadOperationResolver {
	return &bugUpdateHeadOperationResolver{}
}

func (bugRootSubResolver) BugUpdateHeadTimelineItem() graph.BugUpdateHeadTimelineItemResolver {
	return &bugUpdateHeadTimelineItem{}
}

func (bugRootSubResolver) BugAddReviewOperation() graph.BugAddReviewOperationResolver {
	return &bugAddReviewOperationResolver{}
}

func (bugRootSubResolver) BugAddReviewTimelineItem() graph.BugAddReviewTimelineItemResolver {
	return &bugAddReviewTimelineItem{}
}

func (bugRootSubResolver) BugAddReviewCommentOperation() graph.BugAddReviewCommentOperationResolver {
	return &bugAddReviewCommentOperationResolver{}
}

func (bugRootSubResolver) BugAddReviewCommentTimelineItem() graph.BugAddReviewCommentTimelineItemResolver {
	return &bugAddReviewCommentTimelineItem{}
}

func (bugRootSubResolver) Review() graph.ReviewResolver {
	return &reviewResolver{}
}

func (bugRootSubResolver) ReviewComment() graph.ReviewCommentResolver {
	return &reviewCommentResolver{}
}
