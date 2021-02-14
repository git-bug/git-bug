package query

import (
	"fmt"
	"strings"
	"unicode"
)

type tokenKind int

const (
	_ tokenKind = iota
	tokenKindKV
	tokenKindKVV
	tokenKindSearch
)

type token struct {
	kind tokenKind

	// KV and KVV
	qualifier string
	value     string

	// KVV only
	subQualifier string

	// Search
	term string
}

func newTokenKV(qualifier, value string) token {
	return token{
		kind:      tokenKindKV,
		qualifier: qualifier,
		value:     value,
	}
}

func newTokenKVV(qualifier, subQualifier, value string) token {
	return token{
		kind:      tokenKindKVV,
		qualifier: qualifier,
		subQualifier: subQualifier,
		value:     value,
	}
}

func newTokenSearch(term string) token {
	return token{
		kind: tokenKindSearch,
		term: term,
	}
}

// tokenize parse and break a input into tokens ready to be
// interpreted later by a parser to get the semantic.
func tokenize(query string) ([]token, error) {
	fields, err := splitQuery(query)
	if err != nil {
		return nil, err
	}

	var tokens []token
	for _, field := range fields {
		// Split using ':' as separator, but separators inside '"' don't count.
		quoted := false
		split := strings.FieldsFunc(field, func(r rune) bool {
			if r == '"' {
				quoted = !quoted
			}
			return !quoted && r == ':'
		})
		if (strings.HasPrefix(field, ":")) {
			split = append([]string{""}, split...)
		}
		if (strings.HasSuffix(field, ":")) {
			split = append(split, "")
		}
		if (quoted) {
			return nil, fmt.Errorf("can't tokenize \"%s\": unmatched quote", field)
		}

		// full text search
		if len(split) == 1 {
			tokens = append(tokens, newTokenSearch(removeQuote(field)))
			continue
		}

		if len(split) > 3 {
			return nil, fmt.Errorf("can't tokenize \"%s\": too many separators", field)
		}

		if len(split[0]) == 0 {
			return nil, fmt.Errorf("can't tokenize \"%s\": empty qualifier", field)
		}

		if len(split) == 2 {
			if len(split[1]) == 0 {
				return nil, fmt.Errorf("empty value for qualifier \"%s\"", split[0])
			}

			tokens = append(tokens, newTokenKV(split[0], removeQuote(split[1])))
		} else {
			if len(split[1]) == 0 {
				return nil, fmt.Errorf("empty sub-qualifier for qualifier \"%s\"", split[0])
			}

			if len(split[2]) == 0 {
				return nil, fmt.Errorf("empty value for qualifier \"%s:%s\"", split[0], split[1])
			}

			tokens = append(tokens, newTokenKVV(split[0], removeQuote(split[1]), removeQuote(split[2])))
		}
	}
	return tokens, nil
}

// split the query into chunks by splitting on whitespaces but respecting
// quotes
func splitQuery(query string) ([]string, error) {
	lastQuote := rune(0)
	inQuote := false

	isToken := func(r rune) bool {
		switch {
		case !inQuote && isQuote(r):
			lastQuote = r
			inQuote = true
			return true
		case inQuote && r == lastQuote:
			lastQuote = rune(0)
			inQuote = false
			return true
		case inQuote:
			return true
		default:
			return !unicode.IsSpace(r)
		}
	}

	var result []string
	var token strings.Builder
	for _, r := range query {
		if isToken(r) {
			token.WriteRune(r)
		} else {
			if token.Len() > 0 {
				result = append(result, token.String())
				token.Reset()
			}
		}
	}

	if inQuote {
		return nil, fmt.Errorf("unmatched quote")
	}

	if token.Len() > 0 {
		result = append(result, token.String())
	}

	return result, nil
}

func isQuote(r rune) bool {
	return r == '"' || r == '\''
}

func removeQuote(field string) string {
	runes := []rune(field)
	if len(runes) >= 2 {
		r1 := runes[0]
		r2 := runes[len(runes)-1]

		if r1 == r2 && isQuote(r1) {
			return string(runes[1 : len(runes)-1])
		}
	}
	return field
}
