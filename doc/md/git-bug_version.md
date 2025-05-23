## git-bug version

Print version information

### Synopsis


Print version information.

Format:
  git-bug <version> [commit[/dirty]] <compiler version> <platform> <arch>

Format Description:
  <version> may be one of:
  	- A semantic version string, prefixed with a "v", e.g. v1.2.3
  	- "undefined" (if not provided, or built with an invalid version string)

  [commit], if present, is the commit hash that was checked out during the
  build. This may be suffixed with '/dirty' if there were local file
  modifications. This is indicative of your build being patched, or modified in
  some way from the commit.

  <compiler version> is the version of the go compiler used for the build.

  <platform> is the target platform (GOOS).

  <arch> is the target architecture (GOARCH).


```
git-bug version [flags]
```

### Examples

```
git bug version
```

### Options

```
  -h, --help   help for version
```

### SEE ALSO

* [git-bug](git-bug.md)	 - A bug tracker embedded in Git

