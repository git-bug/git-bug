package gitea

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	gitea "gitea.dev/sdk"
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/input"
	"github.com/git-bug/git-bug/repository"
)

var (
	ErrBadProjectURL = errors.New("bad project url")
)

func (g *Gitea) ValidParams() map[string]interface{} {
	return map[string]interface{}{
		"URL":        nil,
		"Login":      nil,
		"CredPrefix": nil,
		"TokenRaw":   nil,
	}
}

func (g *Gitea) Configure(repo *cache.RepoCache, params core.BridgeParams, interactive bool) (core.Configuration, error) {
	var err error
	var baseURL, owner, project string

	// get project url
	switch {
	case params.URL != "":
		baseURL, owner, project, err = splitURL(params.URL)
		if err != nil {
			return nil, err
		}
	default:
		// terminal prompt
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the gitea project URL via the --url option.")
		}
		baseURL, owner, project, err = promptURL(repo)
		if err != nil {
			return nil, errors.Wrap(err, "url prompt")
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
	case params.TokenRaw != "":
		token := auth.NewToken(target, params.TokenRaw)
		login, err = getLoginFromToken(baseURL, token)
		if err != nil {
			return nil, err
		}
		token.SetMetadata(auth.MetaKeyLogin, login)
		token.SetMetadata(auth.MetaKeyBaseURL, baseURL)
		cred = token
	default:
		if params.Login == "" {
			if !interactive {
				return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the login name via the --login option.")
			}
			login, err = input.Prompt("Gitea login", "login", input.Required)
			if err != nil {
				return nil, err
			}
		} else {
			login = params.Login
		}
		// validate username
		_, err = validateUsername(baseURL, login)
		if err != nil {
			return nil, errors.Wrap(err, "username validation")
		}
		if !interactive {
			return nil, fmt.Errorf("Non-interactive-mode is active. Please specify the access token via the --token option.")
		}
		cred, err = promptTokenOptions(repo, login, baseURL)
		if err != nil {
			return nil, err
		}
	}

	token, ok := cred.(*auth.Token)
	if !ok {
		return nil, fmt.Errorf("the Gitea bridge only handle token credentials")
	}

	// verify access to the repository with token
	_, err = validateProject(baseURL, owner, project, token)
	if err != nil {
		return nil, errors.Wrap(err, "project validation")
	}

	conf := make(core.Configuration)
	conf[core.ConfigKeyTarget] = target
	conf[confKeyBaseURL] = baseURL
	conf[confKeyOwner] = owner
	conf[confKeyProject] = project
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

	return conf, core.FinishConfig(repo, metaKeyGiteaLogin, login)
}

func (g *Gitea) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}
	if _, ok := conf[confKeyBaseURL]; !ok {
		return fmt.Errorf("missing %s key", confKeyBaseURL)
	}
	if _, ok := conf[confKeyOwner]; !ok {
		return fmt.Errorf("missing %s key", confKeyOwner)
	}
	if _, ok := conf[confKeyProject]; !ok {
		return fmt.Errorf("missing %s key", confKeyProject)
	}
	if _, ok := conf[confKeyDefaultLogin]; !ok {
		return fmt.Errorf("missing %s key", confKeyDefaultLogin)
	}

	return nil
}

func promptTokenOptions(repo repository.RepoKeyring, login, baseURL string) (auth.Credential, error) {
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, login),
		auth.WithMeta(auth.MetaKeyBaseURL, baseURL),
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
		return promptToken(baseURL)
	default:
		panic("missed case")
	}
}

func promptToken(baseURL string) (*auth.Token, error) {
	fmt.Printf("You can generate a new token by visiting %s.\n", path.Join(baseURL, "user/settings/applications"))
	fmt.Println()

	re := regexp.MustCompile(`^[a-z0-9]{40}$`)

	var login string

	validator := func(name string, value string) (complaint string, err error) {
		if !re.MatchString(value) {
			return "token has incorrect format", nil
		}
		login, err = getLoginFromToken(baseURL, auth.NewToken(target, value))
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
	token.SetMetadata(auth.MetaKeyBaseURL, baseURL)

	return token, nil
}

func promptURL(repo repository.RepoCommon) (string, string, string, error) {
	validRemotes, err := getRemoteURLs(repo)
	if err != nil {
		return "", "", "", err
	}

	validator := func(name, value string) (string, error) {
		_, _, _, err := splitURL(value)
		if err != nil {
			return err.Error(), nil
		}
		return "", nil
	}

	url, err := input.PromptURLWithRemote("Gitea project URL", "URL", validRemotes, input.Required, input.IsURL, validator)
	if err != nil {
		return "", "", "", err
	}

	return splitURL(url)
}

// repoPageSegments are path segments Gitea and Forgejo put after
// /{owner}/{repo} in web URLs, such as /owner/repo/issues/12.
var repoPageSegments = map[string]bool{
	"issues": true, "pulls": true, "src": true, "commits": true,
	"commit": true, "branches": true, "tags": true, "releases": true,
	"wiki": true, "milestones": true, "labels": true, "projects": true,
	"actions": true, "activity": true, "settings": true, "compare": true,
	"-": true,
}

// scpURL matches scp-like git remotes such as git@gitea.com:owner/repo.git.
var scpURL = regexp.MustCompile(`^[^@/]+@([^:/]+):(.+)$`)

// splitURL splits a Gitea/Forgejo repository URL into the instance base URL
// (with a trailing slash), the owner and the repository name.
//
// It accepts web URLs, including repository pages like /owner/repo/issues/1,
// instances served under a subpath (https://host/gitea/owner/repo), clone URLs
// ending in .git, and git://, ssh:// and scp-like remotes. Remote protocols are
// mapped to https on the same host, since that is where the API lives.
func splitURL(rawUrl string) (baseURL, owner, project string, err error) {
	var scheme, host, urlPath string

	if m := scpURL.FindStringSubmatch(rawUrl); m != nil && !strings.Contains(rawUrl, "://") {
		scheme, host, urlPath = "https", m[1], m[2]
	} else {
		parsed, perr := url.Parse(rawUrl)
		if perr != nil || parsed.Host == "" {
			return "", "", "", ErrBadProjectURL
		}
		switch parsed.Scheme {
		case "http", "https":
			scheme, host = parsed.Scheme, parsed.Host
		case "ssh", "git", "git+ssh", "ssh+git":
			// The SSH/git port says nothing about where the web API is served.
			scheme, host = "https", parsed.Hostname()
		default:
			return "", "", "", ErrBadProjectURL
		}
		urlPath = parsed.Path
	}

	var segments []string
	for _, seg := range strings.Split(urlPath, "/") {
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	// The repository is the pair right before the first repository page
	// segment, or the last two segments when there is none.
	repoEnd := len(segments)
	for i := 2; i < len(segments); i++ {
		if repoPageSegments[segments[i]] {
			repoEnd = i
			break
		}
	}
	if repoEnd < 2 {
		return "", "", "", ErrBadProjectURL
	}

	owner = segments[repoEnd-2]
	project = strings.TrimSuffix(segments[repoEnd-1], ".git")
	if owner == "" || project == "" {
		return "", "", "", ErrBadProjectURL
	}

	base := url.URL{Scheme: scheme, Host: host, Path: "/"}
	if prefix := segments[:repoEnd-2]; len(prefix) > 0 {
		base.Path = "/" + strings.Join(prefix, "/") + "/"
	}
	return base.String(), owner, project, nil
}

func getRemoteURLs(repo repository.RepoCommon) ([]string, error) {
	remotes, err := repo.GetRemotes()
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(remotes))
	for _, url := range remotes {
		urls = append(urls, url)
	}

	sort.Strings(urls)

	return urls, nil
}

func validateUsername(baseURL, username string) (bool, error) {
	/* FIXME: We're directly calling gitea.NewClient here because buildClient
	 * *requires* a token, which we may not have yet. Ideally the token to
	 * buildClient should be optional. */
	client, err := gitea.NewClient(baseURL)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, _, err = client.GetUserInfo(ctx, username)
	if err != nil {
		return false, errors.Wrap(err, "user not found")
	}

	return true, nil
}

func validateProject(baseURL, owner, project string, token *auth.Token) (bool, error) {
	client, err := buildClient(baseURL, token)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, _, err = client.GetRepo(ctx, owner, project)
	if err != nil {
		return false, errors.Wrap(err, "wrong token scope or non-existent project")
	}

	return true, nil
}

func getLoginFromToken(baseURL string, token *auth.Token) (string, error) {
	client, err := buildClient(baseURL, token)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	user, _, err := client.GetMyUserInfo(ctx)
	if err != nil {
		return "", err
	}
	if user.UserName == "" {
		return "", fmt.Errorf("gitea say username is empty")
	}

	return user.UserName, nil
}
