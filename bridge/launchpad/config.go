package launchpad

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/repository"
)

const keyProject = "project"

func (*Launchpad) Configure(repo repository.RepoCommon, params core.BridgeParams) (core.Configuration, error) {
	conf := make(core.Configuration)

	if params.Project == "" {
		projectName, err := promptProjectName()
		if err != nil {
			return nil, err
		}

		conf[keyProject] = projectName
	}

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

func validateProject() (bool, error) {
	return false, nil
}
