package ado

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
)

// wiqlDateFormat is the literal format used for WIQL date comparisons. WIQL
// compares [System.ChangedDate] with date precision and rejects a time
// component, so the query floors "since" to the day. This over-selects items
// changed earlier on the same day, which is harmless: the importer is
// idempotent and ImportAll additionally filters on the precise ChangedDate.
const wiqlDateFormat = "2006-01-02"

// buildWiql builds the WIQL query used for incremental import. It selects all
// work item ids in the project changed on or after "since", oldest first so the
// importer processes history in order.
func buildWiql(project string, since time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = '%s'",
		escapeWiql(project))
	if !since.IsZero() {
		fmt.Fprintf(&b, " AND [System.ChangedDate] >= '%s'",
			since.UTC().Format(wiqlDateFormat))
	}
	b.WriteString(" ORDER BY [System.ChangedDate] ASC")
	return b.String()
}

// escapeWiql escapes a single-quoted WIQL string literal by doubling quotes.
func escapeWiql(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// isClosedState reports whether an Azure DevOps state name maps to git-bug's
// "closed" status. Matching is case-insensitive.
func isClosedState(state string, closedStates []string) bool {
	state = strings.TrimSpace(state)
	for _, cs := range closedStates {
		if strings.EqualFold(state, strings.TrimSpace(cs)) {
			return true
		}
	}
	return false
}

// closedStatesFromConf returns the configured closed-state names, or the
// built-in default set when unset.
func closedStatesFromConf(conf core.Configuration) []string {
	raw, ok := conf[confKeyClosedStates]
	if !ok || strings.TrimSpace(raw) == "" {
		return defaultClosedStates
	}
	return splitAndTrim(raw, ",")
}

// parseTags splits an Azure DevOps System.Tags string into individual tags.
// Azure DevOps stores tags separated by "; ".
func parseTags(s string) []string {
	return splitAndTrim(s, ";")
}

// joinTags renders a tag slice back into the Azure DevOps System.Tags format.
func joinTags(tags []string) string {
	return strings.Join(tags, "; ")
}

// tagSetDifference returns the tags added and removed going from setA to setB.
func tagSetDifference(setA, setB []string) (added, removed []string) {
	inA := make(map[string]bool, len(setA))
	for _, t := range setA {
		inA[t] = true
	}
	inB := make(map[string]bool, len(setB))
	for _, t := range setB {
		inB[t] = true
	}
	for _, t := range setB {
		if !inA[t] {
			added = append(added, t)
		}
	}
	for _, t := range setA {
		if !inB[t] {
			removed = append(removed, t)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// splitAndTrim splits s on sep, trims whitespace and drops empty entries.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseDisplayString parses an Azure DevOps "Display Name <email>" identity
// string into its name and email parts.
func parseDisplayString(s string) (name, email string) {
	s = strings.TrimSpace(s)
	open := strings.LastIndex(s, "<")
	closeIdx := strings.LastIndex(s, ">")
	if open >= 0 && closeIdx > open {
		email = strings.TrimSpace(s[open+1 : closeIdx])
		name = strings.TrimSpace(s[:open])
		return name, email
	}
	return s, ""
}

// fieldString converts a work item revision field value into a string. Azure
// DevOps returns the fields the bridge reads (title, state, tags, description)
// as strings; anything else falls back to fmt formatting, and nil becomes "".
func fieldString(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	default:
		return fmt.Sprint(val)
	}
}
