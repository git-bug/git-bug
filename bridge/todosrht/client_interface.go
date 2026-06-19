package todosrht

import (
	"context"
)

// TodosrhtClient defines the interface for the SourceHut client.
type TodosrhtClient interface {
	GetTracker(ctx context.Context, name string) (*Tracker, error)
	TrackerExists(ctx context.Context, name string) (bool, error)
	GetTickets(ctx context.Context, trackerName string, cursor *string) ([]Ticket, *string, error)
	GetEvents(ctx context.Context, trackerName string, ticketID int, cursor *string) ([]Event, *string, error)
	CreateTicket(ctx context.Context, trackerID int, input SubmitTicketInput) (*Ticket, error)
	CreateComment(ctx context.Context, trackerID, ticketID int, input SubmitCommentInput) (*Event, error)
	UpdateTicket(ctx context.Context, trackerID, ticketID int, input UpdateTicketInput) (*Ticket, error)
	UpdateTicketStatus(ctx context.Context, trackerID, ticketID int, input UpdateStatusInput) (*Event, error)
	AddLabel(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error)
	RemoveLabel(ctx context.Context, trackerID, ticketID, labelID int) (*Event, error)
	GetLabels(ctx context.Context, trackerID int, cursor *string) (*LabelCursor, error)
	CreateLabel(ctx context.Context, trackerID int, name, foregroundColor, backgroundColor string) (*Label, error)
	DeleteLabel(ctx context.Context, labelID int) (*Label, error)
	AssignUser(ctx context.Context, trackerID, ticketID, userID int) (*Event, error)
	UnassignUser(ctx context.Context, trackerID, ticketID, userID int) (*Event, error)
}
