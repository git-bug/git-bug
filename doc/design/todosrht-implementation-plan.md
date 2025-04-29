# todo.sr.ht Bridge Implementation Plan

## Overview

This document provides a comprehensive implementation plan for replacing the existing JIRA bridge in `bridge/todosrht/` with a proper todo.sr.ht bridge using GraphQL API.

## Current State Analysis

The existing `bridge/todosrht/` directory contains **JIRA bridge code**, not a todo.sr.ht bridge. This requires complete replacement.

### Key Issues with Current Implementation

1. **Wrong API**: Uses JIRA REST API (`/rest/api/2/`) instead of todo.sr.ht GraphQL
2. **Wrong Authentication**: JIRA session/basic auth instead of Bearer tokens
3. **Wrong Data Model**: JIRA issues/projects instead of todo.sr.ht tickets/trackers
4. **Package Declaration Issues**: Test files declare `package github` instead of `package todosrht`

## Implementation Roadmap

### Phase 1: Foundation (High Priority)

#### 1.1 Replace client.go
**Status**: ❌ Not Started
**Tasks**:
- [ ] Remove all JIRA REST API structures and methods
- [ ] Implement GraphQL client with Bearer token authentication
- [ ] Add todo.sr.ht specific types (Tracker, Ticket, Event, Entity, etc.)
- [ ] Implement GraphQL query/mutation execution
- [ ] Add proper error handling for GraphQL responses
- [ ] Fix missing imports (`bytes` package)

**Key Code Changes**:
```go
// Authentication
type authTransport struct {
    token string
    base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    req.Header.Set("Authorization", "Bearer "+t.token)
    req.Header.Set("Content-Type", "application/json")
    return t.base.RoundTrip(req)
}

// Core Types
type TodoSClient struct {
    client  *http.Client
    baseURL string
    token   string
}

type Tracker struct {
    Id          int        `json:"id"`
    Created     Time       `json:"created"`
    Updated     Time       `json:"updated"`
    Owner       Entity     `json:"owner"`
    Name        string     `json:"name"`
    Description  string     `json:"description"`
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
```

#### 1.2 Update config.go
**Status**: ❌ Not Started
**Tasks**:
- [ ] Change from JIRA project to todo.sr.ht tracker configuration
- [ ] Update authentication to use personal access tokens only
- [ ] Modify validation for todo.sr.ht specifics
- [ ] Remove JIRA-specific configuration options

**Configuration Changes**:
```go
// New config keys
confKeyTrackerName    = "tracker"      // todo.sr.ht tracker name
confKeyBaseUrl        = "base-url"    // https://todo.sr.ht
confKeyToken         = "token"        // personal access token
confKeyDefaultLogin  = "default-login" // username for identity matching

// Remove JIRA-specific keys
- confKeyProject (replaced with tracker)
- confKeyCredentialType (not needed for bearer tokens)
- confKeyIDMap, confKeyIDRevMap (todo.sr.ht has fixed enums)
```

#### 1.3 Fix todosrht.go
**Status**: ❌ Not Started
**Tasks**:
- [ ] Update bridge registration for todo.sr.ht
- [ ] Replace JIRA client builder with GraphQL client
- [ ] Update meta keys and constants for todo.sr.ht
- [ ] Fix Client interface references

**Meta Key Updates**:
```go
const (
    metaKeyTodoSRHTId         = "todosrht-id"
    metaKeyTodoSRHTTracker     = "todosrht-tracker"
    metaKeyTodoSRHTRef        = "todosrht-ref"
    metaKeyTodoSRHTUser       = "todosrht-user"
    metaKeyTodoSRHTBaseUrl    = "todosrht-base-url"
    metaKeyTodoSRHTExportTime = "todosrht-export-time"
    metaKeyTodoSRHTLogin      = "todosrht-login"
)
```

### Phase 2: Import/Export Logic (Medium Priority)

#### 2.1 Rewrite import.go
**Status**: ❌ Not Started
**Tasks**:
- [ ] Replace JIRA search with GraphQL ticket queries
- [ ] Map todo.sr.ht events to git-bug operations
- [ ] Handle pagination via GraphQL cursors (not JIRA page numbers)
- [ ] Map todo.sr.ht status/resolution enums to git-bug status
- [ ] Process different event types (Created, Comment, StatusChange, LabelUpdate)
- [ ] Handle Entity interface (User vs ExternalUser)

**Event Mapping Strategy**:
```go
// Status mapping
func mapTodoSRHTStatusToGitBug(status TicketStatus) common.Status {
    switch status {
    case TicketStatusReported, TicketStatusConfirmed,
         TicketStatusInProgress, TicketStatusPending:
        return common.OpenStatus
    case TicketStatusResolved:
        return common.ClosedStatus
    }
}

// Event mapping
func mapEventToOperation(event Event) (dag.Operation, error) {
    for _, change := range event.Changes {
        switch change := change.(type) {
        case Created:
            return mapCreatedEvent(change)
        case Comment:
            return mapCommentEvent(change)
        case StatusChange:
            return mapStatusChangeEvent(change)
        case LabelUpdate:
            return mapLabelUpdateEvent(change)
        }
    }
}
```

#### 2.2 Rewrite export.go
**Status**: ❌ Not Started
**Tasks**:
- [ ] Replace JIRA mutations with GraphQL mutations
- [ ] Map git-bug operations to todo.sr.ht events
- [ ] Handle label synchronization via GraphQL mutations
- [ ] Error handling for GraphQL constraints
- [ ] Handle todo.sr.ht resolution requirements

**Mutation Strategy**:
```go
// Export operations
func (exporter *todosrhtExporter) exportOperation(op dag.Operation) error {
    switch op := op.(type) {
    case *bug.CreateOperation:
        return exporter.createTicket(op)
    case *bug.AddCommentOperation:
        return exporter.createComment(op)
    case *bug.SetStatusOperation:
        return exporter.updateTicketStatus(op)
    case *bug.SetTitleOperation:
        return exporter.updateTicket(op)
    case *bug.LabelChangeOperation:
        return exporter.updateLabels(op)
    }
}
```

### Phase 3: Integration (Low Priority)

#### 3.1 Fix Test Files
**Status**: ❌ Not Started
**Tasks**:
- [ ] Update package declarations from `package github` to `package todosrht`
- [ ] Update test cases for GraphQL API responses
- [ ] Add integration tests for todo.sr.ht
- [ ] Mock GraphQL client for testing

**Files to Fix**:
- `config_test.go` - package declaration
- `export_test.go` - package declaration
- `import_test.go` - package declaration
- `import_integration_test.go` - package declaration

#### 3.2 Documentation Updates
**Status**: ❌ Not Started
**Tasks**:
- [ ] Update bridge documentation for todo.sr.ht
- [ ] Add configuration guide for todo.sr.ht specifics
- [ ] Document authentication setup (personal access tokens)
- [ ] Update README with todo.sr.ht bridge info

## Technical Specifications

### Authentication
```go
// Bearer token authentication
Authorization: Bearer <personal-access-token>

// Required scopes: TICKETS, TRACKERS, EVENTS, PROFILE
// Token URL: https://meta.sr.ht/oauth2
```

### API Endpoints
```go
// GraphQL endpoint
baseURL := "https://todo.sr.ht"

// Full query URL
queryURL := baseURL + "/query"
```

### Data Mapping

#### Status Mapping
| todo.sr.ht | git-bug |
|-------------|----------|
| REPORTED | Open |
| CONFIRMED | Open |
| IN_PROGRESS | Open |
| PENDING | Open |
| RESOLVED | Closed |

#### Event Type Mapping
| todo.sr.ht | git-bug Operation |
|-------------|-------------------|
| CREATED | CreateOperation |
| COMMENT | AddCommentOperation |
| STATUS_CHANGE | SetStatusOperation |
| LABEL_ADDED | LabelChangeOperation (add) |
| LABEL_REMOVED | LabelChangeOperation (remove) |

#### GraphQL Query Examples
```graphql
# Fetch tracker tickets
query GetTrackerTickets($trackerId: Int!, $cursor: Cursor) {
  tracker(id: $trackerId) {
    tickets(cursor: $cursor) {
      results {
        id, subject, body, status, resolution,
        created, updated, ref,
        submitter { canonicalName, username, email },
        labels { id, name, backgroundColor, foregroundColor },
        assignees { canonicalName, username, email }
      }
      cursor
    }
  }
}

# Fetch ticket events
query GetTicketEvents($ticketId: Int!, $cursor: Cursor) {
  ticket(id: $ticketId) {
    events(cursor: $cursor) {
      results {
        id, created,
        changes {
          __typename
          ... on Created { eventType, author { canonicalName } }
          ... on Comment { eventType, text, author { canonicalName } }
          ... on StatusChange {
            eventType, oldStatus, newStatus,
            oldResolution, newResolution,
            editor { canonicalName }
          }
          ... on LabelUpdate {
            eventType, label { id, name },
            labeler { canonicalName }
          }
        }
      }
      cursor
    }
  }
}
```

### GraphQL Mutations
```graphql
# Create ticket
mutation SubmitTicket($trackerId: Int!, $input: SubmitTicketInput!) {
  submitTicket(trackerId: $trackerId, input: $input) {
    id, created, subject, body, status, resolution, ref
  }
}

# Add comment
mutation SubmitComment($trackerId: Int!, $ticketId: Int!, $input: SubmitCommentInput!) {
  submitComment(trackerId: $trackerId, ticketId: $ticketId, input: $input) {
    id, created
  }
}

# Update status
mutation UpdateTicketStatus($trackerId: Int!, $ticketId: Int!, $input: UpdateStatusInput!) {
  updateTicketStatus(trackerId: $trackerId, ticketId: $ticketId, input: $input) {
    id, created
  }
}

# Add label
mutation LabelTicket($trackerId: Int!, $ticketId: Int!, $labelId: Int!) {
  labelTicket(trackerId: $trackerId, ticketId: $ticketId, labelId: $labelId) {
    id, created
  }
}
```

## Implementation Priority

### Immediate (Critical)
1. **Fix package declarations** - Update all test files from `package github` to `package todosrht`
2. **Complete GraphQL client** - Fix imports and add missing `bytes` package in client.go
3. **Update bridge registration** - Make todosrht.go compatible with new client

### High Priority
4. **Replace import.go** - Core functionality for importing tickets/events
5. **Replace export.go** - Core functionality for exporting operations
6. **Update config.go** - Configuration for todo.sr.ht specifics

### Medium Priority
7. **Comprehensive testing** - Unit and integration tests
8. **Error handling** - Robust GraphQL error handling
9. **Performance optimization** - Efficient pagination and caching

### Low Priority
10. **Documentation** - User guides and API documentation
11. **Advanced features** - Webhooks, bulk operations
12. **Monitoring** - Metrics and logging

## Success Criteria

- [ ] All package declarations corrected
- [ ] GraphQL client successfully connects to todo.sr.ht
- [ ] Can fetch trackers and tickets via GraphQL
- [ ] Can import tickets with all event types
- [ ] Can export all git-bug operation types
- [ ] Proper error handling for GraphQL failures
- [ ] All tests pass
- [ ] Documentation is complete

## Notes

1. **GraphQL Complexity**: Be mindful of query complexity limits (200 default)
2. **Pagination**: Use cursor-based pagination, not offset/limit
3. **Entity Types**: Handle both User and ExternalUser interfaces
4. **Error Handling**: GraphQL returns structured error arrays
5. **Authentication**: Personal access tokens are preferred over OAuth 2.0 for scripts

This implementation plan provides a complete roadmap for transforming the existing JIRA bridge into a proper todo.sr.ht bridge with full GraphQL API integration.
