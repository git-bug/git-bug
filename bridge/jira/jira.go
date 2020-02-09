// Package jira contains the Jira bridge implementation
package jira

import (
	"sort"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
)

const (
	target = "jira"

	metaKeyJiraLogin = "jira-login"

	keyServer          = "server"
	keyProject         = "project"
	keyCredentialsType = "credentials-type"
	keyCredentialsFile = "credentials-file"
	keyUsername        = "username"
	keyPassword        = "password"
	keyIDMap           = "bug-id-map"
	keyIDRevMap        = "bug-id-revmap"
	keyCreateDefaults  = "create-issue-defaults"
	keyCreateGitBug    = "create-issue-gitbug-id"

	defaultTimeout = 60 * time.Second
)

var _ core.BridgeImpl = &Jira{}

// Jira Main object for the bridge
type Jira struct{}

// Target returns "jira"
func (*Jira) Target() string {
	return target
}

func (*Jira) LoginMetaKey() string {
	return metaKeyJiraLogin
}

// NewImporter returns the jira importer
func (*Jira) NewImporter() core.Importer {
	return &jiraImporter{}
}

// NewExporter returns the jira exporter
func (*Jira) NewExporter() core.Exporter {
	return &jiraExporter{}
}

// stringInSlice returns true if needle is found in haystack
func stringInSlice(needle string, haystack []string) bool {
	for _, match := range haystack {
		if match == needle {
			return true
		}
	}
	return false
}

// Given two string slices, return three lists containing:
// 1. elements found only in the first input list
// 2. elements found only in the second input list
// 3. elements found in both input lists
func setSymmetricDifference(setA, setB []string) ([]string, []string, []string) {
	sort.Strings(setA)
	sort.Strings(setB)

	maxLen := len(setA) + len(setB)
	onlyA := make([]string, 0, maxLen)
	onlyB := make([]string, 0, maxLen)
	both := make([]string, 0, maxLen)

	idxA := 0
	idxB := 0

	for idxA < len(setA) && idxB < len(setB) {
		if setA[idxA] < setB[idxB] {
			// In the first set, but not the second
			onlyA = append(onlyA, setA[idxA])
			idxA++
		} else if setA[idxA] > setB[idxB] {
			// In the second set, but not the first
			onlyB = append(onlyB, setB[idxB])
			idxB++
		} else {
			// In both
			both = append(both, setA[idxA])
			idxA++
			idxB++
		}
	}

	for ; idxA < len(setA); idxA++ {
		// Leftovers in the first set, not the second
		onlyA = append(onlyA, setA[idxA])
	}

	for ; idxB < len(setB); idxB++ {
		// Leftovers in the second set, not the first
		onlyB = append(onlyB, setB[idxB])
	}

	return onlyA, onlyB, both
}
