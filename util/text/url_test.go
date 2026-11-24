package text

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidHTTPSURL(t *testing.T) {
	// the examples of testdata/url.json in the spec
	cases := []struct {
		name  string
		url   string
		valid bool
	}{
		{"a GitHub avatar", "https://avatars.githubusercontent.com/u/1234567?v=4", true},
		{"a Gravatar avatar, with a query", "https://secure.gravatar.com/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346?s=80&d=identicon", true},
		{"an escape and a fragment", "https://example.com/a%20b.png#x", true},
		{"a host alone", "https://example.com", true},
		{"an internationalized name, written in punycode", "https://xn--bcher-kva.example/a.png", true},
		{"a hyphen, more than two labels", "https://a-b.example.co.uk/a.png", true},
		{"a label of 63 characters", "https://" + strings.Repeat("a", 63) + ".example.com/a.png", true},
		{"2048 characters", "https://example.com/" + strings.Repeat("a", 2028), true},
		{"http", "http://example.com/a.png", false},
		{"the scheme is written in lowercase", "HTTPS://example.com/a.png", false},
		{"a relative URL", "/api/graphql", false},
		{"a data URL", "data:image/png;base64,iVBORw0KGgo=", false},
		{"a file URL", "file:///etc/passwd", false},
		{"without a host", "https://", false},
		{"user information before the host", "https://user@example.com/a.png", false},
		{"a space", "https://exa mple.com/a.png", false},
		{"a space in the path", "https://example.com/a b.png", false},
		{"an escape without two hexadecimal digits", "https://example.com/%zz", false},
		{"a character outside ASCII", "https://example.com/\u00e9.png", false},
		{"a port", "https://example.com:8443/a.png", false},
		{"a port, even the default one", "https://example.com:443/a.png", false},
		{"an IPv4 address", "https://127.0.0.1/a.png", false},
		{"a link-local IPv4 address", "https://169.254.169.254/latest/meta-data/", false},
		{"127.0.0.1 as a single number, read as an IPv4 address by many parsers", "https://2130706433/a.png", false},
		{"127.0.0.1 in hexadecimal and shortened, read as an IPv4 address by many parsers", "https://0x7f.1/a.png", false},
		{"an IPv6 address", "https://[::1]/a.png", false},
		{"a name of a single label", "https://localhost/a.png", false},
		{"a last label starting with a digit", "https://example.123/a.png", false},
		{"an empty label", "https://a..example.com/a.png", false},
		{"a trailing dot, an empty last label", "https://example.com./a.png", false},
		{"a label of 64 characters", "https://" + strings.Repeat("a", 64) + ".example.com/a.png", false},
		{"a backslash, read as a slash by some parsers", "https://example.com\\@evil.example/a.png", false},
		{"2049 characters: an avatar URL has at most 2048", "https://example.com/" + strings.Repeat("a", 2029), false},
		{"empty", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.valid, ValidHTTPSURL(c.url), c.url)
		})
	}
}
