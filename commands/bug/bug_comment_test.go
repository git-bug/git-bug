package bugcmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
	"github.com/git-bug/git-bug/commands/cmdtest"
	"github.com/git-bug/git-bug/commands/execenv"
)

func TestBugComment(t *testing.T) {
	const golden = "testdata/comment/message-only"

	env, bug := testenv.NewTestEnvAndBug(t)

	require.NoError(t, runBugComment(env, []string{bug.Human()}))

	requireCommentsEqual(t, golden, env)
}

const gitDateFormat = "Mon Jan 2 15:04:05 2006 -0700"

type parsedComment struct {
	author  string
	id      string
	date    time.Time
	message string
}

type parseFunc func(*parsedComment, string)

type commentParser struct {
	t        *testing.T
	fn       parseFunc
	comments []parsedComment
}

func parseComments(t *testing.T, env *execenv.Env) []parsedComment {
	t.Helper()

	parser := &commentParser{
		t:        t,
		comments: []parsedComment{},
	}

	comment := &parsedComment{}
	parser.fn = parser.parseAuthor

	for _, line := range strings.Split(env.Out.String(), "\n") {
		parser.fn(comment, line)
	}

	parser.comments = append(parser.comments, *comment)

	return parser.comments
}

func (p *commentParser) parseAuthor(comment *parsedComment, line string) {
	p.t.Helper()

	tkns := strings.Split(line, ": ")
	require.Len(p.t, tkns, 2)
	require.Equal(p.t, "Author", tkns[0])

	comment.author = tkns[1]
	p.fn = p.parseID
}

func (p *commentParser) parseID(comment *parsedComment, line string) {
	p.t.Helper()

	tkns := strings.Split(line, ": ")
	require.Len(p.t, tkns, 2)
	require.Equal(p.t, "Id", tkns[0])

	comment.id = tkns[1]
	p.fn = p.parseDate
}

func (p *commentParser) parseDate(comment *parsedComment, line string) {
	p.t.Helper()

	tkns := strings.Split(line, ": ")
	require.Len(p.t, tkns, 2)
	require.Equal(p.t, "Date", tkns[0])

	date, err := time.Parse(gitDateFormat, tkns[1])
	require.NoError(p.t, err)

	comment.date = date
	p.fn = p.parseMessage
}

func (p *commentParser) parseMessage(comment *parsedComment, line string) {
	p.t.Helper()

	if strings.HasPrefix(line, "Author: ") {
		p.comments = append(p.comments, *comment)
		comment = &parsedComment{}
		p.parseAuthor(comment, line)

		return
	}

	require.True(p.t, line == "" || strings.HasPrefix(line, "    "))

	comment.message = strings.Join([]string{comment.message, line}, "\n")
}

func normalizeParsedComments(t *testing.T, comments []parsedComment) []parsedComment {
	t.Helper()

	prefix := 0x1234567
	date, err := time.Parse(gitDateFormat, "Fri Aug 19 07:00:00 2022 +1900")
	require.NoError(t, err)

	var out []parsedComment

	for i, comment := range comments {
		comment.id = fmt.Sprintf("%7x", prefix+i)
		comment.date = date.Add(time.Duration(i) * time.Minute)
		out = append(out, comment)
	}

	return out
}

func requireCommentsEqual(t *testing.T, golden string, env *execenv.Env) {
	t.Helper()

	const goldenFilePattern = "%s-%d-golden.txt"

	comments := parseComments(t, env)
	comments = normalizeParsedComments(t, comments)

	if *cmdtest.Update {
		for i, comment := range comments {
			fileName := fmt.Sprintf(goldenFilePattern, golden, i)
			require.NoError(t, os.WriteFile(fileName, []byte(comment.message), 0644))
		}
	}

	prefix := 0x1234567
	date, err := time.Parse(gitDateFormat, "Fri Aug 19 07:00:00 2022 +1900")
	require.NoError(t, err)

	for i, comment := range comments {
		require.Equal(t, "John Doe", comment.author)
		require.Equal(t, fmt.Sprintf("%7x", prefix+i), comment.id)
		require.Equal(t, date.Add(time.Duration(i)*time.Minute), comment.date)

		fileName := fmt.Sprintf(goldenFilePattern, golden, i)
		exp, err := os.ReadFile(fileName)
		require.NoError(t, err)
		require.Equal(t, strings.ReplaceAll(string(exp), "\r", ""), strings.ReplaceAll(comment.message, "\r", ""))
	}
}
