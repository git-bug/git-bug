package launchpad

/*
 * A wrapper around the Launchpad API. The documentation can be found at:
 * https://launchpad.net/+apidoc/devel.html
 *
 * TODO:
 * - Retrieve bug status
 * - Retrieve activity log
 * - SearchTasks should yield bugs one by one
 *
 * TODO (maybe):
 * - Authentication (this might help retrieving email adresses)
 */

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const apiRoot = "https://api.launchpad.net/devel"

// Person describes a person on Launchpad (a bug owner, a message author, ...).
type LPPerson struct {
	Name  string `json:"display_name"`
	Login string `json:"name"`
}

// Caching all the LPPerson we know.
// The keys are links to an owner page, such as
// https://api.launchpad.net/devel/~login
var personCache = make(map[string]LPPerson)

// LPBug describes a Launchpad bug.
type LPBug struct {
	Title       string   `json:"title"`
	ID          int      `json:"id"`
	Owner       LPPerson `json:"owner_link"`
	Description string   `json:"description"`
	CreatedAt   string   `json:"date_created"`
	Messages    []LPMessage
}

// LPMessage describes a comment on a bug report
type LPMessage struct {
	Content   string   `json:"content"`
	CreatedAt string   `json:"date_created"`
	Owner     LPPerson `json:"owner_link"`
	ID        string   `json:"self_link"`
}

type launchpadBugEntry struct {
	BugLink  string `json:"bug_link"`
	SelfLink string `json:"self_link"`
}

type launchpadAnswer struct {
	Entries  []launchpadBugEntry `json:"entries"`
	Start    int                 `json:"start"`
	NextLink string              `json:"next_collection_link"`
}

type launchpadMessageAnswer struct {
	Entries  []LPMessage `json:"entries"`
	NextLink string      `json:"next_collection_link"`
}

type launchpadAPI struct {
	client *http.Client
}

func (lapi *launchpadAPI) Init() error {
	lapi.client = &http.Client{
		Timeout: defaultTimeout,
	}
	return nil
}

func (lapi *launchpadAPI) SearchTasks(ctx context.Context, project string) ([]LPBug, error) {
	var bugs []LPBug

	// First, let us build the URL. Not all statuses are included by
	// default, so we have to explicitely enumerate them.
	validStatuses := [13]string{
		"New", "Incomplete", "Opinion", "Invalid",
		"Won't Fix", "Expired", "Confirmed", "Triaged",
		"In Progress", "Fix Committed", "Fix Released",
		"Incomplete (with response)", "Incomplete (without response)",
	}
	queryParams := url.Values{}
	queryParams.Add("ws.op", "searchTasks")
	queryParams.Add("order_by", "-date_last_updated")
	for _, validStatus := range validStatuses {
		queryParams.Add("status", validStatus)
	}
	lpURL := fmt.Sprintf("%s/%s?%s", apiRoot, project, queryParams.Encode())

	for {
		req, err := http.NewRequest("GET", lpURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := lapi.client.Do(req)
		if err != nil {
			return nil, err
		}

		var result launchpadAnswer

		err = json.NewDecoder(resp.Body).Decode(&result)
		_ = resp.Body.Close()

		if err != nil {
			return nil, err
		}

		for _, bugEntry := range result.Entries {
			bug, err := lapi.queryBug(ctx, bugEntry.BugLink)
			if err == nil {
				bugs = append(bugs, bug)
			}
		}

		// Launchpad only returns 75 results at a time. We get the next
		// page and run another query, unless there is no other page.
		lpURL = result.NextLink
		if lpURL == "" {
			break
		}
	}

	return bugs, nil
}

func (lapi *launchpadAPI) queryBug(ctx context.Context, url string) (LPBug, error) {
	var bug LPBug

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return bug, err
	}
	req = req.WithContext(ctx)

	resp, err := lapi.client.Do(req)
	if err != nil {
		return bug, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&bug); err != nil {
		return bug, err
	}

	/* Fetch messages */
	messagesCollectionLink := fmt.Sprintf("%s/bugs/%d/messages", apiRoot, bug.ID)
	messages, err := lapi.queryMessages(ctx, messagesCollectionLink)
	if err != nil {
		return bug, err
	}
	bug.Messages = messages

	return bug, nil
}

func (lapi *launchpadAPI) queryMessages(ctx context.Context, messagesURL string) ([]LPMessage, error) {
	var messages []LPMessage

	for {
		req, err := http.NewRequest("GET", messagesURL, nil)
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)

		resp, err := lapi.client.Do(req)
		if err != nil {
			return nil, err
		}

		var result launchpadMessageAnswer

		err = json.NewDecoder(resp.Body).Decode(&result)
		_ = resp.Body.Close()

		if err != nil {
			return nil, err
		}

		messages = append(messages, result.Entries...)

		// Launchpad only returns 75 results at a time. We get the next
		// page and run another query, unless there is no other page.
		messagesURL = result.NextLink
		if messagesURL == "" {
			break
		}
	}
	return messages, nil
}
