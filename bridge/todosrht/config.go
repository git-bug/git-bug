package todosrht

import (
	"context"
	"fmt"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/input"
	"github.com/git-bug/git-bug/repository"
)

const moreConfigText = `
NOTE: There are a few optional configuration values that you can additionally
set in your git configuration to influence the behavior of the bridge. Please
see the notes at:
https://github.com/git-bug/git-bug/blob/master/doc/todosrht_bridge.md
`

func (*TodoSourceHut) ValidParams() map[string]interface{} {
	return map[string]interface{}{
		"BaseURL":    nil,
		"Login":      nil,
		"CredPrefix": nil,
		"Tracker":    nil,
		"TokenRaw":   nil,
	}
}

// Configure sets up the bridge configuration
func (j *TodoSourceHut) Configure(repo *cache.RepoCache, params core.BridgeParams, interactive bool) (core.Configuration, error) {
	var err error

	baseURL := params.BaseURL
	if baseURL == "" {
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the TODOSRHT server URL via the --base-url option.")
		}
		// terminal prompt
		baseURL, err = input.Prompt("TODOSRHT server URL", "URL", input.Required, input.IsURL)
		if err != nil {
			return nil, err
		}
	}

	trackerName := params.Project // params.Project is used to pass the tracker name
	if trackerName == "" {
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the TODOSRHT tracker name via the --tracker option.")
		}
		trackerName, err = input.Prompt("TODOSRHT tracker name", "tracker", input.Required)
		if err != nil {
			return nil, err
		}
	}

	var login string
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
			login, err = input.Prompt("TODOSRHT login", "login", input.Required)
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
			cred, err = promptToken(repo, login, baseURL)
			if err != nil {
				return nil, err
			}
		} else {
			token := auth.NewToken(target, params.TokenRaw)
			token.SetMetadata(auth.MetaKeyLogin, login)
			token.SetMetadata(auth.MetaKeyBaseURL, baseURL)
			cred = token
		}
	}

	tokenCred, ok := cred.(*auth.Token)
	if !ok {
		return nil, fmt.Errorf("the SourceHut bridge only handles token credentials")
	}

	conf := make(core.Configuration)
	conf[core.ConfigKeyTarget] = target
	conf[confKeyBaseUrl] = baseURL
	conf[confKeyTrackerName] = trackerName
	conf[confKeyDefaultLogin] = login

	err = j.ValidateConfig(conf)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Attempting to verify credentials...\n")
	client := NewTodoSClient(context.TODO(), baseURL, tokenCred.Value)

	// verify access to the tracker with credentials
	fmt.Printf("Checking tracker ...\n")
	_, err = client.GetTracker(context.TODO(), trackerName)
	if err != nil {
		return nil, fmt.Errorf(
			"Tracker %s doesn't exist on %s, or authentication credentials for (%s) are invalid",
			trackerName, baseURL, login)
	}

	// don't forget to store the now known valid token
	if !auth.IdExist(repo, cred.ID()) {
		err = auth.Store(repo, cred)
		if err != nil {
			return nil, err
		}
	}

	err = core.FinishConfig(repo, metaKeyTodoSourceHutLogin, login)
	if err != nil {
		return nil, err
	}

	fmt.Print(moreConfigText)
	return conf, nil
}

// ValidateConfig returns true if all required keys are present
func (*TodoSourceHut) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}
	if _, ok := conf[confKeyBaseUrl]; !ok {
		return fmt.Errorf("missing %s key", confKeyBaseUrl)
	}
	if _, ok := conf[confKeyTrackerName]; !ok {
		return fmt.Errorf("missing %s key", confKeyTrackerName)
	}
	if _, ok := conf[confKeyDefaultLogin]; !ok {
		return fmt.Errorf("missing %s key", confKeyDefaultLogin)
	}

	return nil
}

func promptToken(repo repository.RepoKeyring, login, baseURL string) (auth.Credential, error) {
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, login),
		auth.WithMeta(auth.MetaKeyBaseURL, baseURL),
	)
	if err != nil {
		return nil, err
	}

	cred, index, err := input.PromptCredential(
		target,
		"token",
		creds,
		[]string{"enter my token"},
	)
	switch {
	case err != nil:
		return nil, err
	case cred != nil:
		return cred, nil
	case index == 0:
		tokenValue, err := input.PromptPassword("Token", "token", input.Required)
		if err != nil {
			return nil, err
		}
		t := auth.NewToken(target, tokenValue)
		t.SetMetadata(auth.MetaKeyLogin, login)
		t.SetMetadata(auth.MetaKeyBaseURL, baseURL)
		return t, nil
	default:
		panic("missed case")
	}
}
