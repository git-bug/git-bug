package todosrht

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebuggingTransportRedactsToken(t *testing.T) {
	const secret = "super-secret-token-value"
	body := strings.Repeat("x", maxDebugBodySize*2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer "+secret, r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	// A pipe can fill and block client.Do before the test starts reading it,
	// especially on Windows. Capture to a file instead.
	logFile, err := os.CreateTemp(t.TempDir(), "stderr-*")
	require.NoError(t, err)
	oldStderr := os.Stderr
	t.Cleanup(func() {
		os.Stderr = oldStderr
		_ = logFile.Close()
	})
	os.Stderr = logFile

	client := &http.Client{Transport: &authTransport{
		token: secret,
		base:  &debuggingTransport{base: http.DefaultTransport},
	}}
	req, err := http.NewRequestWithContext(context.Background(), "POST", srv.URL, strings.NewReader(`{"query":"q"}`))
	require.NoError(t, err)
	resp, err := client.Do(req)

	os.Stderr = oldStderr
	require.NoError(t, err)
	defer resp.Body.Close()

	logged, err := os.ReadFile(logFile.Name())
	require.NoError(t, err)

	// the caller still gets the full response body
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, body, string(got))

	assert.NotContains(t, string(logged), secret)
	assert.Contains(t, string(logged), "Bearer [REDACTED]")
	assert.Contains(t, string(logged), `{"query":"q"}`)
	assert.Contains(t, string(logged), "bytes truncated")
}
