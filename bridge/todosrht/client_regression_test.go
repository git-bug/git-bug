package todosrht

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMutationInputsOmitOptionalFields(t *testing.T) {
	newBody := "new"
	emptyBody := ""
	for _, tc := range []struct {
		name  string
		input interface{}
		want  string
	}{
		{"comment", SubmitCommentInput{Text: "hello"}, `{"text":"hello"}`},
		{"title", UpdateTicketInput{Subject: "new"}, `{"subject":"new"}`},
		{"body", UpdateTicketInput{Body: &newBody}, `{"body":"new"}`},
		{"empty_body", UpdateTicketInput{Body: &emptyBody}, `{"body":null}`},
		{"status", UpdateStatusInput{Status: TicketStatusReported}, `{"status":"REPORTED"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.input)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(encoded))
		})
	}
}

func TestTokenNotForwardedOnRedirect(t *testing.T) {
	var received string
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("Authorization")
		fmt.Fprint(w, `{ "data": {} }`)
	}))
	defer destination.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL+"/query", http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	client := NewTodoSClient(context.Background(), source.URL, "secret")
	client.client.Transport = &authTransport{token: "secret", base: source.Client().Transport}
	err := client.executeRequest(context.Background(), `query { version { major } }`, nil, &struct{}{})
	require.ErrorContains(t, err, "cross-origin redirect")
	require.Empty(t, received)
}

func TestTrackerQueriesUseSchemaPaths(t *testing.T) {
	for _, ref := range []string{"tracker", "~owner/tracker"} {
		t.Run(ref, func(t *testing.T) {
			var queries []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request GraphQLRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				queries = append(queries, request.Query)
				require.Equal(t, "tracker", request.Variables["trackerName"])
				if strings.HasPrefix(ref, "~") {
					require.Equal(t, "owner", request.Variables["owner"])
					require.Contains(t, request.Query, "me: user(username: $owner)")
				} else {
					require.NotContains(t, request.Query, "$owner:")
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"data":{"me":{"tracker":{"id":7,"name":"tracker","tickets":{"results":[]},"labels":{"results":[]},"ticket":{"events":{"results":[]}}}}}}`)
			}))
			defer server.Close()
			client := NewTodoSClient(context.Background(), server.URL, "secret")
			tracker, err := client.GetTracker(context.Background(), ref)
			require.NoError(t, err)
			require.Equal(t, 7, tracker.Id)
			_, _, err = client.GetTickets(context.Background(), ref, nil)
			require.NoError(t, err)
			_, _, err = client.GetEvents(context.Background(), ref, 1, nil)
			require.NoError(t, err)
			_, err = client.GetLabels(context.Background(), ref, nil)
			require.NoError(t, err)
			require.Len(t, queries, 4)
			for _, query := range queries {
				require.NotContains(t, query, "tracker(id:")
				require.NotContains(t, query, "{\n\t\ttracker(name:")
			}
		})
	}
}
