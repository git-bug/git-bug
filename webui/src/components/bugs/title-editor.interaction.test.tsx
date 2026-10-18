import { MockedProvider } from "@apollo/client/testing/react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { describe, it, expect, vi } from "vitest";

import { BugSetTitleDocument } from "@/__generated__/graphql";
import { useAuth } from "@/lib/auth";

import { TitleEditor } from "./title-editor";

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

function renderEditor(props: { title?: string; repo?: string | null } = {}) {
  const { title = "Original title", repo = "myrepo" } = props;
  return render(
    <MockedProvider showWarnings={false}>
      <Suspense>
        <TitleEditor bugPrefix="bug1" title={title} humanId="h1" repo={repo} />
      </Suspense>
    </MockedProvider>,
  );
}

describe("TitleEditor — display", () => {
  it("renders the title and human id in read mode", () => {
    renderEditor({ title: "Fix login crash" });
    expect(screen.getByText("Fix login crash")).toBeInTheDocument();
    expect(screen.getByText("#h1")).toBeInTheDocument();
  });

  it("shows the pencil button on hover (auth gate)", () => {
    renderEditor();
    expect(screen.getByTitle("Edit title")).toBeInTheDocument();
  });

  it("hides the pencil button when no user is logged in", async () => {
    vi.mocked(useAuth).mockReturnValueOnce({ user: null });

    renderEditor();
    expect(screen.queryByTitle("Edit title")).toBeNull();
  });
});

describe("TitleEditor — editing", () => {
  it("enters edit mode on pencil click", () => {
    renderEditor();
    fireEvent.click(screen.getByTitle("Edit title"));
    expect(screen.getByRole("textbox")).toHaveValue("Original title");
  });

  it("cancels on Escape — reverts to original title", () => {
    renderEditor({ title: "Original" });
    fireEvent.click(screen.getByTitle("Edit title"));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "Changed" } });
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Escape" });
    expect(screen.getByText("Original")).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("Save button is disabled when input is empty", () => {
    renderEditor();
    fireEvent.click(screen.getByTitle("Edit title"));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "   " } });
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });
});

describe("TitleEditor — mutation variables", () => {
  const PREFIX = "bug1";
  const REPO = "myrepo";

  it("fires mutation with prefix, trimmed title, and repoRef on Enter", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugSetTitleDocument, variables: matchVars },
      result: { data: { bugSetTitle: { bug: { id: "1", title: "New title" } } } },
    };

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <TitleEditor bugPrefix={PREFIX} title="Old title" humanId="h1" repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    fireEvent.click(screen.getByTitle("Edit title"));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "  New title  " } });
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, title: "New title", repoRef: REPO },
    });
    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
  });

  it("fires mutation on Save button click", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugSetTitleDocument, variables: matchVars },
      result: { data: { bugSetTitle: { bug: { id: "1", title: "New title" } } } },
    };

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <TitleEditor bugPrefix={PREFIX} title="Old title" humanId="h1" repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    fireEvent.click(screen.getByTitle("Edit title"));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "New title" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, title: "New title", repoRef: REPO },
    });
    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
  });

  it("works with null repoRef (default repo)", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugSetTitleDocument, variables: matchVars },
      result: { data: { bugSetTitle: { bug: { id: "1", title: "New" } } } },
    };

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <TitleEditor bugPrefix={PREFIX} title="Old" humanId="h1" repo={null} />
        </Suspense>
      </MockedProvider>,
    );

    fireEvent.click(screen.getByTitle("Edit title"));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "New" } });
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, title: "New", repoRef: null },
    });
    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
  });

  it("does NOT fire mutation when the title is unchanged", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugSetTitleDocument, variables: matchVars },
      result: { data: { bugSetTitle: { bug: { id: "1", title: "Same" } } } },
    };

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <TitleEditor bugPrefix={PREFIX} title="Same" humanId="h1" repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    fireEvent.click(screen.getByTitle("Edit title"));
    // Value unchanged
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

    // Give Apollo a tick to fire the mutation if it were going to
    await new Promise((r) => setTimeout(r, 50));
    expect(matchVars).not.toHaveBeenCalled();
  });
});

describe("TitleEditor — failures", () => {
  it("stays in edit mode with the title and shows the error when saving fails", async () => {
    const mock = {
      request: { query: BugSetTitleDocument, variables: () => true },
      error: new Error("network down"),
    };
    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <TitleEditor bugPrefix="bug1" title="Old title" humanId="h1" repo="myrepo" />
        </Suspense>
      </MockedProvider>,
    );

    fireEvent.click(screen.getByTitle("Edit title"));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "New title" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("Failed to save the title: network down")).toBeInTheDocument();
    expect(screen.getByRole("textbox")).toHaveValue("New title");

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByTitle("Edit title"));
    expect(screen.queryByText(/Failed to save the title/)).toBeNull();
  });
});
