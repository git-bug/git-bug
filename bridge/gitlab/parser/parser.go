package parser

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"regexp"
	"strings"
)

type parser interface {
	Parse() (string, error)
}

type parserType int

const (
	TitleParser parserType = iota
)

// NewWithInput returns a new parser instance
func NewWithInput(t parserType, input string) parser {
	var p parser

	switch t {
	case TitleParser:
		p = titleParser{input: input}
	}

	return p
}

type titleParser struct {
	input string
}

// Parse is used to fetch the new title from a "changed title" event
//
// this func is a great example of something that is _extremely_ fragile; the
// input string is pulled from the body of a gitlab message containing html
// fragments, and has changed on at least [one occasion][0], breaking our test
// pipelines and preventing feature development. i think querying for an issue's
// _iterations_ [1] would likely be a better approach.
//
// example p.input values:
// - changed title from **some title** to **some{+ new +}title**
// - changed title from **some{- new-} title** to **some title**
// - <p>changed title from <code class="idiff">some title</code> to <code class="idiff">some<span class="idiff left addition"> new</span> title</code></p>
//
// [0]: https://github.com/git-bug/git-bug/issues/1367
// [1]: https://docs.gitlab.com/api/resource_iteration_events/#list-project-issue-iteration-events
func (p titleParser) Parse() (string, error) {
	var reHTML = regexp.MustCompile(`.* to <code\s+class="idiff"\s*>(.*?)</code>`)
	var reMD = regexp.MustCompile(`.* to \*\*(.*)\*\*`)

	matchHTML := reHTML.FindAllStringSubmatch(p.input, -1)
	matchMD := reMD.FindAllStringSubmatch(p.input, -1)

	if len(matchHTML) == 1 {
		t, err := p.stripHTML(matchHTML[0][1])
		if err != nil {
			return "", fmt.Errorf("unable to strip HTML from new title: %q", t)
		}
		return strings.TrimSpace(t), nil
	}

	if len(matchMD) == 1 {
		reDiff := regexp.MustCompile(`{\+(.*?)\+}`)

		t := matchMD[0][1]
		t = reDiff.ReplaceAllString(t, "$1")

		return strings.TrimSpace(t), nil
	}

	return "", fmt.Errorf(
		"failed to extract title: html=%d md=%d input=%q",
		len(matchHTML),
		len(matchMD),
		p.input,
	)
}

// stripHTML removes all html tags from a provided string
func (p titleParser) stripHTML(s string) (string, error) {
	nodes, err := html.Parse(strings.NewReader(s))
	if err != nil {
		// return the original unmodified string in the event html.Parse()
		// fails; let the downstream callsites decide if they want to proceed
		// with the value or not.
		return s, err
	}

	var buf bytes.Buffer
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(nodes)

	return buf.String(), nil
}
