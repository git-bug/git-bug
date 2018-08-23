package bug

type OperationIterator struct {
	bug       *Bug
	packIndex int
	opIndex   int
}

func NewOperationIterator(bug Interface) *OperationIterator {
	return &OperationIterator{
		bug:       bugFromInterface(bug),
		packIndex: 0,
		opIndex:   -1,
	}
}

func (it *OperationIterator) Next() bool {
	// Special case of the staging area
	if it.packIndex == len(it.bug.packs) {
		pack := it.bug.staging
		it.opIndex++
		return it.opIndex < len(pack.Operations)
	}

	if it.packIndex >= len(it.bug.packs) {
		return false
	}

	pack := it.bug.packs[it.packIndex]

	it.opIndex++

	if it.opIndex < len(pack.Operations) {
		return true
	}

	// Note: this iterator doesn't handle the empty pack case
	it.opIndex = 0
	it.packIndex++

	// Special case of the non-empty staging area
	if it.packIndex == len(it.bug.packs) && len(it.bug.staging.Operations) > 0 {
		return true
	}

	return it.packIndex < len(it.bug.packs)
}

func (it *OperationIterator) Value() Operation {
	// Special case of the staging area
	if it.packIndex == len(it.bug.packs) {
		pack := it.bug.staging

		if it.opIndex >= len(pack.Operations) {
			panic("Iterator is not valid anymore")
		}

		return pack.Operations[it.opIndex]
	}

	if it.packIndex >= len(it.bug.packs) {
		panic("Iterator is not valid anymore")
	}

	pack := it.bug.packs[it.packIndex]

	if it.opIndex >= len(pack.Operations) {
		panic("Iterator is not valid anymore")
	}

	return pack.Operations[it.opIndex]
}
