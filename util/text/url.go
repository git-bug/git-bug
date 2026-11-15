package text

import (
	"strings"
)

// maxURLLength is the length of an HTTPS URL at most.
const maxURLLength = 2048

// ValidHTTPSURL tells if the string is an HTTPS URL, one a client may fetch on
// its own: "https://", a host named in full, then optionally a path, a query or
// a fragment, in ASCII. The rule is exact, so it holds no port, no IP address and
// no name of a single label. See values.md §9 in the spec.
func ValidHTTPSURL(s string) bool {
	// the rule allows ASCII only, so the length in bytes is the one in code points
	if len(s) > maxURLLength {
		return false
	}
	rest, ok := strings.CutPrefix(s, "https://")
	if !ok {
		return false
	}
	host, rest := rest, ""
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host, rest = host[:i], host[i:]
	}
	return validHost(host) && validURLRest(rest)
}

// CleanupHTTPSURL returns the string if it is an HTTPS URL (see ValidHTTPSURL),
// or else the empty string: for a URL from elsewhere, such as a bridge, which
// can be left out but not kept as it is.
func CleanupHTTPSURL(s string) string {
	if ValidHTTPSURL(s) {
		return s
	}
	return ""
}

// validHost tells if the string is two or more labels separated by ".", each of
// 1 to 63 ASCII letters, digits and "-", the last one starting with a letter.
func validHost(host string) bool {
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if len(label) < 1 || len(label) > 63 {
			return false
		}
		for i := 0; i < len(label); i++ {
			if !isASCIILetter(label[i]) && !isASCIIDigit(label[i]) && label[i] != '-' {
				return false
			}
		}
	}
	return isASCIILetter(labels[len(labels)-1][0])
}

// validURLRest tells if the string, empty or starting with "/", "?" or "#", holds
// only the characters allowed after the host, each "%" followed by two
// hexadecimal digits.
func validURLRest(rest string) bool {
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		switch {
		case c == '%':
			if i+2 >= len(rest) || !isHexDigit(rest[i+1]) || !isHexDigit(rest[i+2]) {
				return false
			}
			i += 2
		case isASCIILetter(c), isASCIIDigit(c), strings.IndexByte("-._~:/?#[]@!$&'()*+,;=", c) >= 0:
		default:
			return false
		}
	}
	return true
}

func isASCIILetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }

func isHexDigit(c byte) bool {
	return isASCIIDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}
