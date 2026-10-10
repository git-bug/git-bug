// Package core contains the target-agnostic code to define and run a bridge
package core

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/git-bug/gitconfig"
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

var ErrImportNotSupported = errors.New("import is not supported")
var ErrExportNotSupported = errors.New("export is not supported")

const (
	ConfigKeyTarget = "target"

	MetaKeyOrigin = "origin"

	// the configuration of bridge <name> is in [git-bug "bridge.<name>"]
	bridgeConfigSection          = "git-bug"
	bridgeConfigSubsectionPrefix = "bridge."

	configKeyLastImportTime = "lastImportTime"
)

// bridgeConfigKey returns the key of the configuration entry name of the bridge.
func bridgeConfigKey(bridge, name string) string {
	return bridgeConfigSection + "." + bridgeConfigSubsectionPrefix + bridge + "." + name
}

var bridgeImpl map[string]reflect.Type
var bridgeLoginMetaKey map[string]string

// Bridge is a wrapper around a BridgeImpl that will bind low-level
// implementation with utility code to provide high-level functions.
type Bridge struct {
	Name           string
	repo           *cache.RepoCache
	impl           BridgeImpl
	importer       Importer
	exporter       Exporter
	conf           Configuration
	initImportDone bool
	initExportDone bool
}

// Register will register a new BridgeImpl
func Register(impl BridgeImpl) {
	if bridgeImpl == nil {
		bridgeImpl = make(map[string]reflect.Type)
	}
	if bridgeLoginMetaKey == nil {
		bridgeLoginMetaKey = make(map[string]string)
	}
	bridgeImpl[impl.Target()] = reflect.TypeOf(impl).Elem()
	bridgeLoginMetaKey[impl.Target()] = impl.LoginMetaKey()
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

// TargetExist return true if the given target has a bridge implementation
func TargetExist(target string) bool {
	_, ok := bridgeImpl[target]
	return ok
}

// LoginMetaKey return the metadata key used to store the remote bug-tracker login
// on the user identity. The corresponding value is used to match identities and
// credentials.
func LoginMetaKey(target string) (string, error) {
	metaKey, ok := bridgeLoginMetaKey[target]
	if !ok {
		return "", fmt.Errorf("unknown bridge target %v", target)
	}

	return metaKey, nil
}

// Instantiate a new Bridge for a repo, from the given target and name
func NewBridge(repo *cache.RepoCache, target string, name string) (*Bridge, error) {
	implType, ok := bridgeImpl[target]
	if !ok {
		return nil, fmt.Errorf("unknown bridge target %v", target)
	}

	impl := reflect.New(implType).Interface().(BridgeImpl)

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

	target := conf[ConfigKeyTarget]
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
		return nil, fmt.Errorf("multiple bridge are configured, you need to select one explicitly")
	}

	return LoadBridge(repo, bridges[0])
}

// ConfiguredBridges return the list of bridge that are configured for the given
// repo
func ConfiguredBridges(repo repository.RepoConfig) ([]string, error) {
	cfg, err := repo.Config().Read()
	if err != nil {
		return nil, errors.Wrap(err, "can't read configured bridges")
	}

	var result []string
	for _, sub := range cfg.Subsections(bridgeConfigSection) {
		if name, ok := strings.CutPrefix(sub, bridgeConfigSubsectionPrefix); ok {
			result = append(result, name)
		}
	}

	return result, nil
}

// Check if a bridge exist
func BridgeExist(repo repository.RepoConfig, name string) bool {
	cfg, err := repo.Config().Read()
	if err != nil {
		return false
	}

	for range cfg.Subsection(bridgeConfigSection, bridgeConfigSubsectionPrefix+name) {
		return true
	}
	return false
}

// Remove a configured bridge
func RemoveBridge(repo repository.RepoConfig, name string) error {
	re := regexp.MustCompile(`^[a-zA-Z0-9]+`)

	if !re.MatchString(name) {
		return fmt.Errorf("bad bridge fullname: %s", name)
	}

	return repo.Config().Update(func(f *gitconfig.File) error {
		_, err := f.RemoveSubsection(bridgeConfigSection, bridgeConfigSubsectionPrefix+name)
		return err
	})
}

// Configure run the target specific configuration process
func (b *Bridge) Configure(params BridgeParams, interactive bool) error {
	validateParams(params, b.impl)

	conf, err := b.impl.Configure(b.repo, params, interactive)
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

func validateParams(params BridgeParams, impl BridgeImpl) {
	validParams := impl.ValidParams()

	paramsValue := reflect.ValueOf(params)
	paramsType := paramsValue.Type()

	for i := 0; i < paramsValue.NumField(); i++ {
		name := paramsType.Field(i).Name
		val := paramsValue.Field(i).Interface().(string)
		_, valid := validParams[name]
		if val != "" && !valid {
			_, _ = fmt.Fprintln(os.Stderr, params.fieldWarning(name, impl.Target()))
		}
	}
}

func (b *Bridge) storeConfig(conf Configuration) error {
	err := b.repo.Config().Update(func(f *gitconfig.File) error {
		for key, val := range conf {
			if err := f.ReplaceAll(bridgeConfigKey(b.Name, key), val); err != nil {
				return err
			}
		}
		return nil
	})
	return errors.Wrap(err, "error while storing bridge configuration")
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

func loadConfig(repo repository.RepoConfig, name string) (Configuration, error) {
	cfg, err := repo.Config().Read()
	if err != nil {
		return nil, errors.Wrap(err, "error while reading bridge configuration")
	}

	result := make(Configuration)
	for e := range cfg.Subsection(bridgeConfigSection, bridgeConfigSubsectionPrefix+name) {
		result[e.Key.Name] = e.Value
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

func (b *Bridge) ensureImportInit(ctx context.Context) error {
	if b.initImportDone {
		return nil
	}

	importer := b.getImporter()
	if importer != nil {
		err := importer.Init(ctx, b.repo, b.conf)
		if err != nil {
			return err
		}
	}

	b.initImportDone = true
	return nil
}

func (b *Bridge) ensureExportInit(ctx context.Context) error {
	if b.initExportDone {
		return nil
	}

	exporter := b.getExporter()
	if exporter != nil {
		err := exporter.Init(ctx, b.repo, b.conf)
		if err != nil {
			return err
		}
	}

	b.initExportDone = true
	return nil
}

func (b *Bridge) ImportAllSince(ctx context.Context, since time.Time) (<-chan ImportResult, error) {
	// 5 seconds before the actual start just to be sure.
	importStartTime := time.Now().Add(-5 * time.Second)

	importer := b.getImporter()
	if importer == nil {
		return nil, ErrImportNotSupported
	}

	err := b.ensureConfig()
	if err != nil {
		return nil, err
	}

	err = b.ensureImportInit(ctx)
	if err != nil {
		return nil, err
	}

	events, err := importer.ImportAll(ctx, b.repo, since)
	if err != nil {
		return nil, err
	}

	out := make(chan ImportResult)
	go func() {
		defer close(out)
		noError := true

		// relay all events while checking that everything went well
		for event := range events {
			if event.Event == ImportEventError {
				noError = false
			}
			out <- event
		}

		// store the last import time ONLY if no error happened
		if noError {
			err = b.repo.Config().Update(func(f *gitconfig.File) error {
				return f.ReplaceAll(bridgeConfigKey(b.Name, configKeyLastImportTime), strconv.FormatInt(importStartTime.Unix(), 10))
			})
		}
	}()

	return out, nil
}

func (b *Bridge) ImportAll(ctx context.Context) (<-chan ImportResult, error) {
	// If possible, restart from the last import time
	cfg, err := b.repo.Config().Read()
	if err != nil {
		return nil, err
	}
	lastImport, err := cfg.Int(bridgeConfigKey(b.Name, configKeyLastImportTime), 0)
	if err == nil && lastImport != 0 {
		return b.ImportAllSince(ctx, time.Unix(lastImport, 0))
	}

	return b.ImportAllSince(ctx, time.Time{})
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

	err = b.ensureExportInit(ctx)
	if err != nil {
		return nil, err
	}

	return exporter.ExportAll(ctx, b.repo, since)
}
