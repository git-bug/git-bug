## git-bug cache rebuild

Rebuild the local cache from the git data

### Synopsis

Rebuild the local cache from the git data.

The cache is not aware of changes made to git-bug's references by other tools, for example when fetching or pushing them with git directly. Use this command to make git-bug reflect them.

```
git-bug cache rebuild [flags]
```

### Options

```
  -h, --help   help for rebuild
```

### SEE ALSO

* [git-bug cache](git-bug_cache.md)	 - Manage the local cache

