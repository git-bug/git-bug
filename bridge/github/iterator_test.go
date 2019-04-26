package github

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func Test_Iterator(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	user := os.Getenv("GITHUB_USER")
	project := os.Getenv("GITHUB_PROJECT")

	i := newIterator(map[string]string{
		keyToken:  token,
		"user":    user,
		"project": project,
	}, time.Time{})
	//time.Now().Add(-14*24*time.Hour))

	for i.NextIssue() {
		v := i.IssueValue()
		fmt.Printf("   issue = id:%v title:%v\n", v.Id, v.Title)

		for i.NextIssueEdit() {
			v := i.IssueEditValue()
			fmt.Printf("issue edit = %v\n", string(*v.Diff))
		}

		for i.NextTimeline() {
			v := i.TimelineValue()
			fmt.Printf("timeline = type:%v\n", v.Typename)

			if v.Typename == "IssueComment" {
				for i.NextCommentEdit() {

					_ = i.CommentEditValue()

					fmt.Printf("comment edit\n")
				}
			}
		}
	}

	fmt.Println(i.Error())
	fmt.Println(i.Count())
}
