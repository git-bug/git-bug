package jira

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
)

const moreConfigText = `
NOTE: There are a few optional configuration values that you can additionally
set in your git configuration to influence the behavior of the bridge. Please
see the notes at:
https://github.com/MichaelMure/git-bug/blob/master/doc/jira_bridge.md
`

const credTypeText = `
JIRA has recently altered it's authentication strategies. Servers deployed
prior to October 1st 2019 must use "SESSION" authentication, whereby the REST
client logs in with an actual username and password, is assigned a session, and
passes the session cookie with each request. JIRA Cloud and servers deployed
after October 1st 2019 must use "TOKEN" authentication. You must create a user
API token and the client will provide this along with your username with each
request.`

// Configure sets up the bridge configuration
func (g *Jira) Configure(repo *cache.RepoCache, params core.BridgeParams) (core.Configuration, error) {
	conf := make(core.Configuration)
	conf[core.ConfigKeyTarget] = target

	var err error

	// if params.Token != "" || params.TokenStdin {
	// 	return nil, fmt.Errorf(
	// 		"JIRA session tokens are extremely short lived. We don't store them " +
	// 			"in the configuration, so they are not valid for this bridge.")
	// }

	if params.Owner != "" {
		fmt.Println("warning: --owner is ineffective for a Jira bridge")
	}

	serverURL := params.URL
	if serverURL == "" {
		// terminal prompt
		serverURL, err = input.Prompt("JIRA server URL", "URL", input.Required)
		if err != nil {
			return nil, err
		}
	}
	conf[keyServer] = serverURL

	project := params.Project
	if project == "" {
		project, err = input.Prompt("JIRA project key", "project", input.Required)
		if err != nil {
			return nil, err
		}
	}
	conf[keyProject] = project

	fmt.Println(credTypeText)
	credType, err := input.PromptChoice("Authentication mechanism", []string{"SESSION", "TOKEN"})
	if err != nil {
		return nil, err
	}

	switch credType {
	case 1:
		conf[keyCredentialsType] = "SESSION"
	case 2:
		conf[keyCredentialsType] = "TOKEN"
	}

	fmt.Println("How would you like to store your JIRA login credentials?")
	credTargetChoice, err := input.PromptChoice("Credential storage", []string{
		"sidecar JSON file: Your credentials will be stored in a JSON sidecar next" +
			"to your git config. Note that it will contain your JIRA password in clear" +
			"text.",
		"git-config: Your credentials will be stored in the git config. Note that" +
			"it will contain your JIRA password in clear text.",
		"username in config, askpass: Your username will be stored in the git" +
			"config. We will ask you for your password each time you execute the bridge.",
	})
	if err != nil {
		return nil, err
	}

	username, err := input.Prompt("JIRA username", "username", input.Required)
	if err != nil {
		return nil, err
	}

	password, err := input.PromptPassword("Password", "password", input.Required)
	if err != nil {
		return nil, err
	}

	switch credTargetChoice {
	case 1:
		// TODO: a validator to see if the path is writable ?
		credentialsFile, err := input.Prompt("Credentials file path", "path", input.Required)
		if err != nil {
			return nil, err
		}
		conf[keyCredentialsFile] = credentialsFile
		jsonData, err := json.Marshal(&SessionQuery{Username: username, Password: password})
		if err != nil {
			return nil, err
		}
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

	fmt.Printf("Attempting to login with credentials...\n")
	client := NewClient(serverURL, nil)
	err = client.Login(conf)
	if err != nil {
		return nil, err
	}

	// verify access to the project with credentials
	fmt.Printf("Checking project ...\n")
	_, err = client.GetProject(project)
	if err != nil {
		return nil, fmt.Errorf(
			"Project %s doesn't exist on %s, or authentication credentials for (%s)"+
				" are invalid",
			project, serverURL, username)
	}

	fmt.Print(moreConfigText)
	return conf, nil
}

// ValidateConfig returns true if all required keys are present
func (*Jira) ValidateConfig(conf core.Configuration) error {
	if v, ok := conf[core.ConfigKeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.ConfigKeyTarget)
	} else if v != target {
		return fmt.Errorf("unexpected target name: %v", v)
	}

	if _, ok := conf[keyProject]; !ok {
		return fmt.Errorf("missing %s key", keyProject)
	}

	return nil
}
