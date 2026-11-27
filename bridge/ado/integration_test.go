package ado

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/text"
)

// adoTestEnv returns the live test parameters, or skips the test when ADO_TEST
// is not set. It never writes to the remote.
func adoTestEnv(t *testing.T) (baseURL, org, project, pat string) {
	t.Helper()
	if os.Getenv("ADO_TEST") == "" {
		t.Skip("set ADO_TEST=1 (plus ADO_ORG, ADO_PROJECT, ADO_PAT) to run the live integration test")
	}
	org = os.Getenv("ADO_ORG")
	project = os.Getenv("ADO_PROJECT")
	pat = os.Getenv("ADO_PAT")
	baseURL = os.Getenv("ADO_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if org == "" || project == "" || pat == "" {
		t.Fatal("ADO_ORG, ADO_PROJECT and ADO_PAT must all be set")
	}
	return baseURL, org, project, pat
}

// TestADOIntegration exercises the read-only client path against a real Azure
// DevOps organization. It never writes to the remote.
func TestADOIntegration(t *testing.T) {
	baseURL, org, project, pat := adoTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := NewClient(ctx, baseURL, org, project, pat)

	p, err := c.GetProject()
	require.NoError(t, err)
	require.NotEmpty(t, p.Name)
	t.Logf("project: name=%q id=%s", p.Name, p.ID)

	ids, err := c.SearchWorkItemIDs(buildWiql(project, time.Now().AddDate(0, 0, -30)))
	require.NoError(t, err)
	if len(ids) == 0 {
		ids, err = c.SearchWorkItemIDs(buildWiql(project, time.Time{}))
		require.NoError(t, err)
	}
	t.Logf("WIQL matched %d work item id(s)", len(ids))
	if len(ids) == 0 {
		t.Skip("no work items in project")
	}

	sample := ids
	if len(sample) > 5 {
		sample = sample[:5]
	}
	items, err := c.GetWorkItems(sample)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	for _, wi := range items {
		t.Logf("  #%d state=%q closed=%v type=%q author=%q tags=%v title=%q",
			wi.ID, wi.Fields.State, isClosedState(wi.Fields.State, defaultClosedStates),
			wi.Fields.WorkItemType, wi.Fields.CreatedBy.Key(),
			parseTags(wi.Fields.Tags), truncate(wi.Fields.Title, 60))
		require.False(t, wi.Fields.CreatedDate.IsZero(), "work item #%d has zero CreatedDate", wi.ID)
	}

	first := items[0].ID
	comments, err := c.GetComments(first)
	require.NoError(t, err)
	t.Logf("work item #%d has %d comment(s)", first, len(comments))

	updates, err := c.GetUpdates(first)
	require.NoError(t, err)
	t.Logf("work item #%d has %d revision(s)", first, len(updates))
}

// TestADOImporterIntegration drives the real importer against a real git-bug
// backend on a small sample of live work items, then verifies idempotency. It
// is read-only against Azure DevOps.
func TestADOImporterIntegration(t *testing.T) {
	baseURL, org, project, pat := adoTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	c := NewClient(ctx, baseURL, org, project, pat)

	ids, err := c.SearchWorkItemIDs(buildWiql(project, time.Now().AddDate(0, 0, -30)))
	require.NoError(t, err)
	if len(ids) == 0 {
		t.Skip("no recent work items")
	}
	// most recently changed items are at the end of the ASC ordering
	candidates := ids
	if len(candidates) > 30 {
		candidates = candidates[len(candidates)-30:]
	}
	items, err := c.GetWorkItems(candidates)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	sample := items
	if len(sample) > 3 {
		sample = sample[:3]
	}

	repo := repository.CreateGoGitTestRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	defer backend.Close()

	out := make(chan core.ImportResult, 4096)
	ai := &adoImporter{
		conf: core.Configuration{
			confKeyBaseURL:      baseURL,
			confKeyOrganization: org,
			confKeyProject:      project,
		},
		client: c,
		out:    out,
	}

	for _, wi := range sample {
		b, err := ai.ensureWorkItem(backend, wi)
		require.NoError(t, err, "ensureWorkItem #%d", wi.ID)

		comments, err := c.GetComments(wi.ID)
		require.NoError(t, err)
		for _, cm := range comments {
			require.NoError(t, ai.ensureComment(backend, b, cm))
		}

		updates, err := c.GetUpdates(wi.ID)
		require.NoError(t, err)
		require.NoError(t, ai.ensureUpdates(backend, b, wi, updates))
		if b.NeedCommit() {
			require.NoError(t, b.Commit())
		}

		snap := b.Snapshot()
		require.Equal(t, text.CleanupOneLine(wi.Fields.Title), snap.Title, "title mismatch for #%d", wi.ID)
		opsBefore := len(snap.Operations)
		t.Logf("bug %s from #%d: %d ops, status=%v (ado state %q) comments=%d revisions=%d title=%q",
			b.Id().Human(), wi.ID, opsBefore, snap.Status, wi.Fields.State,
			len(comments), len(updates), truncate(snap.Title, 50))

		b2, err := ai.ensureWorkItem(backend, wi)
		require.NoError(t, err)
		for _, cm := range comments {
			require.NoError(t, ai.ensureComment(backend, b2, cm))
		}
		require.NoError(t, ai.ensureUpdates(backend, b2, wi, updates))
		if b2.NeedCommit() {
			require.NoError(t, b2.Commit())
		}
		require.Equal(t, opsBefore, len(b2.Snapshot().Operations), "second import pass changed op count for #%d", wi.ID)
	}
	quoted := make([]string, len(defaultClosedStates))
	for i, s := range defaultClosedStates {
		quoted[i] = "'" + escapeWiql(s) + "'"
	}
	closedWiql := fmt.Sprintf(
		"SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = '%s' AND [System.State] IN (%s) ORDER BY [System.ChangedDate] DESC",
		escapeWiql(project), strings.Join(quoted, ","))
	closedIDs, err := c.SearchWorkItemIDs(closedWiql)
	require.NoError(t, err)
	if len(closedIDs) > 0 {
		citems, err := c.GetWorkItems(closedIDs[:1])
		require.NoError(t, err)
		require.NotEmpty(t, citems)
		closed := citems[0]

		b, err := ai.ensureWorkItem(backend, closed)
		require.NoError(t, err)
		updates, err := c.GetUpdates(closed.ID)
		require.NoError(t, err)
		require.NoError(t, ai.ensureUpdates(backend, b, closed, updates))
		if b.NeedCommit() {
			require.NoError(t, b.Commit())
		}
		snap := b.Snapshot()
		require.Equal(t, common.ClosedStatus, snap.Status, "closed work item #%d should import as a closed bug", closed.ID)
		hasStatusOp := false
		for _, op := range snap.Operations {
			if _, ok := op.(*bug.SetStatusOperation); ok {
				hasStatusOp = true
				break
			}
		}
		require.True(t, hasStatusOp, "closed work item #%d should have a status-change op", closed.ID)
		t.Logf("closed item #%d (%q) imported as closed bug %s (%d ops)",
			closed.ID, closed.Fields.State, b.Id().Human(), len(snap.Operations))
	} else {
		t.Log("no closed work items found; skipping close-replay assertion")
	}

	close(out)

	require.GreaterOrEqual(t, len(backend.Bugs().AllIds()), 1)
	require.GreaterOrEqual(t, len(backend.Identities().AllIds()), 1)
	t.Logf("imported %d bug(s), %d identity(ies)",
		len(backend.Bugs().AllIds()), len(backend.Identities().AllIds()))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
