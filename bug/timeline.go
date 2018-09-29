package bug

import "github.com/MichaelMure/git-bug/util/git"

type TimelineItem interface {
	// Hash return the hash of the item
	Hash() (git.Hash, error)
}

// CreateTimelineItem replace a Create operation in the Timeline and hold its edition history
type CreateTimelineItem struct {
	hash    git.Hash
	History []Comment
}

func NewCreateTimelineItem(hash git.Hash, comment Comment) *CreateTimelineItem {
	return &CreateTimelineItem{
		hash: hash,
		History: []Comment{
			comment,
		},
	}
}

func (c *CreateTimelineItem) Hash() (git.Hash, error) {
	return c.hash, nil
}

func (c *CreateTimelineItem) LastState() Comment {
	if len(c.History) == 0 {
		panic("no history yet")
	}

	return c.History[len(c.History)-1]
}

// CommentTimelineItem replace a Comment in the Timeline and hold its edition history
type CommentTimelineItem struct {
	hash    git.Hash
	History []Comment
}

func NewCommentTimelineItem(hash git.Hash, comment Comment) *CommentTimelineItem {
	return &CommentTimelineItem{
		hash: hash,
		History: []Comment{
			comment,
		},
	}
}

func (c *CommentTimelineItem) Hash() (git.Hash, error) {
	return c.hash, nil
}

func (c *CommentTimelineItem) LastState() Comment {
	if len(c.History) == 0 {
		panic("no history yet")
	}

	return c.History[len(c.History)-1]
}
