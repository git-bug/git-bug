package github

import (
	"context"
	"fmt"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/shurcooL/githubv4"
)

type githubImporter struct{}

func (*githubImporter) ImportAll(repo *cache.RepoCache, conf core.Configuration) error {
	client := buildClient(conf)

	type Event struct {
		CreatedAt githubv4.DateTime
		Actor     struct {
			Login     githubv4.String
			AvatarUrl githubv4.String
		}
	}

	var q struct {
		Repository struct {
			Issues struct {
				Nodes []struct {
					Title    string
					Timeline struct {
						Nodes []struct {
							Typename githubv4.String `graphql:"__typename"`

							// Issue
							IssueComment struct {
								Author struct {
									Login     githubv4.String
									AvatarUrl githubv4.String
								}
								BodyText  githubv4.String
								CreatedAt githubv4.DateTime

								// TODO: edition
							} `graphql:"... on IssueComment"`

							// Label
							LabeledEvent struct {
								Event
								Label struct {
									Color githubv4.String
									Name  githubv4.String
								}
							} `graphql:"... on LabeledEvent"`
							UnlabeledEvent struct {
								Event
								Label struct {
									Color githubv4.String
									Name  githubv4.String
								}
							} `graphql:"... on UnlabeledEvent"`

							// Status
							ClosedEvent struct {
								Event
							} `graphql:"... on  ClosedEvent"`
							ReopenedEvent struct {
								Event
							} `graphql:"... on  ReopenedEvent"`

							// Title
							RenamedTitleEvent struct {
								Event
								CurrentTitle  githubv4.String
								PreviousTitle githubv4.String
							} `graphql:"... on RenamedTitleEvent"`
						}
						PageInfo struct {
							EndCursor   githubv4.String
							HasNextPage bool
						}
					} `graphql:"timeline(first: $timelineFirst, after: $timelineAfter)"`
				}
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"issues(first: $issueFirst, after: $issueAfter)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner":         githubv4.String(conf[keyUser]),
		"name":          githubv4.String(conf[keyProject]),
		"issueFirst":    githubv4.Int(1),
		"issueAfter":    (*githubv4.String)(nil),
		"timelineFirst": githubv4.Int(10),
		"timelineAfter": (*githubv4.String)(nil),
	}

	for {
		err := client.Query(context.TODO(), &q, variables)
		if err != nil {
			return err
		}

		for _, event := range q.Repository.Issues.Nodes[0].Timeline.Nodes {
			fmt.Println(event)
		}

		if !q.Repository.Issues.Nodes[0].Timeline.PageInfo.HasNextPage {
			break
		}
		variables["timelineAfter"] = githubv4.NewString(q.Repository.Issues.Nodes[0].Timeline.PageInfo.EndCursor)
	}

	return nil
}

func (*githubImporter) Import(repo *cache.RepoCache, conf core.Configuration, id string) error {
	fmt.Println(conf)
	fmt.Println("IMPORT")

	return nil
}
