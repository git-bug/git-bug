package ado

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
)

func TestBuildWiql(t *testing.T) {
	since := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	withSince := buildWiql("My Project", since)
	want := "SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = 'My Project'" +
		" AND [System.ChangedDate] >= '2026-01-02' ORDER BY [System.ChangedDate] ASC"
	if withSince != want {
		t.Errorf("buildWiql with since:\n got: %s\nwant: %s", withSince, want)
	}

	zero := buildWiql("P", time.Time{})
	wantZero := "SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = 'P' ORDER BY [System.ChangedDate] ASC"
	if zero != wantZero {
		t.Errorf("buildWiql zero since:\n got: %s\nwant: %s", zero, wantZero)
	}
}

func TestEscapeWiql(t *testing.T) {
	if got := escapeWiql("O'Brien's Project"); got != "O''Brien''s Project" {
		t.Errorf("escapeWiql = %q, want %q", got, "O''Brien''s Project")
	}
}

func TestIsClosedState(t *testing.T) {
	cases := []struct {
		state  string
		closed bool
	}{
		{"New", false},
		{"Active", false},
		{"Resolved", false},
		{"Closed", true},
		{"closed", true}, // case-insensitive
		{"Done", true},
		{"Completed", true},
		{"Removed", true},
		{"", false},
	}
	for _, tc := range cases {
		if got := isClosedState(tc.state, defaultClosedStates); got != tc.closed {
			t.Errorf("isClosedState(%q) = %v, want %v", tc.state, got, tc.closed)
		}
	}
}

func TestClosedStatesFromConf(t *testing.T) {
	def := closedStatesFromConf(core.Configuration{})
	if len(def) != len(defaultClosedStates) {
		t.Errorf("default closed states = %v, want %v", def, defaultClosedStates)
	}

	custom := closedStatesFromConf(core.Configuration{confKeyClosedStates: "Done, Shipped ,Won't Fix"})
	want := []string{"Done", "Shipped", "Won't Fix"}
	if len(custom) != len(want) {
		t.Fatalf("custom closed states = %v, want %v", custom, want)
	}
	for i := range want {
		if custom[i] != want[i] {
			t.Errorf("custom[%d] = %q, want %q", i, custom[i], want[i])
		}
	}
}

func TestParseAndJoinTags(t *testing.T) {
	tags := parseTags("bug; ui ; ; regression")
	want := []string{"bug", "ui", "regression"}
	if len(tags) != len(want) {
		t.Fatalf("parseTags = %v, want %v", tags, want)
	}
	for i := range want {
		if tags[i] != want[i] {
			t.Errorf("tags[%d] = %q, want %q", i, tags[i], want[i])
		}
	}

	if got := joinTags(want); got != "bug; ui; regression" {
		t.Errorf("joinTags = %q, want %q", got, "bug; ui; regression")
	}
	if got := parseTags(""); len(got) != 0 {
		t.Errorf("parseTags(empty) = %v, want empty", got)
	}
}

func TestTagSetDifference(t *testing.T) {
	added, removed := tagSetDifference(
		[]string{"a", "b", "c"},
		[]string{"b", "c", "d", "e"},
	)
	if len(added) != 2 || added[0] != "d" || added[1] != "e" {
		t.Errorf("added = %v, want [d e]", added)
	}
	if len(removed) != 1 || removed[0] != "a" {
		t.Errorf("removed = %v, want [a]", removed)
	}
}

func TestParseDisplayString(t *testing.T) {
	name, email := parseDisplayString("Jane Doe <jane@example.com>")
	if name != "Jane Doe" || email != "jane@example.com" {
		t.Errorf("parseDisplayString = (%q, %q), want (Jane Doe, jane@example.com)", name, email)
	}

	name, email = parseDisplayString("Just A Name")
	if name != "Just A Name" || email != "" {
		t.Errorf("parseDisplayString plain = (%q, %q), want (Just A Name, )", name, email)
	}
}

func TestFieldString(t *testing.T) {
	if got := fieldString(nil); got != "" {
		t.Errorf("fieldString(nil) = %q, want empty", got)
	}
	if got := fieldString("hello"); got != "hello" {
		t.Errorf("fieldString(string) = %q, want hello", got)
	}
	if got := fieldString(42); got != "42" {
		t.Errorf("fieldString(int) = %q, want 42", got)
	}
}

func TestConfOr(t *testing.T) {
	conf := core.Configuration{"a": "x", "b": ""}
	if got := confOr(conf, "a", "def"); got != "x" {
		t.Errorf("confOr set = %q, want x", got)
	}
	if got := confOr(conf, "b", "def"); got != "def" {
		t.Errorf("confOr empty = %q, want def", got)
	}
	if got := confOr(conf, "missing", "def"); got != "def" {
		t.Errorf("confOr missing = %q, want def", got)
	}
}

func TestIdentityUnmarshalObject(t *testing.T) {
	var id Identity
	err := json.Unmarshal([]byte(`{"displayName":"Jane","uniqueName":"jane@x.com","id":"guid-1","descriptor":"desc-1"}`), &id)
	if err != nil {
		t.Fatalf("unmarshal object: %v", err)
	}
	if id.DisplayName != "Jane" || id.Key() != "jane@x.com" || id.Email() != "jane@x.com" {
		t.Errorf("identity object = %+v, key=%q email=%q", id, id.Key(), id.Email())
	}
}

func TestIdentityUnmarshalString(t *testing.T) {
	var id Identity
	err := json.Unmarshal([]byte(`"John Doe <john@x.com>"`), &id)
	if err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}
	if id.DisplayName != "John Doe" || id.UniqueName != "john@x.com" || id.Key() != "john@x.com" {
		t.Errorf("identity string = %+v, key=%q", id, id.Key())
	}
}

func TestIdentityKeyFallback(t *testing.T) {
	id := Identity{DisplayName: "No Email"}
	if id.Key() != "No Email" {
		t.Errorf("Key fallback = %q, want No Email", id.Key())
	}
	if id.Email() != "" {
		t.Errorf("Email = %q, want empty", id.Email())
	}
}
