package bug

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

// TODO: move to entity/dag?

func TestValidate(t *testing.T) {
	repo := repository.NewMockRepoClock()

	makeIdentity := func(t *testing.T, name, email string) *identity.Identity {
		i, err := identity.NewIdentity(repo, name, email)
		require.NoError(t, err)
		return i
	}

	rene := makeIdentity(t, "René Descartes", "rene@descartes.fr")

	unix := time.Now().Unix()

	good := []Operation{
		NewCreateOp(rene, unix, "title", "message", nil),
		NewSetTitleOp(rene, unix, "title2", "title1"),
		NewAddCommentOp(rene, unix, "message2", nil),
		NewSetStatusOp(rene, unix, common.ClosedStatus),
		NewLabelChangeOperation(rene, unix, []common.Label{"added"}, []common.Label{"removed"}),
	}

	for _, op := range good {
		if err := op.Validate(); err != nil {
			t.Fatal(err)
		}
	}

	bad := []Operation{
		// opbase
		NewSetStatusOp(makeIdentity(t, "", "rene@descartes.fr"), unix, common.ClosedStatus),
		NewSetStatusOp(makeIdentity(t, "René Descartes\u001b", "rene@descartes.fr"), unix, common.ClosedStatus),
		NewSetStatusOp(makeIdentity(t, "René Descartes", "rene@descartes.fr\u001b"), unix, common.ClosedStatus),
		NewSetStatusOp(makeIdentity(t, "René \nDescartes", "rene@descartes.fr"), unix, common.ClosedStatus),
		NewSetStatusOp(makeIdentity(t, "René Descartes", "rene@\ndescartes.fr"), unix, common.ClosedStatus),
		&CreateOperation{OpBase: dag.NewOpBase(CreateOp, rene, 0),
			Title:   "title",
			Message: "message",
		},

		NewCreateOp(rene, unix, "multi\nline", "message", nil),
		NewCreateOp(rene, unix, "title", "message", []repository.Hash{repository.Hash("invalid")}),
		NewCreateOp(rene, unix, "title\u001b", "message", nil),
		NewCreateOp(rene, unix, "title", "message\u001b", nil),
		NewSetTitleOp(rene, unix, "multi\nline", "title1"),
		NewSetTitleOp(rene, unix, "title", "multi\nline"),
		NewSetTitleOp(rene, unix, "title\u001b", "title2"),
		NewSetTitleOp(rene, unix, "title", "title2\u001b"),
		NewAddCommentOp(rene, unix, "message\u001b", nil),
		NewAddCommentOp(rene, unix, "message", []repository.Hash{repository.Hash("invalid")}),
		NewSetStatusOp(rene, unix, 1000),
		NewSetStatusOp(rene, unix, 0),
		NewLabelChangeOperation(rene, unix, []common.Label{}, []common.Label{}),
		NewLabelChangeOperation(rene, unix, []common.Label{"multi\nline"}, []common.Label{}),
	}

	for i, op := range bad {
		if err := op.Validate(); err == nil {
			t.Fatal("validation should have failed", i, op)
		}
	}
}

func TestUnknownOpUnmarshaler(t *testing.T) {
	// An operation type that no version of this client knows about should be
	// preserved as an UnknownOperation rather than causing a panic.
	raw, err := json.Marshal(struct {
		Type      dag.OperationType `json:"type"`
		Timestamp int64             `json:"timestamp"`
		Nonce     []byte            `json:"nonce"`
		Extra     string            `json:"future_field"`
	}{
		Type:      dag.OperationType(999),
		Timestamp: time.Now().Unix(),
		Nonce:     make([]byte, 20),
		Extra:     "preserved",
	})
	require.NoError(t, err)

	op, err := operationUnmarshaler(t.Context(), raw, nil)
	require.NoError(t, err)

	unknown, ok := op.(*dag.UnknownOperation[*Snapshot])
	require.True(t, ok, "expected *dag.UnknownOperation[*Snapshot]")
	require.JSONEq(t, string(raw), string(unknown.RawJSON))
}

func TestUnknownOpMarshalRoundTrip(t *testing.T) {
	// Fields from an unknown operation (e.g. from a future client) must survive
	// a serialize/deserialize cycle without data loss.
	raw, err := json.Marshal(struct {
		Type      dag.OperationType `json:"type"`
		Timestamp int64             `json:"timestamp"`
		Nonce     []byte            `json:"nonce"`
		Extra     string            `json:"future_field"`
	}{
		Type:      dag.OperationType(999),
		Timestamp: time.Now().Unix(),
		Nonce:     make([]byte, 20),
		Extra:     "preserved",
	})
	require.NoError(t, err)

	op, err := operationUnmarshaler(t.Context(), raw, nil)
	require.NoError(t, err)

	marshaled, err := json.Marshal(op)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(marshaled))
}

func TestMetadata(t *testing.T) {
	repo := repository.NewMockRepoClock()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	op := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)

	op.SetMetadata("key", "value")

	val, ok := op.GetMetadata("key")
	require.True(t, ok)
	require.Equal(t, val, "value")
}

func TestID(t *testing.T) {
	repo := repotest.NewGoGitRepo(t, false)

	repos := []repository.Repo{
		repository.NewMockRepo(),
		repo,
	}

	for _, repo := range repos {
		rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
		require.NoError(t, err)
		err = rene.Commit(t.Context(), repo)
		require.NoError(t, err)

		b, op, err := Create(rene, time.Now().Unix(), "title", "message", nil, nil)
		require.NoError(t, err)

		id1 := op.Id()
		require.NoError(t, id1.Validate())

		err = b.Commit(t.Context(), repo)
		require.NoError(t, err)

		op2 := b.FirstOp()

		id2 := op2.Id()
		require.NoError(t, id2.Validate())
		require.Equal(t, id1, id2)

		b2, err := Read(t.Context(), repo, b.Id())
		require.NoError(t, err)

		op3 := b2.FirstOp()

		id3 := op3.Id()
		require.NoError(t, id3.Validate())
		require.Equal(t, id1, id3)
	}
}

func TestValidityExamples(t *testing.T) {
	repo := repository.NewMockRepoClock()
	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	// the validity examples of testdata/bug.json in the spec, each completed with
	// the common members it doesn't set
	common := map[string]json.RawMessage{
		"timestamp": json.RawMessage(`1609459200`),
		"nonce":     json.RawMessage(`"AAAAAAAAAAAAAAAAAAAAAAAAAAAA"`),
	}
	cases := []struct {
		name  string
		op    string
		valid bool
	}{
		{"files is omitted: no file", `{"type":1,"title":"crash","message":""}`, true},
		{"null reads as absent: no file", `{"type":1,"title":"crash","message":"","files":null}`, true},
		{"an isolate that is closed", `{"type":1,"title":"\u05d0 \u2067b\u2069","message":""}`, true},
		{"a comment may be blank", `{"type":3,"message":""}`, true},
		{"a comment is multi-line text: U+2028 is allowed", `{"type":3,"message":"line one\u2028line two"}`, true},
		{"removed is omitted: no label removed", `{"type":5,"added":["bug"]}`, true},
		{"a nonce of 20 bytes, padded", `{"type":4,"status":2,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAA="}`, true},
		{"a git hash of 40 characters, SHA-1", `{"type":3,"message":"","files":["e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"]}`, true},
		{"a metadata value is multi-line text", `{"type":1,"title":"crash","message":"","metadata":{"github-id":"I_1","note":"a\nb"}}`, true},
		{"the unused bits of a nonce are ignored", `{"type":4,"status":2,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAB="}`, true},
		{"without a title", `{"type":1,"message":""}`, false},
		{"without a message", `{"type":1,"title":"crash"}`, false},
		{"a message that is null is absent", `{"type":1,"title":"crash","message":null}`, false},
		{"without was", `{"type":2,"title":"crash on start"}`, false},
		{"without a message", `{"type":3}`, false},
		{"without a status", `{"type":4}`, false},
		{"a timestamp is not 0", `{"type":4,"status":2,"timestamp":0}`, false},
		{"a timestamp that is null is absent", `{"type":4,"status":2,"timestamp":null}`, false},
		{"a nonce that is null is absent", `{"type":4,"status":2,"nonce":null}`, false},
		{"a title that is null is absent", `{"type":1,"title":null,"message":""}`, false},
		{"a blank title", `{"type":1,"title":"   ","message":""}`, false},
		{"a title of a Hangul filler is blank", `{"type":1,"title":"\u3164","message":""}`, false},
		{"a title is text: no tab", `{"type":1,"title":"a\tb","message":""}`, false},
		{"a title is text: no line separator", `{"type":1,"title":"a\u2028b","message":""}`, false},
		{"an isolate left open", `{"type":1,"title":"a\u2067b","message":""}`, false},
		{"an element of an array is never null", `{"type":1,"title":"crash","message":"","files":[null]}`, false},
		{"a right-to-left override", `{"type":3,"message":"a\u202eb"}`, false},
		{"a nonce without its padding", `{"type":4,"status":2,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAA"}`, false},
		{"a nonce in the URL-safe alphabet", `{"type":4,"status":2,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAA-"}`, false},
		{"a nonce with a line break", `{"type":4,"status":2,"nonce":"AAAAAAAAAAAAAA\nAAAAAAAAAAAAAA"}`, false},
		{"a nonce of 19 bytes", `{"type":4,"status":2,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAA=="}`, false},
		{"a git hash has 40 or 64 characters", `{"type":3,"message":"","files":["e69de29b"]}`, false},
		{"a git hash is in lowercase", `{"type":3,"message":"","files":["E69DE29BB2D1D6434B8B29AE775AD8C2E48C5391"]}`, false},
		{"an operation ID is in lowercase", `{"type":8,"target":"AD3FE2D371F680AE2B9D012631FE2602EBF99EFEDCB8FDCA7A926D65BA85056E"}`, false},
		{"neither added nor removed", `{"type":5}`, false},
		{"a blank label", `{"type":5,"added":[" "]}`, false},
		{"an element of an array is never null", `{"type":5,"added":[null]}`, false},
		{"a metadata key is text: no line break", `{"type":1,"title":"crash","message":"","metadata":{"a\nb":"x"}}`, false},
		{"a metadata value holds no NUL", `{"type":8,"target":"ad3fe2d371f680ae2b9d012631fe2602ebf99efedcb8fdca7a926d65ba85056e","new_metadata":{"k":"a\u0000b"}}`, false},
		{"a value that is null in a map", `{"type":8,"target":"ad3fe2d371f680ae2b9d012631fe2602ebf99efedcb8fdca7a926d65ba85056e","new_metadata":{"github-id":null}}`, false},
		// not in the spec
		{"a value that is null in metadata", `{"type":3,"message":"","metadata":{"github-id":null}}`, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var members map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(c.op), &members))
			for name, value := range common {
				if _, ok := members[name]; !ok {
					members[name] = value
				}
			}
			raw, err := json.Marshal(members)
			require.NoError(t, err)

			op, err := operationUnmarshaler(t.Context(), raw, nil)
			if err == nil {
				// a read takes the author from the operation pack: give the
				// operation one, keeping what it decoded
				base := reflect.ValueOf(op).Elem().FieldByName("OpBase").Addr().Interface().(*dag.OpBase)
				authored := dag.NewOpBase(base.OperationType, rene, base.UnixTime)
				authored.Nonce, authored.Metadata = base.Nonce, base.Metadata
				*base = authored
				err = op.Validate()
			}
			if c.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
