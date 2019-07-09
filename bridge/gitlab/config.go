package gitlab

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	neturl "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/repository"
)

const (
	target      = "gitlab"
	gitlabV4Url = "https://gitlab.com/api/v4"
	keyID       = "id"
	keyTarget   = "target"
	keyToken    = "token"

	defaultTimeout = 60 * time.Second
)

//note to my self: bridge configure --target=gitlab --url=$URL

var (
	ErrBadProjectURL = errors.New("bad project url")
)

func (*Gitlab) Configure(repo repository.RepoCommon, params core.BridgeParams) (core.Configuration, error) {
	if params.Project != "" {
		fmt.Println("warning: --project is ineffective for a gitlab bridge")
	}
	if params.Owner != "" {
		fmt.Println("warning: --owner is ineffective for a gitlab bridge")
	}

	conf := make(core.Configuration)
	var err error
	var url string
	var token string
	var projectID string

	// get project url
	if params.URL != "" {
		url = params.URL

	} else {
		// remote suggestions
		remotes, err := repo.GetRemotes()
		if err != nil {
			return nil, err
		}

		// terminal prompt
		url, err = promptURL(remotes)
		if err != nil {
			return nil, err
		}
	}

	// get user token
	if params.Token != "" {
		token = params.Token
	} else {
		token, err = promptTokenOptions(url)
		if err != nil {
			return nil, err
		}
	}

	var ok bool
	// validate project url and get it ID
	ok, projectID, err = validateProjectURL(url, token)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid project id or wrong token scope")
	}

	conf[keyID] = projectID
	conf[keyToken] = token
	conf[keyTarget] = target

	return conf, nil
}

func (*Gitlab) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[keyTarget]; !ok {
		return fmt.Errorf("missing %s key", keyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}

	if _, ok := conf[keyToken]; !ok {
		return fmt.Errorf("missing %s key", keyToken)
	}

	if _, ok := conf[keyID]; !ok {
		return fmt.Errorf("missing %s key", keyID)
	}

	return nil
}

func requestToken(note, username, password string, scope string) (*http.Response, error) {
	return requestTokenWith2FA(note, username, password, "", scope)
}

//TODO: FIX THIS ONE
func requestTokenWith2FA(note, username, password, otpCode string, scope string) (*http.Response, error) {
	url := fmt.Sprintf("%s/authorizations", gitlabV4Url)
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

	req.SetBasicAuth(username, password)
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

func promptTokenOptions(url string) (string, error) {
	for {
		fmt.Println()
		fmt.Println("[1]: user provided token")
		fmt.Println("[2]: interactive token creation")
		fmt.Print("Select option: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println()
		if err != nil {
			return "", err
		}

		line = strings.TrimRight(line, "\n")

		index, err := strconv.Atoi(line)
		if err != nil || (index != 1 && index != 2) {
			fmt.Println("invalid input")
			continue
		}

		if index == 1 {
			return promptToken()
		}

		return loginAndRequestToken(url)
	}
}

func promptToken() (string, error) {
	fmt.Println("You can generate a new token by visiting https://gitlab.com/settings/tokens.")
	fmt.Println("Choose 'Generate new token' and set the necessary access scope for your repository.")
	fmt.Println()
	fmt.Println("The access scope depend on the type of repository.")
	fmt.Println("Public:")
	fmt.Println("  - 'public_repo': to be able to read public repositories")
	fmt.Println("Private:")
	fmt.Println("  - 'repo'       : to be able to read private repositories")
	fmt.Println()

	re, err := regexp.Compile(`^[a-zA-Z0-9\-]{20}`)
	if err != nil {
		panic("regexp compile:" + err.Error())
	}

	for {
		fmt.Print("Enter token: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		token := strings.TrimRight(line, "\n")
		if re.MatchString(token) {
			return token, nil
		}

		fmt.Println("token is invalid")
	}
}

// TODO: FIX THIS ONE TOO
func loginAndRequestToken(url string) (string, error) {

	// prompt project visibility to know the token scope needed for the repository
	isPublic, err := promptProjectVisibility()
	if err != nil {
		return "", err
	}

	username, err := promptUsername()
	if err != nil {
		return "", err
	}

	password, err := promptPassword()
	if err != nil {
		return "", err
	}

	var scope string
	//TODO: Gitlab scopes
	if isPublic {
		// public_repo is requested to be able to read public repositories
		scope = "public_repo"
	} else {
		// 'repo' is request to be able to read private repositories
		// /!\ token will have read/write rights on every private repository you have access to
		scope = "repo"
	}

	// Attempt to authenticate and create a token

	note := fmt.Sprintf("git-bug - %s/%s", url)

	resp, err := requestToken(note, username, password, scope)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Handle 2FA is needed
	OTPHeader := resp.Header.Get("X-GitHub-OTP")
	if resp.StatusCode == http.StatusUnauthorized && OTPHeader != "" {
		otpCode, err := prompt2FA()
		if err != nil {
			return "", err
		}

		resp, err = requestTokenWith2FA(note, username, password, otpCode, scope)
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

func promptUsername() (string, error) {
	for {
		fmt.Print("username: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimRight(line, "\n")

		ok, err := validateUsername(line)
		if err != nil {
			return "", err
		}
		if ok {
			return line, nil
		}

		fmt.Println("invalid username")
	}
}

func promptURL(remotes map[string]string) (string, error) {
	validRemotes := getValidGitlabRemoteURLs(remotes)
	if len(validRemotes) > 0 {
		for {
			fmt.Println("\nDetected projects:")

			// print valid remote gitlab urls
			for i, remote := range validRemotes {
				fmt.Printf("[%d]: %v\n", i+1, remote)
			}

			fmt.Printf("\n[0]: Another project\n\n")
			fmt.Printf("Select option: ")

			line, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				return "", err
			}

			line = strings.TrimRight(line, "\n")

			index, err := strconv.Atoi(line)
			if err != nil || (index < 0 && index >= len(validRemotes)) {
				fmt.Println("invalid input")
				continue
			}

			// if user want to enter another project url break this loop
			if index == 0 {
				break
			}

			return validRemotes[index-1], nil
		}
	}

	// manually enter gitlab url
	for {
		fmt.Print("Gitlab project URL: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		url := strings.TrimRight(line, "\n")
		if line == "" {
			fmt.Println("URL is empty")
			continue
		}

		return url, nil
	}
}

func splitURL(url string) (string, string, error) {
	cleanUrl := strings.TrimSuffix(url, ".git")
	objectUrl, err := neturl.Parse(cleanUrl)
	if err != nil {
		return "", "", nil
	}

	return fmt.Sprintf("%s%s", objectUrl.Host, objectUrl.Path), objectUrl.Path, nil
}

func getValidGitlabRemoteURLs(remotes map[string]string) []string {
	urls := make([]string, 0, len(remotes))
	for _, u := range remotes {
		url, _, err := splitURL(u)
		if err != nil {
			continue
		}

		urls = append(urls, url)
	}

	return urls
}

func validateUsername(username string) (bool, error) {
	// no need for a token for this action
	client := buildClient("")

	users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &username})
	if err != nil {
		return false, err
	}

	if len(users) == 0 {
		return false, fmt.Errorf("username not found")
	} else if len(users) > 1 {
		return false, fmt.Errorf("found multiple matches")
	}

	return users[0].Username == username, nil
}

func validateProjectURL(url, token string) (bool, string, error) {
	client := buildClient(token)

	_, projectPath, err := splitURL(url)
	if err != nil {
		return false, "", err
	}

	project, _, err := client.Projects.GetProject(projectPath[1:], &gitlab.GetProjectOptions{})
	if err != nil {
		return false, "", err
	}
	projectID := strconv.Itoa(project.ID)

	return true, projectID, nil
}

func promptPassword() (string, error) {
	for {
		fmt.Print("password: ")

		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		// new line for coherent formatting, ReadPassword clip the normal new line
		// entered by the user
		fmt.Println()

		if err != nil {
			return "", err
		}

		if len(bytePassword) > 0 {
			return string(bytePassword), nil
		}

		fmt.Println("password is empty")
	}
}

func prompt2FA() (string, error) {
	for {
		fmt.Print("two-factor authentication code: ")

		byte2fa, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", err
		}

		if len(byte2fa) > 0 {
			return string(byte2fa), nil
		}

		fmt.Println("code is empty")
	}
}

func promptProjectVisibility() (bool, error) {
	for {
		fmt.Println("[1]: public")
		fmt.Println("[2]: private")
		fmt.Print("repository visibility: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println()
		if err != nil {
			return false, err
		}

		line = strings.TrimRight(line, "\n")

		index, err := strconv.Atoi(line)
		if err != nil || (index != 0 && index != 1) {
			fmt.Println("invalid input")
			continue
		}

		// return true for public repositories, false for private
		return index == 0, nil
	}
}
