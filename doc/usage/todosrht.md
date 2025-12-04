# SourceHut Bridge

This document provides specific details for configuring and using SourceHut bridge.

## Configuration

To configure a SourceHut bridge, use `git bug bridge new` command and select `todosrht` as bridge type.

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

## Features

The SourceHut bridge supports the following functionality:

### Import Features
- **Ticket Import**: Import all tickets from a tracker with full history
- **Event Import**: Import comments, status changes, and label updates
- **Label Import**: Import existing labels and their colors
- **Assignment Events**: Import user assignment/unassignment events (with warnings)
- **User Mapping**: Automatically map SourceHut users to git-bug identities

### Export Features
- **Ticket Creation**: Create new tickets on SourceHut from git-bug
- **Comment Export**: Export comments as SourceHut comment events
- **Status Updates**: Export status changes (open/close) to SourceHut
- **Label Management**: 
  - Add/remove existing labels
  - **Auto-create missing labels** with default colors
- **Title Updates**: Export title changes to SourceHut
- **Body Updates**: Export ticket body/description changes

### Advanced Features
- **Public Tracker Access**: Access any public tracker, not just your own
- **Label Creation**: Automatically create labels that don't exist on the tracker
- **Error Handling**: Robust error handling with detailed messages
- **Validation**: Input validation for labels and other operations

## Usage

Once configured, you can use `git bug bridge pull <bridge_name>` to import tickets and events, and `git bug bridge push <bridge_name>` to export your local changes.

### Examples

```bash
# Configure a new bridge
git bug bridge new
# Select "todosrht" and follow the prompts

# Import all tickets from a tracker
git bug bridge pull my-todosrht-bridge

# Export local changes to SourceHut
git bug bridge push my-todosrht-bridge

# Pull changes since a specific date
git bug bridge pull my-todosrht-bridge --since "2023-01-01"
```

## Troubleshooting

### "Tracker doesn't exist" Error
If you get this error during bridge configuration:
1. Verify the tracker name is correct
2. Ensure the tracker is public or you have access
3. Check that the URL format is: `https://todo.sr.ht/~owner/tracker-name`

### Label Creation Issues
- Labels are created with default colors (white text, black background)
- Label names are validated (max 50 characters, no empty names)
- Check tracker permissions if label creation fails

### Assignment Events
- Assignment events are imported but generate warnings
- git-bug doesn't have native assignment support
- Events are preserved in import history for future compatibility

### Authentication Issues
- Ensure your token has all required scopes
- Verify token hasn't expired
- Check that your login matches the token owner