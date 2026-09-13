// Paginated commit history grouped by date. Uses MockedProvider so the query
// resolves to real-looking data in Storybook without a running backend.
import { MockedProvider } from "@apollo/client/testing/react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import type { Decorator } from "@storybook/react-vite";
import { Suspense } from "react";

import { withRouter } from "@/../.storybook/decorators";
import { CommitListDocument } from "@/__generated__/graphql";

import { CommitList } from "./commit-list";

function makeCommit(i: number, daysAgo: number) {
  const hash = `abc${String(i).padStart(3, "0")}def456789`.padEnd(40, "0");
  return {
    hash,
    shortHash: hash.slice(0, 7),
    message: [
      "Fix login page crash on empty email",
      "Add pagination to issue list",
      "Refactor label editor to use compound component",
      "Update dependencies",
      "Fix broken link in README",
    ][i % 5]!,
    authorName: i % 2 === 0 ? "Jane Doe" : "Bob Smith",
    date: new Date(Date.now() - daysAgo * 86400 * 1000).toISOString(),
  };
}

const TODAY_COMMITS = [makeCommit(0, 0), makeCommit(1, 0), makeCommit(2, 0)];
const YESTERDAY_COMMITS = [makeCommit(3, 1), makeCommit(4, 1)];
const ALL_COMMITS = [...TODAY_COMMITS, ...YESTERDAY_COMMITS];

function withCommitsMock(commits: typeof ALL_COMMITS): Decorator {
  const mocks = [
    {
      request: {
        query: CommitListDocument,
        variables: { repo: "_", ref: "main", path: null, after: null, first: 30 },
      },
      result: {
        data: {
          repository: {
            // Selected by the query and required by the cache key policy
            name: "_",
            commits: {
              nodes: commits,
              pageInfo: { hasNextPage: false, endCursor: null },
            },
          },
        },
      },
    },
  ];
  return (Story) => (
    <MockedProvider mocks={mocks}>
      <Suspense fallback={<div style={{ padding: 16 }}>Loading…</div>}>
        <Story />
      </Suspense>
    </MockedProvider>
  );
}

const meta = {
  component: CommitList,
  decorators: [withRouter],
  parameters: { layout: "padded" },
} satisfies Meta<typeof CommitList>;

export default meta;
type Story = StoryObj<typeof meta>;

export const WithCommits: Story = {
  decorators: [withCommitsMock(ALL_COMMITS)],
  args: {
    repo: "_",
    ref_: "main",
  },
};

export const EmptyRepo: Story = {
  decorators: [withCommitsMock([])],
  args: {
    repo: "_",
    ref_: "main",
  },
};
