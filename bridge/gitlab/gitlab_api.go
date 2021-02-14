package gitlab

import (
	"context"
	"sync"
	"time"

	"github.com/MichaelMure/git-bug/util/text"
	"github.com/xanzy/go-gitlab"
)

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
			issues, resp, err := client.Issues.ListProjectIssues(pid, &opts)
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

func IssueEvents(ctx context.Context, client *gitlab.Client, issue *gitlab.Issue) <-chan Event {
	cs := []<-chan Event{
		Notes(ctx, client, issue),
		LabelEvents(ctx, client, issue),
		StateEvents(ctx, client, issue),
	}

	var wg sync.WaitGroup
	out := make(chan Event)

	output := func(c <-chan Event) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func Notes(ctx context.Context, client *gitlab.Client, issue *gitlab.Issue) <-chan Event {

	out := make(chan Event)

	go func() {
		defer close(out)

		opts := gitlab.ListIssueNotesOptions{
			OrderBy: gitlab.String("created_at"),
			Sort:    gitlab.String("asc"),
		}

		for {
			notes, resp, err := client.Notes.ListIssueNotes(issue.ProjectID, issue.IID, &opts)

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

func LabelEvents(ctx context.Context, client *gitlab.Client, issue *gitlab.Issue) <-chan Event {

	out := make(chan Event)

	go func() {
		defer close(out)

		opts := gitlab.ListLabelEventsOptions{}

		for {
			events, resp, err := client.ResourceLabelEvents.ListIssueLabelEvents(issue.ProjectID, issue.IID, &opts)

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

func StateEvents(ctx context.Context, client *gitlab.Client, issue *gitlab.Issue) <-chan Event {

	out := make(chan Event)

	go func() {
		defer close(out)

		opts := gitlab.ListStateEventsOptions{}

		for {
			events, resp, err := client.ResourceStateEvents.ListIssueStateEvents(issue.ProjectID, issue.IID, &opts)
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
