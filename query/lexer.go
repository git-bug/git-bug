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
		kind:         tokenKindKVV,
		qualifier:    qualifier,
		subQualifier: subQualifier,
		value:        value,
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
	fields, err := splitFunc(query, unicode.IsSpace)
	if err != nil {
		return nil, err
	}

	var tokens []token
	for _, field := range fields {
		chunks, err := splitFunc(field, func(r rune) bool { return r == ':' })
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(field, ":") || strings.HasSuffix(field, ":") {
			return nil, fmt.Errorf("empty qualifier or value")
		}

		// pre-process chunks
		for i, chunk := range chunks {
			if len(chunk) == 0 {
				return nil, fmt.Errorf("empty qualifier or value")
			}
			chunks[i] = removeQuote(chunk)
		}

		switch len(chunks) {
		case 1: // full text search
			tokens = append(tokens, newTokenSearch(chunks[0]))

		case 2: // KV
			tokens = append(tokens, newTokenKV(chunks[0], chunks[1]))

		case 3: // KVV
			tokens = append(tokens, newTokenKVV(chunks[0], chunks[1], chunks[2]))

		default:
			return nil, fmt.Errorf("can't tokenize \"%s\": too many separators", field)
		}
	}
	return tokens, nil
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

// split the input into chunks by splitting according to separatorFunc but respecting
// quotes
func splitFunc(input string, separatorFunc func(r rune) bool) ([]string, error) {
	lastQuote := rune(0)
	inQuote := false

	// return true if it's part of a chunk, or false if it's a rune that delimit one, as determined by the separatorFunc.
	isChunk := func(r rune) bool {
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
			return !separatorFunc(r)
		}
	}

	var result []string
	var chunk strings.Builder
	for _, r := range input {
		if isChunk(r) {
			chunk.WriteRune(r)
		} else {
			if chunk.Len() > 0 {
				result = append(result, chunk.String())
				chunk.Reset()
			}
		}
	}

	if inQuote {
		return nil, fmt.Errorf("unmatched quote")
	}

	if chunk.Len() > 0 {
		result = append(result, chunk.String())
	}

	return result, nil
}

func isQuote(r rune) bool {
	return r == '"' || r == '\''
}
