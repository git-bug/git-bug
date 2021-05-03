## git-bug webui

Launch the web UI.

### Synopsis

Launch the web UI.

Available git config:
  git-bug.webui.open [bool]: control the automatic opening of the web UI in the default browser


```
git-bug webui [flags]
```

### Options

```
      --host string    Network address or hostname to listen to (default to 127.0.0.1) (default "127.0.0.1")
      --open           Automatically open the web UI in the default browser
      --no-open        Prevent the automatic opening of the web UI in the default browser
  -p, --port int       Port to listen to (default to random available port)
      --read-only      Whether to run the web UI in read-only mode
  -q, --query string   The query to open in the web UI bug list
  -h, --help           help for webui
```

### SEE ALSO

* [git-bug](git-bug.md)	 - A bug tracker embedded in Git.

