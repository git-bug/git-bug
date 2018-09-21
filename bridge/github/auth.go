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
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

const githubV3Url = "https://api.github.com"

func Configure() (map[string]string, error) {
	fmt.Println("git-bug will generate an access token in your Github profile.")
	fmt.Println("TODO: describe token")
	fmt.Println()

	tokenName, err := promptTokenName()
	if err != nil {
		return nil, err
	}

	fmt.Println()

	username, err := promptUsername()
	if err != nil {
		return nil, err
	}

	fmt.Println()

	password, err := promptPassword()
	if err != nil {
		return nil, err
	}

	fmt.Println()

	// Attempt to authenticate and create a token

	var note string
	if tokenName == "" {
		note = "git-bug"
	} else {
		note = fmt.Sprintf("git-bug - %s", tokenName)
	}

	url := fmt.Sprintf("%s/authorizations", githubV3Url)
	params := struct {
		Scopes      []string `json:"scopes"`
		Note        string   `json:"note"`
		Fingerprint string   `json:"fingerprint"`
	}{
		Scopes:      []string{"repo"},
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
	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return decodeBody(resp.Body)
	}

	// Handle 2FA is needed
	OTPHeader := resp.Header.Get("X-GitHub-OTP")
	if resp.StatusCode == http.StatusUnauthorized && OTPHeader != "" {
		otpCode, err := prompt2FA()
		if err != nil {
			return nil, err
		}

		req.Header.Set("X-GitHub-OTP", otpCode)

		resp2, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		defer resp2.Body.Close()

		if resp2.StatusCode == http.StatusCreated {
			return decodeBody(resp.Body)
		}
	}

	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Error %v: %v\n", resp.StatusCode, string(b))

	return nil, nil
}

func decodeBody(body io.ReadCloser) (map[string]string, error) {
	data, _ := ioutil.ReadAll(body)

	aux := struct {
		Token string `json:"token"`
	}{}

	err := json.Unmarshal(data, &aux)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"token": aux.Token,
	}, nil
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
		fmt.Println("Username:")

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

func promptTokenName() (string, error) {
	fmt.Println("To help distinguish the token, you can optionally provide a description")
	fmt.Println("Token name:")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimRight(line, "\n"), nil
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
		fmt.Println("Password:")

		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
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
		fmt.Println("Two-factor authentication code:")

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
