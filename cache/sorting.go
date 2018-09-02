package cache

import "sort"

type OrderBy int

const (
	_ OrderBy = iota
	OrderById
	OrderByCreation
	OrderByEdit
)

type OrderDirection int

const (
	_ OrderDirection = iota
	OrderAscending
	OrderDescending
)

func (c *RepoCache) AllBugsId(order OrderBy, direction OrderDirection) []string {
	if order == OrderById {
		return c.orderIds(direction)
	}

	excerpts := c.allExcerpt()

	var sorter sort.Interface

	switch order {
	case OrderByCreation:
		sorter = BugsByCreationTime(excerpts)
	case OrderByEdit:
		sorter = BugsByEditTime(excerpts)
	default:
		panic("missing sort type")
	}

	if direction == OrderDescending {
		sorter = sort.Reverse(sorter)
	}

	sort.Sort(sorter)

	result := make([]string, len(excerpts))

	for i, val := range excerpts {
		result[i] = val.Id
	}

	return result
}

func (c *RepoCache) orderIds(direction OrderDirection) []string {
	result := make([]string, len(c.excerpts))

	i := 0
	for key := range c.excerpts {
		result[i] = key
		i++
	}

	var sorter sort.Interface = sort.StringSlice(result)

	if direction == OrderDescending {
		sorter = sort.Reverse(sorter)
	}

	sort.Sort(sorter)

	return result
}
