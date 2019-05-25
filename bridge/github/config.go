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
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/repository"
)

const (
	githubV3Url = "https://api.github.com"
	keyOwner    = "owner"
	keyProject  = "project"
	keyToken    = "token"

	defaultTimeout = 5 * time.Second
)

var (
	rxGithubURL = regexp.MustCompile(`github\.com[\/:]([^\/]*[a-z]+)\/([^\/]*[a-z]+)`)
)

func (*Github) Configure(repo repository.RepoCommon, params core.BridgeParams) (core.Configuration, error) {
	conf := make(core.Configuration)
	var err error
	var token string
	var owner string
	var project string

	// getting owner and project name
	if params.Owner != "" && params.Project != "" {
		// first try to use params if both or project and owner are provided
		owner = params.Owner
		project = params.Project

	} else if params.URL != "" {
		// try to parse params URL and extract owner and project
		_, owner, project, err = splitURL(params.URL)
		if err != nil {
			return nil, err
		}

	} else {
		// remote suggestions
		remotes, err := repo.GetRemotes()
		if err != nil {
			return nil, err
		}

		// terminal prompt
		owner, project, err = promptURL(remotes)
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

	// try to get token from params if provided, else use terminal prompt to either
	// enter a token or login and generate a new one
	if params.Token != "" {
		token = params.Token

	} else {
		token, err = promptTokenOptions(owner, project)
		if err != nil {
			return nil, err
		}
	}

	// verify access to the repository with token
	ok, err = validateProject(owner, project, token)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("project doesn't exist or authentication token has a wrong scope")
	}

	conf[keyToken] = token
	conf[keyOwner] = owner
	conf[keyProject] = project

	return conf, nil
}

func (*Github) ValidateConfig(conf core.Configuration) error {
	if _, ok := conf[keyToken]; !ok {
		return fmt.Errorf("missing %s key", keyToken)
	}

	if _, ok := conf[keyOwner]; !ok {
		return fmt.Errorf("missing %s key", keyOwner)
	}

	if _, ok := conf[keyProject]; !ok {
		return fmt.Errorf("missing %s key", keyProject)
	}

	return nil
}

func requestToken(note, username, password string, scope string) (*http.Response, error) {
	return requestTokenWith2FA(note, username, password, "", scope)
}

func requestTokenWith2FA(note, username, password, otpCode string, scope string) (*http.Response, error) {
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

func promptTokenOptions(owner, project string) (string, error) {
	for {
		fmt.Println()
		fmt.Println("[0]: i have my own token")
		fmt.Println("[1]: login and generate token")
		fmt.Print("Select option: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println()
		if err != nil {
			return "", err
		}

		line = strings.TrimRight(line, "\n")

		index, err := strconv.Atoi(line)
		if err != nil || (index != 0 && index != 1) {
			fmt.Println("invalid input")
			continue
		}

		if index == 0 {
			return promptToken()
		}

		return loginAndRequestToken(owner, project)
	}
}

func promptToken() (string, error) {
	for {
		fmt.Print("Enter token: ")

		byteToken, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()

		if err != nil {
			return "", err
		}

		if len(byteToken) > 0 {
			return string(byteToken), nil
		}

		fmt.Println("token is empty")
	}
}

func loginAndRequestToken(owner, project string) (string, error) {
	fmt.Println("git-bug will now generate an access token in your Github profile. Your credential are not stored and are only used to generate the token. The token is stored in the repository git config.")
	fmt.Println()
	fmt.Println("Depending on your configuration the token will have one of the following scopes:")
	fmt.Println("  - 'user:email': to be able to read public-only users email")
	fmt.Println("  - 'repo'      : to be able to read private repositories")
	// fmt.Println("The token will have the \"repo\" permission, giving it read/write access to your repositories and issues. There is no narrower scope available, sorry :-|")
	fmt.Println()

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
	if isPublic {
		// user:email is requested to be able to read public emails
		//     - a private email will stay private, even with this token
		scope = "user:email"
	} else {
		// 'repo' is request to be able to read private repositories
		// /!\ token will have read/write rights on every private repository you have access to
		scope = "repo"
	}

	// Attempt to authenticate and create a token

	note := fmt.Sprintf("git-bug - %s/%s", owner, project)

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
		token, err := decodeBody(resp.Body)
		if err != nil {
			return "", err
		}

		return token, nil
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

func promptURL(remotes map[string]string) (string, string, error) {
	validRemotes := valideGithubURLRemotes(remotes)
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

			// get owner and project with index
			_, owner, project, _ := splitURL(validRemotes[index-1])
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

		line = strings.TrimRight(line, "\n")
		if line == "" {
			fmt.Println("URL is empty")
			continue
		}

		// get owner and project from url
		_, owner, project, err := splitURL(line)
		if err != nil {
			fmt.Println(err)
			continue
		}

		return owner, project, nil
	}
}

func splitURL(url string) (string, string, string, error) {
	res := rxGithubURL.FindStringSubmatch(url)
	if res == nil {
		return "", "", "", fmt.Errorf("bad github project url")
	}

	return res[0], res[1], res[2], nil
}

func valideGithubURLRemotes(remotes map[string]string) []string {
	urls := make([]string, 0, len(remotes))
	for _, url := range remotes {
		// split url can work again with shortURL
		shortURL, _, _, err := splitURL(url)
		if err == nil {
			urls = append(urls, shortURL)
		}
	}

	return urls
}

func validateUsername(username string) (bool, error) {
	url := fmt.Sprintf("%s/users/%s", githubV3Url, username)

	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}

	err = resp.Body.Close()
	if err != nil {
		return false, err
	}

	return resp.StatusCode == http.StatusOK, nil
}

func validateProject(owner, project, token string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", githubV3Url, owner, project)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

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

		if len(byte2fa) != 6 {
			fmt.Println("invalid 2FA code size")
			continue
		}

		str2fa := string(byte2fa)
		_, err = strconv.Atoi(str2fa)
		if err != nil {
			fmt.Println("2fa code must be digits only")
			continue
		}

		return str2fa, nil
	}
}

func promptProjectVisibility() (bool, error) {
	for {
		fmt.Println("[0]: public")
		fmt.Println("[1]: private")
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
