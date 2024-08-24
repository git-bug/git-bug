package cmdjson

import (
	"time"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/util/lamport"
)

type Identity struct {
	Id      string `json:"id"`
	HumanId string `json:"human_id"`
	Name    string `json:"name"`
	Login   string `json:"login"`
}

func NewIdentity(i identity.Interface) Identity {
	return Identity{
		Id:      i.Id().String(),
		HumanId: i.Id().Human(),
		Name:    i.Name(),
		Login:   i.Login(),
	}
}

func NewIdentityFromExcerpt(excerpt *cache.IdentityExcerpt) Identity {
	return Identity{
		Id:      excerpt.Id().String(),
		HumanId: excerpt.Id().Human(),
		Name:    excerpt.Name,
		Login:   excerpt.Login,
	}
}

type Time struct {
	Timestamp int64        `json:"timestamp"`
	Time      time.Time    `json:"time"`
	Lamport   lamport.Time `json:"lamport,omitempty"`
}

func NewTime(t time.Time, l lamport.Time) Time {
	return Time{
		Timestamp: t.Unix(),
		Time:      t,
		Lamport:   l,
	}
}
