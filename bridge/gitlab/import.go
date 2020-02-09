package gitlab

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xanzy/go-gitlab"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/text"
)

// gitlabImporter implement the Importer interface
type gitlabImporter struct {
	conf core.Configuration

	// default user client
	client *gitlab.Client

	// iterator
	iterator *iterator

	// send only channel
	out chan<- core.ImportResult
}

func (gi *gitlabImporter) Init(repo *cache.RepoCache, conf core.Configuration) error {
	gi.conf = conf

	creds, err := auth.List(repo, auth.WithTarget(target), auth.WithKind(auth.KindToken))
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrMissingIdentityToken
	}

	gi.client, err = buildClient(conf[keyGitlabBaseUrl], creds[0].(*auth.Token))
	if err != nil {
		return err
	}

	return nil
}

// ImportAll iterate over all the configured repository issues (notes) and ensure the creation
// of the missing issues / comments / label events / title changes ...
func (gi *gitlabImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	gi.iterator = NewIterator(ctx, gi.client, 10, gi.conf[keyProjectID], since)
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(gi.out)

		// Loop over all matching issues
		for gi.iterator.NextIssue() {
			issue := gi.iterator.IssueValue()

			// create issue
			b, err := gi.ensureIssue(repo, issue)
			if err != nil {
				err := fmt.Errorf("issue creation: %v", err)
				out <- core.NewImportError(err, "")
				return
			}

			// Loop over all notes
			for gi.iterator.NextNote() {
				note := gi.iterator.NoteValue()
				if err := gi.ensureNote(repo, b, note); err != nil {
					err := fmt.Errorf("note creation: %v", err)
					out <- core.NewImportError(err, entity.Id(strconv.Itoa(note.ID)))
					return
				}
			}

			// Loop over all label events
			for gi.iterator.NextLabelEvent() {
				labelEvent := gi.iterator.LabelEventValue()
				if err := gi.ensureLabelEvent(repo, b, labelEvent); err != nil {
					err := fmt.Errorf("label event creation: %v", err)
					out <- core.NewImportError(err, entity.Id(strconv.Itoa(labelEvent.ID)))
					return
				}
			}

			if !b.NeedCommit() {
				out <- core.NewImportNothing(b.Id(), "no imported operation")
			} else if err := b.Commit(); err != nil {
				// commit bug state
				err := fmt.Errorf("bug commit: %v", err)
				out <- core.NewImportError(err, "")
				return
			}
		}

		if err := gi.iterator.Error(); err != nil {
			out <- core.NewImportError(err, "")
		}
	}()

	return out, nil
}

func (gi *gitlabImporter) ensureIssue(repo *cache.RepoCache, issue *gitlab.Issue) (*cache.BugCache, error) {
	// ensure issue author
	author, err := gi.ensurePerson(repo, issue.Author.ID)
	if err != nil {
		return nil, err
	}

	// resolve bug
	b, err := repo.ResolveBugCreateMetadata(metaKeyGitlabUrl, issue.WebURL)
	if err == nil {
		return b, nil
	}
	if err != bug.ErrBugNotExist {
		return nil, err
	}

	// if bug was never imported
	cleanText, err := text.Cleanup(issue.Description)
	if err != nil {
		return nil, err
	}

	// create bug
	b, _, err = repo.NewBugRaw(
		author,
		issue.CreatedAt.Unix(),
		issue.Title,
		cleanText,
		nil,
		map[string]string{
			core.MetaKeyOrigin:   target,
			metaKeyGitlabId:      parseID(issue.IID),
			metaKeyGitlabUrl:     issue.WebURL,
			metaKeyGitlabProject: gi.conf[keyProjectID],
			metaKeyGitlabBaseUrl: gi.conf[keyGitlabBaseUrl],
		},
	)

	if err != nil {
		return nil, err
	}

	// importing a new bug
	gi.out <- core.NewImportBug(b.Id())

	return b, nil
}

func (gi *gitlabImporter) ensureNote(repo *cache.RepoCache, b *cache.BugCache, note *gitlab.Note) error {
	gitlabID := parseID(note.ID)

	id, errResolve := b.ResolveOperationWithMetadata(metaKeyGitlabId, gitlabID)
	if errResolve != nil && errResolve != cache.ErrNoMatchingOp {
		return errResolve
	}

	// ensure issue author
	author, err := gi.ensurePerson(repo, note.Author.ID)
	if err != nil {
		return err
	}

	noteType, body := GetNoteType(note)
	switch noteType {
	case NOTE_CLOSED:
		if errResolve == nil {
			return nil
		}

		op, err := b.CloseRaw(
			author,
			note.CreatedAt.Unix(),
			map[string]string{
				metaKeyGitlabId: gitlabID,
			},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(op.Id())

	case NOTE_REOPENED:
		if errResolve == nil {
			return nil
		}

		op, err := b.OpenRaw(
			author,
			note.CreatedAt.Unix(),
			map[string]string{
				metaKeyGitlabId: gitlabID,
			},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(op.Id())

	case NOTE_DESCRIPTION_CHANGED:
		issue := gi.iterator.IssueValue()

		firstComment := b.Snapshot().Comments[0]
		// since gitlab doesn't provide the issue history
		// we should check for "changed the description" notes and compare issue texts
		// TODO: Check only one time and ignore next 'description change' within one issue
		if errResolve == cache.ErrNoMatchingOp && issue.Description != firstComment.Message {
			// comment edition
			op, err := b.EditCommentRaw(
				author,
				note.UpdatedAt.Unix(),
				firstComment.Id(),
				issue.Description,
				map[string]string{
					metaKeyGitlabId: gitlabID,
				},
			)
			if err != nil {
				return err
			}

			gi.out <- core.NewImportTitleEdition(op.Id())
		}

	case NOTE_COMMENT:
		cleanText, err := text.Cleanup(body)
		if err != nil {
			return err
		}

		// if we didn't import the comment
		if errResolve == cache.ErrNoMatchingOp {

			// add comment operation
			op, err := b.AddCommentRaw(
				author,
				note.CreatedAt.Unix(),
				cleanText,
				nil,
				map[string]string{
					metaKeyGitlabId: gitlabID,
				},
			)
			if err != nil {
				return err
			}
			gi.out <- core.NewImportComment(op.Id())
			return nil
		}

		// if comment was already exported

		// search for last comment update
		comment, err := b.Snapshot().SearchComment(id)
		if err != nil {
			return err
		}

		// compare local bug comment with the new note body
		if comment.Message != cleanText {
			// comment edition
			op, err := b.EditCommentRaw(
				author,
				note.UpdatedAt.Unix(),
				comment.Id(),
				cleanText,
				nil,
			)

			if err != nil {
				return err
			}
			gi.out <- core.NewImportCommentEdition(op.Id())
		}

		return nil

	case NOTE_TITLE_CHANGED:
		// title change events are given new notes
		if errResolve == nil {
			return nil
		}

		op, err := b.SetTitleRaw(
			author,
			note.CreatedAt.Unix(),
			body,
			map[string]string{
				metaKeyGitlabId: gitlabID,
			},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportTitleEdition(op.Id())

	case NOTE_UNKNOWN,
		NOTE_ASSIGNED,
		NOTE_UNASSIGNED,
		NOTE_CHANGED_MILESTONE,
		NOTE_REMOVED_MILESTONE,
		NOTE_CHANGED_DUEDATE,
		NOTE_REMOVED_DUEDATE,
		NOTE_LOCKED,
		NOTE_UNLOCKED,
		NOTE_MENTIONED_IN_ISSUE,
		NOTE_MENTIONED_IN_MERGE_REQUEST:

		return nil

	default:
		panic("unhandled note type")
	}

	return nil
}

func (gi *gitlabImporter) ensureLabelEvent(repo *cache.RepoCache, b *cache.BugCache, labelEvent *gitlab.LabelEvent) error {
	_, err := b.ResolveOperationWithMetadata(metaKeyGitlabId, parseID(labelEvent.ID))
	if err != cache.ErrNoMatchingOp {
		return err
	}

	// ensure issue author
	author, err := gi.ensurePerson(repo, labelEvent.User.ID)
	if err != nil {
		return err
	}

	switch labelEvent.Action {
	case "add":
		_, err = b.ForceChangeLabelsRaw(
			author,
			labelEvent.CreatedAt.Unix(),
			[]string{labelEvent.Label.Name},
			nil,
			map[string]string{
				metaKeyGitlabId: parseID(labelEvent.ID),
			},
		)

	case "remove":
		_, err = b.ForceChangeLabelsRaw(
			author,
			labelEvent.CreatedAt.Unix(),
			nil,
			[]string{labelEvent.Label.Name},
			map[string]string{
				metaKeyGitlabId: parseID(labelEvent.ID),
			},
		)

	default:
		err = fmt.Errorf("unexpected label event action")
	}

	return err
}

func (gi *gitlabImporter) ensurePerson(repo *cache.RepoCache, id int) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(metaKeyGitlabId, strconv.Itoa(id))
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}

	user, _, err := gi.client.Users.GetUser(id)
	if err != nil {
		return nil, err
	}

	i, err = repo.NewIdentityRaw(
		user.Name,
		user.PublicEmail,
		user.AvatarURL,
		map[string]string{
			// because Gitlab
			metaKeyGitlabId:    strconv.Itoa(id),
			metaKeyGitlabLogin: user.Username,
		},
	)
	if err != nil {
		return nil, err
	}

	gi.out <- core.NewImportIdentity(i.Id())
	return i, nil
}

func parseID(id int) string {
	return fmt.Sprintf("%d", id)
}
