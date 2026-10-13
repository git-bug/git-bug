import type { MockedResponse } from "@apollo/client/testing";
import { MockedProvider } from "@apollo/client/testing/react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { describe, it, expect, vi } from "vitest";

import type { FragmentType } from "@/__generated__/fragment-masking";
import { BugChangeLabelsDocument, BugDetailDocument } from "@/__generated__/graphql";
import type { LABEL_FIELDS_FRAGMENT } from "@/components/shared/label-badge";
import { useAuth } from "@/lib/auth";

import { LabelEditor } from "./label-editor";

type MockUser = {
  id: string;
  humanId: string;
  displayName: string;
  avatarUrl: string | null;
} | null;
vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn<() => { user: MockUser }>(() => ({
    user: { id: "user-1", humanId: "u1", displayName: "Test User", avatarUrl: null },
  })),
}));

const REFETCH_MOCK = {
  request: { query: BugDetailDocument, variables: vi.fn().mockReturnValue(true) },
  result: { data: { repository: null } },
  maxUsageCount: 10,
};

// useFragment is a passthrough so plain objects work as fragment types.
type LabelData = { name: string; color: { R: number; G: number; B: number } } & FragmentType<
  typeof LABEL_FIELDS_FRAGMENT
>;

const LABEL_BUG: LabelData = { name: "bug", color: { R: 255, G: 0, B: 0 } };
const LABEL_URGENT: LabelData = { name: "urgent", color: { R: 255, G: 165, B: 0 } };

function renderEditor(props: {
  currentLabels?: LabelData[];
  validLabels?: LabelData[];
  repo?: string | null;
  mocks?: MockedResponse<any, any>[];
}) {
  const {
    currentLabels = [],
    validLabels = [LABEL_BUG, LABEL_URGENT],
    repo = "myrepo",
    mocks = [],
  } = props;

  return render(
    <MockedProvider mocks={mocks} showWarnings={false}>
      <Suspense>
        <LabelEditor
          bugPrefix="bug1"
          currentLabels={currentLabels}
          validLabels={validLabels}
          repo={repo}
        />
      </Suspense>
    </MockedProvider>,
  );
}

function openPopover() {
  // floating-ui's useRole sets role="combobox" on the trigger button
  fireEvent.click(screen.getByRole("combobox"));
}

describe("LabelEditor — display", () => {
  it("shows no gear icon when there is no logged-in user", async () => {
    vi.mocked(useAuth).mockReturnValueOnce({ user: null });

    renderEditor({});
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("shows no gear icon when validLabels is empty", () => {
    renderEditor({ validLabels: [] });
    expect(screen.queryByRole("button")).toBeNull();
  });

  it('shows "None yet" when there are no current labels', () => {
    renderEditor({ currentLabels: [] });
    expect(screen.getByText("None yet")).toBeInTheDocument();
  });

  it("renders current labels", () => {
    renderEditor({ currentLabels: [LABEL_BUG] });
    expect(screen.getByText("bug")).toBeInTheDocument();
  });
});

describe("LabelEditor — popover", () => {
  it("opens the label list on gear click", () => {
    renderEditor({});
    expect(screen.queryByText("Apply labels")).toBeNull();
    openPopover();
    expect(screen.getByText("Apply labels")).toBeInTheDocument();
  });

  it("lists all valid labels in the popover", () => {
    renderEditor({});
    openPopover();
    expect(screen.getByText("bug")).toBeInTheDocument();
    expect(screen.getByText("urgent")).toBeInTheDocument();
  });
});

describe("LabelEditor — mutation variables", () => {
  const PREFIX = "bug1";
  const REPO = "myrepo";

  it("adds a label — passes prefix, repoRef, added, and empty Removed", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugChangeLabelsDocument, variables: matchVars },
      result: { data: { bugChangeLabels: { bug: { id: "1", labels: [] } } } },
    };

    renderEditor({ currentLabels: [], mocks: [mock, REFETCH_MOCK] });
    openPopover();
    fireEvent.click(screen.getByRole("option", { name: "bug" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, repoRef: REPO, added: ["bug"], Removed: [] },
    });
  });

  it("removes a label — passes prefix, repoRef, empty added, and Removed", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugChangeLabelsDocument, variables: matchVars },
      result: { data: { bugChangeLabels: { bug: { id: "1", labels: [] } } } },
    };

    renderEditor({ currentLabels: [LABEL_BUG], mocks: [mock, REFETCH_MOCK] });
    openPopover();
    fireEvent.click(screen.getByRole("option", { name: "bug" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, repoRef: REPO, added: [], Removed: ["bug"] },
    });
  });

  it("works with null repoRef (default repo)", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugChangeLabelsDocument, variables: matchVars },
      result: { data: { bugChangeLabels: { bug: { id: "1", labels: [] } } } },
    };

    renderEditor({ currentLabels: [], repo: null, mocks: [mock, REFETCH_MOCK] });
    openPopover();
    fireEvent.click(screen.getByRole("option", { name: "bug" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: "bug1", repoRef: null, added: ["bug"], Removed: [] },
    });
  });
});
