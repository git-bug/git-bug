package todosrht

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// GraphQL client wrapper for todo.sr.ht
type TodoSClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewTodoSClient creates a new GraphQL client for todo.sr.ht
func NewTodoSClient(ctx context.Context, baseURL, token string) *TodoSClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &authTransport{
			token: token,
			base:  http.DefaultTransport,
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

func (u User) GetCanonicalName() string { return u.CanonicalName }

type ExternalUser struct {
	CanonicalName string `json:"canonicalName"`
	ExternalId    string `json:"externalId"`
	ExternalUrl   string `json:"externalUrl"`
}

func (u ExternalUser) GetCanonicalName() string { return u.CanonicalName }

type Tracker struct {
	Id          int        `json:"id"`
	Created     Time       `json:"created"`
	Updated     Time       `json:"updated"`
	Owner       Entity     `json:"owner"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Visibility  Visibility `json:"visibility"`
}

type Ticket struct {
	Id           int              `json:"id"`
	Created      Time             `json:"created"`
	Updated      Time             `json:"updated"`
	Submitter    Entity           `json:"submitter"`
	Tracker      Tracker          `json:"tracker"`
	Ref          string           `json:"ref"`
	Subject      string           `json:"subject"`
	Body         string           `json:"body"`
	Status       TicketStatus     `json:"status"`
	Resolution   TicketResolution `json:"resolution"`
	Authenticity Authenticity     `json:"authenticity"`
	Labels       []Label          `json:"labels"`
	Assignees    []Entity         `json:"assignees"`
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
	Id      int           `json:"id"`
	Created Time          `json:"created"`
	Changes []EventDetail `json:"changes"`
	Ticket  Ticket        `json:"ticket"`
}

// Event detail types
type EventDetail interface {
	GetEventType() EventType
	GetTicket() Ticket
}

type Created struct {
	EventTypeVal EventType `json:"eventType"`
	TicketVal    Ticket    `json:"ticket"`
	Author       Entity    `json:"author"`
}

func (c Created) GetEventType() EventType { return c.EventTypeVal }
func (c Created) GetTicket() Ticket       { return c.TicketVal }

type Assignment struct {
	EventTypeVal EventType `json:"eventType"`
	TicketVal    Ticket    `json:"ticket"`
	Assigner     Entity    `json:"assigner"`
	Assignee     Entity    `json:"assignee"`
}

func (a Assignment) GetEventType() EventType { return a.EventTypeVal }
func (a Assignment) GetTicket() Ticket       { return a.TicketVal }

type Comment struct {
	EventTypeVal EventType    `json:"eventType"`
	TicketVal    Ticket       `json:"ticket"`
	Author       Entity       `json:"author"`
	Text         string       `json:"text"`
	Authenticity Authenticity `json:"authenticity"`
	SupersededBy *Comment     `json:"supersededBy"`
}

func (c Comment) GetEventType() EventType { return c.EventTypeVal }
func (c Comment) GetTicket() Ticket       { return c.TicketVal }

type LabelUpdate struct {
	EventTypeVal EventType `json:"eventType"`
	TicketVal    Ticket    `json:"ticket"`
	Labeler      Entity    `json:"labeler"`
	Label        Label     `json:"label"`
}

func (l LabelUpdate) GetEventType() EventType { return l.EventTypeVal }
func (l LabelUpdate) GetTicket() Ticket       { return l.TicketVal }

type StatusChange struct {
	EventTypeVal  EventType        `json:"eventType"`
	TicketVal     Ticket           `json:"ticket"`
	Editor        Entity           `json:"editor"`
	OldStatus     TicketStatus     `json:"oldStatus"`
	NewStatus     TicketStatus     `json:"newStatus"`
	OldResolution TicketResolution `json:"oldResolution"`
	NewResolution TicketResolution `json:"newResolution"`
}

func (s StatusChange) GetEventType() EventType { return s.EventTypeVal }
func (s StatusChange) GetTicket() Ticket       { return s.TicketVal }

type UserMention struct {
	EventTypeVal EventType `json:"eventType"`
	TicketVal    Ticket    `json:"ticket"`
	Author       Entity    `json:"author"`
	Mentioned    Entity    `json:"mentioned"`
}

func (u UserMention) GetEventType() EventType { return u.EventTypeVal }
func (u UserMention) GetTicket() Ticket       { return u.TicketVal }

type TicketMention struct {
	EventTypeVal EventType `json:"eventType"`
	TicketVal    Ticket    `json:"ticket"`
	Author       Entity    `json:"author"`
	Mentioned    Ticket    `json:"mentioned"`
}

func (t TicketMention) GetEventType() EventType { return t.EventTypeVal }
func (t TicketMention) GetTicket() Ticket       { return t.TicketVal }

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
	query GetTracker($name: String!) {
		me {
			tracker(name: $name) {
				id
				created
				updated
				name
				description
				visibility
			}
		}
	}
`

const getTicketsQuery = `
	query GetTickets($trackerId: Int!, $cursor: Cursor) {
		tracker(id: $trackerId) {
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
					}
					labels {
						id
						name
						backgroundColor
						foregroundColor
					}
					assignees {
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
					}
				}
				cursor
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
			}
			labels {
				id
				name
				backgroundColor
				foregroundColor
			}
			assignees {
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
			}
		}
	}
`

const getEventsQuery = `
	query GetEvents($ticketId: Int!, $cursor: Cursor) {
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
							}
						}
						... on Comment {
							eventType
							text
							authenticity
							author {
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
							}
						}
						... on StatusChange {
							eventType
							oldStatus
							newStatus
							oldResolution
							newResolution
							editor {
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
							}
						}
					}
				}
				cursor
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
		updateTicket(trackerId: Int!, ticketId: Int!, input: $input) {
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
		return fmt.Errorf("GraphQL errors: %v", response.Errors)
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
			Tracker *Tracker `json:"tracker"`
		} `json:"me"`
	}

	variables := map[string]interface{}{
		"name": name,
	}

	err := c.executeRequest(ctx, getTrackerQuery, variables, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch tracker")
	}

	return result.Me.Tracker, nil
}

// GetTickets fetches tickets from a tracker with pagination
func (c *TodoSClient) GetTickets(ctx context.Context, trackerID int, cursor *string) ([]Ticket, *string, error) {
	var result struct {
		Tracker struct {
			Tickets TicketCursor `json:"tickets"`
		} `json:"tracker"`
	}

	variables := map[string]interface{}{
		"trackerId": trackerID,
		"cursor":    cursor,
	}

	err := c.executeRequest(ctx, getTicketsQuery, variables, &result)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch tickets")
	}

	return result.Tracker.Tickets.Results, result.Tracker.Tickets.Cursor, nil
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
func (c *TodoSClient) GetEvents(ctx context.Context, ticketID int, cursor *string) ([]Event, *string, error) {
	var result struct {
		Ticket struct {
			Events EventCursor `json:"events"`
		} `json:"ticket"`
	}

	variables := map[string]interface{}{
		"ticketId": ticketID,
		"cursor":   cursor,
	}

	err := c.executeRequest(ctx, getEventsQuery, variables, &result)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch events")
	}

	return result.Ticket.Events.Results, result.Ticket.Events.Cursor, nil
}

// GetLabels fetches labels for a tracker with pagination
func (c *TodoSClient) GetLabels(ctx context.Context, trackerID int, cursor *string) (*LabelCursor, error) {
	var result struct {
		Tracker struct {
			Labels LabelCursor `json:"labels"`
		} `json:"tracker"`
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
