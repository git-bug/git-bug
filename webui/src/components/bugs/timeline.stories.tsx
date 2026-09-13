import type { Meta, StoryObj } from "@storybook/react-vite";

import { withApollo, withCachedFragments, withRouter } from "@/../.storybook/decorators";
import { Status } from "@/__generated__/enums";
import { makeFragmentData } from "@/__generated__/fragment-masking";

import { Timeline, TIMELINE_ITEMS_FRAGMENT } from "./timeline";

const meta = {
  component: Timeline,
  decorators: [withRouter, withApollo],
  parameters: { a11y: { disable: true } },
} satisfies Meta<typeof Timeline>;

export default meta;
type Story = StoryObj<typeof meta>;

const jane = {
  __typename: "Identity" as const,
  id: "u1",
  humanId: "jane1",
  displayName: "Jane Doe",
  avatarUrl: null,
};

const bob = {
  __typename: "Identity" as const,
  id: "u2",
  humanId: "bob1",
  displayName: "Bob Smith",
  avatarUrl: "https://github.com/shadcn.png",
};

const baseTimelineData = {
  __typename: "BugTimelineItemConnection" as const,
  nodes: [
    {
      __typename: "BugCreateTimelineItem" as const,
      id: "create-1",
      author: jane,
      message: "This is the **initial bug report** with some markdown.\n\n- Item 1\n- Item 2",
      createdAt: new Date(Date.now() - 86400 * 1000).toISOString(),
      lastEdit: new Date(Date.now() - 86400 * 1000).toISOString(),
      edited: false,
    },
    {
      __typename: "BugLabelChangeTimelineItem" as const,
      id: "label-1",
      author: { humanId: "bob1", displayName: "Bob Smith" },
      date: new Date(Date.now() - 43200 * 1000).toISOString(),
      added: [{ __typename: "Label" as const, name: "bug", color: { R: 252, G: 41, B: 41 } }],
      removed: [],
    },
    {
      __typename: "BugAddCommentTimelineItem" as const,
      id: "comment-1",
      author: bob,
      message: "I can reproduce this. The issue is in the login handler.",
      createdAt: new Date(Date.now() - 21600 * 1000).toISOString(),
      lastEdit: new Date(Date.now() - 21600 * 1000).toISOString(),
      edited: false,
    },
    {
      __typename: "BugSetTitleTimelineItem" as const,
      id: "title-1",
      author: { humanId: "jane1", displayName: "Jane Doe" },
      date: new Date(Date.now() - 7200 * 1000).toISOString(),
      title: "Login page crash on empty email input",
      was: "Login page crash",
    },
    {
      __typename: "BugAddCommentTimelineItem" as const,
      id: "comment-2",
      author: jane,
      message: "Fixed in commit abc123. The email validator was not handling empty strings.",
      createdAt: new Date(Date.now() - 3600 * 1000).toISOString(),
      lastEdit: new Date(Date.now() - 1800 * 1000).toISOString(),
      edited: true,
    },
    {
      __typename: "BugSetStatusTimelineItem" as const,
      id: "status-1",
      author: { humanId: "jane1", displayName: "Jane Doe" },
      date: new Date(Date.now() - 3600 * 1000).toISOString(),
      status: Status.Closed,
    },
  ],
};

const baseTimeline = makeFragmentData(baseTimelineData, TIMELINE_ITEMS_FRAGMENT);

export const FullTimeline: Story = {
  decorators: [withCachedFragments([TIMELINE_ITEMS_FRAGMENT, "TimelineItems", baseTimelineData])],
  args: {
    repo: "_",
    bugPrefix: "abc123",
    timeline: baseTimeline,
  },
};

const createOnlyTimelineData = {
  __typename: "BugTimelineItemConnection" as const,
  nodes: [baseTimelineData.nodes[0]!],
};

const createOnlyTimeline = makeFragmentData(createOnlyTimelineData, TIMELINE_ITEMS_FRAGMENT);

export const CreateOnly: Story = {
  decorators: [
    withCachedFragments([TIMELINE_ITEMS_FRAGMENT, "TimelineItems", createOnlyTimelineData]),
  ],
  args: {
    repo: "_",
    bugPrefix: "abc123",
    timeline: createOnlyTimeline,
  },
};

const emptyMessageTimelineData = {
  __typename: "BugTimelineItemConnection" as const,
  nodes: [
    {
      __typename: "BugCreateTimelineItem" as const,
      id: "create-empty",
      author: jane,
      message: "",
      createdAt: new Date().toISOString(),
      lastEdit: new Date().toISOString(),
      edited: false,
    },
  ],
};

const emptyMessageTimeline = makeFragmentData(emptyMessageTimelineData, TIMELINE_ITEMS_FRAGMENT);

export const EmptyMessage: Story = {
  decorators: [
    withCachedFragments([TIMELINE_ITEMS_FRAGMENT, "TimelineItems", emptyMessageTimelineData]),
  ],
  args: {
    repo: "_",
    bugPrefix: "abc123",
    timeline: emptyMessageTimeline,
  },
};

const statusReopenTimelineData = {
  __typename: "BugTimelineItemConnection" as const,
  nodes: [
    baseTimelineData.nodes[0]!,
    {
      __typename: "BugSetStatusTimelineItem" as const,
      id: "status-close",
      author: { humanId: "bob1", displayName: "Bob Smith" },
      date: new Date(Date.now() - 7200 * 1000).toISOString(),
      status: Status.Closed,
    },
    {
      __typename: "BugSetStatusTimelineItem" as const,
      id: "status-reopen",
      author: { humanId: "jane1", displayName: "Jane Doe" },
      date: new Date(Date.now() - 3600 * 1000).toISOString(),
      status: Status.Open,
    },
  ],
};

const statusReopenTimeline = makeFragmentData(statusReopenTimelineData, TIMELINE_ITEMS_FRAGMENT);

export const StatusReopen: Story = {
  decorators: [
    withCachedFragments([TIMELINE_ITEMS_FRAGMENT, "TimelineItems", statusReopenTimelineData]),
  ],
  args: {
    repo: "_",
    bugPrefix: "abc123",
    timeline: statusReopenTimeline,
  },
};
