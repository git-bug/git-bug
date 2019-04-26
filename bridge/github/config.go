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
	"strings"
	"syscall"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/repository"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	githubV3Url = "https://api.github.com"
	keyUser     = "user"
	keyProject  = "project"
	keyToken    = "token"
)

func (*Github) Configure(repo repository.RepoCommon) (core.Configuration, error) {
	conf := make(core.Configuration)

	fmt.Println()
	fmt.Println("git-bug will now generate an access token in your Github profile. Your credential are not stored and are only used to generate the token. The token is stored in the repository git config.")
	fmt.Println()
	fmt.Println("The token will have the following scopes:")
	fmt.Println("  - user:email: to be able to read public-only users email")
	// fmt.Println("The token will have the \"repo\" permission, giving it read/write access to your repositories and issues. There is no narrower scope available, sorry :-|")
	fmt.Println()

	projectUser, projectName, err := promptURL()
	if err != nil {
		return nil, err
	}

	conf[keyUser] = projectUser
	conf[keyProject] = projectName

	username, err := promptUsername()
	if err != nil {
		return nil, err
	}

	password, err := promptPassword()
	if err != nil {
		return nil, err
	}

	// Attempt to authenticate and create a token

	note := fmt.Sprintf("git-bug - %s/%s", projectUser, projectName)

	resp, err := requestToken(note, username, password)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Handle 2FA is needed
	OTPHeader := resp.Header.Get("X-GitHub-OTP")
	if resp.StatusCode == http.StatusUnauthorized && OTPHeader != "" {
		otpCode, err := prompt2FA()
		if err != nil {
			return nil, err
		}

		resp, err = requestTokenWith2FA(note, username, password, otpCode)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
	}

	if resp.StatusCode == http.StatusCreated {
		token, err := decodeBody(resp.Body)
		if err != nil {
			return nil, err
		}
		conf[keyToken] = token
		return conf, nil
	}

	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Error %v: %v\n", resp.StatusCode, string(b))

	return nil, nil
}

func (*Github) ValidateConfig(conf core.Configuration) error {
	if _, ok := conf[keyToken]; !ok {
		return fmt.Errorf("missing %s key", keyToken)
	}

	if _, ok := conf[keyUser]; !ok {
		return fmt.Errorf("missing %s key", keyUser)
	}

	if _, ok := conf[keyProject]; !ok {
		return fmt.Errorf("missing %s key", keyProject)
	}

	return nil
}

func requestToken(note, username, password string) (*http.Response, error) {
	return requestTokenWith2FA(note, username, password, "")
}

func requestTokenWith2FA(note, username, password, otpCode string) (*http.Response, error) {
	url := fmt.Sprintf("%s/authorizations", githubV3Url)
	params := struct {
		Scopes      []string `json:"scopes"`
		Note        string   `json:"note"`
		Fingerprint string   `json:"fingerprint"`
	}{
		// user:email is requested to be able to read public emails
		//     - a private email will stay private, even with this token
		Scopes:      []string{"user:email"},
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

	client := http.Client{}

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

func promptURL() (string, string, error) {
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

		projectUser, projectName, err := splitURL(line)

		if err != nil {
			fmt.Println(err)
			continue
		}

		return projectUser, projectName, nil
	}
}

func splitURL(url string) (string, string, error) {
	re, err := regexp.Compile(`github\.com\/([^\/]*)\/([^\/]*)`)
	if err != nil {
		panic(err)
	}

	res := re.FindStringSubmatch(url)

	if res == nil {
		return "", "", fmt.Errorf("bad github project url")
	}

	return res[1], res[2], nil
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
		if err != nil {
			return "", err
		}

		if len(byte2fa) > 0 {
			return string(byte2fa), nil
		}

		fmt.Println("code is empty")
	}
}
