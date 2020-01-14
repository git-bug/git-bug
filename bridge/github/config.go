package github

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	text "github.com/MichaelMure/go-term-text"
	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/colors"
)

const (
	target      = "github"
	githubV3Url = "https://api.github.com"
	keyOwner    = "owner"
	keyProject  = "project"

	defaultTimeout = 60 * time.Second
)

var (
	ErrBadProjectURL = errors.New("bad project url")
)

func (g *Github) Configure(repo *cache.RepoCache, params core.BridgeParams) (core.Configuration, error) {
	if params.BaseURL != "" {
		fmt.Println("warning: --base-url is ineffective for a Github bridge")
	}

	conf := make(core.Configuration)
	var err error
	var owner string
	var project string

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

	// validate project owner
	ok, err := validateUsername(owner)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid parameter owner: %v", owner)
	}

	login := params.Login
	if login == "" {
		validator := func(name string, value string) (string, error) {
			ok, err := validateUsername(value)
			if err != nil {
				return "", err
			}
			if !ok {
				return "invalid login", nil
			}
			return "", nil
		}

		login, err = input.Prompt("Github login", "login", input.Required, validator)
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
	case params.TokenRaw != "":
		cred = auth.NewToken(params.TokenRaw, target)
		cred.Metadata()[auth.MetaKeyLogin] = login
	default:
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

	conf[core.ConfigKeyTarget] = target
	conf[keyOwner] = owner
	conf[keyProject] = project

	err = g.ValidateConfig(conf)
	if err != nil {
		return nil, err
	}

	// TODO
	func(login string) error {
		// if no user exist with the given login
		_, err := repo.ResolveIdentityLogin(login)
		if err != nil && err != identity.ErrIdentityNotExist {
			return err
		}

		// tag the default user with the github login, if any
		// if not,
		user, err := repo.GetUserIdentity()
		if err == identity.ErrNoIdentitySet {
			return nil
		}
		if err != nil {
			return err
		}

		repo.GetUserIdentity()

	}(login)

	// Todo: if no user exist with the given login
	// - tag the default user with the github login
	// - add a command to manually tag a user ?

	// don't forget to store the now known valid token
	if !auth.IdExist(repo, cred.ID()) {
		err = auth.Store(repo, cred)
		if err != nil {
			return nil, err
		}
	}

	return conf, nil
}

func (*Github) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}

	if _, ok := conf[keyOwner]; !ok {
		return fmt.Errorf("missing %s key", keyOwner)
	}

	if _, ok := conf[keyProject]; !ok {
		return fmt.Errorf("missing %s key", keyProject)
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

func promptTokenOptions(repo repository.RepoConfig, login, owner, project string) (auth.Credential, error) {
	for {
		creds, err := auth.List(repo, auth.WithTarget(target), auth.WithMeta(auth.MetaKeyLogin, login))
		if err != nil {
			return nil, err
		}

		fmt.Println()
		fmt.Println("[1]: enter my token")
		fmt.Println("[2]: interactive token creation")

		if len(creds) > 0 {
			sort.Sort(auth.ById(creds))

			fmt.Println()
			fmt.Println("Existing tokens for Github:")
			for i, cred := range creds {
				token := cred.(*auth.Token)
				fmt.Printf("[%d]: %s => %s (login: %s, %s)\n",
					i+3,
					colors.Cyan(token.ID().Human()),
					colors.Red(text.TruncateMax(token.Value, 10)),
					token.Metadata()[auth.MetaKeyLogin],
					token.CreateTime().Format(time.RFC822),
				)
			}
		}

		fmt.Println()
		fmt.Print("Select option: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println()
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		index, err := strconv.Atoi(line)
		if err != nil || index < 1 || index > len(creds)+2 {
			fmt.Println("invalid input")
			continue
		}

		switch index {
		case 1:
			value, err := promptToken()
			if err != nil {
				return nil, err
			}
			token := auth.NewToken(value, target)
			token.Metadata()[auth.MetaKeyLogin] = login
			return token, nil
		case 2:
			value, err := loginAndRequestToken(login, owner, project)
			if err != nil {
				return nil, err
			}
			token := auth.NewToken(value, target)
			token.Metadata()[auth.MetaKeyLogin] = login
			return token, nil
		default:
			return creds[index-3], nil
		}
	}
}

func promptToken() (string, error) {
	fmt.Println("You can generate a new token by visiting https://github.com/settings/tokens.")
	fmt.Println("Choose 'Generate new token' and set the necessary access scope for your repository.")
	fmt.Println()
	fmt.Println("The access scope depend on the type of repository.")
	fmt.Println("Public:")
	fmt.Println("  - 'public_repo': to be able to read public repositories")
	fmt.Println("Private:")
	fmt.Println("  - 'repo'       : to be able to read private repositories")
	fmt.Println()

	re, err := regexp.Compile(`^[a-zA-Z0-9]{40}`)
	if err != nil {
		panic("regexp compile:" + err.Error())
	}

	validator := func(name string, value string) (complaint string, err error) {
		if re.MatchString(value) {
			return "", nil
		}
		return "token has incorrect format", nil
	}

	return input.Prompt("Enter token", "token", input.Required, validator)
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
	i, err := input.PromptChoice("repository visibility", []string{"public", "private"})
	if err != nil {
		return "", err
	}
	isPublic := i == 0

	password, err := input.PromptPassword("Password", "password", input.Required)
	if err != nil {
		return "", err
	}

	var scope string
	if isPublic {
		// public_repo is requested to be able to read public repositories
		scope = "public_repo"
	} else {
		// 'repo' is request to be able to read private repositories
		// /!\ token will have read/write rights on every private repository you have access to
		scope = "repo"
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
	// remote suggestions
	remotes, err := repo.GetRemotes()
	if err != nil {
		return "", "", err
	}

	validRemotes := getValidGithubRemoteURLs(remotes)
	if len(validRemotes) > 0 {
		for {
			fmt.Println("\nDetected projects:")

			// print valid remote github urls
			for i, remote := range validRemotes {
				fmt.Printf("[%d]: %v\n", i+1, remote)
			}

			fmt.Printf("\n[0]: Another project\n\n")
			fmt.Printf("Select option: ")

			line, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				return "", "", err
			}

			line = strings.TrimSpace(line)

			index, err := strconv.Atoi(line)
			if err != nil || index < 0 || index > len(validRemotes) {
				fmt.Println("invalid input")
				continue
			}

			// if user want to enter another project url break this loop
			if index == 0 {
				break
			}

			// get owner and project with index
			owner, project, _ := splitURL(validRemotes[index-1])
			return owner, project, nil
		}
	}

	// manually enter github url
	for {
		fmt.Print("Github project URL: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", "", err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Println("URL is empty")
			continue
		}

		// get owner and project from url
		owner, project, err := splitURL(line)
		if err != nil {
			fmt.Println(err)
			continue
		}

		return owner, project, nil
	}
}

// splitURL extract the owner and project from a github repository URL. It will remove the
// '.git' extension from the URL before parsing it.
// Note that Github removes the '.git' extension from projects names at their creation
func splitURL(url string) (owner string, project string, err error) {
	cleanURL := strings.TrimSuffix(url, ".git")

	re, err := regexp.Compile(`github\.com[/:]([a-zA-Z0-9\-_]+)/([a-zA-Z0-9\-_.]+)`)
	if err != nil {
		panic("regexp compile:" + err.Error())
	}

	res := re.FindStringSubmatch(cleanURL)
	if res == nil {
		return "", "", ErrBadProjectURL
	}

	owner = res[1]
	project = res[2]
	return
}

func getValidGithubRemoteURLs(remotes map[string]string) []string {
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

	return urls
}

func validateUsername(username string) (bool, error) {
	url := fmt.Sprintf("%s/users/%s", githubV3Url, username)

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}

	err = resp.Body.Close()
	if err != nil {
		return false, err
	}

	return resp.StatusCode == http.StatusOK, nil
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
