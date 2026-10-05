// Package jira contains the Jira bridge implementation
package jira

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/commands/input"
	"github.com/git-bug/git-bug/repository"
)

const (
	target = "jira"

	metaKeyJiraId         = "jira-id"
	metaKeyJiraDerivedId  = "jira-derived-id"
	metaKeyJiraKey        = "jira-key"
	metaKeyJiraUser       = "jira-user"
	metaKeyJiraProject    = "jira-project"
	metaKeyJiraBaseUrl    = "jira-base-url"
	metaKeyJiraExportTime = "jira-export-time"
	metaKeyJiraLogin      = "jira-login"

	confKeyBaseUrl        = "base-url"
	confKeyProject        = "project"
	confKeyDefaultLogin   = "default-login"
	confKeyCredentialType = "credentials-type" // "SESSION" or "TOKEN"
	confKeyCredentialID   = "credential-id"
	confKeyIDMap          = "bug-id-map"
	confKeyIDRevMap       = "bug-id-revmap"
	// the issue type when exporting a new bug. Default is Story (10001)
	confKeyCreateDefaults = "create-issue-defaults"
	// if set, the bridge fill this JIRA field with the `git-bug` id when exporting
	confKeyCreateGitBug = "create-issue-gitbug-id"

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

func buildClient(ctx context.Context, baseURL string, credType string, cred auth.Credential) (*Client, error) {
	client := NewClient(ctx, baseURL)

	var login, password string

	switch cred := cred.(type) {
	case *auth.Token:
		if credType != "TOKEN" {
			return nil, fmt.Errorf("token credential requires Jira TOKEN authentication")
		}
		var ok bool
		login, ok = cred.GetMetadata(auth.MetaKeyLogin)
		if !ok || login == "" {
			return nil, fmt.Errorf("Jira token has no login")
		}
		password = cred.Value
	case *auth.LoginPassword:
		login = cred.Login
		password = cred.Password
	case *auth.Login:
		login = cred.Login
		p, err := input.PromptPassword(fmt.Sprintf("Password for %s", login), "password", input.Required)
		if err != nil {
			return nil, err
		}
		password = p
	default:
		return nil, fmt.Errorf("unsupported Jira credential %T", cred)
	}

	err := client.Login(credType, login, password)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// List credentials compatible with the selected authentication mode, including
// legacy TOKEN configurations that stored API tokens as passwords.
func listCredentials(repo repository.RepoKeyring, credType string, opts ...auth.ListOption) ([]auth.Credential, error) {
	var credentials []auth.Credential
	if credType == "TOKEN" {
		tokens, err := auth.List(repo, append(append([]auth.ListOption{}, opts...), auth.WithKind(auth.KindToken))...)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, tokens...)
	}
	passwords, err := auth.List(repo, append(append([]auth.ListOption{}, opts...), auth.WithKind(auth.KindLoginPassword), auth.WithKind(auth.KindLogin))...)
	if err != nil {
		return nil, err
	}
	// Legacy TOKEN configurations used password credentials for API tokens.
	// Honor the newest stored secret across both forms for unpinned bridges.
	credentials = append(credentials, passwords...)
	sort.SliceStable(credentials, func(i, j int) bool {
		if credentials[i].Kind() == auth.KindLogin || credentials[j].Kind() == auth.KindLogin {
			return credentials[i].Kind() != auth.KindLogin && credentials[j].Kind() == auth.KindLogin
		}
		return credentials[i].CreateTime().After(credentials[j].CreateTime())
	})
	return credentials, nil
}

func configuredCredentials(repo repository.RepoKeyring, conf core.Configuration, opts ...auth.ListOption) ([]auth.Credential, error) {
	credentials, err := listCredentials(repo, conf[confKeyCredentialType], opts...)
	if err != nil {
		return nil, err
	}
	if id := conf[confKeyCredentialID]; id != "" {
		for i, credential := range credentials {
			if credential.ID().String() == id {
				copy(credentials[1:i+1], credentials[:i])
				credentials[0] = credential
				return credentials, nil
			}
		}
		return nil, fmt.Errorf("configured Jira credential is missing or does not match this bridge")
	}
	return credentials, nil
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
