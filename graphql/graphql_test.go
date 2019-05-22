package graphql

import (
	"net/http/httptest"
	"testing"

	"github.com/vektah/gqlgen/client"

	"github.com/MichaelMure/git-bug/graphql/models"
	"github.com/MichaelMure/git-bug/misc/random_bugs"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/test"
)

func CreateFilledRepo(bugNumber int) repository.ClockedRepo {
	repo := test.CreateRepo(false)

	var seed int64 = 42
	options := random_bugs.DefaultOptions()

	options.BugNumber = bugNumber

	random_bugs.CommitRandomBugsWithSeed(repo, options, seed)
	return repo
}

func TestQueries(t *testing.T) {
	repo := CreateFilledRepo(10)

	handler, err := NewHandler(repo)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(handler)
	c := client.New(srv.URL)

	query := `
     query {
        defaultRepository {
          allBugs(first: 2) {
            pageInfo {
              endCursor
              hasNextPage
              startCursor
              hasPreviousPage
            }
            nodes{
              author {
                name
                email
                avatarUrl
              }
      
              createdAt
              humanId
              id
              lastEdit
              status
              title

              actors(first: 10) {
                pageInfo {
                  endCursor
                  hasNextPage
                  startCursor
                  hasPreviousPage
                }
                nodes {
                  id
                  humanId
                  name
                  displayName
                }
              }

              participants(first: 10) {
                pageInfo {
                  endCursor
                  hasNextPage
                  startCursor
                  hasPreviousPage
                }
                nodes {
                  id
                  humanId
                  name
                  displayName
                }
              }
      
              comments(first: 2) {
                pageInfo {
                  endCursor
                  hasNextPage
                  startCursor
                  hasPreviousPage
                }
                nodes {
                  files
                  message
                }
              }
      
              operations(first: 20) {
                pageInfo {
                  endCursor
                  hasNextPage
                  startCursor
                  hasPreviousPage
                }
                nodes {
                  author {
                    name
                    email
                    avatarUrl
                  }
                  date
                  ... on CreateOperation {
                    title
                    message
                    files
                  }
                  ... on SetTitleOperation {
                    title
                    was
                  }
                  ... on AddCommentOperation {
                    files
                    message
                  }
                  ... on SetStatusOperation {
                    status
                  }
                  ... on LabelChangeOperation {
                    added {
                      name
                      color {
                        R
                        G
                        B
                      }
                    }
                    removed {
                      name
                      color {
                        R
                        G
                        B
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }`

	type Identity struct {
		Id          string `json:"id"`
		HumanId     string `json:"humanId"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		AvatarUrl   string `json:"avatarUrl"`
		DisplayName string `json:"displayName"`
	}

	type Label struct {
		Name  string
		Color struct {
			R, G, B int
		}
	}

	var resp struct {
		DefaultRepository struct {
			AllBugs struct {
				PageInfo models.PageInfo
				Nodes    []struct {
					Author    Identity
					CreatedAt string `json:"createdAt"`
					HumanId   string `json:"humanId"`
					Id        string
					LastEdit  string `json:"lastEdit"`
					Status    string
					Title     string

					Actors struct {
						PageInfo models.PageInfo
						Nodes    []Identity
					}

					Participants struct {
						PageInfo models.PageInfo
						Nodes    []Identity
					}

					Comments struct {
						PageInfo models.PageInfo
						Nodes    []struct {
							Files   []string
							Message string
						}
					}

					Operations struct {
						PageInfo models.PageInfo
						Nodes    []struct {
							Author  Identity
							Date    string
							Title   string
							Files   []string
							Message string
							Was     string
							Status  string
							Added   []Label
							Removed []Label
						}
					}
				}
			}
		}
	}

	c.MustPost(query, &resp)
}
