package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/input"
	"github.com/pkg/errors"
)

var errDone = errors.New("Iteration Done")
var errTransitionNotFound = errors.New("Transition not found")
var errTransitionNotAllowed = errors.New("Transition not allowed")

// =============================================================================
// Extended JSON
// =============================================================================

const TimeFormat = "2006-01-02T15:04:05.999999999Z0700"

// ParseTime parse an RFC3339 string with nanoseconds
func ParseTime(timeStr string) (time.Time, error) {
	out, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		out, err = time.Parse(TimeFormat, timeStr)
	}
	return out, err
}

// MyTime is just a time.Time with a JSON serialization
type MyTime struct {
	time.Time
}

// UnmarshalJSON parses an RFC3339 date string into a time object
// borrowed from: https://stackoverflow.com/a/39180230/141023
func (self *MyTime) UnmarshalJSON(data []byte) (err error) {
	str := string(data)

	// Get rid of the quotes "" around the value.
	// A second option would be to include them in the date format string
	// instead, like so below:
	//   time.Parse(`"`+time.RFC3339Nano+`"`, s)
	str = str[1 : len(str)-1]

	timeObj, err := ParseTime(str)
	self.Time = timeObj
	return
}

// =============================================================================
// JSON Objects
// =============================================================================

// Session credential cookie name/value pair received after logging in and
// required to be sent on all subsequent requests
type Session struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SessionResponse the JSON object returned from a /session query (login)
type SessionResponse struct {
	Session Session `json:"session"`
}

// SessionQuery the JSON object that is POSTed to the /session endpoint
// in order to login and get a session cookie
type SessionQuery struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// User the JSON object representing a JIRA user
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/user
type User struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	Key          string `json:"key"`
	Name         string `json:"name"`
}

// Comment the JSON object for a single comment item returned in a list of
// comments
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue-getComments
type Comment struct {
	ID           string `json:"id"`
	Body         string `json:"body"`
	Author       User   `json:"author"`
	UpdateAuthor User   `json:"updateAuthor"`
	Created      MyTime `json:"created"`
	Updated      MyTime `json:"updated"`
}

// CommentPage the JSON object holding a single page of comments returned
// either by direct query or within an issue query
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue-getComments
type CommentPage struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// NextStartAt return the index of the first item on the next page
func (self *CommentPage) NextStartAt() int {
	return self.StartAt + len(self.Comments)
}

// IsLastPage return true if there are no more items beyond this page
func (self *CommentPage) IsLastPage() bool {
	return self.NextStartAt() >= self.Total
}

// IssueFields the JSON object returned as the "fields" member of an issue.
// There are a very large number of fields and many of them are custom. We
// only grab a few that we need.
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue-getIssue
type IssueFields struct {
	Creator     User        `json:"creator"`
	Created     MyTime      `json:"created"`
	Description string      `json:"description"`
	Summary     string      `json:"summary"`
	Comments    CommentPage `json:"comment"`
	Labels      []string    `json:"labels"`
}

// ChangeLogItem "field-change" data within a changelog entry. A single
// changelog entry might effect multiple fields. For example, closing an issue
// generally requires a change in "status" and "resolution"
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue-getIssue
type ChangeLogItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldtype"`
	From       string `json:"from"`
	FromString string `json:"fromString"`
	To         string `json:"to"`
	ToString   string `json:"toString"`
}

// ChangeLogEntry One entry in a changelog
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue-getIssue
type ChangeLogEntry struct {
	ID      string          `json:"id"`
	Author  User            `json:"author"`
	Created MyTime          `json:"created"`
	Items   []ChangeLogItem `json:"items"`
}

// ChangeLogPage A collection of changes to issue metadata
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue-getIssue
type ChangeLogPage struct {
	StartAt    int              `json:"startAt"`
	MaxResults int              `json:"maxResults"`
	Total      int              `json:"total"`
	IsLast     bool             `json:"isLast"` // Cloud-only
	Entries    []ChangeLogEntry `json:"histories"`
	Values     []ChangeLogEntry `json:"values"`
}

// NextStartAt return the index of the first item on the next page
func (self *ChangeLogPage) NextStartAt() int {
	return self.StartAt + len(self.Entries)
}

// IsLastPage return true if there are no more items beyond this page
func (self *ChangeLogPage) IsLastPage() bool {
	// NOTE(josh): The "isLast" field is returned on JIRA cloud, but not on
	// JIRA server. If we can distinguish which one we are working with, we can
	// possibly rely on that instead.
	return self.NextStartAt() >= self.Total
}

// Issue Top-level object for an issue
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue-getIssue
type Issue struct {
	ID        string        `json:"id"`
	Key       string        `json:"key"`
	Fields    IssueFields   `json:"fields"`
	ChangeLog ChangeLogPage `json:"changelog"`
}

// SearchResult The result type from querying the search endpoint
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/search
type SearchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// NextStartAt return the index of the first item on the next page
func (self *SearchResult) NextStartAt() int {
	return self.StartAt + len(self.Issues)
}

// IsLastPage return true if there are no more items beyond this page
func (self *SearchResult) IsLastPage() bool {
	return self.NextStartAt() >= self.Total
}

// SearchRequest the JSON object POSTed to the /search endpoint
type SearchRequest struct {
	JQL        string   `json:"jql"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Fields     []string `json:"fields"`
}

// Project the JSON object representing a project. Note that we don't use all
// the fields so we have only implemented a couple.
type Project struct {
	ID  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}

// IssueType the JSON object representing an issue type (i.e. "bug", "task")
// Note that we don't use all the fields so we have only implemented a couple.
type IssueType struct {
	ID string `json:"id"`
}

// IssueCreateFields fields that are included in an IssueCreate request
type IssueCreateFields struct {
	Project     Project   `json:"project"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	IssueType   IssueType `json:"issuetype"`
}

// IssueCreate the JSON object that is POSTed to the /issue endpoint to create
// a new issue
type IssueCreate struct {
	Fields IssueCreateFields `json:"fields"`
}

// IssueCreateResult the JSON object returned after issue creation.
type IssueCreateResult struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

// CommentCreate the JSOn object that is POSTed to the /comment endpoint to
// create a new comment
type CommentCreate struct {
	Body string `json:"body"`
}

// StatusCategory the JSON object representing a status category
type StatusCategory struct {
	ID        int    `json:"id"`
	Key       string `json:"key"`
	Self      string `json:"self"`
	ColorName string `json:"colorName"`
	Name      string `json:"name"`
}

// Status the JSON object representing a status (i.e. "Open", "Closed")
type Status struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Self           string         `json:"self"`
	Description    string         `json:"description"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

// Transition the JSON object represenging a transition from one Status to
// another Status in a JIRA workflow
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   Status `json:"to"`
}

// TransitionList the JSON object returned from the /transitions endpoint
type TransitionList struct {
	Transitions []Transition `json:"transitions"`
}

// ServerInfo general server information returned by the /serverInfo endpoint.
// Notably `ServerTime` will tell you the time on the server.
type ServerInfo struct {
	BaseURL          string `json:"baseUrl"`
	Version          string `json:"version"`
	VersionNumbers   []int  `json:"versionNumbers"`
	BuildNumber      int    `json:"buildNumber"`
	BuildDate        MyTime `json:"buildDate"`
	ServerTime       MyTime `json:"serverTime"`
	ScmInfo          string `json:"scmInfo"`
	BuildPartnerName string `json:"buildPartnerName"`
	ServerTitle      string `json:"serverTitle"`
}

// =============================================================================
// REST Client
// =============================================================================

// ClientTransport wraps http.RoundTripper by adding a
// "Content-Type=application/json" header
type ClientTransport struct {
	underlyingTransport http.RoundTripper
	basicAuthString     string
}

// RoundTrip overrides the default by adding the content-type header
func (ct *ClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Content-Type", "application/json")
	if ct.basicAuthString != "" {
		req.Header.Add("Authorization",
			fmt.Sprintf("Basic %s", ct.basicAuthString))
	}

	return ct.underlyingTransport.RoundTrip(req)
}

func (ct *ClientTransport) SetCredentials(username string, token string) {
	credString := fmt.Sprintf("%s:%s", username, token)
	ct.basicAuthString = base64.StdEncoding.EncodeToString([]byte(credString))
}

// Client Thin wrapper around the http.Client providing jira-specific methods
// for APIendpoints
type Client struct {
	*http.Client
	serverURL string
	ctx       context.Context
}

// NewClient Construct a new client connected to the provided server and
// utilizing the given context for asynchronous events
func NewClient(serverURL string, ctx context.Context) *Client {
	cookiJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Transport: &ClientTransport{underlyingTransport: http.DefaultTransport},
		Jar:       cookiJar,
	}

	return &Client{client, serverURL, ctx}
}

// Login POST credentials to the /session endpoing and get a session cookie
func (client *Client) Login(conf core.Configuration) error {
	credType := conf[keyCredentialsType]

	if conf[keyCredentialsFile] != "" {
		content, err := ioutil.ReadFile(conf[keyCredentialsFile])
		if err != nil {
			return err
		}

		switch credType {
		case "SESSION":
			return client.RefreshSessionTokenRaw(content)
		case "TOKEN":
			var params SessionQuery
			err := json.Unmarshal(content, &params)
			if err != nil {
				return err
			}
			return client.SetTokenCredentials(params.Username, params.Password)
		}
		return fmt.Errorf("Unexpected credType: %s", credType)
	}

	username := conf[keyUsername]
	if username == "" {
		return fmt.Errorf(
			"Invalid configuration lacks both a username and credentials sidecar " +
				"path. At least one is required.")
	}

	password := conf[keyPassword]
	if password == "" {
		var err error
		password, err = input.PromptPassword("Password", "password", input.Required)
		if err != nil {
			return err
		}
	}

	switch credType {
	case "SESSION":
		return client.RefreshSessionToken(username, password)
	case "TOKEN":
		return client.SetTokenCredentials(username, password)
	}
	return fmt.Errorf("Unexpected credType: %s", credType)
}

// RefreshSessionToken formulate the JSON request object from the user
// credentials and POST it to the /session endpoing and get a session cookie
func (client *Client) RefreshSessionToken(username, password string) error {
	params := SessionQuery{
		Username: username,
		Password: password,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	return client.RefreshSessionTokenRaw(data)
}

// SetTokenCredentials POST credentials to the /session endpoing and get a
// session cookie
func (client *Client) SetTokenCredentials(username, password string) error {
	switch transport := client.Transport.(type) {
	case *ClientTransport:
		transport.SetCredentials(username, password)
	default:
		return fmt.Errorf("Invalid transport type")
	}
	return nil
}

// RefreshSessionTokenRaw POST credentials to the /session endpoing and get a
// session cookie
func (client *Client) RefreshSessionTokenRaw(credentialsJSON []byte) error {
	postURL := fmt.Sprintf("%s/rest/auth/1/session", client.serverURL)

	req, err := http.NewRequest("POST", postURL, bytes.NewBuffer(credentialsJSON))
	if err != nil {
		return err
	}

	urlobj, err := url.Parse(client.serverURL)
	if err != nil {
		fmt.Printf("Failed to parse %s\n", client.serverURL)
	} else {
		// Clear out cookies
		client.Jar.SetCookies(urlobj, []*http.Cookie{})
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	response, err := client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf(
			"error creating token %v: %s", response.StatusCode, content)
	}

	data, _ := ioutil.ReadAll(response.Body)
	var aux SessionResponse
	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	var cookies []*http.Cookie
	cookie := &http.Cookie{
		Name:  aux.Session.Name,
		Value: aux.Session.Value,
	}
	cookies = append(cookies, cookie)
	client.Jar.SetCookies(urlobj, cookies)

	return nil
}

// =============================================================================
// Endpoint Wrappers
// =============================================================================

// Search Perform an issue a JQL search on the /search endpoint
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/search
func (client *Client) Search(jql string, maxResults int, startAt int) (*SearchResult, error) {
	url := fmt.Sprintf("%s/rest/api/2/search", client.serverURL)

	requestBody, err := json.Marshal(SearchRequest{
		JQL:        jql,
		StartAt:    startAt,
		MaxResults: maxResults,
		Fields: []string{
			"comment",
			"created",
			"creator",
			"description",
			"labels",
			"status",
			"summary"}})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"HTTP response %d, query was %s, %s", response.StatusCode,
			url, requestBody)
		return nil, err
	}

	var message SearchResult

	data, _ := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &message)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &message, nil
}

// SearchIterator cursor within paginated results from the /search endpoint
type SearchIterator struct {
	client       *Client
	jql          string
	searchResult *SearchResult
	Err          error

	pageSize int
	itemIdx  int
}

// HasError returns true if the iterator is holding an error
func (si *SearchIterator) HasError() bool {
	if si.Err == errDone {
		return false
	}
	if si.Err == nil {
		return false
	}
	return true
}

// HasNext returns true if there is another item available in the result set
func (si *SearchIterator) HasNext() bool {
	return si.Err == nil && si.itemIdx < len(si.searchResult.Issues)
}

// Next Return the next item in the result set and advance the iterator.
// Advancing the iterator may require fetching a new page.
func (si *SearchIterator) Next() *Issue {
	if si.Err != nil {
		return nil
	}

	issue := si.searchResult.Issues[si.itemIdx]
	if si.itemIdx+1 < len(si.searchResult.Issues) {
		// We still have an item left in the currently cached page
		si.itemIdx++
	} else {
		if si.searchResult.IsLastPage() {
			si.Err = errDone
		} else {
			// There are still more pages to fetch, so fetch the next page and
			// cache it
			si.searchResult, si.Err = si.client.Search(
				si.jql, si.pageSize, si.searchResult.NextStartAt())
			// NOTE(josh): we don't deal with the error now, we just cache it.
			// HasNext() will return false and the caller can check the error
			// afterward.
			si.itemIdx = 0
		}
	}
	return &issue
}

// IterSearch return an iterator over paginated results for a JQL search
func (client *Client) IterSearch(jql string, pageSize int) *SearchIterator {
	result, err := client.Search(jql, pageSize, 0)

	iter := &SearchIterator{
		client:       client,
		jql:          jql,
		searchResult: result,
		Err:          err,
		pageSize:     pageSize,
		itemIdx:      0,
	}

	return iter
}

// GetIssue fetches an issue object via the /issue/{IssueIdOrKey} endpoint
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue
func (client *Client) GetIssue(idOrKey string, fields []string, expand []string,
	properties []string) (*Issue, error) {

	url := fmt.Sprintf("%s/rest/api/2/issue/%s", client.serverURL, idOrKey)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err := fmt.Errorf("Creating request %v", err)
		return nil, err
	}

	query := request.URL.Query()
	if len(fields) > 0 {
		query.Add("fields", strings.Join(fields, ","))
	}
	if len(expand) > 0 {
		query.Add("expand", strings.Join(expand, ","))
	}
	if len(properties) > 0 {
		query.Add("properties", strings.Join(properties, ","))
	}
	request.URL.RawQuery = query.Encode()

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"HTTP response %d, query was %s", response.StatusCode,
			request.URL.String())
		return nil, err
	}

	var issue Issue

	data, _ := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &issue)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &issue, nil
}

// GetComments returns a page of comments via the issue/{IssueIdOrKey}/comment
// endpoint
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue-getComment
func (client *Client) GetComments(idOrKey string, maxResults int, startAt int) (*CommentPage, error) {
	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s/comment", client.serverURL, idOrKey)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err := fmt.Errorf("Creating request %v", err)
		return nil, err
	}

	query := request.URL.Query()
	if maxResults > 0 {
		query.Add("maxResults", fmt.Sprintf("%d", maxResults))
	}
	if startAt > 0 {
		query.Add("startAt", fmt.Sprintf("%d", startAt))
	}
	request.URL.RawQuery = query.Encode()

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"HTTP response %d, query was %s", response.StatusCode,
			request.URL.String())
		return nil, err
	}

	var comments CommentPage

	data, _ := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &comments)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &comments, nil
}

// CommentIterator cursor within paginated results from the /comment endpoint
type CommentIterator struct {
	client  *Client
	idOrKey string
	message *CommentPage
	Err     error

	pageSize int
	itemIdx  int
}

// HasError returns true if the iterator is holding an error
func (ci *CommentIterator) HasError() bool {
	if ci.Err == errDone {
		return false
	}
	if ci.Err == nil {
		return false
	}
	return true
}

// HasNext returns true if there is another item available in the result set
func (ci *CommentIterator) HasNext() bool {
	return ci.Err == nil && ci.itemIdx < len(ci.message.Comments)
}

// Next Return the next item in the result set and advance the iterator.
// Advancing the iterator may require fetching a new page.
func (ci *CommentIterator) Next() *Comment {
	if ci.Err != nil {
		return nil
	}

	comment := ci.message.Comments[ci.itemIdx]
	if ci.itemIdx+1 < len(ci.message.Comments) {
		// We still have an item left in the currently cached page
		ci.itemIdx++
	} else {
		if ci.message.IsLastPage() {
			ci.Err = errDone
		} else {
			// There are still more pages to fetch, so fetch the next page and
			// cache it
			ci.message, ci.Err = ci.client.GetComments(
				ci.idOrKey, ci.pageSize, ci.message.NextStartAt())
			// NOTE(josh): we don't deal with the error now, we just cache it.
			// HasNext() will return false and the caller can check the error
			// afterward.
			ci.itemIdx = 0
		}
	}
	return &comment
}

// IterComments returns an iterator over paginated comments within an issue
func (client *Client) IterComments(idOrKey string, pageSize int) *CommentIterator {
	message, err := client.GetComments(idOrKey, pageSize, 0)

	iter := &CommentIterator{
		client:   client,
		idOrKey:  idOrKey,
		message:  message,
		Err:      err,
		pageSize: pageSize,
		itemIdx:  0,
	}

	return iter
}

// GetChangeLog fetchs one page of the changelog for an issue via the
// /issue/{IssueIdOrKey}/changelog endpoint (for JIRA cloud) or
// /issue/{IssueIdOrKey} with (fields=*none&expand=changelog)
// (for JIRA server)
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue
func (client *Client) GetChangeLog(idOrKey string, maxResults int, startAt int) (*ChangeLogPage, error) {
	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s/changelog", client.serverURL, idOrKey)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err := fmt.Errorf("Creating request %v", err)
		return nil, err
	}

	query := request.URL.Query()
	if maxResults > 0 {
		query.Add("maxResults", fmt.Sprintf("%d", maxResults))
	}
	if startAt > 0 {
		query.Add("startAt", fmt.Sprintf("%d", startAt))
	}
	request.URL.RawQuery = query.Encode()

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		// The issue/{IssueIdOrKey}/changelog endpoint is only available on JIRA cloud
		// products, not on JIRA server. In order to get the information we have to
		// query the issue and ask for a changelog expansion. Unfortunately this means
		// that the changelog is not paginated and we have to fetch the entire thing
		// at once. Hopefully things don't break for very long changelogs.
		issue, err := client.GetIssue(
			idOrKey, []string{"*none"}, []string{"changelog"}, []string{})
		if err != nil {
			return nil, err
		}

		return &issue.ChangeLog, nil
	}

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"HTTP response %d, query was %s", response.StatusCode,
			request.URL.String())
		return nil, err
	}

	var changelog ChangeLogPage

	data, _ := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &changelog)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	// JIRA cloud returns changelog entries in the "values" list, whereas JIRA
	// server returns them in the "histories" list when embedded in an issue
	// object.
	changelog.Entries = changelog.Values
	changelog.Values = nil

	return &changelog, nil
}

// ChangeLogIterator cursor within paginated results from the /search endpoint
type ChangeLogIterator struct {
	client  *Client
	idOrKey string
	message *ChangeLogPage
	Err     error

	pageSize int
	itemIdx  int
}

// HasError returns true if the iterator is holding an error
func (cli *ChangeLogIterator) HasError() bool {
	if cli.Err == errDone {
		return false
	}
	if cli.Err == nil {
		return false
	}
	return true
}

// HasNext returns true if there is another item available in the result set
func (cli *ChangeLogIterator) HasNext() bool {
	return cli.Err == nil && cli.itemIdx < len(cli.message.Entries)
}

// Next Return the next item in the result set and advance the iterator.
// Advancing the iterator may require fetching a new page.
func (cli *ChangeLogIterator) Next() *ChangeLogEntry {
	if cli.Err != nil {
		return nil
	}

	item := cli.message.Entries[cli.itemIdx]
	if cli.itemIdx+1 < len(cli.message.Entries) {
		// We still have an item left in the currently cached page
		cli.itemIdx++
	} else {
		if cli.message.IsLastPage() {
			cli.Err = errDone
		} else {
			// There are still more pages to fetch, so fetch the next page and
			// cache it
			cli.message, cli.Err = cli.client.GetChangeLog(
				cli.idOrKey, cli.pageSize, cli.message.NextStartAt())
			// NOTE(josh): we don't deal with the error now, we just cache it.
			// HasNext() will return false and the caller can check the error
			// afterward.
			cli.itemIdx = 0
		}
	}
	return &item
}

// IterChangeLog returns an iterator over entries in the changelog for an issue
func (client *Client) IterChangeLog(idOrKey string, pageSize int) *ChangeLogIterator {
	message, err := client.GetChangeLog(idOrKey, pageSize, 0)

	iter := &ChangeLogIterator{
		client:   client,
		idOrKey:  idOrKey,
		message:  message,
		Err:      err,
		pageSize: pageSize,
		itemIdx:  0,
	}

	return iter
}

// GetProject returns the project JSON object given its id or key
func (client *Client) GetProject(projectIDOrKey string) (*Project, error) {
	url := fmt.Sprintf(
		"%s/rest/api/2/project/%s", client.serverURL, projectIDOrKey)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"HTTP response %d, query was %s", response.StatusCode, url)
		return nil, err
	}

	var project Project

	data, _ := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &project)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &project, nil
}

// CreateIssue creates a new JIRA issue and returns it
func (client *Client) CreateIssue(projectIDOrKey, title, body string,
	extra map[string]interface{}) (*IssueCreateResult, error) {

	url := fmt.Sprintf("%s/rest/api/2/issue", client.serverURL)

	fields := make(map[string]interface{})
	fields["summary"] = title
	fields["description"] = body
	for key, value := range extra {
		fields[key] = value
	}

	// If the project string is an integer than assume it is an ID. Otherwise it
	// is a key.
	_, err := strconv.Atoi(projectIDOrKey)
	if err == nil {
		fields["project"] = map[string]string{"id": projectIDOrKey}
	} else {
		fields["project"] = map[string]string{"key": projectIDOrKey}
	}

	message := make(map[string]interface{})
	message["fields"] = fields

	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		content, _ := ioutil.ReadAll(response.Body)
		err := fmt.Errorf(
			"HTTP response %d, query was %s\n  data: %s\n  response: %s",
			response.StatusCode, request.URL.String(), data, content)
		return nil, err
	}

	var result IssueCreateResult

	data, _ = ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &result)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &result, nil
}

// UpdateIssueTitle changes the "summary" field of a JIRA issue
func (client *Client) UpdateIssueTitle(issueKeyOrID, title string) (time.Time, error) {

	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s", client.serverURL, issueKeyOrID)
	var responseTime time.Time

	// NOTE(josh): Since updates are a list of heterogeneous objects let's just
	// manually build the JSON text
	data, err := json.Marshal(title)
	if err != nil {
		return responseTime, err
	}

	var buffer bytes.Buffer
	_, _ = fmt.Fprintf(&buffer, `{"update":{"summary":[`)
	_, _ = fmt.Fprintf(&buffer, `{"set":%s}`, data)
	_, _ = fmt.Fprintf(&buffer, `]}}`)

	data = buffer.Bytes()
	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return responseTime, err
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return responseTime, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		content, _ := ioutil.ReadAll(response.Body)
		err := fmt.Errorf(
			"HTTP response %d, query was %s\n  data: %s\n  response: %s",
			response.StatusCode, request.URL.String(), data, content)
		return responseTime, err
	}

	dateHeader, ok := response.Header["Date"]
	if !ok || len(dateHeader) != 1 {
		// No "Date" header, or empty, or multiple of them. Regardless, we don't
		// have a date we can return
		return responseTime, nil
	}

	responseTime, err = http.ParseTime(dateHeader[0])
	if err != nil {
		return time.Time{}, err
	}

	return responseTime, nil
}

// UpdateIssueBody changes the "description" field of a JIRA issue
func (client *Client) UpdateIssueBody(issueKeyOrID, body string) (time.Time, error) {

	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s", client.serverURL, issueKeyOrID)
	var responseTime time.Time
	// NOTE(josh): Since updates are a list of heterogeneous objects let's just
	// manually build the JSON text
	data, err := json.Marshal(body)
	if err != nil {
		return responseTime, err
	}

	var buffer bytes.Buffer
	_, _ = fmt.Fprintf(&buffer, `{"update":{"description":[`)
	_, _ = fmt.Fprintf(&buffer, `{"set":%s}`, data)
	_, _ = fmt.Fprintf(&buffer, `]}}`)

	data = buffer.Bytes()
	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return responseTime, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return responseTime, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		content, _ := ioutil.ReadAll(response.Body)
		err := fmt.Errorf(
			"HTTP response %d, query was %s\n  data: %s\n  response: %s",
			response.StatusCode, request.URL.String(), data, content)
		return responseTime, err
	}

	dateHeader, ok := response.Header["Date"]
	if !ok || len(dateHeader) != 1 {
		// No "Date" header, or empty, or multiple of them. Regardless, we don't
		// have a date we can return
		return responseTime, nil
	}

	responseTime, err = http.ParseTime(dateHeader[0])
	if err != nil {
		return time.Time{}, err
	}

	return responseTime, nil
}

// AddComment adds a new comment to an issue (and returns it).
func (client *Client) AddComment(issueKeyOrID, body string) (*Comment, error) {
	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s/comment", client.serverURL, issueKeyOrID)

	params := CommentCreate{Body: body}
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		content, _ := ioutil.ReadAll(response.Body)
		err := fmt.Errorf(
			"HTTP response %d, query was %s\n  data: %s\n  response: %s",
			response.StatusCode, request.URL.String(), data, content)
		return nil, err
	}

	var result Comment

	data, _ = ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &result)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &result, nil
}

// UpdateComment changes the text of a comment
func (client *Client) UpdateComment(issueKeyOrID, commentID, body string) (
	*Comment, error) {
	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s/comment/%s", client.serverURL, issueKeyOrID,
		commentID)

	params := CommentCreate{Body: body}
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"HTTP response %d, query was %s", response.StatusCode,
			request.URL.String())
		return nil, err
	}

	var result Comment

	data, _ = ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &result)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &result, nil
}

// UpdateLabels changes labels for an issue
func (client *Client) UpdateLabels(issueKeyOrID string, added, removed []bug.Label) (time.Time, error) {
	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s/", client.serverURL, issueKeyOrID)
	var responseTime time.Time

	// NOTE(josh): Since updates are a list of heterogeneous objects let's just
	// manually build the JSON text
	var buffer bytes.Buffer
	_, _ = fmt.Fprintf(&buffer, `{"update":{"labels":[`)
	first := true
	for _, label := range added {
		if !first {
			_, _ = fmt.Fprintf(&buffer, ",")
		}
		_, _ = fmt.Fprintf(&buffer, `{"add":"%s"}`, label)
		first = false
	}
	for _, label := range removed {
		if !first {
			_, _ = fmt.Fprintf(&buffer, ",")
		}
		_, _ = fmt.Fprintf(&buffer, `{"remove":"%s"}`, label)
		first = false
	}
	_, _ = fmt.Fprintf(&buffer, "]}}")

	data := buffer.Bytes()
	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return responseTime, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return responseTime, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		content, _ := ioutil.ReadAll(response.Body)
		err := fmt.Errorf(
			"HTTP response %d, query was %s\n  data: %s\n  response: %s",
			response.StatusCode, request.URL.String(), data, content)
		return responseTime, err
	}

	dateHeader, ok := response.Header["Date"]
	if !ok || len(dateHeader) != 1 {
		// No "Date" header, or empty, or multiple of them. Regardless, we don't
		// have a date we can return
		return responseTime, nil
	}

	responseTime, err = http.ParseTime(dateHeader[0])
	if err != nil {
		return time.Time{}, err
	}

	return responseTime, nil
}

// GetTransitions returns a list of available transitions for an issue
func (client *Client) GetTransitions(issueKeyOrID string) (*TransitionList, error) {

	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s/transitions", client.serverURL, issueKeyOrID)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err := fmt.Errorf("Creating request %v", err)
		return nil, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"HTTP response %d, query was %s", response.StatusCode,
			request.URL.String())
		return nil, err
	}

	var message TransitionList

	data, _ := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &message)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &message, nil
}

func getTransitionTo(tlist *TransitionList, desiredStateNameOrID string) *Transition {
	for _, transition := range tlist.Transitions {
		if transition.To.ID == desiredStateNameOrID {
			return &transition
		} else if transition.To.Name == desiredStateNameOrID {
			return &transition
		}
	}
	return nil
}

// DoTransition changes the "status" of an issue
func (client *Client) DoTransition(issueKeyOrID string, transitionID string) (time.Time, error) {
	url := fmt.Sprintf(
		"%s/rest/api/2/issue/%s/transitions", client.serverURL, issueKeyOrID)
	var responseTime time.Time

	// TODO(josh)[767ee72]: Figure out a good way to "configure" the
	// open/close state mapping. It would be *great* if we could actually
	// *compute* the necessary transitions and prompt for missing metatdata...
	// but that is complex
	var buffer bytes.Buffer
	_, _ = fmt.Fprintf(&buffer,
		`{"transition":{"id":"%s"}, "resolution": {"name": "Done"}}`,
		transitionID)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(buffer.Bytes()))
	if err != nil {
		return responseTime, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return responseTime, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		err := errors.Wrap(errTransitionNotAllowed, fmt.Sprintf(
			"HTTP response %d, query was %s", response.StatusCode,
			request.URL.String()))
		return responseTime, err
	}

	dateHeader, ok := response.Header["Date"]
	if !ok || len(dateHeader) != 1 {
		// No "Date" header, or empty, or multiple of them. Regardless, we don't
		// have a date we can return
		return responseTime, nil
	}

	responseTime, err = http.ParseTime(dateHeader[0])
	if err != nil {
		return time.Time{}, err
	}

	return responseTime, nil
}

// GetServerInfo Fetch server information from the /serverinfo endpoint
// https://docs.atlassian.com/software/jira/docs/api/REST/8.2.6/#api/2/issue
func (client *Client) GetServerInfo() (*ServerInfo, error) {
	url := fmt.Sprintf("%s/rest/api/2/serverinfo", client.serverURL)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err := fmt.Errorf("Creating request %v", err)
		return nil, err
	}

	if client.ctx != nil {
		ctx, cancel := context.WithTimeout(client.ctx, defaultTimeout)
		defer cancel()
		request = request.WithContext(ctx)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("Performing request %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"HTTP response %d, query was %s", response.StatusCode,
			request.URL.String())
		return nil, err
	}

	var message ServerInfo

	data, _ := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(data, &message)
	if err != nil {
		err := fmt.Errorf("Decoding response %v", err)
		return nil, err
	}

	return &message, nil
}

// GetServerTime returns the current time on the server
func (client *Client) GetServerTime() (MyTime, error) {
	var result MyTime
	info, err := client.GetServerInfo()
	if err != nil {
		return result, err
	}
	return info.ServerTime, nil
}
