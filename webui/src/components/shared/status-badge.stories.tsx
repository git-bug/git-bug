import type { Meta, StoryObj } from "@storybook/react-vite";

import { Status } from "@/__generated__/enums";

import { StatusBadge } from "./status-badge";

const meta = {
  component: StatusBadge,
  argTypes: {
    status: {
      control: "select",
      options: [Status.Open, Status.Closed],
    },
  },
} satisfies Meta<typeof StatusBadge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Open: Story = {
  args: { status: Status.Open },
};

export const Closed: Story = {
  args: { status: Status.Closed },
};
