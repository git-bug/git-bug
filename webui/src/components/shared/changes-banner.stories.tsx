import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { ChangesBanner } from "./changes-banner";

const meta = {
  component: ChangesBanner,
  args: { onRefresh: fn() },
} satisfies Meta<typeof ChangesBanner>;

export default meta;
type Story = StoryObj<typeof meta>;

export const OneChange: Story = {
  args: { count: 1 },
};

export const SeveralChanges: Story = {
  args: { count: 3 },
  play: async ({ args, canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole("button", { name: "Refresh" }));
    await expect(args.onRefresh).toHaveBeenCalledOnce();
  },
};

export const NoChange: Story = {
  args: { count: 0 },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).queryByRole("status")).toBeNull();
  },
};
