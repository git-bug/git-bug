package github

import "github.com/shurcooL/githubv4"

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
	} `graphql:"updateIssueComment(input:$input)"`
}

type createLabelMutation struct {
	CreateLabel struct {
		Label struct {
			ID string `graphql:"id"`
		} `graphql:"label"`
	} `graphql:"createLabel(input:{repositoryId: $repositoryId, name: $name, color: $color})"`
}

type removeLabelsFromLabelableMutation struct {
	AddLabels struct{} `graphql:"removeLabelsFromLabelable(input:$input)"`
}

type addLabelsToLabelableMutation struct {
	RemoveLabels struct{} `graphql:"addLabelsToLabelable(input:$input)"`
}

type createLabelInput struct {
	Color        githubv4.String  `json:"color"`
	Description  *githubv4.String `json:"description"`
	Name         githubv4.String  `json:"name"`
	RepositoryID githubv4.ID      `json:"repositoryId"`
}
