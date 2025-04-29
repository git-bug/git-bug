# SourceHut Bridge

This document provides specific details for configuring and using the SourceHut bridge.

## Configuration

To configure a SourceHut bridge, use the `git bug bridge new` command and select `todosrht` as the bridge type.

You will need to provide:
- **SourceHut server URL**: Typically `https://todo.sr.ht`
- **Tracker name**: The full name of your tracker (e.g., `~user/project-name`)
- **Login**: Your SourceHut username
- **Access token**: A personal access token generated from your SourceHut account.

### Generating an Access Token

1. Go to [meta.sr.ht/oauth2](https://meta.sr.ht/oauth2).
2. Click on "Personal access tokens".
3. Click "Generate new token".
4. Provide a description for the token (e.g., "git-bug integration").
5. Select the following scopes:
    - `TRACKERS` (read/write)
    - `TICKETS` (read/write)
    - `EVENTS` (read)
    - `PROFILE` (read)
6. Click "Submit" and copy the generated token.

## Usage

Once configured, you can use `git bug bridge pull <bridge_name>` to import tickets and events, and `git bug bridge push <bridge_name>` to export your local changes.