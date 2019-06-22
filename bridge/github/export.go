package github

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/shurcooL/githubv4"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/git"
)

var (
	ErrMissingIdentityToken = errors.New("missing identity token")
)

// githubImporter implement the Importer interface
type githubExporter struct {
	conf core.Configuration

	// number of exported bugs
	exportedBugs int

	// export only bugs taged with one of these origins
	onlyOrigins []string

	// cache identities clients
	identityClient map[string]*githubv4.Client

	// map identity with their tokens
	identityToken map[string]string

	// github repository ID
	repositoryID string

	// cache identifiers used to speed up exporting operations
	// cleared for each bug
	cachedIDs map[string]string

	// cache labels used to speed up exporting labels events
	cachedLabels map[string]string
}

// Init .
func (ge *githubExporter) Init(conf core.Configuration) error {
	ge.conf = conf
	//TODO: initialize with multiple tokens
	ge.identityToken = make(map[string]string)
	ge.identityClient = make(map[string]*githubv4.Client)
	ge.cachedIDs = make(map[string]string)
	ge.cachedLabels = make(map[string]string)
	return nil
}

func (ge *githubExporter) allowOrigin(origin string) bool {
	if ge.onlyOrigins == nil {
		return true
	}

	for _, o := range ge.onlyOrigins {
		if origin == o {
			return true
		}
	}

	return false
}

// getIdentityClient return an identity github api v4 client
// if no client were found it will initilize it from the known tokens map and cache it for next it use
func (ge *githubExporter) getIdentityClient(id string) (*githubv4.Client, error) {
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
func (ge *githubExporter) ExportAll(repo *cache.RepoCache, since time.Time) error {
	user, err := repo.GetUserIdentity()
	if err != nil {
		return err
	}

	ge.identityToken[user.Id()] = ge.conf[keyToken]

	// get repository node id
	ge.repositoryID, err = getRepositoryNodeID(
		ge.conf[keyOwner],
		ge.conf[keyProject],
		ge.conf[keyToken],
	)

	if err != nil {
		return err
	}

	allBugsIds := repo.AllBugsIds()
bugLoop:
	for _, id := range allBugsIds {
		b, err := repo.ResolveBug(id)
		if err != nil {
			return err
		}

		snapshot := b.Snapshot()

		// ignore issues created before since date
		if snapshot.CreatedAt.Before(since) {
			continue
		}

		for _, p := range snapshot.Participants {
			// if we have a token for one of the participants
			for userId := range ge.identityToken {
				if p.Id() == userId {
					// try to export the bug and it associated events
					if err := ge.exportBug(b, since); err != nil {
						return err
					}

					continue bugLoop
				}
			}
		}
	}

	return nil
}

// exportBug publish bugs and related events
func (ge *githubExporter) exportBug(b *cache.BugCache, since time.Time) error {
	snapshot := b.Snapshot()

	var bugGithubID string
	var bugGithubURL string
	var bugCreationHash string

	// Special case:
	// if a user try to export a bug that is not already exported to Github (or imported
	// from Github) and he is not the author of the bug. There is nothing we can do.

	// first operation is always createOp
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	author := createOp.GetAuthor()

	// skip bug if origin is not allowed
	origin, ok := createOp.GetMetadata(keyOrigin)
	if ok && !ge.allowOrigin(origin) {
		// TODO print a warn ?
		return nil
	}

	// get github bug ID
	githubID, ok := createOp.GetMetadata(keyGithubId)
	if ok {
		githubURL, ok := createOp.GetMetadata(keyGithubUrl)
		if !ok {
			// if we find github ID, github URL must be found too
			panic("expected to find github issue URL")
		}
		// will be used to mark operation related to a bug as exported
		bugGithubID = githubID
		bugGithubURL = githubURL

	} else {
		// check that we have a token for operation author
		client, err := ge.getIdentityClient(author.Id())
		if err != nil {
			// if bug is still not exported and we do not have the author stop the execution

			fmt.Println("warning: skipping issue due to missing token for bug creator")
			// this is not an error, don't export bug
			return nil
		}

		// create bug
		id, url, err := createGithubIssue(client, ge.repositoryID, createOp.Title, createOp.Message)
		if err != nil {
			return errors.Wrap(err, "exporting github issue")
		}

		hash, err := createOp.Hash()
		if err != nil {
			return errors.Wrap(err, "comment hash")
		}

		// mark bug creation operation as exported
		if err := markOperationAsExported(b, hash, id, url); err != nil {
			return errors.Wrap(err, "marking operation as exported")
		}

		// commit operation to avoid creating multiple issues with multiple pushes
		if err := b.CommitAsNeeded(); err != nil {
			return errors.Wrap(err, "bug commit")
		}

		// cache bug github ID and URL
		bugGithubID = id
		bugGithubURL = url
	}

	// get createOp hash
	hash, err := createOp.Hash()
	if err != nil {
		return err
	}

	bugCreationHash = hash.String()

	// cache operation github id
	ge.cachedIDs[bugCreationHash] = bugGithubID

	for _, op := range snapshot.Operations[1:] {
		// ignore SetMetadata operations
		if _, ok := op.(*bug.SetMetadataOperation); ok {
			continue
		}

		// get operation hash
		hash, err := op.Hash()
		if err != nil {
			return errors.Wrap(err, "operation hash")
		}

		// ignore imported (or exported) operations from github
		// cache the ID of already exported or imported issues and events from Github
		if id, ok := op.GetMetadata(keyGithubId); ok {
			ge.cachedIDs[hash.String()] = id
			continue
		}

		opAuthor := op.GetAuthor()
		client, err := ge.getIdentityClient(opAuthor.Id())
		if err != nil {
			// don't export operation
			continue
		}

		var id, url string
		switch op.(type) {
		case *bug.AddCommentOperation:
			opr := op.(*bug.AddCommentOperation)

			// send operation to github
			id, url, err = addCommentGithubIssue(client, bugGithubID, opr.Message)
			if err != nil {
				return errors.Wrap(err, "adding comment")
			}

			hash, err = opr.Hash()
			if err != nil {
				return errors.Wrap(err, "comment hash")
			}

		case *bug.EditCommentOperation:

			opr := op.(*bug.EditCommentOperation)
			targetHash := opr.Target.String()

			// Since github doesn't consider the issue body as a comment
			if targetHash == bugCreationHash {

				// case bug creation operation: we need to edit the Github issue
				if err := updateGithubIssueBody(client, bugGithubID, opr.Message); err != nil {
					return errors.Wrap(err, "editing issue")
				}

				id = bugGithubID
				url = bugGithubURL

			} else {

				// case comment edition operation: we need to edit the Github comment
				commentID, ok := ge.cachedIDs[targetHash]
				if !ok {
					panic("unexpected error: comment id not found")
				}

				eid, eurl, err := editCommentGithubIssue(client, commentID, opr.Message)
				if err != nil {
					return errors.Wrap(err, "editing comment")
				}

				// use comment id/url instead of issue id/url
				id = eid
				url = eurl
			}

			hash, err = opr.Hash()
			if err != nil {
				return errors.Wrap(err, "comment hash")
			}

		case *bug.SetStatusOperation:
			opr := op.(*bug.SetStatusOperation)
			if err := updateGithubIssueStatus(client, bugGithubID, opr.Status); err != nil {
				return errors.Wrap(err, "editing status")
			}

			hash, err = opr.Hash()
			if err != nil {
				return errors.Wrap(err, "set status operation hash")
			}

			id = bugGithubID
			url = bugGithubURL

		case *bug.SetTitleOperation:
			opr := op.(*bug.SetTitleOperation)
			if err := updateGithubIssueTitle(client, bugGithubID, opr.Title); err != nil {
				return errors.Wrap(err, "editing title")
			}

			hash, err = opr.Hash()
			if err != nil {
				return errors.Wrap(err, "set title operation hash")
			}

			id = bugGithubID
			url = bugGithubURL

		case *bug.LabelChangeOperation:
			opr := op.(*bug.LabelChangeOperation)
			if err := ge.updateGithubIssueLabels(client, bugGithubID, opr.Added, opr.Removed); err != nil {
				return errors.Wrap(err, "updating labels")
			}

			hash, err = opr.Hash()
			if err != nil {
				return errors.Wrap(err, "label change operation hash")
			}

			id = bugGithubID
			url = bugGithubURL

		default:
			panic("unhandled operation type case")
		}

		// mark operation as exported
		if err := markOperationAsExported(b, hash, id, url); err != nil {
			return errors.Wrap(err, "marking operation as exported")
		}

		if err := b.CommitAsNeeded(); err != nil {
			return errors.Wrap(err, "bug commit")
		}
	}

	return nil
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
		return "", fmt.Errorf("error retrieving repository node id %v", resp.StatusCode)
	}

	aux := struct {
		NodeID string `json:"node_id"`
	}{}

	data, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return "", err
	}

	return aux.NodeID, nil
}

func markOperationAsExported(b *cache.BugCache, target git.Hash, githubID, githubURL string) error {
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
	q := labelQuery{}
	variables := map[string]interface{}{"name": label}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Query(ctx, &q, variables); err != nil {
		return "", err
	}

	return q.Repository.Label.ID, nil
}

func (ge *githubExporter) createGithubLabel(gc *githubv4.Client, label, labelColor string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/labels", githubV3Url, ge.conf[keyOwner], ge.conf[keyProject])

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	req, err := http.NewRequest("POST", url, nil)
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
		ID     string `json:"id"`
		NodeID string `json:"node_id"`
		Color  string `json:"color"`
	}{}

	data, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return "", err
	}

	return aux.NodeID, nil
}

// create github label using api v4
func (ge *githubExporter) createGithubLabelV4(gc *githubv4.Client, label, labelColor string) (string, error) {
	m := &createLabelMutation{}
	input := createLabelInput{
		RepositoryID: ge.repositoryID,
		Name:         githubv4.String(label),
		Color:        githubv4.String(labelColor),
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel()

	if err := gc.Mutate(ctx, m, input, nil); err != nil {
		return "", err
	}

	return m.CreateLabel.Label.ID, nil
}

// randomHexColor return a random hex color code
func randomHexColor() string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "fffff"
	}

	return hex.EncodeToString(bytes)
}

func (ge *githubExporter) getOrCreateGithubLabelID(gc *githubv4.Client, repositoryID, label string) (string, error) {
	// try to get label id
	labelID, err := ge.getGithubLabelID(gc, label)
	if err == nil {
		return labelID, nil
	}

	// random color
	//TODO: no random
	color := randomHexColor()

	// create label and return id
	labelID, err = ge.createGithubLabel(gc, label, color)
	if err != nil {
		return "", err
	}

	return labelID, nil
}

func (ge *githubExporter) getLabelsIDs(gc *githubv4.Client, repositoryID string, labels []bug.Label) ([]githubv4.ID, error) {
	ids := make([]githubv4.ID, 0, len(labels))
	var err error

	// check labels ids
	for _, l := range labels {
		label := string(l)

		id, ok := ge.cachedLabels[label]
		if !ok {
			// try to query label id
			id, err = ge.getOrCreateGithubLabelID(gc, repositoryID, label)
			if err != nil {
				return nil, errors.Wrap(err, "get or create github label")
			}

			// cache label id
			ge.cachedLabels[label] = id
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

	comment := m.IssueComment
	return commentID, comment.URL, nil
}

func updateGithubIssueStatus(gc *githubv4.Client, id string, status bug.Status) error {
	m := &updateIssueMutation{}

	// set state
	state := githubv4.IssueStateClosed
	if status == bug.OpenStatus {
		state = githubv4.IssueStateOpen
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

	addedIDs, err := ge.getLabelsIDs(gc, labelableID, added)
	if err != nil {
		return errors.Wrap(err, "getting added labels ids")
	}

	m := &addLabelsToLabelableMutation{}
	inputAdd := githubv4.AddLabelsToLabelableInput{
		LabelableID: labelableID,
		LabelIDs:    addedIDs,
	}

	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, defaultTimeout)

	// add labels
	if err := gc.Mutate(ctx, m, inputAdd, nil); err != nil {
		cancel()
		return err
	}

	cancel()
	removedIDs, err := ge.getLabelsIDs(gc, labelableID, added)
	if err != nil {
		return errors.Wrap(err, "getting added labels ids")
	}

	m2 := &removeLabelsFromLabelableMutation{}
	inputRemove := githubv4.RemoveLabelsFromLabelableInput{
		LabelableID: labelableID,
		LabelIDs:    removedIDs,
	}

	ctx2, cancel2 := context.WithTimeout(parentCtx, defaultTimeout)
	defer cancel2()

	// remove label labels
	if err := gc.Mutate(ctx2, m2, inputRemove, nil); err != nil {
		return err
	}

	return nil
}
