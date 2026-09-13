/* eslint-disable */
/** Internal type. DO NOT USE DIRECTLY. */
type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
/** Internal type. DO NOT USE DIRECTLY. */
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
import type { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type BugAddCommentAndCloseInput = {
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The collection of file's hash required for the first message. */
  files?: Array<string> | null | undefined;
  /** The message to be added to the bug. */
  message: string;
  /** The bug ID's prefix. */
  prefix: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
};

export type BugAddCommentAndReopenInput = {
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The collection of file's hash required for the first message. */
  files?: Array<string> | null | undefined;
  /** The message to be added to the bug. */
  message: string;
  /** The bug ID's prefix. */
  prefix: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
};

export type BugAddCommentInput = {
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The collection of file's hash required for the first message. */
  files?: Array<string> | null | undefined;
  /** The message to be added to the bug. */
  message: string;
  /** The bug ID's prefix. */
  prefix: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
};

export type BugChangeLabelInput = {
  /** The list of label to remove. */
  Removed?: Array<string> | null | undefined;
  /** The list of label to add. */
  added?: Array<string> | null | undefined;
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The bug ID's prefix. */
  prefix: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
};

export type BugCreateInput = {
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The collection of file's hash required for the first message. */
  files?: Array<string> | null | undefined;
  /** The first message of the new bug. */
  message: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
  /** The title of the new bug. */
  title: string;
};

export type BugEditCommentInput = {
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The collection of file's hash required for the first message. */
  files?: Array<string> | null | undefined;
  /** The new message to be set. */
  message: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
  /** A prefix of the CombinedId of the comment to be changed. */
  targetPrefix: string;
};

export type BugSetTitleInput = {
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The bug ID's prefix. */
  prefix: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
  /** The new title. */
  title: string;
};

export type BugStatusCloseInput = {
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The bug ID's prefix. */
  prefix: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
};

export type BugStatusOpenInput = {
  /** A unique identifier for the client performing the mutation. */
  clientMutationId?: string | null | undefined;
  /** The bug ID's prefix. */
  prefix: string;
  /** The name of the repository. If not set, the default repository is used. */
  repoRef?: string | null | undefined;
};

/** How a file was affected by a commit. */
export type GitChangeStatus =
  /** File was created in this commit. */
  | 'ADDED'
  /** File was removed in this commit. */
  | 'DELETED'
  /** File content changed in this commit. */
  | 'MODIFIED'
  /** File was moved or renamed in this commit. */
  | 'RENAMED';

/** The role of a line within a unified diff hunk. */
export type GitDiffLineType =
  /** A line added in the new version. */
  | 'ADDED'
  /** An unchanged line present in both old and new versions. */
  | 'CONTEXT'
  /** A line removed from the old version. */
  | 'DELETED';

/** The type of object a git tree entry points to. */
export type GitObjectType =
  /** A regular or executable file. */
  | 'BLOB'
  /** A git submodule. */
  | 'SUBMODULE'
  /** A symbolic link. */
  | 'SYMLINK'
  /** A directory. */
  | 'TREE';

/** The kind of git reference: a branch, a tag, or a detached commit. */
export type GitRefType =
  /** A local branch (refs/heads/*). */
  | 'BRANCH'
  /** A detached HEAD pointing directly at a commit. */
  | 'COMMIT'
  /** An annotated or lightweight tag (refs/tags/*). */
  | 'TAG';

export type Status =
  | 'CLOSED'
  | 'OPEN';

export type BugAddCommentMutationVariables = Exact<{
  input: BugAddCommentInput;
}>;


export type BugAddCommentMutation = { bugAddComment: { __typename: 'BugAddCommentPayload', bug: { __typename: 'Bug', id: string } } };

export type BugAddCommentAndCloseMutationVariables = Exact<{
  input: BugAddCommentAndCloseInput;
}>;


export type BugAddCommentAndCloseMutation = { bugAddCommentAndClose: { __typename: 'BugAddCommentAndClosePayload', bug: { __typename: 'Bug', id: string } } };

export type BugAddCommentAndReopenMutationVariables = Exact<{
  input: BugAddCommentAndReopenInput;
}>;


export type BugAddCommentAndReopenMutation = { bugAddCommentAndReopen: { __typename: 'BugAddCommentAndReopenPayload', bug: { __typename: 'Bug', id: string } } };

export type BugStatusOpenMutationVariables = Exact<{
  input: BugStatusOpenInput;
}>;


export type BugStatusOpenMutation = { bugStatusOpen: { __typename: 'BugStatusOpenPayload', bug: { __typename: 'Bug', id: string, status: Status } } };

export type BugStatusCloseMutationVariables = Exact<{
  input: BugStatusCloseInput;
}>;


export type BugStatusCloseMutation = { bugStatusClose: { __typename: 'BugStatusClosePayload', bug: { __typename: 'Bug', id: string, status: Status } } };

export type BugChangeLabelsMutationVariables = Exact<{
  input?: BugChangeLabelInput | null | undefined;
}>;


export type BugChangeLabelsMutation = { bugChangeLabels: { __typename: 'BugChangeLabelPayload', bug: { __typename: 'Bug', id: string, labels: Array<(
        { __typename: 'Label', name: string }
        & { ' $fragmentRefs'?: { 'LabelFieldsFragment': LabelFieldsFragment } }
      )> } } };

export type BugCreateCommentFieldsFragment = { __typename: 'BugCreateTimelineItem', id: string, message: string, createdAt: string, lastEdit: string, edited: boolean, author: (
    { __typename: 'Identity', id: string, humanId: string, displayName: string }
    & { ' $fragmentRefs'?: { 'IdentitySummaryFragment': IdentitySummaryFragment } }
  ) } & { ' $fragmentName'?: 'BugCreateCommentFieldsFragment' };

export type BugAddCommentFieldsFragment = { __typename: 'BugAddCommentTimelineItem', id: string, message: string, createdAt: string, lastEdit: string, edited: boolean, author: (
    { __typename: 'Identity', id: string, humanId: string, displayName: string }
    & { ' $fragmentRefs'?: { 'IdentitySummaryFragment': IdentitySummaryFragment } }
  ) } & { ' $fragmentName'?: 'BugAddCommentFieldsFragment' };

export type LabelChangeFieldsFragment = { __typename: 'BugLabelChangeTimelineItem', date: string, author: { __typename: 'Identity', humanId: string, displayName: string }, added: Array<(
    { __typename: 'Label' }
    & { ' $fragmentRefs'?: { 'LabelFieldsFragment': LabelFieldsFragment } }
  )>, removed: Array<(
    { __typename: 'Label' }
    & { ' $fragmentRefs'?: { 'LabelFieldsFragment': LabelFieldsFragment } }
  )> } & { ' $fragmentName'?: 'LabelChangeFieldsFragment' };

export type StatusChangeFieldsFragment = { __typename: 'BugSetStatusTimelineItem', date: string, status: Status, author: { __typename: 'Identity', humanId: string, displayName: string } } & { ' $fragmentName'?: 'StatusChangeFieldsFragment' };

export type TitleChangeFieldsFragment = { __typename: 'BugSetTitleTimelineItem', date: string, title: string, was: string, author: { __typename: 'Identity', humanId: string, displayName: string } } & { ' $fragmentName'?: 'TitleChangeFieldsFragment' };

export type TimelineItemsFragment = { __typename: 'BugTimelineItemConnection', nodes: Array<
    | (
      { __typename: 'BugAddCommentTimelineItem', id: string }
      & { ' $fragmentRefs'?: { 'BugAddCommentFieldsFragment': BugAddCommentFieldsFragment } }
    )
    | (
      { __typename: 'BugCreateTimelineItem', id: string }
      & { ' $fragmentRefs'?: { 'BugCreateCommentFieldsFragment': BugCreateCommentFieldsFragment } }
    )
    | (
      { __typename: 'BugLabelChangeTimelineItem', id: string }
      & { ' $fragmentRefs'?: { 'LabelChangeFieldsFragment': LabelChangeFieldsFragment } }
    )
    | (
      { __typename: 'BugSetStatusTimelineItem', id: string }
      & { ' $fragmentRefs'?: { 'StatusChangeFieldsFragment': StatusChangeFieldsFragment } }
    )
    | (
      { __typename: 'BugSetTitleTimelineItem', id: string }
      & { ' $fragmentRefs'?: { 'TitleChangeFieldsFragment': TitleChangeFieldsFragment } }
    )
  > } & { ' $fragmentName'?: 'TimelineItemsFragment' };

export type BugEditCommentMutationVariables = Exact<{
  input: BugEditCommentInput;
}>;


export type BugEditCommentMutation = { bugEditComment: { __typename: 'BugEditCommentPayload', bug: { __typename: 'Bug', id: string } } };

export type BugSetTitleMutationVariables = Exact<{
  input: BugSetTitleInput;
}>;


export type BugSetTitleMutation = { bugSetTitle: { __typename: 'BugSetTitlePayload', bug: { __typename: 'Bug', id: string, title: string } } };

export type CommitListQueryVariables = Exact<{
  repo?: string | null | undefined;
  ref: string;
  path?: string | null | undefined;
  after?: string | null | undefined;
  first?: number | null | undefined;
}>;


export type CommitListQuery = { repository: { __typename: 'Repository', name: string | null, commits: { __typename: 'GitCommitConnection', nodes: Array<{ __typename: 'GitCommit', hash: string, shortHash: string, message: string, authorName: string, date: string }>, pageInfo: { __typename: 'PageInfo', hasNextPage: boolean, endCursor: string } } } | null };

export type FileDiffQueryVariables = Exact<{
  repo?: string | null | undefined;
  hash: string;
  path: string;
}>;


export type FileDiffQuery = { repository: { __typename: 'Repository', name: string | null, commit: { __typename: 'GitCommit', diff: { __typename: 'GitFileDiff', path: string, oldPath: string | null, isBinary: boolean, isNew: boolean, isDelete: boolean, hunks: Array<{ __typename: 'GitDiffHunk', oldStart: number, oldLines: number, newStart: number, newLines: number, lines: Array<{ __typename: 'GitDiffLine', type: GitDiffLineType, content: string, oldLine: number, newLine: number }> }> } | null } | null } | null };

export type FileViewerBlobFragment = { __typename: 'GitBlob', path: string, hash: string, text: string | null, size: number, isBinary: boolean, isTruncated: boolean } & { ' $fragmentName'?: 'FileViewerBlobFragment' };

export type RefSelectorRefsFragment = { __typename: 'GitRefConnection', nodes: Array<{ __typename: 'GitRef', name: string, shortName: string, type: GitRefType }> } & { ' $fragmentName'?: 'RefSelectorRefsFragment' };

export type IdentitySummaryFragment = { __typename: 'Identity', id: string, humanId: string, displayName: string, avatarUrl: string | null } & { ' $fragmentName'?: 'IdentitySummaryFragment' };

export type BugSummaryFragment = { __typename: 'Bug', id: string, humanId: string, status: Status, title: string, createdAt: string, labels: Array<(
    { __typename: 'Label', name: string }
    & { ' $fragmentRefs'?: { 'LabelFieldsFragment': LabelFieldsFragment } }
  )>, author: (
    { __typename: 'Identity' }
    & { ' $fragmentRefs'?: { 'IdentitySummaryFragment': IdentitySummaryFragment } }
  ), comments: { __typename: 'BugCommentConnection', totalCount: number } } & { ' $fragmentName'?: 'BugSummaryFragment' };

export type LabelFieldsFragment = { __typename: 'Label', name: string, color: { __typename: 'Color', R: number, G: number, B: number } } & { ' $fragmentName'?: 'LabelFieldsFragment' };

export type RepositoriesQueryVariables = Exact<{ [key: string]: never; }>;


export type RepositoriesQuery = { repositories: { __typename: 'RepositoryConnection', totalCount: number, nodes: Array<{ __typename: 'Repository', name: string | null }> } };

export type RepositoryQueryVariables = Exact<{
  ref?: string | null | undefined;
}>;


export type RepositoryQuery = { repository: { __typename: 'Repository', name: string | null } | null };

export type CommitsQueryVariables = Exact<{
  repo?: string | null | undefined;
  gitRef: string;
  after?: string | null | undefined;
}>;


export type CommitsQuery = { repository: { __typename: 'Repository', name: string | null, commits: { __typename: 'GitCommitConnection', nodes: Array<{ __typename: 'GitCommit', hash: string }> } } | null };

export type UserIdentityQueryVariables = Exact<{ [key: string]: never; }>;


export type UserIdentityQuery = { repository: { __typename: 'Repository', name: string | null, userIdentity: (
      { __typename: 'Identity', id: string, humanId: string, displayName: string, avatarUrl: string | null, name: string | null, email: string | null, login: string | null }
      & { ' $fragmentRefs'?: { 'IdentitySummaryFragment': IdentitySummaryFragment } }
    ) | null } | null };

export type CodePageRefsQueryVariables = Exact<{
  repo?: string | null | undefined;
}>;


export type CodePageRefsQuery = { repository: { __typename: 'Repository', name: string | null, head: { __typename: 'GitRef', shortName: string } | null, refs: (
      { __typename: 'GitRefConnection' }
      & { ' $fragmentRefs'?: { 'RefSelectorRefsFragment': RefSelectorRefsFragment } }
    ) } | null };

export type CodePageBlobQueryVariables = Exact<{
  repo?: string | null | undefined;
  ref: string;
  path: string;
}>;


export type CodePageBlobQuery = { repository: { __typename: 'Repository', name: string | null, blob: (
      { __typename: 'GitBlob' }
      & { ' $fragmentRefs'?: { 'FileViewerBlobFragment': FileViewerBlobFragment } }
    ) | null } | null };

export type CodePageTreeQueryVariables = Exact<{
  repo?: string | null | undefined;
  ref: string;
  path?: string | null | undefined;
}>;


export type CodePageTreeQuery = { repository: { __typename: 'Repository', name: string | null, tree: Array<{ __typename: 'GitTreeEntry', name: string, type: GitObjectType, hash: string }> } | null };

export type CodePageLastCommitsQueryVariables = Exact<{
  repo?: string | null | undefined;
  ref: string;
  path?: string | null | undefined;
  names: Array<string> | string;
}>;


export type CodePageLastCommitsQuery = { repository: { __typename: 'Repository', name: string | null, lastCommits: Array<{ __typename: 'GitLastCommit', name: string, commit: { __typename: 'GitCommit', hash: string, shortHash: string, message: string, date: string } }> } | null };

export type CodePageReadmeQueryVariables = Exact<{
  repo?: string | null | undefined;
  ref: string;
  path: string;
}>;


export type CodePageReadmeQuery = { repository: { __typename: 'Repository', name: string | null, blob: { __typename: 'GitBlob', text: string | null } | null } | null };

export type AllIdentitiesQueryVariables = Exact<{
  ref?: string | null | undefined;
}>;


export type AllIdentitiesQuery = { repository: { __typename: 'Repository', name: string | null, allIdentities: { __typename: 'IdentityConnection', nodes: Array<{ __typename: 'Identity', id: string, humanId: string, name: string | null, email: string | null, login: string | null, displayName: string, avatarUrl: string | null }> } } | null };

export type ValidLabelsQueryVariables = Exact<{
  ref?: string | null | undefined;
}>;


export type ValidLabelsQuery = { repository: { __typename: 'Repository', name: string | null, validLabels: { __typename: 'LabelConnection', nodes: Array<(
        { __typename: 'Label', name: string, color: { __typename: 'Color', R: number, G: number, B: number } }
        & { ' $fragmentRefs'?: { 'LabelFieldsFragment': LabelFieldsFragment } }
      )> } } | null };

export type BugDetailQueryVariables = Exact<{
  ref?: string | null | undefined;
  prefix: string;
}>;


export type BugDetailQuery = { repository: { __typename: 'Repository', name: string | null, bug: (
      { __typename: 'Bug', humanId: string, title: string, status: Status, createdAt: string, lastEdit: string, labels: Array<(
        { __typename: 'Label', name: string }
        & { ' $fragmentRefs'?: { 'LabelFieldsFragment': LabelFieldsFragment } }
      )>, author: { __typename: 'Identity', humanId: string, displayName: string }, participants: { __typename: 'IdentityConnection', nodes: Array<(
          { __typename: 'Identity', id: string, humanId: string, displayName: string, avatarUrl: string | null }
          & { ' $fragmentRefs'?: { 'IdentitySummaryFragment': IdentitySummaryFragment } }
        )> }, timeline: (
        { __typename: 'BugTimelineItemConnection' }
        & { ' $fragmentRefs'?: { 'TimelineItemsFragment': TimelineItemsFragment } }
      ) }
      & { ' $fragmentRefs'?: { 'BugSummaryFragment': BugSummaryFragment } }
    ) | null } | null };

export type BugListQueryVariables = Exact<{
  ref?: string | null | undefined;
  openQuery: string;
  closedQuery: string;
  listQuery: string;
  first?: number | null | undefined;
  after?: string | null | undefined;
}>;


export type BugListQuery = { repository: { __typename: 'Repository', name: string | null, openCount: { __typename: 'BugConnection', totalCount: number }, closedCount: { __typename: 'BugConnection', totalCount: number }, bugs: { __typename: 'BugConnection', totalCount: number, pageInfo: { __typename: 'PageInfo', hasNextPage: boolean, endCursor: string }, nodes: Array<(
        { __typename: 'Bug', id: string, humanId: string, status: Status, title: string, createdAt: string, labels: Array<(
          { __typename: 'Label', name: string }
          & { ' $fragmentRefs'?: { 'LabelFieldsFragment': LabelFieldsFragment } }
        )>, author: (
          { __typename: 'Identity', humanId: string, displayName: string }
          & { ' $fragmentRefs'?: { 'IdentitySummaryFragment': IdentitySummaryFragment } }
        ), comments: { __typename: 'BugCommentConnection', totalCount: number } }
        & { ' $fragmentRefs'?: { 'BugSummaryFragment': BugSummaryFragment } }
      )> } } | null };

export type BugCreateMutationVariables = Exact<{
  input: BugCreateInput;
}>;


export type BugCreateMutation = { bugCreate: { __typename: 'BugCreatePayload', bug: { __typename: 'Bug', id: string, humanId: string } } };

export type UserProfileQueryVariables = Exact<{
  ref?: string | null | undefined;
  prefix: string;
  openQuery: string;
  closedQuery: string;
  listQuery: string;
  after?: string | null | undefined;
}>;


export type UserProfileQuery = { repository: { __typename: 'Repository', name: string | null, identity: { __typename: 'Identity', id: string, humanId: string, name: string | null, email: string | null, login: string | null, displayName: string, avatarUrl: string | null, isProtected: boolean } | null, openCount: { __typename: 'BugConnection', totalCount: number }, closedCount: { __typename: 'BugConnection', totalCount: number }, bugs: { __typename: 'BugConnection', totalCount: number, pageInfo: { __typename: 'PageInfo', hasNextPage: boolean, endCursor: string }, nodes: Array<(
        { __typename: 'Bug', id: string, humanId: string, status: Status, title: string, createdAt: string, labels: Array<(
          { __typename: 'Label', name: string }
          & { ' $fragmentRefs'?: { 'LabelFieldsFragment': LabelFieldsFragment } }
        )>, author: (
          { __typename: 'Identity', humanId: string }
          & { ' $fragmentRefs'?: { 'IdentitySummaryFragment': IdentitySummaryFragment } }
        ), comments: { __typename: 'BugCommentConnection', totalCount: number } }
        & { ' $fragmentRefs'?: { 'BugSummaryFragment': BugSummaryFragment } }
      )> } } | null };

export type CommitPageDetailQueryVariables = Exact<{
  repo?: string | null | undefined;
  hash: string;
}>;


export type CommitPageDetailQuery = { repository: { __typename: 'Repository', name: string | null, commit: { __typename: 'GitCommit', hash: string, shortHash: string, message: string, fullMessage: string, authorName: string, authorEmail: string, date: string, parents: Array<string>, files: { __typename: 'GitChangedFileConnection', nodes: Array<{ __typename: 'GitChangedFile', path: string, oldPath: string | null, status: GitChangeStatus }> } } | null } | null };

export const IdentitySummaryFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}}]} as unknown as DocumentNode<IdentitySummaryFragment, unknown>;
export const BugCreateCommentFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugCreateCommentFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugCreateTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastEdit"}},{"kind":"Field","name":{"kind":"Name","value":"edited"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}}]} as unknown as DocumentNode<BugCreateCommentFieldsFragment, unknown>;
export const BugAddCommentFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugAddCommentFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugAddCommentTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastEdit"}},{"kind":"Field","name":{"kind":"Name","value":"edited"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}}]} as unknown as DocumentNode<BugAddCommentFieldsFragment, unknown>;
export const LabelFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}}]} as unknown as DocumentNode<LabelFieldsFragment, unknown>;
export const LabelChangeFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugLabelChangeTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"added"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"removed"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}}]} as unknown as DocumentNode<LabelChangeFieldsFragment, unknown>;
export const StatusChangeFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"StatusChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetStatusTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]} as unknown as DocumentNode<StatusChangeFieldsFragment, unknown>;
export const TitleChangeFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TitleChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetTitleTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"was"}}]}}]} as unknown as DocumentNode<TitleChangeFieldsFragment, unknown>;
export const TimelineItemsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TimelineItems"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugTimelineItemConnection"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"__typename"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugCreateTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"BugCreateCommentFields"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugAddCommentTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"BugAddCommentFields"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugLabelChangeTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelChangeFields"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetStatusTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"StatusChangeFields"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetTitleTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TitleChangeFields"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugCreateCommentFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugCreateTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastEdit"}},{"kind":"Field","name":{"kind":"Name","value":"edited"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugAddCommentFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugAddCommentTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastEdit"}},{"kind":"Field","name":{"kind":"Name","value":"edited"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugLabelChangeTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"added"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"removed"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"StatusChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetStatusTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TitleChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetTitleTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"was"}}]}}]} as unknown as DocumentNode<TimelineItemsFragment, unknown>;
export const FileViewerBlobFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"FileViewerBlob"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"GitBlob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"text"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"isBinary"}},{"kind":"Field","name":{"kind":"Name","value":"isTruncated"}}]}}]} as unknown as DocumentNode<FileViewerBlobFragment, unknown>;
export const RefSelectorRefsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"RefSelectorRefs"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"GitRefConnection"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"shortName"}},{"kind":"Field","name":{"kind":"Name","value":"type"}}]}}]}}]} as unknown as DocumentNode<RefSelectorRefsFragment, unknown>;
export const BugSummaryFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Bug"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}}]} as unknown as DocumentNode<BugSummaryFragment, unknown>;
export const BugAddCommentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugAddComment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BugAddCommentInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugAddComment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<BugAddCommentMutation, BugAddCommentMutationVariables>;
export const BugAddCommentAndCloseDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugAddCommentAndClose"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BugAddCommentAndCloseInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugAddCommentAndClose"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<BugAddCommentAndCloseMutation, BugAddCommentAndCloseMutationVariables>;
export const BugAddCommentAndReopenDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugAddCommentAndReopen"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BugAddCommentAndReopenInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugAddCommentAndReopen"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<BugAddCommentAndReopenMutation, BugAddCommentAndReopenMutationVariables>;
export const BugStatusOpenDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugStatusOpen"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BugStatusOpenInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugStatusOpen"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]}}]} as unknown as DocumentNode<BugStatusOpenMutation, BugStatusOpenMutationVariables>;
export const BugStatusCloseDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugStatusClose"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BugStatusCloseInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugStatusClose"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]}}]} as unknown as DocumentNode<BugStatusCloseMutation, BugStatusCloseMutationVariables>;
export const BugChangeLabelsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugChangeLabels"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"BugChangeLabelInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugChangeLabels"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}}]} as unknown as DocumentNode<BugChangeLabelsMutation, BugChangeLabelsMutationVariables>;
export const BugEditCommentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugEditComment"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BugEditCommentInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugEditComment"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<BugEditCommentMutation, BugEditCommentMutationVariables>;
export const BugSetTitleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugSetTitle"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetTitleInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugSetTitle"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}}]}}]}}]}}]} as unknown as DocumentNode<BugSetTitleMutation, BugSetTitleMutationVariables>;
export const CommitListDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CommitList"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"path"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"after"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"first"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"commits"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"path"},"value":{"kind":"Variable","name":{"kind":"Name","value":"path"}}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"after"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"Variable","name":{"kind":"Name","value":"first"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"shortHash"}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"authorName"}},{"kind":"Field","name":{"kind":"Name","value":"date"}}]}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}}]}}]}}]}}]} as unknown as DocumentNode<CommitListQuery, CommitListQueryVariables>;
export const FileDiffDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"FileDiff"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"hash"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"path"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"commit"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"hash"},"value":{"kind":"Variable","name":{"kind":"Name","value":"hash"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"diff"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"path"},"value":{"kind":"Variable","name":{"kind":"Name","value":"path"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"oldPath"}},{"kind":"Field","name":{"kind":"Name","value":"isBinary"}},{"kind":"Field","name":{"kind":"Name","value":"isNew"}},{"kind":"Field","name":{"kind":"Name","value":"isDelete"}},{"kind":"Field","name":{"kind":"Name","value":"hunks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"oldStart"}},{"kind":"Field","name":{"kind":"Name","value":"oldLines"}},{"kind":"Field","name":{"kind":"Name","value":"newStart"}},{"kind":"Field","name":{"kind":"Name","value":"newLines"}},{"kind":"Field","name":{"kind":"Name","value":"lines"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"content"}},{"kind":"Field","name":{"kind":"Name","value":"oldLine"}},{"kind":"Field","name":{"kind":"Name","value":"newLine"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<FileDiffQuery, FileDiffQueryVariables>;
export const RepositoriesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Repositories"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repositories"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode<RepositoriesQuery, RepositoriesQueryVariables>;
export const RepositoryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Repository"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<RepositoryQuery, RepositoryQueryVariables>;
export const CommitsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Commits"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"gitRef"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"after"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"commits"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"gitRef"}}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"after"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hash"}}]}}]}}]}}]}}]} as unknown as DocumentNode<CommitsQuery, CommitsQueryVariables>;
export const UserIdentityDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UserIdentity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"userIdentity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"email"}},{"kind":"Field","name":{"kind":"Name","value":"login"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}}]} as unknown as DocumentNode<UserIdentityQuery, UserIdentityQueryVariables>;
export const CodePageRefsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CodePageRefs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"head"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"shortName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"refs"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"RefSelectorRefs"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"RefSelectorRefs"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"GitRefConnection"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"shortName"}},{"kind":"Field","name":{"kind":"Name","value":"type"}}]}}]}}]} as unknown as DocumentNode<CodePageRefsQuery, CodePageRefsQueryVariables>;
export const CodePageBlobDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CodePageBlob"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"path"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"blob"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"path"},"value":{"kind":"Variable","name":{"kind":"Name","value":"path"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"FileViewerBlob"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"FileViewerBlob"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"GitBlob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"text"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"isBinary"}},{"kind":"Field","name":{"kind":"Name","value":"isTruncated"}}]}}]} as unknown as DocumentNode<CodePageBlobQuery, CodePageBlobQueryVariables>;
export const CodePageTreeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CodePageTree"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"path"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"tree"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"path"},"value":{"kind":"Variable","name":{"kind":"Name","value":"path"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"hash"}}]}}]}}]}}]} as unknown as DocumentNode<CodePageTreeQuery, CodePageTreeQueryVariables>;
export const CodePageLastCommitsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CodePageLastCommits"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"path"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"names"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"lastCommits"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"path"},"value":{"kind":"Variable","name":{"kind":"Name","value":"path"}}},{"kind":"Argument","name":{"kind":"Name","value":"names"},"value":{"kind":"Variable","name":{"kind":"Name","value":"names"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"commit"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"shortHash"}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"date"}}]}}]}}]}}]}}]} as unknown as DocumentNode<CodePageLastCommitsQuery, CodePageLastCommitsQueryVariables>;
export const CodePageReadmeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CodePageReadme"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"path"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"blob"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}},{"kind":"Argument","name":{"kind":"Name","value":"path"},"value":{"kind":"Variable","name":{"kind":"Name","value":"path"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"text"}}]}}]}}]}}]} as unknown as DocumentNode<CodePageReadmeQuery, CodePageReadmeQueryVariables>;
export const AllIdentitiesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AllIdentities"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"allIdentities"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1000"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"email"}},{"kind":"Field","name":{"kind":"Name","value":"login"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}}]}}]}}]}}]} as unknown as DocumentNode<AllIdentitiesQuery, AllIdentitiesQueryVariables>;
export const ValidLabelsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ValidLabels"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"validLabels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}}]} as unknown as DocumentNode<ValidLabelsQuery, ValidLabelsQueryVariables>;
export const BugDetailDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BugDetail"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"prefix"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"bug"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"prefix"},"value":{"kind":"Variable","name":{"kind":"Name","value":"prefix"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"BugSummary"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastEdit"}},{"kind":"Field","name":{"kind":"Name","value":"participants"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"20"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"timeline"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"250"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TimelineItems"}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugCreateCommentFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugCreateTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastEdit"}},{"kind":"Field","name":{"kind":"Name","value":"edited"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugAddCommentFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugAddCommentTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastEdit"}},{"kind":"Field","name":{"kind":"Name","value":"edited"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugLabelChangeTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"added"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"removed"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"StatusChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetStatusTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TitleChangeFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetTitleTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"was"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Bug"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TimelineItems"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugTimelineItemConnection"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"__typename"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugCreateTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"BugCreateCommentFields"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugAddCommentTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"BugAddCommentFields"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugLabelChangeTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelChangeFields"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetStatusTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"StatusChangeFields"}}]}},{"kind":"InlineFragment","typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"BugSetTitleTimelineItem"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TitleChangeFields"}}]}}]}}]}}]} as unknown as DocumentNode<BugDetailQuery, BugDetailQueryVariables>;
export const BugListDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"BugList"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"openQuery"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"closedQuery"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"listQuery"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"first"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"after"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","alias":{"kind":"Name","value":"openCount"},"name":{"kind":"Name","value":"allBugs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"openQuery"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"closedCount"},"name":{"kind":"Name","value":"allBugs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"closedQuery"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"bugs"},"name":{"kind":"Name","value":"allBugs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"listQuery"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"Variable","name":{"kind":"Name","value":"first"}}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"after"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}},{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"BugSummary"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Bug"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode<BugListQuery, BugListQueryVariables>;
export const BugCreateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BugCreate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BugCreateInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bugCreate"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bug"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}}]}}]}}]}}]} as unknown as DocumentNode<BugCreateMutation, BugCreateMutationVariables>;
export const UserProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UserProfile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ref"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"prefix"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"openQuery"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"closedQuery"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"listQuery"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"after"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ref"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"identity"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"prefix"},"value":{"kind":"Variable","name":{"kind":"Name","value":"prefix"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"email"}},{"kind":"Field","name":{"kind":"Name","value":"login"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}},{"kind":"Field","name":{"kind":"Name","value":"isProtected"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"openCount"},"name":{"kind":"Name","value":"allBugs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"openQuery"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"closedCount"},"name":{"kind":"Name","value":"allBugs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"closedQuery"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"1"}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"bugs"},"name":{"kind":"Name","value":"allBugs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"query"},"value":{"kind":"Variable","name":{"kind":"Name","value":"listQuery"}}},{"kind":"Argument","name":{"kind":"Name","value":"first"},"value":{"kind":"IntValue","value":"25"}},{"kind":"Argument","name":{"kind":"Name","value":"after"},"value":{"kind":"Variable","name":{"kind":"Name","value":"after"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"pageInfo"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"endCursor"}}]}},{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"BugSummary"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"LabelFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Label"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"color"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"R"}},{"kind":"Field","name":{"kind":"Name","value":"G"}},{"kind":"Field","name":{"kind":"Name","value":"B"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IdentitySummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Identity"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"displayName"}},{"kind":"Field","name":{"kind":"Name","value":"avatarUrl"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"BugSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"Bug"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"humanId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"labels"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"LabelFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"author"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IdentitySummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"comments"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]} as unknown as DocumentNode<UserProfileQuery, UserProfileQueryVariables>;
export const CommitPageDetailDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"CommitPageDetail"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"repo"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"hash"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repository"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ref"},"value":{"kind":"Variable","name":{"kind":"Name","value":"repo"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"commit"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"hash"},"value":{"kind":"Variable","name":{"kind":"Name","value":"hash"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"shortHash"}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"fullMessage"}},{"kind":"Field","name":{"kind":"Name","value":"authorName"}},{"kind":"Field","name":{"kind":"Name","value":"authorEmail"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"parents"}},{"kind":"Field","name":{"kind":"Name","value":"files"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"nodes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"oldPath"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<CommitPageDetailQuery, CommitPageDetailQueryVariables>;