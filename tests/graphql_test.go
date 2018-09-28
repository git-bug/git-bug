package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/MichaelMure/git-bug/graphql"
	"github.com/MichaelMure/git-bug/graphql/models"
	"github.com/vektah/gqlgen/client"
)

func TestQueries(t *testing.T) {
	repo := createFilledRepo(10)

	handler, err := graphql.NewHandler(repo)
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
                    added
                    removed
                  }
                }
              }
            }
          }
        }
      }`

	type Person struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarUrl string `json:"avatarUrl"`
	}

	var resp struct {
		DefaultRepository struct {
			AllBugs struct {
				PageInfo models.PageInfo
				Nodes    []struct {
					Author    Person
					CreatedAt string `json:"createdAt"`
					HumanId   string `json:"humanId"`
					Id        string
					LastEdit  string `json:"lastEdit"`
					Status    string
					Title     string

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
							Author  Person
							Date    string
							Title   string
							Files   []string
							Message string
							Was     string
							Status  string
							Added   []string
							Removed []string
						}
					}
				}
			}
		}
	}

	c.MustPost(query, &resp)
}
