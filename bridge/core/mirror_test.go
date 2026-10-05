package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/repository"
)

type fakeBridgeImpl struct {
	exporter Exporter
}

func (fakeBridgeImpl) Target() string                     { return "fake" }
func (fakeBridgeImpl) NewImporter() Importer              { return nil }
func (f fakeBridgeImpl) NewExporter() Exporter            { return f.exporter }
func (fakeBridgeImpl) ValidateConfig(Configuration) error { return nil }
func (fakeBridgeImpl) LoginMetaKey() string               { return "fake-login" }
func (fakeBridgeImpl) ValidParams() map[string]interface{} {
	return nil
}
func (fakeBridgeImpl) Configure(*cache.RepoCache, BridgeParams, bool) (Configuration, error) {
	return nil, nil
}

type plainExporter struct{ initialized bool }

func (e *plainExporter) Init(context.Context, *cache.RepoCache, Configuration) error {
	e.initialized = true
	return nil
}

func (e *plainExporter) ExportAll(context.Context, *cache.RepoCache, time.Time) (<-chan ExportResult, error) {
	out := make(chan ExportResult)
	close(out)
	return out, nil
}

type foreignExporter struct {
	plainExporter
	foreignAtInit bool
	foreign       bool
}

func (e *foreignExporter) EnableForeign() { e.foreign = true }

func (e *foreignExporter) Init(ctx context.Context, repo *cache.RepoCache, conf Configuration) error {
	e.foreignAtInit = e.foreign
	return e.plainExporter.Init(ctx, repo, conf)
}

func newFakeBridge(e Exporter) *Bridge {
	return &Bridge{Name: "test", impl: fakeBridgeImpl{exporter: e}, conf: Configuration{}}
}

func TestForeignPushRejectedByUnsupportedBridge(t *testing.T) {
	e := &plainExporter{}
	_, err := newFakeBridge(e).ExportAllWithOptions(context.Background(), time.Time{}, ExportOptions{Foreign: true})
	assert.ErrorContains(t, err, "fake bridge cannot push bugs from other bug trackers")
	assert.False(t, e.initialized, "the exporter must not start")
}

func TestForeignPushEnabledBeforeInit(t *testing.T) {
	e := &foreignExporter{}
	_, err := newFakeBridge(e).ExportAllWithOptions(context.Background(), time.Time{}, ExportOptions{Foreign: true})
	require.NoError(t, err)
	assert.True(t, e.foreignAtInit)
}

func TestPlainPushLeavesForeignDisabled(t *testing.T) {
	e := &foreignExporter{}
	_, err := newFakeBridge(e).ExportAll(context.Background(), time.Time{})
	require.NoError(t, err)
	assert.False(t, e.foreign)
}

func TestForeignPushAfterInitRejected(t *testing.T) {
	e := &foreignExporter{}
	b := newFakeBridge(e)
	_, err := b.ExportAll(context.Background(), time.Time{})
	require.NoError(t, err)
	_, err = b.ExportAllWithOptions(context.Background(), time.Time{}, ExportOptions{Foreign: true})
	assert.Error(t, err)
	assert.False(t, e.foreign)
}

func TestMirrorMetaKeyScopedToRepository(t *testing.T) {
	a := MirrorMetaKey("issue", "gitea", "https://gitea.com/mcepl/m2crypto")
	b := MirrorMetaKey("issue", "gitea", "https://codefloe.com/mcepl/m2crypto")
	assert.NotEqual(t, a, b)
	assert.NotEqual(t, a, MirrorMetaKey("comment", "gitea", "https://gitea.com/mcepl/m2crypto"))
}

func TestMirrorAttribution(t *testing.T) {
	repo := repository.NewMockRepo()
	author, err := identity.NewIdentityFull(repo, "Jane Doe", "", "jdoe", "", nil)
	require.NoError(t, err)
	at := time.Date(2024, 3, 4, 5, 6, 0, 0, time.FixedZone("CET", 3600))

	assert.Equal(t,
		"_Originally posted by Jane Doe (jdoe) on 2024-03-04 04:06 UTC at https://example.org/o/p/issues/7_\n\n",
		MirrorAttribution(author, at, "https://example.org/o/p/issues/7"))
	assert.Equal(t,
		"_Originally posted by Jane Doe (jdoe) on 2024-03-04 04:06 UTC_\n\n",
		MirrorAttribution(author, at, ""))
}
