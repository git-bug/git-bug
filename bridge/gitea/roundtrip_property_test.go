package gitea

import (
	"context"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/repository"
)

type roundTripBug struct {
	Title    string
	Body     string
	Closed   bool
	Comments []string
	Labels   []string
}

type normalizedBug struct {
	Title    string
	Body     string
	Closed   bool
	Comments []string
	Labels   []string
}

type testLike interface {
	Helper()
	Cleanup(func())
	Fatalf(format string, args ...any)
}

func roundTripBugGen() *rapid.Generator[roundTripBug] {
	shortText := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 _./-]{0,24}`)
	bodyText := rapid.StringMatching(`[a-zA-Z0-9][a-zA-Z0-9 _./\n-]{0,80}`)
	labelText := rapid.StringMatching(`[a-z][a-z0-9/-]{0,16}`)

	return rapid.Custom(func(t *rapid.T) roundTripBug {
		labels := rapid.SliceOfN(labelText, 0, 4).Draw(t, "labels")
		sort.Strings(labels)
		labels = compactStrings(labels)

		return roundTripBug{
			Title:    shortText.Draw(t, "title"),
			Body:     bodyText.Draw(t, "body"),
			Closed:   rapid.Bool().Draw(t, "closed"),
			Comments: rapid.SliceOfN(bodyText, 0, 4).Draw(t, "comments"),
			Labels:   labels,
		}
	})
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func requireNoError(t testLike, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("%v: %v", msgAndArgs[0], err)
		}
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireLen[T any](t testLike, values []T, expected int) {
	t.Helper()
	if len(values) != expected {
		t.Fatalf("expected length %d, got %d", expected, len(values))
	}
}

func assertEqual(t testLike, expected, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("not equal\nexpected: %#v\nactual:   %#v", expected, actual)
	}
}

func newRoundTripRepo(t testLike) (repository.TestedRepo, *cache.RepoCache) {
	t.Helper()
	dir, err := os.MkdirTemp("", "git-bug-gitea-roundtrip-*")
	requireNoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	repo := repository.CreateGoGitTestRepoAtDir(dir, false)
	t.Cleanup(func() {
		err := repo.Close()
		if err != nil {
			t.Fatalf("close repo: %v", err)
		}
	})

	backend, err := cache.NewRepoCacheNoEvents(repo)
	requireNoError(t, err)
	t.Cleanup(func() { _ = backend.Close() })
	return repo, backend
}

func storeRoundTripToken(t testLike, repo repository.TestedRepo, serverURL string) {
	t.Helper()
	token := auth.NewToken(target, "test-token")
	token.SetMetadata(auth.MetaKeyLogin, "testuser")
	token.SetMetadata(auth.MetaKeyBaseURL, serverURL)
	requireNoError(t, auth.Store(repo, token))
}

func seedRoundTripBug(t testLike, backend *cache.RepoCache, spec roundTripBug) {
	t.Helper()
	author, err := backend.Identities().NewRaw(
		"Test User",
		"testuser@example.com",
		"testuser",
		"",
		nil,
		map[string]string{metaKeyGiteaLogin: "testuser"},
	)
	requireNoError(t, err)

	b, _, err := backend.Bugs().NewRaw(
		author,
		time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		spec.Title,
		spec.Body,
		nil,
		nil,
	)
	requireNoError(t, err)

	for i, comment := range spec.Comments {
		_, _, err := b.AddCommentRaw(
			author,
			time.Date(2023, 1, 2, i, 0, 0, 0, time.UTC).Unix(),
			comment,
			nil,
			nil,
		)
		requireNoError(t, err)
	}
	if len(spec.Labels) > 0 {
		_, _, err := b.ChangeLabelsRaw(
			author,
			time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC).Unix(),
			spec.Labels,
			nil,
			nil,
		)
		requireNoError(t, err)
	}
	if spec.Closed {
		_, err := b.CloseRaw(author, time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC).Unix(), nil)
		requireNoError(t, err)
	}
}

func normalizeOnlyBug(t testLike, backend *cache.RepoCache) normalizedBug {
	t.Helper()
	bugIDs := backend.Bugs().AllIds()
	requireLen(t, bugIDs, 1)
	b, err := backend.Bugs().Resolve(bugIDs[0])
	requireNoError(t, err)

	snap := b.Snapshot()
	comments := make([]string, 0, len(snap.Comments))
	for _, comment := range snap.Comments {
		comments = append(comments, comment.Message)
	}
	body := ""
	if len(comments) > 0 {
		body = comments[0]
		comments = comments[1:]
	}
	labels := make([]string, 0, len(snap.Labels))
	for _, label := range snap.Labels {
		labels = append(labels, string(label))
	}
	sort.Strings(labels)

	return normalizedBug{
		Title:    snap.Title,
		Body:     body,
		Closed:   snap.Status == common.ClosedStatus,
		Comments: comments,
		Labels:   labels,
	}
}

func runExportAll(t testLike, exporter core.Exporter, backend *cache.RepoCache, conf core.Configuration) {
	t.Helper()
	requireNoError(t, exporter.Init(context.Background(), backend, conf))
	ch, err := exporter.ExportAll(context.Background(), backend, time.Time{})
	requireNoError(t, err)
	for result := range ch {
		requireNoError(t, result.Err)
	}
}

func importFromFake(t testLike, serverURL string) *cache.RepoCache {
	t.Helper()
	repo, imported := newRoundTripRepo(t)
	storeRoundTripToken(t, repo, serverURL)
	gi := &giteaImporter{}
	requireNoError(t, gi.Init(context.Background(), imported, roundTripConfig(serverURL)))
	ch, err := gi.ImportAll(context.Background(), imported, time.Time{})
	requireNoError(t, err)
	var results []core.ImportResult
	for r := range ch {
		results = append(results, r)
	}
	if errs := collectErrors(results); len(errs) > 0 {
		t.Fatalf("import errors: %v", errs)
	}
	return imported
}

func roundTripConfig(serverURL string) core.Configuration {
	return core.Configuration{
		core.ConfigKeyTarget: target,
		confKeyBaseURL:       serverURL,
		confKeyOwner:         "owner",
		confKeyProject:       "project",
		confKeyDefaultLogin:  "testuser",
	}
}

func TestGiteaExportImportRoundTripProperty(t *testing.T) {
	exporter := (&Gitea{}).NewExporter()
	if exporter == nil {
		t.Skip("Gitea exporter is not wired yet; property compiles and should be enabled with the exporter")
	}

	rapid.Check(t, func(rt *rapid.T) {
		spec := roundTripBugGen().Draw(rt, "bug")
		fake := (&giteatest.FakeAPI{Owner: "owner", Project: "project"}).NewServerFor(rt)
		repo, source := newRoundTripRepo(rt)
		storeRoundTripToken(rt, repo, fake.URL)
		seedRoundTripBug(rt, source, spec)

		runExportAll(rt, exporter, source, roundTripConfig(fake.URL))
		imported := importFromFake(rt, fake.URL)

		assertEqual(rt, normalizeOnlyBug(rt, source), normalizeOnlyBug(rt, imported))
	})
}

func TestGiteaExportImportRoundTripIdempotentProperty(t *testing.T) {
	exporter := (&Gitea{}).NewExporter()
	if exporter == nil {
		t.Skip("Gitea exporter is not wired yet; property compiles and should be enabled with the exporter")
	}

	rapid.Check(t, func(rt *rapid.T) {
		spec := roundTripBugGen().Draw(rt, "bug")
		fake := (&giteatest.FakeAPI{Owner: "owner", Project: "project"}).NewServerFor(rt)
		repo, source := newRoundTripRepo(rt)
		storeRoundTripToken(rt, repo, fake.URL)
		seedRoundTripBug(rt, source, spec)

		runExportAll(rt, exporter, source, roundTripConfig(fake.URL))
		firstImport := importFromFake(rt, fake.URL)
		firstState := normalizeOnlyBug(rt, firstImport)

		runExportAll(rt, exporter, source, roundTripConfig(fake.URL))
		secondImport := importFromFake(rt, fake.URL)
		assertEqual(rt, firstState, normalizeOnlyBug(rt, secondImport))
	})
}
