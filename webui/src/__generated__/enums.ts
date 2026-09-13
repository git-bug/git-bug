export const EntityEventType = {
  Created: 'CREATED',
  Removed: 'REMOVED',
  Updated: 'UPDATED'
} as const;

export type EntityEventType = typeof EntityEventType[keyof typeof EntityEventType];
/** How a file was affected by a commit. */
export const GitChangeStatus = {
  /** File was created in this commit. */
  Added: 'ADDED',
  /** File was removed in this commit. */
  Deleted: 'DELETED',
  /** File content changed in this commit. */
  Modified: 'MODIFIED',
  /** File was moved or renamed in this commit. */
  Renamed: 'RENAMED'
} as const;

export type GitChangeStatus = typeof GitChangeStatus[keyof typeof GitChangeStatus];
/** The role of a line within a unified diff hunk. */
export const GitDiffLineType = {
  /** A line added in the new version. */
  Added: 'ADDED',
  /** An unchanged line present in both old and new versions. */
  Context: 'CONTEXT',
  /** A line removed from the old version. */
  Deleted: 'DELETED'
} as const;

export type GitDiffLineType = typeof GitDiffLineType[keyof typeof GitDiffLineType];
/** The type of object a git tree entry points to. */
export const GitObjectType = {
  /** A regular or executable file. */
  Blob: 'BLOB',
  /** A git submodule. */
  Submodule: 'SUBMODULE',
  /** A symbolic link. */
  Symlink: 'SYMLINK',
  /** A directory. */
  Tree: 'TREE'
} as const;

export type GitObjectType = typeof GitObjectType[keyof typeof GitObjectType];
/** The kind of git reference: a branch, a tag, or a detached commit. */
export const GitRefType = {
  /** A local branch (refs/heads/*). */
  Branch: 'BRANCH',
  /** A detached HEAD pointing directly at a commit. */
  Commit: 'COMMIT',
  /** An annotated or lightweight tag (refs/tags/*). */
  Tag: 'TAG'
} as const;

export type GitRefType = typeof GitRefType[keyof typeof GitRefType];
export const LabelChangeStatus = {
  Added: 'ADDED',
  AlreadySet: 'ALREADY_SET',
  DoesntExist: 'DOESNT_EXIST',
  DuplicateInOp: 'DUPLICATE_IN_OP',
  Removed: 'REMOVED'
} as const;

export type LabelChangeStatus = typeof LabelChangeStatus[keyof typeof LabelChangeStatus];
export const Status = {
  Closed: 'CLOSED',
  Open: 'OPEN'
} as const;

export type Status = typeof Status[keyof typeof Status];