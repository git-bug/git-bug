package ado

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient spins up an httptest server with the given handler and returns
// a Client pointed at it.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewClient(context.Background(), srv.URL, "myorg", "myproj", "SECRET")
	return c, srv
}

func TestClientAuthHeader(t *testing.T) {
	var gotAuth string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"id":"p","name":"myproj"}`))
	})
	defer srv.Close()

	if _, err := c.GetProject(); err != nil {
		t.Fatalf("GetProject: %v", err)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(":SECRET"))
	if gotAuth != wantAuth {
		t.Errorf("auth header = %q, want %q", gotAuth, wantAuth)
	}
}

func TestClientGetProjectPath(t *testing.T) {
	var gotPath string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"guid","name":"myproj"}`))
	})
	defer srv.Close()

	p, err := c.GetProject()
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if gotPath != "/myorg/_apis/projects/myproj" {
		t.Errorf("path = %q, want /myorg/_apis/projects/myproj", gotPath)
	}
	if p.Name != "myproj" || p.ID != "guid" {
		t.Errorf("project = %+v", p)
	}
}

func TestClientSearchWorkItemIDs(t *testing.T) {
	var gotQuery string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal(body, &req)
		gotQuery = req.Query
		_, _ = w.Write([]byte(`{"workItems":[{"id":1},{"id":2},{"id":3}]}`))
	})
	defer srv.Close()

	ids, err := c.SearchWorkItemIDs("SELECT [System.Id] FROM WorkItems")
	if err != nil {
		t.Fatalf("SearchWorkItemIDs: %v", err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[2] != 3 {
		t.Errorf("ids = %v, want [1 2 3]", ids)
	}
	if !strings.Contains(gotQuery, "System.Id") {
		t.Errorf("query not forwarded: %q", gotQuery)
	}
}

func TestClientGetWorkItems(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "ids=10,11") {
			t.Errorf("missing ids in query: %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"count":2,"value":[
			{"id":10,"rev":1,"fields":{"System.Title":"First","System.State":"New","System.CreatedBy":{"displayName":"A","uniqueName":"a@x.com"}}},
			{"id":11,"rev":3,"fields":{"System.Title":"Second","System.State":"Closed","System.Tags":"bug; ui"}}
		]}`))
	})
	defer srv.Close()

	items, err := c.GetWorkItems([]int{10, 11})
	if err != nil {
		t.Fatalf("GetWorkItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Fields.Title != "First" || items[0].Fields.CreatedBy.Key() != "a@x.com" {
		t.Errorf("item0 = %+v", items[0])
	}
	if items[1].Fields.State != "Closed" || joinTags(parseTags(items[1].Fields.Tags)) != "bug; ui" {
		t.Errorf("item1 = %+v", items[1])
	}
}

func TestClientCreateWorkItem(t *testing.T) {
	var gotPath, gotContentType string
	var gotOps []patchOp
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotOps)
		_, _ = w.Write([]byte(`{"id":42,"rev":1,"url":"http://x/42","fields":{"System.Title":"T"}}`))
	})
	defer srv.Close()

	wi, err := c.CreateWorkItem("Task", []patchOp{
		{Op: "add", Path: "/fields/System.Title", Value: "T"},
	})
	if err != nil {
		t.Fatalf("CreateWorkItem: %v", err)
	}
	if wi.ID != 42 {
		t.Errorf("created id = %d, want 42", wi.ID)
	}
	if gotPath != "/myorg/myproj/_apis/wit/workitems/$Task" {
		t.Errorf("path = %q", gotPath)
	}
	if gotContentType != jsonPatchContentType {
		t.Errorf("content-type = %q, want %q", gotContentType, jsonPatchContentType)
	}
	if len(gotOps) != 1 || gotOps[0].Path != "/fields/System.Title" {
		t.Errorf("ops = %+v", gotOps)
	}
}

func TestClientAddComment(t *testing.T) {
	var gotPath string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(body, &req)
		_, _ = w.Write([]byte(`{"id":7,"text":"` + req.Text + `"}`))
	})
	defer srv.Close()

	cm, err := c.AddComment(42, "hello world")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if cm.ID != 7 || cm.Text != "hello world" {
		t.Errorf("comment = %+v", cm)
	}
	if gotPath != "/myorg/myproj/_apis/wit/workItems/42/comments" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestClientErrorStatus(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"TF400813: unauthorized"}`))
	})
	defer srv.Close()

	_, err := c.GetProject()
	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %v, want it to mention 401", err)
	}
}

func TestClientDeleteWorkItem(t *testing.T) {
	var gotMethod, gotQuery string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
	})
	defer srv.Close()

	if err := c.DeleteWorkItem(42, false); err != nil {
		t.Fatalf("DeleteWorkItem(recycle): %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if strings.Contains(gotQuery, "destroy") {
		t.Errorf("recycle-bin delete must omit the destroy param, got query %q", gotQuery)
	}

	if err := c.DeleteWorkItem(42, true); err != nil {
		t.Fatalf("DeleteWorkItem(destroy): %v", err)
	}
	if !strings.Contains(gotQuery, "destroy=true") {
		t.Errorf("permanent delete must include destroy=true, got query %q", gotQuery)
	}
}
