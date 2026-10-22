package repository

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/execabs"
)

var _ RepoRemote = &GitRepo{}

// GitRepo is the remote operations of a git directory, running the git
// binary: they get git's own transports and configuration, ssh's included,
// where go-git reimplements them.
//
// It never prompts: a password or a passphrase git or ssh would ask for, or the
// confirmation of an unknown host, fails the operation instead, as with
// go-git. Credentials come from an ssh agent, keys without a passphrase, or a
// credential helper.
type GitRepo struct {
	git    string
	gitDir string
	config RepoConfig
	// localEnv is the environment variables pointing git to a repository, or
	// to parts of one, that run removes
	localEnv map[string]bool
}

// minGitVersion is the oldest git with all the options run uses:
// --no-write-fetch-head and --no-auto-maintenance came with 2.29.
var minGitVersion = [2]int{2, 29}

// NewGitRepo returns the GitRepo of the git directory gitDir, whose remotes
// are those of config. It fails if no git binary is found in the PATH, or if
// it's older than minGitVersion.
func NewGitRepo(gitDir string, config RepoConfig) (*GitRepo, error) {
	// never from the configuration: the repository's own could name any
	// executable
	git, err := execabs.LookPath("git")
	if err != nil {
		return nil, err
	}
	// git runs from another directory than the current one, see run
	gitDir, err = filepath.Abs(gitDir)
	if err != nil {
		return nil, err
	}

	out, err := exec.Command(git, "version").Output()
	if err != nil {
		return nil, fmt.Errorf("git version: %w", err)
	}
	version, err := parseGitVersion(string(out))
	if err != nil {
		return nil, err
	}
	if version[0] < minGitVersion[0] || version[0] == minGitVersion[0] && version[1] < minGitVersion[1] {
		return nil, fmt.Errorf("git %d.%d or later is needed, found %d.%d", minGitVersion[0], minGitVersion[1], version[0], version[1])
	}

	// git started from a hook, or from a script run by one, has variables such
	// as GIT_OBJECT_DIRECTORY or GIT_INDEX_FILE set for the repository it
	// runs in, which would override ours. git lists them: remove them, as git
	// does when it runs itself in another repository, a submodule. As git
	// does too, keep those of the configuration given with -c, such as with
	// git -c http.proxy=<proxy> bug push.
	out, err = exec.Command(git, "--git-dir="+gitDir, "rev-parse", "--local-env-vars").Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-parse: %w", err)
	}
	localEnv := make(map[string]bool)
	for _, name := range strings.Fields(string(out)) {
		if name != "GIT_CONFIG_PARAMETERS" && name != "GIT_CONFIG_COUNT" {
			localEnv[name] = true
		}
	}

	return &GitRepo{git: git, gitDir: gitDir, config: config, localEnv: localEnv}, nil
}

// parseGitVersion returns the major and minor version of git from the output
// of git version, such as "git version 2.39.5 (Apple Git-154)" or
// "git version 2.45.1.windows.1".
func parseGitVersion(out string) ([2]int, error) {
	var version [2]int
	fields := strings.Fields(out)
	if len(fields) < 3 || fields[0] != "git" || fields[1] != "version" {
		return version, fmt.Errorf("unexpected git version %q", strings.TrimSpace(out))
	}
	numbers := strings.SplitN(fields[2], ".", 3)
	if len(numbers) < 2 {
		return version, fmt.Errorf("unexpected git version %q", strings.TrimSpace(out))
	}
	for i := range version {
		n, err := strconv.Atoi(numbers[i])
		if err != nil {
			return version, fmt.Errorf("unexpected git version %q", strings.TrimSpace(out))
		}
		version[i] = n
	}
	return version, nil
}

// FetchRefs retrieves the refs of the given namespaces (and the data they
// point to) from a remote, and stores them as that remote's tracking refs.
// Local refs are left untouched.
func (repo *GitRepo) FetchRefs(ctx context.Context, remote string, namespaces ...string) (string, error) {
	// from git 2.48, git can update refs/remotes/<remote>/HEAD on fetch;
	// older ones ignore the setting
	config := []string{"-c", "remote." + remote + ".followRemoteHEAD=never"}
	refspecs := make([]string, len(namespaces))
	for i, namespace := range namespaces {
		refspecs[i] = refPrefix(namespace) + "*:" + trackingRefPrefix(remote, namespace) + "*"
	}
	return repo.run(ctx, config, "fetch", remote, refspecs)
}

// PushRefs sends the local refs of the given namespaces (and the data they
// point to) to a remote, and updates that remote's tracking refs to match.
func (repo *GitRepo) PushRefs(ctx context.Context, remote string, namespaces ...string) (string, error) {
	// git updates the tracking ref of a pushed ref only if a fetch refspec of
	// the remote maps it, which the configured ones don't for our namespaces:
	// add them for this run. -c adds to the configured ones, refs/heads/*
	// stays. A mirror remote would push every ref, and refuse our refspecs:
	// push only ours. The push options of push.pushOption are for the code,
	// such as skipping a CI run, and a remote not accepting any would refuse
	// the push: an empty value clears them.
	config := make([]string, 0, 2*len(namespaces)+4)
	config = append(config, "-c", "remote."+remote+".mirror=false", "-c", "push.pushOption=")
	refspecs := make([]string, len(namespaces))
	for i, namespace := range namespaces {
		config = append(config, "-c", "remote."+remote+".fetch="+refPrefix(namespace)+"*:"+trackingRefPrefix(remote, namespace)+"*")
		refspecs[i] = refPrefix(namespace) + "*:" + refPrefix(namespace) + "*"
	}
	return repo.run(ctx, config, "push", remote, refspecs)
}

// run runs a git command, fetch or push, with the remote and the refspecs as
// its arguments, and config as git's options before it. It returns what git
// printed.
func (repo *GitRepo) run(ctx context.Context, config []string, command string, remote string, refspecs []string) (string, error) {
	// the remote comes from the command line or the configuration: only a
	// configured one is run, never a URL. --end-of-options also keeps one
	// looking like an option, such as --upload-pack=<command>, from being
	// taken as one.
	remotes, err := repo.config.GetRemotes()
	if err != nil {
		return "", err
	}
	if _, ok := remotes[remote]; !ok {
		return "", fmt.Errorf("unknown remote %q", remote)
	}

	// the repository's hooks are about its code: a pre-push hook running
	// tests, or a reference-transaction one, has no say on the bugs. A hooks
	// directory that doesn't exist runs none.
	args := []string{"--git-dir=" + repo.gitDir, "-c", "core.hooksPath=" + os.DevNull}
	args = append(args, config...)
	args = append(args, command)
	if command == "fetch" {
		// git also applies the remote's configured fetch refspecs to the
		// fetched refs, which a mirror-style one (+refs/*:refs/*) would map
		// onto the local refs: only ours apply. Tags are fetched apart from
		// the refspecs, following remote.<name>.tagOpt or the tags pointing
		// into the fetched history: none are. FETCH_HEAD stays the one of the
		// user's last fetch, and submodules aren't fetched. In a partial
		// clone, git would leave out the objects the filter excludes, to
		// fetch them on demand, which go-git can't: all of them are fetched.
		// The repository's maintenance, such as a repack, is left to the
		// user's own fetches.
		args = append(args, "--refmap=", "--no-tags", "--no-write-fetch-head", "--no-recurse-submodules", "--no-filter", "--no-auto-maintenance")
	}
	if command == "push" {
		// following push.followTags, git would also push the annotated tags
		// pointing to commits the remote has, such as the code's releases:
		// only ours are pushed. Following push.recurseSubmodules, it would
		// push the submodules too, or only them. Following push.gpgSign, it
		// would sign the push, which a remote not supporting it refuses, and
		// ask for the key's passphrase: the operations carry their own
		// signatures.
		args = append(args, "--no-follow-tags", "--no-recurse-submodules", "--signed=false")
	}
	args = append(args, "--end-of-options", remote)
	args = append(args, refspecs...)

	// git runs an askpass program, from GIT_ASKPASS, core.askPass or
	// SSH_ASKPASS, before checking GIT_TERMINAL_PROMPT. Set but empty,
	// GIT_ASKPASS disables the three. Credential helpers still run.
	env := make([]string, 0, len(os.Environ())+3)
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !repo.localEnv[name] {
			env = append(env, kv)
		}
	}
	env = append(env, "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=")
	// ssh prompts on the terminal whatever its input is. Keep it from doing so,
	// unless the user chose the ssh command, which git takes from
	// GIT_SSH_COMMAND, core.sshCommand then GIT_SSH: it's then theirs.
	cfg, err := repo.config.Config().Read()
	if err != nil {
		return "", err
	}
	if os.Getenv("GIT_SSH_COMMAND") == "" && cfg.Value("core.sshCommand", "") == "" && os.Getenv("GIT_SSH") == "" {
		env = append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}

	// git takes a relative remote URL as relative to the directory it runs
	// from, which --git-dir makes the current one: run it from where git would,
	// the worktree, or the git directory of a bare repository. For a linked
	// worktree, it's the main one, which the remotes are configured for.
	bare, err := cfg.Bool("core.bare", false)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(repo.gitDir)
	switch worktree := cfg.Value("core.worktree", ""); {
	case bare:
		dir = repo.gitDir
	case filepath.IsAbs(worktree):
		dir = worktree
	case worktree != "":
		dir = filepath.Join(repo.gitDir, worktree)
	}

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, repo.git, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Env = env
	cmd.Dir = dir
	// canceling ctx kills git, not the ssh it started, which would keep the
	// output open, and Run waiting
	cmd.WaitDelay = time.Second

	err = cmd.Run()
	if ctx.Err() != nil {
		return "", fmt.Errorf("git %s: %w", command, ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", command, err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}
