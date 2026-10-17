package graphql

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/git-bug/gitconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/misc/random_bugs"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestQueries(t *testing.T) {
	repo := repotest.NewGoGitRepo(t, false)

	random_bugs.FillRepoWithSeed(repo, 10, 42)

	mrc := cache.NewMultiRepoCache()
	t.Cleanup(func() { require.NoError(t, mrc.Close()) })
	rc, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	mrc.Add("", rc)

	handler := NewHandler(mrc, nil, false)

	c := client.New(handler)

	query := `
     query {
        repository {
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
                  ... on BugCreateOperation {
                    title
                    message
                    files
                  }
                  ... on BugSetTitleOperation {
                    title
                    was
                  }
                  ... on BugAddCommentOperation {
                    files
                    message
                  }
                  ... on BugSetStatusOperation {
                    status
                  }
                  ... on BugLabelChangeOperation {
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
		Repository struct {
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

	err = c.Post(query, &resp)
	assert.NoError(t, err)
}

// TestGitBrowseQueries exercises the git-browsing GraphQL fields (commit, blob,
// tree, commits, lastCommits) against a synthetic fixture repo with the same
// commit graph used by RepoBrowseTest:
//
//	c1 ── c2 ── c3   refs/heads/main
//	       └────────  refs/heads/feature
//	c1 ←── refs/tags/v1.0
func TestGitBrowseQueries(t *testing.T) {
	repo := repotest.NewGoGitRepo(t, false)
	require.NoError(t, repo.Config().Update(func(f *gitconfig.File) error {
		return f.Set("init.defaultBranch", "main")
	}))

	// ── build fixture ─────────────────────────────────────────────────────────

	readmeV1 := []byte("# Hello\n")
	readmeV3 := []byte("# Hello\n\n## Updated\n")
	mainV1 := []byte("package main\n")
	mainV2 := []byte("package main\n\n// updated\n")
	libV1 := []byte("package lib\n")
	utilV1 := []byte("package util\n")

	hReadmeV1, err := repo.StoreData(readmeV1)
	require.NoError(t, err)
	hReadmeV3, err := repo.StoreData(readmeV3)
	require.NoError(t, err)
	hMainV1, err := repo.StoreData(mainV1)
	require.NoError(t, err)
	hMainV2, err := repo.StoreData(mainV2)
	require.NoError(t, err)
	hLibV1, err := repo.StoreData(libV1)
	require.NoError(t, err)
	hUtilV1, err := repo.StoreData(utilV1)
	require.NoError(t, err)

	srcTreeV1, err := repo.StoreTree([]repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: hLibV1, Name: "lib.go"},
	})
	require.NoError(t, err)
	rootTreeV1, err := repo.StoreTree([]repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: hReadmeV1, Name: "README.md"},
		{ObjectType: repository.Blob, Hash: hMainV1, Name: "main.go"},
		{ObjectType: repository.Tree, Hash: srcTreeV1, Name: "src"},
	})
	require.NoError(t, err)

	srcTreeV2, err := repo.StoreTree([]repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: hLibV1, Name: "lib.go"},
		{ObjectType: repository.Blob, Hash: hUtilV1, Name: "util.go"},
	})
	require.NoError(t, err)
	rootTreeV2, err := repo.StoreTree([]repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: hReadmeV1, Name: "README.md"},
		{ObjectType: repository.Blob, Hash: hMainV2, Name: "main.go"},
		{ObjectType: repository.Tree, Hash: srcTreeV2, Name: "src"},
	})
	require.NoError(t, err)

	rootTreeV3, err := repo.StoreTree([]repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: hReadmeV3, Name: "README.md"},
		{ObjectType: repository.Blob, Hash: hMainV2, Name: "main.go"},
		{ObjectType: repository.Tree, Hash: srcTreeV2, Name: "src"},
	})
	require.NoError(t, err)

	c1, err := repo.StoreCommit(rootTreeV1)
	require.NoError(t, err)
	c2, err := repo.StoreCommit(rootTreeV2, c1)
	require.NoError(t, err)
	c3, err := repo.StoreCommit(rootTreeV3, c2)
	require.NoError(t, err)

	require.NoError(t, repo.SetBranch("main", c3))
	require.NoError(t, repo.SetBranch("feature", c2))
	require.NoError(t, repo.SetTag("v1.0", c1))

	// ── set up GraphQL handler ─────────────────────────────────────────────────

	mrc := cache.NewMultiRepoCache()
	t.Cleanup(func() { require.NoError(t, mrc.Close()) })
	rc, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	mrc.Add("", rc)
	c := client.New(NewHandler(mrc, nil, false))

	// ── commit ────────────────────────────────────────────────────────────────

	t.Run("commit", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Commit struct {
					Hash    string
					Parents []string
				}
			}
		}
		require.NoError(t, c.Post(`query($hash: String!) {
			repository { commit(hash: $hash) { hash parents } }
		}`, &resp, client.Var("hash", string(c3))))
		got := resp.Repository.Commit
		require.Equal(t, string(c3), got.Hash)
		require.Equal(t, []string{string(c2)}, got.Parents)
	})

	t.Run("commit_not_found", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Commit *struct{ Hash string }
			}
		}
		require.NoError(t, c.Post(`query {
			repository { commit(hash: "0000000000000000000000000000000000000000") { hash } }
		}`, &resp))
		require.Nil(t, resp.Repository.Commit)
	})

	// ── blob ──────────────────────────────────────────────────────────────────

	t.Run("blob", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Blob struct {
					Hash     string
					IsBinary bool
					Size     int
					Text     *string
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository { blob(ref: "main", path: "README.md") { hash isBinary size text } }
		}`, &resp))
		got := resp.Repository.Blob
		require.Equal(t, string(hReadmeV3), got.Hash)
		require.False(t, got.IsBinary)
		require.Equal(t, len(readmeV3), got.Size)
		require.NotNil(t, got.Text)
		require.Equal(t, string(readmeV3), *got.Text)
	})

	t.Run("blob_not_found", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Blob *struct{ Hash string }
			}
		}
		require.NoError(t, c.Post(`query {
			repository { blob(ref: "main", path: "nonexistent.go") { hash } }
		}`, &resp))
		require.Nil(t, resp.Repository.Blob)
	})

	// ── tree ──────────────────────────────────────────────────────────────────

	t.Run("tree", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Tree []struct {
					Name string
					Type string `json:"type"`
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository { tree(ref: "main", path: "") { name type } }
		}`, &resp))
		byName := make(map[string]string)
		for _, e := range resp.Repository.Tree {
			byName[e.Name] = e.Type
		}
		require.Equal(t, "BLOB", byName["README.md"])
		require.Equal(t, "BLOB", byName["main.go"])
		require.Equal(t, "TREE", byName["src"])
	})

	t.Run("tree_lastCommit", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Tree []struct {
					Name       string
					LastCommit struct{ Hash string }
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository { tree(ref: "main", path: "") { name lastCommit { hash } } }
		}`, &resp))
		byName := make(map[string]string)
		for _, e := range resp.Repository.Tree {
			byName[e.Name] = e.LastCommit.Hash
		}
		require.Equal(t, string(c3), byName["README.md"]) // changed in c3
		require.Equal(t, string(c2), byName["main.go"])   // changed in c2
		require.Equal(t, string(c2), byName["src"])       // util.go added in c2
	})

	// ── commits ───────────────────────────────────────────────────────────────

	t.Run("commits", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Commits struct {
					TotalCount int
					PageInfo   struct{ HasNextPage bool }
					Nodes      []struct{ Hash string }
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository {
				commits(ref: "main", first: 2) {
					totalCount pageInfo { hasNextPage } nodes { hash }
				}
			}
		}`, &resp))
		got := resp.Repository.Commits
		require.Equal(t, 2, got.TotalCount)
		require.True(t, got.PageInfo.HasNextPage)
		require.Equal(t, string(c3), got.Nodes[0].Hash)
		require.Equal(t, string(c2), got.Nodes[1].Hash)
	})

	// ── refs ─────────────────────────────────────────────────────────────────

	t.Run("refs_all", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Refs struct {
					TotalCount int
					Nodes      []struct {
						Name      string
						ShortName string
						Type      string `json:"type"`
						Hash      string
					}
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository { refs { totalCount nodes { name shortName type hash } } }
		}`, &resp))
		nodes := resp.Repository.Refs.Nodes
		require.Equal(t, 3, resp.Repository.Refs.TotalCount)
		byShort := make(map[string]struct {
			Name string
			Type string
			Hash string
		})
		for _, n := range nodes {
			byShort[n.ShortName] = struct {
				Name string
				Type string
				Hash string
			}{n.Name, n.Type, n.Hash}
		}
		require.Equal(t, "refs/heads/feature", byShort["feature"].Name)
		require.Equal(t, "BRANCH", byShort["feature"].Type)
		require.Equal(t, string(c2), byShort["feature"].Hash)
		require.Equal(t, "refs/heads/main", byShort["main"].Name)
		require.Equal(t, "BRANCH", byShort["main"].Type)
		require.Equal(t, string(c3), byShort["main"].Hash)
		require.Equal(t, "refs/tags/v1.0", byShort["v1.0"].Name)
		require.Equal(t, "TAG", byShort["v1.0"].Type)
		require.Equal(t, string(c1), byShort["v1.0"].Hash)
	})

	t.Run("refs_branch_filter", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Refs struct {
					TotalCount int
					Nodes      []struct{ ShortName string }
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository { refs(type: BRANCH) { totalCount nodes { shortName } } }
		}`, &resp))
		require.Equal(t, 2, resp.Repository.Refs.TotalCount)
		names := make([]string, len(resp.Repository.Refs.Nodes))
		for i, n := range resp.Repository.Refs.Nodes {
			names[i] = n.ShortName
		}
		require.ElementsMatch(t, []string{"main", "feature"}, names)
	})

	t.Run("refs_tag_filter", func(t *testing.T) {
		var resp struct {
			Repository struct {
				Refs struct {
					TotalCount int
					Nodes      []struct{ ShortName string }
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository { refs(type: TAG) { totalCount nodes { shortName } } }
		}`, &resp))
		require.Equal(t, 1, resp.Repository.Refs.TotalCount)
		require.Equal(t, "v1.0", resp.Repository.Refs.Nodes[0].ShortName)
	})

	t.Run("refs_commit_filter_error", func(t *testing.T) {
		var resp struct {
			Repository struct{ Refs *struct{ TotalCount int } }
		}
		err := c.Post(`query {
			repository { refs(type: COMMIT) { totalCount } }
		}`, &resp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "COMMIT")
	})

	// ── head ─────────────────────────────────────────────────────────────────

	t.Run("head", func(t *testing.T) {
		// a new repository's HEAD points to refs/heads/master
		require.NoError(t, repo.SetBranch("master", c3))
		var resp struct {
			Repository struct {
				Head struct {
					Name      string
					ShortName string
					Type      string `json:"type"`
					Hash      string
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository { head { name shortName type hash } }
		}`, &resp))
		got := resp.Repository.Head
		require.Equal(t, "refs/heads/master", got.Name)
		require.Equal(t, "master", got.ShortName)
		require.Equal(t, "BRANCH", got.Type)
		require.Equal(t, string(c3), got.Hash)
	})

	// ── lastCommits ───────────────────────────────────────────────────────────

	t.Run("lastCommits", func(t *testing.T) {
		var resp struct {
			Repository struct {
				LastCommits []struct {
					Name   string
					Commit struct{ Hash string }
				}
			}
		}
		require.NoError(t, c.Post(`query {
			repository {
				lastCommits(ref: "main", names: ["README.md", "main.go"]) {
					name commit { hash }
				}
			}
		}`, &resp))
		byName := make(map[string]string)
		for _, lc := range resp.Repository.LastCommits {
			byName[lc.Name] = lc.Commit.Hash
		}
		require.Equal(t, string(c3), byName["README.md"]) // changed in c3
		require.Equal(t, string(c2), byName["main.go"])   // changed in c2
	})
}

func TestSubscriptions(t *testing.T) {
	repo := repotest.NewGoGitRepo(t, false)

	mrc := cache.NewMultiRepoCache()
	t.Cleanup(func() { require.NoError(t, mrc.Close()) })
	rc, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	mrc.Add("", rc)

	rene, err := rc.Identities().New("René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, rc.SetUserIdentity(rene))

	h := NewHandler(mrc, nil, false)
	c := client.New(h)

	// A subscription starts asynchronously, so a change made right after
	// subscribing may come before it: keepDoing repeats the change until the
	// subscription gets one.  The changes run in a goroutine, so they report
	// errors with assert: require can't stop the test from there.
	keepDoing := func(t *testing.T, change func()) (stop func()) {
		done := make(chan struct{})
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			for {
				change()
				select {
				case <-done:
					return
				case <-time.After(50 * time.Millisecond):
				}
			}
		}()
		return func() {
			close(done)
			<-stopped
		}
	}

	b, _, err := rc.Bugs().New("subscribed", "body")
	require.NoError(t, err)
	other, _, err := rc.Bugs().New("not subscribed", "body")
	require.NoError(t, err)

	type entityChange struct {
		Event string
		Id    string
	}

	t.Run("given entities", func(t *testing.T) {
		sub := c.Websocket(fmt.Sprintf(`subscription { entitiesChanged(type: BUG, ids: ["%s"]) { event id } }`, b.Id()))
		t.Cleanup(func() { _ = sub.Close() })

		stop := keepDoing(t, func() {
			_, err := other.SetTitle("other title")
			assert.NoError(t, err)
			assert.NoError(t, other.Commit())
			_, err = b.SetTitle("new title")
			assert.NoError(t, err)
			assert.NoError(t, b.Commit())
		})
		var resp struct{ EntitiesChanged []entityChange }
		err := sub.Next(&resp)
		stop()
		require.NoError(t, err)
		require.Equal(t, []entityChange{{Event: "UPDATED", Id: b.Id().String()}}, resp.EntitiesChanged)
	})

	t.Run("all entities of a type", func(t *testing.T) {
		sub := c.Websocket(`subscription { entitiesChanged(type: BUG) { event id } }`)
		t.Cleanup(func() { _ = sub.Close() })

		created := make(map[string]bool)
		var mu sync.Mutex
		stop := keepDoing(t, func() {
			b, _, err := rc.Bugs().New("new bug", "body")
			assert.NoError(t, err)
			mu.Lock()
			created[b.Id().String()] = true
			mu.Unlock()
		})
		var resp struct{ EntitiesChanged []entityChange }
		err := sub.Next(&resp)
		stop()
		require.NoError(t, err)
		require.NotEmpty(t, resp.EntitiesChanged)
		for _, change := range resp.EntitiesChanged {
			require.Equal(t, "CREATED", change.Event)
			require.True(t, created[change.Id])
		}
	})

	t.Run("a client leaving doesn't hold the changes", func(t *testing.T) {
		sub := c.Websocket(fmt.Sprintf(`subscription { entitiesChanged(type: BUG, ids: ["%s"]) { id } }`, b.Id()))

		stop := keepDoing(t, func() {
			_, err := b.SetTitle("title")
			assert.NoError(t, err)
			assert.NoError(t, b.Commit())
		})
		var resp struct{ EntitiesChanged []entityChange }
		err := sub.Next(&resp)
		stop()
		require.NoError(t, err)
		require.NoError(t, sub.Close())

		done := make(chan struct{})
		go func() {
			defer close(done)
			for i := 0; i < 20; i++ {
				_, err := b.SetTitle(fmt.Sprintf("title %d", i))
				assert.NoError(t, err)
				assert.NoError(t, b.Commit())
			}
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("changes blocked by a subscription whose client left")
		}
	})
}
