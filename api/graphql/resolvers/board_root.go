package resolvers

import "github.com/git-bug/git-bug/api/graphql/graph"

type boardRootSubResolver struct{}

func (boardRootSubResolver) BoardColumn() graph.BoardColumnResolver {
	return &boardColumnResolver{}
}

func (boardRootSubResolver) BoardItemBug() graph.BoardItemBugResolver {
	return &boardItemBugResolver{}
}

func (boardRootSubResolver) BoardItemDraft() graph.BoardItemDraftResolver {
	return &boardItemDraftResolver{}
}

func (boardRootSubResolver) BoardAddItemDraftOperation() graph.BoardAddItemDraftOperationResolver {
	return &boardAddItemDraftOperationResolver{}
}

func (boardRootSubResolver) BoardAddItemEntityOperation() graph.BoardAddItemEntityOperationResolver {
	return &boardAddItemEntityOperationResolver{}
}

func (boardRootSubResolver) BoardCreateOperation() graph.BoardCreateOperationResolver {
	return &boardCreateOperationResolver{}
}

func (boardRootSubResolver) BoardSetDescriptionOperation() graph.BoardSetDescriptionOperationResolver {
	return &boardSetDescriptionOperationResolver{}
}

func (boardRootSubResolver) BoardSetTitleOperation() graph.BoardSetTitleOperationResolver {
	return &boardSetTitleOperationResolver{}
}
