package cmdjson

import (
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/board"
	"github.com/git-bug/git-bug/entity"
)

type BoardSnapshot struct {
	Id         string `json:"id"`
	HumanId    string `json:"human_id"`
	CreateTime Time   `json:"create_time"`
	EditTime   Time   `json:"edit_time"`

	Title       string        `json:"title"`
	Description string        `json:"description"`
	Actors      []Identity    `json:"participants"`
	Columns     []BoardColumn `json:"columns"`
}

func NewBoardSnapshot(backend *cache.RepoCache, snapshot *board.Snapshot) (BoardSnapshot, error) {
	jsonBoard := BoardSnapshot{
		Id:          snapshot.Id().String(),
		HumanId:     snapshot.Id().Human(),
		CreateTime:  NewTime(snapshot.CreateTime, 0),
		EditTime:    NewTime(snapshot.EditTime(), 0),
		Title:       snapshot.Title,
		Description: snapshot.Description,
	}

	jsonBoard.Actors = make([]Identity, len(snapshot.Actors))
	for i, element := range snapshot.Actors {
		jsonBoard.Actors[i] = NewIdentity(element)
	}

	jsonBoard.Columns = make([]BoardColumn, len(snapshot.Columns))
	for i, column := range snapshot.Columns {
		jsonColumn, err := NewBoardColumn(backend, column)
		if err != nil {
			return BoardSnapshot{}, err
		}
		jsonBoard.Columns[i] = jsonColumn
	}

	return jsonBoard, nil
}

type BoardColumn struct {
	Id      string `json:"id"`
	HumanId string `json:"human_id"`
	Name    string `json:"name"`
	Items   []any  `json:"items"`
}

func NewBoardColumn(backend *cache.RepoCache, column *board.Column) (BoardColumn, error) {
	jsonColumn := BoardColumn{
		Id:      column.CombinedId.String(),
		HumanId: column.CombinedId.Human(),
		Name:    column.Name,
	}
	jsonColumn.Items = make([]any, len(column.Items))
	for j, item := range column.Items {
		switch item := item.(type) {
		case *board.Draft:
			jsonColumn.Items[j] = NewBoardDraftItem(item)
		case *board.BugItem:
			jsonItem, err := NewBoardBugItem(backend, item)
			if err != nil {
				return BoardColumn{}, err
			}
			jsonColumn.Items[j] = jsonItem
		default:
			panic("unknown item type")
		}
	}
	return jsonColumn, nil
}

type BoardDraftItem struct {
	Type    string   `json:"type"`
	Id      string   `json:"id"`
	HumanId string   `json:"human_id"`
	Author  Identity `json:"author"`
	Title   string   `json:"title"`
}

func NewBoardDraftItem(item *board.Draft) BoardDraftItem {
	return BoardDraftItem{
		Type:    "draft",
		Id:      item.CombinedId().String(),
		HumanId: item.CombinedId().Human(),
		Author:  NewIdentity(item.Author()),
		Title:   item.Title(),
	}
}

// BoardBugItem is a board item referring to a bug. The bug's fields are missing
// when the bug is not available locally.
type BoardBugItem struct {
	Type    string    `json:"type"`
	Id      string    `json:"id"`
	HumanId string    `json:"human_id"`
	BugId   string    `json:"bug_id"`
	Author  *Identity `json:"author,omitempty"`
	Title   string    `json:"title,omitempty"`
}

func NewBoardBugItem(backend *cache.RepoCache, item *board.BugItem) (BoardBugItem, error) {
	jsonItem := BoardBugItem{
		Type:    "bug",
		Id:      item.CombinedId().String(),
		HumanId: item.CombinedId().Human(),
		BugId:   item.BugId.String(),
	}

	excerpt, err := backend.Bugs().ResolveExcerpt(item.BugId)
	if entity.IsErrNotFound(err) {
		return jsonItem, nil
	}
	if err != nil {
		return BoardBugItem{}, err
	}
	jsonItem.Title = excerpt.Title

	author, err := backend.Identities().ResolveExcerpt(excerpt.AuthorId)
	if err != nil {
		return BoardBugItem{}, err
	}
	jsonAuthor := NewIdentityFromExcerpt(author)
	jsonItem.Author = &jsonAuthor

	return jsonItem, nil
}

type BoardExcerpt struct {
	Id         string `json:"id"`
	HumanId    string `json:"human_id"`
	CreateTime Time   `json:"create_time"`
	EditTime   Time   `json:"edit_time"`

	Title       string     `json:"title"`
	Description string     `json:"description"`
	Actors      []Identity `json:"participants"`

	Items    int               `json:"items"`
	Metadata map[string]string `json:"metadata"`
}

func NewBoardExcerpt(backend *cache.RepoCache, b *cache.BoardExcerpt) (BoardExcerpt, error) {
	jsonBoard := BoardExcerpt{
		Id:          b.Id().String(),
		HumanId:     b.Id().Human(),
		CreateTime:  NewTime(b.CreateTime(), b.CreateLamportTime),
		EditTime:    NewTime(b.EditTime(), b.EditLamportTime),
		Title:       b.Title,
		Description: b.Description,
		Items:       b.ItemCount,
		Metadata:    b.CreateMetadata,
	}

	jsonBoard.Actors = make([]Identity, len(b.Actors))
	for i, element := range b.Actors {
		participant, err := backend.Identities().ResolveExcerpt(element)
		if err != nil {
			return BoardExcerpt{}, err
		}
		jsonBoard.Actors[i] = NewIdentityFromExcerpt(participant)
	}
	return jsonBoard, nil
}
