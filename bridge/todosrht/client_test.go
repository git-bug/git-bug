package todosrht

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackerExists(t *testing.T) {
	t.Run("Public tracker exists", func(t *testing.T) {
		mockClient := &MockClient{
			MockTrackerExists: func(ctx context.Context, name string) (bool, error) {
				return name == "existing-tracker", nil
			},
		}

		exists, err := mockClient.TrackerExists(context.Background(), "existing-tracker")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Public tracker does not exist", func(t *testing.T) {
		mockClient := &MockClient{
			MockTrackerExists: func(ctx context.Context, name string) (bool, error) {
				return name == "existing-tracker", nil
			},
		}

		exists, err := mockClient.TrackerExists(context.Background(), "nonexistent-tracker")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Error handling", func(t *testing.T) {
		mockClient := &MockClient{
			MockTrackerExists: func(ctx context.Context, name string) (bool, error) {
				return false, assert.AnError
			},
		}

		_, err := mockClient.TrackerExists(context.Background(), "test-tracker")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "assert.AnError")
	})
}

func TestCreateLabel(t *testing.T) {
	t.Run("Create label successfully", func(t *testing.T) {
		mockClient := &MockClient{
			MockCreateLabel: func(ctx context.Context, trackerID int, name, foregroundColor, backgroundColor string) (*Label, error) {
				return &Label{
					Id:              123,
					Name:            name,
					ForegroundColor: foregroundColor,
					BackgroundColor: backgroundColor,
				}, nil
			},
		}

		label, err := mockClient.CreateLabel(context.Background(), 1, "bug", "#ffffff", "#000000")
		require.NoError(t, err)
		assert.Equal(t, 123, label.Id)
		assert.Equal(t, "bug", label.Name)
		assert.Equal(t, "#ffffff", label.ForegroundColor)
		assert.Equal(t, "#000000", label.BackgroundColor)
	})

	t.Run("Create label error", func(t *testing.T) {
		mockClient := &MockClient{
			MockCreateLabel: func(ctx context.Context, trackerID int, name, foregroundColor, backgroundColor string) (*Label, error) {
				return nil, assert.AnError
			},
		}

		_, err := mockClient.CreateLabel(context.Background(), 1, "bug", "#ffffff", "#000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "assert.AnError")
	})
}

func TestAssignUser(t *testing.T) {
	t.Run("Assign user successfully", func(t *testing.T) {
		mockClient := &MockClient{
			MockAssignUser: func(ctx context.Context, trackerID, ticketID, userID int) (*Event, error) {
				return &Event{Id: 456}, nil
			},
		}

		event, err := mockClient.AssignUser(context.Background(), 1, 101, 42)
		require.NoError(t, err)
		assert.Equal(t, 456, event.Id)
	})

	t.Run("Assign user error", func(t *testing.T) {
		mockClient := &MockClient{
			MockAssignUser: func(ctx context.Context, trackerID, ticketID, userID int) (*Event, error) {
				return nil, assert.AnError
			},
		}

		_, err := mockClient.AssignUser(context.Background(), 1, 101, 42)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "assert.AnError")
	})
}

func TestDeleteLabel(t *testing.T) {
	t.Run("Delete label successfully", func(t *testing.T) {
		mockClient := &MockClient{
			MockDeleteLabel: func(ctx context.Context, labelID int) (*Label, error) {
				return &Label{Id: labelID, Name: "deleted-label"}, nil
			},
		}

		label, err := mockClient.DeleteLabel(context.Background(), 123)
		require.NoError(t, err)
		assert.Equal(t, 123, label.Id)
		assert.Equal(t, "deleted-label", label.Name)
	})

	t.Run("Delete label error", func(t *testing.T) {
		mockClient := &MockClient{
			MockDeleteLabel: func(ctx context.Context, labelID int) (*Label, error) {
				return nil, assert.AnError
			},
		}

		_, err := mockClient.DeleteLabel(context.Background(), 123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "assert.AnError")
	})
}
