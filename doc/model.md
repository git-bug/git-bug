# Data model

The biggest problem when creating a distributed bug tracker is that there is no central authoritative server (doh!). This implies some constraints.

## Anybody can create and edit bugs at the same time as you

To deal with this problem, you need a way to merge these changes in a meaningful way.

Instead of storing directly the final bug data, we store a series of edit `Operation`. One of such operation could looks like this:

```json
{
  "type": "SET_TITLE",
  "author": {
    "name": "René Descartes",
    "email": "rene@descartes.fr"
  },
  "timestamp": 1533640589,
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
 +-----------+          +-----------+             "ops"    +-----------+
 |  Commit   |---------->   Tree    |---------+------------|   Blob    | (OperationPack)
 +-----------+          +-----------+         |            +-----------+
       |                                      |
       |                                      |
       |                                      |   "root"   +-----------+ 
 +-----------+          +-----------+         +------------|   Blob    | (OperationPack)
 |  Commit   |---------->   Tree    |-- ...   |            +-----------+
 +-----------+          +-----------+         |
       |                                      |
       |                                      |   "media"  +-----------+        +-----------+
       |                                      +------------|   Tree    |---+--->|   Blob    | bug.jpg
 +-----------+          +-----------+                      +-----------+   |    +-----------+
 |  Commit   |---------->   Tree    |-- ...                                |
 +-----------+          +-----------+                                      |    +-----------+
                                                                           +--->|   Blob    | demo.mp4
                                                                                +-----------+
```

Now that we have this, we can easily merge our bugs without conflict. When pulling bug's update from a remote, we will simply add our new operations (that is, new `Commit`), if any, at the end of the chain. In git terms, it's just a `rebase`.

## You can't have a simple consecutive index for your bugs

The same way git can't have a simple counter as identifier for it's commit as SVN do, we can't have consecutive identifiers for bugs.

`git-bug` use as identifier the hash of the first commit in the chain of commit of the bug. As this hash is ultimately computed with the content of the `CREATE` operation that include title, message and a timestamp, it will be unique and prevent collision.

The same way as git does, this hash is displayed truncated to a 7 characters string to human user. Note that when specifying a bug id in a command, you can enter as few character as you want as long as there is no ambiguity. If multiple bugs match your prefix, `git-bug` will complain and display the potential matches.

## You can't rely on the time provided by other people (their clock might by off) for anything other than just display

When in the context of a single bug, events are already ordered without the need of a timestamp. An `OperationPack` is an ordered array of operations. A chain of commit orders `OperationPack` with each other.

Now, to be able to order bugs by creation or last edition time, `git-bug` use a [Lamport logical clock](https://en.wikipedia.org/wiki/Lamport_timestamps). A Lamport clock is a simple counter of event. When a new bug is created, its creation time will be the highest time value we are aware of plus one. This declare a causality in the event and allow to order bugs.

When bugs are push/pull to a git remote, it might happen that bugs get the same logical time. This means that they were created or edited concurrently. In this case, `git-bug` will use the timestamp as a second layer of sorting. While the timestamp might be incorrect due to a badly set clock, the drift in sorting is bounded by the first sorting using the logical clock. That means that if users synchronize their bugs regularly, the timestamp will rarely be used, and should still provide a kinda accurate sorting when needed.

These clocks are stored in the chain of commit of each bug, as entries in each main git `Tree`. The first commit will have both a creation time and edit time clock, while a later commit will only have an edit time clock. A naive way could be to serialize the clock in a git `Blob` and reference it in the `Tree` as `"create-clock"` for example. The problem is that it would generate a lot of blobs that would need to be exchanged later for what is basically just a number.

Instead, the clock value is serialized directly in the `Tree` entry name (for example: `"create-clock-4"`). As a Tree entry need to reference something, we reference the git `Blob` with an empty content. As all of these entries will reference the same `Blob`, no network transfer is needed as long as you already have any bug in your repository.


Example of Tree of the first commit of a bug:
```
100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391	create-clock-14
100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391	edit-clock-137
100644 blob a020a85baa788e12699a4d83dd735578f0d78c75	ops
100644 blob a020a85baa788e12699a4d83dd735578f0d78c75	root 
```
Note that both `"ops"` and `"root"` entry reference the same OperationPack as it's the first commit in the chain.


Example of Tree of a later commit of a bug:
```
100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391	edit-clock-154
100644 blob 68383346c1a9503f28eec888efd300e9fc179ca0	ops
100644 blob a020a85baa788e12699a4d83dd735578f0d78c75	root
```
Note that the `"root"` entry still reference the same root OperationPack. Also, all the clocks reference the same empty `Blob`.
