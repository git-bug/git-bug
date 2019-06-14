package github

type createIssueMutation struct {
	CreateIssue struct {
		Issue struct {
			ID  string `graphql:"id"`
			URL string `graphql:"url"`
		}
	} `graphql:"createIssue(input:$input)"`
}

type updateIssueMutation struct {
	UpdateIssue struct {
		Issue struct {
			ID  string `graphql:"id"`
			URL string `graphql:"url"`
		}
	} `graphql:"updateIssue(input:$input)"`
}

type addCommentToIssueMutation struct {
	AddComment struct {
		CommentEdge struct {
			Node struct {
				ID  string `graphql:"id"`
				URL string `graphql:"url"`
			}
		}
	} `graphql:"addComment(input:$input)"`
}

type updateIssueCommentMutation struct {
	IssueComment struct {
		ID  string `graphql:"id"`
		URL string `graphql:"url"`
	} `graphql:"addComment(input:$input)"`
}

type removeLabelsMutation struct {
}
