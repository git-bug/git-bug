package usercmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
)

type userShowOptions struct {
	fields string
}

func newUserShowCommand(env *execenv.Env) *cobra.Command {
	options := userShowOptions{}

	cmd := &cobra.Command{
		Use:     "user show [USER_ID]",
		Short:   "Display a user identity",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runUserShow(env, options, args)
		}),
		ValidArgsFunction: completion.User(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	fields := []string{"email", "humanId", "id", "lastModification", "lastModificationLamports", "login", "metadata", "name"}
	flags.StringVarP(&options.fields, "field", "f", "",
		"Select field to display. Valid values are ["+strings.Join(fields, ",")+"]")
	cmd.RegisterFlagCompletionFunc("field", completion.From(fields))

	return cmd
}

func runUserShow(env *execenv.Env, opts userShowOptions, args []string) error {
	if len(args) > 1 {
		return errors.New("only one identity can be displayed at a time")
	}

	var id *cache.IdentityCache
	var err error
	if len(args) == 1 {
		id, err = env.Backend.Identities().ResolvePrefix(args[0])
	} else {
		id, err = env.Backend.GetUserIdentity()
	}

	if err != nil {
		return err
	}

	if opts.fields != "" {
		switch opts.fields {
		case "email":
			env.Out.Printf("%s\n", id.Email())
		case "login":
			env.Out.Printf("%s\n", id.Login())
		case "humanId":
			env.Out.Printf("%s\n", id.Id().Human())
		case "id":
			env.Out.Printf("%s\n", id.Id())
		case "lastModification":
			env.Out.Printf("%s\n", id.LastModification().
				Time().Format("Mon Jan 2 15:04:05 2006 -0700"))
		case "lastModificationLamport":
			for name, t := range id.LastModificationLamports() {
				env.Out.Printf("%s\n%d\n", name, t)
			}
		case "metadata":
			for key, value := range id.ImmutableMetadata() {
				env.Out.Printf("%s\n%s\n", key, value)
			}
		case "name":
			env.Out.Printf("%s\n", id.Name())

		default:
			return fmt.Errorf("\nUnsupported field: %s\n", opts.fields)
		}

		return nil
	}

	env.Out.Printf("Id: %s\n", id.Id())
	env.Out.Printf("Name: %s\n", id.Name())
	env.Out.Printf("Email: %s\n", id.Email())
	env.Out.Printf("Login: %s\n", id.Login())
	env.Out.Printf("Last modification: %s\n", id.LastModification().Time().Format("Mon Jan 2 15:04:05 2006 -0700"))
	env.Out.Printf("Last moditication (lamport):\n")
	for name, t := range id.LastModificationLamports() {
		env.Out.Printf("\t%s: %d", name, t)
	}
	env.Out.Println("Metadata:")
	for key, value := range id.ImmutableMetadata() {
		env.Out.Printf("    %s --> %s\n", key, value)
	}
	// env.Out.Printf("Protected: %v\n", id.IsProtected())

	return nil
}
