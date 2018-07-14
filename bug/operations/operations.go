package operations

import "encoding/gob"

// Package initialisation used to register operation's type for (de)serialization
func init() {
	gob.Register(AddCommentOperation{})
	gob.Register(CreateOperation{})
	gob.Register(SetTitleOperation{})
}
