## git-bug bridge push

Push updates to remote bug tracker

### Synopsis

Push updates to remote bug tracker.

By default, bugs imported from another bug tracker are not pushed. With
--foreign they are mirrored too, which makes the same bug available in
several trackers. The first foreign push needs a remote tracker without any
issue or pull request; later ones need a tracker holding only mirrored
issues. Mirrored content appears as written by the token owner, with a note
naming the original author.

```
git-bug bridge push [NAME] [flags]
```

### Options

```
      --foreign   Also push bugs imported from other bug trackers (needs an empty or mirror-only remote tracker)
  -v, --verbose   Explain why issues were not pushed  -h, --help      help for push
```

### SEE ALSO

* [git-bug bridge](git-bug_bridge.md)	 - List bridges to other bug trackers

