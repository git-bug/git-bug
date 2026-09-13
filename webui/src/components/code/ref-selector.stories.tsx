import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { withApollo, withCachedFragments } from "@/../.storybook/decorators";
import { GitRefType } from "@/__generated__/enums";
import { makeFragmentData } from "@/__generated__/fragment-masking";

import { RefSelector, REF_SELECTOR_REFS_FRAGMENT } from "./ref-selector";

const sampleRefsData = {
  __typename: "GitRefConnection" as const,
  nodes: [
    {
      __typename: "GitRef" as const,
      name: "refs/heads/main",
      shortName: "main",
      type: GitRefType.Branch,
    },
    {
      __typename: "GitRef" as const,
      name: "refs/heads/develop",
      shortName: "develop",
      type: GitRefType.Branch,
    },
    {
      __typename: "GitRef" as const,
      name: "refs/heads/feature/auth",
      shortName: "feature/auth",
      type: GitRefType.Branch,
    },
    {
      __typename: "GitRef" as const,
      name: "refs/heads/fix/login",
      shortName: "fix/login",
      type: GitRefType.Branch,
    },
    {
      __typename: "GitRef" as const,
      name: "refs/tags/v1.0.0",
      shortName: "v1.0.0",
      type: GitRefType.Tag,
    },
    {
      __typename: "GitRef" as const,
      name: "refs/tags/v1.1.0",
      shortName: "v1.1.0",
      type: GitRefType.Tag,
    },
    {
      __typename: "GitRef" as const,
      name: "refs/tags/v2.0.0-rc1",
      shortName: "v2.0.0-rc1",
      type: GitRefType.Tag,
    },
  ],
};

const branchesOnlyData = {
  __typename: "GitRefConnection" as const,
  nodes: sampleRefsData.nodes.filter((r) => r.type === GitRefType.Branch),
};

const sampleRefs = makeFragmentData(sampleRefsData, REF_SELECTOR_REFS_FRAGMENT);
const branchesOnly = makeFragmentData(branchesOnlyData, REF_SELECTOR_REFS_FRAGMENT);

const meta = {
  component: RefSelector,
  decorators: [
    withApollo,
    withCachedFragments(
      [REF_SELECTOR_REFS_FRAGMENT, "RefSelectorRefs", sampleRefsData],
      [REF_SELECTOR_REFS_FRAGMENT, "RefSelectorRefs", branchesOnlyData],
    ),
  ],
  parameters: { a11y: { disable: true } },
} satisfies Meta<typeof RefSelector>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    refs: sampleRefs,
    currentRef: "main",
    onSelect: fn(),
  },
};

export const OnTag: Story = {
  args: {
    refs: sampleRefs,
    currentRef: "v1.1.0",
    onSelect: fn(),
  },
};

export const BranchesOnly: Story = {
  args: {
    refs: branchesOnly,
    currentRef: "develop",
    onSelect: fn(),
  },
};
