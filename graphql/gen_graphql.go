// +build ignore

package main

import (
	"fmt"

	"github.com/99designs/gqlgen/cmd"
)

func main() {
	fmt.Println("Generating graphql code ...")

	cmd.Execute()
}
