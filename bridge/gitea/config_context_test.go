package gitea

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
)

func TestConfigureHonorsCancellationBeforeTokenValidation(t *testing.T) {
	_, backend := newRoundTripRepo(t, t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := (&Gitea{}).Configure(ctx, backend, core.BridgeParams{
		URL:      server.URL + "/owner/project",
		TokenRaw: "test-token",
	}, false)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, requests.Load(), "canceled setup must not send the token to the server")
}
