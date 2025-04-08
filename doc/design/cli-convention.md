## Pattern

CLI commands should consistently follow this pattern:

```
xxx                 --> list xxx things if list, otherwise show one
xxx new             --> create thing
xxx rm              --> delete thing
xxx show ID         --> show one
xxx show            --> show one with "select" implied ID
xxx yyy             --> action commands for that thing, or subcommand
xxx select|deselect --> select/deselect implied ID
```
