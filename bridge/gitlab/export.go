package gitlab

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
)

var (
	ErrMissingIdentityToken = errors.New("missing identity token")
)

// gitlabExporter implement the Exporter interface
type gitlabExporter struct {
	conf core.Configuration

	// cache identities clients
	identityClient map[string]*gitlab.Client

	// map identities with their tokens
	identityToken map[string]string

	// gitlab repository ID
	repositoryID string

	// cache identifiers used to speed up exporting operations
	// cleared for each bug
	cachedOperationIDs map[string]string

	// cache labels used to speed up exporting labels events
	cachedLabels map[string]string
}

// Init .
func (ge *gitlabExporter) Init(conf core.Configuration) error {
	ge.conf = conf
	//TODO: initialize with multiple tokens
	ge.identityToken = make(map[string]string)
	ge.identityClient = make(map[string]*gitlab.Client)
	ge.cachedOperationIDs = make(map[string]string)
	ge.cachedLabels = make(map[string]string)

	return nil
}

// getIdentityClient return a gitlab v4 API client configured with the access token of the given identity.
// if no client were found it will initialize it from the known tokens map and cache it for next use
func (ge *gitlabExporter) getIdentityClient(id entity.Id) (*gitlab.Client, error) {
	client, ok := ge.identityClient[id.String()]
	if ok {
		return client, nil
	}

	// get token
	token, ok := ge.identityToken[id.String()]
	if !ok {
		return nil, ErrMissingIdentityToken
	}

	// create client
	client = buildClient(token)
	// cache client
	ge.identityClient[id.String()] = client

	return client, nil
}

// ExportAll export all event made by the current user to Gitlab
func (ge *gitlabExporter) ExportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	out := make(chan core.ExportResult)

	user, err := repo.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	ge.identityToken[user.Id().String()] = ge.conf[keyToken]

	// get repository node id
	ge.repositoryID = ge.conf[keyProjectID]

	go func() {
		defer close(out)

		var allIdentitiesIds []entity.Id
		for id := range ge.identityToken {
			allIdentitiesIds = append(allIdentitiesIds, entity.Id(id))
		}

		allBugsIds := repo.AllBugsIds()

		for _, id := range allBugsIds {
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
			} else {
				out <- core.NewExportNothing(id, "not an actor")
			}
		}
	}()

	return out, nil
}

// exportBug publish bugs and related events
func (ge *gitlabExporter) exportBug(ctx context.Context, b *cache.BugCache, since time.Time, out chan<- core.ExportResult) {
	snapshot := b.Snapshot()

	var err error
	var bugGitlabID int
	var bugGitlabIDString string
	var bugCreationId string

	//labels := make([]string, 0)

	// Special case:
	// if a user try to export a bug that is not already exported to Gitlab (or imported
	// from Gitlab) and we do not have the token of the bug author, there is nothing we can do.

	// skip bug if origin is not allowed
	origin, ok := snapshot.GetCreateMetadata(core.KeyOrigin)
	if ok && origin != target {
		out <- core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
		return
	}

	// first operation is always createOp
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	author := snapshot.Author

	// get gitlab bug ID
	gitlabID, ok := snapshot.GetCreateMetadata(keyGitlabId)
	if ok {
		_, ok := snapshot.GetCreateMetadata(keyGitlabUrl)
		if !ok {
			// if we find gitlab ID, gitlab URL must be found too
			err := fmt.Errorf("expected to find gitlab issue URL")
			out <- core.NewExportError(err, b.Id())
			return
		}

		out <- core.NewExportNothing(b.Id(), "bug already exported")

		// will be used to mark operation related to a bug as exported
		bugGitlabIDString = gitlabID
		bugGitlabID, err = strconv.Atoi(bugGitlabIDString)
		if err != nil {
			panic("unexpected gitlab id format")
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
		id, url, err := createGitlabIssue(ctx, client, ge.repositoryID, createOp.Title, createOp.Message)
		if err != nil {
			err := errors.Wrap(err, "exporting gitlab issue")
			out <- core.NewExportError(err, b.Id())
			return
		}

		idString := strconv.Itoa(id)
		out <- core.NewExportBug(b.Id())

		// mark bug creation operation as exported
		if err := markOperationAsExported(b, createOp.Id(), idString, url); err != nil {
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

	var actualLabels []string
	for _, op := range snapshot.Operations[1:] {
		// ignore SetMetadata operations
		if _, ok := op.(*bug.SetMetadataOperation); ok {
			continue
		}

		// ignore operations already existing in gitlab (due to import or export)
		// cache the ID of already exported or imported issues and events from Gitlab
		if id, ok := op.GetMetadata(keyGitlabId); ok {
			ge.cachedOperationIDs[op.Id().String()] = id
			out <- core.NewExportNothing(op.Id(), "already exported operation")
			continue
		}

		opAuthor := op.GetAuthor()
		client, err := ge.getIdentityClient(opAuthor.Id())
		if err != nil {
			out <- core.NewExportNothing(op.Id(), "missing operation author token")
			continue
		}

		var id int
		var idString, url string
		switch op.(type) {
		case *bug.AddCommentOperation:
			opr := op.(*bug.AddCommentOperation)

			// send operation to gitlab
			id, err = addCommentGitlabIssue(ctx, client, ge.repositoryID, bugGitlabID, opr.Message)
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
			opr := op.(*bug.EditCommentOperation)
			targetId := opr.Target.String()

			// Since gitlab doesn't consider the issue body as a comment
			if targetId == bugCreationId {

				// case bug creation operation: we need to edit the Gitlab issue
				if err := updateGitlabIssueBody(ctx, client, ge.repositoryID, bugGitlabID, opr.Message); err != nil {
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
					panic("unexpected error: comment id not found")
				}

				commentIDint, err := strconv.Atoi(commentID)
				if err != nil {
					panic("unexpected comment id format")
				}

				if err := editCommentGitlabIssue(ctx, client, ge.repositoryID, commentIDint, id, opr.Message); err != nil {
					err := errors.Wrap(err, "editing comment")
					out <- core.NewExportError(err, b.Id())
					return
				}

				out <- core.NewExportCommentEdition(op.Id())
				id = commentIDint
			}

		case *bug.SetStatusOperation:
			opr := op.(*bug.SetStatusOperation)
			if err := updateGitlabIssueStatus(ctx, client, idString, id, opr.Status); err != nil {
				err := errors.Wrap(err, "editing status")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportStatusChange(op.Id())
			id = bugGitlabID

		case *bug.SetTitleOperation:
			opr := op.(*bug.SetTitleOperation)
			if err := updateGitlabIssueTitle(ctx, client, ge.repositoryID, id, opr.Title); err != nil {
				err := errors.Wrap(err, "editing title")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportTitleEdition(op.Id())
			id = bugGitlabID

		case *bug.LabelChangeOperation:
			opr := op.(*bug.LabelChangeOperation)
			// we need to set the actual list of labels at each label change operation
			// because gitlab update issue requests need directly the latest list of the verison

			if len(opr.Added) != 0 {
				for _, label := range opr.Added {
					actualLabels = append(actualLabels, string(label))
				}
			}

			if len(opr.Removed) != 0 {
				var newActualLabels []string
				for _, label := range actualLabels {
					for _, l := range opr.Removed {
						if label == string(l) {
							continue
						}

						newActualLabels = append(newActualLabels, label)
					}
				}
				actualLabels = newActualLabels
			}

			if err := updateGitlabIssueLabels(ctx, client, ge.repositoryID, bugGitlabID, actualLabels); err != nil {
				err := errors.Wrap(err, "updating labels")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportLabelChange(op.Id())
			id = bugGitlabID
		default:
			panic("unhandled operation type case")
		}

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
	}
}

func markOperationAsExported(b *cache.BugCache, target entity.Id, gitlabID, gitlabURL string) error {
	_, err := b.SetMetadata(
		target,
		map[string]string{
			keyGitlabId:  gitlabID,
			keyGitlabUrl: gitlabURL,
		},
	)

	return err
}

func (ge *gitlabExporter) getGitlabLabelID(label string) (string, error) {
	id, ok := ge.cachedLabels[label]
	if !ok {
		return "", fmt.Errorf("non cached label")
	}

	return id, nil
}

// get label from gitlab
func (ge *gitlabExporter) loadLabelsFromGitlab(ctx context.Context, gc *gitlab.Client) error {
	page := 1
	for ; ; page++ {
		ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
		labels, _, err := gc.Labels.ListLabels(
			ge.repositoryID,
			&gitlab.ListLabelsOptions{
				Page: page,
			},
			gitlab.WithContext(ctx),
		)
		if err != nil {
			return err
		}

		if len(labels) == 0 {
			break
		}
		for _, label := range labels {
			ge.cachedLabels[label.Name] = strconv.Itoa(label.ID)
		}
	}

	return nil
}

func (ge *gitlabExporter) createGitlabLabel(ctx context.Context, gc *gitlab.Client, label bug.Label) error {
	client := buildClient(ge.conf[keyToken])

	// RGBA to hex color
	rgba := label.RGBA()
	hexColor := fmt.Sprintf("%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
	name := label.String()

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	_, _, err := client.Labels.CreateLabel(ge.repositoryID,
		&gitlab.CreateLabelOptions{
			Name:  &name,
			Color: &hexColor,
		},
		gitlab.WithContext(ctx),
	)

	return err
}

// create a gitlab. issue and return it ID
func createGitlabIssue(ctx context.Context, gc *gitlab.Client, repositoryID, title, body string) (int, string, error) {
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
		return 0, "", err
	}

	return issue.IID, issue.WebURL, nil
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
	_, _, err := gc.Issues.UpdateIssue(
		repositoryID, issueID,
		&gitlab.UpdateIssueOptions{
			Labels: labels,
		},
		gitlab.WithContext(ctx),
	)

	return err
}
