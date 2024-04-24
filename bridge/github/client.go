package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"

	"github.com/MichaelMure/git-bug/bridge/core"
)

var _ Client = &githubv4.Client{}

// Client is an interface conforming with githubv4.Client
type Client interface {
	Mutate(context.Context, interface{}, githubv4.Input, map[string]interface{}) error
	Query(context.Context, interface{}, map[string]interface{}) error
}

// rateLimitHandlerClient wraps the Github client and adds improved error handling and handling of
// Github's GraphQL rate limit.
type rateLimitHandlerClient struct {
	sc Client
}

func newRateLimitHandlerClient(httpClient *http.Client) *rateLimitHandlerClient {
	return &rateLimitHandlerClient{sc: githubv4.NewClient(httpClient)}
}

// mutate calls the github api with a graphql mutation and sends a core.ExportResult for each rate limiting event
func (c *rateLimitHandlerClient) mutate(ctx context.Context, m interface{}, input githubv4.Input, vars map[string]interface{}, out chan<- core.ExportResult) error {
	// prepare a closure for the mutation
	mutFun := func(ctx context.Context) error {
		return c.sc.Mutate(ctx, m, input, vars)
	}
	callback := func(msg string) {
		select {
		case <-ctx.Done():
		case out <- core.NewExportRateLimiting(msg):
		}
	}
	return c.callAPIAndRetry(ctx, mutFun, callback)
}

// queryImport calls the github api with a graphql query, and sends an ImportEvent for each rate limiting event
func (c *rateLimitHandlerClient) queryImport(ctx context.Context, query interface{}, vars map[string]interface{}, importEvents chan<- ImportEvent) error {
	// prepare a closure for the query
	queryFun := func(ctx context.Context) error {
		return c.sc.Query(ctx, query, vars)
	}
	callback := func(msg string) {
		select {
		case <-ctx.Done():
		case importEvents <- RateLimitingEvent{msg}:
		}
	}
	return c.callAPIAndRetry(ctx, queryFun, callback)
}

// queryExport calls the github api with a graphql query, and sends a core.ExportResult for each rate limiting event
func (c *rateLimitHandlerClient) queryExport(ctx context.Context, query interface{}, vars map[string]interface{}, out chan<- core.ExportResult) error {
	// prepare a closure for the query
	queryFun := func(ctx context.Context) error {
		return c.sc.Query(ctx, query, vars)
	}
	callback := func(msg string) {
		select {
		case <-ctx.Done():
		case out <- core.NewExportRateLimiting(msg):
		}
	}
	return c.callAPIAndRetry(ctx, queryFun, callback)
}

// queryPrintMsgs calls the github api with a graphql query, and prints a message to stdout for every rate limiting event .
func (c *rateLimitHandlerClient) queryPrintMsgs(ctx context.Context, query interface{}, vars map[string]interface{}) error {
	// prepare a closure for the query
	queryFun := func(ctx context.Context) error {
		return c.sc.Query(ctx, query, vars)
	}
	callback := func(msg string) {
		fmt.Println(msg)
	}
	return c.callAPIAndRetry(ctx, queryFun, callback)
}

// callAPIAndRetry calls the Github GraphQL API (indirectly through callAPIDealWithLimit) and in
// case of error it repeats the request to the Github API. The parameter `apiCall` is intended to be
// a closure containing a query or a mutation to the Github GraphQL API.
func (c *rateLimitHandlerClient) callAPIAndRetry(ctx context.Context, apiCall func(context.Context) error, rateLimitEvent func(msg string)) error {
	var err error
	if err = c.callAPIDealWithLimit(ctx, apiCall, rateLimitEvent); err == nil {
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
			err = c.callAPIDealWithLimit(ctx, apiCall, rateLimitEvent)
			if err == nil {
				return nil
			}
		}
	}
	return err
}

// callAPIDealWithLimit calls the Github GraphQL API and if the Github API returns a rate limiting
// error, then it waits until the rate limit is reset, and it repeats the request to the API. The
// parameter `apiCall` is intended to be a closure containing a query or a mutation to the Github
// GraphQL API.
func (c *rateLimitHandlerClient) callAPIDealWithLimit(ctx context.Context, apiCall func(context.Context) error, rateLimitCallback func(msg string)) error {
	qctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	// call the function fun()
	err := apiCall(qctx)
	if err == nil {
		return nil
	}
	// matching the error string
	if strings.Contains(err.Error(), "API rate limit exceeded") ||
		strings.Contains(err.Error(), "was submitted too quickly") {
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
		msg := fmt.Sprintf(
			"Github GraphQL API rate limit. This process will sleep until %s.",
			resetTime.String(),
		)
		// Send message about rate limiting event.
		rateLimitCallback(msg)

		// sanitize the reset time, in case the local clock is wrong
		waitTime := time.Until(resetTime)
		if waitTime < 0 {
			waitTime = 10 * time.Second
		}
		if waitTime > 30*time.Second {
			waitTime = 30 * time.Second
		}

		// Pause current goroutine
		timer := time.NewTimer(waitTime)
		select {
		case <-ctx.Done():
			stop(timer)
			return ctx.Err()
		case <-timer.C:
		}
		// call the function apiCall() again
		qctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
		err = apiCall(qctx)
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
