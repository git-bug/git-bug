// Package ado contains the Azure DevOps (Azure Boards) bridge implementation.
//
// The bridge synchronises git-bug bugs with Azure DevOps work items in both
// directions. Authentication is done with a Personal Access Token (PAT), which
// is sent as HTTP Basic auth (empty username, PAT as password) on every call.
package ado

import (
	"context"
	"fmt"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
)

const (
	target = "azuredevops"

	// defaultBaseURL is the instance root for Azure DevOps Services. On-prem
	// Azure DevOps Server users override this with --base-url.
	defaultBaseURL = "https://dev.azure.com"

	// metadata stored on the imported/exported bug and its operations
	metaKeyAdoID         = "ado-id"           // work item id, e.g. "1234"
	metaKeyAdoURL        = "ado-url"          // work item REST url
	metaKeyAdoProject    = "ado-project"      // project name or id
	metaKeyAdoOrg        = "ado-organization" // organization / collection
	metaKeyAdoBaseURL    = "ado-base-url"     // instance root url
	metaKeyAdoDerivedID  = "ado-derived-id"   // derived id for a single field change
	metaKeyAdoExportTime = "ado-export-time"  // time an operation was exported

	// metadata stored on the user identity
	metaKeyAdoLogin = "ado-login" // login used to match identities and credentials
	metaKeyAdoUser  = "ado-user"  // remote user unique key (email/descriptor/id)

	// configuration keys, stored under git-bug.bridge.<name>.*
	confKeyBaseURL      = "base-url"
	confKeyOrganization = "organization"
	confKeyProject      = "project"
	confKeyDefaultLogin = "default-login"
	// confKeyCreateType is the work item type used when exporting a new bug.
	// "Task" exists in every stock process (Basic, Agile, Scrum, CMMI).
	confKeyCreateType = "create-work-item-type"
	// confKeyClosedStates is a comma separated list of state names that map to
	// git-bug's "closed" status. Anything else maps to "open".
	confKeyClosedStates = "closed-states"
	// confKeyOpenStateTarget / confKeyClosedStateTarget are the ADO state names
	// used when exporting a git-bug open/close operation.
	confKeyOpenStateTarget   = "open-state-target"
	confKeyClosedStateTarget = "closed-state-target"

	defaultCreateType        = "Task"
	defaultOpenStateTarget   = "Active"
	defaultClosedStateTarget = "Closed"
	defaultTimeout           = 60 * time.Second
)

// defaultClosedStates lists the stock ADO states that mean "done" across the
// built-in processes.
var defaultClosedStates = []string{"Closed", "Done", "Completed", "Removed"}

var _ core.BridgeImpl = &AzureDevOps{}

// AzureDevOps is the main object for the bridge.
type AzureDevOps struct{}

// Target returns "azuredevops".
func (*AzureDevOps) Target() string {
	return target
}

// LoginMetaKey returns the identity metadata key holding the remote login.
func (*AzureDevOps) LoginMetaKey() string {
	return metaKeyAdoLogin
}

// NewImporter returns the Azure DevOps importer.
func (*AzureDevOps) NewImporter() core.Importer {
	return &adoImporter{}
}

// NewExporter returns the Azure DevOps exporter.
func (*AzureDevOps) NewExporter() core.Exporter {
	return &adoExporter{}
}

// buildClient builds an Azure DevOps REST client from a PAT credential.
func buildClient(ctx context.Context, baseURL, organization, project string, cred auth.Credential) (*Client, error) {
	token, ok := cred.(*auth.Token)
	if !ok {
		return nil, fmt.Errorf("the azuredevops bridge only supports token (PAT) credentials, got %s", cred.Kind())
	}
	return NewClient(ctx, baseURL, organization, project, token.Value), nil
}
