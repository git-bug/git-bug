package repository

import (
	"strings"
	"unicode"

	"github.com/blevesearch/snowballstem"
	"github.com/blevesearch/snowballstem/english"
)

// textTokens returns the tokens of a text, as it is indexed and searched: its
// words lowercased and stemmed, without the most common English words, so that
// "Crashes on login" and "crash login" give the same tokens.
//
// The words are stemmed with Porter2, the English stemmer of Snowball, rather
// than with the porter tokenizer of FTS5, which implements the original Porter
// algorithm: Porter2 matches more of the forms of a word (dying and die, skies
// and sky), and merges fewer unrelated words (news and new, communism and
// communication).
//
// The languages written without spaces between words (Chinese, Japanese,
// Korean) can't be split into words without a dictionary: a run of their
// characters gives its overlapping pairs of characters instead, "数据库" giving
// "数据 据库". A word of two characters or more is then found as a phrase of
// pairs, a single character only when it stands alone.
func textTokens(text string) []string {
	var tokens []string
	var word, cjk []rune

	flushWord := func() {
		if len(word) == 0 {
			return
		}
		w := strings.Trim(strings.ToLower(string(word)), "'")
		w = strings.TrimSuffix(w, "'s")
		w = strings.ReplaceAll(w, "'", "")
		word = word[:0]
		if _, stop := stopWords[w]; w == "" || stop {
			return
		}
		env := snowballstem.NewEnv(w)
		english.Stem(env)
		tokens = append(tokens, env.Current())
	}
	flushCJK := func() {
		if len(cjk) == 1 {
			tokens = append(tokens, string(cjk))
		}
		for i := 0; i+1 < len(cjk); i++ {
			tokens = append(tokens, string(cjk[i:i+2]))
		}
		cjk = cjk[:0]
	}

	for _, r := range text {
		switch {
		// ー, the long vowel mark of kana, is of no script of its own
		case unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) || r == 'ー' || r == 'ｰ':
			flushWord()
			cjk = append(cjk, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r):
			flushCJK()
			word = append(word, r)
		case (r == '\'' || r == '’') && len(word) > 0:
			// inside a word: don't, server's
			word = append(word, '\'')
		default:
			flushWord()
			flushCJK()
		}
	}
	flushWord()
	flushCJK()

	return tokens
}

// stopWords are the words too common to be worth indexing or searching, the
// list of Lucene.
var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "but": {}, "by": {},
	"for": {}, "if": {}, "in": {}, "into": {}, "is": {}, "it": {}, "no": {}, "not": {}, "of": {},
	"on": {}, "or": {}, "such": {}, "that": {}, "the": {}, "their": {}, "then": {}, "there": {},
	"these": {}, "they": {}, "this": {}, "to": {}, "was": {}, "will": {}, "with": {},
}
