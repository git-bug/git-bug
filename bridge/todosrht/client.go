package todosrht

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"strings"
	"time"

	"github.com/pkg/errors"
)

// GraphQL client wrapper for todo.sr.ht
type TodoSClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// debuggingTransport is a http.RoundTripper that prints out the request
// and response details.
type debuggingTransport struct {
	base http.RoundTripper
}

func (t *debuggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		fmt.Printf("failed to dump request: %v\n", err)
	} else {
		fmt.Printf("--- Request ---\n%s\n", string(dump))
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		// Don't dump response on error, as resp might be nil
		return nil, err
	}

	dump, err = httputil.DumpResponse(resp, true)
	if err != nil {
		fmt.Printf("failed to dump response: %v\n", err)
	} else {
		fmt.Printf("--- Response ---\n%s\n", string(dump))
	}

	return resp, nil
}

// NewTodoSClient creates a new GraphQL client for todo.sr.ht
func NewTodoSClient(ctx context.Context, baseURL, token string) *TodoSClient {
	var baseTransport http.RoundTripper = http.DefaultTransport
	if os.Getenv("GIT_BUG_DEBUG") == "1" {
		baseTransport = &debuggingTransport{base: baseTransport}
	}
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &authTransport{
			token: token,
			base:  baseTransport,
		},
	}

	return &TodoSClient{
		client:  httpClient,
		baseURL: baseURL,
		token:   token,
	}
}

// authTransport adds Authorization header to requests
type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Content-Type", "application/json")
	return t.base.RoundTrip(req)
}

// Core GraphQL types based on schema.graphqls

type Entity interface {
	GetCanonicalName() string
}

type User struct {
	Id            int    `json:"id"`
	Created       Time   `json:"created"`
	Updated       Time   `json:"updated"`
	CanonicalName string `json:"canonicalName"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Url           string `json:"url"`
	Location      string `json:"location"`
	Bio           string `json:"bio"`
}

func (u *User) GetCanonicalName() string { return u.CanonicalName }

type ExternalUser struct {
	CanonicalName string `json:"canonicalName"`
	ExternalId    string `json:"externalId"`
	ExternalUrl   string `json:"externalUrl"`
}

func (u *ExternalUser) GetCanonicalName() string { return u.CanonicalName }

type EmailAddress struct {
	CanonicalName string `json:"canonicalName"`
	Mailbox       string `json:"mailbox"`
	Name          string `json:"name"`
}

func (e *EmailAddress) GetCanonicalName() string { return e.CanonicalName }

type Tracker struct {
	Id          int              `json:"id"`
	Created     Time             `json:"created"`
	Updated     Time             `json:"updated"`
	Owner       *json.RawMessage `json:"owner"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Visibility  Visibility       `json:"visibility"`
}

type Ticket struct {
	Id           int               `json:"id"`
	Created      Time              `json:"created"`
	Updated      Time              `json:"updated"`
	Submitter    *json.RawMessage  `json:"submitter"`
	Tracker      Tracker           `json:"tracker"`
	Ref          string            `json:"ref"`
	Subject      string            `json:"subject"`
	Body         string            `json:"body"`
	Status       TicketStatus      `json:"status"`
	Resolution   TicketResolution  `json:"resolution"`
	Authenticity Authenticity      `json:"authenticity"`
	Labels       []Label           `json:"labels"`
	Assignees    []json.RawMessage `json:"assignees"`
}

type Label struct {
	Id              int     `json:"id"`
	Created         Time    `json:"created"`
	Name            string  `json:"name"`
	Tracker         Tracker `json:"tracker"`
	BackgroundColor string  `json:"backgroundColor"`
	ForegroundColor string  `json:"foregroundColor"`
}

type Event struct {
	Id      int               `json:"id"`
	Created Time              `json:"created"`
	Changes []json.RawMessage `json:"changes"`
	Ticket  Ticket            `json:"ticket"`
}

// Event detail types
type EventDetail interface {
	GetEventType() EventType
}

type Created struct {
	EventTypeVal EventType        `json:"eventType"`
	Author       *json.RawMessage `json:"author"`
}

func (c *Created) GetEventType() EventType { return c.EventTypeVal }

type Assignment struct {
	EventTypeVal EventType        `json:"eventType"`
	Assigner     *json.RawMessage `json:"assigner"`
	Assignee     *json.RawMessage `json:"assignee"`
}

func (a *Assignment) GetEventType() EventType { return a.EventTypeVal }

type Comment struct {
	EventTypeVal EventType        `json:"eventType"`
	Author       *json.RawMessage `json:"author"`
	Text         string           `json:"text"`
	Authenticity Authenticity     `json:"authenticity"`
	SupersededBy *Comment         `json:"supersededBy"`
}

func (c *Comment) GetEventType() EventType { return c.EventTypeVal }

type LabelUpdate struct {
	EventTypeVal EventType        `json:"eventType"`
	Labeler      *json.RawMessage `json:"labeler"`
	Label        Label            `json:"label"`
}

func (l *LabelUpdate) GetEventType() EventType { return l.EventTypeVal }

type StatusChange struct {
	EventTypeVal  EventType        `json:"eventType"`
	Editor        *json.RawMessage `json:"editor"`
	OldStatus     TicketStatus     `json:"oldStatus"`
	NewStatus     TicketStatus     `json:"newStatus"`
	OldResolution TicketResolution `json:"oldResolution"`
	NewResolution TicketResolution `json:"newResolution"`
}

func (s *StatusChange) GetEventType() EventType { return s.EventTypeVal }

type UserMention struct {
	EventTypeVal EventType        `json:"eventType"`
	Author       *json.RawMessage `json:"author"`
	Mentioned    *json.RawMessage `json:"mentioned"`
}

func (u *UserMention) GetEventType() EventType { return u.EventTypeVal }

type TicketMention struct {
	EventTypeVal EventType        `json:"eventType"`
	Author       *json.RawMessage `json:"author"`
	Mentioned    Ticket           `json:"mentioned"`
}

func (t *TicketMention) GetEventType() EventType { return t.EventTypeVal }

// Enums
type Visibility string

const (
	VisibilityPublic   Visibility = "PUBLIC"
	VisibilityUnlisted Visibility = "UNLISTED"
	VisibilityPrivate  Visibility = "PRIVATE"
)

type TicketStatus string

const (
	TicketStatusReported   TicketStatus = "REPORTED"
	TicketStatusConfirmed  TicketStatus = "CONFIRMED"
	TicketStatusInProgress TicketStatus = "IN_PROGRESS"
	TicketStatusPending    TicketStatus = "PENDING"
	TicketStatusResolved   TicketStatus = "RESOLVED"
)

type TicketResolution string

const (
	TicketResolutionUnresolved  TicketResolution = "UNRESOLVED"
	TicketResolutionClosed      TicketResolution = "CLOSED"
	TicketResolutionFixed       TicketResolution = "FIXED"
	TicketResolutionImplemented TicketResolution = "IMPLEMENTED"
	TicketResolutionWontFix     TicketResolution = "WONT_FIX"
	TicketResolutionByDesign    TicketResolution = "BY_DESIGN"
	TicketResolutionInvalid     TicketResolution = "INVALID"
	TicketResolutionDuplicate   TicketResolution = "DUPLICATE"
	TicketResolutionNotOurBug   TicketResolution = "NOT_OUR_BUG"
)

type Authenticity string

const (
	AuthenticityAuthentic       Authenticity = "AUTHENTIC"
	AuthenticityUnauthenticated Authenticity = "UNAUTHENTICATED"
	AuthenticityTampered        Authenticity = "TAMPERED"
)

type EventType string

const (
	EventTypeCreated         EventType = "CREATED"
	EventTypeComment         EventType = "COMMENT"
	EventTypeStatusChange    EventType = "STATUS_CHANGE"
	EventTypeLabelAdded      EventType = "LABEL_ADDED"
	EventTypeLabelRemoved    EventType = "LABEL_REMOVED"
	EventTypeAssignedUser    EventType = "ASSIGNED_USER"
	EventTypeUnassignedUser  EventType = "UNASSIGNED_USER"
	EventTypeUserMentioned   EventType = "USER_MENTIONED"
	EventTypeTicketMentioned EventType = "TICKET_MENTIONED"
)

// Cursor types for pagination
type TrackerCursor struct {
	Results []Tracker `json:"results"`
	Cursor  *string   `json:"cursor"`
}

type TicketCursor struct {
	Results []Ticket `json:"results"`
	Cursor  *string  `json:"cursor"`
}

type EventCursor struct {
	Results []Event `json:"results"`
	Cursor  *string `json:"cursor"`
}

type LabelCursor struct {
	Results []Label `json:"results"`
	Cursor  *string `json:"cursor"`
}

// Custom Time type to handle todo.sr.ht timestamp format
type Time time.Time

func (t *Time) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if str == "" {
		*t = Time(time.Time{})
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	*t = Time(parsed)
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339))
}

func (t Time) Unix() int64 {
	return time.Time(t).Unix()
}

type typeName struct {
	TypeName string `json:"__typename"`
}

// GetSubmitter unmarshals the submitter field into the appropriate Entity type
func (t *Ticket) GetSubmitter() (Entity, error) {
	return UnmarshalEntity(t.Submitter)
}

func (t *Ticket) GetAssignees() ([]Entity, error) {
	var assignees []Entity
	for _, raw := range t.Assignees {
		assignee, err := UnmarshalEntity(&raw)
		if err != nil {
			return nil, err
		}
		assignees = append(assignees, assignee)
	}
	return assignees, nil
}

func UnmarshalEntity(raw *json.RawMessage) (Entity, error) {
	if raw == nil {
		return nil, nil
	}

	var tn typeName
	if err := json.Unmarshal(*raw, &tn); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal __typename for entity")
	}

	var entity Entity
	switch tn.TypeName {
	case "User":
		entity = &User{}
	case "ExternalUser":
		entity = &ExternalUser{}
	case "EmailAddress":
		entity = &EmailAddress{}
	default:
		// Gracefully handle unknown entity types
		return nil, nil
	}

	if err := json.Unmarshal(*raw, entity); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal entity of type %s", tn.TypeName)
	}

	return entity, nil
}

func (e *Event) GetChanges() ([]EventDetail, error) {
	var changes []EventDetail
	for _, raw := range e.Changes {
		var tn typeName
		if err := json.Unmarshal(raw, &tn); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal __typename for event change")
		}

		var change EventDetail
		switch tn.TypeName {
		case "Created":
			change = &Created{}
		case "Assignment":
			change = &Assignment{}
		case "Comment":
			change = &Comment{}
		case "LabelUpdate":
			change = &LabelUpdate{}
		case "StatusChange":
			change = &StatusChange{}
		case "UserMention":
			change = &UserMention{}
		case "TicketMention":
			change = &TicketMention{}
		default:
			// Gracefully handle unknown event types
			continue
		}
		if err := json.Unmarshal(raw, change); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal change of type %s", tn.TypeName)
		}
		changes = append(changes, change)
	}
	return changes, nil
}


// GraphQL request/response structures
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

// GraphQL queries
const getTrackerQuery = `
	query GetTracker {
		me {
			trackers {
				results {
					id
					created
					updated
					name
					description
					visibility
				}
			}
		}
	}
`

const getTrackerByNameQuery = `
	query GetTrackerByName($name: String!) {
		tracker(name: $name) {
			id
			name
			description
			visibility
		}
	}
`

const getTicketsQuery = `
	query GetTickets($trackerName: String!, $cursor: Cursor) {
		me {
			tracker(name: $trackerName) {
				tickets(cursor: $cursor) {
					results {
						id
						created
						updated
						subject
						body
						status
						resolution
						ref
						submitter {
							__typename
							... on User {
								canonicalName
								username
								email
							}
							... on ExternalUser {
								canonicalName
								externalId
								externalUrl
							}
							... on EmailAddress {
								canonicalName
								mailbox
								name
							}
						}
						labels {
							id
							name
							backgroundColor
							foregroundColor
						}
						assignees {
							__typename
							... on User {
								canonicalName
								username
								email
							}
							... on ExternalUser {
								canonicalName
								externalId
								externalUrl
							}
							... on EmailAddress {
								canonicalName
								mailbox
								name
							}
						}
					}
					cursor
				}
			}
		}
	}
`

const getTicketQuery = `
	query GetTicket($id: Int!) {
		ticket(id: $id) {
			id
			created
			updated
			subject
			body
			status
			resolution
			ref
			submitter {
				__typename
				... on User {
					canonicalName
					username
					email
				}
				... on ExternalUser {
					canonicalName
					externalId
					externalUrl
				}
				... on EmailAddress {
					canonicalName
					mailbox
					name
				}
			}
			labels {
				id
				name
				backgroundColor
				foregroundColor
			}
			assignees {
				__typename
				... on User {
					canonicalName
					username
					email
				}
				... on ExternalUser {
					canonicalName
					externalId
					externalUrl
				}
				... on EmailAddress {
					canonicalName
					mailbox
					name
				}
			}
		}
	}
`

const getEventsQuery = `
	query GetEvents($trackerName: String!, $ticketId: Int!, $cursor: Cursor) {
		me {
			tracker(name: $trackerName) {
				ticket(id: $ticketId) {
					events(cursor: $cursor) {
						results {
							id
							created
							changes {
								__typename
								... on Created {
									eventType
									author {
										__typename
										... on User {
											canonicalName
											username
											email
										}
									... on ExternalUser {
											canonicalName
											externalId
											externalUrl
										}
									... on EmailAddress {
											canonicalName
											mailbox
											name
										}
								}
								}
							... on Comment {
								eventType
								text
								authenticity
								author {
									__typename
									... on User {
										canonicalName
										username
										email
									}
									... on ExternalUser {
										canonicalName
										externalId
										externalUrl
									}
									... on EmailAddress {
										canonicalName
										mailbox
										name
									}
								}
								}
							... on StatusChange {
								eventType
								oldStatus
								newStatus
								oldResolution
								newResolution
								editor {
									__typename
									... on User {
										canonicalName
										username
										email
									}
									... on ExternalUser {
										canonicalName
										externalId
										externalUrl
									}
									... on EmailAddress {
										canonicalName
										mailbox
										name
									}
								}
								}
							... on LabelUpdate {
								eventType
								label {
									id
									name
									backgroundColor
									foregroundColor
								}
								labeler {
									__typename
									... on User {
										canonicalName
										username
										email
									}
									... on ExternalUser {
										canonicalName
										externalId
										externalUrl
									}
									... on EmailAddress {
										canonicalName
										mailbox
										name
									}
								}
								}
							... on Assignment {
								eventType
								assigner {
									__typename
									... on User {
										canonicalName
										username
										email
									}
									... on ExternalUser {
										canonicalName
										externalId
										externalUrl
									}
									... on EmailAddress {
										canonicalName
										mailbox
										name
									}
								}
								assignee {
									__typename
									... on User {
										canonicalName
										username
										email
									}
									... on ExternalUser {
										canonicalName
										externalId
										externalUrl
									}
									... on EmailAddress {
										canonicalName
										mailbox
										name
									}
								}
							}
							... on UserMention {
								eventType
								author {
									__typename
									... on User {
										canonicalName
										username
										email
									}
									... on ExternalUser {
										canonicalName
										externalId
										externalUrl
									}
									... on EmailAddress {
										canonicalName
										mailbox
										name
									}
								}
								mentioned {
									__typename
									... on User {
										canonicalName
										username
										email
									}
									... on ExternalUser {
										canonicalName
										externalId
										externalUrl
									}
									... on EmailAddress {
										canonicalName
										mailbox
										name
									}
								}
							}
							... on TicketMention {
								eventType
								author {
									__typename
									... on User {
										canonicalName
										username
										email
									}
									... on ExternalUser {
										canonicalName
										externalId
										externalUrl
									}
									... on EmailAddress {
										canonicalName
										mailbox
										name
									}
								}
								mentioned {
									id
									subject
								}
							}
						}
						cursor
					}
				}
			}
		}
	}
`

const getLabelsQuery = `
	query GetLabels($trackerId: Int!, $cursor: Cursor) {
		tracker(id: $trackerId) {
			labels(cursor: $cursor) {
				results {
					id
					created
					name
					backgroundColor
					foregroundColor
				}
				cursor
			}
		}
	}
`

// GraphQL mutations
const submitTicketMutation = `
	mutation SubmitTicket($trackerId: Int!, $input: SubmitTicketInput!) {
		submitTicket(trackerId: $trackerId, input: $input) {
			id
			created
			updated
			subject
			body
			status
			resolution
			ref
		}
	}
`

const submitCommentMutation = `
	mutation SubmitComment($trackerId: Int!, $ticketId: Int!, $input: SubmitCommentInput!) {
		submitComment(trackerId: $trackerId, ticketId: $ticketId, input: $input) {
			id
			created
		}
	}
`

const updateTicketStatusMutation = `
	mutation UpdateTicketStatus($trackerId: Int!, $ticketId: Int!, $input: UpdateStatusInput!) {
		updateTicketStatus(trackerId: $trackerId, ticketId: $ticketId, input: $input) {
			id
			created
		}
	}
`

const updateTicketMutation = `
	mutation UpdateTicket($trackerId: Int!, $ticketId: Int!, $input: UpdateTicketInput!) {
		updateTicket(trackerId: $trackerId, ticketId: $ticketId, input: $input) {
			id
			created
			updated
			subject
			body
		}
	}
`

const labelTicketMutation = `
	mutation LabelTicket($trackerId: Int!, $ticketId: Int!, $labelId: Int!) {
		labelTicket(trackerId: $trackerId, ticketId: $ticketId, labelId: $labelId) {
			id
			created
		}
	}
`

const unlabelTicketMutation = `
	mutation UnlabelTicket($trackerId: Int!, $ticketId: Int!, $labelId: Int!) {
		unlabelTicket(trackerId: $trackerId, ticketId: $ticketId, labelId: $labelId) {
			id
			created
		}
	}
`

const createLabelMutation = `
	mutation CreateLabel($trackerId: Int!, $name: String!, $foregroundColor: String!, $backgroundColor: String!) {
		createLabel(trackerId: $trackerId, name: $name, foregroundColor: $foregroundColor, backgroundColor: $backgroundColor) {
			id
			name
			foregroundColor
			backgroundColor
		}
	}
`

const deleteLabelMutation = `
	mutation DeleteLabel($id: Int!) {
		deleteLabel(id: $id) {
			id
			name
		}
	}
`

const assignUserMutation = `
	mutation AssignUser($trackerId: Int!, $ticketId: Int!, $userId: Int!) {
		assignUser(trackerId: $trackerId, ticketId: $ticketId, userId: $userId) {
			id
			created
		}
	}
`

const unassignUserMutation = `
	mutation UnassignUser($trackerId: Int!, $ticketId: Int!, $userId: Int!) {
		unassignUser(trackerId: $trackerId, ticketId: $ticketId, userId: $userId) {
			id
			created
		}
	}
`

// Input types
type SubmitTicketInput struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type SubmitCommentInput struct {
	Text       string           `json:"text"`
	Status     TicketStatus     `json:"status"`
	Resolution TicketResolution `json:"resolution"`
}

type UpdateStatusInput struct {
	Status     TicketStatus     `json:"status"`
	Resolution TicketResolution `json:"resolution"`
}

type UpdateTicketInput struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// Helper method to execute GraphQL requests
func (c *TodoSClient) executeRequest(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/query", bytes.NewBuffer(jsonBody))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	var response GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return errors.Wrap(err, "failed to decode response")
	}

	if len(response.Errors) > 0 {
		var errs []string
		for _, e := range response.Errors {
			errs = append(errs, e.Message)
		}
		return fmt.Errorf("GraphQL errors: %s", strings.Join(errs, ", "))
	}

	if err := json.Unmarshal(response.Data, result); err != nil {
		return errors.Wrap(err, "failed to unmarshal result")
	}

	return nil
}

var _ TodosrhtClient = &TodoSClient{}

// Client methods

// GetTracker fetches a tracker by name
func (c *TodoSClient) GetTracker(ctx context.Context, name string) (*Tracker, error) {
	var result struct {
		Me struct {
			Trackers struct {
				Results []Tracker `json:"results"`
			}
		}
	}

	err := c.executeRequest(ctx, getTrackerQuery, nil, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch tracker")
	}

	// Extract just the tracker name if it contains owner prefix
	trackerName := name
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			trackerName = parts[1]
		}
	}

	// Find the tracker by name
	for _, tracker := range result.Me.Trackers.Results {
		if tracker.Name == trackerName {
			return &tracker, nil
		}
	}

	// Not found
	return nil, nil
}

// TrackerExists checks if a tracker exists without requiring authentication
func (c *TodoSClient) TrackerExists(ctx context.Context, name string) (bool, error) {
	// Create a client without authentication for public access
	publicClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &publicTransport{
			base: http.DefaultTransport,
		},
	}

	reqBody := GraphQLRequest{
		Query: getTrackerByNameQuery,
		Variables: map[string]interface{}{
			"name": name,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return false, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/query", bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := publicClient.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// If we get unauthorized, the tracker might exist but requires auth
		return true, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, nil // Tracker likely doesn't exist
	}

	var response GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, errors.Wrap(err, "failed to decode response")
	}

	if len(response.Errors) > 0 {
		// Check if the error indicates tracker doesn't exist
		for _, err := range response.Errors {
			if strings.Contains(err.Message, "not found") || strings.Contains(err.Message, "does not exist") {
				return false, nil
			}
		}
		return false, fmt.Errorf("GraphQL errors: %v", response.Errors)
	}

	var result struct {
		Tracker *Tracker `json:"tracker"`
	}

	if err := json.Unmarshal(response.Data, &result); err != nil {
		return false, errors.Wrap(err, "failed to unmarshal result")
	}

	return result.Tracker != nil, nil
}

// publicTransport adds only Content-Type header for public requests
type publicTransport struct {
	base http.RoundTripper
}

func (t *publicTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Content-Type", "application/json")
	return t.base.RoundTrip(req)
}

// GetTickets fetches tickets from a tracker with pagination
func (c *TodoSClient) GetTickets(ctx context.Context, trackerName string, cursor *string) ([]Ticket, *string, error) {
	var result struct {
		Me struct {
			Tracker struct {
				Tickets TicketCursor `json:"tickets"`
			}
		}
	}

	variables := map[string]interface{}{
		"trackerName": trackerName,
		"cursor":      cursor,
	}

	err := c.executeRequest(ctx, getTicketsQuery, variables, &result)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch tickets")
	}

	return result.Me.Tracker.Tickets.Results, result.Me.Tracker.Tickets.Cursor, nil
}

// GetTicket fetches a specific ticket
func (c *TodoSClient) GetTicket(ctx context.Context, id int) (*Ticket, error) {
	var result struct {
		Ticket *Ticket `json:"ticket"`
	}

	variables := map[string]interface{}{
		"id": id,
	}

	err := c.executeRequest(ctx, getTicketQuery, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch ticket")
	}

	return result.Ticket, nil
}

// GetEvents fetches events for a ticket with pagination
func (c *TodoSClient) GetEvents(ctx context.Context, trackerName string, ticketID int, cursor *string) ([]Event, *string, error) {
	var result struct {
		Me struct {
			Tracker struct {
				Ticket struct {
					Events EventCursor `json:"events"`
				}
			}
		}
	}

	variables := map[string]interface{}{
		"trackerName": trackerName,
		"ticketId":    ticketID,
		"cursor":      cursor,
	}

	err := c.executeRequest(ctx, getEventsQuery, variables, &result)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch events")
	}

	return result.Me.Tracker.Ticket.Events.Results, result.Me.Tracker.Ticket.Events.Cursor, nil
}

// GetLabels fetches labels for a tracker with pagination
func (c *TodoSClient) GetLabels(ctx context.Context, trackerID int, cursor *string) (*LabelCursor, error) {
	var result struct {
		Tracker struct {
			Labels LabelCursor `json:"labels"`
		}
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"cursor":    cursor,
	}

	err := c.executeRequest(ctx, getLabelsQuery, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch labels")
	}

	return &result.Tracker.Labels, nil
}

// CreateTicket creates a new ticket
func (c *TodoSClient) CreateTicket(ctx context.Context, trackerID int, input SubmitTicketInput) (*Ticket, error) {
	var result struct {
		SubmitTicket Ticket `json:"submitTicket"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"input":     input,
	}

	err := c.executeRequest(ctx, submitTicketMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ticket")
	}

	return &result.SubmitTicket, nil
}

// CreateComment creates a new comment on a ticket
func (c *TodoSClient) CreateComment(ctx context.Context, trackerID, ticketID int, input SubmitCommentInput) (*Event, error) {
	var result struct {
		SubmitComment Event `json:"submitComment"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"ticketId":  ticketID,
		"input":     input,
	}

	err := c.executeRequest(ctx, submitCommentMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create comment")
	}

	return &result.SubmitComment, nil
}

// UpdateTicketStatus updates the status and resolution of a ticket
func (c *TodoSClient) UpdateTicketStatus(ctx context.Context, trackerID, ticketID int, input UpdateStatusInput) (*Event, error) {
	var result struct {
		UpdateTicketStatus Event `json:"updateTicketStatus"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"ticketId":  ticketID,
		"input":     input,
	}

	err := c.executeRequest(ctx, updateTicketStatusMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update ticket status")
	}

	return &result.UpdateTicketStatus, nil
}

// UpdateTicket updates the subject or body of a ticket
func (c *TodoSClient) UpdateTicket(ctx context.Context, trackerID, ticketID int, input UpdateTicketInput) (*Ticket, error) {
	var result struct {
		UpdateTicket Ticket `json:"updateTicket"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"ticketId":  ticketID,
		"input":     input,
	}

	err := c.executeRequest(ctx, updateTicketMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update ticket")
	}

	return &result.UpdateTicket, nil
}

// AddLabel adds a label to a ticket
func (c *TodoSClient) AddLabel(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error) {
	var result struct {
		LabelTicket Event `json:"labelTicket"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"ticketId":  ticketID,
		"labelId":   labelID,
	}

	err := c.executeRequest(ctx, labelTicketMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add label")
	}

	return &result.LabelTicket, nil
}

// RemoveLabel removes a label from a ticket
func (c *TodoSClient) RemoveLabel(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error) {
	var result struct {
		UnlabelTicket Event `json:"unlabelTicket"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"ticketId":  ticketID,
		"labelId":   labelID,
	}

	err := c.executeRequest(ctx, unlabelTicketMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to remove label")
	}

	return &result.UnlabelTicket, nil
}

// CreateLabel creates a new label in a tracker
func (c *TodoSClient) CreateLabel(ctx context.Context, trackerID int, name, foregroundColor, backgroundColor string) (*Label, error) {
	var result struct {
		CreateLabel Label `json:"createLabel"`
	}

	variables := map[string]interface{}{
		"trackerId":       trackerID,
		"name":            name,
		"foregroundColor": foregroundColor,
		"backgroundColor": backgroundColor,
	}

	err := c.executeRequest(ctx, createLabelMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create label")
	}

	return &result.CreateLabel, nil
}

// DeleteLabel deletes a label from a tracker
func (c *TodoSClient) DeleteLabel(ctx context.Context, labelID int) (*Label, error) {
	var result struct {
		DeleteLabel Label `json:"deleteLabel"`
	}

	variables := map[string]interface{}{
		"id": labelID,
	}

	err := c.executeRequest(ctx, deleteLabelMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to delete label")
	}

	return &result.DeleteLabel, nil
}

// AssignUser assigns a user to a ticket
func (c *TodoSClient) AssignUser(ctx context.Context, trackerID, ticketID, userID int) (*Event, error) {
	var result struct {
		AssignUser Event `json:"assignUser"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"ticketId":  ticketID,
		"userId":    userID,
	}

	err := c.executeRequest(ctx, assignUserMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign user")
	}

	return &result.AssignUser, nil
}

// UnassignUser unassigns a user from a ticket
func (c *TodoSClient) UnassignUser(ctx context.Context, trackerID, ticketID, userID int) (*Event, error) {
	var result struct {
		UnassignUser Event `json:"unassignUser"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"ticketId":  ticketID,
		"userId":    userID,
	}

	err := c.executeRequest(ctx, unassignUserMutation, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unassign user")
	}

	return &result.UnassignUser, nil
}