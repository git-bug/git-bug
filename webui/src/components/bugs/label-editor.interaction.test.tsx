import type { MockedResponse } from "@apollo/client/testing";
import { MockedProvider } from "@apollo/client/testing/react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { describe, it, expect, onTestFinished, vi } from "vitest";

import type { FragmentType } from "@/__generated__/fragment-masking";
import { BugChangeLabelsDocument } from "@/__generated__/graphql";
import type { LABEL_FIELDS_FRAGMENT } from "@/components/shared/label-badge";
import { toastManager } from "@/components/ui/toast";
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

describe("LabelEditor — applying labels", () => {
  const PREFIX = "bug1";
  const REPO = "myrepo";

  function mockChangeLabels() {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugChangeLabelsDocument, variables: matchVars },
      result: { data: { bugChangeLabels: { bug: { id: "1", labels: [] } } } },
    };
    return { matchVars, mock };
  }

  const toggle = (name: string) => fireEvent.click(screen.getByRole("option", { name }));
  const closePopover = () => fireEvent.keyDown(document.body, { key: "Escape" });

  it("adds a label on close — passes prefix, repoRef, added, and empty Removed", async () => {
    const { matchVars, mock } = mockChangeLabels();
    renderEditor({ currentLabels: [], mocks: [mock] });
    openPopover();
    toggle("bug");
    expect(matchVars).not.toHaveBeenCalled();
    closePopover();

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, repoRef: REPO, added: ["bug"], Removed: [] },
    });
  });

  it("removes a label on close — passes prefix, repoRef, empty added, and Removed", async () => {
    const { matchVars, mock } = mockChangeLabels();
    renderEditor({ currentLabels: [LABEL_BUG], mocks: [mock] });
    openPopover();
    toggle("bug");
    closePopover();

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, repoRef: REPO, added: [], Removed: ["bug"] },
    });
  });

  it("applies all the toggled labels in a single mutation", async () => {
    const { matchVars, mock } = mockChangeLabels();
    renderEditor({ currentLabels: [LABEL_BUG], mocks: [mock] });
    openPopover();
    toggle("bug");
    toggle("urgent");
    closePopover();

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledTimes(1);
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, repoRef: REPO, added: ["urgent"], Removed: ["bug"] },
    });
  });

  it("applies nothing when a label is toggled back", async () => {
    const { matchVars, mock } = mockChangeLabels();
    renderEditor({ currentLabels: [], mocks: [mock] });
    openPopover();
    toggle("bug");
    toggle("bug");
    closePopover();

    await waitFor(() => expect(screen.queryByText("Apply labels")).toBeNull());
    expect(matchVars).not.toHaveBeenCalled();
  });

  it("applies the labels when clicking outside", async () => {
    const { matchVars, mock } = mockChangeLabels();
    renderEditor({ currentLabels: [], mocks: [mock] });
    openPopover();
    toggle("bug");
    fireEvent.pointerDown(document.body);
    fireEvent.click(document.body);

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
  });

  it("shows the toggled labels as selected before applying them", () => {
    renderEditor({ currentLabels: [] });
    openPopover();
    toggle("bug");
    expect(screen.getByRole("option", { name: "bug" })).toHaveAttribute("aria-selected", "true");
  });

  it("works with null repoRef (default repo)", async () => {
    const { matchVars, mock } = mockChangeLabels();
    renderEditor({ currentLabels: [], repo: null, mocks: [mock] });
    openPopover();
    toggle("bug");
    closePopover();

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: "bug1", repoRef: null, added: ["bug"], Removed: [] },
    });
  });
});

describe("LabelEditor — failed save", () => {
  it("restores the choices and reports the error", async () => {
    const addToast = vi.spyOn(toastManager, "add");
    onTestFinished(() => addToast.mockRestore());
    const mock = {
      request: { query: BugChangeLabelsDocument, variables: () => true },
      error: new Error("network down"),
    };
    renderEditor({ currentLabels: [], mocks: [mock] });
    openPopover();
    fireEvent.click(screen.getByRole("option", { name: "bug" }));
    fireEvent.keyDown(document.body, { key: "Escape" });

    await waitFor(() =>
      expect(addToast).toHaveBeenCalledWith({
        title: "Failed to save the labels",
        description: "network down",
      }),
    );
    openPopover();
    expect(screen.getByRole("option", { name: "bug" })).toHaveAttribute("aria-selected", "true");
  });

  it("can't be reopened while saving", async () => {
    const mock = {
      request: { query: BugChangeLabelsDocument, variables: () => true },
      result: { data: { bugChangeLabels: { bug: { id: "1", labels: [] } } } },
      delay: 50,
    };
    renderEditor({ currentLabels: [], mocks: [mock] });
    openPopover();
    fireEvent.click(screen.getByRole("option", { name: "bug" }));
    fireEvent.keyDown(document.body, { key: "Escape" });

    await waitFor(() => expect(screen.getByRole("combobox")).toBeDisabled());
    await waitFor(() => expect(screen.getByRole("combobox")).toBeEnabled());
  });
});

describe("LabelEditor — leaving with the picker open", () => {
  it("applies the choices, like closing the picker", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugChangeLabelsDocument, variables: matchVars },
      result: { data: { bugChangeLabels: { bug: { id: "1", labels: [] } } } },
    };
    const { unmount } = renderEditor({ currentLabels: [], mocks: [mock] });
    openPopover();
    fireEvent.click(screen.getByRole("option", { name: "bug" }));
    expect(matchVars).not.toHaveBeenCalled();
    unmount();

    await waitFor(() =>
      expect(matchVars).toHaveBeenCalledWith({
        input: { prefix: "bug1", repoRef: "myrepo", added: ["bug"], Removed: [] },
      }),
    );
  });

  it("applies nothing when nothing was toggled", () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugChangeLabelsDocument, variables: matchVars },
      result: { data: { bugChangeLabels: { bug: { id: "1", labels: [] } } } },
    };
    const { unmount } = renderEditor({ currentLabels: [], mocks: [mock] });
    openPopover();
    unmount();

    expect(matchVars).not.toHaveBeenCalled();
  });
});
