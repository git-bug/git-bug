package operations

import (
	"github.com/MichaelMure/git-bug/bug"
)

// Package initialisation used to register operation's type for (de)serialization
func init() {
	bug.Register(bug.CreateOp, CreateOperation{})
	bug.Register(bug.SetTitleOp, SetTitleOperation{})
	bug.Register(bug.AddCommentOp, AddCommentOperation{})
	bug.Register(bug.SetStatusOp, SetStatusOperation{})
	bug.Register(bug.LabelChangeOp, LabelChangeOperation{})
}
