package board

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

func TestResolvers(t *testing.T) {
	repo := repository.NewMockRepo()

	rs := entity.Resolvers{
		&identity.IdentityStub{}: identity.NewStubResolver(),
		&identity.Identity{}:     identity.NewSimpleResolver(repo),
		&bug.Bug{}:               bug.NewSimpleResolver(repo),
	}

	ide, err := entity.Resolve[identity.Interface](rs, "foo")
	require.NoError(t, err)

	fmt.Println(ide)
}
