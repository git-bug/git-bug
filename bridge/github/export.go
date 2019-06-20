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

	"github.com/shurcooL/githubv4"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
)

// githubImporter implement the Importer interface
type githubExporter struct {
	gc   *githubv4.Client
	conf core.Configuration

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
	ge.gc = buildClient(conf["token"])
	ge.cachedIDs = make(map[string]string)
	ge.cachedLabels = make(map[string]string)
	return nil
}

// ExportAll export all event made by the current user to Github
func (ge *githubExporter) ExportAll(repo *cache.RepoCache, since time.Time) error {
	user, err := repo.GetUserIdentity()
	if err != nil {
		return err
	}

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

		// if identity participated in a bug
		for _, p := range snapshot.Participants {
			if p.Id() == user.Id() {
				// try to export the bug and it associated events
				if err := ge.exportBug(b, user.Identity, since); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// exportBug publish bugs and related events
func (ge *githubExporter) exportBug(b *cache.BugCache, user identity.Interface, since time.Time) error {
	snapshot := b.Snapshot()

	var bugGithubID string
	var bugGithubURL string
	var bugCreationHash string

	// Special case:
	// if a user try to export a bug that is not already exported to Github (or imported
	// from Github) and he is not the author of the bug. There is nothing we can do.

	// first operation is always createOp
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	bugAuthorID := createOp.OpBase.Author.Id()

	// get github bug ID
	githubID, ok := createOp.GetMetadata(keyGithubId)
	if ok {
		githubURL, ok := createOp.GetMetadata(keyGithubId)
		if !ok {
			// if we find github ID, github URL must be found too
			panic("expected to find github issue URL")
		}

		// will be used to mark operation related to a bug as exported
		bugGithubID = githubID
		bugGithubURL = githubURL

	} else if !ok && bugAuthorID == user.Id() {
		// create bug
		id, url, err := ge.createGithubIssue(ge.repositoryID, createOp.Title, createOp.Message)
		if err != nil {
			return fmt.Errorf("creating exporting github issue %v", err)
		}

		hash, err := createOp.Hash()
		if err != nil {
			return fmt.Errorf("comment hash: %v", err)
		}

		// mark bug creation operation as exported
		if err := markOperationAsExported(b, hash, id, url); err != nil {
			return fmt.Errorf("marking operation as exported: %v", err)
		}

		// cache bug github ID and URL
		bugGithubID = id
		bugGithubURL = url
	} else {
		// if bug is still not exported and user cannot author bug stop the execution

		//TODO: maybe print a warning ?
		// this is not an error
		return nil
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
			return fmt.Errorf("reading operation hash: %v", err)
		}

		// ignore imported (or exported) operations from github
		// cache the ID of already exported or imported issues and events from Github
		if id, ok := op.GetMetadata(keyGithubId); ok {
			ge.cachedIDs[hash.String()] = id
			continue
		}

		switch op.(type) {
		case *bug.AddCommentOperation:
			opr := op.(*bug.AddCommentOperation)

			// send operation to github
			id, url, err := ge.addCommentGithubIssue(bugGithubID, opr.Message)
			if err != nil {
				return fmt.Errorf("adding comment: %v", err)
			}

			hash, err := opr.Hash()
			if err != nil {
				return fmt.Errorf("comment hash: %v", err)
			}

			// mark operation as exported
			if err := markOperationAsExported(b, hash, id, url); err != nil {
				return fmt.Errorf("marking operation as exported: %v", err)
			}

		case *bug.EditCommentOperation:

			id := bugGithubID
			url := bugGithubURL
			opr := op.(*bug.EditCommentOperation)
			targetHash := opr.Target.String()

			// Since github doesn't consider the issue body as a comment
			if targetHash == bugCreationHash {
				// case bug creation operation: we need to edit the Github issue
				if err := ge.updateGithubIssueBody(bugGithubID, opr.Message); err != nil {
					return fmt.Errorf("editing issue: %v", err)
				}

			} else {
				// case comment edition operation: we need to edit the Github comment
				commentID, ok := ge.cachedIDs[targetHash]
				if !ok {
					panic("unexpected error: comment id not found")
				}

				eid, eurl, err := ge.editCommentGithubIssue(commentID, opr.Message)
				if err != nil {
					return fmt.Errorf("editing comment: %v", err)
				}

				// use comment id/url instead of issue id/url
				id = eid
				url = eurl
			}

			hash, err := opr.Hash()
			if err != nil {
				return fmt.Errorf("comment hash: %v", err)
			}

			// mark operation as exported
			if err := markOperationAsExported(b, hash, id, url); err != nil {
				return fmt.Errorf("marking operation as exported: %v", err)
			}

		case *bug.SetStatusOperation:
			opr := op.(*bug.SetStatusOperation)
			if err := ge.updateGithubIssueStatus(bugGithubID, opr.Status); err != nil {
				return fmt.Errorf("updating status %v: %v", bugGithubID, err)
			}

			hash, err := opr.Hash()
			if err != nil {
				return fmt.Errorf("comment hash: %v", err)
			}

			// mark operation as exported
			if err := markOperationAsExported(b, hash, bugGithubID, bugGithubURL); err != nil {
				return fmt.Errorf("marking operation as exported: %v", err)
			}

		case *bug.SetTitleOperation:
			opr := op.(*bug.SetTitleOperation)
			if err := ge.updateGithubIssueTitle(bugGithubID, opr.Title); err != nil {
				return fmt.Errorf("editing comment %v: %v", bugGithubID, err)
			}

			hash, err := opr.Hash()
			if err != nil {
				return fmt.Errorf("comment hash: %v", err)
			}

			// mark operation as exported
			if err := markOperationAsExported(b, hash, bugGithubID, bugGithubURL); err != nil {
				return fmt.Errorf("marking operation as exported: %v", err)
			}

		case *bug.LabelChangeOperation:
			opr := op.(*bug.LabelChangeOperation)
			if err := ge.updateGithubIssueLabels(bugGithubID, opr.Added, opr.Removed); err != nil {
				return fmt.Errorf("updating labels %v: %v", bugGithubID, err)
			}

			hash, err := opr.Hash()
			if err != nil {
				return fmt.Errorf("comment hash: %v", err)
			}

			// mark operation as exported
			if err := markOperationAsExported(b, hash, bugGithubID, bugGithubURL); err != nil {
				return fmt.Errorf("marking operation as exported: %v", err)
			}

		default:
			panic("unhandled operation type case")
		}

	}

	if err := b.CommitAsNeeded(); err != nil {
		return fmt.Errorf("bug commit: %v", err)
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
func (ge *githubExporter) getGithubLabelID(label string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/labels/%s", githubV3Url, ge.conf[keyOwner], ge.conf[keyProject], label)

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", ge.conf[keyToken]))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("error getting label: status code: %v", resp.StatusCode)
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

// create github label using api v3
func (ge *githubExporter) createGithubLabel(label, labelColor string) (string, error) {
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

// randomHexColor return a random hex color code
func randomHexColor() string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "fffff"
	}

	// fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)

	return hex.EncodeToString(bytes)
}

func (ge *githubExporter) getOrCreateGithubLabelID(repositoryID, label string) (string, error) {
	// try to get label id
	labelID, err := ge.getGithubLabelID(label)
	if err == nil {
		return labelID, nil
	}

	// random color
	//TODO: no random
	color := randomHexColor()

	// create label and return id
	labelID, err = ge.createGithubLabel(label, color)
	if err != nil {
		return "", err
	}

	return labelID, nil
}

func (ge *githubExporter) getLabelsIDs(repositoryID string, labels []bug.Label) ([]githubv4.ID, error) {
	ids := make([]githubv4.ID, 0, len(labels))
	var err error

	// check labels ids
	for _, l := range labels {
		label := string(l)

		id, ok := ge.cachedLabels[label]
		if !ok {
			// try to query label id
			id, err = ge.getOrCreateGithubLabelID(repositoryID, label)
			if err != nil {
				return nil, fmt.Errorf("get or create github label: %v", err)
			}

			// cache label id
			ge.cachedLabels[label] = id
		}

		ids = append(ids, githubv4.ID(id))
	}

	return ids, nil
}

// create a github issue and return it ID
func (ge *githubExporter) createGithubIssue(repositoryID, title, body string) (string, string, error) {
	m := &createIssueMutation{}
	input := &githubv4.CreateIssueInput{
		RepositoryID: repositoryID,
		Title:        githubv4.String(title),
		Body:         (*githubv4.String)(&body),
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return "", "", err
	}

	issue := m.CreateIssue.Issue
	return issue.ID, issue.URL, nil
}

// add a comment to an issue and return it ID
func (ge *githubExporter) addCommentGithubIssue(subjectID string, body string) (string, string, error) {
	m := &addCommentToIssueMutation{}
	input := &githubv4.AddCommentInput{
		SubjectID: subjectID,
		Body:      githubv4.String(body),
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return "", "", err
	}

	node := m.AddComment.CommentEdge.Node
	return node.ID, node.URL, nil
}

func (ge *githubExporter) editCommentGithubIssue(commentID, body string) (string, string, error) {
	m := &updateIssueCommentMutation{}
	input := &githubv4.UpdateIssueCommentInput{
		ID:   commentID,
		Body: githubv4.String(body),
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return "", "", err
	}

	comment := m.IssueComment
	return commentID, comment.URL, nil
}

func (ge *githubExporter) updateGithubIssueStatus(id string, status bug.Status) error {
	m := &updateIssueMutation{}

	// set state
	state := githubv4.IssueStateClosed
	if status == bug.OpenStatus {
		state = githubv4.IssueStateOpen
	}

	input := &githubv4.UpdateIssueInput{
		ID:    id,
		State: &state,
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return err
	}

	return nil
}

func (ge *githubExporter) updateGithubIssueBody(id string, body string) error {
	m := &updateIssueMutation{}
	input := &githubv4.UpdateIssueInput{
		ID:   id,
		Body: (*githubv4.String)(&body),
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return err
	}

	return nil
}

func (ge *githubExporter) updateGithubIssueTitle(id, title string) error {
	m := &updateIssueMutation{}
	input := &githubv4.UpdateIssueInput{
		ID:    id,
		Title: (*githubv4.String)(&title),
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return err
	}

	return nil
}

// update github issue labels
func (ge *githubExporter) updateGithubIssueLabels(labelableID string, added, removed []bug.Label) error {
	addedIDs, err := ge.getLabelsIDs(labelableID, added)
	if err != nil {
		return fmt.Errorf("getting added labels ids: %v", err)
	}

	m := &addLabelsToLabelableMutation{}
	inputAdd := &githubv4.AddLabelsToLabelableInput{
		LabelableID: labelableID,
		LabelIDs:    addedIDs,
	}

	// add labels
	if err := ge.gc.Mutate(context.TODO(), m, inputAdd, nil); err != nil {
		return err
	}

	removedIDs, err := ge.getLabelsIDs(labelableID, added)
	if err != nil {
		return fmt.Errorf("getting added labels ids: %v", err)
	}

	m2 := &removeLabelsFromLabelableMutation{}
	inputRemove := &githubv4.RemoveLabelsFromLabelableInput{
		LabelableID: labelableID,
		LabelIDs:    removedIDs,
	}

	// remove label labels
	if err := ge.gc.Mutate(context.TODO(), m2, inputRemove, nil); err != nil {
		return err
	}

	return nil
}
