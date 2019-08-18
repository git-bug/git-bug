// Package core contains the target-agnostic code to define and run a bridge
package core

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

var ErrImportNotSupported = errors.New("import is not supported")
var ErrExportNotSupported = errors.New("export is not supported")

const (
	KeyTarget = "target"
	KeyOrigin = "origin"

	bridgeConfigKeyPrefix = "git-bug.bridge"
)

var bridgeImpl map[string]reflect.Type

// BridgeParams holds parameters to simplify the bridge configuration without
// having to make terminal prompts.
type BridgeParams struct {
	Owner   string
	Project string
	URL     string
	Token   string
}

// Bridge is a wrapper around a BridgeImpl that will bind low-level
// implementation with utility code to provide high-level functions.
type Bridge struct {
	Name     string
	repo     *cache.RepoCache
	impl     BridgeImpl
	importer Importer
	exporter Exporter
	conf     Configuration
	initDone bool
}

// Register will register a new BridgeImpl
func Register(impl BridgeImpl) {
	if bridgeImpl == nil {
		bridgeImpl = make(map[string]reflect.Type)
	}
	bridgeImpl[impl.Target()] = reflect.TypeOf(impl)
}

// Targets return all known bridge implementation target
func Targets() []string {
	var result []string

	for key := range bridgeImpl {
		result = append(result, key)
	}

	sort.Strings(result)

	return result
}

// Instantiate a new Bridge for a repo, from the given target and name
func NewBridge(repo *cache.RepoCache, target string, name string) (*Bridge, error) {
	implType, ok := bridgeImpl[target]
	if !ok {
		return nil, fmt.Errorf("unknown bridge target %v", target)
	}

	impl := reflect.New(implType).Elem().Interface().(BridgeImpl)

	bridge := &Bridge{
		Name: name,
		repo: repo,
		impl: impl,
	}

	return bridge, nil
}

// LoadBridge instantiate a new bridge from a repo configuration
func LoadBridge(repo *cache.RepoCache, name string) (*Bridge, error) {
	conf, err := loadConfig(repo, name)
	if err != nil {
		return nil, err
	}

	target := conf[KeyTarget]
	bridge, err := NewBridge(repo, target, name)
	if err != nil {
		return nil, err
	}

	err = bridge.impl.ValidateConfig(conf)
	if err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	// will avoid reloading configuration before an export or import call
	bridge.conf = conf
	return bridge, nil
}

// Attempt to retrieve a default bridge for the given repo. If zero or multiple
// bridge exist, it fails.
func DefaultBridge(repo *cache.RepoCache) (*Bridge, error) {
	bridges, err := ConfiguredBridges(repo)
	if err != nil {
		return nil, err
	}

	if len(bridges) == 0 {
		return nil, fmt.Errorf("no configured bridge")
	}

	if len(bridges) > 1 {
		return nil, fmt.Errorf("multiple bridge are configured, you need to select one explicitely")
	}

	return LoadBridge(repo, bridges[0])
}

// ConfiguredBridges return the list of bridge that are configured for the given
// repo
func ConfiguredBridges(repo repository.RepoCommon) ([]string, error) {
	configs, err := repo.ReadConfigs(bridgeConfigKeyPrefix + ".")
	if err != nil {
		return nil, errors.Wrap(err, "can't read configured bridges")
	}

	re, err := regexp.Compile(bridgeConfigKeyPrefix + `.([^.]+)`)
	if err != nil {
		panic(err)
	}

	set := make(map[string]interface{})

	for key := range configs {
		res := re.FindStringSubmatch(key)

		if res == nil {
			continue
		}

		set[res[1]] = nil
	}

	result := make([]string, len(set))

	i := 0
	for key := range set {
		result[i] = key
		i++
	}

	return result, nil
}

// Check if a bridge exist
func BridgeExist(repo repository.RepoCommon, name string) bool {
	keyPrefix := fmt.Sprintf("git-bug.bridge.%s.", name)

	conf, err := repo.ReadConfigs(keyPrefix)

	return err == nil && len(conf) > 0
}

// Remove a configured bridge
func RemoveBridge(repo repository.RepoCommon, name string) error {
	re, err := regexp.Compile(`^[a-zA-Z0-9]+`)
	if err != nil {
		panic(err)
	}

	if !re.MatchString(name) {
		return fmt.Errorf("bad bridge fullname: %s", name)
	}

	keyPrefix := fmt.Sprintf("git-bug.bridge.%s", name)
	return repo.RmConfigs(keyPrefix)
}

// Configure run the target specific configuration process
func (b *Bridge) Configure(params BridgeParams) error {
	conf, err := b.impl.Configure(b.repo, params)
	if err != nil {
		return err
	}

	err = b.impl.ValidateConfig(conf)
	if err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	b.conf = conf
	return b.storeConfig(conf)
}

func (b *Bridge) storeConfig(conf Configuration) error {
	for key, val := range conf {
		storeKey := fmt.Sprintf("git-bug.bridge.%s.%s", b.Name, key)

		err := b.repo.StoreConfig(storeKey, val)
		if err != nil {
			return errors.Wrap(err, "error while storing bridge configuration")
		}
	}

	return nil
}

func (b *Bridge) ensureConfig() error {
	if b.conf == nil {
		conf, err := loadConfig(b.repo, b.Name)
		if err != nil {
			return err
		}
		b.conf = conf
	}

	return nil
}

func loadConfig(repo repository.RepoCommon, name string) (Configuration, error) {
	keyPrefix := fmt.Sprintf("git-bug.bridge.%s.", name)

	pairs, err := repo.ReadConfigs(keyPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "error while reading bridge configuration")
	}

	result := make(Configuration, len(pairs))
	for key, value := range pairs {
		key := strings.TrimPrefix(key, keyPrefix)
		result[key] = value
	}

	return result, nil
}

func (b *Bridge) getImporter() Importer {
	if b.importer == nil {
		b.importer = b.impl.NewImporter()
	}

	return b.importer
}

func (b *Bridge) getExporter() Exporter {
	if b.exporter == nil {
		b.exporter = b.impl.NewExporter()
	}

	return b.exporter
}

func (b *Bridge) ensureInit() error {
	if b.initDone {
		return nil
	}

	importer := b.getImporter()
	if importer != nil {
		err := importer.Init(b.conf)
		if err != nil {
			return err
		}
	}

	exporter := b.getExporter()
	if exporter != nil {
		err := exporter.Init(b.conf)
		if err != nil {
			return err
		}
	}

	b.initDone = true

	return nil
}

func (b *Bridge) ImportAll(ctx context.Context, since time.Time) (<-chan ImportResult, error) {
	importer := b.getImporter()
	if importer == nil {
		return nil, ErrImportNotSupported
	}

	err := b.ensureConfig()
	if err != nil {
		return nil, err
	}

	err = b.ensureInit()
	if err != nil {
		return nil, err
	}

	return importer.ImportAll(ctx, b.repo, since)
}

func (b *Bridge) ExportAll(ctx context.Context, since time.Time) (<-chan ExportResult, error) {
	exporter := b.getExporter()
	if exporter == nil {
		return nil, ErrExportNotSupported
	}

	err := b.ensureConfig()
	if err != nil {
		return nil, err
	}

	err = b.ensureInit()
	if err != nil {
		return nil, err
	}

	return exporter.ExportAll(ctx, b.repo, since)
}
