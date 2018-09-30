package bug

import (
	"github.com/MichaelMure/git-bug/util/git"
)

type TimelineItem interface {
	// Hash return the hash of the item
	Hash() (git.Hash, error)
}

type CommentHistoryStep struct {
	Message  string
	UnixTime Timestamp
}

// CreateTimelineItem replace a Create operation in the Timeline and hold its edition history
type CreateTimelineItem struct {
	CommentTimelineItem
}

func NewCreateTimelineItem(hash git.Hash, comment Comment) *CreateTimelineItem {
	return &CreateTimelineItem{
		CommentTimelineItem: CommentTimelineItem{
			hash:      hash,
			Author:    comment.Author,
			Message:   comment.Message,
			Files:     comment.Files,
			CreatedAt: comment.UnixTime,
			LastEdit:  comment.UnixTime,
			History: []CommentHistoryStep{
				{
					Message:  comment.Message,
					UnixTime: comment.UnixTime,
				},
			},
		},
	}
}

// CommentTimelineItem replace a Comment in the Timeline and hold its edition history
type CommentTimelineItem struct {
	hash      git.Hash
	Author    Person
	Message   string
	Files     []git.Hash
	CreatedAt Timestamp
	LastEdit  Timestamp
	History   []CommentHistoryStep
}

func NewCommentTimelineItem(hash git.Hash, comment Comment) *CommentTimelineItem {
	return &CommentTimelineItem{
		hash:      hash,
		Author:    comment.Author,
		Message:   comment.Message,
		Files:     comment.Files,
		CreatedAt: comment.UnixTime,
		LastEdit:  comment.UnixTime,
		History: []CommentHistoryStep{
			{
				Message:  comment.Message,
				UnixTime: comment.UnixTime,
			},
		},
	}
}

func (c *CommentTimelineItem) Hash() (git.Hash, error) {
	return c.hash, nil
}

// Append will append a new comment in the history and update the other values
func (c *CommentTimelineItem) Append(comment Comment) {
	c.Message = comment.Message
	c.Files = comment.Files
	c.LastEdit = comment.UnixTime
	c.History = append(c.History, CommentHistoryStep{
		Message:  comment.Message,
		UnixTime: comment.UnixTime,
	})
}

// Edited say if the comment was edited
func (c *CommentTimelineItem) Edited() bool {
	return len(c.History) > 1
}
