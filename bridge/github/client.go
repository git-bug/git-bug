package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/shurcooL/githubv4"
)

// client wrapps the Github GraphQL client from github.com/shurcool/githubv4
// and adds revised error handling and handling of Github's GraphQL rate limit.
type client struct {
	// the client from  github.com/shurcool/githubv4
	sc *githubv4.Client
}

func newClient(httpClient *http.Client) *client {
	return &client{sc: githubv4.NewClient(httpClient)}
}

type RateLimitingEvent struct {
	msg string
}

// mutate calls the github api with a graphql mutation and it emits for each rate limiting event an export result.
func (c *client) mutate(ctx context.Context, m interface{}, input githubv4.Input, vars map[string]interface{}, out chan<- core.ExportResult) error {
	// prepare a closure for the mutation
	mutFun := func(ctx context.Context) error {
		return c.sc.Mutate(ctx, m, input, vars)
	}
	limitEvents := make(chan RateLimitingEvent)
	defer close(limitEvents)
	go func() {
		for e := range limitEvents {
			select {
			case <-ctx.Done():
				return
			case out <- core.NewExportRateLimiting(e.msg):
			}
		}
	}()
	return c.callWithRetry(mutFun, ctx, limitEvents)
}

// queryWithLimitEvents calls the github api with a graphql query and it sends rate limiting events on the channel limitEvents.
func (c *client) queryWithLimitEvents(ctx context.Context, query interface{}, vars map[string]interface{}, limitEvents chan<- RateLimitingEvent) error {
	// prepare a closure fot the query
	queryFun := func(ctx context.Context) error {
		return c.sc.Query(ctx, query, vars)
	}
	return c.callWithRetry(queryFun, ctx, limitEvents)
}

// queryPrintMsgs calls the github api with a graphql query and it prints for ever rate limiting event a message to stdout.
func (c *client) queryPrintMsgs(ctx context.Context, query interface{}, vars map[string]interface{}) error {
	queryFun := func(ctx context.Context) error {
		return c.sc.Query(ctx, query, vars)
	}
	limitEvents := make(chan RateLimitingEvent)
	defer close(limitEvents)
	go func() {
		for e := range limitEvents {
			fmt.Println(e.msg)
		}
	}()
	return c.callWithRetry(queryFun, ctx, limitEvents)
}

// queryWithImportEvents calls the github api with a graphql query and it sends rate limiting events to the channel of import events.
func (c *client) queryWithImportEvents(ctx context.Context, query interface{}, vars map[string]interface{}, importEvents chan<- ImportEvent) error {
	// forward rate limiting events to channel of import events
	limitEvents := make(chan RateLimitingEvent)
	defer close(limitEvents)
	go func() {
		for e := range limitEvents {
			select {
			case <-ctx.Done():
				return
			case importEvents <- e:
			}
		}
	}()
	return c.queryWithLimitEvents(ctx, query, vars, limitEvents)
}

func (c *client) callWithRetry(fun func(context.Context) error, ctx context.Context, events chan<- RateLimitingEvent) error {
	var err error
	if err = c.callWithLimit(fun, ctx, events); err == nil {
		return nil
	}
	// failure; the reason may be temporary network problems or internal errors
	// on the github servers. Internal errors on the github servers are quite common.
	// Retry
	retries := 3
	for i := 0; i < retries; i++ {
		// wait a few seconds before retry
		sleepTime := time.Duration(8*(i+1)) * time.Second
		timer := time.NewTimer(sleepTime)
		select {
		case <-ctx.Done():
			stop(timer)
			return ctx.Err()
		case <-timer.C:
			err = c.callWithLimit(fun, ctx, events)
			if err == nil {
				return nil
			}
		}
	}
	return err
}

func (c *client) callWithLimit(fun func(context.Context) error, ctx context.Context, events chan<- RateLimitingEvent) error {
	qctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	// call the function fun()
	err := fun(qctx)
	if err == nil {
		return nil
	}
	// matching the error string
	if strings.Contains(err.Error(), "API rate limit exceeded") {
		// a rate limit error
		qctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
		// Use a separate query to get Github rate limiting information.
		limitQuery := rateLimitQuery{}
		if err := c.sc.Query(qctx, &limitQuery, map[string]interface{}{}); err != nil {
			return err
		}
		// Get the time when Github will reset the rate limit of their API.
		resetTime := limitQuery.RateLimit.ResetAt.Time
		// Add a few seconds for good measure
		resetTime = resetTime.Add(8 * time.Second)
		msg := fmt.Sprintf("Github GraphQL API rate limit. This process will sleep until %s.", resetTime.String())
		// Send message about rate limiting event.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case events <- RateLimitingEvent{msg}:
		}
		// Pause current goroutine
		timer := time.NewTimer(time.Until(resetTime))
		select {
		case <-ctx.Done():
			stop(timer)
			return ctx.Err()
		case <-timer.C:
		}
		// call the function fun() again
		qctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
		err = fun(qctx)
		return err // might be nil
	} else {
		return err
	}
}

func stop(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
