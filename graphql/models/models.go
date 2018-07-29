package models

type Repository struct {}

type Bug struct {
	ID         string              `json:"id"`
	HumanId    string              `json:"humanId"`
	Title      string              `json:"title"`
	Status     Status              `json:"status"`
	Labels     []string            `json:"labels"`
}
type OperationConnection struct {
	PageInfo   PageInfo        `json:"pageInfo"`
	TotalCount int             `json:"totalCount"`
}
type CommentConnection struct {
	PageInfo   PageInfo      `json:"pageInfo"`
	TotalCount int           `json:"totalCount"`
}
type BugConnection struct {
	PageInfo   PageInfo  `json:"pageInfo"`
	TotalCount int       `json:"totalCount"`
}
