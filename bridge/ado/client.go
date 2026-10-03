package ado

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// apiVersion is used for the stable work item tracking endpoints.
	apiVersion = "7.1"
	// apiVersionComments is required for the work item Comments API, which is
	// still flagged as preview by Azure DevOps.
	apiVersionComments = "7.1-preview.4"

	// getWorkItemsBatchSize is the maximum number of ids accepted by the
	// "get list of work items" endpoint.
	getWorkItemsBatchSize = 200

	jsonPatchContentType = "application/json-patch+json"
	jsonContentType      = "application/json"
)

// Client is a minimal Azure DevOps REST client covering only the work item
// endpoints the bridge needs. It authenticates with a PAT sent as HTTP Basic
// auth (empty username, PAT as password).
type Client struct {
	baseURL      string // instance root, e.g. https://dev.azure.com
	organization string
	project      string
	token        string
	http         *http.Client
	ctx          context.Context
}

// NewClient builds a new Azure DevOps client.
func NewClient(ctx context.Context, baseURL, organization, project, token string) *Client {
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		organization: organization,
		project:      project,
		token:        token,
		http:         &http.Client{Timeout: defaultTimeout},
		ctx:          ctx,
	}
}

func (c *Client) orgURL() string {
	return c.baseURL + "/" + url.PathEscape(c.organization)
}

func (c *Client) projectURL() string {
	return c.orgURL() + "/" + url.PathEscape(c.project)
}

// do performs an HTTP request and decodes a JSON response into out (if non-nil).
func (c *Client) do(method, rawURL, contentType string, body []byte, out interface{}) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(c.ctx, method, rawURL, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(":"+c.token)))
	req.Header.Set("Accept", jsonContentType)
	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("azure devops API %s %s: %s: %s",
			method, rawURL, resp.Status, strings.TrimSpace(string(data)))
	}

	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

/*
 * Data model
 */

// Identity is an Azure DevOps identity reference. Azure DevOps returns identity
// fields either as an object or, on older field payloads, as a "Name <email>"
// string, so UnmarshalJSON accepts both.
type Identity struct {
	DisplayName string
	UniqueName  string
	ID          string
	Descriptor  string
}

func (i *Identity) UnmarshalJSON(data []byte) error {
	var obj struct {
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
		ID          string `json:"id"`
		Descriptor  string `json:"descriptor"`
	}
	if err := json.Unmarshal(data, &obj); err == nil &&
		(obj.DisplayName != "" || obj.UniqueName != "" || obj.ID != "") {
		i.DisplayName = obj.DisplayName
		i.UniqueName = obj.UniqueName
		i.ID = obj.ID
		i.Descriptor = obj.Descriptor
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	i.DisplayName, i.UniqueName = parseDisplayString(s)
	return nil
}

// Key returns a stable unique key for the identity, preferring the email.
func (i Identity) Key() string {
	switch {
	case i.UniqueName != "":
		return i.UniqueName
	case i.Descriptor != "":
		return i.Descriptor
	case i.ID != "":
		return i.ID
	default:
		return i.DisplayName
	}
}

// Email returns a best-effort email for the identity, or an empty string.
func (i Identity) Email() string {
	if strings.Contains(i.UniqueName, "@") {
		return i.UniqueName
	}
	return ""
}

// Project holds the subset of Azure DevOps project fields the bridge uses.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// WorkItemFields holds the System.* fields the bridge reads.
type WorkItemFields struct {
	Title        string    `json:"System.Title"`
	Description  string    `json:"System.Description"`
	State        string    `json:"System.State"`
	WorkItemType string    `json:"System.WorkItemType"`
	Tags         string    `json:"System.Tags"`
	CreatedBy    Identity  `json:"System.CreatedBy"`
	CreatedDate  time.Time `json:"System.CreatedDate"`
	ChangedDate  time.Time `json:"System.ChangedDate"`
	AssignedTo   Identity  `json:"System.AssignedTo"`
}

// WorkItem is a single Azure DevOps work item.
type WorkItem struct {
	ID     int            `json:"id"`
	Rev    int            `json:"rev"`
	Fields WorkItemFields `json:"fields"`
	URL    string         `json:"url"`
}

// FieldUpdate is the old/new value pair for one field in a work item revision.
type FieldUpdate struct {
	OldValue interface{} `json:"oldValue"`
	NewValue interface{} `json:"newValue"`
}

// WorkItemUpdate is a single revision of a work item, listing the fields that
// changed in that revision.
type WorkItemUpdate struct {
	ID          int                    `json:"id"`
	Rev         int                    `json:"rev"`
	RevisedBy   Identity               `json:"revisedBy"`
	RevisedDate time.Time              `json:"revisedDate"`
	Fields      map[string]FieldUpdate `json:"fields"`
}

// Comment is a single work item comment.
type Comment struct {
	ID           int       `json:"id"`
	WorkItemID   int       `json:"workItemId"`
	Text         string    `json:"text"`
	CreatedBy    Identity  `json:"createdBy"`
	CreatedDate  time.Time `json:"createdDate"`
	ModifiedBy   Identity  `json:"modifiedBy"`
	ModifiedDate time.Time `json:"modifiedDate"`
	URL          string    `json:"url"`
}

// patchOp is a single JSON Patch operation used to create or update work items.
type patchOp struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

/*
 * Endpoints
 */

// GetProject fetches project metadata, used to verify access during Configure.
func (c *Client) GetProject() (*Project, error) {
	u := fmt.Sprintf("%s/_apis/projects/%s?api-version=%s",
		c.orgURL(), url.PathEscape(c.project), apiVersion)
	var p Project
	if err := c.do(http.MethodGet, u, "", nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SearchWorkItemIDs runs a WIQL query and returns the matching work item ids.
func (c *Client) SearchWorkItemIDs(wiql string) ([]int, error) {
	u := fmt.Sprintf("%s/_apis/wit/wiql?api-version=%s", c.projectURL(), apiVersion)
	body, err := json.Marshal(struct {
		Query string `json:"query"`
	}{Query: wiql})
	if err != nil {
		return nil, err
	}

	var resp struct {
		WorkItems []struct {
			ID int `json:"id"`
		} `json:"workItems"`
	}
	if err := c.do(http.MethodPost, u, jsonContentType, body, &resp); err != nil {
		return nil, err
	}

	ids := make([]int, len(resp.WorkItems))
	for i, wi := range resp.WorkItems {
		ids[i] = wi.ID
	}
	return ids, nil
}

// GetWorkItems fetches work items by id, transparently batching to respect the
// Azure DevOps per-call limit.
func (c *Client) GetWorkItems(ids []int) ([]WorkItem, error) {
	var out []WorkItem
	for start := 0; start < len(ids); start += getWorkItemsBatchSize {
		end := start + getWorkItemsBatchSize
		if end > len(ids) {
			end = len(ids)
		}

		strIDs := make([]string, 0, end-start)
		for _, id := range ids[start:end] {
			strIDs = append(strIDs, strconv.Itoa(id))
		}

		u := fmt.Sprintf("%s/_apis/wit/workitems?ids=%s&$expand=all&api-version=%s",
			c.orgURL(), strings.Join(strIDs, ","), apiVersion)

		var resp struct {
			Count int        `json:"count"`
			Value []WorkItem `json:"value"`
		}
		if err := c.do(http.MethodGet, u, "", nil, &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Value...)
	}
	return out, nil
}

// GetWorkItem fetches a single work item by id.
func (c *Client) GetWorkItem(id int) (*WorkItem, error) {
	u := fmt.Sprintf("%s/_apis/wit/workitems/%d?$expand=all&api-version=%s",
		c.orgURL(), id, apiVersion)
	var wi WorkItem
	if err := c.do(http.MethodGet, u, "", nil, &wi); err != nil {
		return nil, err
	}
	return &wi, nil
}

// GetUpdates returns the revision history of a work item.
func (c *Client) GetUpdates(workItemID int) ([]WorkItemUpdate, error) {
	u := fmt.Sprintf("%s/_apis/wit/workItems/%d/updates?api-version=%s",
		c.projectURL(), workItemID, apiVersion)
	var resp struct {
		Count int              `json:"count"`
		Value []WorkItemUpdate `json:"value"`
	}
	if err := c.do(http.MethodGet, u, "", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Value, nil
}

// GetComments returns all comments of a work item.
func (c *Client) GetComments(workItemID int) ([]Comment, error) {
	u := fmt.Sprintf("%s/_apis/wit/workItems/%d/comments?api-version=%s",
		c.projectURL(), workItemID, apiVersionComments)
	var resp struct {
		TotalCount int       `json:"totalCount"`
		Count      int       `json:"count"`
		Comments   []Comment `json:"comments"`
	}
	if err := c.do(http.MethodGet, u, "", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Comments, nil
}

// CreateWorkItem creates a new work item of the given type.
func (c *Client) CreateWorkItem(workItemType string, ops []patchOp) (*WorkItem, error) {
	u := fmt.Sprintf("%s/_apis/wit/workitems/$%s?api-version=%s",
		c.projectURL(), url.PathEscape(workItemType), apiVersion)
	body, err := json.Marshal(ops)
	if err != nil {
		return nil, err
	}
	var wi WorkItem
	if err := c.do(http.MethodPost, u, jsonPatchContentType, body, &wi); err != nil {
		return nil, err
	}
	return &wi, nil
}

// UpdateWorkItem applies a JSON Patch to an existing work item.
func (c *Client) UpdateWorkItem(id int, ops []patchOp) (*WorkItem, error) {
	u := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=%s",
		c.orgURL(), id, apiVersion)
	body, err := json.Marshal(ops)
	if err != nil {
		return nil, err
	}
	var wi WorkItem
	if err := c.do(http.MethodPatch, u, jsonPatchContentType, body, &wi); err != nil {
		return nil, err
	}
	return &wi, nil
}

// AddComment posts a new comment on a work item.
func (c *Client) AddComment(workItemID int, text string) (*Comment, error) {
	u := fmt.Sprintf("%s/_apis/wit/workItems/%d/comments?api-version=%s",
		c.projectURL(), workItemID, apiVersionComments)
	body, err := json.Marshal(struct {
		Text string `json:"text"`
	}{Text: text})
	if err != nil {
		return nil, err
	}
	var cm Comment
	if err := c.do(http.MethodPost, u, jsonContentType, body, &cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

// UpdateComment edits an existing work item comment.
func (c *Client) UpdateComment(workItemID, commentID int, text string) (*Comment, error) {
	u := fmt.Sprintf("%s/_apis/wit/workItems/%d/comments/%d?api-version=%s",
		c.projectURL(), workItemID, commentID, apiVersionComments)
	body, err := json.Marshal(struct {
		Text string `json:"text"`
	}{Text: text})
	if err != nil {
		return nil, err
	}
	var cm Comment
	if err := c.do(http.MethodPatch, u, jsonContentType, body, &cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

// DeleteWorkItem deletes a work item. When destroy is true the item is removed
// permanently, bypassing the recycle bin (this requires extra permission);
// otherwise it is moved to the recycle bin. The destroy query parameter is
// omitted entirely when false, because Azure DevOps treats its mere presence as
// a permanent-delete request and then denies it without the elevated permission.
func (c *Client) DeleteWorkItem(id int, destroy bool) error {
	u := fmt.Sprintf("%s/_apis/wit/workitems/%d?api-version=%s", c.orgURL(), id, apiVersion)
	if destroy {
		u += "&destroy=true"
	}
	return c.do(http.MethodDelete, u, "", nil, nil)
}
