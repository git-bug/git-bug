import type { Meta, StoryObj } from "@storybook/react-vite";

import { withRouter } from "@/../.storybook/decorators";
import { GitObjectType } from "@/__generated__/enums";

import { FileTree } from "./file-tree";

const meta = {
  component: FileTree,
  decorators: [withRouter],
} satisfies Meta<typeof FileTree>;

export default meta;
type Story = StoryObj<typeof meta>;

const entries = [
  {
    name: "src",
    type: GitObjectType.Tree,
    hash: "abc1",
    lastCommit: {
      hash: "def1",
      shortHash: "def1",
      message: "refactor: restructure source directory",
      date: new Date(Date.now() - 86400 * 1000).toISOString(),
    },
  },
  {
    name: "docs",
    type: GitObjectType.Tree,
    hash: "abc2",
    lastCommit: {
      hash: "def2",
      shortHash: "def2",
      message: "docs: update getting started guide",
      date: new Date(Date.now() - 172800 * 1000).toISOString(),
    },
  },
  {
    name: "README.md",
    type: GitObjectType.Blob,
    hash: "abc3",
    lastCommit: {
      hash: "def3",
      shortHash: "def3",
      message: "docs: add badges to README",
      date: new Date(Date.now() - 3600 * 1000).toISOString(),
    },
  },
  {
    name: "package.json",
    type: GitObjectType.Blob,
    hash: "abc4",
    lastCommit: {
      hash: "def4",
      shortHash: "def4",
      message: "chore: bump dependencies",
      date: new Date(Date.now() - 7200 * 1000).toISOString(),
    },
  },
  {
    name: ".gitignore",
    type: GitObjectType.Blob,
    hash: "abc5",
  },
];

export const RootDirectory: Story = {
  args: {
    repo: "my-repo",
    currentRef: "main",
    currentPath: "",
    entries,
  },
};

export const SubDirectory: Story = {
  args: {
    repo: "my-repo",
    currentRef: "main",
    currentPath: "src",
    entries: [
      {
        name: "components",
        type: GitObjectType.Tree,
        hash: "abc6",
        lastCommit: {
          hash: "def5",
          shortHash: "def5",
          message: "feat: add button component",
          date: new Date(Date.now() - 3600 * 1000).toISOString(),
        },
      },
      {
        name: "index.ts",
        type: GitObjectType.Blob,
        hash: "abc7",
        lastCommit: {
          hash: "def6",
          shortHash: "def6",
          message: "fix: correct export paths",
          date: new Date(Date.now() - 7200 * 1000).toISOString(),
        },
      },
    ],
  },
};
