package todosrht

import (
	"context"
	"fmt"
)

// MockClient is a mock implementation of the TodosrhtClient for testing purposes.
type MockClient struct {
	MockGetTracker         func(ctx context.Context, name string) (*Tracker, error)
	MockTrackerExists      func(ctx context.Context, name string) (bool, error)
	MockGetTickets         func(ctx context.Context, trackerName string, cursor *string) ([]Ticket, *string, error)
	MockGetEvents          func(ctx context.Context, ticketID int, cursor *string) ([]Event, *string, error)
	MockCreateTicket       func(ctx context.Context, trackerID int, input SubmitTicketInput) (*Ticket, error)
	MockCreateComment      func(ctx context.Context, trackerID, ticketID int, input SubmitCommentInput) (*Event, error)
	MockUpdateTicket       func(ctx context.Context, trackerID, ticketID int, input UpdateTicketInput) (*Ticket, error)
	MockUpdateTicketStatus func(ctx context.Context, trackerID, ticketID int, input UpdateStatusInput) (*Event, error)
	MockAddLabel           func(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error)
	MockRemoveLabel        func(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error)
	MockGetLabels          func(ctx context.Context, trackerID int, cursor *string) (*LabelCursor, error)
	MockCreateLabel        func(ctx context.Context, trackerID int, name, foregroundColor, backgroundColor string) (*Label, error)
	MockDeleteLabel        func(ctx context.Context, labelID int) (*Label, error)
	MockAssignUser         func(ctx context.Context, trackerID, ticketID, userID int) (*Event, error)
	MockUnassignUser       func(ctx context.Context, trackerID, ticketID, userID int) (*Event, error)
}

// Ensure MockClient implements the TodosrhtClient interface
var _ TodosrhtClient = &MockClient{}

func (m *MockClient) GetTracker(ctx context.Context, name string) (*Tracker, error) {
	if m.MockGetTracker != nil {
		return m.MockGetTracker(ctx, name)
	}
	return nil, fmt.Errorf("GetTracker not implemented")
}

func (m *MockClient) TrackerExists(ctx context.Context, name string) (bool, error) {
	if m.MockTrackerExists != nil {
		return m.MockTrackerExists(ctx, name)
	}
	return false, fmt.Errorf("TrackerExists not implemented")
}
func (m *MockClient) GetTickets(ctx context.Context, trackerName string, cursor *string) ([]Ticket, *string, error) {
	if m.MockGetTickets != nil {
		return m.MockGetTickets(ctx, trackerName, cursor)
	}
	return nil, nil, fmt.Errorf("GetTickets not implemented")
}
func (m *MockClient) GetEvents(ctx context.Context, ticketID int, cursor *string) ([]Event, *string, error) {
	if m.MockGetEvents != nil {
		return m.MockGetEvents(ctx, ticketID, cursor)
	}
	return nil, nil, fmt.Errorf("GetEvents not implemented")
}
func (m *MockClient) CreateTicket(ctx context.Context, trackerID int, input SubmitTicketInput) (*Ticket, error) {
	if m.MockCreateTicket != nil {
		return m.MockCreateTicket(ctx, trackerID, input)
	}
	return nil, fmt.Errorf("CreateTicket not implemented")
}
func (m *MockClient) CreateComment(ctx context.Context, trackerID, ticketID int, input SubmitCommentInput) (*Event, error) {
	if m.MockCreateComment != nil {
		return m.MockCreateComment(ctx, trackerID, ticketID, input)
	}
	return nil, fmt.Errorf("CreateComment not implemented")
}
func (m *MockClient) UpdateTicket(ctx context.Context, trackerID, ticketID int, input UpdateTicketInput) (*Ticket, error) {
	if m.MockUpdateTicket != nil {
		return m.MockUpdateTicket(ctx, trackerID, ticketID, input)
	}
	return nil, fmt.Errorf("UpdateTicket not implemented")
}
func (m *MockClient) UpdateTicketStatus(ctx context.Context, trackerID, ticketID int, input UpdateStatusInput) (*Event, error) {
	if m.MockUpdateTicketStatus != nil {
		return m.MockUpdateTicketStatus(ctx, trackerID, ticketID, input)
	}
	return nil, fmt.Errorf("UpdateTicketStatus not implemented")
}
func (m *MockClient) AddLabel(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error) {
	if m.MockAddLabel != nil {
		return m.MockAddLabel(ctx, trackerID, ticketID, labelID)
	}
	return nil, fmt.Errorf("AddLabel not implemented")
}
func (m *MockClient) RemoveLabel(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error) {
	if m.MockRemoveLabel != nil {
		return m.MockRemoveLabel(ctx, trackerID, ticketID, labelID)
	}
	return nil, fmt.Errorf("RemoveLabel not implemented")
}
func (m *MockClient) GetLabels(ctx context.Context, trackerID int, cursor *string) (*LabelCursor, error) {
	if m.MockGetLabels != nil {
		return m.MockGetLabels(ctx, trackerID, cursor)
	}
	return nil, fmt.Errorf("GetLabels not implemented")
}

func (m *MockClient) CreateLabel(ctx context.Context, trackerID int, name, foregroundColor, backgroundColor string) (*Label, error) {
	if m.MockCreateLabel != nil {
		return m.MockCreateLabel(ctx, trackerID, name, foregroundColor, backgroundColor)
	}
	return nil, fmt.Errorf("CreateLabel not implemented")
}

func (m *MockClient) DeleteLabel(ctx context.Context, labelID int) (*Label, error) {
	if m.MockDeleteLabel != nil {
		return m.MockDeleteLabel(ctx, labelID)
	}
	return nil, fmt.Errorf("DeleteLabel not implemented")
}

func (m *MockClient) AssignUser(ctx context.Context, trackerID, ticketID, userID int) (*Event, error) {
	if m.MockAssignUser != nil {
		return m.MockAssignUser(ctx, trackerID, ticketID, userID)
	}
	return nil, fmt.Errorf("AssignUser not implemented")
}

func (m *MockClient) UnassignUser(ctx context.Context, trackerID, ticketID, userID int) (*Event, error) {
	if m.MockUnassignUser != nil {
		return m.MockUnassignUser(ctx, trackerID, ticketID, userID)
	}
	return nil, fmt.Errorf("UnassignUser not implemented")
}
