package gitlab

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/util/text"
	"github.com/xanzy/go-gitlab"
)

// Issues returns a channel with gitlab project issues, ascending order.
func Issues(ctx context.Context, client *gitlab.Client, pid string, since time.Time) <-chan *gitlab.Issue {
	out := make(chan *gitlab.Issue)

	go func() {
		defer close(out)

		opts := gitlab.ListProjectIssuesOptions{
			UpdatedAfter: &since,
			Scope:        gitlab.String("all"),
			Sort:         gitlab.String("asc"),
		}

		for {
			issues, resp, err := client.Issues.ListProjectIssues(pid, &opts, gitlab.WithContext(ctx))
			if err != nil {
				return
			}

			for _, issue := range issues {
				out <- issue
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
			OrderBy: gitlab.String("created_at"),
			Sort:    gitlab.String("asc"),
		}

		for {
			notes, resp, err := client.Notes.ListIssueNotes(issue.ProjectID, issue.IID, &opts, gitlab.WithContext(ctx))

			if err != nil {
				out <- ErrorEvent{Err: err, Time: time.Now()}
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
