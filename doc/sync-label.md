# Sync Label

## Motivation

I am currently using git-bug to manage my bug/feature/task list for a number
of projects. Some of those projects have "external" bug trackers (e.g.
my day job, or github mirrors of side projects).

My personal bug repo covers 20 projects. I prefer the experience of
interacting with a single git-bug session in `termui` versus a half-dozen
different laggy-ass hyper-ajax web 2.0 online trackers.

I would love to sync my personal bug repo with these external trackers but
there are a couple of problems:

1. Our JIRA at work has millions of bugs. I obviously don't interact with
   all of them. I only want to sync bugs that are "relevant" to me... where
   "relevant" can presumably be defined by matching some criteria on metadata
   in JIRA.
2. Our gitlab at work likewise has thousands of open pull requests. I'm not sure
   git-bug is necessarily the best tool for doing code-review right now, but
   at the very least it might be nice to trach which code-reviews that I'm
   involved in are still open and require attention.
3. Some of the bugs in my repo really shouldn't be pushed to a public location
   because they might contain "sensitive" information about my employer.
4. I don't want bugs for irrelevant projects to pollute the issue list on
   my github mirrors.

## Proposed Solution

My proposed solution is to restict synchronization to bugs that match certain
filters, and to modify bugs as they are imported/exported according to rules
in the git config. Initially, the filter will support matching labels, and
modifiers will support adding or removing labels.

NOTE: I may also take a stab at implementing a JIRA bridge.

### Case Study

Consider the project `git-bug-test` which has a github mirror at
`cheshirekow.com/git-bug-test`. Ideally we would specify in our git-config
something like this:

```config
[git-bug "bridge.github.github-git-bug-test"]
	user = cheshirekow
	project = git-bug-test
	token = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	import-filter="assignee=cheshirekow"
	import-mod="label=+test,label=+github"
	export-filter="labels=test&github"
	export-mod="label=-test,label=-github"
```

with the following semantics:

**import**: Only issues matching `assignee=cheshirekow` will be queried
from github. Any issue that is created or updated is given the labels `test`
and `github` in `git-bug`.

**export**: Only bugs which have *both* labels `test` and `github` will be
considered for export. When we export the bugs, we remove the labels `test`
and `github`. Or, rather, we do not synchronize those labels with github.

### Open questions:

To what extent can this implementation be generic between different remote
bug systems? Each remote system will have different sets of metadata that we
may or may not be able to filter on.

Should we validate that `import-mod` matches `export-filter`? If not then
bugs can be pulled but not pushed. Maybe that is desirable?

It looks like git-bug already stores github specific metadata (for the github
bridge). Does that mean we don't need the `import-mod`? I think we do still
need it because we want to be able to *find* those bugs after import, and we
would like them to be collated with bugs we've created locally.

The exporter looks like it doesn't update bugs after the initial export. See
`github/export.go:207`. I think there was some mesage in the docs about
incremental export is "almost supported".

## Implementation notes

### Filtering github issues on input

The main loop-over-bugs is in `github/import.go:51`. It uses the
`githubImporter.iterator` to step through bugs. The iterator is implemented
in `github/iterator.go`. It uses the `github.com/shurcooL/githubv4` go
package. I think the query is composed of the `variables["xxx"] =` assignments
starting on line `github/iterator.go:102` through line `134`. I think this is
where we add additional query parameters.

The [github api documentation][1] describes the GraphQL schema.

I think the query starts with an "issueTimelineQuery" which, I think, returns
issues in timeline order. In the graphQL schema it looks like the root query
is [`repository`][2] and it's [`issues`][3] connection to [`Issue`][4] objects.

[1]: https://developer.github.com/v4/object/issue/
[2]: https://developer.github.com/v4/object/repository/
[3]: https://developer.github.com/v4/object/issueconnection/
[4]: https://developer.github.com/v4/object/issue/

I'm not sure if it would be easier to add query parameters (I don't know
GraphQL) or just to filter in the iterator. Maybe I can start there. The
main iterator interface is `NextIssue` at `github/iterator.go:172`. I don't
fully follow what's happening here but the issue is actually retrieved by
`queryIssue(()` at `github/iterator.go:153` and it appears that `NextIssue`
returns true if the iterator has an issue cached (and it may also perform a
query to fetch and an issue if needed).

The issue itself is retrieved via `IssueValue()` at `github/iterator.go:203`,
which just returns `i.timeline.query.Repository.Issues.Nodes[0]`.

I think the easiest thing to do is to modify `github.iterator`, moving
`NextIssue` deeper down the stack and add a wrapper `NextFilteredIssue`.
`NextFilteredIssue` will call next issue, and then inspect the issue against
filters. If it is accepted it returns, like `NextIssue` would have. Otherwise,
it continues to iterate over `NextIssue` until either the issues run out or
we find the next one that matches.

### Assigning labels on import

The creation of bugs in `git-bug` I think happens within `ensureIssue`,
defined at `import.go:87`. This in turn calls
`cache.RepoCache.ResolveBugCreateMetadata` defined at `cache/repo_cache.go:450`.
`ResolveBug` returns the bug if it exists, otherwise returns an error. The
createion of a new bug appears to be done by `cache.NewBugRaw`. I think the
best place to add modification logic is at the end of `ensureIssue`.
`ensureIssue()` returns the bug that was either found or newly created. Right
before returning it we could make any necessary modifications, adding or
removing labels, etc.

### Filtering bugs on export

I think the main function for exporting bugs is `ExportAll()` in
`github/export.go:82`. The iteration I think happens on on line 123:

```
		allBugsIds := repo.AllBugsIds()

		for _, id := range allBugsIds {
```

The actual export appears to go through `exportBug()` at line 152.
`exportBug` is defined at line 162. There already appears to be some filtering.
On line `github/export.go:177` we see:

```
	// skip bug if origin is not allowed
	origin, ok := snapshot.GetCreateMetadata(keyOrigin)
	if ok && origin != target {
		out <- core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
```

`GetCreateMetadata` seems appears to be a dictionary lookup, where
the key is added if it doesn't exist. This looks like it wont push bugs that
were pulled from a different github repository.

Seems like this is a good place to do our filtering too. We have the bug so we
can just return if the bug doesn't match the filter for that repository.

### Assigning (removing) labels on export

The github issue apppears to be created at `github/export.go:222`, the call
to `createGithubIssue()`. The output channel get's sent a `NewExportBug`
object but it seems to contain the bugs internal id... not the github id...
so I'm not sure if that is is relevant.

I think the remaining bug data is sent in the loop starting at
`github/export.go:253`:

```
	for _, op := range snapshot.Operations[1:] {
		// ignore SetMetadata operations
		if _, ok := op.(*bug.SetMetadataOperation); ok {
			continue
		}

...
```

I recall reading that `git-bug` stores a list of "update operations" for each
bug, so I think this loop is walking through the list of operations and
updating the github issue with all of those operations. The switch statement
on `github/export.go:265`:

```
		var id, url string
		switch op.(type) {
		case *bug.AddCommentOperation:
			opr := op.(*bug.AddCommentOperation)
...
```

seems to enumerate the different types of operations and presumably updates
the github bug according to the operation. `github/export.go:361` I think is
the case of interest to us. It's the case for `bug.LabelChangeOperation`.
Perhaps we can just look here and if the label matches any of the ones that
we are removing, we simply fall out of the switch without calling
`updateGithubIssueLabels()`.


### Adding config options

I think the configuration options are managed internally through the
`core.Configuration` object that gets passed to the importers and exporters.
The github exporter, for example, stores this as it's `.conf` member. The
object seems to act like a dictionary with array accessor. It seems that
`ge.conf[keyToken]` is used a few places throughout the exporter. `keyToken`
is a const defined as `"token"` in `config.go` so I think we just access the
config as a dictionary. Presumably if we add the config options above we
would just access them as `ge.conf["import-filter"]` (though probably use a
constant to aid static analysis).

### Filter and modification logic

We'll need to parse out the filter and modification specification from the
config file, and then turn that into a callable that can classify bugs
(in the case of "-filter" options) modify bugs (in the case of "import-mod")
or, classify operations (in the case of "export-mod").

... time to brush up on my `golang`.
