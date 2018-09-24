package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/spf13/cobra"
)

func runBridgeConfigure(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	target, err := promptTarget()
	if err != nil {
		return err
	}

	name, err := promptName()
	if err != nil {
		return err
	}

	b, err := bridge.NewBridge(backend, target, name)
	if err != nil {
		return err
	}

	err = b.Configure()
	if err != nil {
		return err
	}

	return nil
}

func promptTarget() (string, error) {
	targets := bridge.Targets()

	for {
		fmt.Printf("target (%s): ", strings.Join(targets, ","))

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimRight(line, "\n")

		for _, t := range targets {
			if t == line {
				return t, nil
			}
		}

		fmt.Println("invalid target")
	}
}

func promptName() (string, error) {
	defaultName := "default"

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
	Use:   "configure",
	Short: "Configure a new bridge",
	RunE:  runBridgeConfigure,
}

func init() {
	bridgeCmd.AddCommand(bridgeConfigureCmd)
}
