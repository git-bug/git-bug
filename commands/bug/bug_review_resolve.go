package bugcmd

import (
	"fmt"
	"strings"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entity"
)

// resolveReviewByPrefix scans bugs with kind=PR in the cache for a review
// whose combined id starts with the given prefix. Returns the bug and the
// review's combined id. Errors on ambiguity or miss.
func resolveReviewByPrefix(env *execenv.Env, prefix string) (*cache.BugCache, entity.CombinedId, error) {
	var matches []struct {
		bug        *cache.BugCache
		combinedId entity.CombinedId
	}

	// Walk all bugs; PR support currently treats review ids as scoped within
	// the PR namespace. Future work: maintain a review-id index in the cache
	// to avoid this O(n_bugs * n_reviews) scan on large repos.
	for _, id := range env.Backend.Bugs().AllIds() {
		b, err := env.Backend.Bugs().Resolve(id)
		if err != nil {
			continue
		}
		snap := b.Snapshot()
		for _, r := range snap.Reviews {
			if strings.HasPrefix(r.CombinedId().String(), prefix) {
				matches = append(matches, struct {
					bug        *cache.BugCache
					combinedId entity.CombinedId
				}{b, r.CombinedId()})
			}
		}
	}

	switch len(matches) {
	case 0:
		return nil, entity.UnsetCombinedId, fmt.Errorf("no review matching prefix %q", prefix)
	case 1:
		return matches[0].bug, matches[0].combinedId, nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.combinedId.Human()
		}
		return nil, entity.UnsetCombinedId, fmt.Errorf("review prefix %q is ambiguous: %s", prefix, strings.Join(ids, ", "))
	}
}

// findReviewCommentByPrefix scans a snapshot's review comments for one whose
// combined id starts with the given prefix. Returns the comment and true on a
// unique match.
func findReviewCommentByPrefix(snap *bug.Snapshot, prefix string) (bug.ReviewComment, bool) {
	var match bug.ReviewComment
	found := 0
	for _, r := range snap.Reviews {
		for _, c := range r.Comments {
			if strings.HasPrefix(c.CombinedId().String(), prefix) {
				match = c
				found++
			}
		}
	}
	if found == 1 {
		return match, true
	}
	return bug.ReviewComment{}, false
}
