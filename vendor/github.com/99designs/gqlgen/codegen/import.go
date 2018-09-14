package codegen

import (
	"strconv"
)

type Import struct {
	Name string
	Path string

	alias string
}

type Imports struct {
	imports []*Import
	destDir string
}

func (i *Import) Write() string {
	return i.Alias() + " " + strconv.Quote(i.Path)
}

func (i *Import) Alias() string {
	if i.alias == "" {
		panic("alias called before imports are finalized")
	}

	return i.alias
}
