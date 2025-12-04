package todosrht

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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

// parseTodoURL extracts base URL and tracker name from a full todo.sr.ht URL
// Expected format: https://todo.sr.ht/~owner/tracker-name
func parseTodoURL(fullURL string) (baseURL, trackerName string, err error) {
	if fullURL == "" {
		return "", "", nil
	}

	parsed, err := url.Parse(fullURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL format: %v", err)
	}

	// Extract base URL
	baseURL = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	// Extract tracker name from path
	path := strings.TrimPrefix(parsed.Path, "/")
	if path == "" {
		return "", "", fmt.Errorf("URL does not contain a tracker path")
	}

	// Expected path format: ~owner/tracker-name
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tracker path format, expected ~owner/tracker-name")
	}

	if !strings.HasPrefix(parts[0], "~") {
		return "", "", fmt.Errorf("invalid owner format, expected ~owner")
	}

	trackerName = parts[1] // Use only the tracker-name part

	return baseURL, trackerName, nil
}

func (*TodoSourceHut) ValidParams() map[string]interface{} {
	return map[string]interface{}{
		"URL":        nil,
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

	var baseURL, trackerName string

	// Try to parse URL if provided
	if params.URL != "" {
		baseURL, trackerName, err = parseTodoURL(params.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse todo.sr.ht URL: %v", err)
		}
	}

	// If we couldn't extract from URL, use individual parameters
	if baseURL == "" {
		baseURL = params.BaseURL
		if baseURL == "" {
			if !interactive {
				return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the TODOSRHT server URL via the --base-url option or provide a full URL with --url.")
			}
			// terminal prompt
			baseURL, err = input.Prompt("TODOSRHT server URL", "URL", input.Required, input.IsURL)
			if err != nil {
				return nil, err
			}
		}
	}

	if trackerName == "" {
		trackerName = params.Project // params.Project is used to pass the tracker name
		if trackerName == "" {
			if !interactive {
				return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the TODOSRHT tracker name via the --tracker option or provide a full URL with --url.")
			}
			trackerName, err = input.Prompt("TODOSRHT tracker name", "tracker", input.Required)
			if err != nil {
				return nil, err
			}
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

	// First check if tracker exists (without authentication)
	fmt.Printf("Checking tracker existence ...\n")
	publicClient := NewTodoSClient(context.TODO(), baseURL, "")
	exists, err := publicClient.TrackerExists(context.TODO(), trackerName)
	if err != nil {
		return nil, fmt.Errorf("Failed to check tracker existence: %v", err)
	}
	if !exists {
		return nil, fmt.Errorf("Tracker %s doesn't exist on %s", trackerName, baseURL)
	}

	// Then verify access to the tracker with credentials
	tokenPreview := ""
	if len(tokenCred.Value) > 6 {
		tokenPreview = tokenCred.Value[:6] + "..."
	} else {
		tokenPreview = tokenCred.Value
	}
	fmt.Printf("Verifying authentication credentials (token: %s) ...\n", tokenPreview)
	_, err = client.GetTracker(context.TODO(), trackerName)
	if err != nil {
		return nil, fmt.Errorf(
			"Authentication credentials for (%s) are invalid or insufficient to access tracker %s.\nFor more details, set GIT_BUG_DEBUG=1 and try again",
			login, trackerName)
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
