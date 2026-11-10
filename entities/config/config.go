// Package config contains the repository config: the settings shared by every
// clone of a repository, stored as a singleton entity.
package config

import (
	"context"
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

var _ dag.Mutable[*Snapshot, Operation] = &Config{}
var _ dag.Tracked[*Snapshot, Operation] = &Config{}
var _ entity.Interface = &Config{}

// 1: original format
const formatVersion = 1

const Typename = "config"
const Namespace = "config"

var def = dag.Definition{
	Typename:             Typename,
	Namespace:            Namespace,
	SingletonKey:         "config",
	OperationUnmarshaler: operationUnmarshaler,
	FormatVersion:        formatVersion,
}

// Mutable is what the functions creating operations (LabelsSet, ...) need from a
// config, see dag.Mutable.
type Mutable = dag.Mutable[*Snapshot, Operation]

// Config holds the data of the repository config, organized in a way close to
// how it will be persisted inside Git. This is the data structure used to merge
// two different versions of the same Config.
type Config struct {
	*dag.Entity
}

// NewConfig create a new Config
func NewConfig() *Config {
	return wrapper(dag.New(def))
}

func wrapper(e *dag.Entity) *Config {
	return &Config{Entity: e}
}

func simpleResolvers(repo repository.Repo) entity.Resolvers {
	return entity.Resolvers{
		&identity.Identity{}: identity.NewSimpleResolver(repo),
	}
}

// Read will read the config from a repository, or return an entity.ErrNotFound
// if there is none
func Read(ctx context.Context, repo repository.Repo) (*Config, error) {
	return ReadWithResolver(ctx, repo, simpleResolvers(repo))
}

// Head returns the commit the reference of the config points to, or
// repository.ErrNotFound if there is none
func Head(ctx context.Context, repo repository.Repo) (repository.Hash, error) {
	return repo.ResolveRef(ctx, def.Namespace, def.SingletonKey)
}

// ReadWithResolver will read the config from a repository, with custom resolvers
func ReadWithResolver(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers) (*Config, error) {
	return dag.ReadSingleton(ctx, def, wrapper, repo, resolvers)
}

// EnsureClocks makes sure that the clocks ordering the config are usable, and
// rebuilds them from the config if not. See dag.EnsureClocks.
func EnsureClocks(ctx context.Context, repo repository.Repo) error {
	return dag.EnsureClocks(ctx, def, repo)
}

// Validate check if the Config data is valid
func (c *Config) Validate() error {
	if err := c.Entity.Validate(); err != nil {
		return err
	}

	// The very first Op should be a CreateOp
	firstOp := c.FirstOp()
	if firstOp == nil || firstOp.Type() != CreateOp {
		return fmt.Errorf("first operation should be a Create op")
	}

	// Check that there is no more CreateOp op
	for i, op := range c.Entity.Operations() {
		if i == 0 {
			continue
		}
		if op.Type() == CreateOp {
			return fmt.Errorf("only one Create op allowed")
		}
	}

	return nil
}

// Append add a new Operation to the Config
func (c *Config) Append(op Operation) {
	c.Entity.Append(op)
}

// CommitOperations writes the given operations, see dag.Entity.CommitOperations
func (c *Config) CommitOperations(ctx context.Context, repo repository.Repo, ops []Operation) error {
	// Sanity check: the Create op comes first, and only once. Operations committed
	// here are built by code, not read from elsewhere: breaking that is a misuse.
	creation := c.Entity.LastCommit() == ""
	converted := make([]dag.Operation, len(ops))
	for i, op := range ops {
		isCreate := op.Type() == CreateOp
		if creation && i == 0 && !isCreate {
			panic("the first operation of a new config should be a Create op")
		}
		if isCreate && (!creation || i > 0) {
			panic("only one Create op allowed in a config")
		}
		converted[i] = op
	}
	return c.Entity.CommitOperations(ctx, repo, converted)
}

// Operations return the ordered operations
func (c *Config) Operations() []Operation {
	source := c.Entity.Operations()
	result := make([]Operation, len(source))
	for i, op := range source {
		result[i] = op.(Operation)
	}
	return result
}

// Compile the config in an easily usable snapshot
func (c *Config) Compile() *Snapshot {
	snap := &Snapshot{}

	for _, op := range c.Operations() {
		if !allowed(op, snap) {
			continue
		}
		op.Apply(snap)
		snap.Operations = append(snap.Operations, op)
	}

	return snap
}

// allowed tells whether the author of an operation may make it, given the config
// as it is before that operation. A refused operation is left out of the snapshot.
// Anyone may change any setting.
func allowed(_ Operation, _ *Snapshot) bool {
	return true
}

// FirstOp lookup for the very first operation of the config.
// For a valid Config, this operation should be a CreateOp
func (c *Config) FirstOp() Operation {
	if fo := c.Entity.FirstOp(); fo != nil {
		return fo.(Operation)
	}
	return nil
}

// LastOp lookup for the very last operation of the config.
// For a valid Config, should never be nil
func (c *Config) LastOp() Operation {
	if lo := c.Entity.LastOp(); lo != nil {
		return lo.(Operation)
	}
	return nil
}
