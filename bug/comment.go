package bug

import "github.com/MichaelMure/git-bug/util"

type Comment struct {
	Author  Person
	Message string
	Media   []util.Hash
}
