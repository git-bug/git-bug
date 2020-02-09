package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func runBridgeAuthShow(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	cred, err := auth.LoadWithPrefix(repo, args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Id: %s\n", cred.ID())
	fmt.Printf("Target: %s\n", cred.Target())
	fmt.Printf("Kind: %s\n", cred.Kind())
	fmt.Printf("Creation: %s\n", cred.CreateTime().Format(time.RFC822))

	switch cred := cred.(type) {
	case *auth.Token:
		fmt.Printf("Value: %s\n", cred.Value)
	}

	fmt.Println("Metadata:")

	meta := make([]string, 0, len(cred.Metadata()))
	for key, value := range cred.Metadata() {
		meta = append(meta, fmt.Sprintf("    %s --> %s\n", key, value))
	}
	sort.Strings(meta)

	fmt.Print(strings.Join(meta, ""))

	return nil
}

var bridgeAuthShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Display an authentication credential.",
	PreRunE: loadRepo,
	RunE:    runBridgeAuthShow,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeAuthCmd.AddCommand(bridgeAuthShowCmd)
}
