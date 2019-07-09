package gitlab

import (
	"bufio"
	"fmt"
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
	keyID       = "project-id"
	keyTarget   = "target"
	keyToken    = "token"

	defaultTimeout = 60 * time.Second
)

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
	ok, id, err := validateProjectURL(url, token)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid project id or wrong token scope")
	}

	conf[keyID] = strconv.Itoa(id)
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

func requestToken(client *gitlab.Client, userID int, name string, scopes ...string) (string, error) {
	impToken, _, err := client.Users.CreateImpersonationToken(
		userID,
		&gitlab.CreateImpersonationTokenOptions{
			Name:   &name,
			Scopes: &scopes,
		},
	)
	if err != nil {
		return "", err
	}

	return impToken.Token, nil
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
	fmt.Println("'api' scope access : access scope: to be able to make api calls")
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

func loginAndRequestToken(url string) (string, error) {
	username, err := promptUsername()
	if err != nil {
		return "", err
	}

	password, err := promptPassword()
	if err != nil {
		return "", err
	}

	// Attempt to authenticate and create a token

	note := fmt.Sprintf("git-bug - %s", url)

	ok, id, err := validateUsername(username)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("invalid username")
	}

	client, err := buildClientFromUsernameAndPassword(username, password)
	if err != nil {
		return "", err
	}

	fmt.Println(username, password)

	return requestToken(client, id, note, "api")
}

func promptUsername() (string, error) {
	for {
		fmt.Print("username: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimRight(line, "\n")

		ok, _, err := validateUsername(line)
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

func getProjectPath(url string) (string, error) {

	cleanUrl := strings.TrimSuffix(url, ".git")
	objectUrl, err := neturl.Parse(cleanUrl)
	if err != nil {
		return "", nil
	}

	return objectUrl.Path[1:], nil
}

func getValidGitlabRemoteURLs(remotes map[string]string) []string {
	urls := make([]string, 0, len(remotes))
	for _, u := range remotes {
		path, err := getProjectPath(u)
		if err != nil {
			continue
		}

		urls = append(urls, fmt.Sprintf("%s%s", "gitlab.com", path))
	}

	return urls
}

func validateUsername(username string) (bool, int, error) {
	// no need for a token for this action
	client := buildClient("")

	users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &username})
	if err != nil {
		return false, 0, err
	}

	if len(users) == 0 {
		return false, 0, fmt.Errorf("username not found")
	} else if len(users) > 1 {
		return false, 0, fmt.Errorf("found multiple matches")
	}

	if users[0].Username == username {
		return true, users[0].ID, nil
	}

	return false, 0, nil
}

func validateProjectURL(url, token string) (bool, int, error) {
	client := buildClient(token)

	projectPath, err := getProjectPath(url)
	if err != nil {
		return false, 0, err
	}

	project, _, err := client.Projects.GetProject(projectPath, &gitlab.GetProjectOptions{})
	if err != nil {
		return false, 0, err
	}

	return true, project.ID, nil
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
