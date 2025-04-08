# Search filters

When performing a search (e.g. listing issues), you can use different qualifiers
to narrow the results. This page provides an overview of these filters, and how
to use them.

<!-- mdformat-toc start --slug=github --maxlevel=4 --minlevel=2 -->

- [Overview](#overview)
- [Filtering](#filtering)
  - [Filtering by status](#filtering-by-status)
  - [Filtering by author](#filtering-by-author)
  - [Filtering by participant](#filtering-by-participant)
  - [Filtering by actor](#filtering-by-actor)
  - [Filtering by label](#filtering-by-label)
  - [Filtering by title](#filtering-by-title)
  - [Filtering by missing feature](#filtering-by-missing-feature)
- [Sorting](#sorting)
  - [Sort by Id](#sort-by-id)
  - [Sort by Creation time](#sort-by-creation-time)
  - [Sort by Edit time](#sort-by-edit-time)

<!-- mdformat-toc end -->

## Overview<a name="overview"></a>

The query filters in `git-bug` have a familiar look and feel:

```
status:open sort:edit
```

**Key things to know**

- All queries are case insensitive
- You can combine as many qualifiers as you want
- If you have a space in your qualifier, be sure to wrap it in double quotes. As
  an example, `author:"René Descartes"` would filter for issues opened by
  `René Descartes`, whereas `author:René Descartes` filter for `René` as the
  author and return issues that contain `Descartes` somewhere in the title,
  description, or comments.
- Instead of a complete ID, you can use any prefix length, as long as it is long
  enough to be unique (similar to git commit hashes). For example,
  `participant=9ed1a` would match against participants with an ID of
  `9ed1af428...` and `9ed1ae24a...`

## Filtering<a name="filtering"></a>

### Filtering by status<a name="filtering-by-status"></a>

You can filter bugs based on their status.

| Qualifier       | Example                             |
| --------------- | ----------------------------------- |
| `status:open`   | `status:open` matches open bugs     |
| `status:closed` | `status:closed` matches closed bugs |

### Filtering by author<a name="filtering-by-author"></a>

You can filter based on the person who opened the bug.

| Qualifier      | Example                                                                          |
| -------------- | -------------------------------------------------------------------------------- |
| `author:QUERY` | `author:descartes` matches bugs opened by `René Descartes` or `Robert Descartes` |
|                | `author:"rené descartes"` matches bugs opened by `René Descartes`                |

### Filtering by participant<a name="filtering-by-participant"></a>

You can filter based on the person who participated in any activity related to
the bug (opened bug or added a comment).

| Qualifier           | Example                                                                                            |
| ------------------- | -------------------------------------------------------------------------------------------------- |
| `participant:QUERY` | `participant:descartes` matches bugs opened or commented by `René Descartes` or `Robert Descartes` |
|                     | `participant:"rené descartes"` matches bugs opened or commented by `René Descartes`                |

### Filtering by actor<a name="filtering-by-actor"></a>

You can filter based on the person who interacted with the bug.

| Qualifier     | Example                                                                         |
| ------------- | ------------------------------------------------------------------------------- |
| `actor:QUERY` | `actor:descartes` matches bugs edited by `René Descartes` or `Robert Descartes` |
|               | `actor:"rené descartes"` matches bugs edited by `René Descartes`                |

> [!NOTE]
> Interactions with issues include opening the bug, adding comments, adding or
> removing labels, etc.

### Filtering by label<a name="filtering-by-label"></a>

You can filter based on the bug's label.

| Qualifier     | Example                                                                   |
| ------------- | ------------------------------------------------------------------------- |
| `label:LABEL` | `label:prod` matches bugs with the label `prod`                           |
|               | `label:"Good first issue"` matches bugs with the label `Good first issue` |

### Filtering by title<a name="filtering-by-title"></a>

You can filter based on the bug's title.

| Qualifier     | Example                                                                        |
| ------------- | ------------------------------------------------------------------------------ |
| `title:TITLE` | `title:Critical` matches bugs with a title containing `Critical`               |
|               | `title:"Typo in string"` matches bugs with a title containing `Typo in string` |

### Filtering by missing feature<a name="filtering-by-missing-feature"></a>

You can filter bugs based on the absence of something.

| Qualifier  | Example                                |
| ---------- | -------------------------------------- |
| `no:label` | `no:label` matches bugs with no labels |

## Sorting<a name="sorting"></a>

You can sort results by adding a `sort:` qualifier to your query. “Descending”
means most recent time or largest ID first, whereas “Ascending” means oldest
time or smallest ID first.

Note: to deal with differently-set clocks on distributed computers, `git-bug`
uses a logical clock internally rather than timestamps to order bug changes over
time. That means that the timestamps recorded might not match the returned
ordering. To learn more, we encourage you to read about \[time \]\[data-model\].

### Sort by Id<a name="sort-by-id"></a>

| Qualifier                  | Example                                               |
| -------------------------- | ----------------------------------------------------- |
| `sort:id-desc`             | `sort:id-desc` will sort bugs by their descending Ids |
| `sort:id` or `sort:id-asc` | `sort:id` will sort bugs by their ascending Ids       |

### Sort by Creation time<a name="sort-by-creation-time"></a>

You can sort bugs by their creation time.

| Qualifier                               | Example                                                             |
| --------------------------------------- | ------------------------------------------------------------------- |
| `sort:creation` or `sort:creation-desc` | `sort:creation` will sort bugs by their descending creation time    |
| `sort:creation-asc`                     | `sort:creation-asc` will sort bugs by their ascending creation time |

### Sort by Edit time<a name="sort-by-edit-time"></a>

You can sort bugs by their edit time.

| Qualifier                       | Example                                                             |
| ------------------------------- | ------------------------------------------------------------------- |
| `sort:edit` or `sort:edit-desc` | `sort:edit` will sort bugs by their descending last edition time    |
| `sort:edit-asc`                 | `sort:edit-asc` will sort bugs by their ascending last edition time |

______________________________________________________________________

##### See more

- [A description of the data model][docs/design/model]
- [How to use bridges][docs/usage/bridges]
- [Learn about the native interfaces][docs/usage/interfaces]
- [Understanding the workflow models][docs/usage/workflows]
- :house: [Documentation home][docs/home]

[docs/design/model]: ../design/data-model.md#you-cant-rely-on-the-time-provided-by-other-people-their-clock-might-by-off-for-anything-other-than-just-display
[docs/home]: ../README.md
[docs/usage/bridges]: ./bridges.md
[docs/usage/interfaces]: ./interfaces.md
[docs/usage/workflows]: ./workflows.md
