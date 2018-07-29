package resolvers

import (
	"github.com/MichaelMure/git-bug/cache"
)

type RootResolver struct {
	cache.RootCache
}

func NewRootResolver() *RootResolver {
	return &RootResolver{
		RootCache: cache.NewCache(),
	}
}

func (r RootResolver) Query() QueryResolver {
	return &rootQueryResolver{
		cache: &r.RootCache,
	}
}

func (RootResolver) AddCommentOperation() AddCommentOperationResolver {
	return &addCommentOperationResolver{}
}

func (r RootResolver) Bug() BugResolver {
	return &bugResolver{
		cache: &r.RootCache,
	}
}

func (RootResolver) CreateOperation() CreateOperationResolver {
	return &createOperationResolver{}
}

func (RootResolver) LabelChangeOperation() LabelChangeOperationResolver {
	return &labelChangeOperation{}
}

func (r RootResolver) Repository() RepositoryResolver {
	return &repoResolver{
		cache: &r.RootCache,
	}
}

func (RootResolver) SetStatusOperation() SetStatusOperationResolver {
	return &setStatusOperationResolver{}
}

func (RootResolver) SetTitleOperation() SetTitleOperationResolver {
	return &setTitleOperationResolver{}
}
