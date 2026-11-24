package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func searchKeys(t *testing.T, client *Client) []string {
	t.Helper()
	var keys []string
	iter := client.IterSearch("project=TEST", 2)
	for iter.HasNext() {
		keys = append(keys, iter.Next().Key)
	}
	require.False(t, iter.HasError(), "%v", iter.Err)
	return keys
}

func TestSearchCloudPagesWithNextPageToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/search/jql", r.URL.Path)
		var body map[string]any
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}
		assert.NotContains(t, body, "startAt")

		switch body["nextPageToken"] {
		case nil:
			_, _ = w.Write([]byte(`{"issues": [{"key": "TEST-1"}, {"key": "TEST-2"}], "nextPageToken": "page2"}`))
		case "page2":
			_, _ = w.Write([]byte(`{"issues": [{"key": "TEST-3"}], "isLast": true}`))
		default:
			t.Errorf("unexpected nextPageToken %v", body["nextPageToken"])
		}
	}))
	defer server.Close()

	client := NewClient(context.Background(), server.URL)
	require.Equal(t, []string{"TEST-1", "TEST-2", "TEST-3"}, searchKeys(t, client))
}

func TestSearchDataCenterFallsBackToStartAt(t *testing.T) {
	searchJQLCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/2/search/jql" {
			searchJQLCalls++
			http.NotFound(w, r)
			return
		}
		assert.Equal(t, "/rest/api/2/search", r.URL.Path)
		var body SearchRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			return
		}
		assert.Empty(t, body.NextPageToken)

		switch body.StartAt {
		case 0:
			_, _ = w.Write([]byte(`{"startAt": 0, "total": 3, "issues": [{"key": "TEST-1"}, {"key": "TEST-2"}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"startAt": 2, "total": 3, "issues": [{"key": "TEST-3"}]}`))
		default:
			t.Errorf("unexpected startAt %d", body.StartAt)
		}
	}))
	defer server.Close()

	client := NewClient(context.Background(), server.URL)
	require.Equal(t, []string{"TEST-1", "TEST-2", "TEST-3"}, searchKeys(t, client))
	require.Equal(t, 1, searchJQLCalls)
}
