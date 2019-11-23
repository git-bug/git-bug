package jira

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/repository"
)

const (
	target             = "jira"
	keyServer          = "server"
	keyProject         = "project"
	keyCredentialsFile = "credentials-file"
	keyUsername        = "username"
	keyPassword        = "password"
	keyIDMap           = "bug-id-map"
	keyCreateDefaults  = "create-issue-defaults"
	keyCreateGitBug    = "create-issue-gitbug-id"

	defaultTimeout = 60 * time.Second
)

const moreConfigText = `
NOTE: There are a few optional configuration values that you can additionally
set in your git configuration to influence the behavior of the bridge. Please
see the notes at:
https://github.com/MichaelMure/git-bug/blob/master/doc/jira_bridge.md
`

// Configure sets up the bridge configuration
func (g *Jira) Configure(
	repo repository.RepoCommon, params core.BridgeParams) (
	core.Configuration, error) {
	conf := make(core.Configuration)
	var err error
	var url string
	var project string
	var credentialsFile string
	var username string
	var password string
	var serverURL string

	if params.Token != "" || params.TokenStdin {
		return nil, fmt.Errorf(
			"JIRA session tokens are extremely short lived. We don't store them " +
				"in the configuration, so they are not valid for this bridge.")
	}

	if params.Owner != "" {
		return nil, fmt.Errorf("owner doesn't make sense for jira")
	}

	serverURL = params.URL
	if url == "" {
		// terminal prompt
		serverURL, err = prompt("JIRA server URL", "URL")
		if err != nil {
			return nil, err
		}
	}

	project = params.Project
	if project == "" {
		project, err = prompt("JIRA project key", "project")
		if err != nil {
			return nil, err
		}
	}

	choice, err := promptCredentialOptions(serverURL)
	if err != nil {
		return nil, err
	}

	if choice == 1 {
		credentialsFile, err = prompt("Credentials file path", "path")
		if err != nil {
			return nil, err
		}
	}

	username, err = prompt("JIRA username", "username")
	if err != nil {
		return nil, err
	}

	password, err = input.PromptPassword()
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(
		&SessionQuery{Username: username, Password: password})
	if err != nil {
		return nil, err
	}

	fmt.Printf("Attempting to login with credentials...\n")
	client := NewClient(serverURL, nil)
	err = client.RefreshTokenRaw(jsonData)
	if err != nil {
		return nil, err
	}

	// verify access to the project with credentials
	_, err = client.GetProject(project)
	if err != nil {
		return nil, fmt.Errorf(
			"Project %s doesn't exist on %s, or authentication credentials for (%s)"+
				" are invalid",
			project, serverURL, username)
	}

	conf[core.KeyTarget] = target
	conf[keyServer] = serverURL
	conf[keyProject] = project
	switch choice {
	case 1:
		conf[keyCredentialsFile] = credentialsFile
		err = ioutil.WriteFile(credentialsFile, jsonData, 0644)
		if err != nil {
			return nil, errors.Wrap(
				err, fmt.Sprintf("Unable to write credentials to %s", credentialsFile))
		}
	case 2:
		conf[keyUsername] = username
		conf[keyPassword] = password
	case 3:
		conf[keyUsername] = username
	}

	err = g.ValidateConfig(conf)
	if err != nil {
		return nil, err
	}

	fmt.Print(moreConfigText)
	return conf, nil
}

// ValidateConfig returns true if all required keys are present
func (*Jira) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.KeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.KeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}

	if _, ok := conf[keyProject]; !ok {
		return fmt.Errorf("missing %s key", keyProject)
	}

	return nil
}

const credentialsText = `
How would you like to store your JIRA login credentials?
[1]: sidecar JSON file: Your credentials will be stored in a JSON sidecar next
     to your git config. Note that it will contain your JIRA password in clear
     text.
[2]: git-config: Your credentials will be stored in the git config. Note that
     it will contain your JIRA password in clear text.
[3]: username in config, askpass: Your username will be stored in the git
     config. We will ask you for your password each time you execute the bridge.
`

func promptCredentialOptions(serverURL string) (int, error) {
	fmt.Print(credentialsText)
	for {
		fmt.Print("Select option: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println()
		if err != nil {
			return -1, err
		}

		line = strings.TrimRight(line, "\n")

		index, err := strconv.Atoi(line)
		if err != nil || (index != 1 && index != 2 && index != 3) {
			fmt.Println("invalid input")
			continue
		}

		return index, nil
	}
}

func prompt(description, name string) (string, error) {
	for {
		fmt.Printf("%s: ", description)

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimRight(line, "\n")
		if line == "" {
			fmt.Printf("%s is empty\n", name)
			continue
		}

		return line, nil
	}
}
