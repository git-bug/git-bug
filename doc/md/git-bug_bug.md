## git-bug bug

List bugs

### Synopsis

Display a summary of each bugs.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language, flags, a natural language full text search, or a combination of the aforementioned.

```
git-bug bug [QUERY] [flags]
```

### Examples

```
List open bugs sorted by last edition with a query:
git bug status:open sort:edit-desc

List closed bugs sorted by creation with flags:
git bug --status closed --by creation

Do a full text search of all bugs:
git bug "foo bar" baz

Use queries, flags, and full text search:
git bug status:open --by creation "foo bar" baz

```

### Options

```
  -s, --status strings        Filter by status. Valid values are [open,closed]
  -a, --author strings        Filter by author
  -m, --metadata strings      Filter by metadata. Example: github-url=URL
  -p, --participant strings   Filter by participant
  -A, --actor strings         Filter by actor
  -l, --label strings         Filter by label
  -t, --title strings         Filter by title
  -n, --no strings            Filter by absence of something. Valid values are [label]
  -b, --by string             Sort the results by a characteristic. Valid values are [id,creation,edit] (default "creation")
  -d, --direction string      Select the sorting direction. Valid values are [asc,desc] (default "asc")
  -f, --format string         Select the output formatting style. Valid values are [default,plain,id,json,org-mode]
  -h, --help                  help for bug
```

### SEE ALSO

* [git-bug](git-bug.md)	 - A bug tracker embedded in Git
* [git-bug bug comment](git-bug_bug_comment.md)	 - List a bug's comments
* [git-bug bug deselect](git-bug_bug_deselect.md)	 - Clear the implicitly selected bug
* [git-bug bug label](git-bug_bug_label.md)	 - Display labels of a bug
* [git-bug bug new](git-bug_bug_new.md)	 - Create a new bug
* [git-bug bug rm](git-bug_bug_rm.md)	 - Remove an existing bug
* [git-bug bug select](git-bug_bug_select.md)	 - Select a bug for implicit use in future commands
* [git-bug bug show](git-bug_bug_show.md)	 - Display the details of a bug
* [git-bug bug status](git-bug_bug_status.md)	 - Display the status of a bug
* [git-bug bug title](git-bug_bug_title.md)	 - Display the title of a bug

