package gitlab

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/util/text"
	"gitlab.com/gitlab-org/api/client-go"
)

// IssueResult is either a gitlab issue, or the error that interrupted the listing.
type IssueResult struct {
	Issue *gitlab.Issue
	Err   error
}

// Issues returns a channel with gitlab project issues, ascending order.
// An error while listing is reported on the channel and ends the iteration.
func Issues(ctx context.Context, client *gitlab.Client, pid string, since time.Time) <-chan IssueResult {
	out := make(chan IssueResult)

	go func() {
		defer close(out)

		opts := gitlab.ListProjectIssuesOptions{
			UpdatedAfter: &since,
			Scope:        gitlab.Ptr("all"),
			Sort:         gitlab.Ptr("asc"),
		}

		for {
			issues, resp, err := client.Issues.ListProjectIssues(pid, &opts, gitlab.WithContext(ctx))
			if err != nil {
				out <- IssueResult{Err: err}
				return
			}

			for _, issue := range issues {
				out <- IssueResult{Issue: issue}
			}

			if resp.CurrentPage >= resp.TotalPages {
				break
			}

			opts.Page = resp.NextPage
		}
	}()

	return out
}

// Notes returns a channel with note events
func Notes(ctx context.Context, client *gitlab.Client, issue *gitlab.Issue) <-chan Event {
	out := make(chan Event)

	go func() {
		defer close(out)

		opts := gitlab.ListIssueNotesOptions{
			OrderBy: gitlab.Ptr("created_at"),
			Sort:    gitlab.Ptr("asc"),
		}

		for {
			notes, resp, err := client.Notes.ListIssueNotes(issue.ProjectID, issue.IID, &opts, gitlab.WithContext(ctx))

			if err != nil {
				out <- ErrorEvent{Err: err, Time: time.Now()}
				return
			}

			for _, note := range notes {
				out <- NoteEvent{*note}
			}

			if resp.CurrentPage >= resp.TotalPages {
				break
			}

			opts.Page = resp.NextPage
		}
	}()

	return out
}

// LabelEvents returns a channel with label events.
func LabelEvents(ctx context.Context, client *gitlab.Client, issue *gitlab.Issue) <-chan Event {
	out := make(chan Event)

	go func() {
		defer close(out)

		opts := gitlab.ListLabelEventsOptions{}

		for {
			events, resp, err := client.ResourceLabelEvents.ListIssueLabelEvents(issue.ProjectID, issue.IID, &opts, gitlab.WithContext(ctx))

			if err != nil {
				out <- ErrorEvent{Err: err, Time: time.Now()}
				return
			}

			for _, e := range events {
				le := LabelEvent{*e}
				le.Label.Name = text.CleanupOneLine(le.Label.Name)
				out <- le
			}

			if resp.CurrentPage >= resp.TotalPages {
				break
			}

			opts.Page = resp.NextPage
		}
	}()

	return out
}

// StateEvents returns a channel with state change events.
func StateEvents(ctx context.Context, client *gitlab.Client, issue *gitlab.Issue) <-chan Event {
	out := make(chan Event)

	go func() {
		defer close(out)

		opts := gitlab.ListStateEventsOptions{}

		for {
			events, resp, err := client.ResourceStateEvents.ListIssueStateEvents(issue.ProjectID, issue.IID, &opts, gitlab.WithContext(ctx))
			if err != nil {
				out <- ErrorEvent{Err: err, Time: time.Now()}
				return
			}

			for _, e := range events {
				out <- StateEvent{*e}
			}

			if resp.CurrentPage >= resp.TotalPages {
				break
			}

			opts.Page = resp.NextPage
		}
	}()

	return out
}
