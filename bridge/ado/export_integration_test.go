package ado

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

// TestADOExporterIntegration drives the real exporter against a live Azure
// DevOps project. It WRITES one work item and then deletes it, so it is gated
// behind ADO_TEST_EXPORT in addition to the usual ADO_* variables.
func TestADOExporterIntegration(t *testing.T) {
	if os.Getenv("ADO_TEST_EXPORT") == "" {
		t.Skip("set ADO_TEST_EXPORT=1 (this WRITES a work item to the remote) to run the export test")
	}
	org := os.Getenv("ADO_ORG")
	project := os.Getenv("ADO_PROJECT")
	pat := os.Getenv("ADO_PAT")
	baseURL := os.Getenv("ADO_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	require.NotEmpty(t, org)
	require.NotEmpty(t, project)
	require.NotEmpty(t, pat)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	const login = "git-bug-bridge-test@example.com"

	repo := repository.CreateGoGitTestRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	defer backend.Close()

	author, err := backend.Identities().New("git-bug bridge test", login)
	require.NoError(t, err)
	author.SetMetadata(metaKeyAdoLogin, login)
	require.NoError(t, author.Commit())
	require.NoError(t, backend.SetUserIdentity(author))

	token := auth.NewToken(target, pat)
	token.SetMetadata(auth.MetaKeyLogin, login)
	token.SetMetadata(auth.MetaKeyBaseURL, baseURL)
	require.NoError(t, auth.Store(repo, token))

	marker := fmt.Sprintf("[git-bug bridge test] safe to delete %d", time.Now().Unix())
	b, _, err := backend.Bugs().New(marker, "Automated export smoke test. This work item will be deleted.")
	require.NoError(t, err)
	_, _, err = b.AddComment("Exported comment from the git-bug Azure DevOps bridge test.")
	require.NoError(t, err)
	_, err = b.ForceChangeLabels([]string{"git-bug-bridge-test"}, nil)
	require.NoError(t, err)
	_, err = b.Close()
	require.NoError(t, err)
	require.NoError(t, b.Commit())

	conf := core.Configuration{
		core.ConfigKeyTarget:     target,
		confKeyBaseURL:           baseURL,
		confKeyOrganization:      org,
		confKeyProject:           project,
		confKeyDefaultLogin:      login,
		confKeyCreateType:        "Task",
		confKeyClosedStateTarget: "Done",
	}
	exporter := &adoExporter{}
	require.NoError(t, exporter.Init(ctx, backend, conf))

	events, err := exporter.ExportAll(ctx, backend, time.Time{})
	require.NoError(t, err)
	for ev := range events {
		require.NoError(t, ev.Err, ev.String())
	}

	b2, err := backend.Bugs().Resolve(b.Id())
	require.NoError(t, err)
	adoIDStr, ok := b2.Snapshot().GetCreateMetadata(metaKeyAdoID)
	require.True(t, ok, "create operation should be tagged with the Azure DevOps id")
	adoID, err := strconv.Atoi(adoIDStr)
	require.NoError(t, err)
	t.Logf("exported bug %s -> Azure DevOps work item #%d", b.Id().Human(), adoID)

	client := NewClient(ctx, baseURL, org, project, pat)

	t.Cleanup(func() {
		if err := client.DeleteWorkItem(adoID, true); err == nil {
			t.Logf("permanently deleted test work item #%d", adoID)
			return
		}
		if err := client.DeleteWorkItem(adoID, false); err == nil {
			t.Logf("moved test work item #%d to the recycle bin (permanent delete not permitted)", adoID)
			return
		}
		t.Logf("WARNING: could not delete work item #%d, please delete it manually", adoID)
	})

	wi, err := client.GetWorkItem(adoID)
	require.NoError(t, err)
	require.Equal(t, marker, wi.Fields.Title)
	require.Contains(t, parseTags(wi.Fields.Tags), "git-bug-bridge-test")
	require.True(t, isClosedState(wi.Fields.State, defaultClosedStates),
		"remote state %q should map to closed", wi.Fields.State)
	t.Logf("verified remote work item: title=%q type=%q state=%q tags=%v",
		wi.Fields.Title, wi.Fields.WorkItemType, wi.Fields.State, parseTags(wi.Fields.Tags))

	comments, err := client.GetComments(adoID)
	require.NoError(t, err)
	require.NotEmpty(t, comments)
	t.Logf("verified %d remote comment(s)", len(comments))
}
