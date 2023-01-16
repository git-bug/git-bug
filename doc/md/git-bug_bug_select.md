## git-bug bug select

Select a bug for implicit use in future commands

### Synopsis

Select a bug for implicit use in future commands.

This command allows you to omit any bug ID argument, for example:
  git bug show
instead of
  git bug show 2f153ca

The complementary command is "git bug deselect" performing the opposite operation.


```
git-bug bug select BUG_ID [flags]
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

* [git-bug bug](git-bug_bug.md)	 - List bugs

