package launchpad

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/repository"
)

const keyProject = "project"

var (
	rxLaunchpadURL = regexp.MustCompile(`launchpad\.net[\/:]([^\/]*[a-z]+)`)
)

func (*Launchpad) Configure(repo repository.RepoCommon, params core.BridgeParams) (core.Configuration, error) {
	if params.Token != "" {
		fmt.Println("warning: --token is ineffective for a Launchpad bridge")
	}
	if params.Owner != "" {
		fmt.Println("warning: --owner is ineffective for a Launchpad bridge")
	}

	conf := make(core.Configuration)
	var err error
	var project string

	if params.Project != "" {
		project = params.Project

	} else if params.URL != "" {
		// get project name from url
		_, project, err = splitURL(params.URL)
		if err != nil {
			return nil, err
		}

	} else {
		// get project name from terminal prompt
		project, err = promptProjectName()
		if err != nil {
			return nil, err
		}
	}

	// verify project
	ok, err := validateProject(project)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("project doesn't exist")
	}

	conf[keyProject] = project
	return conf, nil
}

func (*Launchpad) ValidateConfig(conf core.Configuration) error {
	if _, ok := conf[keyProject]; !ok {
		return fmt.Errorf("missing %s key", keyProject)
	}

	return nil
}

func promptProjectName() (string, error) {
	for {
		fmt.Print("Launchpad project name: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimRight(line, "\n")

		if line == "" {
			fmt.Println("Project name is empty")
			continue
		}

		return line, nil
	}
}

func validateProject(project string) (bool, error) {
	url := fmt.Sprintf("%s/%s", apiRoot, project)

	client := := &http.Client{
		Timeout: defaultTimeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}

	return resp.StatusCode == http.StatusOK, nil
}

func splitURL(url string) (string, string, error) {
	res := rxLaunchpadURL.FindStringSubmatch(url)
	if res == nil {
		return "", "", fmt.Errorf("bad Launchpad project url")
	}

	return res[0], res[1], nil
}
