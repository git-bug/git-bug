package ado

import (
	"context"
	"fmt"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/input"
)

const moreConfigText = `
NOTE: There are a few optional configuration values that you can set in your git
configuration to influence the behavior of the bridge:
  - git-bug.bridge.<name>.create-work-item-type (default "Task")
  - git-bug.bridge.<name>.closed-states (default "Closed, Done, Completed, Removed")
  - git-bug.bridge.<name>.open-state-target (default "Active")
  - git-bug.bridge.<name>.closed-state-target (default "Closed")
`

const patHelpText = `
The Azure DevOps bridge authenticates with a Personal Access Token (PAT). Create
one at https://dev.azure.com/<organization>/_usersSettings/tokens with at least
the "Work Items (Read & Write)" scope.`

func (*AzureDevOps) ValidParams() map[string]interface{} {
	return map[string]interface{}{
		"BaseURL":    nil,
		"Owner":      nil, // Azure DevOps organization / collection
		"Project":    nil,
		"Login":      nil,
		"CredPrefix": nil,
		"TokenRaw":   nil,
	}
}

// Configure sets up the bridge configuration.
func (a *AzureDevOps) Configure(repo *cache.RepoCache, params core.BridgeParams, interactive bool) (core.Configuration, error) {
	var err error

	baseURL := params.BaseURL
	if baseURL == "" {
		if interactive {
			baseURL, err = input.PromptDefault(
				"Azure DevOps instance URL", "URL", defaultBaseURL, input.Required, input.IsURL)
			if err != nil {
				return nil, err
			}
		} else {
			baseURL = defaultBaseURL
		}
	}

	organization := params.Owner
	if organization == "" {
		if !interactive {
			return nil, fmt.Errorf("non-interactive mode is active; please specify the Azure DevOps organization via the --owner option")
		}
		organization, err = input.Prompt("Azure DevOps organization", "organization", input.Required)
		if err != nil {
			return nil, err
		}
	}

	project := params.Project
	if project == "" {
		if !interactive {
			return nil, fmt.Errorf("non-interactive mode is active; please specify the project via the --project option")
		}
		project, err = input.Prompt("Azure DevOps project", "project", input.Required)
		if err != nil {
			return nil, err
		}
	}

	login := params.Login
	if login == "" {
		if !interactive {
			return nil, fmt.Errorf("non-interactive mode is active; please specify the login via the --login option")
		}
		login, err = input.Prompt("Azure DevOps login (email)", "login", input.Required)
		if err != nil {
			return nil, err
		}
	}

	var cred auth.Credential
	switch {
	case params.CredPrefix != "":
		cred, err = auth.LoadWithPrefix(repo, params.CredPrefix)
		if err != nil {
			return nil, err
		}
		if _, ok := cred.(*auth.Token); !ok {
			return nil, fmt.Errorf("the azuredevops bridge only supports token (PAT) credentials")
		}
	default:
		token := params.TokenRaw
		if token == "" {
			if !interactive {
				return nil, fmt.Errorf("non-interactive mode is active; please specify the PAT via the --token option")
			}
			fmt.Println(patHelpText)
			token, err = input.PromptPassword("Personal Access Token", "token", input.Required)
			if err != nil {
				return nil, err
			}
		}
		t := auth.NewToken(target, token)
		t.SetMetadata(auth.MetaKeyLogin, login)
		t.SetMetadata(auth.MetaKeyBaseURL, baseURL)
		cred = t
	}

	conf := make(core.Configuration)
	conf[core.ConfigKeyTarget] = target
	conf[confKeyBaseURL] = baseURL
	conf[confKeyOrganization] = organization
	conf[confKeyProject] = project
	conf[confKeyDefaultLogin] = login

	if err = a.ValidateConfig(conf); err != nil {
		return nil, err
	}

	fmt.Printf("Checking project access ...\n")
	client, err := buildClient(context.TODO(), baseURL, organization, project, cred)
	if err != nil {
		return nil, err
	}
	if _, err = client.GetProject(); err != nil {
		return nil, fmt.Errorf(
			"project %s doesn't exist in organization %s on %s, or the token is invalid: %w",
			project, organization, baseURL, err)
	}

	// store the now-validated token
	if !auth.IdExist(repo, cred.ID()) {
		if err = auth.Store(repo, cred); err != nil {
			return nil, err
		}
	}

	if err = core.FinishConfig(repo, metaKeyAdoLogin, login); err != nil {
		return nil, err
	}

	fmt.Print(moreConfigText)
	return conf, nil
}

// ValidateConfig returns an error if a required configuration key is missing.
func (*AzureDevOps) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}
	for _, key := range []string{confKeyBaseURL, confKeyOrganization, confKeyProject, confKeyDefaultLogin} {
		if _, ok := conf[key]; !ok {
			return fmt.Errorf("missing %s key", key)
		}
	}
	return nil
}
