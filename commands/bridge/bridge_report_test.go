package bridgecmd

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entity"
)

const (
	bugA = entity.Id("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	bugB = entity.Id("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
)

func exportEvents(results ...core.ExportResult) <-chan core.ExportResult {
	ch := make(chan core.ExportResult, len(results))
	for _, r := range results {
		ch <- r
	}
	close(ch)
	return ch
}

func importEvents(results ...core.ImportResult) <-chan core.ImportResult {
	ch := make(chan core.ImportResult, len(results))
	for _, r := range results {
		ch <- r
	}
	close(ch)
	return ch
}

func newOut() *execenv.TestOut {
	return &execenv.TestOut{Buffer: &bytes.Buffer{}}
}

func skippedEvents() <-chan core.ExportResult {
	return exportEvents(
		core.NewExportNothing(bugA, core.ReasonNoTokenActor),
		core.NewExportNothing(bugB, core.ReasonNoTokenActor),
		core.NewExportNothing(bugA, "issue belongs to another repository"),
		core.NewExportNothing(bugB, core.ReasonNothingExported),
		core.NewExportBug(bugA),
	)
}

func TestReportExportResultsHidesSkipsByDefault(t *testing.T) {
	out := newOut()
	require.NoError(t, reportExportResults(out, "b", skippedEvents(), false))
	assert.Equal(t, "[aaaaaaa] new issue: "+bugA.String()+"\nexported 1 issues with b bridge\n", out.String())
}

func TestReportExportResultsVerboseSummarizesSkips(t *testing.T) {
	out := newOut()
	require.NoError(t, reportExportResults(out, "b", skippedEvents(), true))
	assert.Equal(t,
		"[aaaaaaa] new issue: "+bugA.String()+"\n"+
			"skipped 2 issues: "+core.ReasonNoTokenActor+"\n"+
			"skipped 1 issues: issue belongs to another repository\n"+
			"exported 1 issues with b bridge\n",
		out.String(),
		"reasons keep their first-seen order; up-to-date issues are not listed")
}

func TestReportExportResultsFailsOnErrors(t *testing.T) {
	out := newOut()
	err := reportExportResults(out, "b", exportEvents(
		core.NewExportError(errors.New("boom"), bugA),
		core.NewExportError(errors.New("bang"), bugB),
	), false)
	assert.EqualError(t, err, "2 export error(s) with b bridge")
	assert.Contains(t, out.String(), "boom")
	assert.Contains(t, out.String(), "exported 0 issues with b bridge")
}

func TestReportImportResultsFailsOnErrors(t *testing.T) {
	out := newOut()
	err := reportImportResults(out, "b", importEvents(
		core.NewImportBug(bugA),
		core.NewImportError(errors.New("boom"), bugB),
	))
	assert.EqualError(t, err, "1 import error(s) with b bridge")
	assert.Contains(t, out.String(), "imported 1 issues and 0 identities with b bridge")
}

func TestReportImportResultsIgnoresCancellation(t *testing.T) {
	out := newOut()
	err := reportImportResults(out, "b", importEvents(core.NewImportError(context.Canceled, "")))
	assert.NoError(t, err)
	assert.Equal(t, "imported 0 issues and 0 identities with b bridge\n", out.String())
}
