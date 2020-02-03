package gitlab

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
)

var (
	ErrMissingIdentityToken = errors.New("missing identity token")
)

// gitlabExporter implement the Exporter interface
type gitlabExporter struct {
	conf core.Configuration

	// cache identities clients
	identityClient map[entity.Id]*gitlab.Client

	// gitlab repository ID
	repositoryID string

	// cache identifiers used to speed up exporting operations
	// cleared for each bug
	cachedOperationIDs map[string]string
}

// Init .
func (ge *gitlabExporter) Init(repo *cache.RepoCache, conf core.Configuration) error {
	ge.conf = conf
	ge.identityClient = make(map[entity.Id]*gitlab.Client)
	ge.cachedOperationIDs = make(map[string]string)

	// get repository node id
	ge.repositoryID = ge.conf[keyProjectID]

	// preload all clients
	err := ge.cacheAllClient(repo)
	if err != nil {
		return err
	}

	return nil
}

func (ge *gitlabExporter) cacheAllClient(repo *cache.RepoCache) error {
	creds, err := auth.List(repo, auth.WithTarget(target), auth.WithKind(auth.KindToken))
	if err != nil {
		return err
	}

	for _, cred := range creds {
		login, ok := cred.GetMetadata(auth.MetaKeyLogin)
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "credential %s is not tagged with a Gitlab login\n", cred.ID().Human())
			continue
		}

		user, err := repo.ResolveIdentityImmutableMetadata(metaKeyGitlabLogin, login)
		if err == identity.ErrIdentityNotExist {
			continue
		}
		if err != nil {
			return nil
		}

		if _, ok := ge.identityClient[user.Id()]; !ok {
			client, err := buildClient(ge.conf[keyGitlabBaseUrl], creds[0].(*auth.Token))
			if err != nil {
				return err
			}
			ge.identityClient[user.Id()] = client
		}
	}

	return nil
}

// getIdentityClient return a gitlab v4 API client configured with the access token of the given identity.
func (ge *gitlabExporter) getIdentityClient(userId entity.Id) (*gitlab.Client, error) {
	client, ok := ge.identityClient[userId]
	if ok {
		return client, nil
	}

	return nil, ErrMissingIdentityToken
}

// ExportAll export all event made by the current user to Gitlab
func (ge *gitlabExporter) ExportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	out := make(chan core.ExportResult)

	go func() {
		defer close(out)

		allIdentitiesIds := make([]entity.Id, 0, len(ge.identityClient))
		for id := range ge.identityClient {
			allIdentitiesIds = append(allIdentitiesIds, id)
		}

		allBugsIds := repo.AllBugsIds()

		for _, id := range allBugsIds {
			select {
			case <-ctx.Done():
				return
			default:
				b, err := repo.ResolveBug(id)
				if err != nil {
					out <- core.NewExportError(err, id)
					return
				}

				snapshot := b.Snapshot()

				// ignore issues created before since date
				// TODO: compare the Lamport time instead of using the unix time
				if snapshot.CreatedAt.Before(since) {
					out <- core.NewExportNothing(b.Id(), "bug created before the since date")
					continue
				}

				if snapshot.HasAnyActor(allIdentitiesIds...) {
					// try to export the bug and it associated events
					ge.exportBug(ctx, b, since, out)
				}
			}
		}
	}()

	return out, nil
}

// exportBug publish bugs and related events
func (ge *gitlabExporter) exportBug(ctx context.Context, b *cache.BugCache, since time.Time, out chan<- core.ExportResult) {
	snapshot := b.Snapshot()

	var bugUpdated bool
	var err error
	var bugGitlabID int
	var bugGitlabIDString string
	var GitlabBaseUrl string
	var bugCreationId string

	// Special case:
	// if a user try to export a bug that is not already exported to Gitlab (or imported
	// from Gitlab) and we do not have the token of the bug author, there is nothing we can do.

	// skip bug if origin is not allowed
	origin, ok := snapshot.GetCreateMetadata(core.MetaKeyOrigin)
	if ok && origin != target {
		out <- core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
		return
	}

	// first operation is always createOp
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	author := snapshot.Author

	// get gitlab bug ID
	gitlabID, ok := snapshot.GetCreateMetadata(metaKeyGitlabId)
	if ok {
		gitlabBaseUrl, ok := snapshot.GetCreateMetadata(metaKeyGitlabBaseUrl)
		if ok && gitlabBaseUrl != ge.conf[keyGitlabBaseUrl] {
			out <- core.NewExportNothing(b.Id(), "skipping issue imported from another Gitlab instance")
			return
		}

		projectID, ok := snapshot.GetCreateMetadata(metaKeyGitlabProject)
		if !ok {
			err := fmt.Errorf("expected to find gitlab project id")
			out <- core.NewExportError(err, b.Id())
			return
		}

		if projectID != ge.conf[keyProjectID] {
			out <- core.NewExportNothing(b.Id(), "skipping issue imported from another repository")
			return
		}

		// will be used to mark operation related to a bug as exported
		bugGitlabIDString = gitlabID
		bugGitlabID, err = strconv.Atoi(bugGitlabIDString)
		if err != nil {
			out <- core.NewExportError(fmt.Errorf("unexpected gitlab id format: %s", bugGitlabIDString), b.Id())
			return
		}

	} else {
		// check that we have a token for operation author
		client, err := ge.getIdentityClient(author.Id())
		if err != nil {
			// if bug is still not exported and we do not have the author stop the execution
			out <- core.NewExportNothing(b.Id(), fmt.Sprintf("missing author token"))
			return
		}

		// create bug
		_, id, url, err := createGitlabIssue(ctx, client, ge.repositoryID, createOp.Title, createOp.Message)
		if err != nil {
			err := errors.Wrap(err, "exporting gitlab issue")
			out <- core.NewExportError(err, b.Id())
			return
		}

		idString := strconv.Itoa(id)
		out <- core.NewExportBug(b.Id())

		_, err = b.SetMetadata(
			createOp.Id(),
			map[string]string{
				metaKeyGitlabId:      idString,
				metaKeyGitlabUrl:     url,
				metaKeyGitlabProject: ge.repositoryID,
				metaKeyGitlabBaseUrl: GitlabBaseUrl,
			},
		)
		if err != nil {
			err := errors.Wrap(err, "marking operation as exported")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// commit operation to avoid creating multiple issues with multiple pushes
		if err := b.CommitAsNeeded(); err != nil {
			err := errors.Wrap(err, "bug commit")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// cache bug gitlab ID and URL
		bugGitlabID = id
		bugGitlabIDString = idString
	}

	bugCreationId = createOp.Id().String()
	// cache operation gitlab id
	ge.cachedOperationIDs[bugCreationId] = bugGitlabIDString

	labelSet := make(map[string]struct{})
	for _, op := range snapshot.Operations[1:] {
		// ignore SetMetadata operations
		if _, ok := op.(*bug.SetMetadataOperation); ok {
			continue
		}

		// ignore operations already existing in gitlab (due to import or export)
		// cache the ID of already exported or imported issues and events from Gitlab
		if id, ok := op.GetMetadata(metaKeyGitlabId); ok {
			ge.cachedOperationIDs[op.Id().String()] = id
			continue
		}

		opAuthor := op.GetAuthor()
		client, err := ge.getIdentityClient(opAuthor.Id())
		if err != nil {
			continue
		}

		var id int
		var idString, url string
		switch op := op.(type) {
		case *bug.AddCommentOperation:

			// send operation to gitlab
			id, err = addCommentGitlabIssue(ctx, client, ge.repositoryID, bugGitlabID, op.Message)
			if err != nil {
				err := errors.Wrap(err, "adding comment")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportComment(op.Id())

			idString = strconv.Itoa(id)
			// cache comment id
			ge.cachedOperationIDs[op.Id().String()] = idString

		case *bug.EditCommentOperation:
			targetId := op.Target.String()

			// Since gitlab doesn't consider the issue body as a comment
			if targetId == bugCreationId {

				// case bug creation operation: we need to edit the Gitlab issue
				if err := updateGitlabIssueBody(ctx, client, ge.repositoryID, bugGitlabID, op.Message); err != nil {
					err := errors.Wrap(err, "editing issue")
					out <- core.NewExportError(err, b.Id())
					return
				}

				out <- core.NewExportCommentEdition(op.Id())
				id = bugGitlabID

			} else {

				// case comment edition operation: we need to edit the Gitlab comment
				commentID, ok := ge.cachedOperationIDs[targetId]
				if !ok {
					out <- core.NewExportError(fmt.Errorf("unexpected error: comment id not found"), op.Target)
					return
				}

				commentIDint, err := strconv.Atoi(commentID)
				if err != nil {
					out <- core.NewExportError(fmt.Errorf("unexpected comment id format"), op.Target)
					return
				}

				if err := editCommentGitlabIssue(ctx, client, ge.repositoryID, bugGitlabID, commentIDint, op.Message); err != nil {
					err := errors.Wrap(err, "editing comment")
					out <- core.NewExportError(err, b.Id())
					return
				}

				out <- core.NewExportCommentEdition(op.Id())
				id = commentIDint
			}

		case *bug.SetStatusOperation:
			if err := updateGitlabIssueStatus(ctx, client, ge.repositoryID, bugGitlabID, op.Status); err != nil {
				err := errors.Wrap(err, "editing status")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportStatusChange(op.Id())
			id = bugGitlabID

		case *bug.SetTitleOperation:
			if err := updateGitlabIssueTitle(ctx, client, ge.repositoryID, bugGitlabID, op.Title); err != nil {
				err := errors.Wrap(err, "editing title")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportTitleEdition(op.Id())
			id = bugGitlabID

		case *bug.LabelChangeOperation:
			// we need to set the actual list of labels at each label change operation
			// because gitlab update issue requests need directly the latest list of the verison

			for _, label := range op.Added {
				labelSet[label.String()] = struct{}{}
			}

			for _, label := range op.Removed {
				delete(labelSet, label.String())
			}

			labels := make([]string, 0, len(labelSet))
			for key := range labelSet {
				labels = append(labels, key)
			}

			if err := updateGitlabIssueLabels(ctx, client, ge.repositoryID, bugGitlabID, labels); err != nil {
				err := errors.Wrap(err, "updating labels")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportLabelChange(op.Id())
			id = bugGitlabID
		default:
			panic("unhandled operation type case")
		}

		idString = strconv.Itoa(id)
		// mark operation as exported
		if err := markOperationAsExported(b, op.Id(), idString, url); err != nil {
			err := errors.Wrap(err, "marking operation as exported")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// commit at each operation export to avoid exporting same events multiple times
		if err := b.CommitAsNeeded(); err != nil {
			err := errors.Wrap(err, "bug commit")
			out <- core.NewExportError(err, b.Id())
			return
		}

		bugUpdated = true
	}

	if !bugUpdated {
		out <- core.NewExportNothing(b.Id(), "nothing has been exported")
	}
}

func markOperationAsExported(b *cache.BugCache, target entity.Id, gitlabID, gitlabURL string) error {
	_, err := b.SetMetadata(
		target,
		map[string]string{
			metaKeyGitlabId:  gitlabID,
			metaKeyGitlabUrl: gitlabURL,
		},
	)

	return err
}

// create a gitlab. issue and return it ID
func createGitlabIssue(ctx context.Context, gc *gitlab.Client, repositoryID, title, body string) (int, int, string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	issue, _, err := gc.Issues.CreateIssue(
		repositoryID,
		&gitlab.CreateIssueOptions{
			Title:       &title,
			Description: &body,
		},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		return 0, 0, "", err
	}

	return issue.ID, issue.IID, issue.WebURL, nil
}

// add a comment to an issue and return it ID
func addCommentGitlabIssue(ctx context.Context, gc *gitlab.Client, repositoryID string, issueID int, body string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	note, _, err := gc.Notes.CreateIssueNote(
		repositoryID, issueID,
		&gitlab.CreateIssueNoteOptions{
			Body: &body,
		},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		return 0, err
	}

	return note.ID, nil
}

func editCommentGitlabIssue(ctx context.Context, gc *gitlab.Client, repositoryID string, issueID, noteID int, body string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	_, _, err := gc.Notes.UpdateIssueNote(
		repositoryID, issueID, noteID,
		&gitlab.UpdateIssueNoteOptions{
			Body: &body,
		},
		gitlab.WithContext(ctx),
	)

	return err
}

func updateGitlabIssueStatus(ctx context.Context, gc *gitlab.Client, repositoryID string, issueID int, status bug.Status) error {
	var state string

	switch status {
	case bug.OpenStatus:
		state = "reopen"
	case bug.ClosedStatus:
		state = "close"
	default:
		panic("unknown bug state")
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	_, _, err := gc.Issues.UpdateIssue(
		repositoryID, issueID,
		&gitlab.UpdateIssueOptions{
			StateEvent: &state,
		},
		gitlab.WithContext(ctx),
	)

	return err
}

func updateGitlabIssueBody(ctx context.Context, gc *gitlab.Client, repositoryID string, issueID int, body string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	_, _, err := gc.Issues.UpdateIssue(
		repositoryID, issueID,
		&gitlab.UpdateIssueOptions{
			Description: &body,
		},
		gitlab.WithContext(ctx),
	)

	return err
}

func updateGitlabIssueTitle(ctx context.Context, gc *gitlab.Client, repositoryID string, issueID int, title string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	_, _, err := gc.Issues.UpdateIssue(
		repositoryID, issueID,
		&gitlab.UpdateIssueOptions{
			Title: &title,
		},
		gitlab.WithContext(ctx),
	)

	return err
}

// update gitlab. issue labels
func updateGitlabIssueLabels(ctx context.Context, gc *gitlab.Client, repositoryID string, issueID int, labels []string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	gitlabLabels := gitlab.Labels(labels)
	_, _, err := gc.Issues.UpdateIssue(
		repositoryID, issueID,
		&gitlab.UpdateIssueOptions{
			Labels: &gitlabLabels,
		},
		gitlab.WithContext(ctx),
	)

	return err
}
