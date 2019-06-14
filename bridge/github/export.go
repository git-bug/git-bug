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

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/shurcooL/githubv4"
)

const (
	keyGithubIdExport  = "github-id"
	keyGithubUrlExport = "github-url"
)

// githubImporter implement the Importer interface
type githubExporter struct {
	gc           *githubv4.Client
	conf         core.Configuration
	cachedLabels map[string]githubv4.ID
}

// Init .
func (ge *githubExporter) Init(conf core.Configuration) error {
	ge.gc = buildClient(conf["token"])
	ge.conf = conf
	ge.cachedLabels = make(map[string]githubv4.ID)
	return nil
}

// ExportAll export all event made by the current user to Github
func (ge *githubExporter) ExportAll(repo *cache.RepoCache, since time.Time) error {
	identity, err := repo.GetUserIdentity()
	if err != nil {
		return err
	}

	allBugsIds := repo.AllBugsIds()

	// collect bugs
	bugs := make([]*cache.BugCache, 0)
	for _, id := range allBugsIds {
		b, err := repo.ResolveBug(id)
		if err != nil {
			return err
		}

		snapshot := b.Snapshot()

		// ignore issues edited before since date
		if snapshot.LastEditTime().Before(since) {
			continue
		}

		// if identity participated in a bug
		for _, p := range snapshot.Participants {
			if p.Id() == identity.Id() {
				bugs = append(bugs, b)
			}
		}
	}

	// get repository node id
	repositoryID, err := getRepositoryNodeID(
		ge.conf[keyOwner],
		ge.conf[keyProject],
		ge.conf[keyToken],
	)
	if err != nil {
		return err
	}

	for _, b := range bugs {
		snapshot := b.Snapshot()
		bugGithubID := ""

		for _, op := range snapshot.Operations {
			// treat only operations after since date
			if op.Time().Before(since) {
				continue
			}

			// ignore SetMetadata operations
			if _, ok := op.(*bug.SetMetadataOperation); ok {
				continue
			}

			// ignore imported issues and operations from github
			if _, ok := op.GetMetadata(keyGithubId); ok {
				continue
			}

			// get operation hash
			hash, err := op.Hash()
			if err != nil {
				return fmt.Errorf("reading operation hash: %v", err)
			}

			// ignore already exported issues and operations
			if _, err := b.ResolveOperationWithMetadata("github-exported-op", hash.String()); err != nil {
				continue
			}

			switch op.(type) {
			case *bug.CreateOperation:
				opr := op.(*bug.CreateOperation)
				//TODO export files
				bugGithubID, err = ge.createGithubIssue(repositoryID, opr.Title, opr.Message)
				if err != nil {
					return fmt.Errorf("exporting bug %v: %v", b.HumanId(), err)
				}

			case *bug.AddCommentOperation:
				opr := op.(*bug.AddCommentOperation)
				bugGithubID, err = ge.addCommentGithubIssue(bugGithubID, opr.Message)
				if err != nil {
					return fmt.Errorf("adding comment %v: %v", "", err)
				}

			case *bug.EditCommentOperation:
				opr := op.(*bug.EditCommentOperation)
				if err := ge.editCommentGithubIssue(bugGithubID, opr.Message); err != nil {
					return fmt.Errorf("editing comment %v: %v", "", err)
				}

			case *bug.SetStatusOperation:
				opr := op.(*bug.SetStatusOperation)
				if err := ge.updateGithubIssueStatus(bugGithubID, opr.Status); err != nil {
					return fmt.Errorf("updating status %v: %v", bugGithubID, err)
				}

			case *bug.SetTitleOperation:
				opr := op.(*bug.SetTitleOperation)
				if err := ge.updateGithubIssueTitle(bugGithubID, opr.Title); err != nil {
					return fmt.Errorf("editing comment %v: %v", bugGithubID, err)
				}

			case *bug.LabelChangeOperation:
				opr := op.(*bug.LabelChangeOperation)
				if err := ge.updateGithubIssueLabels(bugGithubID, opr.Added, opr.Removed); err != nil {
					return fmt.Errorf("updating labels %v: %v", bugGithubID, err)
				}

			default:
				// ignore other type of operations
			}

		}

		if err := b.CommitAsNeeded(); err != nil {
			return fmt.Errorf("bug commit: %v", err)
		}

		fmt.Printf("debug: %v", bugGithubID)
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

func (ge *githubExporter) markOperationAsExported(b *cache.BugCache, opHash string) error {
	return nil
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

	return hex.EncodeToString(bytes)
}

func (ge *githubExporter) getOrCreateGithubLabelID(repositoryID, label string) (string, error) {
	// try to get label id
	labelID, err := ge.getGithubLabelID(label)
	if err == nil {
		return labelID, nil
	}

	// random color
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
func (ge *githubExporter) createGithubIssue(repositoryID, title, body string) (string, error) {
	m := &createIssueMutation{}
	input := &githubv4.CreateIssueInput{
		RepositoryID: repositoryID,
		Title:        githubv4.String(title),
		Body:         (*githubv4.String)(&body),
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return "", err
	}

	return m.CreateIssue.Issue.ID, nil
}

// add a comment to an issue and return it ID
func (ge *githubExporter) addCommentGithubIssue(subjectID string, body string) (string, error) {
	m := &addCommentToIssueMutation{}
	input := &githubv4.AddCommentInput{
		SubjectID: subjectID,
		Body:      githubv4.String(body),
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return "", err
	}

	return m.AddComment.CommentEdge.Node.ID, nil
}

func (ge *githubExporter) editCommentGithubIssue(commentID, body string) error {
	m := &updateIssueCommentMutation{}
	input := &githubv4.UpdateIssueCommentInput{
		ID:   commentID,
		Body: githubv4.String(body),
	}

	if err := ge.gc.Mutate(context.TODO(), m, input, nil); err != nil {
		return err
	}

	return nil
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

	m := &updateIssueMutation{}
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

	inputRemove := &githubv4.RemoveLabelsFromLabelableInput{
		LabelableID: labelableID,
		LabelIDs:    removedIDs,
	}

	// remove label labels
	if err := ge.gc.Mutate(context.TODO(), m, inputRemove, nil); err != nil {
		return err
	}

	return nil
}
