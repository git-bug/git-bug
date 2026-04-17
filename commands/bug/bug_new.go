package bugcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	buginput "github.com/git-bug/git-bug/commands/bug/input"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/text"
)

type bugNewOptions struct {
	title          string
	message        string
	messageFile    string
	nonInteractive bool

	// Pull-request flags. --pr turns the new bug into a PR; --base and --head
	// are required with --pr. --head-commit defaults to HEAD; --draft creates
	// the PR in draft state.
	pr         bool
	baseRef    string
	headRef    string
	headCommit string
	draft      bool
}

func newBugNewCommand(env *execenv.Env) *cobra.Command {
	options := bugNewOptions{}

	cmd := &cobra.Command{
		Use:     "new",
		Short:   "Create a new bug",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugNew(env, options)
		}),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the issue")
	flags.StringVarP(&options.message, "message", "m", "",
		"Provide a message to describe the issue")
	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	flags.BoolVar(&options.pr, "pr", false,
		"Create a pull-request instead of a plain issue. Requires --base and --head.")
	flags.StringVar(&options.baseRef, "base", "",
		"For --pr: the base branch the PR targets (e.g. refs/heads/main).")
	flags.StringVar(&options.headRef, "head", "",
		"For --pr: the head branch the PR proposes to merge.")
	flags.StringVar(&options.headCommit, "head-commit", "",
		"For --pr: the head commit hash. Defaults to the current HEAD of --head if resolvable.")
	flags.BoolVar(&options.draft, "draft", false,
		"For --pr: mark the PR as draft (not ready for review).")

	return cmd
}

func runBugNew(env *execenv.Env, opts bugNewOptions) error {
	var err error
	if opts.messageFile != "" && opts.message == "" {
		opts.title, opts.message, err = buginput.BugCreateFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.messageFile == "" && (opts.message == "" || opts.title == "") {
		opts.title, opts.message, err = buginput.BugCreateEditorInput(env.Backend, opts.title, opts.message)

		if err == buginput.ErrEmptyTitle {
			env.Out.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	title := text.CleanupOneLine(opts.title)
	message := text.Cleanup(opts.message)

	if opts.pr {
		if opts.baseRef == "" || opts.headRef == "" {
			return fmt.Errorf("--pr requires --base and --head")
		}

		headCommit := opts.headCommit
		if headCommit == "" {
			// Try to resolve the tip of --head in the working repository.
			hash, err := resolveRefTip(env, opts.headRef)
			if err != nil {
				return fmt.Errorf("--head-commit not supplied and could not resolve %s: %w", opts.headRef, err)
			}
			headCommit = hash
		}

		b, _, err := env.Backend.Bugs().NewPR(title, message, opts.baseRef, opts.headRef, headCommit, opts.draft)
		if err != nil {
			return err
		}
		env.Out.Printf("%s created (pr)\n", b.Id().Human())
		return nil
	}

	if opts.baseRef != "" || opts.headRef != "" || opts.headCommit != "" || opts.draft {
		return fmt.Errorf("--base / --head / --head-commit / --draft require --pr")
	}

	b, _, err := env.Backend.Bugs().New(title, message)
	if err != nil {
		return err
	}

	env.Out.Printf("%s created\n", b.Id().Human())

	return nil
}

// resolveRefTip returns the commit hash a ref points to in the working repository.
func resolveRefTip(env *execenv.Env, ref string) (string, error) {
	h, err := env.Repo.ResolveRef(ref)
	if err == nil {
		return string(h), nil
	}
	// Allow callers to pass a short branch name like "feature"; try the full heads path.
	if h2, err2 := env.Repo.ResolveRef("refs/heads/" + ref); err2 == nil {
		return string(h2), nil
	}
	if err == repository.ErrNotFound {
		return "", fmt.Errorf("ref %q not found", ref)
	}
	return "", err
}
