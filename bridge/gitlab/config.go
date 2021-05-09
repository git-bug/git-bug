package gitlab

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/repository"
)

var (
	ErrBadProjectURL = errors.New("bad project url")
)

func (g *Gitlab) ValidParams() map[string]interface{} {
	return map[string]interface{}{
		"URL":        nil,
		"BaseURL":    nil,
		"Login":      nil,
		"CredPrefix": nil,
		"TokenRaw":   nil,
	}
}

func (g *Gitlab) Configure(repo *cache.RepoCache, params core.BridgeParams, interactive bool) (core.Configuration, error) {
	var err error
	var baseUrl string

	switch {
	case params.BaseURL != "":
		baseUrl = params.BaseURL
	default:
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the gitlab instance URL via the --base-url option.")
		}
		baseUrl, err = input.PromptDefault("Gitlab server URL", "URL", defaultBaseURL, input.Required, input.IsURL)
		if err != nil {
			return nil, errors.Wrap(err, "base url prompt")
		}
	}

	var projectURL string

	// get project url
	switch {
	case params.URL != "":
		projectURL = params.URL
	default:
		// terminal prompt
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the gitlab project URL via the --url option.")
		}
		projectURL, err = promptProjectURL(repo, baseUrl)
		if err != nil {
			return nil, errors.Wrap(err, "url prompt")
		}
	}

	if !strings.HasPrefix(projectURL, params.BaseURL) {
		return nil, fmt.Errorf("base URL (%s) doesn't match the project URL (%s)", params.BaseURL, projectURL)
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
	case params.TokenRaw != "":
		token := auth.NewToken(target, params.TokenRaw)
		login, err = getLoginFromToken(baseUrl, token)
		if err != nil {
			return nil, err
		}
		token.SetMetadata(auth.MetaKeyLogin, login)
		token.SetMetadata(auth.MetaKeyBaseURL, baseUrl)
		cred = token
	default:
		if params.Login == "" {
			if !interactive {
				return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the login name via the --login option.")
			}
			// TODO: validate username
			login, err = input.Prompt("Gitlab login", "login", input.Required)
		} else {
			// TODO: validate username
			login = params.Login
		}
		if err != nil {
			return nil, err
		}
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the access token via the --token option.")
		}
		cred, err = promptTokenOptions(repo, login, baseUrl)
		if err != nil {
			return nil, err
		}
	}

	token, ok := cred.(*auth.Token)
	if !ok {
		return nil, fmt.Errorf("the Gitlab bridge only handle token credentials")
	}

	// validate project url and get its ID
	id, err := validateProjectURL(baseUrl, projectURL, token)
	if err != nil {
		return nil, errors.Wrap(err, "project validation")
	}

	conf := make(core.Configuration)
	conf[core.ConfigKeyTarget] = target
	conf[confKeyProjectID] = strconv.Itoa(id)
	conf[confKeyGitlabBaseUrl] = baseUrl
	conf[confKeyDefaultLogin] = login

	err = g.ValidateConfig(conf)
	if err != nil {
		return nil, err
	}

	// don't forget to store the now known valid token
	if !auth.IdExist(repo, cred.ID()) {
		err = auth.Store(repo, cred)
		if err != nil {
			return nil, err
		}
	}

	return conf, core.FinishConfig(repo, metaKeyGitlabLogin, login)
}

func (g *Gitlab) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}
	if _, ok := conf[confKeyGitlabBaseUrl]; !ok {
		return fmt.Errorf("missing %s key", confKeyGitlabBaseUrl)
	}
	if _, ok := conf[confKeyProjectID]; !ok {
		return fmt.Errorf("missing %s key", confKeyProjectID)
	}
	if _, ok := conf[confKeyDefaultLogin]; !ok {
		return fmt.Errorf("missing %s key", confKeyDefaultLogin)
	}

	return nil
}

func promptTokenOptions(repo repository.RepoKeyring, login, baseUrl string) (auth.Credential, error) {
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, login),
		auth.WithMeta(auth.MetaKeyBaseURL, baseUrl),
	)
	if err != nil {
		return nil, err
	}

	cred, index, err := input.PromptCredential(target, "token", creds, []string{
		"enter my token",
	})
	switch {
	case err != nil:
		return nil, err
	case cred != nil:
		return cred, nil
	case index == 0:
		return promptToken(baseUrl)
	default:
		panic("missed case")
	}
}

func promptToken(baseUrl string) (*auth.Token, error) {
	fmt.Printf("You can generate a new token by visiting %s.\n", path.Join(baseUrl, "profile/personal_access_tokens"))
	fmt.Println("Choose 'Create personal access token' and set the necessary access scope for your repository.")
	fmt.Println()
	fmt.Println("'api' access scope: to be able to make api calls")
	fmt.Println()

	re := regexp.MustCompile(`^[a-zA-Z0-9\-\_]{20}$`)

	var login string

	validator := func(name string, value string) (complaint string, err error) {
		if !re.MatchString(value) {
			return "token has incorrect format", nil
		}
		login, err = getLoginFromToken(baseUrl, auth.NewToken(target, value))
		if err != nil {
			return fmt.Sprintf("token is invalid: %v", err), nil
		}
		return "", nil
	}

	rawToken, err := input.Prompt("Enter token", "token", input.Required, validator)
	if err != nil {
		return nil, err
	}

	token := auth.NewToken(target, rawToken)
	token.SetMetadata(auth.MetaKeyLogin, login)
	token.SetMetadata(auth.MetaKeyBaseURL, baseUrl)

	return token, nil
}

func promptProjectURL(repo repository.RepoCommon, baseUrl string) (string, error) {
	validRemotes, err := getValidGitlabRemoteURLs(repo, baseUrl)
	if err != nil {
		return "", err
	}

	return input.PromptURLWithRemote("Gitlab project URL", "URL", validRemotes, input.Required)
}

func getProjectPath(baseUrl, projectUrl string) (string, error) {
	cleanUrl := strings.TrimSuffix(projectUrl, ".git")
	cleanUrl = strings.Replace(cleanUrl, "git@", "https://", 1)
	objectUrl, err := url.Parse(cleanUrl)
	if err != nil {
		return "", ErrBadProjectURL
	}

	objectBaseUrl, err := url.Parse(baseUrl)
	if err != nil {
		return "", ErrBadProjectURL
	}

	if objectUrl.Hostname() != objectBaseUrl.Hostname() {
		return "", fmt.Errorf("base url and project url hostnames doesn't match")
	}
	return objectUrl.Path[1:], nil
}

func getValidGitlabRemoteURLs(repo repository.RepoCommon, baseUrl string) ([]string, error) {
	remotes, err := repo.GetRemotes()
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(remotes))
	for _, u := range remotes {
		p, err := getProjectPath(baseUrl, u)
		if err != nil {
			continue
		}

		urls = append(urls, fmt.Sprintf("%s/%s", baseUrl, p))
	}

	return urls, nil
}

func validateProjectURL(baseUrl, url string, token *auth.Token) (int, error) {
	projectPath, err := getProjectPath(baseUrl, url)
	if err != nil {
		return 0, err
	}

	client, err := buildClient(baseUrl, token)
	if err != nil {
		return 0, err
	}

	project, _, err := client.Projects.GetProject(projectPath, &gitlab.GetProjectOptions{})
	if err != nil {
		return 0, errors.Wrap(err, "wrong token scope ou non-existent project")
	}

	return project.ID, nil
}

func getLoginFromToken(baseUrl string, token *auth.Token) (string, error) {
	client, err := buildClient(baseUrl, token)
	if err != nil {
		return "", err
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		return "", err
	}
	if user.Username == "" {
		return "", fmt.Errorf("gitlab say username is empty")
	}

	return user.Username, nil
}
