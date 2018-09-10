## git-bug ls

List bugs

### Synopsis

Display a summary of each bugs.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language or with flags.

```
git-bug ls [<query>] [flags]
```

### Examples

```
List open bugs sorted by last edition with a query:
git bug ls "status:open sort:edit-desc"

List closed bugs sorted by creation with flags:
git bug ls --status closed --by creation

```

### Options

```
  -s, --status strings     Filter by status. Valid values are [open,closed]
  -a, --author strings     Filter by author
  -l, --label strings      Filter by label
  -n, --no strings         Filter by absence of something. Valid values are [label]
  -b, --by string          Sort the results by a characteristic. Valid values are [id,creation,edit] (default "creation")
  -d, --direction string   Select the sorting direction. Valid values are [asc,desc] (default "asc")
  -h, --help               help for ls
```

### SEE ALSO

* [git-bug](git-bug.md)	 - A bugtracker embedded in Git

