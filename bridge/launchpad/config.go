package launchpad

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/repository"
)

var ErrBadProjectURL = errors.New("bad Launchpad project URL")

const (
	target         = "launchpad-preview"
	keyProject     = "project"
	defaultTimeout = 60 * time.Second
)

func (l *Launchpad) Configure(repo repository.RepoCommon, params core.BridgeParams) (core.Configuration, error) {
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
		project, err = splitURL(params.URL)
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
	conf[core.KeyTarget] = target

	err = l.ValidateConfig(conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (*Launchpad) ValidateConfig(conf core.Configuration) error {
	if _, ok := conf[keyProject]; !ok {
		return fmt.Errorf("missing %s key", keyProject)
	}

	if _, ok := conf[core.KeyTarget]; !ok {
		return fmt.Errorf("missing %s key", core.KeyTarget)
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

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}

	return resp.StatusCode == http.StatusOK, nil
}

// extract project name from url
func splitURL(url string) (string, error) {
	re, err := regexp.Compile(`launchpad\.net[\/:]([^\/]*[a-z]+)`)
	if err != nil {
		panic("regexp compile:" + err.Error())
	}

	res := re.FindStringSubmatch(url)
	if res == nil {
		return "", ErrBadProjectURL
	}

	return res[1], nil
}
