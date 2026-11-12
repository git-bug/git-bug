package cache

// TODO: a cache for singleton entities (see dag.Definition.SingletonKey), to come with
// the first singleton entity type, the project config, which will shape its API.
//
// A singleton is not addressed by id: it lives in refs/<namespace>/<SingletonKey> and
// there is at most one per namespace. SubCache doesn't fit it: its index, excerpts,
// prefix resolution and search all exist to serve many entities found by id. The
// singleton cache is instead an alternate implementation of cacheMgmt, registered in
// RepoCache.subcaches next to the SubCaches:
//
//   - it holds at most one shared entity, read with dag.ReadSingleton; not found
//     means there is no singleton yet
//   - it follows the changes in its namespace through refsChanged, whatever the keys
//   - it commits through the usual CommitOperations / Repair retry loop. A Repair
//     failing with entity.ErrNotFound means that the reference holds another entity,
//     for instance one created concurrently: the caller then reads that one with
//     dag.ReadSingleton and applies its change again on top of it
//   - it merges, pushes and removes through the dag functions (dag.MergeAll,
//     dag.RemoveAll). The merge results of a singleton have an empty id, and
//     entity.MergeStatusConflict when the local and remote ones have no common
//     history
