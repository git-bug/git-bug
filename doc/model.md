# Data model

The biggest problem when creating a distributed bug tracker is that there is no central authoritative server (doh!). This imply some constraint.

## Anybody can create and edit bugs at the same time as you

To deal with this problem, you need a way to merge these changes in a meaningful way.

Instead of storing directly the final bug data, we store a series of edit `Operation`. One of such operation could looks like this:

```json
{
  "type": "SET_TITLE",
  "title": "This title is better"
}
```

Note: Json provided for readability. Internally it's a golang struct.

These `Operation` are aggregated in an `OperationPack`, a simple array. An `OperationPack` represent an edit session of a bug. We store this pack in git as a git `Blob`, that is arbitrary serialized data.

To reference our `OperationPack` we create a git `Tree`, that is a tree of reference (`Blob` of sub-`Tree`). If our edit operation include a media (for instance in a message), we can store that media as a `Blob` and reference it here under `"/media"`. 

To complete the picture, we create a git `Commit` that reference our `Tree`. Each time we add more `Operation` to our bug, we add a new `Commit` with the same data-structure to form a chain of `Commit`.

This chain of `Commit` is made available as a git `Reference` under `refs/bugs/<bug-id>`. We can later use this reference to push our data to a git remote. As git will push any data needed as well, everything will be pushed to the remote including the medias.

For convenience and performance, each `Tree` reference the very first `OperationPack` of the bug under `"/root"`. That way we can easily access the very first `Operation`, the `CREATE` operation. This operation contains important data for the bug like the author.

Here is the complete picture:

```
 refs/bugs/<bug-id>
       |
       |
       |
 +-----------+          +-----------+           "ops"    +-----------+
 |  Commit   |---------->   Tree    |-------|------------|   Blob    | (OperationPack)
 +-----------+          +-----------+       |            +-----------+
       |                                    |
       |                                    |
       |                                    |   "root"   +-----------+ 
 +-----------+          +-----------+       |------------|   Blob    | (OperationPack)
 |  Commit   |---------->   Tree    |       |            +-----------+
 +-----------+          +-----------+       |
       |                                    |
       |                                    |   "media"  +-----------+    +-----------+
       |                                    +------------|   Tree    |--->|   Blob    | bug.jpg
 +-----------+          +-----------+                    +-----------+    +-----------+
 |  Commit   |---------->   Tree    |
 +-----------+          +-----------+
```

Now that we have this, we can easily merge our bugs without conflict. When pulling bug's update from a remote, we will simply add our new operations (that is, new `Commit`), if any, at the end of the chain. In git terms, it's just a `rebase`.

## You can't have a simple consecutive index for your bugs

TODO: complete when stable in the code

--> essentially a semi-random ID + truncation for human consumption

## You can't rely on the time provided by other people (their clock might by off) for anything other than just display

TODO: complete when stable in the code

--> inside a bug, we have a de facto ordering with the chain of commit

--> to order bugs, we can use a Lamport clock + timestamp when concurrent editing
