# todo.sr.ht Bridge Design and Architecture

## Overview

This document describes the design and architecture of the todo.sr.ht (SourceHut) bridge in `bridge/todosrht/`. The bridge synchronizes git-bug issues with tickets on a todo.sr.ht tracker using SourceHut's GraphQL API.

## Architecture

The bridge is structured into four main components:

1. **GraphQL Client (`client.go`)**: Wraps HTTP transport, authentication via personal access tokens, request construction, GraphQL query validation, cursor-based pagination, and structured error extraction.
2. **Configuration (`config.go`)**: Handles bridge setup (URL parsing, `--project` tracker naming, credential lookup, and storing base URL metadata).
3. **Importer (`import.go`)**: Fetches tickets and timeline events from the remote tracker, sorts events chronologically, reconciles remote title/body edits, deduplicates multi-change events, and creates corresponding git-bug operations.
4. **Exporter (`export.go`)**: Checks local bug operations against remote tickets, validates tracker and instance associations, creates tickets/comments/labels/status changes via GraphQL mutations, and reconciles remote label state idempotently across retries.

## Data Model and Mapping

### Status and Resolution

git-bug uses binary status (`OpenStatus` and `ClosedStatus`). SourceHut uses `TicketStatus` and `TicketResolution`.

| SourceHut Status | SourceHut Resolution | git-bug Status |
|---|---|---|
| REPORTED, CONFIRMED, IN_PROGRESS, PENDING | UNRESOLVED | Open |
| RESOLVED | FIXED (default on export) | Closed |

On export of a closed bug, the status is set to `RESOLVED` with `FIXED` resolution. When reopened, the status is set to `REPORTED` with `UNRESOLVED` resolution.

### Event Mapping

SourceHut tickets record changes as timeline events. Each event contains one or more change details:

| SourceHut Event / Change | git-bug Operation | Notes |
|---|---|---|
| `Created` | `CreateOperation` | Carries initial submitter and ticket ID |
| `Comment` | `AddCommentOperation` / `EditCommentOperation` | `supersededBy` links revisions to the original comment; text cleanup applied |
| `StatusChange` | `SetStatusOperation` | Open or Close operations |
| `LabelUpdate` | `LabelChangeOperation` | Labels added or removed |
| `Assignment` | Metadata marker | Warned once; user assignments not tracked in git-bug |
| `UserMention` / `TicketMention` | Metadata marker | Warned once; mentions are not stateful operations |

Operations created from events are tagged with `todosrht-id: <event.id>:<change.index>` to prevent multiple matching errors when repeat pulls occur on multi-change events.

### Identity and Multi-Instance Separation

Imported identities carry `todosrht-base-url` and `todosrht-login`. Configured identities use an instance-specific `todosrht-login:<base-url>` key, allowing an explicitly configured local identity to represent accounts on several instances. Unscoped logins alone are never used to match remote users.

## Authentication and Security

- Communication requires HTTPS, except for local loopback development.
- Authentication uses SourceHut personal access tokens with scopes:
  `todo.sr.ht/PROFILE:RO todo.sr.ht/EVENTS:RO todo.sr.ht/TRACKERS:RW todo.sr.ht/TICKETS:RW`
- Pre-existing credentials added via `git bug bridge auth add-token` are associated with the bridge base URL on first configuration.
- Ticket IDs found on existing bugs are only accepted if verified against matching tracker and base URL metadata.
