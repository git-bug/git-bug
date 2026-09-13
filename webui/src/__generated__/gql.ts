/* eslint-disable */
import * as types from './graphql';
import type { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';

/**
 * Map of all GraphQL operations in the project.
 *
 * This map has several performance disadvantages:
 * 1. It is not tree-shakeable, so it will include all operations in the project.
 * 2. It is not minifiable, so the string of a GraphQL query will be multiple times inside the bundle.
 * 3. It does not support dead code elimination, so it will add unused operations.
 *
 * Therefore it is highly recommended to use the babel or swc plugin for production.
 * Learn more about it here: https://the-guild.dev/graphql/codegen/plugins/presets/preset-client#reducing-bundle-size
 */
type Documents = {
    "\n  mutation BugAddComment($input: BugAddCommentInput!) {\n    bugAddComment(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n": typeof types.BugAddCommentDocument,
    "\n  mutation BugAddCommentAndClose($input: BugAddCommentAndCloseInput!) {\n    bugAddCommentAndClose(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n": typeof types.BugAddCommentAndCloseDocument,
    "\n  mutation BugAddCommentAndReopen($input: BugAddCommentAndReopenInput!) {\n    bugAddCommentAndReopen(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n": typeof types.BugAddCommentAndReopenDocument,
    "\n  mutation BugStatusOpen($input: BugStatusOpenInput!) {\n    bugStatusOpen(input: $input) {\n      bug {\n        id\n        status\n      }\n    }\n  }\n": typeof types.BugStatusOpenDocument,
    "\n  mutation BugStatusClose($input: BugStatusCloseInput!) {\n    bugStatusClose(input: $input) {\n      bug {\n        id\n        status\n      }\n    }\n  }\n": typeof types.BugStatusCloseDocument,
    "\n  mutation BugChangeLabels($input: BugChangeLabelInput) {\n    bugChangeLabels(input: $input) {\n      bug {\n        id\n        labels {\n          name\n          ...LabelFields\n        }\n      }\n    }\n  }\n": typeof types.BugChangeLabelsDocument,
    "\n  fragment BugCreateCommentFields on BugCreateTimelineItem {\n    id\n    author {\n      id\n      humanId\n      displayName\n      ...IdentitySummary\n    }\n    message\n    createdAt\n    lastEdit\n    edited\n  }\n": typeof types.BugCreateCommentFieldsFragmentDoc,
    "\n  fragment BugAddCommentFields on BugAddCommentTimelineItem {\n    id\n    author {\n      id\n      humanId\n      displayName\n      ...IdentitySummary\n    }\n    message\n    createdAt\n    lastEdit\n    edited\n  }\n": typeof types.BugAddCommentFieldsFragmentDoc,
    "\n  fragment LabelChangeFields on BugLabelChangeTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    added {\n      ...LabelFields\n    }\n    removed {\n      ...LabelFields\n    }\n  }\n": typeof types.LabelChangeFieldsFragmentDoc,
    "\n  fragment StatusChangeFields on BugSetStatusTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    status\n  }\n": typeof types.StatusChangeFieldsFragmentDoc,
    "\n  fragment TitleChangeFields on BugSetTitleTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    title\n    was\n  }\n": typeof types.TitleChangeFieldsFragmentDoc,
    "\n  fragment TimelineItems on BugTimelineItemConnection {\n    nodes {\n      __typename\n      id\n      ... on BugCreateTimelineItem {\n        ...BugCreateCommentFields\n      }\n      ... on BugAddCommentTimelineItem {\n        ...BugAddCommentFields\n      }\n      ... on BugLabelChangeTimelineItem {\n        ...LabelChangeFields\n      }\n      ... on BugSetStatusTimelineItem {\n        ...StatusChangeFields\n      }\n      ... on BugSetTitleTimelineItem {\n        ...TitleChangeFields\n      }\n    }\n  }\n": typeof types.TimelineItemsFragmentDoc,
    "\n  mutation BugEditComment($input: BugEditCommentInput!) {\n    bugEditComment(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n": typeof types.BugEditCommentDocument,
    "\n  mutation BugSetTitle($input: BugSetTitleInput!) {\n    bugSetTitle(input: $input) {\n      bug {\n        id\n        title\n      }\n    }\n  }\n": typeof types.BugSetTitleDocument,
    "\n  query CommitList($repo: String, $ref: String!, $path: String, $after: String, $first: Int) {\n    repository(ref: $repo) {\n      name\n      commits(ref: $ref, path: $path, after: $after, first: $first) {\n        nodes {\n          hash\n          shortHash\n          message\n          authorName\n          date\n        }\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n      }\n    }\n  }\n": typeof types.CommitListDocument,
    "\n  query FileDiff($repo: String, $hash: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      commit(hash: $hash) {\n        diff(path: $path) {\n          path\n          oldPath\n          isBinary\n          isNew\n          isDelete\n          hunks {\n            oldStart\n            oldLines\n            newStart\n            newLines\n            lines {\n              type\n              content\n              oldLine\n              newLine\n            }\n          }\n        }\n      }\n    }\n  }\n": typeof types.FileDiffDocument,
    "\n  fragment FileViewerBlob on GitBlob {\n    path\n    hash\n    text\n    size\n    isBinary\n    isTruncated\n  }\n": typeof types.FileViewerBlobFragmentDoc,
    "\n  fragment RefSelectorRefs on GitRefConnection {\n    nodes {\n      name\n      shortName\n      type\n    }\n  }\n": typeof types.RefSelectorRefsFragmentDoc,
    "\n  fragment IdentitySummary on Identity {\n    id\n    humanId\n    displayName\n    avatarUrl\n  }\n": typeof types.IdentitySummaryFragmentDoc,
    "\n  fragment BugSummary on Bug {\n    id\n    humanId\n    status\n    title\n    labels {\n      name\n      ...LabelFields\n    }\n    author {\n      ...IdentitySummary\n    }\n    createdAt\n    comments {\n      totalCount\n    }\n  }\n": typeof types.BugSummaryFragmentDoc,
    "\n  fragment LabelFields on Label {\n    name\n    color {\n      R\n      G\n      B\n    }\n  }\n": typeof types.LabelFieldsFragmentDoc,
    "\n  query Repositories {\n    repositories {\n      nodes {\n        name\n      }\n      totalCount\n    }\n  }\n": typeof types.RepositoriesDocument,
    "\n  query Repository($ref: String) {\n    repository(ref: $ref) {\n      name\n    }\n  }\n": typeof types.RepositoryDocument,
    "\n      query Commits($repo: String, $gitRef: String!, $after: String) {\n        repository(ref: $repo) {\n          name\n          commits(ref: $gitRef, after: $after) {\n            nodes {\n              hash\n            }\n          }\n        }\n      }\n    ": typeof types.CommitsDocument,
    "\n  query UserIdentity {\n    repository {\n      userIdentity {\n        ...IdentitySummary\n        id\n        humanId\n        displayName\n        avatarUrl\n        name\n        email\n        login\n      }\n    }\n  }\n": typeof types.UserIdentityDocument,
    "\n  query CodePageRefs($repo: String) {\n    repository(ref: $repo) {\n      name\n      head {\n        shortName\n      }\n      refs {\n        ...RefSelectorRefs\n      }\n    }\n  }\n": typeof types.CodePageRefsDocument,
    "\n  query CodePageBlob($repo: String, $ref: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      blob(ref: $ref, path: $path) {\n        ...FileViewerBlob\n      }\n    }\n  }\n": typeof types.CodePageBlobDocument,
    "\n  query CodePageTree($repo: String, $ref: String!, $path: String) {\n    repository(ref: $repo) {\n      name\n      tree(ref: $ref, path: $path) {\n        name\n        type\n        hash\n      }\n    }\n  }\n": typeof types.CodePageTreeDocument,
    "\n  query CodePageLastCommits($repo: String, $ref: String!, $path: String, $names: [String!]!) {\n    repository(ref: $repo) {\n      name\n      lastCommits(ref: $ref, path: $path, names: $names) {\n        name\n        commit {\n          hash\n          shortHash\n          message\n          date\n        }\n      }\n    }\n  }\n": typeof types.CodePageLastCommitsDocument,
    "\n  query CodePageReadme($repo: String, $ref: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      blob(ref: $ref, path: $path) {\n        text\n      }\n    }\n  }\n": typeof types.CodePageReadmeDocument,
    "\n  query AllIdentities($ref: String) {\n    repository(ref: $ref) {\n      name\n      allIdentities(first: 1000) {\n        nodes {\n          id\n          humanId\n          name\n          email\n          login\n          displayName\n          avatarUrl\n        }\n      }\n    }\n  }\n": typeof types.AllIdentitiesDocument,
    "\n  query ValidLabels($ref: String) {\n    repository(ref: $ref) {\n      name\n      validLabels {\n        nodes {\n          name\n          color {\n            R\n            G\n            B\n          }\n          ...LabelFields\n        }\n      }\n    }\n  }\n": typeof types.ValidLabelsDocument,
    "\n  query BugDetail($ref: String, $prefix: String!) {\n    repository(ref: $ref) {\n      name\n      bug(prefix: $prefix) {\n        ...BugSummary\n        humanId\n        title\n        status\n        createdAt\n        labels {\n          name\n          ...LabelFields\n        }\n        author {\n          humanId\n          displayName\n        }\n        lastEdit\n        participants(first: 20) {\n          nodes {\n            ...IdentitySummary\n            id\n            humanId\n            displayName\n            avatarUrl\n          }\n        }\n        timeline(first: 250) {\n          ...TimelineItems\n        }\n      }\n    }\n  }\n": typeof types.BugDetailDocument,
    "\n  query BugList(\n    $ref: String\n    $openQuery: String!\n    $closedQuery: String!\n    $listQuery: String!\n    $first: Int\n    $after: String\n  ) {\n    repository(ref: $ref) {\n      name\n      openCount: allBugs(query: $openQuery, first: 1) {\n        totalCount\n      }\n      closedCount: allBugs(query: $closedQuery, first: 1) {\n        totalCount\n      }\n      bugs: allBugs(query: $listQuery, first: $first, after: $after) {\n        totalCount\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n        nodes {\n          ...BugSummary\n          id\n          humanId\n          status\n          title\n          createdAt\n          labels {\n            name\n            ...LabelFields\n          }\n          author {\n            humanId\n            displayName\n            ...IdentitySummary\n          }\n          comments {\n            totalCount\n          }\n        }\n      }\n    }\n  }\n": typeof types.BugListDocument,
    "\n  mutation BugCreate($input: BugCreateInput!) {\n    bugCreate(input: $input) {\n      bug {\n        id\n        humanId\n      }\n    }\n  }\n": typeof types.BugCreateDocument,
    "\n  query UserProfile(\n    $ref: String\n    $prefix: String!\n    $openQuery: String!\n    $closedQuery: String!\n    $listQuery: String!\n    $after: String\n  ) {\n    repository(ref: $ref) {\n      name\n      identity(prefix: $prefix) {\n        id\n        humanId\n        name\n        email\n        login\n        displayName\n        avatarUrl\n        isProtected\n      }\n      openCount: allBugs(query: $openQuery, first: 1) {\n        totalCount\n      }\n      closedCount: allBugs(query: $closedQuery, first: 1) {\n        totalCount\n      }\n      bugs: allBugs(query: $listQuery, first: 25, after: $after) {\n        totalCount\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n        nodes {\n          ...BugSummary\n          id\n          humanId\n          status\n          title\n          createdAt\n          labels {\n            name\n            ...LabelFields\n          }\n          author {\n            humanId\n            ...IdentitySummary\n          }\n          comments {\n            totalCount\n          }\n        }\n      }\n    }\n  }\n": typeof types.UserProfileDocument,
    "\n  query CommitPageDetail($repo: String, $hash: String!) {\n    repository(ref: $repo) {\n      name\n      commit(hash: $hash) {\n        hash\n        shortHash\n        message\n        fullMessage\n        authorName\n        authorEmail\n        date\n        parents\n        files {\n          nodes {\n            path\n            oldPath\n            status\n          }\n        }\n      }\n    }\n  }\n": typeof types.CommitPageDetailDocument,
};
const documents: Documents = {
    "\n  mutation BugAddComment($input: BugAddCommentInput!) {\n    bugAddComment(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n": types.BugAddCommentDocument,
    "\n  mutation BugAddCommentAndClose($input: BugAddCommentAndCloseInput!) {\n    bugAddCommentAndClose(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n": types.BugAddCommentAndCloseDocument,
    "\n  mutation BugAddCommentAndReopen($input: BugAddCommentAndReopenInput!) {\n    bugAddCommentAndReopen(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n": types.BugAddCommentAndReopenDocument,
    "\n  mutation BugStatusOpen($input: BugStatusOpenInput!) {\n    bugStatusOpen(input: $input) {\n      bug {\n        id\n        status\n      }\n    }\n  }\n": types.BugStatusOpenDocument,
    "\n  mutation BugStatusClose($input: BugStatusCloseInput!) {\n    bugStatusClose(input: $input) {\n      bug {\n        id\n        status\n      }\n    }\n  }\n": types.BugStatusCloseDocument,
    "\n  mutation BugChangeLabels($input: BugChangeLabelInput) {\n    bugChangeLabels(input: $input) {\n      bug {\n        id\n        labels {\n          name\n          ...LabelFields\n        }\n      }\n    }\n  }\n": types.BugChangeLabelsDocument,
    "\n  fragment BugCreateCommentFields on BugCreateTimelineItem {\n    id\n    author {\n      id\n      humanId\n      displayName\n      ...IdentitySummary\n    }\n    message\n    createdAt\n    lastEdit\n    edited\n  }\n": types.BugCreateCommentFieldsFragmentDoc,
    "\n  fragment BugAddCommentFields on BugAddCommentTimelineItem {\n    id\n    author {\n      id\n      humanId\n      displayName\n      ...IdentitySummary\n    }\n    message\n    createdAt\n    lastEdit\n    edited\n  }\n": types.BugAddCommentFieldsFragmentDoc,
    "\n  fragment LabelChangeFields on BugLabelChangeTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    added {\n      ...LabelFields\n    }\n    removed {\n      ...LabelFields\n    }\n  }\n": types.LabelChangeFieldsFragmentDoc,
    "\n  fragment StatusChangeFields on BugSetStatusTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    status\n  }\n": types.StatusChangeFieldsFragmentDoc,
    "\n  fragment TitleChangeFields on BugSetTitleTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    title\n    was\n  }\n": types.TitleChangeFieldsFragmentDoc,
    "\n  fragment TimelineItems on BugTimelineItemConnection {\n    nodes {\n      __typename\n      id\n      ... on BugCreateTimelineItem {\n        ...BugCreateCommentFields\n      }\n      ... on BugAddCommentTimelineItem {\n        ...BugAddCommentFields\n      }\n      ... on BugLabelChangeTimelineItem {\n        ...LabelChangeFields\n      }\n      ... on BugSetStatusTimelineItem {\n        ...StatusChangeFields\n      }\n      ... on BugSetTitleTimelineItem {\n        ...TitleChangeFields\n      }\n    }\n  }\n": types.TimelineItemsFragmentDoc,
    "\n  mutation BugEditComment($input: BugEditCommentInput!) {\n    bugEditComment(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n": types.BugEditCommentDocument,
    "\n  mutation BugSetTitle($input: BugSetTitleInput!) {\n    bugSetTitle(input: $input) {\n      bug {\n        id\n        title\n      }\n    }\n  }\n": types.BugSetTitleDocument,
    "\n  query CommitList($repo: String, $ref: String!, $path: String, $after: String, $first: Int) {\n    repository(ref: $repo) {\n      name\n      commits(ref: $ref, path: $path, after: $after, first: $first) {\n        nodes {\n          hash\n          shortHash\n          message\n          authorName\n          date\n        }\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n      }\n    }\n  }\n": types.CommitListDocument,
    "\n  query FileDiff($repo: String, $hash: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      commit(hash: $hash) {\n        diff(path: $path) {\n          path\n          oldPath\n          isBinary\n          isNew\n          isDelete\n          hunks {\n            oldStart\n            oldLines\n            newStart\n            newLines\n            lines {\n              type\n              content\n              oldLine\n              newLine\n            }\n          }\n        }\n      }\n    }\n  }\n": types.FileDiffDocument,
    "\n  fragment FileViewerBlob on GitBlob {\n    path\n    hash\n    text\n    size\n    isBinary\n    isTruncated\n  }\n": types.FileViewerBlobFragmentDoc,
    "\n  fragment RefSelectorRefs on GitRefConnection {\n    nodes {\n      name\n      shortName\n      type\n    }\n  }\n": types.RefSelectorRefsFragmentDoc,
    "\n  fragment IdentitySummary on Identity {\n    id\n    humanId\n    displayName\n    avatarUrl\n  }\n": types.IdentitySummaryFragmentDoc,
    "\n  fragment BugSummary on Bug {\n    id\n    humanId\n    status\n    title\n    labels {\n      name\n      ...LabelFields\n    }\n    author {\n      ...IdentitySummary\n    }\n    createdAt\n    comments {\n      totalCount\n    }\n  }\n": types.BugSummaryFragmentDoc,
    "\n  fragment LabelFields on Label {\n    name\n    color {\n      R\n      G\n      B\n    }\n  }\n": types.LabelFieldsFragmentDoc,
    "\n  query Repositories {\n    repositories {\n      nodes {\n        name\n      }\n      totalCount\n    }\n  }\n": types.RepositoriesDocument,
    "\n  query Repository($ref: String) {\n    repository(ref: $ref) {\n      name\n    }\n  }\n": types.RepositoryDocument,
    "\n      query Commits($repo: String, $gitRef: String!, $after: String) {\n        repository(ref: $repo) {\n          name\n          commits(ref: $gitRef, after: $after) {\n            nodes {\n              hash\n            }\n          }\n        }\n      }\n    ": types.CommitsDocument,
    "\n  query UserIdentity {\n    repository {\n      userIdentity {\n        ...IdentitySummary\n        id\n        humanId\n        displayName\n        avatarUrl\n        name\n        email\n        login\n      }\n    }\n  }\n": types.UserIdentityDocument,
    "\n  query CodePageRefs($repo: String) {\n    repository(ref: $repo) {\n      name\n      head {\n        shortName\n      }\n      refs {\n        ...RefSelectorRefs\n      }\n    }\n  }\n": types.CodePageRefsDocument,
    "\n  query CodePageBlob($repo: String, $ref: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      blob(ref: $ref, path: $path) {\n        ...FileViewerBlob\n      }\n    }\n  }\n": types.CodePageBlobDocument,
    "\n  query CodePageTree($repo: String, $ref: String!, $path: String) {\n    repository(ref: $repo) {\n      name\n      tree(ref: $ref, path: $path) {\n        name\n        type\n        hash\n      }\n    }\n  }\n": types.CodePageTreeDocument,
    "\n  query CodePageLastCommits($repo: String, $ref: String!, $path: String, $names: [String!]!) {\n    repository(ref: $repo) {\n      name\n      lastCommits(ref: $ref, path: $path, names: $names) {\n        name\n        commit {\n          hash\n          shortHash\n          message\n          date\n        }\n      }\n    }\n  }\n": types.CodePageLastCommitsDocument,
    "\n  query CodePageReadme($repo: String, $ref: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      blob(ref: $ref, path: $path) {\n        text\n      }\n    }\n  }\n": types.CodePageReadmeDocument,
    "\n  query AllIdentities($ref: String) {\n    repository(ref: $ref) {\n      name\n      allIdentities(first: 1000) {\n        nodes {\n          id\n          humanId\n          name\n          email\n          login\n          displayName\n          avatarUrl\n        }\n      }\n    }\n  }\n": types.AllIdentitiesDocument,
    "\n  query ValidLabels($ref: String) {\n    repository(ref: $ref) {\n      name\n      validLabels {\n        nodes {\n          name\n          color {\n            R\n            G\n            B\n          }\n          ...LabelFields\n        }\n      }\n    }\n  }\n": types.ValidLabelsDocument,
    "\n  query BugDetail($ref: String, $prefix: String!) {\n    repository(ref: $ref) {\n      name\n      bug(prefix: $prefix) {\n        ...BugSummary\n        humanId\n        title\n        status\n        createdAt\n        labels {\n          name\n          ...LabelFields\n        }\n        author {\n          humanId\n          displayName\n        }\n        lastEdit\n        participants(first: 20) {\n          nodes {\n            ...IdentitySummary\n            id\n            humanId\n            displayName\n            avatarUrl\n          }\n        }\n        timeline(first: 250) {\n          ...TimelineItems\n        }\n      }\n    }\n  }\n": types.BugDetailDocument,
    "\n  query BugList(\n    $ref: String\n    $openQuery: String!\n    $closedQuery: String!\n    $listQuery: String!\n    $first: Int\n    $after: String\n  ) {\n    repository(ref: $ref) {\n      name\n      openCount: allBugs(query: $openQuery, first: 1) {\n        totalCount\n      }\n      closedCount: allBugs(query: $closedQuery, first: 1) {\n        totalCount\n      }\n      bugs: allBugs(query: $listQuery, first: $first, after: $after) {\n        totalCount\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n        nodes {\n          ...BugSummary\n          id\n          humanId\n          status\n          title\n          createdAt\n          labels {\n            name\n            ...LabelFields\n          }\n          author {\n            humanId\n            displayName\n            ...IdentitySummary\n          }\n          comments {\n            totalCount\n          }\n        }\n      }\n    }\n  }\n": types.BugListDocument,
    "\n  mutation BugCreate($input: BugCreateInput!) {\n    bugCreate(input: $input) {\n      bug {\n        id\n        humanId\n      }\n    }\n  }\n": types.BugCreateDocument,
    "\n  query UserProfile(\n    $ref: String\n    $prefix: String!\n    $openQuery: String!\n    $closedQuery: String!\n    $listQuery: String!\n    $after: String\n  ) {\n    repository(ref: $ref) {\n      name\n      identity(prefix: $prefix) {\n        id\n        humanId\n        name\n        email\n        login\n        displayName\n        avatarUrl\n        isProtected\n      }\n      openCount: allBugs(query: $openQuery, first: 1) {\n        totalCount\n      }\n      closedCount: allBugs(query: $closedQuery, first: 1) {\n        totalCount\n      }\n      bugs: allBugs(query: $listQuery, first: 25, after: $after) {\n        totalCount\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n        nodes {\n          ...BugSummary\n          id\n          humanId\n          status\n          title\n          createdAt\n          labels {\n            name\n            ...LabelFields\n          }\n          author {\n            humanId\n            ...IdentitySummary\n          }\n          comments {\n            totalCount\n          }\n        }\n      }\n    }\n  }\n": types.UserProfileDocument,
    "\n  query CommitPageDetail($repo: String, $hash: String!) {\n    repository(ref: $repo) {\n      name\n      commit(hash: $hash) {\n        hash\n        shortHash\n        message\n        fullMessage\n        authorName\n        authorEmail\n        date\n        parents\n        files {\n          nodes {\n            path\n            oldPath\n            status\n          }\n        }\n      }\n    }\n  }\n": types.CommitPageDetailDocument,
};

/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 *
 *
 * @example
 * ```ts
 * const query = graphql(`query GetUser($id: ID!) { user(id: $id) { name } }`);
 * ```
 *
 * The query argument is unknown!
 * Please regenerate the types.
 */
export function graphql(source: string): unknown;

/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugAddComment($input: BugAddCommentInput!) {\n    bugAddComment(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugAddComment($input: BugAddCommentInput!) {\n    bugAddComment(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugAddCommentAndClose($input: BugAddCommentAndCloseInput!) {\n    bugAddCommentAndClose(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugAddCommentAndClose($input: BugAddCommentAndCloseInput!) {\n    bugAddCommentAndClose(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugAddCommentAndReopen($input: BugAddCommentAndReopenInput!) {\n    bugAddCommentAndReopen(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugAddCommentAndReopen($input: BugAddCommentAndReopenInput!) {\n    bugAddCommentAndReopen(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugStatusOpen($input: BugStatusOpenInput!) {\n    bugStatusOpen(input: $input) {\n      bug {\n        id\n        status\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugStatusOpen($input: BugStatusOpenInput!) {\n    bugStatusOpen(input: $input) {\n      bug {\n        id\n        status\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugStatusClose($input: BugStatusCloseInput!) {\n    bugStatusClose(input: $input) {\n      bug {\n        id\n        status\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugStatusClose($input: BugStatusCloseInput!) {\n    bugStatusClose(input: $input) {\n      bug {\n        id\n        status\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugChangeLabels($input: BugChangeLabelInput) {\n    bugChangeLabels(input: $input) {\n      bug {\n        id\n        labels {\n          name\n          ...LabelFields\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugChangeLabels($input: BugChangeLabelInput) {\n    bugChangeLabels(input: $input) {\n      bug {\n        id\n        labels {\n          name\n          ...LabelFields\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment BugCreateCommentFields on BugCreateTimelineItem {\n    id\n    author {\n      id\n      humanId\n      displayName\n      ...IdentitySummary\n    }\n    message\n    createdAt\n    lastEdit\n    edited\n  }\n"): (typeof documents)["\n  fragment BugCreateCommentFields on BugCreateTimelineItem {\n    id\n    author {\n      id\n      humanId\n      displayName\n      ...IdentitySummary\n    }\n    message\n    createdAt\n    lastEdit\n    edited\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment BugAddCommentFields on BugAddCommentTimelineItem {\n    id\n    author {\n      id\n      humanId\n      displayName\n      ...IdentitySummary\n    }\n    message\n    createdAt\n    lastEdit\n    edited\n  }\n"): (typeof documents)["\n  fragment BugAddCommentFields on BugAddCommentTimelineItem {\n    id\n    author {\n      id\n      humanId\n      displayName\n      ...IdentitySummary\n    }\n    message\n    createdAt\n    lastEdit\n    edited\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment LabelChangeFields on BugLabelChangeTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    added {\n      ...LabelFields\n    }\n    removed {\n      ...LabelFields\n    }\n  }\n"): (typeof documents)["\n  fragment LabelChangeFields on BugLabelChangeTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    added {\n      ...LabelFields\n    }\n    removed {\n      ...LabelFields\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment StatusChangeFields on BugSetStatusTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    status\n  }\n"): (typeof documents)["\n  fragment StatusChangeFields on BugSetStatusTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    status\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment TitleChangeFields on BugSetTitleTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    title\n    was\n  }\n"): (typeof documents)["\n  fragment TitleChangeFields on BugSetTitleTimelineItem {\n    author {\n      humanId\n      displayName\n    }\n    date\n    title\n    was\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment TimelineItems on BugTimelineItemConnection {\n    nodes {\n      __typename\n      id\n      ... on BugCreateTimelineItem {\n        ...BugCreateCommentFields\n      }\n      ... on BugAddCommentTimelineItem {\n        ...BugAddCommentFields\n      }\n      ... on BugLabelChangeTimelineItem {\n        ...LabelChangeFields\n      }\n      ... on BugSetStatusTimelineItem {\n        ...StatusChangeFields\n      }\n      ... on BugSetTitleTimelineItem {\n        ...TitleChangeFields\n      }\n    }\n  }\n"): (typeof documents)["\n  fragment TimelineItems on BugTimelineItemConnection {\n    nodes {\n      __typename\n      id\n      ... on BugCreateTimelineItem {\n        ...BugCreateCommentFields\n      }\n      ... on BugAddCommentTimelineItem {\n        ...BugAddCommentFields\n      }\n      ... on BugLabelChangeTimelineItem {\n        ...LabelChangeFields\n      }\n      ... on BugSetStatusTimelineItem {\n        ...StatusChangeFields\n      }\n      ... on BugSetTitleTimelineItem {\n        ...TitleChangeFields\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugEditComment($input: BugEditCommentInput!) {\n    bugEditComment(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugEditComment($input: BugEditCommentInput!) {\n    bugEditComment(input: $input) {\n      bug {\n        id\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugSetTitle($input: BugSetTitleInput!) {\n    bugSetTitle(input: $input) {\n      bug {\n        id\n        title\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugSetTitle($input: BugSetTitleInput!) {\n    bugSetTitle(input: $input) {\n      bug {\n        id\n        title\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query CommitList($repo: String, $ref: String!, $path: String, $after: String, $first: Int) {\n    repository(ref: $repo) {\n      name\n      commits(ref: $ref, path: $path, after: $after, first: $first) {\n        nodes {\n          hash\n          shortHash\n          message\n          authorName\n          date\n        }\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query CommitList($repo: String, $ref: String!, $path: String, $after: String, $first: Int) {\n    repository(ref: $repo) {\n      name\n      commits(ref: $ref, path: $path, after: $after, first: $first) {\n        nodes {\n          hash\n          shortHash\n          message\n          authorName\n          date\n        }\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query FileDiff($repo: String, $hash: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      commit(hash: $hash) {\n        diff(path: $path) {\n          path\n          oldPath\n          isBinary\n          isNew\n          isDelete\n          hunks {\n            oldStart\n            oldLines\n            newStart\n            newLines\n            lines {\n              type\n              content\n              oldLine\n              newLine\n            }\n          }\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query FileDiff($repo: String, $hash: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      commit(hash: $hash) {\n        diff(path: $path) {\n          path\n          oldPath\n          isBinary\n          isNew\n          isDelete\n          hunks {\n            oldStart\n            oldLines\n            newStart\n            newLines\n            lines {\n              type\n              content\n              oldLine\n              newLine\n            }\n          }\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment FileViewerBlob on GitBlob {\n    path\n    hash\n    text\n    size\n    isBinary\n    isTruncated\n  }\n"): (typeof documents)["\n  fragment FileViewerBlob on GitBlob {\n    path\n    hash\n    text\n    size\n    isBinary\n    isTruncated\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment RefSelectorRefs on GitRefConnection {\n    nodes {\n      name\n      shortName\n      type\n    }\n  }\n"): (typeof documents)["\n  fragment RefSelectorRefs on GitRefConnection {\n    nodes {\n      name\n      shortName\n      type\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment IdentitySummary on Identity {\n    id\n    humanId\n    displayName\n    avatarUrl\n  }\n"): (typeof documents)["\n  fragment IdentitySummary on Identity {\n    id\n    humanId\n    displayName\n    avatarUrl\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment BugSummary on Bug {\n    id\n    humanId\n    status\n    title\n    labels {\n      name\n      ...LabelFields\n    }\n    author {\n      ...IdentitySummary\n    }\n    createdAt\n    comments {\n      totalCount\n    }\n  }\n"): (typeof documents)["\n  fragment BugSummary on Bug {\n    id\n    humanId\n    status\n    title\n    labels {\n      name\n      ...LabelFields\n    }\n    author {\n      ...IdentitySummary\n    }\n    createdAt\n    comments {\n      totalCount\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  fragment LabelFields on Label {\n    name\n    color {\n      R\n      G\n      B\n    }\n  }\n"): (typeof documents)["\n  fragment LabelFields on Label {\n    name\n    color {\n      R\n      G\n      B\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query Repositories {\n    repositories {\n      nodes {\n        name\n      }\n      totalCount\n    }\n  }\n"): (typeof documents)["\n  query Repositories {\n    repositories {\n      nodes {\n        name\n      }\n      totalCount\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query Repository($ref: String) {\n    repository(ref: $ref) {\n      name\n    }\n  }\n"): (typeof documents)["\n  query Repository($ref: String) {\n    repository(ref: $ref) {\n      name\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n      query Commits($repo: String, $gitRef: String!, $after: String) {\n        repository(ref: $repo) {\n          name\n          commits(ref: $gitRef, after: $after) {\n            nodes {\n              hash\n            }\n          }\n        }\n      }\n    "): (typeof documents)["\n      query Commits($repo: String, $gitRef: String!, $after: String) {\n        repository(ref: $repo) {\n          name\n          commits(ref: $gitRef, after: $after) {\n            nodes {\n              hash\n            }\n          }\n        }\n      }\n    "];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query UserIdentity {\n    repository {\n      userIdentity {\n        ...IdentitySummary\n        id\n        humanId\n        displayName\n        avatarUrl\n        name\n        email\n        login\n      }\n    }\n  }\n"): (typeof documents)["\n  query UserIdentity {\n    repository {\n      userIdentity {\n        ...IdentitySummary\n        id\n        humanId\n        displayName\n        avatarUrl\n        name\n        email\n        login\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query CodePageRefs($repo: String) {\n    repository(ref: $repo) {\n      name\n      head {\n        shortName\n      }\n      refs {\n        ...RefSelectorRefs\n      }\n    }\n  }\n"): (typeof documents)["\n  query CodePageRefs($repo: String) {\n    repository(ref: $repo) {\n      name\n      head {\n        shortName\n      }\n      refs {\n        ...RefSelectorRefs\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query CodePageBlob($repo: String, $ref: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      blob(ref: $ref, path: $path) {\n        ...FileViewerBlob\n      }\n    }\n  }\n"): (typeof documents)["\n  query CodePageBlob($repo: String, $ref: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      blob(ref: $ref, path: $path) {\n        ...FileViewerBlob\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query CodePageTree($repo: String, $ref: String!, $path: String) {\n    repository(ref: $repo) {\n      name\n      tree(ref: $ref, path: $path) {\n        name\n        type\n        hash\n      }\n    }\n  }\n"): (typeof documents)["\n  query CodePageTree($repo: String, $ref: String!, $path: String) {\n    repository(ref: $repo) {\n      name\n      tree(ref: $ref, path: $path) {\n        name\n        type\n        hash\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query CodePageLastCommits($repo: String, $ref: String!, $path: String, $names: [String!]!) {\n    repository(ref: $repo) {\n      name\n      lastCommits(ref: $ref, path: $path, names: $names) {\n        name\n        commit {\n          hash\n          shortHash\n          message\n          date\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query CodePageLastCommits($repo: String, $ref: String!, $path: String, $names: [String!]!) {\n    repository(ref: $repo) {\n      name\n      lastCommits(ref: $ref, path: $path, names: $names) {\n        name\n        commit {\n          hash\n          shortHash\n          message\n          date\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query CodePageReadme($repo: String, $ref: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      blob(ref: $ref, path: $path) {\n        text\n      }\n    }\n  }\n"): (typeof documents)["\n  query CodePageReadme($repo: String, $ref: String!, $path: String!) {\n    repository(ref: $repo) {\n      name\n      blob(ref: $ref, path: $path) {\n        text\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query AllIdentities($ref: String) {\n    repository(ref: $ref) {\n      name\n      allIdentities(first: 1000) {\n        nodes {\n          id\n          humanId\n          name\n          email\n          login\n          displayName\n          avatarUrl\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query AllIdentities($ref: String) {\n    repository(ref: $ref) {\n      name\n      allIdentities(first: 1000) {\n        nodes {\n          id\n          humanId\n          name\n          email\n          login\n          displayName\n          avatarUrl\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query ValidLabels($ref: String) {\n    repository(ref: $ref) {\n      name\n      validLabels {\n        nodes {\n          name\n          color {\n            R\n            G\n            B\n          }\n          ...LabelFields\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query ValidLabels($ref: String) {\n    repository(ref: $ref) {\n      name\n      validLabels {\n        nodes {\n          name\n          color {\n            R\n            G\n            B\n          }\n          ...LabelFields\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query BugDetail($ref: String, $prefix: String!) {\n    repository(ref: $ref) {\n      name\n      bug(prefix: $prefix) {\n        ...BugSummary\n        humanId\n        title\n        status\n        createdAt\n        labels {\n          name\n          ...LabelFields\n        }\n        author {\n          humanId\n          displayName\n        }\n        lastEdit\n        participants(first: 20) {\n          nodes {\n            ...IdentitySummary\n            id\n            humanId\n            displayName\n            avatarUrl\n          }\n        }\n        timeline(first: 250) {\n          ...TimelineItems\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query BugDetail($ref: String, $prefix: String!) {\n    repository(ref: $ref) {\n      name\n      bug(prefix: $prefix) {\n        ...BugSummary\n        humanId\n        title\n        status\n        createdAt\n        labels {\n          name\n          ...LabelFields\n        }\n        author {\n          humanId\n          displayName\n        }\n        lastEdit\n        participants(first: 20) {\n          nodes {\n            ...IdentitySummary\n            id\n            humanId\n            displayName\n            avatarUrl\n          }\n        }\n        timeline(first: 250) {\n          ...TimelineItems\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query BugList(\n    $ref: String\n    $openQuery: String!\n    $closedQuery: String!\n    $listQuery: String!\n    $first: Int\n    $after: String\n  ) {\n    repository(ref: $ref) {\n      name\n      openCount: allBugs(query: $openQuery, first: 1) {\n        totalCount\n      }\n      closedCount: allBugs(query: $closedQuery, first: 1) {\n        totalCount\n      }\n      bugs: allBugs(query: $listQuery, first: $first, after: $after) {\n        totalCount\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n        nodes {\n          ...BugSummary\n          id\n          humanId\n          status\n          title\n          createdAt\n          labels {\n            name\n            ...LabelFields\n          }\n          author {\n            humanId\n            displayName\n            ...IdentitySummary\n          }\n          comments {\n            totalCount\n          }\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query BugList(\n    $ref: String\n    $openQuery: String!\n    $closedQuery: String!\n    $listQuery: String!\n    $first: Int\n    $after: String\n  ) {\n    repository(ref: $ref) {\n      name\n      openCount: allBugs(query: $openQuery, first: 1) {\n        totalCount\n      }\n      closedCount: allBugs(query: $closedQuery, first: 1) {\n        totalCount\n      }\n      bugs: allBugs(query: $listQuery, first: $first, after: $after) {\n        totalCount\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n        nodes {\n          ...BugSummary\n          id\n          humanId\n          status\n          title\n          createdAt\n          labels {\n            name\n            ...LabelFields\n          }\n          author {\n            humanId\n            displayName\n            ...IdentitySummary\n          }\n          comments {\n            totalCount\n          }\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation BugCreate($input: BugCreateInput!) {\n    bugCreate(input: $input) {\n      bug {\n        id\n        humanId\n      }\n    }\n  }\n"): (typeof documents)["\n  mutation BugCreate($input: BugCreateInput!) {\n    bugCreate(input: $input) {\n      bug {\n        id\n        humanId\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query UserProfile(\n    $ref: String\n    $prefix: String!\n    $openQuery: String!\n    $closedQuery: String!\n    $listQuery: String!\n    $after: String\n  ) {\n    repository(ref: $ref) {\n      name\n      identity(prefix: $prefix) {\n        id\n        humanId\n        name\n        email\n        login\n        displayName\n        avatarUrl\n        isProtected\n      }\n      openCount: allBugs(query: $openQuery, first: 1) {\n        totalCount\n      }\n      closedCount: allBugs(query: $closedQuery, first: 1) {\n        totalCount\n      }\n      bugs: allBugs(query: $listQuery, first: 25, after: $after) {\n        totalCount\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n        nodes {\n          ...BugSummary\n          id\n          humanId\n          status\n          title\n          createdAt\n          labels {\n            name\n            ...LabelFields\n          }\n          author {\n            humanId\n            ...IdentitySummary\n          }\n          comments {\n            totalCount\n          }\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query UserProfile(\n    $ref: String\n    $prefix: String!\n    $openQuery: String!\n    $closedQuery: String!\n    $listQuery: String!\n    $after: String\n  ) {\n    repository(ref: $ref) {\n      name\n      identity(prefix: $prefix) {\n        id\n        humanId\n        name\n        email\n        login\n        displayName\n        avatarUrl\n        isProtected\n      }\n      openCount: allBugs(query: $openQuery, first: 1) {\n        totalCount\n      }\n      closedCount: allBugs(query: $closedQuery, first: 1) {\n        totalCount\n      }\n      bugs: allBugs(query: $listQuery, first: 25, after: $after) {\n        totalCount\n        pageInfo {\n          hasNextPage\n          endCursor\n        }\n        nodes {\n          ...BugSummary\n          id\n          humanId\n          status\n          title\n          createdAt\n          labels {\n            name\n            ...LabelFields\n          }\n          author {\n            humanId\n            ...IdentitySummary\n          }\n          comments {\n            totalCount\n          }\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query CommitPageDetail($repo: String, $hash: String!) {\n    repository(ref: $repo) {\n      name\n      commit(hash: $hash) {\n        hash\n        shortHash\n        message\n        fullMessage\n        authorName\n        authorEmail\n        date\n        parents\n        files {\n          nodes {\n            path\n            oldPath\n            status\n          }\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query CommitPageDetail($repo: String, $hash: String!) {\n    repository(ref: $repo) {\n      name\n      commit(hash: $hash) {\n        hash\n        shortHash\n        message\n        fullMessage\n        authorName\n        authorEmail\n        date\n        parents\n        files {\n          nodes {\n            path\n            oldPath\n            status\n          }\n        }\n      }\n    }\n  }\n"];

export function graphql(source: string) {
  return (documents as any)[source] ?? {};
}

export type DocumentType<TDocumentNode extends DocumentNode<any, any>> = TDocumentNode extends DocumentNode<  infer TType,  any>  ? TType  : never;