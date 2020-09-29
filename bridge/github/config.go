package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/repository"
)

var (
	ErrBadProjectURL = errors.New("bad project url")
)

func (g *Github) ValidParams() map[string]interface{} {
	return map[string]interface{}{
		"URL":        nil,
		"Login":      nil,
		"CredPrefix": nil,
		"TokenRaw":   nil,
		"Owner":      nil,
		"Project":    nil,
	}
}

func (g *Github) Configure(repo *cache.RepoCache, params core.BridgeParams) (core.Configuration, error) {
	var err error
	var owner string
	var project string
	var ok bool

	// getting owner and project name
	switch {
	case params.Owner != "" && params.Project != "":
		// first try to use params if both or project and owner are provided
		owner = params.Owner
		project = params.Project
	case params.URL != "":
		// try to parse params URL and extract owner and project
		owner, project, err = splitURL(params.URL)
		if err != nil {
			return nil, err
		}
	default:
		// terminal prompt
		owner, project, err = promptURL(repo)
		if err != nil {
			return nil, err
		}
	}

	// validate project owner and override with the correct case
	ok, owner, err = validateUsername(owner)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid parameter owner: %v", owner)
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
		login, err = getLoginFromToken(token)
		if err != nil {
			return nil, err
		}
		token.SetMetadata(auth.MetaKeyLogin, login)
		cred = token
	default:
		if params.Login == "" {
			login, err = promptLogin()
		} else {
			// validate login and override with the correct case
			ok, login, err = validateUsername(params.Login)
			if !ok {
				return nil, fmt.Errorf("invalid parameter login: %v", params.Login)
			}
		}
		if err != nil {
			return nil, err
		}
		cred, err = promptTokenOptions(repo, login, owner, project)
		if err != nil {
			return nil, err
		}
	}

	token, ok := cred.(*auth.Token)
	if !ok {
		return nil, fmt.Errorf("the Github bridge only handle token credentials")
	}

	// verify access to the repository with token
	ok, err = validateProject(owner, project, token)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("project doesn't exist or authentication token has an incorrect scope")
	}

	conf := make(core.Configuration)
	conf[core.ConfigKeyTarget] = target
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

	return conf, core.FinishConfig(repo, metaKeyGithubLogin, login)
}

func (*Github) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
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

func requestToken(note, login, password string, scope string) (*http.Response, error) {
	return requestTokenWith2FA(note, login, password, "", scope)
}

func requestTokenWith2FA(note, login, password, otpCode string, scope string) (*http.Response, error) {
	url := fmt.Sprintf("%s/authorizations", githubV3Url)
	params := struct {
		Scopes      []string `json:"scopes"`
		Note        string   `json:"note"`
		Fingerprint string   `json:"fingerprint"`
	}{
		Scopes:      []string{scope},
		Note:        note,
		Fingerprint: randomFingerprint(),
	}

	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(login, password)
	req.Header.Set("Content-Type", "application/json")

	if otpCode != "" {
		req.Header.Set("X-GitHub-OTP", otpCode)
	}

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	return client.Do(req)
}

func decodeBody(body io.ReadCloser) (string, error) {
	data, _ := ioutil.ReadAll(body)

	aux := struct {
		Token string `json:"token"`
	}{}

	err := json.Unmarshal(data, &aux)
	if err != nil {
		return "", err
	}

	if aux.Token == "" {
		return "", fmt.Errorf("no token found in response: %s", string(data))
	}

	return aux.Token, nil
}

func randomFingerprint() string {
	// Doesn't have to be crypto secure, it's just to avoid token collision
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 32)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func promptTokenOptions(repo repository.RepoKeyring, login, owner, project string) (auth.Credential, error) {
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, login),
	)
	if err != nil {
		return nil, err
	}

	cred, index, err := input.PromptCredential(target, "token", creds, []string{
		"enter my token",
		"interactive token creation",
	})
	switch {
	case err != nil:
		return nil, err
	case cred != nil:
		return cred, nil
	case index == 0:
		return promptToken()
	case index == 1:
		value, err := loginAndRequestToken(login, owner, project)
		if err != nil {
			return nil, err
		}
		token := auth.NewToken(target, value)
		token.SetMetadata(auth.MetaKeyLogin, login)
		return token, nil
	default:
		panic("missed case")
	}
}

func promptToken() (*auth.Token, error) {
	fmt.Println("You can generate a new token by visiting https://github.com/settings/tokens.")
	fmt.Println("Choose 'Generate new token' and set the necessary access scope for your repository.")
	fmt.Println()
	fmt.Println("The access scope depend on the type of repository.")
	fmt.Println("Public:")
	fmt.Println("  - 'public_repo': to be able to read public repositories")
	fmt.Println("Private:")
	fmt.Println("  - 'repo'       : to be able to read private repositories")
	fmt.Println()

	re := regexp.MustCompile(`^[a-zA-Z0-9]{40}$`)

	var login string

	validator := func(name string, value string) (complaint string, err error) {
		if !re.MatchString(value) {
			return "token has incorrect format", nil
		}
		login, err = getLoginFromToken(auth.NewToken(target, value))
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

	return token, nil
}

func loginAndRequestToken(login, owner, project string) (string, error) {
	fmt.Println("git-bug will now generate an access token in your Github profile. Your credential are not stored and are only used to generate the token. The token is stored in the global git config.")
	fmt.Println()
	fmt.Println("The access scope depend on the type of repository.")
	fmt.Println("Public:")
	fmt.Println("  - 'public_repo': to be able to read public repositories")
	fmt.Println("Private:")
	fmt.Println("  - 'repo'       : to be able to read private repositories")
	fmt.Println()

	// prompt project visibility to know the token scope needed for the repository
	index, err := input.PromptChoice("repository visibility", []string{"public", "private"})
	if err != nil {
		return "", err
	}
	scope := []string{"public_repo", "repo"}[index]

	password, err := input.PromptPassword("Password", "password", input.Required)
	if err != nil {
		return "", err
	}

	// Attempt to authenticate and create a token
	note := fmt.Sprintf("git-bug - %s/%s", owner, project)
	resp, err := requestToken(note, login, password, scope)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Handle 2FA is needed
	OTPHeader := resp.Header.Get("X-GitHub-OTP")
	if resp.StatusCode == http.StatusUnauthorized && OTPHeader != "" {
		otpCode, err := input.PromptPassword("Two-factor authentication code", "code", input.Required)
		if err != nil {
			return "", err
		}

		resp, err = requestTokenWith2FA(note, login, password, otpCode, scope)
		if err != nil {
			return "", err
		}

		defer resp.Body.Close()
	}

	if resp.StatusCode == http.StatusCreated {
		return decodeBody(resp.Body)
	}

	b, _ := ioutil.ReadAll(resp.Body)
	return "", fmt.Errorf("error creating token %v: %v", resp.StatusCode, string(b))
}

func promptURL(repo repository.RepoCommon) (string, string, error) {
	validRemotes, err := getValidGithubRemoteURLs(repo)
	if err != nil {
		return "", "", err
	}

	validator := func(name, value string) (string, error) {
		_, _, err := splitURL(value)
		if err != nil {
			return err.Error(), nil
		}
		return "", nil
	}

	url, err := input.PromptURLWithRemote("Github project URL", "URL", validRemotes, input.Required, validator)
	if err != nil {
		return "", "", err
	}

	return splitURL(url)
}

// splitURL extract the owner and project from a github repository URL. It will remove the
// '.git' extension from the URL before parsing it.
// Note that Github removes the '.git' extension from projects names at their creation
func splitURL(url string) (owner string, project string, err error) {
	cleanURL := strings.TrimSuffix(url, ".git")

	re := regexp.MustCompile(`github\.com[/:]([a-zA-Z0-9\-_]+)/([a-zA-Z0-9\-_.]+)`)

	res := re.FindStringSubmatch(cleanURL)
	if res == nil {
		return "", "", ErrBadProjectURL
	}

	owner = res[1]
	project = res[2]
	return
}

func getValidGithubRemoteURLs(repo repository.RepoCommon) ([]string, error) {
	remotes, err := repo.GetRemotes()
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(remotes))
	for _, url := range remotes {
		// split url can work again with shortURL
		owner, project, err := splitURL(url)
		if err == nil {
			shortURL := fmt.Sprintf("%s/%s/%s", "github.com", owner, project)
			urls = append(urls, shortURL)
		}
	}

	sort.Strings(urls)

	return urls, nil
}

func promptLogin() (string, error) {
	var login string

	validator := func(_ string, value string) (string, error) {
		ok, fixed, err := validateUsername(value)
		if err != nil {
			return "", err
		}
		if !ok {
			return "invalid login", nil
		}
		login = fixed
		return "", nil
	}

	_, err := input.Prompt("Github login", "login", input.Required, validator)
	if err != nil {
		return "", err
	}

	return login, nil
}

func validateUsername(username string) (bool, string, error) {
	url := fmt.Sprintf("%s/users/%s", githubV3Url, username)

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false, "", err
	}

	if resp.StatusCode != http.StatusOK {
		return false, "", nil
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}

	err = resp.Body.Close()
	if err != nil {
		return false, "", err
	}

	var decoded struct {
		Login string `json:"login"`
	}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		return false, "", err
	}

	if decoded.Login == "" {
		return false, "", fmt.Errorf("validateUsername: missing login in the response")
	}

	return true, decoded.Login, nil
}

func validateProject(owner, project string, token *auth.Token) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", githubV3Url, owner, project)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token.Value))

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	err = resp.Body.Close()
	if err != nil {
		return false, err
	}

	return resp.StatusCode == http.StatusOK, nil
}

func getLoginFromToken(token *auth.Token) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	client := buildClient(token)

	var q loginQuery

	err := client.Query(ctx, &q, nil)
	if err != nil {
		return "", err
	}
	if q.Viewer.Login == "" {
		return "", fmt.Errorf("github say username is empty")
	}

	return q.Viewer.Login, nil
}
