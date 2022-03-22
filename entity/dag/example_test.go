package dag_test

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

// This file explains how to define a replicated data structure, stored and using git as a medium for
// synchronisation. To do this, we'll use the entity/dag package, which will do all the complex handling.
//
// The example we'll use here is a small shared configuration with two fields. One of them is special as
// it also defines who is allowed to change said configuration. Note: this example is voluntarily a bit
// complex with operation linking to identities and logic rules, to show that how something more complex
// than a toy would look like. That said, it's still a simplified example: in git-bug for example, more
// layers are added for caching, memory handling and to provide an easier to use API.
//
// Let's start by defining the document/structure we are going to share:

// Snapshot is the compiled view of a ProjectConfig
type Snapshot struct {
	// Administrator is the set of users with the higher level of access
	Administrator map[identity.Interface]struct{}
	// SignatureRequired indicate that all git commit need to be signed
	SignatureRequired bool
}

// HasAdministrator returns true if the given identity is included in the administrator.
func (snap *Snapshot) HasAdministrator(i identity.Interface) bool {
	for admin, _ := range snap.Administrator {
		if admin.Id() == i.Id() {
			return true
		}
	}
	return false
}

// Now, we will not edit this configuration directly. Instead, we are going to apply "operations" on it.
// Those are the ones that will be stored and shared. Doing things that way allow merging concurrent editing
// and deal with conflict.
//
// Here, we will define three operations:
// - SetSignatureRequired is a simple operation that set or unset the SignatureRequired boolean
// - AddAdministrator is more complex and add a new administrator in the Administrator set
// - RemoveAdministrator is the counterpart the remove administrators
//
// Note: there is some amount of boilerplate for operations. In a real project, some of that can be
// factorized and simplified.

// Operation is the operation interface acting on Snapshot
type Operation interface {
	dag.Operation

	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot *Snapshot)
}

type OperationType int

const (
	_ OperationType = iota
	SetSignatureRequiredOp
	AddAdministratorOp
	RemoveAdministratorOp
)

// SetSignatureRequired is an operation to set/unset if git signature are required.
type SetSignatureRequired struct {
	author        identity.Interface
	OperationType OperationType `json:"type"`
	Value         bool          `json:"value"`
}

func NewSetSignatureRequired(author identity.Interface, value bool) *SetSignatureRequired {
	return &SetSignatureRequired{author: author, OperationType: SetSignatureRequiredOp, Value: value}
}

func (ssr *SetSignatureRequired) Id() entity.Id {
	// the Id of the operation is the hash of the serialized data.
	// we could memorize the Id when deserializing, but that will do
	data, _ := json.Marshal(ssr)
	return entity.DeriveId(data)
}

func (ssr *SetSignatureRequired) Validate() error {
	if ssr.author == nil {
		return fmt.Errorf("author not set")
	}
	return ssr.author.Validate()
}

func (ssr *SetSignatureRequired) Author() identity.Interface {
	return ssr.author
}

// Apply is the function that makes changes on the snapshot
func (ssr *SetSignatureRequired) Apply(snapshot *Snapshot) {
	// check that we are allowed to change the config
	if _, ok := snapshot.Administrator[ssr.author]; !ok {
		return
	}
	snapshot.SignatureRequired = ssr.Value
}

// AddAdministrator is an operation to add a new administrator in the set
type AddAdministrator struct {
	author        identity.Interface
	OperationType OperationType        `json:"type"`
	ToAdd         []identity.Interface `json:"to_add"`
}

// addAdministratorJson is a helper struct to deserialize identities with a concrete type.
type addAdministratorJson struct {
	ToAdd []identity.IdentityStub `json:"to_add"`
}

func NewAddAdministratorOp(author identity.Interface, toAdd ...identity.Interface) *AddAdministrator {
	return &AddAdministrator{author: author, OperationType: AddAdministratorOp, ToAdd: toAdd}
}

func (aa *AddAdministrator) Id() entity.Id {
	// we could memorize the Id when deserializing, but that will do
	data, _ := json.Marshal(aa)
	return entity.DeriveId(data)
}

func (aa *AddAdministrator) Validate() error {
	// Let's enforce an arbitrary rule
	if len(aa.ToAdd) == 0 {
		return fmt.Errorf("nothing to add")
	}
	if aa.author == nil {
		return fmt.Errorf("author not set")
	}
	return aa.author.Validate()
}

func (aa *AddAdministrator) Author() identity.Interface {
	return aa.author
}

// Apply is the function that makes changes on the snapshot
func (aa *AddAdministrator) Apply(snapshot *Snapshot) {
	// check that we are allowed to change the config ... or if there is no admin yet
	if !snapshot.HasAdministrator(aa.author) && len(snapshot.Administrator) != 0 {
		return
	}
	for _, toAdd := range aa.ToAdd {
		snapshot.Administrator[toAdd] = struct{}{}
	}
}

// RemoveAdministrator is an operation to remove an administrator from the set
type RemoveAdministrator struct {
	author        identity.Interface
	OperationType OperationType        `json:"type"`
	ToRemove      []identity.Interface `json:"to_remove"`
}

// removeAdministratorJson is a helper struct to deserialize identities with a concrete type.
type removeAdministratorJson struct {
	ToRemove []identity.Interface `json:"to_remove"`
}

func NewRemoveAdministratorOp(author identity.Interface, toRemove ...identity.Interface) *RemoveAdministrator {
	return &RemoveAdministrator{author: author, OperationType: RemoveAdministratorOp, ToRemove: toRemove}
}

func (ra *RemoveAdministrator) Id() entity.Id {
	// the Id of the operation is the hash of the serialized data.
	// we could memorize the Id when deserializing, but that will do
	data, _ := json.Marshal(ra)
	return entity.DeriveId(data)
}

func (ra *RemoveAdministrator) Validate() error {
	// Let's enforce some rules. If we return an error, this operation will be
	// considered invalid and will not be included in our data.
	if len(ra.ToRemove) == 0 {
		return fmt.Errorf("nothing to remove")
	}
	if ra.author == nil {
		return fmt.Errorf("author not set")
	}
	return ra.author.Validate()
}

func (ra *RemoveAdministrator) Author() identity.Interface {
	return ra.author
}

// Apply is the function that makes changes on the snapshot
func (ra *RemoveAdministrator) Apply(snapshot *Snapshot) {
	// check if we are allowed to make changes
	if !snapshot.HasAdministrator(ra.author) {
		return
	}
	// special rule: we can't end up with no administrator
	stillSome := false
	for admin, _ := range snapshot.Administrator {
		if admin != ra.author {
			stillSome = true
			break
		}
	}
	if !stillSome {
		return
	}
	// apply
	for _, toRemove := range ra.ToRemove {
		delete(snapshot.Administrator, toRemove)
	}
}

// Now, let's create the main object (the entity) we are going to manipulate: ProjectConfig.
// This object wrap a dag.Entity, which makes it inherit some methods and provide all the complex
// DAG handling. Additionally, ProjectConfig is the place where we can add functions specific for that type.

type ProjectConfig struct {
	// this is really all we need
	*dag.Entity
}

func NewProjectConfig() *ProjectConfig {
	return &ProjectConfig{Entity: dag.New(def)}
}

// a Definition describes a few properties of the Entity, a sort of configuration to manipulate the
// DAG of operations
var def = dag.Definition{
	Typename:             "project config",
	Namespace:            "conf",
	OperationUnmarshaler: operationUnmarshaller,
	FormatVersion:        1,
}

// operationUnmarshaller is a function doing the de-serialization of the JSON data into our own
// concrete Operations. If needed, we can use the resolver to connect to other entities.
func operationUnmarshaller(author identity.Interface, raw json.RawMessage, resolver identity.Resolver) (dag.Operation, error) {
	var t struct {
		OperationType OperationType `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var value interface{}

	switch t.OperationType {
	case AddAdministratorOp:
		value = &addAdministratorJson{}
	case RemoveAdministratorOp:
		value = &removeAdministratorJson{}
	case SetSignatureRequiredOp:
		value = &SetSignatureRequired{}
	default:
		panic(fmt.Sprintf("unknown operation type %v", t.OperationType))
	}

	err := json.Unmarshal(raw, &value)
	if err != nil {
		return nil, err
	}

	var op Operation

	switch value := value.(type) {
	case *SetSignatureRequired:
		value.author = author
		op = value
	case *addAdministratorJson:
		// We need something less straightforward to deserialize and resolve identities
		aa := &AddAdministrator{
			author:        author,
			OperationType: AddAdministratorOp,
			ToAdd:         make([]identity.Interface, len(value.ToAdd)),
		}
		for i, stub := range value.ToAdd {
			iden, err := resolver.ResolveIdentity(stub.Id())
			if err != nil {
				return nil, err
			}
			aa.ToAdd[i] = iden
		}
		op = aa
	case *removeAdministratorJson:
		// We need something less straightforward to deserialize and resolve identities
		ra := &RemoveAdministrator{
			author:        author,
			OperationType: RemoveAdministratorOp,
			ToRemove:      make([]identity.Interface, len(value.ToRemove)),
		}
		for i, stub := range value.ToRemove {
			iden, err := resolver.ResolveIdentity(stub.Id())
			if err != nil {
				return nil, err
			}
			ra.ToRemove[i] = iden
		}
		op = ra
	default:
		panic(fmt.Sprintf("unknown operation type %T", value))
	}

	return op, nil
}

// Compile compute a view of the final state. This is what we would use to display the state
// in a user interface.
func (pc ProjectConfig) Compile() *Snapshot {
	// Note: this would benefit from caching, but it's a simple example
	snap := &Snapshot{
		// default value
		Administrator:     make(map[identity.Interface]struct{}),
		SignatureRequired: false,
	}
	for _, op := range pc.Operations() {
		op.(Operation).Apply(snap)
	}
	return snap
}

// Read is a helper to load a ProjectConfig from a Repository
func Read(repo repository.ClockedRepo, id entity.Id) (*ProjectConfig, error) {
	e, err := dag.Read(def, repo, identity.NewSimpleResolver(repo), id)
	if err != nil {
		return nil, err
	}
	return &ProjectConfig{Entity: e}, nil
}

func Example_entity() {
	// Note: this example ignore errors for readability
	// Note: variable names get a little confusing as we are simulating both side in the same function

	// Let's start by defining two git repository and connecting them as remote
	repoRenePath, _ := os.MkdirTemp("", "")
	repoIsaacPath, _ := os.MkdirTemp("", "")
	repoRene, _ := repository.InitGoGitRepo(repoRenePath)
	repoIsaac, _ := repository.InitGoGitRepo(repoIsaacPath)
	_ = repoRene.AddRemote("origin", repoIsaacPath)
	_ = repoIsaac.AddRemote("origin", repoRenePath)

	// Now we need identities and to propagate them
	rene, _ := identity.NewIdentity(repoRene, "René Descartes", "rene@descartes.fr")
	isaac, _ := identity.NewIdentity(repoRene, "Isaac Newton", "isaac@newton.uk")
	_ = rene.Commit(repoRene)
	_ = isaac.Commit(repoRene)
	_ = identity.Pull(repoIsaac, "origin")

	// create a new entity
	confRene := NewProjectConfig()

	// add some operations
	confRene.Append(NewAddAdministratorOp(rene, rene))
	confRene.Append(NewAddAdministratorOp(rene, isaac))
	confRene.Append(NewSetSignatureRequired(rene, true))

	// Rene commits on its own repo
	_ = confRene.Commit(repoRene)

	// Isaac pull and read the config
	_ = dag.Pull(def, repoIsaac, identity.NewSimpleResolver(repoIsaac), "origin", isaac)
	confIsaac, _ := Read(repoIsaac, confRene.Id())

	// Compile gives the current state of the config
	snapshot := confIsaac.Compile()
	for admin, _ := range snapshot.Administrator {
		fmt.Println(admin.DisplayName())
	}

	// Isaac add more operations
	confIsaac.Append(NewSetSignatureRequired(isaac, false))
	reneFromIsaacRepo, _ := identity.ReadLocal(repoIsaac, rene.Id())
	confIsaac.Append(NewRemoveAdministratorOp(isaac, reneFromIsaacRepo))
	_ = confIsaac.Commit(repoIsaac)
}
