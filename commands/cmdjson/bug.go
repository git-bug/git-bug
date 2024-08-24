package cmdjson

import (
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
)

type BugSnapshot struct {
	Id           string       `json:"id"`
	HumanId      string       `json:"human_id"`
	CreateTime   Time         `json:"create_time"`
	EditTime     Time         `json:"edit_time"`
	Status       string       `json:"status"`
	Labels       []bug.Label  `json:"labels"`
	Title        string       `json:"title"`
	Author       Identity     `json:"author"`
	Actors       []Identity   `json:"actors"`
	Participants []Identity   `json:"participants"`
	Comments     []BugComment `json:"comments"`
}

func NewBugSnapshot(snap *bug.Snapshot) BugSnapshot {
	jsonBug := BugSnapshot{
		Id:         snap.Id().String(),
		HumanId:    snap.Id().Human(),
		CreateTime: NewTime(snap.CreateTime, 0),
		EditTime:   NewTime(snap.EditTime(), 0),
		Status:     snap.Status.String(),
		Labels:     snap.Labels,
		Title:      snap.Title,
		Author:     NewIdentity(snap.Author),
	}

	jsonBug.Actors = make([]Identity, len(snap.Actors))
	for i, element := range snap.Actors {
		jsonBug.Actors[i] = NewIdentity(element)
	}

	jsonBug.Participants = make([]Identity, len(snap.Participants))
	for i, element := range snap.Participants {
		jsonBug.Participants[i] = NewIdentity(element)
	}

	jsonBug.Comments = make([]BugComment, len(snap.Comments))
	for i, comment := range snap.Comments {
		jsonBug.Comments[i] = NewBugComment(comment)
	}

	return jsonBug
}

type BugComment struct {
	Id      string   `json:"id"`
	HumanId string   `json:"human_id"`
	Author  Identity `json:"author"`
	Message string   `json:"message"`
}

func NewBugComment(comment bug.Comment) BugComment {
	return BugComment{
		Id:      comment.CombinedId().String(),
		HumanId: comment.CombinedId().Human(),
		Author:  NewIdentity(comment.Author),
		Message: comment.Message,
	}
}

type BugExcerpt struct {
	Id         string `json:"id"`
	HumanId    string `json:"human_id"`
	CreateTime Time   `json:"create_time"`
	EditTime   Time   `json:"edit_time"`

	Status       string      `json:"status"`
	Labels       []bug.Label `json:"labels"`
	Title        string      `json:"title"`
	Actors       []Identity  `json:"actors"`
	Participants []Identity  `json:"participants"`
	Author       Identity    `json:"author"`

	Comments int               `json:"comments"`
	Metadata map[string]string `json:"metadata"`
}

func NewBugExcerpt(backend *cache.RepoCache, excerpt *cache.BugExcerpt) (BugExcerpt, error) {
	jsonBug := BugExcerpt{
		Id:         excerpt.Id().String(),
		HumanId:    excerpt.Id().Human(),
		CreateTime: NewTime(excerpt.CreateTime(), excerpt.CreateLamportTime),
		EditTime:   NewTime(excerpt.EditTime(), excerpt.EditLamportTime),
		Status:     excerpt.Status.String(),
		Labels:     excerpt.Labels,
		Title:      excerpt.Title,
		Comments:   excerpt.LenComments,
		Metadata:   excerpt.CreateMetadata,
	}

	author, err := backend.Identities().ResolveExcerpt(excerpt.AuthorId)
	if err != nil {
		return BugExcerpt{}, err
	}
	jsonBug.Author = NewIdentityFromExcerpt(author)

	jsonBug.Actors = make([]Identity, len(excerpt.Actors))
	for i, element := range excerpt.Actors {
		actor, err := backend.Identities().ResolveExcerpt(element)
		if err != nil {
			return BugExcerpt{}, err
		}
		jsonBug.Actors[i] = NewIdentityFromExcerpt(actor)
	}

	jsonBug.Participants = make([]Identity, len(excerpt.Participants))
	for i, element := range excerpt.Participants {
		participant, err := backend.Identities().ResolveExcerpt(element)
		if err != nil {
			return BugExcerpt{}, err
		}
		jsonBug.Participants[i] = NewIdentityFromExcerpt(participant)
	}

	return jsonBug, nil
}
