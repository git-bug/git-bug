package core

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/pkg/errors"
)

var ErrImportNorSupported = errors.New("import is not supported")
var ErrExportNorSupported = errors.New("export is not supported")

var bridgeImpl map[string]reflect.Type

// Bridge is a wrapper around a BridgeImpl that will bind low-level
// implementation with utility code to provide high-level functions.
type Bridge struct {
	Name string
	repo *cache.RepoCache
	impl BridgeImpl
	conf Configuration
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

	return result
}

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

func ConfiguredBridges(repo repository.RepoCommon) ([]string, error) {
	configs, err := repo.ReadConfigs("git-bug.bridge.")
	if err != nil {
		return nil, errors.Wrap(err, "can't read configured bridges")
	}

	re, err := regexp.Compile(`git-bug.bridge.([^\.]+\.[^\.]+)`)
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

func RemoveBridge(repo repository.RepoCommon, fullName string) error {
	re, err := regexp.Compile(`^[^\.]+\.[^\.]+$`)
	if err != nil {
		panic(err)
	}

	if !re.MatchString(fullName) {
		return fmt.Errorf("bad bridge fullname: %s", fullName)
	}

	keyPrefix := fmt.Sprintf("git-bug.bridge.%s", fullName)
	return repo.RmConfigs(keyPrefix)
}

func (b *Bridge) Configure() error {
	conf, err := b.impl.Configure(b.repo)
	if err != nil {
		return err
	}

	b.conf = conf

	return b.storeConfig(conf)
}

func (b *Bridge) storeConfig(conf Configuration) error {
	for key, val := range conf {
		storeKey := fmt.Sprintf("git-bug.bridge.%s.%s.%s", b.impl.Target(), b.Name, key)

		err := b.repo.StoreConfig(storeKey, val)
		if err != nil {
			return errors.Wrap(err, "error while storing bridge configuration")
		}
	}

	return nil
}

func (b Bridge) getConfig() (Configuration, error) {
	var err error
	if b.conf == nil {
		b.conf, err = b.loadConfig()
		if err != nil {
			return nil, err
		}
	}

	return b.conf, nil
}

func (b Bridge) loadConfig() (Configuration, error) {
	keyPrefix := fmt.Sprintf("git-bug.bridge.%s.%s.", b.impl.Target(), b.Name)

	pairs, err := b.repo.ReadConfigs(keyPrefix)
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

func (b Bridge) ImportAll() error {
	importer := b.impl.Importer()
	if importer == nil {
		return ErrImportNorSupported
	}

	conf, err := b.getConfig()
	if err != nil {
		return err
	}

	return b.impl.Importer().ImportAll(b.repo, conf)
}

func (b Bridge) Import(id string) error {
	importer := b.impl.Importer()
	if importer == nil {
		return ErrImportNorSupported
	}

	conf, err := b.getConfig()
	if err != nil {
		return err
	}

	return b.impl.Importer().Import(b.repo, conf, id)
}

func (b Bridge) ExportAll() error {
	exporter := b.impl.Exporter()
	if exporter == nil {
		return ErrExportNorSupported
	}

	conf, err := b.getConfig()
	if err != nil {
		return err
	}

	return b.impl.Exporter().ExportAll(b.repo, conf)
}

func (b Bridge) Export(id string) error {
	exporter := b.impl.Exporter()
	if exporter == nil {
		return ErrExportNorSupported
	}

	conf, err := b.getConfig()
	if err != nil {
		return err
	}

	return b.impl.Exporter().Export(b.repo, conf, id)
}
