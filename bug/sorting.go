package bug

type BugsByCreationTime []*Bug

func (b BugsByCreationTime) Len() int {
	return len(b)
}

func (b BugsByCreationTime) Less(i, j int) bool {
	if b[i].createTime < b[j].createTime {
		return true
	}

	if b[i].createTime > b[j].createTime {
		return false
	}

	// When the logical clocks are identical, that means we had a concurrent
	// edition. In this case we rely on the timestamp. While the timestamp might
	// be incorrect due to a badly set clock, the drift in sorting is bounded
	// by the first sorting using the logical clock. That means that if users
	// synchronize their bugs regularly, the timestamp will rarely be used, and
	// should still provide a kinda accurate sorting when needed.
	return b[i].firstOp().Time().Before(b[j].firstOp().Time())
}

func (b BugsByCreationTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

type BugsByEditTime []*Bug

func (b BugsByEditTime) Len() int {
	return len(b)
}

func (b BugsByEditTime) Less(i, j int) bool {
	if b[i].editTime < b[j].editTime {
		return true
	}

	if b[i].editTime > b[j].editTime {
		return false
	}

	// When the logical clocks are identical, that means we had a concurrent
	// edition. In this case we rely on the timestamp. While the timestamp might
	// be incorrect due to a badly set clock, the drift in sorting is bounded
	// by the first sorting using the logical clock. That means that if users
	// synchronize their bugs regularly, the timestamp will rarely be used, and
	// should still provide a kinda accurate sorting when needed.
	return b[i].lastOp().Time().Before(b[j].lastOp().Time())
}

func (b BugsByEditTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
