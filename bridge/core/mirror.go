package core

import (
	"fmt"
	"time"

	"github.com/git-bug/git-bug/entities/identity"
)

// MirrorMetaKey returns the metadata key that stores the remote identifier
// of a mirrored bug or operation. kind names what is stored (for example
// "issue" or "comment"); target and repo identify the remote repository,
// so the same bug can be mirrored to several trackers.
func MirrorMetaKey(kind, target, repo string) string {
	return fmt.Sprintf("mirror-%s:%s:%s", kind, target, repo)
}

// MirrorAttribution returns a Markdown line naming the original author and
// time of mirrored content, which the remote shows as written by the token
// owner. source, if not empty, links to the original issue.
func MirrorAttribution(author identity.Interface, at time.Time, source string) string {
	who := author.DisplayName()
	if author.Login() != "" && author.Login() != author.Name() {
		who = fmt.Sprintf("%s (%s)", author.Name(), author.Login())
	}
	line := fmt.Sprintf("_Originally posted by %s on %s", who, at.UTC().Format("2006-01-02 15:04 UTC"))
	if source != "" {
		line += " at " + source
	}
	return line + "_\n\n"
}
