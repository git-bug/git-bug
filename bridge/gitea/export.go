package gitea

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	gitea "gitea.dev/sdk"
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

var (
	ErrMissingIdentityToken = errors.New("missing identity token")
)

// defaultLabelColor is used for labels the exporter has to create upstream.
const defaultLabelColor = "#cccccc"

// giteaExporter implements the Exporter interface. It publishes the
// operations authored by the identity owning the configured token. Gitea
// creates every remote change as the token owner, so operations by other
// authors are left alone.
type giteaExporter struct {
	conf   core.Configuration
	client *gitea.Client
	// exporter is the local identity of the token owner.
	exporter *cache.IdentityCache
	// labels caches repository labels by lowercase name.
	labels map[string]*gitea.Label
}

func (ge *giteaExporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	ge.conf = conf
	ge.labels = nil

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyBaseURL]),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}
	if len(creds) == 0 {
		return ErrMissingIdentityToken
	}

	ge.client, err = buildClient(conf[confKeyBaseURL], creds[0].(*auth.Token))
	if err != nil {
		return err
	}

	login := conf[confKeyDefaultLogin]
	ge.exporter, err = repo.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaScopedLogin, scopedLogin(conf[confKeyBaseURL], login))
	if entity.IsErrNotFound(err) {
		// Identities created by bridge configuration or older imports only
		// carry the unscoped login.
		ge.exporter, err = repo.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, login)
	}
	if entity.IsErrNotFound(err) {
		return ErrMissingIdentityToken
	}
	return err
}

func (ge *giteaExporter) ExportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	out := make(chan core.ExportResult)

	go func() {
		defer close(out)

		for _, id := range repo.Bugs().AllIds() {
			if ctx.Err() != nil {
				return
			}
			b, err := repo.Bugs().Resolve(id)
			if err != nil {
				send(ctx, out, core.NewExportError(err, id))
				return
			}

			snapshot := b.Snapshot()
			if snapshot.CreateTime.Before(since) {
				send(ctx, out, core.NewExportNothing(b.Id(), "bug created before the since date"))
				continue
			}
			if !snapshot.HasAnyActor(ge.exporter.Id()) {
				continue
			}
			if !ge.exportBug(ctx, b, out) {
				return
			}
		}
	}()

	return out, nil
}

func send(ctx context.Context, out chan<- core.ExportResult, result core.ExportResult) {
	select {
	case out <- result:
	case <-ctx.Done():
	}
}

// exportBug publishes one bug. It returns false when export must stop.
func (ge *giteaExporter) exportBug(ctx context.Context, b *cache.BugCache, out chan<- core.ExportResult) bool {
	fail := func(err error, when string) bool {
		send(ctx, out, core.NewExportError(errors.Wrap(err, when), b.Id()))
		return false
	}

	snapshot := b.Snapshot()
	if origin, ok := snapshot.GetCreateMetadata(core.MetaKeyOrigin); ok && origin != target {
		send(ctx, out, core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin)))
		return true
	}

	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	var index int64
	if rawID, ok := snapshot.GetCreateMetadata(metaKeyGiteaID); ok {
		meta := func(key string) string { v, _ := snapshot.GetCreateMetadata(key); return v }
		if meta(metaKeyGiteaBaseURL) != ge.conf[confKeyBaseURL] ||
			meta(metaKeyGiteaOwner) != ge.conf[confKeyOwner] ||
			meta(metaKeyGiteaProject) != ge.conf[confKeyProject] {
			send(ctx, out, core.NewExportNothing(b.Id(), "issue belongs to another Gitea repository"))
			return true
		}
		var err error
		index, err = strconv.ParseInt(rawID, 10, 64)
		if err != nil {
			return fail(err, "parse Gitea issue index")
		}
	} else {
		if snapshot.Author.Id() != ge.exporter.Id() {
			send(ctx, out, core.NewExportNothing(b.Id(), "missing author token"))
			return true
		}
		issue, err := ge.createIssue(ctx, createOp.Title, createOp.Message)
		if err != nil {
			return fail(err, "create Gitea issue")
		}
		index = issue.Index
		if err := ge.mark(b, createOp.Id(), map[string]string{
			core.MetaKeyOrigin:  target,
			metaKeyGiteaID:      strconv.FormatInt(index, 10),
			metaKeyGiteaOwner:   ge.conf[confKeyOwner],
			metaKeyGiteaProject: ge.conf[confKeyProject],
			metaKeyGiteaBaseURL: ge.conf[confKeyBaseURL],
		}); err != nil {
			return fail(err, "mark issue as exported")
		}
		send(ctx, out, core.NewExportBug(b.Id()))
	}

	commentIDs := map[entity.Id]int64{}
	labels := map[string]string{}
	exported := false

	for _, op := range snapshot.Operations {
		if _, ok := op.(dag.OperationDoesntChangeSnapshot); ok {
			continue
		}

		// Track state as it existed after each operation, including the
		// operations that came from Gitea or other authors.
		if lc, ok := op.(*bug.LabelChangeOperation); ok {
			for _, l := range lc.Added {
				labels[strings.ToLower(string(l))] = string(l)
			}
			for _, l := range lc.Removed {
				delete(labels, strings.ToLower(string(l)))
			}
		}
		if raw, ok := op.GetMetadata(metaKeyGiteaCommentID); ok {
			if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
				if _, isAdd := op.(*bug.AddCommentOperation); isAdd {
					commentIDs[op.Id()] = id
				}
			}
		}

		if op == dag.Operation(createOp) || isExportedOrImported(op) || op.Author().Id() != ge.exporter.Id() {
			continue
		}

		var err error
		var meta map[string]string
		var result core.ExportResult
		switch op := op.(type) {
		case *bug.AddCommentOperation:
			var comment *gitea.Comment
			comment, err = ge.addComment(ctx, index, op.Message)
			if err == nil {
				commentIDs[op.Id()] = comment.ID
				meta = map[string]string{metaKeyGiteaCommentID: strconv.FormatInt(comment.ID, 10)}
				result = core.NewExportComment(b.Id())
			}
		case *bug.EditCommentOperation:
			if op.Target == createOp.Id() {
				body := op.Message
				_, _, err = ge.client.Issues.EditIssue(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject], index, gitea.EditIssueOption{Body: &body})
			} else if id, ok := commentIDs[op.Target]; ok {
				_, _, err = ge.client.Issues.EditIssueComment(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject], id, gitea.EditIssueCommentOption{Body: op.Message})
			} else {
				err = fmt.Errorf("edited comment %s has no Gitea ID", op.Target.Human())
			}
			meta = map[string]string{metaKeyGiteaID: "export"}
			result = core.NewExportCommentEdition(b.Id())
		case *bug.SetTitleOperation:
			_, _, err = ge.client.Issues.EditIssue(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject], index, gitea.EditIssueOption{Title: op.Title})
			meta = map[string]string{metaKeyGiteaID: "export"}
			result = core.NewExportTitleEdition(b.Id())
		case *bug.SetStatusOperation:
			state := gitea.StateOpen
			if op.Status == common.ClosedStatus {
				state = gitea.StateClosed
			}
			_, _, err = ge.client.Issues.EditIssue(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject], index, gitea.EditIssueOption{State: &state})
			meta = map[string]string{metaKeyGiteaID: "export"}
			result = core.NewExportStatusChange(b.Id())
		case *bug.LabelChangeOperation:
			err = ge.replaceLabels(ctx, index, labels)
			meta = map[string]string{metaKeyGiteaID: "export"}
			result = core.NewExportLabelChange(b.Id())
		default:
			continue
		}
		if err != nil {
			return fail(err, fmt.Sprintf("export %T", op))
		}
		if err := ge.mark(b, op.Id(), meta); err != nil {
			return fail(err, "mark operation as exported")
		}
		send(ctx, out, result)
		exported = true
	}

	if !exported {
		send(ctx, out, core.NewExportNothing(b.Id(), "nothing has been exported"))
	}
	return true
}

func isExportedOrImported(op dag.Operation) bool {
	for _, key := range []string{metaKeyGiteaID, metaKeyGiteaCommentID, metaKeyGiteaEvent} {
		if _, ok := op.GetMetadata(key); ok {
			return true
		}
	}
	return false
}

// mark records remote metadata and commits immediately, so a later failure
// does not make the next push repeat the remote change.
func (ge *giteaExporter) mark(b *cache.BugCache, target entity.Id, meta map[string]string) error {
	if _, err := b.SetMetadataRaw(ge.exporter, time.Now().Unix(), target, meta); err != nil {
		return err
	}
	return b.CommitAsNeeded()
}

func (ge *giteaExporter) createIssue(ctx context.Context, title, body string) (*gitea.Issue, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	issue, _, err := ge.client.Issues.CreateIssue(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject],
		gitea.CreateIssueOption{Title: title, Body: body})
	return issue, err
}

func (ge *giteaExporter) addComment(ctx context.Context, index int64, body string) (*gitea.Comment, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	comment, _, err := ge.client.Issues.CreateIssueComment(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject], index,
		gitea.CreateIssueCommentOption{Body: body})
	return comment, err
}

func (ge *giteaExporter) replaceLabels(ctx context.Context, index int64, labels map[string]string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	names := make([]string, 0, len(labels))
	for _, name := range labels {
		names = append(names, name)
	}
	sort.Strings(names)

	ids := make([]int64, 0, len(names))
	for _, name := range names {
		label, err := ge.ensureLabel(ctx, name)
		if err != nil {
			return err
		}
		ids = append(ids, label.ID)
	}
	_, _, err := ge.client.Issues.ReplaceIssueLabels(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject], index,
		gitea.IssueLabelsOption{Labels: ids})
	return err
}

func (ge *giteaExporter) ensureLabel(ctx context.Context, name string) (*gitea.Label, error) {
	if ge.labels == nil {
		ge.labels = map[string]*gitea.Label{}
		for page := 1; ; page++ {
			labels, _, err := ge.client.Repositories.ListRepoLabels(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject],
				gitea.ListLabelsOptions{ListOptions: gitea.ListOptions{Page: page, PageSize: 50}})
			if err != nil {
				ge.labels = nil
				return nil, err
			}
			for _, l := range labels {
				ge.labels[strings.ToLower(l.Name)] = l
			}
			if len(labels) < 50 {
				break
			}
		}
	}
	if l, ok := ge.labels[strings.ToLower(name)]; ok {
		return l, nil
	}
	l, _, err := ge.client.Repositories.CreateLabel(ctx, ge.conf[confKeyOwner], ge.conf[confKeyProject],
		gitea.CreateLabelOption{Name: name, Color: defaultLabelColor})
	if err != nil {
		return nil, err
	}
	ge.labels[strings.ToLower(name)] = l
	return l, nil
}
