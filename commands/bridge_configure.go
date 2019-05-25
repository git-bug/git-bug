package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MichaelMure/git-bug/bridge/core"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

const (
	defaultName = "default"
)

var (
	name         string
	target       string
	bridgeParams core.BridgeParams
)

func runBridgeConfigure(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	if target == "" {
		target, err = promptTarget()
		if err != nil {
			return err
		}
	}

	if name == "" {
		name, err = promptName()
		if err != nil {
			return err
		}
	}

	b, err := bridge.NewBridge(backend, target, name)
	if err != nil {
		return err
	}

	err = b.Configure(bridgeParams)
	if err != nil {
		return err
	}

	fmt.Println("successfully configured bridge")
	return nil
}

func promptTarget() (string, error) {
	targets := bridge.Targets()

	for {
		for i, target := range targets {
			fmt.Printf("[%d]: %s\n", i+1, target)
		}
		fmt.Printf("target: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimRight(line, "\n")

		index, err := strconv.Atoi(line)
		if err != nil || index <= 0 || index > len(targets) {
			fmt.Println("invalid input")
			continue
		}

		return targets[index-1], nil
	}
}

func promptName() (string, error) {
	fmt.Printf("name [%s]: ", defaultName)

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimRight(line, "\n")

	if line == "" {
		return defaultName, nil
	}

	return line, nil
}

var bridgeConfigureCmd = &cobra.Command{
	Use:     "configure",
	Short:   "Configure a new bridge.",
	PreRunE: loadRepo,
	RunE:    runBridgeConfigure,
}

func init() {
	bridgeCmd.AddCommand(bridgeConfigureCmd)
	bridgeConfigureCmd.Flags().StringVarP(&name, "name", "n", "", "Bridge name")
	bridgeConfigureCmd.Flags().StringVarP(&target, "target", "t", "", "Bridge target name. Valid values are [github,gitlab,gitea,launchpad-preview]")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.URL, "url", "u", "", "Repository url")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Owner, "owner", "o", "", "Repository owner")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Token, "token", "T", "", "Authentication token")
	bridgeConfigureCmd.Flags().StringVarP(&bridgeParams.Project, "project", "p", "", "Repository name")
}
