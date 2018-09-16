// Package operations contains the various bug operations. A bug operation is
// an atomic edit operation of a bug state. These operations are applied
// sequentially to compile the current state of the bug.
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
