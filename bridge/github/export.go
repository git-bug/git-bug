package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
)

var (
	ErrMissingIdentityToken = errors.New("missing identity token")
)

// githubExporter implement the Exporter interface
type githubExporter struct {
	conf core.Configuration

	// cache identities clients
	identityClient map[entity.Id]*githubv4.Client

	// map identities with their tokens
	identityToken map[entity.Id]string

	// github repository ID
	repositoryID string

	// cache identifiers used to speed up exporting operations
	// cleared for each bug
	cachedOperationIDs map[entity.Id]string

	// cache labels used to speed up exporting labels events
	cachedLabels map[string]string
}

// Init .
func (ge *githubExporter) Init(conf core.Configuration) error {
	ge.conf = conf
	//TODO: initialize with multiple tokens
	ge.identityToken = make(map[entity.Id]string)
	ge.identityClient = make(map[entity.Id]*githubv4.Client)
	ge.cachedOperationIDs = make(map[entity.Id]string)
	ge.cachedLabels = make(map[string]string)
	return nil
}

// getIdentityClient return a githubv4 API client configured with the access token of the given identity.
// if no client were found it will initialize it from the known tokens map and cache it for next use
func (ge *githubExporter) getIdentityClient(id entity.Id) (*githubv4.Client, error) {
	client, ok := ge.identityClient[id]
	if ok {
		return client, nil
	}

	// get token
	token, ok := ge.identityToken[id]
	if !ok {
		return nil, ErrMissingIdentityToken
	}

	// create client
	client = buildClient(token)
	// cache client
	ge.identityClient[id] = client

	return client, nil
}

// ExportAll export all event made by the current user to Github
func (ge *githubExporter) ExportAll(repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	out := make(chan core.ExportResult)

	user, err := repo.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	ge.identityToken[user.Id()] = ge.conf[keyToken]

	// get repository node id
	ge.repositoryID, err = getRepositoryNodeID(
		ge.conf[keyOwner],
		ge.conf[keyProject],
		ge.conf[keyToken],
	)

	if err != nil {
		return nil, err
	}

	go func() {
		defer close(out)

		var allIdentitiesIds []entity.Id
		for id := range ge.identityToken {
			allIdentitiesIds = append(allIdentitiesIds, id)
		}

		allBugsIds := repo.AllBugsIds()

		for _, id := range allBugsIds {
			b, err := repo.ResolveBug(id)
			if err != nil {
				out <- core.NewExportError(errors.Wrap(err, "can't load bug"), id)
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
				ge.exportBug(b, since, out)
			} else {
				out <- core.NewExportNothing(id, "not an actor")
			}
		}
	}()

	return out, nil
}

// exportBug publish bugs and related events
func (ge *githubExporter) exportBug(b *cache.BugCache, since time.Time, out chan<- core.ExportResult) {
	snapshot := b.Snapshot()

	var bugGithubID string
	var bugGithubURL string

	// Special case:
	// if a user try to export a bug that is not already exported to Github (or imported
	// from Github) and we do not have the token of the bug author, there is nothing we can do.

	// first operation is always createOp
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	author := snapshot.Author

	// skip bug if origin is not allowed
	origin, ok := snapshot.GetCreateMetadata(keyOrigin)
	if ok && origin != target {
		out <- core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
		return
	}

	// get github bug ID
	githubID, ok := snapshot.GetCreateMetadata(keyGithubId)
	if ok {
		githubURL, ok := snapshot.GetCreateMetadata(keyGithubUrl)
		if !ok {
			// if we find github ID, github URL must be found too
			err := fmt.Errorf("incomplete Github metadata: expected to find issue URL")
			out <- core.NewExportError(err, b.Id())
		}

		// extract owner and project
		owner, project, err := splitURL(githubURL)
		if err != nil {
			err := fmt.Errorf("bad project url: %v", err)
			out <- core.NewExportError(err, b.Id())
			return
		}

		// ignore issue comming from other repositories
		if owner != ge.conf[keyOwner] && project != ge.conf[keyProject] {
			out <- core.NewExportNothing(b.Id(), fmt.Sprintf("skipping issue from url:%s", githubURL))
			return
		}

		out <- core.NewExportNothing(b.Id(), "bug already exported")
		// will be used to mark operation related to a bug as exported
		bugGithubID = githubID
		bugGithubURL = githubURL

	} else {
		// check that we have a token for operation author
		client, err := ge.getIdentityClient(author.Id())
		if err != nil {
			// if bug is still not exported and we do not have the author stop the execution
			out <- core.NewExportNothing(b.Id(), fmt.Sprintf("missing author token"))
			return
		}

		// create bug
		id, url, err := createGithubIssue(client, ge.repositoryID, createOp.Title, createOp.Message)
		if err != nil {
			err := errors.Wrap(err, "exporting github issue")
			out <- core.NewExportError(err, b.Id())
			return
		}

		out <- core.NewExportBug(b.Id())

		// mark bug creation operation as exported
		if err := markOperationAsExported(b, createOp.Id(), id, url); err != nil {
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

		// cache bug github ID and URL
		bugGithubID = id
		bugGithubURL = url
	}

	// cache operation github id
	ge.cachedOperationIDs[createOp.Id()] = bugGithubID

	for _, op := range snapshot.Operations[1:] {
		// ignore SetMetadata operations
		if _, ok := op.(*bug.SetMetadataOperation); ok {
			continue
		}

		// ignore operations already existing in github (due to import or export)
		// cache the ID of already exported or imported issues and events from Github
		if id, ok := op.GetMetadata(keyGithubId); ok {
			ge.cachedOperationIDs[op.Id()] = id
			out <- core.NewExportNothing(op.Id(), "already exported operation")
			continue
		}

		opAuthor := op.GetAuthor()
		client, err := ge.getIdentityClient(opAuthor.Id())
		if err != nil {
			out <- core.NewExportNothing(op.Id(), "missing operation author token")
			continue
		}

		var id, url string
		switch op.(type) {
		case *bug.AddCommentOperation:
			opr := op.(*bug.AddCommentOperation)

			// send operation to github
			id, url, err = addCommentGithubIssue(client, bugGithubID, opr.Message)
			if err != nil {
				err := errors.Wrap(err, "adding comment")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportComment(op.Id())

			// cache comment id
			ge.cachedOperationIDs[op.Id()] = id

		case *bug.EditCommentOperation:

			opr := op.(*bug.EditCommentOperation)

			// Since github doesn't consider the issue body as a comment
			if opr.Target == createOp.Id() {

				// case bug creation operation: we need to edit the Github issue
				if err := updateGithubIssueBody(client, bugGithubID, opr.Message); err != nil {
					err := errors.Wrap(err, "editing issue")
					out <- core.NewExportError(err, b.Id())
					return
				}

				out <- core.NewExportCommentEdition(op.Id())

				id = bugGithubID
				url = bugGithubURL

			} else {

				// case comment edition operation: we need to edit the Github comment
				commentID, ok := ge.cachedOperationIDs[opr.Target]
				if !ok {
					panic("unexpected error: comment id not found")
				}

				eid, eurl, err := editCommentGithubIssue(client, commentID, opr.Message)
				if err != nil {
					err := errors.Wrap(err, "editing comment")
					out <- core.NewExportError(err, b.Id())
					return
				}

				out <- core.NewExportCommentEdition(op.Id())

				// use comment id/url instead of issue id/url
				id = eid
				url = eurl
			}

		case *bug.SetStatusOperation:
			opr := op.(*bug.SetStatusOperation)
			if err := updateGithubIssueStatus(client, bugGithubID, opr.Status); err != nil {
				err := errors.Wrap(err, "editing status")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportStatusChange(op.Id())

			id = bugGithubID
			url = bugGithubURL

		case *bug.SetTitleOperation:
			opr := op.(*bug.SetTitleOperation)
			if err := updateGithubIssueTitle(client, bugGithubID, opr.Title); err != nil {
				err := errors.Wrap(err, "editing title")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportTitleEdition(op.Id())

			id = bugGithubID
			url = bugGithubURL

		case *bug.LabelChangeOperation:
			opr := op.(*bug.LabelChangeOperation)
			if err := ge.updateGithubIssueLabels(client, bugGithubID, opr.Added, opr.Removed); err != nil {
				err := errors.Wrap(err, "updating labels")
				out <- core.NewExportError(err, b.Id())
				return
			}

			out <- core.NewExportLabelChange(op.Id())

			id = bugGithubID
			url = bugGithubURL

		default:
			panic("unhandled operation type case")
		}

		// mark operation as exported
		if err := markOperationAsExported(b, op.Id(), id, url); err != nil {
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

// getRepositoryNodeID request github api v3 to get repository node id
func getRepositoryNodeID(owner, project, token string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", githubV3Url, owner, project)

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error %v retrieving repository node id", resp.StatusCode)
	}

	aux := struct {
		NodeID string `json:"node_id"`
	}{}

	data, _ := ioutil.ReadAll(resp.Body)
	err = resp.Body.Close()
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return "", err
	}

	return aux.NodeID, nil
}

func markOperationAsExported(b *cache.BugCache, target entity.Id, githubID, githubURL string) error {
	_, err := b.SetMetadata(
		target,
		map[string]string{
			keyGithubId:  githubID,
			keyGithubUrl: githubURL,
		},
	)

	return err
}

// get label from github
func (ge *githubExporter) getGithubLabelID(gc *githubv4.Client, label string) (string, error) {
	q := &labelQuery{}
	variables := map[string]interface{}{
		"label": githubv4.String(label),
		"owner": githubv4.String(ge.conf[keyOwner]),
		"name":  githubv4.String(ge.conf[keyProject]),
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Query(ctx, q, variables); err != nil {
		return "", err
	}

	// if label id is empty, it means there is no such label in this Github repository
	if q.Repository.Label.ID == "" {
		return "", fmt.Errorf("label not found")
	}

	return q.Repository.Label.ID, nil
}

// create a new label and return it github id
// NOTE: since createLabel mutation is still in preview mode we use github api v3 to create labels
// see https://developer.github.com/v4/mutation/createlabel/ and https://developer.github.com/v4/previews/#labels-preview
func (ge *githubExporter) createGithubLabel(label, color string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/labels", githubV3Url, ge.conf[keyOwner], ge.conf[keyProject])

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	params := struct {
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}{
		Name:  label,
		Color: color,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", ge.conf[keyToken]))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error creating label: response status %v", resp.StatusCode)
	}

	aux := struct {
		ID     int    `json:"id"`
		NodeID string `json:"node_id"`
		Color  string `json:"color"`
	}{}

	data, _ = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return "", err
	}

	return aux.NodeID, nil
}

/**
// create github label using api v4
func (ge *githubExporter) createGithubLabelV4(gc *githubv4.Client, label, labelColor string) (string, error) {
	m := createLabelMutation{}
	input := createLabelInput{
		RepositoryID: ge.repositoryID,
		Name:         githubv4.String(label),
		Color:        githubv4.String(labelColor),
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Mutate(ctx, &m, input, nil); err != nil {
		return "", err
	}

	return m.CreateLabel.Label.ID, nil
}
*/

func (ge *githubExporter) getOrCreateGithubLabelID(gc *githubv4.Client, repositoryID string, label bug.Label) (string, error) {
	// try to get label id
	labelID, err := ge.getGithubLabelID(gc, string(label))
	if err == nil {
		return labelID, nil
	}

	// RGBA to hex color
	rgba := label.RGBA()
	hexColor := fmt.Sprintf("%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)

	labelID, err = ge.createGithubLabel(string(label), hexColor)
	if err != nil {
		return "", err
	}

	return labelID, nil
}

func (ge *githubExporter) getLabelsIDs(gc *githubv4.Client, repositoryID string, labels []bug.Label) ([]githubv4.ID, error) {
	ids := make([]githubv4.ID, 0, len(labels))
	var err error

	// check labels ids
	for _, label := range labels {
		id, ok := ge.cachedLabels[string(label)]
		if !ok {
			// try to query label id
			id, err = ge.getOrCreateGithubLabelID(gc, repositoryID, label)
			if err != nil {
				return nil, errors.Wrap(err, "get or create github label")
			}

			// cache label id
			ge.cachedLabels[string(label)] = id
		}

		ids = append(ids, githubv4.ID(id))
	}

	return ids, nil
}

// create a github issue and return it ID
func createGithubIssue(gc *githubv4.Client, repositoryID, title, body string) (string, string, error) {
	m := &createIssueMutation{}
	input := githubv4.CreateIssueInput{
		RepositoryID: repositoryID,
		Title:        githubv4.String(title),
		Body:         (*githubv4.String)(&body),
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Mutate(ctx, m, input, nil); err != nil {
		return "", "", err
	}

	issue := m.CreateIssue.Issue
	return issue.ID, issue.URL, nil
}

// add a comment to an issue and return it ID
func addCommentGithubIssue(gc *githubv4.Client, subjectID string, body string) (string, string, error) {
	m := &addCommentToIssueMutation{}
	input := githubv4.AddCommentInput{
		SubjectID: subjectID,
		Body:      githubv4.String(body),
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Mutate(ctx, m, input, nil); err != nil {
		return "", "", err
	}

	node := m.AddComment.CommentEdge.Node
	return node.ID, node.URL, nil
}

func editCommentGithubIssue(gc *githubv4.Client, commentID, body string) (string, string, error) {
	m := &updateIssueCommentMutation{}
	input := githubv4.UpdateIssueCommentInput{
		ID:   commentID,
		Body: githubv4.String(body),
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Mutate(ctx, m, input, nil); err != nil {
		return "", "", err
	}

	return commentID, m.UpdateIssueComment.IssueComment.URL, nil
}

func updateGithubIssueStatus(gc *githubv4.Client, id string, status bug.Status) error {
	m := &updateIssueMutation{}

	// set state
	var state githubv4.IssueState

	switch status {
	case bug.OpenStatus:
		state = githubv4.IssueStateOpen
	case bug.ClosedStatus:
		state = githubv4.IssueStateClosed
	default:
		panic("unknown bug state")
	}

	input := githubv4.UpdateIssueInput{
		ID:    id,
		State: &state,
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Mutate(ctx, m, input, nil); err != nil {
		return err
	}

	return nil
}

func updateGithubIssueBody(gc *githubv4.Client, id string, body string) error {
	m := &updateIssueMutation{}
	input := githubv4.UpdateIssueInput{
		ID:   id,
		Body: (*githubv4.String)(&body),
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Mutate(ctx, m, input, nil); err != nil {
		return err
	}

	return nil
}

func updateGithubIssueTitle(gc *githubv4.Client, id, title string) error {
	m := &updateIssueMutation{}
	input := githubv4.UpdateIssueInput{
		ID:    id,
		Title: (*githubv4.String)(&title),
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Mutate(ctx, m, input, nil); err != nil {
		return err
	}

	return nil
}

// update github issue labels
func (ge *githubExporter) updateGithubIssueLabels(gc *githubv4.Client, labelableID string, added, removed []bug.Label) error {
	var errs []string
	var wg sync.WaitGroup

	parentCtx := context.Background()

	if len(added) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			addedIDs, err := ge.getLabelsIDs(gc, labelableID, added)
			if err != nil {
				errs = append(errs, errors.Wrap(err, "getting added labels ids").Error())
				return
			}

			m := &addLabelsToLabelableMutation{}
			inputAdd := githubv4.AddLabelsToLabelableInput{
				LabelableID: labelableID,
				LabelIDs:    addedIDs,
			}

			ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
			defer cancel()

			// add labels
			if err := gc.Mutate(ctx, m, inputAdd, nil); err != nil {
				errs = append(errs, err.Error())
			}
		}()
	}

	if len(removed) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			removedIDs, err := ge.getLabelsIDs(gc, labelableID, removed)
			if err != nil {
				errs = append(errs, errors.Wrap(err, "getting added labels ids").Error())
				return
			}

			m2 := &removeLabelsFromLabelableMutation{}
			inputRemove := githubv4.RemoveLabelsFromLabelableInput{
				LabelableID: labelableID,
				LabelIDs:    removedIDs,
			}

			ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
			defer cancel()

			// remove label labels
			if err := gc.Mutate(ctx, m2, inputRemove, nil); err != nil {
				errs = append(errs, err.Error())
			}
		}()
	}

	wg.Wait()

	if len(errs) == 0 {
		return nil
	}

	return fmt.Errorf("label change error: %v", strings.Join(errs, "\n"))
}
