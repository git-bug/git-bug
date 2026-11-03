package iterator

import (
	"testing"

	gitea "gitea.dev/sdk"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, serverURL string) *gitea.Client {
	t.Helper()
	client, err := gitea.NewClient(serverURL, gitea.SetToken("test"), gitea.SetGiteaVersion("1.24.0"))
	require.NoError(t, err)
	return client
}

// nextLabel advances iter to the next LabelEvent, skipping other event types.
func nextLabel(iter *Iterator) bool {
	for iter.NextEvent() {
		if _, ok := iter.EventValue().(*LabelEvent); ok {
			return true
		}
	}
	return false
}

func labelValue(iter *Iterator) *LabelEvent {
	return iter.EventValue().(*LabelEvent)
}
