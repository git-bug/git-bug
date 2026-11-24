package jsondoc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsondoc(t *testing.T) {
	t.Run("spec examples", func(t *testing.T) {
		// the json examples of testdata/dag-entity.json in the spec, read as a pack
		// of bug operations
		type pack struct {
			Author struct {
				Id string `json:"id"`
			} `json:"author"`
			Ops []struct {
				Type      int               `json:"type"`
				Timestamp int64             `json:"timestamp"`
				Nonce     []byte            `json:"nonce"`
				Title     string            `json:"title"`
				Metadata  map[string]string `json:"metadata"`
			} `json:"ops"`
		}
		cases := []struct {
			name  string
			raw   string
			valid bool
		}{
			{"the reference: every rule followed", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\"}]}", true},
			{"an escaped surrogate pair: U+1F41B", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"bug \\ud83d\\udc1b\"}]}", true},
			{"U+1F41B written in UTF-8", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"bug \U0001f41b\"}]}", true},
			{"a high surrogate alone", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"a\\ud800b\"}]}", false},
			{"a low surrogate alone", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"a\\udc00b\"}]}", false},
			{"a low surrogate before a high one", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"a\\udc00\\ud800b\"}]}", false},
			{"two members of the same name", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\",\"title\":\"other\"}]}", false},
			{"two members of the same name, the second null", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\",\"title\":null}]}", false},
			{"two keys of the same name in a map", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\",\"metadata\":{\"k\":\"a\",\"k\":\"b\"}}]}", false},
			{"Title is an unknown member: the title is test bug", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\",\"Title\":\"other\"}]}", true},
			{"the byte FF, invalid in UTF-8", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"a}]}\xffb\"}]}", false},
			{"an overlong encoding of /, invalid in UTF-8", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"a}]}\xc0\xafb\"}]}", false},
			{"U+D800 encoded in UTF-8, invalid in UTF-8", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"a}]}\xed\xa0\x80b\"}]}", false},
			{"a byte order mark before the document", "\ufeff{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\"}]}", false},
			{"a timestamp of 2^53 - 1, the largest integer", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":9007199254740991,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\"}]}", true},
			{"a timestamp of 2^53, beyond the integers", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":9007199254740992,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\"}]}", false},
			{"a timestamp with a fraction", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1609459200.0,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\"}]}", false},
			{"a timestamp with an exponent", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[{\"type\":1,\"timestamp\":1.6e9,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAAAAAAAA\",\"title\":\"test bug\"}]}", false},
			{"an element of ops that is null", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[null]}", false},
			{"an element of ops that is not an object", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":[1]}", false},
			{"ops that is null is absent: the pack holds no operation, as the pack of a merge commit", "{\"author\":{\"id\":\"b29b7b2e0acb0a91dcf3c6c9d8e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7\"},\"ops\":null}", true},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				var p pack
				err := Unmarshal([]byte(c.raw), &p)
				if !c.valid {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				if len(p.Ops) > 0 && p.Ops[0].Type == 1 {
					// Title is an unknown member, not the title
					require.Contains(t, []string{"test bug", "bug \U0001F41B"}, p.Ops[0].Title)
				}
			})
		}
	})

	t.Run("document", func(t *testing.T) {
		var v any
		for _, raw := range []string{``, `{"a": 1} {"b": 2}`, `{"a": 1`, `[1, 2,]`} {
			require.Error(t, Unmarshal([]byte(raw), &v), raw)
		}
		require.NoError(t, Unmarshal([]byte(" {\"a\": [1, {}]}\n"), &v))
	})

	t.Run("integers", func(t *testing.T) {
		type clock uint64
		var v struct {
			Int   int64            `json:"int"`
			Small int8             `json:"small"`
			Times map[string]clock `json:"times"`
			List  []int            `json:"list"`
			Ptr   *int64           `json:"ptr"`
		}
		for _, raw := range []string{
			`{"int": 9007199254740991, "ptr": -9007199254740991, "times": {"a": 9007199254740991}, "list": [-1, 0, 1]}`,
			`{"small": -128}`,
		} {
			require.NoError(t, Unmarshal([]byte(raw), &v), raw)
		}
		for _, raw := range []string{
			`{"int": 9007199254740992}`,
			`{"int": -9007199254740992}`,
			`{"int": 18446744073709551615}`,
			`{"int": 1.0}`,
			`{"int": 1e3}`,
			`{"int": 1E3}`,
			`{"ptr": 9007199254740992}`,
			`{"times": {"a": 9007199254740992}}`,
			`{"times": {"a": 1.0}}`,
			`{"times": {"a": -1}}`,
			`{"list": [1, 9007199254740992]}`,
			`{"small": 128}`,
		} {
			require.Error(t, Unmarshal([]byte(raw), &v), raw)
		}
		// a number that isn't an integer is free where the format doesn't define
		// an integer
		var f struct {
			Float float64 `json:"float"`
		}
		require.NoError(t, Unmarshal([]byte(`{"float": 1e300, "unknown": 9007199254740992}`), &f))
	})

	t.Run("null", func(t *testing.T) {
		var v struct {
			Title    string            `json:"title"`
			Files    []string          `json:"files"`
			Nested   [][]string        `json:"nested"`
			Metadata map[string]string `json:"metadata"`
		}
		// a member that is null is absent
		require.NoError(t, Unmarshal([]byte(`{"title": null, "files": null, "metadata": null}`), &v))
		require.Equal(t, "", v.Title)
		require.Nil(t, v.Files)
		require.Nil(t, v.Metadata)

		// an array holding null is an error, at any depth
		for _, raw := range []string{`{"files": [null]}`, `{"files": ["a", null]}`, `{"nested": [["a"], [null]]}`} {
			require.ErrorContains(t, Unmarshal([]byte(raw), &v), "null in an array", raw)
		}
		require.NoError(t, Unmarshal([]byte(`{"files": [], "nested": [[]]}`), &v))
		require.Equal(t, []string{}, v.Files)

		// in an array of any Go type
		var a struct {
			Array [2]string         `json:"array"`
			Ptr   *[]string         `json:"ptr"`
			Raws  []json.RawMessage `json:"raws"`
		}
		for _, raw := range []string{`{"array": ["a", null]}`, `{"ptr": [null]}`, `{"raws": [{}, null]}`} {
			require.ErrorContains(t, Unmarshal([]byte(raw), &a), "null in an array", raw)
		}

		// a map holding null is an error, at any depth
		var m struct {
			Metadata map[string]string            `json:"metadata"`
			Nested   map[string]map[string]string `json:"nested"`
			Raws     map[string]json.RawMessage   `json:"raws"`
		}
		for _, raw := range []string{
			`{"metadata": {"k": null}}`,
			`{"metadata": {"a": "x", "k": null}}`,
			`{"nested": {"a": {"k": null}}}`,
			`{"raws": {"k": null}}`,
		} {
			require.ErrorContains(t, Unmarshal([]byte(raw), &m), "null in a map", raw)
		}
		require.NoError(t, Unmarshal([]byte(`{"metadata": {"a": "x"}, "nested": {"a": {}}, "raws": {"k": [null]}}`), &m))
		require.Equal(t, map[string]string{"a": "x"}, m.Metadata)
		require.Equal(t, map[string]map[string]string{"a": {}}, m.Nested)
		require.Equal(t, map[string]json.RawMessage{"k": json.RawMessage(`[null]`)}, m.Raws)
	})

	t.Run("bytes", func(t *testing.T) {
		type nonce struct {
			Nonce []byte `json:"nonce"`
		}

		t.Run("base64", func(t *testing.T) {
			for raw, expected := range map[string][]byte{
				`{"nonce": "AAAA"}`: {0, 0, 0},
				`{"nonce": "AAE="}`: {0, 1},
				`{"nonce": "AA=="}`: {0},
				`{"nonce": "+/+/"}`: {0xfb, 0xff, 0xbf},
				// unused bits are ignored
				`{"nonce": "AAF="}`: {0, 1},
				`{"nonce": "AB=="}`: {0},
			} {
				var v nonce
				require.NoError(t, Unmarshal([]byte(raw), &v), raw)
				require.Equal(t, expected, v.Nonce, raw)
			}
		})

		t.Run("not base64", func(t *testing.T) {
			for _, raw := range []string{
				// unpadded
				`{"nonce": "AAE"}`,
				`{"nonce": "AA"}`,
				// URL-safe alphabet
				`{"nonce": "-_-_"}`,
				// a line break, or any other character
				`{"nonce": "AA\nAA"}`,
				`{"nonce": "AA\r\nAA"}`,
				`{"nonce": "AA AA"}`,
				// never an array of numbers
				`{"nonce": [1, 2, 3]}`,
			} {
				var v nonce
				require.Error(t, Unmarshal([]byte(raw), &v), raw)
			}
		})
	})

	t.Run("names match exactly", func(t *testing.T) {
		var v struct {
			Title string `json:"title"`
		}
		require.NoError(t, Unmarshal([]byte(`{"Title": "other", "TITLE": "other"}`), &v))
		require.Equal(t, "", v.Title)
	})

	t.Run("own unmarshaler", func(t *testing.T) {
		// a value whose type has its own method is left to it: a raw message
		// keeps an array as it is
		var v struct {
			Raw  json.RawMessage   `json:"raw"`
			Raws []json.RawMessage `json:"raws"`
		}
		require.NoError(t, Unmarshal([]byte(`{"raw": [1, null, 2.5], "raws": [{"a": null}, [null]]}`), &v))
		require.Equal(t, `[1, null, 2.5]`, string(v.Raw))
		require.Equal(t, []json.RawMessage{json.RawMessage(`{"a": null}`), json.RawMessage(`[null]`)}, v.Raws)
	})

	t.Run("require", func(t *testing.T) {
		t.Run("present", func(t *testing.T) {
			// a required member may hold its zero value
			require.NoError(t, Require([]byte(`{"type": 3, "message": "", "count": 0}`), "message", "count"))
			require.NoError(t, Require([]byte(`{"type": 3}`)))
		})

		t.Run("absent or null", func(t *testing.T) {
			for _, raw := range []string{
				`{"type": 3, "count": 0}`,
				`{"type": 3, "message": null, "count": 0}`,
				`{"type": 3, "message": "", "count": null}`,
				`{}`,
				`null`,
			} {
				require.ErrorContains(t, Require([]byte(raw), "message", "count"), "missing member", raw)
			}
		})

		t.Run("names match exactly", func(t *testing.T) {
			require.ErrorContains(t, Require([]byte(`{"Message": ""}`), "message"), `missing member "message"`)
		})

		t.Run("top-level members only", func(t *testing.T) {
			require.Error(t, Require([]byte(`{"meta": {"message": ""}}`), "message"))
		})
	})

}
