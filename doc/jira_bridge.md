# JIRA Bridge

## Design Notes

### One bridge = one project

There aren't any huge technical barriers requiring this, but since git-bug lacks
a notion of "project" there is no way to know which project to export new bugs
to as issues. Also, JIRA projects are first-class immutable metadata and so we
*must* get it right on export. Therefore the bridge is configured with the `Key`
for the project it is assigned to. It will only import bugs from that project.

### JIRA fields

The bridge currently does nothing to import any of the JIRA fields that don't
have `git-bug` equivalents ("Assignee", "sprint", "story points", etc).
Hopefully the bridge will be able to enable synchronization of these soon.

### Credentials

JIRA does not support user/personal access tokens. They have experimental
3-legged oauth support but that requires an API token for the app configured
by the server administrator. The only reliable authentication mechanism then is
the username/password and session-token mechanims. We can aquire a session
token programatically from the username/password but these are very short lived
(i.e. hours or less). As such the bridge currently requires an actual username
and password as user credentials. It supports three options:

1. Storing both username and password in a separate file referred to by
   the `git-config` (I like to use `.git/jira-credentials.json`)
2. Storing the username and password in clear-text in the git config
3. Storing the username only in the git config and asking for the password
   on each `push` or `pull`.

### Issue Creation Defaults

When a new issues is created in JIRA there are often certain mandatory fields
that require a value or the creation is rejected. In the issue create form on
the JIRA web interface, these are annotated as "required". The `issuetype` is
always required (e.g. "bug", "story", "task", etc). The set of required metadata
is configurable (in JIRA) per `issuetype` so the set might be different between
"bug" and "story", for example.

For now, the bridge only supports exporting issues as a single `issuetype`. If
no configuration is provied, then the default is `"id": "10001"` which is
`"story"` in the default set of issue types.

In addition to specifying the `issuetype` of issues created on export, the
bridge will also allow you to specify a constant global set of default values
for any additional required fields. See the configuration section below for the
syntax.

For longer term goals, see the section below on workflow validation

### Assign git-bug id to field during issue creation

JIRA allows for the inclusion of custom "fields" in all of their issues. The
JIRA bridge will store the JIRA issue "id" for any bugs which are synchronized
to JIRA, but it can also assign to a custom JIRA `field` the `git-bug` id. This
way the `git-bug` id can be displayed in the JIRA web interface and certain
integration activities become easier.

See the configuration section below on how to specify the custom field where the
JIRA bridge should write this information.


### Workflows and Transitions

JIRA issue states are subject to customizable "workflows" (project managers
apparently validate themselves by introducing developer friction). In general,
issues can only transition from one state to another if there is an edge between
them in the state graph (a.k.a. "workflow"). JIRA calls these edges
"transitions". Furthermore, each transition may include a set of mandatory
fields which must be set in order for the transition to succeed. For example the
transition of `"status"` from `"In Progress"` to `"Closed"` might required a
`"resolution"` (i.e. `"Fixed"` or `"Working as intended"`).

Dealing with complex workflows is going to be challenging. Some long-term
aspirations are described in the section below on "Workflow Validation".
Currently the JIRA bridge isn't very smart about transitions though, so you'll
need to tell it what you want it to do when importing and exporting a state
change (i.e. to "close" or "open" a bug). Currently the bridge accepts
configuration options which map the two `git-bug` statuses ("open", "closed") to
two JIRA statuses. On import, the JIRA status is mapped to a `git-bug` status
(if a mapping exists) and the `git-bug` status is assigned. On export, the
`git-bug` status is mapped to a JIRA status and if a mapping exists the bridge
will query the list of available transitions for the issue. If a transition
exists to the desired state the bridge will attempt to execute the transition.
It does not currently support assigning any fields during the transition so if
any fields are required the transition will fail during export and the status
will be out of sync.

### JIRA Changelog

Some operations on JIRA issues are visible in a timeline view known as the
`changelog`. The JIRA cloud product provides an
`/issue/{issueIdOrKey}/changelog` endpoint which provides a paginated view but
the JIRA server product does not. The changelog is visible by querying the issue
with the `expand=changelog` query parameter. Unfortunately in this case the
entire changelog is provided without paging.

Each changelog entry is identified with a unique string `id`, but within a
single changelog entry is a list of multilple fields that are modified. In other
words a single "event" might atomically change multiple fields. As an example,
when an issue is closed the `"status"` might change to `"closed"` and the
`"resolution"` might change to `"fixed'`.

When a changelog entry is imported by the JIRA bridge, each individual field
that was changed is treated as a separate `git-bug` operation. In other words a
single JIRA change event might create more than one `git-bug` operation.

However, when a `git-bug` operation is exported to JIRA it will only create a
single changelog entry. Furthermore, when we modify JIRA issues over the REST
API JIRA does not provide any information to associate that modification event
with the changelog. We must, therefore, herustically match changelog entries
against operations that we performed in order to not import them as duplicate
events. In order to assist in this matching proceess, the bridge will record the
JIRA server time of the response to the `POST` (as reported by the `"Date"`
response header). During import, we keep an iterator to the list of `git-bug`
operations for the bug mapped to the Jira issue. As we walk the JIRA changelog,
we keep the iterator pointing to the first operation with an annotation which is
*not before* that changelog entry. If the changelog entry is the result of an
exported `git-bug` operation, then this must be that operation. We then scan
through the list of changeitems (changed fields) in the changelog entry, and if
we can match a changed field to the candidate `git-bug` operation then we have
identified the match.

### Unlogged Changes

Comments (creation and edition) do not show up in the JIRA changelog. However
JIRA reports both a `created` and `updated` date for each comment. If we
import a comment which has an `updated` and `created` field which do not match,
then we treat that as a new comment edition. If we do not already have the
comment imported, then we import an empty comment followed by a comment edition.

Because comment editions are not uniquely identified in JIRA we identify them
in `git-bug` by concatinating the JIRA issue `id` with the `updated` time of
the edition.

### Workflow Validation (future)

The long-term plan for the JIRA bridge is to download and store the workflow
specifiations from the JIRA server. This includes the required metadata for
issue creation, and the status state graph, and the set of required metadata for
status transition.

When an existing `git-bug` is initially marked for export, the bridge will hook
in and validate the bug state against the required metadata. Then it will prompt
for any missing metadata using a set of UI components appropriate for the field
schema as reported by JIRA. If the user cancels then the bug will not be marked
for export.

When a bug already marked for JIRA export (including those that were imported)
is modified, the bridge will hook in and validate the modification against the
workflow specifications. It will prompt for any missing metadata as in the
creation process.

During export, the bridge will validate any export operations and skip them if
we know they will fail due to violation of the cached workflow specification
(i.e. missing required fields for a transition). A list of bugs "blocked for
export" will be available to query. A UI command will allow the user to inspect
and resolve any bugs that are "blocked for export".

## Configuration

As mentioned in the notes above, there are a few optional configuration fields
that can be set beyond those that are prompted for during the initial bridge
configuration. You can set these options in your `.git/config` file:

### Issue Creation Defaults

The format for this config entry is a JSON object containing fields you wish to
set during issue creation when exproting bugs. If you provide a value for this
configuration option, it must include at least the `"issuetype"` field, or
the bridge will not be able to export any new issues.

Let's say that we want bugs exported to JIRA to have a default issue type of
"Story" which is `issuetype` with id `10001`. Then we will add the following
entry to our git-config:

```
create-issue-defaults = {"issuetype":"10001"}
```

If you needed an additional required field `customfield_1234` and you wanted to
provide a default value of `"default"` then you would add the following to your
config:

```
create-issue-defaults = {"issuetype":"10001","customfield_1234":"default"}
```

Note that the content of this value is merged verbatim to the JSON object that
is `POST`ed to the JIRA rest API, so you can use arbitrary valid JSON.


### Assign git-bug id to field

If you want the bridge to fill a JIRA field with the `git-bug` id when exporting
issues, then provide the name of the field:

```
create-issue-gitbug-id = "customfield_5678"
```

### Status Map

You can specify the mapping between `git-bug` status and JIRA status id's using
the following:
```
bug-id-map = {"open": "1", "closed": "6"}
```

Note that in JIRA each different `issuetype` can have a different set of
statuses. The bridge doesn't currently support more than one mapping, however.

### Full example

Here is an example configuration with all optional fields set
```
[git-bug "bridge.default"]
	project = PROJ
	credentials-file = .git/jira-credentials.json
	target = jira
	server = https://jira.example.com
	create-issue-defaults = {"issuetype":"10001","customfield_1234":"default"}
	create-issue-gitbug-id = "customfield_5678"
	bug-open-id = 1
	bug-closed-id = 6
```

## To-Do list

* [0cf5c71] Assign git-bug to jira field on import
* [8acce9c] Download and cache workflow representation
* [95e3d45] Implement workflow gui
* [c70e22a] Implement additional query filters for import
* [9ecefaa] Create JIRA mock and add REST unit tests
* [67bf520] Create import/export integration tests
* [1121826] Add unit tests for utilites
* [0597088] Use OS keyring for credentials
* [d3e8f79] Don't count on the `Total` value in paginations


## Using CURL to poke at your JIRA's REST API

If you need to lookup the `id` for any `status`es or the `schema` for any
creation metadata, you can use CURL to query the API from the command line.
Here are a couple of examples to get you started.

### Getting a session token

```
curl \
  --data '{"username":"<username>", "password":"<password>"}' \
  --header "Content-Type: application/json" \
  --request POST \
  <serverUrl>/rest/auth/1/session
```

**Note**: If you have a json pretty printer installed (`sudo apt install jq`),
pipe the output through through that to make things more readable:

```
curl --silent \
  --data '{"username":"<username>", "password":"<password>"}' \
  --header "Content-Type: application/json" \
  --request POST
  <serverUrl>/rest/auth/1/session | jq .
```

example output:
```
{
  "session": {
    "name": "JSESSIONID",
    "value": "{sessionToken}"
  },
  "loginInfo": {
    "loginCount": 268,
    "previousLoginTime": "2019-11-12T08:03:35.300-0800"
  }
}
```

Make note of the output value. On subsequent invocations of `curl`, append the
following command-line option:

```
--cookie "JSESSIONID={sessionToken}"
```

Where `{sessionToken}` is the output from the `POST` above.

### Get a list of issuetype ids

```
curl --silent \
  --cookie "JSESSIONID={sessionToken}" \
  --header "Content-Type: application/json" \
  --request GET https://jira.example.com/rest/api/2/issuetype \
   | jq .
```

**example output**:
```
  {
    "self": "https://jira.example.com/rest/api/2/issuetype/13105",
    "id": "13105",
    "description": "",
    "iconUrl": "https://jira.example.com/secure/viewavatar?size=xsmall&avatarId=10316&avatarType=issuetype",
    "name": "Test Plan Links",
    "subtask": true,
    "avatarId": 10316
  },
  {
    "self": "https://jira.example.com/rest/api/2/issuetype/13106",
    "id": "13106",
    "description": "",
    "iconUrl": "https://jira.example.com/secure/viewavatar?size=xsmall&avatarId=10316&avatarType=issuetype",
    "name": "Enable Initiatives on the project",
    "subtask": true,
    "avatarId": 10316
  },
  ...
```


### Get a list of statuses


```
curl --silent \
  --cookie "JSESSIONID={sessionToken}" \
  --header "Content-Type: application/json" \
  --request GET https://jira.example.com/rest/api/2/project/{projectIdOrKey}/statuses \
   | jq .
```

**example output:**
```
[
  {
    "self": "https://example.com/rest/api/2/issuetype/3",
    "id": "3",
    "name": "Task",
    "subtask": false,
    "statuses": [
      {
        "self": "https://example.com/rest/api/2/status/1",
        "description": "The issue is open and ready for the assignee to start work on it.",
        "iconUrl": "https://example.com/images/icons/statuses/open.png",
        "name": "Open",
        "id": "1",
        "statusCategory": {
          "self": "https://example.com/rest/api/2/statuscategory/2",
          "id": 2,
          "key": "new",
          "colorName": "blue-gray",
          "name": "To Do"
        }
      },
...
```
