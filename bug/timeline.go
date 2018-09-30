package bug

import (
	"github.com/MichaelMure/git-bug/util/git"
)

type TimelineItem interface {
	// Hash return the hash of the item
	Hash() git.Hash
}

type CommentHistoryStep struct {
	Message  string
	UnixTime Timestamp
}

// CommentTimelineItem is a TimelineItem that holds a Comment and its edition history
type CommentTimelineItem struct {
	hash      git.Hash
	Author    Person
	Message   string
	Files     []git.Hash
	CreatedAt Timestamp
	LastEdit  Timestamp
	History   []CommentHistoryStep
}

func NewCommentTimelineItem(hash git.Hash, comment Comment) CommentTimelineItem {
	return CommentTimelineItem{
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

func (c *CommentTimelineItem) Hash() git.Hash {
	return c.hash
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
