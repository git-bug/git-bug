// Write/preview comment form used at the bottom of the bug detail page.
// Shows different close/reopen button labels depending on bug status.
import type { Meta, StoryObj } from "@storybook/react-vite";

import { withApollo, withRouter } from "@/../.storybook/decorators";
import { Status } from "@/__generated__/enums";

import { CommentBox } from "./comment-box";

const meta = {
  component: CommentBox,
  decorators: [withRouter, withApollo],
  parameters: { layout: "padded" },
} satisfies Meta<typeof CommentBox>;

export default meta;
type Story = StoryObj<typeof meta>;

export const OpenIssue: Story = {
  args: {
    bugPrefix: "abc123",
    bugStatus: Status.Open,
    ref_: "_",
  },
};

export const ClosedIssue: Story = {
  args: {
    bugPrefix: "abc123",
    bugStatus: Status.Closed,
    ref_: "_",
  },
};
