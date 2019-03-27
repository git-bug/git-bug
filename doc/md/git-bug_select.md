## git-bug select

Select a bug for implicit use in future commands.

### Synopsis

Select a bug for implicit use in future commands.

This command allows you to omit any bug <id> argument, for example:
  git bug show
instead of
  git bug show 2f153ca

The complementary command is "git bug deselect" performing the opposite operation.


```
git-bug select <id> [flags]
```

### Examples

```
git bug select 2f15
git bug comment
git bug status

```

### Options

```
  -h, --help   help for select
```

### SEE ALSO

* [git-bug](git-bug.md)	 - A bug tracker embedded in Git.

