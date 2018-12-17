package launchpad

/*
 * A wrapper around the Launchpad API. The documentation can be found at:
 * https://launchpad.net/+apidoc/devel.html
 *
 * TODO:
 * - Retrieve all messages associated to bugs
 * - Retrieve bug status
 * - Retrieve activity log
 * - SearchTasks should yield bugs one by one
 *
 * TODO (maybe):
 * - Authentication (this might help retrieving email adresses)
 */

import (
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

func (owner *LPPerson) UnmarshalJSON(data []byte) error {
	type LPPersonX LPPerson // Avoid infinite recursion
	var ownerLink string
	if err := json.Unmarshal(data, &ownerLink); err != nil {
		return err
	}

	// First, try to gather info about the bug owner using our cache.
	if cachedPerson, hasKey := personCache[ownerLink]; hasKey {
		*owner = cachedPerson
		return nil
	}

	// If the bug owner is not already known, we have to send a request.
	req, err := http.NewRequest("GET", ownerLink, nil)
	if err != nil {
		return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()

	var p LPPersonX
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil
	}
	*owner = LPPerson(p)
	// Do not forget to update the cache.
	personCache[ownerLink] = *owner
	return nil
}

// LPBug describes a Launchpad bug.
type LPBug struct {
	Title       string   `json:"title"`
	ID          int      `json:"id"`
	Owner       LPPerson `json:"owner_link"`
	Description string   `json:"description"`
	CreatedAt   string   `json:"date_created"`
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

type launchpadAPI struct {
	client *http.Client
}

func (lapi *launchpadAPI) Init() error {
	lapi.client = &http.Client{}
	return nil
}

func (lapi *launchpadAPI) SearchTasks(project string) ([]LPBug, error) {
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

		defer resp.Body.Close()

		var result launchpadAnswer

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		for _, bugEntry := range result.Entries {
			bug, err := lapi.queryBug(bugEntry.BugLink)
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

func (lapi *launchpadAPI) queryBug(url string) (LPBug, error) {
	var bug LPBug

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return bug, err
	}

	resp, err := lapi.client.Do(req)
	if err != nil {
		return bug, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&bug); err != nil {
		return bug, err
	}

	return bug, nil
}
