package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runBridgeConfigure(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

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
	Use:     "configure",
	Short:   "Configure a new bridge",
	PreRunE: loadRepo,
	RunE:    runBridgeConfigure,
}

func init() {
	bridgeCmd.AddCommand(bridgeConfigureCmd)
}
