package auth

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

var _ Credential = &GitCredential{}

// GitCredential the credentials
// see: https://git-scm.com/docs/git-credential#IOFMT
type GitCredential struct {
	target     string
	createTime time.Time
	Username   string
	Password   string
	meta       map[string]string
}

// NewGitCredential instantiate a new object
func NewGitCredential(target, username, password string) *GitCredential {
	return &GitCredential{
		target:     target,
		createTime: time.Now(),
		Username:   username,
		Password:   password,
	}
}

// NewGitCredentialFromConfig execute "git credential fill"
func NewGitCredentialFromConfig(conf map[string]string) (*GitCredential, error) {
	cred := &GitCredential{}

	cred.target = conf[configKeyTarget]
	if createTime, ok := conf[configKeyCreateTime]; ok {
		if t, err := repository.ParseTimestamp(createTime); err == nil {
			cred.createTime = t
		}
	}
	cred.meta = metaFromConfig(conf)

	cmd := exec.Command("git", "credential", "fill")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	go func() {
		defer stdin.Close()
		io.WriteString(stdin,
			fmt.Sprintf("protocol=https\nhost=%s\n\n", cred.target))
	}()

	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if parts[0] == "username" {
			cred.Username = parts[1]
		} else if parts[0] == "password" {
			cred.Password = parts[1]
		}
	}

	return cred, nil
}

func (cred *GitCredential) ID() entity.Id {
	sum := sha256.Sum256([]byte(cred.target + cred.Username + cred.Password))
	return entity.Id(fmt.Sprintf("%x", sum))
}

func (cred *GitCredential) Target() string {
	return cred.target
}

func (cred *GitCredential) Kind() CredentialKind {
	return KindGitCredential
}

func (cred *GitCredential) CreateTime() time.Time {
	return cred.createTime
}

// Validate ensure token important fields are valid
func (cred *GitCredential) Validate() error {
	if cred.Username == "" {
		return fmt.Errorf("missing username")
	}
	if cred.Password == "" {
		return fmt.Errorf("missing password")
	}
	if cred.target == "" {
		return fmt.Errorf("missing target")
	}
	if cred.createTime.IsZero() || cred.createTime.Equal(time.Time{}) {
		return fmt.Errorf("missing creation time")
	}
	if !core.TargetExist(cred.target) {
		return fmt.Errorf("unknown target")
	}
	return nil
}

func (cred *GitCredential) Metadata() map[string]string {
	return cred.meta
}

func (cred *GitCredential) GetMetadata(key string) (string, bool) {
	val, ok := cred.meta[key]
	return val, ok
}

func (cred *GitCredential) SetMetadata(key string, value string) {
	if cred.meta == nil {
		cred.meta = make(map[string]string)
	}
	cred.meta[key] = value
}

func (cred *GitCredential) toConfig() map[string]string {
	return map[string]string{}
}
