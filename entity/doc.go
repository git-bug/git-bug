// Package entity contains the base common code to define an entity stored
// in a chain of git objects, supporting actions like Push, Pull and Merge.
package entity

// TODO: Bug and Identity are very similar, right ? I expect that this package
// will eventually hold the common code to define an entity and the related
// helpers, errors and so on. When this work is done, it will become easier
// to add new entities, for example to support pull requests.
