package jira

import (
	"context"
	"fmt"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/repository"
)

const moreConfigText = `
NOTE: There are a few optional configuration values that you can additionally
set in your git configuration to influence the behavior of the bridge. Please
see the notes at:
https://github.com/MichaelMure/git-bug/blob/master/doc/jira_bridge.md
`

const credTypeText = `
JIRA has recently altered it's authentication strategies. Servers deployed
prior to October 1st 2019 must use "SESSION" authentication, whereby the REST
client logs in with an actual username and password, is assigned a session, and
passes the session cookie with each request. JIRA Cloud and servers deployed
after October 1st 2019 must use "TOKEN" authentication. You must create a user
API token and the client will provide this along with your username with each
request.`

func (*Jira) ValidParams() map[string]interface{} {
	return map[string]interface{}{
		"BaseURL":    nil,
		"Login":      nil,
		"CredPrefix": nil,
		"Project":    nil,
		"TokenRaw":   nil,
	}
}

// Configure sets up the bridge configuration
func (j *Jira) Configure(repo *cache.RepoCache, params core.BridgeParams, interactive bool) (core.Configuration, error) {
	var err error

	baseURL := params.BaseURL
	if baseURL == "" {
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the JIRA server URL via the --base-url option.")
		}
		// terminal prompt
		baseURL, err = input.Prompt("JIRA server URL", "URL", input.Required, input.IsURL)
		if err != nil {
			return nil, err
		}
	}

	project := params.Project
	if project == "" {
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the JIRA project key via the --project option.")
		}
		project, err = input.Prompt("JIRA project key", "project", input.Required)
		if err != nil {
			return nil, err
		}
	}

	var login string
	var credType string
	var cred auth.Credential

	switch {
	case params.CredPrefix != "":
		cred, err = auth.LoadWithPrefix(repo, params.CredPrefix)
		if err != nil {
			return nil, err
		}
		l, ok := cred.GetMetadata(auth.MetaKeyLogin)
		if !ok {
			return nil, fmt.Errorf("credential doesn't have a login")
		}
		login = l
	default:
		if params.Login == "" {
			if !interactive {
				return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the login name via the --login option.")
			}
			login, err = input.Prompt("JIRA login", "login", input.Required)
			if err != nil {
				return nil, err
			}
		} else {
			login = params.Login
		}
		// TODO: validate username

		if params.TokenRaw == "" {
			if !interactive {
				return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the access token via the --token option.")
			}
			fmt.Println(credTypeText)
			credTypeInput, err := input.PromptChoice("Authentication mechanism", []string{"SESSION", "TOKEN"})
			if err != nil {
				return nil, err
			}
			credType = []string{"SESSION", "TOKEN"}[credTypeInput]
			cred, err = promptCredOptions(repo, login, baseURL)
			if err != nil {
				return nil, err
			}
		} else {
			credType = "TOKEN"
		}
	}

	conf := make(core.Configuration)
	conf[core.ConfigKeyTarget] = target
	conf[confKeyBaseUrl] = baseURL
	conf[confKeyProject] = project
	conf[confKeyCredentialType] = credType
	conf[confKeyDefaultLogin] = login

	err = j.ValidateConfig(conf)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Attempting to login with credentials...\n")
	client, err := buildClient(context.TODO(), baseURL, credType, cred)
	if err != nil {
		return nil, err
	}

	// verify access to the project with credentials
	fmt.Printf("Checking project ...\n")
	_, err = client.GetProject(project)
	if err != nil {
		return nil, fmt.Errorf(
			"Project %s doesn't exist on %s, or authentication credentials for (%s)"+
				" are invalid",
			project, baseURL, login)
	}

	// don't forget to store the now known valid token
	if !auth.IdExist(repo, cred.ID()) {
		err = auth.Store(repo, cred)
		if err != nil {
			return nil, err
		}
	}

	err = core.FinishConfig(repo, metaKeyJiraLogin, login)
	if err != nil {
		return nil, err
	}

	fmt.Print(moreConfigText)
	return conf, nil
}

// ValidateConfig returns true if all required keys are present
func (*Jira) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}
	if _, ok := conf[confKeyBaseUrl]; !ok {
		return fmt.Errorf("missing %s key", confKeyBaseUrl)
	}
	if _, ok := conf[confKeyProject]; !ok {
		return fmt.Errorf("missing %s key", confKeyProject)
	}
	if _, ok := conf[confKeyCredentialType]; !ok {
		return fmt.Errorf("missing %s key", confKeyCredentialType)
	}
	if _, ok := conf[confKeyDefaultLogin]; !ok {
		return fmt.Errorf("missing %s key", confKeyDefaultLogin)
	}

	return nil
}

func promptCredOptions(repo repository.RepoKeyring, login, baseUrl string) (auth.Credential, error) {
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, login),
		auth.WithMeta(auth.MetaKeyBaseURL, baseUrl),
	)
	if err != nil {
		return nil, err
	}

	cred, index, err := input.PromptCredential(target, "password", creds, []string{
		"enter my password",
		"ask my password each time",
	})
	switch {
	case err != nil:
		return nil, err
	case cred != nil:
		return cred, nil
	case index == 0:
		password, err := input.PromptPassword("Password", "password", input.Required)
		if err != nil {
			return nil, err
		}
		lp := auth.NewLoginPassword(target, login, password)
		lp.SetMetadata(auth.MetaKeyLogin, login)
		lp.SetMetadata(auth.MetaKeyBaseURL, baseUrl)
		return lp, nil
	case index == 1:
		l := auth.NewLogin(target, login)
		l.SetMetadata(auth.MetaKeyLogin, login)
		l.SetMetadata(auth.MetaKeyBaseURL, baseUrl)
		return l, nil
	default:
		panic("missed case")
	}
}
