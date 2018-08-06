// +build ignore

package main

import (
	"fmt"
	"github.com/MichaelMure/git-bug/commands"
	"log"
	"os"
	"path"
)

func main() {
	cwd, _ := os.Getwd()
	filepath := path.Join(cwd, "misc", "bash_completion", "git-bug")

	fmt.Println("Generating bash completion file ...")

	//git := &cobra.Command{
	//	Use: "git",
	//	BashCompletionFunction: "qsdhjlkqsdhlsd",
	//}
	//
	//bug := &cobra.Command{
	//	Use: "bug",
	//	BashCompletionFunction: "ZHZLDHKLZDHJKL",
	//}
	//git.AddCommand(bug)

	//for _, sub := range commands.RootCmd.Commands() {
	//	bug.AddCommand(sub)
	//}

	//err := git.GenBashCompletionFile(filepath)
	err := commands.RootCmd.GenBashCompletionFile(filepath)
	if err != nil {
		log.Fatal(err)
	}
}
