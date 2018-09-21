package main

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bridge/github"
)

func main() {
	conf, err := github.Configure()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(conf)
}
