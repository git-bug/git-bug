package bug

import (
	"github.com/MichaelMure/git-bug/identity"
)

// EnsureIdentities walk the graph of operations and make sure that all Identity
// are properly loaded. That is, it replace all the IdentityStub with the full
// Identity, loaded through a Resolver.
func (bug *Bug) EnsureIdentities(resolver identity.Resolver) error {
	it := NewOperationIterator(bug)

	for it.Next() {
		op := it.Value()
		base := op.base()

		if stub, ok := base.Author.(*identity.IdentityStub); ok {
			i, err := resolver.ResolveIdentity(stub.Id())
			if err != nil {
				return err
			}

			base.Author = i
		}
	}
	return nil
}
