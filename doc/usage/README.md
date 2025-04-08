# Usage

## Web UI

You can launch a Web UI with `git bug webui`:

<details><summary>View a feed of bugs</summary>
<p align="center">
  <img src="misc/webui1.png" alt="Web UI screenshot 1" width="880">
</p>
</details>

<details><summary>View comments on bug</summary>
<p align="center">
  <img src="misc/webui2.png" alt="Web UI screenshot 2" width="880">
</p>
</details>

This web UI is packed inside the same binary and serves static content through
an http server running on the local host machine.

The web UI interacts with the backend through a GraphQL API. \[View the
schema\]\[api/gql/schema\] for more information.
